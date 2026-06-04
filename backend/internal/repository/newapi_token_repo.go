package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// newAPIProvisionAdvisoryLockNamespace namespaces the 2-arg advisory lock used
// to serialize per-user relay-token provisioning, avoiding collisions with the
// 1-arg migrations lock and any other advisory locks in this database.
// Value spells "xsjh" in hex.
const newAPIProvisionAdvisoryLockNamespace = 0x78736a68

// sqlRowQueryer is the read surface shared by *sql.DB and *sql.Tx.
type sqlRowQueryer interface {
	sqlExecutor
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type newAPITokenRepository struct {
	db   *sql.DB
	exec sqlRowQueryer
	// isPostgres gates the advisory lock (unsupported on SQLite unit tests).
	isPostgres bool
}

// NewNewAPITokenRepository builds the per-user relay-token mapping repository.
func NewNewAPITokenRepository(client *dbent.Client, sqlDB *sql.DB) service.NewAPITokenRepository {
	return &newAPITokenRepository{
		db:         sqlDB,
		exec:       sqlDB,
		isPostgres: client != nil && supportsRowLock(client),
	}
}

func scanNewAPIUserToken(row *sql.Row) (*service.NewAPIUserToken, error) {
	var (
		m         service.NewAPIUserToken
		lastError sql.NullString
		revokedAt sql.NullTime
	)
	err := row.Scan(&m.UserID, &m.NewAPITokenID, &m.Status, &lastError, &m.CreatedAt, &revokedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrNewAPITokenNotFound
	}
	if err != nil {
		return nil, err
	}
	m.LastError = lastError.String
	if revokedAt.Valid {
		t := revokedAt.Time
		m.RevokedAt = &t
	}
	return &m, nil
}

func (r *newAPITokenRepository) GetByUserID(ctx context.Context, userID int64) (*service.NewAPIUserToken, error) {
	const q = `
		SELECT user_id, newapi_token_id, status, last_error, created_at, revoked_at
		FROM newapi_user_tokens
		WHERE user_id = $1
	`
	return scanNewAPIUserToken(r.exec.QueryRowContext(ctx, q, userID))
}

func (r *newAPITokenRepository) Upsert(ctx context.Context, userID int64, tokenID int) error {
	const q = `
		INSERT INTO newapi_user_tokens (user_id, newapi_token_id, status, last_error, revoked_at)
		VALUES ($1, $2, 'active', NULL, NULL)
		ON CONFLICT (user_id) DO UPDATE SET
			newapi_token_id = EXCLUDED.newapi_token_id,
			status = 'active',
			last_error = NULL,
			revoked_at = NULL
	`
	_, err := r.exec.ExecContext(ctx, q, userID, tokenID)
	return err
}

func (r *newAPITokenRepository) MarkRevoked(ctx context.Context, userID int64) error {
	const q = `
		UPDATE newapi_user_tokens
		SET status = 'revoked', last_error = NULL, revoked_at = $2
		WHERE user_id = $1
	`
	_, err := r.exec.ExecContext(ctx, q, userID, time.Now())
	return err
}

func (r *newAPITokenRepository) MarkRevokeFailed(ctx context.Context, userID int64, lastError string) error {
	const q = `
		UPDATE newapi_user_tokens
		SET status = 'revoke_failed', last_error = $2, revoked_at = $3
		WHERE user_id = $1
	`
	_, err := r.exec.ExecContext(ctx, q, userID, lastError, time.Now())
	return err
}

func (r *newAPITokenRepository) Delete(ctx context.Context, userID int64) error {
	const q = `DELETE FROM newapi_user_tokens WHERE user_id = $1`
	_, err := r.exec.ExecContext(ctx, q, userID)
	return err
}

// WithProvisionLock runs fn inside a transaction holding a per-user advisory
// lock so provisioning is serialized per user. On non-Postgres drivers (unit
// tests), it degrades to a plain transaction without the advisory lock.
func (r *newAPITokenRepository) WithProvisionLock(ctx context.Context, userID int64, fn func(txRepo service.NewAPITokenRepository) error) error {
	if r.db == nil {
		return errors.New("newapi token repo: nil *sql.DB")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if r.isPostgres {
		// 2-arg advisory lock: (namespace, userID) avoids collisions with the
		// 1-arg migrations lock. Released automatically at tx end.
		if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1, $2)", newAPIProvisionAdvisoryLockNamespace, userID); err != nil {
			return err
		}
	}

	txRepo := &newAPITokenRepository{db: r.db, exec: tx, isPostgres: r.isPostgres}
	if err := fn(txRepo); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}
