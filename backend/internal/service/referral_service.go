package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	mathrand "math/rand"
	"strings"
	"time"
)

// ReferralService handles referral system operations
type ReferralService struct {
	referralRepo   ReferralRepository
	userRepo       UserRepository
	settingService *SettingService
}

// NewReferralService creates a new referral service instance
func NewReferralService(
	referralRepo ReferralRepository,
	userRepo UserRepository,
	settingService *SettingService,
) *ReferralService {
	return &ReferralService{
		referralRepo:   referralRepo,
		userRepo:       userRepo,
		settingService: settingService,
	}
}

// GetSettings returns the current referral system settings
func (s *ReferralService) GetSettings(ctx context.Context) *ReferralSettings {
	return &ReferralSettings{
		Enabled:        s.settingService.IsReferralEnabled(ctx),
		RegisterBonus:  s.settingService.GetReferralRegisterBonus(ctx),
		CommissionRate: s.settingService.GetReferralCommissionRate(ctx),
		MaxTotalReward: s.settingService.GetReferralMaxTotalReward(ctx),
	}
}

// IsEnabled checks if the referral system is enabled
func (s *ReferralService) IsEnabled(ctx context.Context) bool {
	return s.settingService.IsReferralEnabled(ctx)
}

// GenerateReferralCode generates a unique referral code
func (s *ReferralService) GenerateReferralCode() string {
	bytes := make([]byte, 6) // 6 bytes = 12 hex chars
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based code using new random source
		rng := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
		code := fmt.Sprintf("%012x", rng.Int63())
		if len(code) > 12 {
			code = code[:12]
		}
		return code
	}
	return hex.EncodeToString(bytes)
}

// GetUserByReferralCode returns a user by their referral code
func (s *ReferralService) GetUserByReferralCode(ctx context.Context, code string) (*User, error) {
	if code == "" {
		return nil, ErrReferralCodeInvalid
	}
	user, err := s.referralRepo.GetUserByReferralCode(ctx, code)
	if err != nil {
		return nil, ErrReferralCodeInvalid
	}
	return user, nil
}

// ProcessRegistrationReferral handles the referral reward for a new registration
// Returns the bonus amount given to each party, or 0 if no referral
func (s *ReferralService) ProcessRegistrationReferral(ctx context.Context, newUserID int64, referrerID int64) (float64, error) {
	if !s.IsEnabled(ctx) {
		return 0, nil
	}

	settings := s.GetSettings(ctx)
	bonus := settings.RegisterBonus

	if bonus <= 0 {
		return 0, nil
	}

	// Check if referrer has reached max reward limit
	if settings.MaxTotalReward > 0 {
		totalReward, err := s.referralRepo.GetTotalRewardByReferrer(ctx, referrerID)
		if err != nil {
			return 0, fmt.Errorf("get total reward: %w", err)
		}
		if totalReward >= settings.MaxTotalReward {
			log.Printf("[Referral] Referrer %d has reached max reward limit (%.2f >= %.2f)", referrerID, totalReward, settings.MaxTotalReward)
			return 0, nil // Silently skip, don't return error
		}
		// Adjust bonus if it would exceed limit
		if totalReward+bonus > settings.MaxTotalReward {
			bonus = settings.MaxTotalReward - totalReward
		}
	}

	// Create a single reward record for this referral relationship
	// reward_amount represents the bonus each party receives (both get equal amount)
	_, err := s.referralRepo.CreateReward(ctx, &ReferralReward{
		ReferrerID:   referrerID,
		RefereeID:    newUserID,
		RewardType:   ReferralRewardTypeRegister,
		RewardAmount: bonus, // Each party receives this amount
	})
	if err != nil {
		return 0, fmt.Errorf("create referral reward: %w", err)
	}

	// Add balance to referrer
	if err := s.userRepo.UpdateBalance(ctx, referrerID, bonus); err != nil {
		return 0, fmt.Errorf("update referrer balance: %w", err)
	}

	// Add balance to new user
	if err := s.userRepo.UpdateBalance(ctx, newUserID, bonus); err != nil {
		return 0, fmt.Errorf("update new user balance: %w", err)
	}

	log.Printf("[Referral] Registration bonus: referrer=%d, referee=%d, amount=%.2f", referrerID, newUserID, bonus)
	return bonus, nil
}

