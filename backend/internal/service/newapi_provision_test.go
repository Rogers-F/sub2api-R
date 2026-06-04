//go:build unit

package service

import (
	"context"
	"errors"
	"sync"
	"testing"
)

// fakeNewAPIClient is an in-memory stand-in for the new-api admin client.
type fakeNewAPIClient struct {
	configured bool

	mu            sync.Mutex
	tokensByName  map[string]int
	nextID        int
	createCalls   int
	deleteCalls   int
	failCreate    error
	failDelete    error
	failFind      error
	failGetKey    error
	getKeyByID    map[int]string
	deletedTokens map[int]bool
}

func newFakeClient() *fakeNewAPIClient {
	return &fakeNewAPIClient{
		configured:    true,
		tokensByName:  map[string]int{},
		nextID:        100,
		getKeyByID:    map[int]string{},
		deletedTokens: map[int]bool{},
	}
}

func (f *fakeNewAPIClient) Configured() bool { return f.configured }

func (f *fakeNewAPIClient) CreateToken(_ context.Context, name string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failCreate != nil {
		return f.failCreate
	}
	f.createCalls++
	f.nextID++
	f.tokensByName[name] = f.nextID
	f.getKeyByID[f.nextID] = "sk-key-" + name
	return nil
}

func (f *fakeNewAPIClient) FindTokenIDByName(_ context.Context, name string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failFind != nil {
		return 0, f.failFind
	}
	return f.tokensByName[name], nil
}

func (f *fakeNewAPIClient) GetTokenKey(_ context.Context, id int) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failGetKey != nil {
		return "", f.failGetKey
	}
	return f.getKeyByID[id], nil
}

func (f *fakeNewAPIClient) DeleteToken(_ context.Context, id int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failDelete != nil {
		return f.failDelete
	}
	f.deleteCalls++
	f.deletedTokens[id] = true
	return nil
}

// fakeTokenRepo is an in-memory NewAPITokenRepository. WithProvisionLock runs
// fn directly (single-process, mutex-guarded) since there is no real DB.
type fakeTokenRepo struct {
	mu   sync.Mutex
	rows map[int64]*NewAPIUserToken
}

func newFakeRepo() *fakeTokenRepo {
	return &fakeTokenRepo{rows: map[int64]*NewAPIUserToken{}}
}

func (r *fakeTokenRepo) GetByUserID(_ context.Context, userID int64) (*NewAPIUserToken, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	row, ok := r.rows[userID]
	if !ok {
		return nil, ErrNewAPITokenNotFound
	}
	cp := *row
	return &cp, nil
}

func (r *fakeTokenRepo) Upsert(_ context.Context, userID int64, tokenID int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows[userID] = &NewAPIUserToken{UserID: userID, NewAPITokenID: tokenID, Status: NewAPITokenStatusActive}
	return nil
}

func (r *fakeTokenRepo) MarkRevoked(_ context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row, ok := r.rows[userID]; ok {
		row.Status = NewAPITokenStatusRevoked
		row.LastError = ""
	}
	return nil
}

func (r *fakeTokenRepo) MarkRevokeFailed(_ context.Context, userID int64, lastError string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if row, ok := r.rows[userID]; ok {
		row.Status = NewAPITokenStatusRevokeFailed
		row.LastError = lastError
	}
	return nil
}

func (r *fakeTokenRepo) Delete(_ context.Context, userID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.rows, userID)
	return nil
}

func (r *fakeTokenRepo) WithProvisionLock(_ context.Context, _ int64, fn func(txRepo NewAPITokenRepository) error) error {
	// Serialize like a real per-user advisory lock would.
	r.mu.Lock()
	locked := true
	unlock := func() {
		if locked {
			r.mu.Unlock()
			locked = false
		}
	}
	defer unlock()
	// The callback uses repo methods that re-lock; release before invoking.
	unlock()
	return fn(r)
}

func TestEnsureTokenID_CreatesWhenMissing(t *testing.T) {
	repo := newFakeRepo()
	client := newFakeClient()
	svc := NewNewAPIProvisionService(repo, client)

	id, err := svc.EnsureTokenID(context.Background(), 42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected positive token id, got %d", id)
	}
	if client.createCalls != 1 {
		t.Fatalf("expected 1 create call, got %d", client.createCalls)
	}
	row, err := repo.GetByUserID(context.Background(), 42)
	if err != nil || row.NewAPITokenID != id || row.Status != NewAPITokenStatusActive {
		t.Fatalf("mapping not persisted correctly: %+v err=%v", row, err)
	}
}

