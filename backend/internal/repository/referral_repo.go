package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type referralRepository struct {
	client *dbent.Client
	sql    sqlQueryer
}

// NewReferralRepository creates a new referral repository
func NewReferralRepository(client *dbent.Client, sqlDB *sql.DB) service.ReferralRepository {
	return &referralRepository{client: client, sql: sqlDB}
}

// sqlQueryerFromContext returns the transaction's sql executor if available, otherwise the default one
func (r *referralRepository) sqlQueryerFromContext(ctx context.Context) sqlQueryer {
	if tx := dbent.TxFromContext(ctx); tx != nil {
		return tx.Client()
	}
	return r.sql
}

// CreateReward creates a new referral reward record
func (r *referralRepository) CreateReward(ctx context.Context, reward *service.ReferralReward) (*service.ReferralReward, error) {
	query := `
		INSERT INTO referral_rewards (referrer_id, referee_id, reward_type, source_type, source_id, source_amount, reward_amount, commission_rate, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	now := time.Now()
	reward.CreatedAt = now

	sqlq := r.sqlQueryerFromContext(ctx)
	args := []any{
		reward.ReferrerID,
		reward.RefereeID,
		reward.RewardType,
		reward.SourceType,
		reward.SourceID,
		reward.SourceAmount,
		reward.RewardAmount,
		reward.CommissionRate,
		now,
	}
	if err := scanSingleRow(ctx, sqlq, query, args, &reward.ID); err != nil {
		return nil, fmt.Errorf("create referral reward: %w", err)
	}
	return reward, nil
}

// GetRewardsByReferrer returns all rewards for a referrer with pagination
func (r *referralRepository) GetRewardsByReferrer(ctx context.Context, referrerID int64, offset, limit int) ([]*service.ReferralReward, int, error) {
	sqlq := r.sqlQueryerFromContext(ctx)

	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM referral_rewards WHERE referrer_id = $1`
	if err := scanSingleRow(ctx, sqlq, countQuery, []any{referrerID}, &total); err != nil {
		return nil, 0, fmt.Errorf("count referral rewards: %w", err)
	}

	// Get rewards
	query := `
		SELECT id, referrer_id, referee_id, reward_type, source_type, source_id, source_amount, reward_amount, commission_rate, created_at
		FROM referral_rewards
		WHERE referrer_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := sqlq.QueryContext(ctx, query, referrerID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("get referral rewards: %w", err)
	}
	defer rows.Close()

	var rewards []*service.ReferralReward
	for rows.Next() {
		var rw service.ReferralReward
		if err := rows.Scan(
			&rw.ID,
			&rw.ReferrerID,
			&rw.RefereeID,
			&rw.RewardType,
			&rw.SourceType,
			&rw.SourceID,
			&rw.SourceAmount,
			&rw.RewardAmount,
			&rw.CommissionRate,
			&rw.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan referral reward: %w", err)
		}
		rewards = append(rewards, &rw)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("rows error: %w", err)
	}

	return rewards, total, nil
}

// GetTotalRewardByReferrer returns the total reward amount for a referrer
func (r *referralRepository) GetTotalRewardByReferrer(ctx context.Context, referrerID int64) (float64, error) {
	sqlq := r.sqlQueryerFromContext(ctx)
	query := `SELECT COALESCE(SUM(reward_amount), 0) FROM referral_rewards WHERE referrer_id = $1`
	var total float64
	if err := scanSingleRow(ctx, sqlq, query, []any{referrerID}, &total); err != nil {
		return 0, fmt.Errorf("get total reward: %w", err)
	}
	return total, nil
}

// GetRewardStats returns reward statistics for a referrer
func (r *referralRepository) GetRewardStats(ctx context.Context, referrerID int64) (totalInvited int, registerReward, commissionReward float64, err error) {
	sqlq := r.sqlQueryerFromContext(ctx)

	// Get total invited count
	countQuery := `SELECT COUNT(DISTINCT referee_id) FROM referral_rewards WHERE referrer_id = $1`
	if err = scanSingleRow(ctx, sqlq, countQuery, []any{referrerID}, &totalInvited); err != nil {
		return 0, 0, 0, fmt.Errorf("get total invited: %w", err)
	}

	// Get reward amounts by type
	statsQuery := `
		SELECT reward_type, COALESCE(SUM(reward_amount), 0)
		FROM referral_rewards
		WHERE referrer_id = $1
		GROUP BY reward_type
	`
	rows, err := sqlq.QueryContext(ctx, statsQuery, referrerID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("get reward stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var rewardType string
		var amount float64
		if err := rows.Scan(&rewardType, &amount); err != nil {
			return 0, 0, 0, fmt.Errorf("scan reward stats: %w", err)
		}
		switch rewardType {
		case service.ReferralRewardTypeRegister:
			registerReward = amount
		case service.ReferralRewardTypeCommission:
			commissionReward = amount
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0, 0, fmt.Errorf("rows error: %w", err)
	}

	return totalInvited, registerReward, commissionReward, nil
}

// GetUserByReferralCode returns a user by their referral code
func (r *referralRepository) GetUserByReferralCode(ctx context.Context, code string) (*service.User, error) {
	sqlq := r.sqlQueryerFromContext(ctx)
	query := `
		SELECT id, email, username, notes, password_hash, role, balance, concurrency, status, token_version, created_at, updated_at
		FROM users
		WHERE referral_code = $1 AND deleted_at IS NULL
	`
	var u service.User
	var tokenVersion sql.NullInt64
	err := scanSingleRow(ctx, sqlq, query, []any{code},
		&u.ID,
		&u.Email,
		&u.Username,
		&u.Notes,
		&u.PasswordHash,
		&u.Role,
		&u.Balance,
		&u.Concurrency,
		&u.Status,
		&tokenVersion,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, service.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by referral code: %w", err)
	}
	if tokenVersion.Valid {
		u.TokenVersion = tokenVersion.Int64
	}
	return &u, nil
}

// GetRefereeCount returns the number of users referred by a referrer
func (r *referralRepository) GetRefereeCount(ctx context.Context, referrerID int64) (int, error) {
	sqlq := r.sqlQueryerFromContext(ctx)
	query := `SELECT COUNT(*) FROM users WHERE referrer_id = $1 AND deleted_at IS NULL`
	var count int
	if err := scanSingleRow(ctx, sqlq, query, []any{referrerID}, &count); err != nil {
		return 0, fmt.Errorf("get referee count: %w", err)
	}
	return count, nil
}
