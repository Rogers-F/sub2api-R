package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

// NewAPIUserToken is the per-user mapping to an externally provisioned relay
// token. It stores ONLY the upstream token id, never the plaintext key.
type NewAPIUserToken struct {
	UserID        int64
	NewAPITokenID int
	Status        string
	LastError     string
	CreatedAt     time.Time
	RevokedAt     *time.Time
}

// new-api mapping status values.
const (
	NewAPITokenStatusActive       = "active"
	NewAPITokenStatusRevoked      = "revoked"
	NewAPITokenStatusRevokeFailed = "revoke_failed"
	newAPITokenNamePrefix         = "xsjh_"
	newAPIKeyCacheTTL             = 10 * time.Minute
)

// NewAPITokenRepository persists the per-user relay-token mapping. The plaintext
// key is never handled here.
type NewAPITokenRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*NewAPIUserToken, error)
	Upsert(ctx context.Context, userID int64, tokenID int) error
	MarkRevoked(ctx context.Context, userID int64) error
	MarkRevokeFailed(ctx context.Context, userID int64, lastError string) error
	Delete(ctx context.Context, userID int64) error
	// WithProvisionLock runs fn inside a transaction that holds a per-user
	// advisory lock (pg_advisory_xact_lock) so provisioning is serialized per
	// user. The provided repo operates within that same transaction.
	WithProvisionLock(ctx context.Context, userID int64, fn func(txRepo NewAPITokenRepository) error) error
}

// NewAPIClient is the subset of the new-api admin client the provisioning
// service depends on (declared here so the service layer owns its port).
type NewAPIClient interface {
	Configured() bool
	CreateToken(ctx context.Context, name string) error
	FindTokenIDByName(ctx context.Context, name string) (int, error)
	GetTokenKey(ctx context.Context, id int) (string, error)
	DeleteToken(ctx context.Context, id int) error
}

// ErrNewAPINotConfigured is returned when the new-api integration is missing
// its required configuration.
var ErrNewAPINotConfigured = errors.New("newapi integration is not configured")

// ErrNewAPITokenNotFound is returned when a user has no mapping row.
var ErrNewAPITokenNotFound = errors.New("newapi user token mapping not found")

// NewAPIProvisionService provisions and manages one relay token per user.
type NewAPIProvisionService struct {
	repo   NewAPITokenRepository
	client NewAPIClient

	mu       sync.Mutex
	keyCache map[int64]cachedKey
}

type cachedKey struct {
	key       string
	expiresAt time.Time
}

// NewNewAPIProvisionService constructs the provisioning service.
func NewNewAPIProvisionService(repo NewAPITokenRepository, client NewAPIClient) *NewAPIProvisionService {
	return &NewAPIProvisionService{
		repo:     repo,
		client:   client,
		keyCache: make(map[int64]cachedKey),
	}
}

func (s *NewAPIProvisionService) tokenName(userID int64) string {
	return newAPITokenNamePrefix + strconv.FormatInt(userID, 10)
}