func TestEnsureTokenID_ReusesActiveMapping(t *testing.T) {
	repo := newFakeRepo()
	client := newFakeClient()
	svc := NewNewAPIProvisionService(repo, client)

	id1, err := svc.EnsureTokenID(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	id2, err := svc.EnsureTokenID(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if id1 != id2 {
		t.Fatalf("expected same id reused, got %d then %d", id1, id2)
	}
	if client.createCalls != 1 {
		t.Fatalf("expected exactly 1 create (reuse), got %d", client.createCalls)
	}
}

func TestEnsureTokenID_ReusesOrphanUpstreamToken(t *testing.T) {
	repo := newFakeRepo()
	client := newFakeClient()
	// Simulate an orphan: upstream token exists but no mapping row.
	client.tokensByName["xsjh_9"] = 555
	client.getKeyByID[555] = "sk-orphan"
	svc := NewNewAPIProvisionService(repo, client)

	id, err := svc.EnsureTokenID(context.Background(), 9)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if id != 555 {
		t.Fatalf("expected orphan id 555 reused, got %d", id)
	}
	if client.createCalls != 0 {
		t.Fatalf("expected no create for orphan reuse, got %d", client.createCalls)
	}
}

func TestEnsureTokenID_NotConfigured(t *testing.T) {
	repo := newFakeRepo()
	client := newFakeClient()
	client.configured = false
	svc := NewNewAPIProvisionService(repo, client)

	if _, err := svc.EnsureTokenID(context.Background(), 1); !errors.Is(err, ErrNewAPINotConfigured) {
		t.Fatalf("expected ErrNewAPINotConfigured, got %v", err)
	}
}

func TestGetTokenKey_CachesInMemory(t *testing.T) {
	repo := newFakeRepo()
	client := newFakeClient()
	svc := NewNewAPIProvisionService(repo, client)

	k1, err := svc.GetTokenKey(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	// Mutate upstream key; cache should still return the original.
	client.mu.Lock()
	for id := range client.getKeyByID {
		client.getKeyByID[id] = "sk-CHANGED"
	}
	client.mu.Unlock()

	k2, err := svc.GetTokenKey(context.Background(), 3)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if k1 != k2 {
		t.Fatalf("expected cached key, got %q then %q", k1, k2)
	}
}

func TestRevokeForUser_SoftFailureRecorded(t *testing.T) {
	repo := newFakeRepo()
	client := newFakeClient()
	svc := NewNewAPIProvisionService(repo, client)

	if _, err := svc.EnsureTokenID(context.Background(), 5); err != nil {
		t.Fatalf("setup: %v", err)
	}
	client.failDelete = errors.New("boom")

	// Must NOT return a hard error (user disable must proceed).
	if err := svc.RevokeForUser(context.Background(), 5); err != nil {
		t.Fatalf("revoke should be soft, got %v", err)
	}
	row, err := repo.GetByUserID(context.Background(), 5)
	if err != nil {
		t.Fatalf("mapping missing: %v", err)
	}
	if row.Status != NewAPITokenStatusRevokeFailed || row.LastError == "" {
		t.Fatalf("expected revoke_failed + last_error, got %+v", row)
	}
}

func TestRevokeForUser_SuccessEvictsCache(t *testing.T) {
	repo := newFakeRepo()
	client := newFakeClient()
	svc := NewNewAPIProvisionService(repo, client)

	if _, err := svc.GetTokenKey(context.Background(), 8); err != nil {
		t.Fatalf("setup: %v", err)
	}
	if err := svc.RevokeForUser(context.Background(), 8); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if client.deleteCalls != 1 {
		t.Fatalf("expected 1 delete, got %d", client.deleteCalls)
	}
	if _, ok := svc.cachedKey(8); ok {
		t.Fatalf("expected cache evicted after revoke")
	}
	row, err := repo.GetByUserID(context.Background(), 8)
	if err != nil || row.Status != NewAPITokenStatusRevoked {
		t.Fatalf("expected revoked status, got %+v err=%v", row, err)
	}
}
