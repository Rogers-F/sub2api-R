package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Referral system errors
var (
	ErrReferralCodeInvalid      = infraerrors.BadRequest("REFERRAL_CODE_INVALID", "invalid referral code")
	ErrReferralSelfReferral     = infraerrors.BadRequest("SELF_REFERRAL", "cannot use your own referral code")
	ErrReferralMaxRewardReached = infraerrors.BadRequest("MAX_REWARD_REACHED", "referrer has reached maximum reward limit")
	ErrReferralDisabled         = infraerrors.BadRequest("REFERRAL_DISABLED", "referral system is disabled")
)

// ReferralReward represents a referral reward record
type ReferralReward struct {
	ID             int64
	ReferrerID     int64
	RefereeID      int64
	RewardType     string  // register | commission
	SourceType     *string // redeem_code
	SourceID       *int64
	SourceAmount   float64
	RewardAmount   float64
	CommissionRate *float64 // Only for commission type
	CreatedAt      time.Time
}

// ReferralInfo contains user's referral information
type ReferralInfo struct {
	ReferralCode       string   `json:"referral_code"`
	ReferralLink       string   `json:"referral_link"`
	TotalInvited       int      `json:"total_invited"`
	TotalReward        float64  `json:"total_reward"`
	RegisterReward     float64  `json:"register_reward"`
	CommissionReward   float64  `json:"commission_reward"`
	UserCommissionRate *float64 `json:"-"` // User's custom commission rate (nil = use global)
}

// ReferralSettings contains referral system settings
type ReferralSettings struct {
	Enabled        bool    `json:"enabled"`
	RegisterBonus  float64 `json:"register_bonus"`
	CommissionRate float64 `json:"commission_rate"`
	MaxTotalReward float64 `json:"max_total_reward"` // 0 = unlimited
}

// ReferralRewardListItem is a simplified reward for list display
type ReferralRewardListItem struct {
	ID           int64     `json:"id"`
	RefereeEmail string    `json:"referee_email"`
	RewardType   string    `json:"reward_type"`
	RewardAmount float64   `json:"reward_amount"`
	SourceAmount float64   `json:"source_amount,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// ReferralRepository defines the referral data access interface
type ReferralRepository interface {
	// CreateReward creates a new referral reward record
	CreateReward(ctx context.Context, reward *ReferralReward) (*ReferralReward, error)

	// GetRewardsByReferrer returns all rewards for a referrer
	GetRewardsByReferrer(ctx context.Context, referrerID int64, offset, limit int) ([]*ReferralReward, int, error)

	// GetTotalRewardByReferrer returns the total reward amount for a referrer
	GetTotalRewardByReferrer(ctx context.Context, referrerID int64) (float64, error)

	// GetRewardStats returns reward statistics for a referrer
	GetRewardStats(ctx context.Context, referrerID int64) (totalInvited int, registerReward, commissionReward float64, err error)

	// GetUserByReferralCode returns a user by their referral code
	GetUserByReferralCode(ctx context.Context, code string) (*User, error)

	// GetRefereeCount returns the number of users referred by a referrer
	GetRefereeCount(ctx context.Context, referrerID int64) (int, error)
}