// ProcessRedeemCommission handles the commission for a balance redeem
// Returns the commission amount given to the referrer, or 0 if no referral
func (s *ReferralService) ProcessRedeemCommission(ctx context.Context, userID int64, referrerID int64, redeemAmount float64, sourceID int64) (float64, error) {
	if !s.IsEnabled(ctx) {
		return 0, nil
	}

	if redeemAmount <= 0 {
		return 0, nil
	}

	settings := s.GetSettings(ctx)
	rate := settings.CommissionRate

	if rate <= 0 {
		return 0, nil
	}

	commission := redeemAmount * rate

	// Check if referrer has reached max reward limit
	if settings.MaxTotalReward > 0 {
		totalReward, err := s.referralRepo.GetTotalRewardByReferrer(ctx, referrerID)
		if err != nil {
			return 0, fmt.Errorf("get total reward: %w", err)
		}
		if totalReward >= settings.MaxTotalReward {
			log.Printf("[Referral] Referrer %d has reached max reward limit for commission", referrerID)
			return 0, nil // Silently skip
		}
		// Adjust commission if it would exceed limit
		if totalReward+commission > settings.MaxTotalReward {
			commission = settings.MaxTotalReward - totalReward
		}
	}

	sourceType := ReferralSourceTypeRedeemCode
	// Create commission reward record
	_, err := s.referralRepo.CreateReward(ctx, &ReferralReward{
		ReferrerID:     referrerID,
		RefereeID:      userID,
		RewardType:     ReferralRewardTypeCommission,
		SourceType:     &sourceType,
		SourceID:       &sourceID,
		SourceAmount:   redeemAmount,
		RewardAmount:   commission,
		CommissionRate: &rate,
	})
	if err != nil {
		return 0, fmt.Errorf("create commission reward: %w", err)
	}

	// Add commission to referrer's balance
	if err := s.userRepo.UpdateBalance(ctx, referrerID, commission); err != nil {
		return 0, fmt.Errorf("update referrer balance: %w", err)
	}

	log.Printf("[Referral] Commission: referrer=%d, user=%d, amount=%.2f, rate=%.2f, source=%d",
		referrerID, userID, commission, rate, sourceID)
	return commission, nil
}

// GetReferralInfo returns the referral information for a user
func (s *ReferralService) GetReferralInfo(ctx context.Context, userID int64, apiBaseURL string) (*ReferralInfo, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}

	var referralCode string
	if user.ReferralCode != nil {
		referralCode = *user.ReferralCode
	}

	// Auto-generate referral code for existing users who don't have one
	if referralCode == "" && s.IsEnabled(ctx) {
		newCode := s.GenerateReferralCode()
		user.ReferralCode = &newCode
		if err := s.userRepo.Update(ctx, user); err != nil {
			log.Printf("[Referral] Failed to generate referral code for user %d: %v", userID, err)
			// Continue without referral code rather than failing
		} else {
			referralCode = newCode
			log.Printf("[Referral] Generated referral code for existing user %d: %s", userID, newCode)
		}
	}

	// Get stats
	totalInvited, registerReward, commissionReward, err := s.referralRepo.GetRewardStats(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get reward stats: %w", err)
	}

	// Build referral link
	referralLink := ""
	if referralCode != "" && apiBaseURL != "" {
		// Remove trailing slash to prevent double slashes in URL
		baseURL := strings.TrimSuffix(apiBaseURL, "/")
		referralLink = fmt.Sprintf("%s/register?ref=%s", baseURL, referralCode)
	}

	return &ReferralInfo{
		ReferralCode:     referralCode,
		ReferralLink:     referralLink,
		TotalInvited:     totalInvited,
		TotalReward:      registerReward + commissionReward,
		RegisterReward:   registerReward,
		CommissionReward: commissionReward,
	}, nil
}

// GetRewards returns the referral rewards for a user with pagination
func (s *ReferralService) GetRewards(ctx context.Context, userID int64, offset, limit int) ([]*ReferralRewardListItem, int, error) {
	rewards, total, err := s.referralRepo.GetRewardsByReferrer(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("get rewards: %w", err)
	}

	// Convert to list items with referee email
	items := make([]*ReferralRewardListItem, 0, len(rewards))
	for _, r := range rewards {
		// Get referee email
		referee, err := s.userRepo.GetByID(ctx, r.RefereeID)
		if err != nil {
			continue // Skip if user not found
		}

		// Mask email
		maskedEmail := maskEmail(referee.Email)

		items = append(items, &ReferralRewardListItem{
			ID:           r.ID,
			RefereeEmail: maskedEmail,
			RewardType:   r.RewardType,
			RewardAmount: r.RewardAmount,
			SourceAmount: r.SourceAmount,
			CreatedAt:    r.CreatedAt,
		})
	}

	return items, total, nil
}

// maskEmail masks an email address for privacy
func maskEmail(email string) string {
	if len(email) < 5 {
		return "***"
	}
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}
	if atIndex <= 0 {
		return "***"
	}
	// Show first 2 chars and domain
	if atIndex <= 2 {
		return email[:1] + "***" + email[atIndex:]
	}
	return email[:2] + "***" + email[atIndex:]
}