// EnsureTokenID returns the new-api token id for the user, provisioning it if
// necessary. The critical section runs inside a per-user advisory-lock tx so
// concurrent callers for the same user do not create duplicate tokens.
func (s *NewAPIProvisionService) EnsureTokenID(ctx context.Context, userID int64) (int, error) {
	if s.client == nil || !s.client.Configured() {
		return 0, ErrNewAPINotConfigured
	}

	var tokenID int
	err := s.repo.WithProvisionLock(ctx, userID, func(txRepo NewAPITokenRepository) error {
		// 1. Already mapped and active -> reuse.
		existing, err := txRepo.GetByUserID(ctx, userID)
		if err != nil && !errors.Is(err, ErrNewAPITokenNotFound) {
			return err
		}
		if existing != nil && existing.Status == NewAPITokenStatusActive && existing.NewAPITokenID > 0 {
			tokenID = existing.NewAPITokenID
			return nil
		}

		name := s.tokenName(userID)

		// 2. Reuse an existing upstream token with our deterministic name
		//    (handles orphans left by a prior failed upsert).
		id, err := s.client.FindTokenIDByName(ctx, name)
		if err != nil {
			return fmt.Errorf("find newapi token: %w", err)
		}

		// 3. None found -> create then look up the id.
		if id <= 0 {
			if err := s.client.CreateToken(ctx, name); err != nil {
				return fmt.Errorf("create newapi token: %w", err)
			}
			id, err = s.client.FindTokenIDByName(ctx, name)
			if err != nil {
				return fmt.Errorf("find newapi token after create: %w", err)
			}
			if id <= 0 {
				return errors.New("newapi token not found after create")
			}
		}

		// 4. Persist the mapping (id only).
		if err := txRepo.Upsert(ctx, userID, id); err != nil {
			return fmt.Errorf("upsert newapi mapping: %w", err)
		}
		tokenID = id
		return nil
	})
	if err != nil {
		return 0, err
	}
	return tokenID, nil
}

// GetTokenKey returns the usable relay key (sk-...) for the user, provisioning
// the token if needed. The plaintext key is cached in memory only, with a short
// TTL, and is never persisted or logged.
func (s *NewAPIProvisionService) GetTokenKey(ctx context.Context, userID int64) (string, error) {
	if key, ok := s.cachedKey(userID); ok {
		return key, nil
	}

	tokenID, err := s.EnsureTokenID(ctx, userID)
	if err != nil {
		return "", err
	}

	key, err := s.client.GetTokenKey(ctx, tokenID)
	if err != nil {
		return "", fmt.Errorf("get newapi token key: %w", err)
	}

	s.storeKey(userID, key)
	return key, nil
}

func (s *NewAPIProvisionService) cachedKey(userID int64) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.keyCache[userID]
	if !ok {
		return "", false
	}
	if time.Now().After(entry.expiresAt) {
		delete(s.keyCache, userID)
		return "", false
	}
	return entry.key, true
}

func (s *NewAPIProvisionService) storeKey(userID int64, key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyCache[userID] = cachedKey{key: key, expiresAt: time.Now().Add(newAPIKeyCacheTTL)}
}

func (s *NewAPIProvisionService) evictKey(userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.keyCache, userID)
}

// RevokeForUser deletes the user's upstream relay token and retires the mapping.
//
// Revocation must never block the caller's primary action (user disable/delete).
// On upstream delete failure we record status='revoke_failed' + last_error and
// return nil (soft failure) so the disable/delete still proceeds. The cached key
// is always evicted.
func (s *NewAPIProvisionService) RevokeForUser(ctx context.Context, userID int64) error {
	s.evictKey(userID)

	if s.client == nil || !s.client.Configured() {
		// Integration off: nothing to revoke upstream. Still drop any mapping.
		return nil
	}

	mapping, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNewAPITokenNotFound) {
			return nil
		}
		// Don't block the caller on a read failure; record nothing we can't.
		logger.LegacyPrintf("service.newapi", "revoke: read mapping failed: user_id=%d err=%v", userID, err)
		return nil
	}
	if mapping == nil || mapping.NewAPITokenID <= 0 {
		return nil
	}

	if err := s.client.DeleteToken(ctx, mapping.NewAPITokenID); err != nil {
		// Soft failure: record + log, but do not fail the caller.
		logger.LegacyPrintf("service.newapi", "revoke: delete upstream token failed: user_id=%d err=%v", userID, err)
		if mErr := s.repo.MarkRevokeFailed(ctx, userID, err.Error()); mErr != nil {
			logger.LegacyPrintf("service.newapi", "revoke: mark revoke_failed failed: user_id=%d err=%v", userID, mErr)
		}
		return nil
	}

	if err := s.repo.MarkRevoked(ctx, userID); err != nil {
		logger.LegacyPrintf("service.newapi", "revoke: mark revoked failed: user_id=%d err=%v", userID, err)
	}
	return nil
}
