package handler

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// ReferralHandler handles referral-related requests
type ReferralHandler struct {
	referralService *service.ReferralService
	settingService  *service.SettingService
}

// NewReferralHandler creates a new ReferralHandler
func NewReferralHandler(referralService *service.ReferralService, settingService *service.SettingService) *ReferralHandler {
	return &ReferralHandler{
		referralService: referralService,
		settingService:  settingService,
	}
}

// ReferralInfoResponse represents the referral info response
type ReferralInfoResponse struct {
	Enabled          bool    `json:"enabled"`
	ReferralCode     string  `json:"referral_code"`
	ReferralLink     string  `json:"referral_link"`
	TotalInvited     int     `json:"total_invited"`
	TotalReward      float64 `json:"total_reward"`
	RegisterReward   float64 `json:"register_reward"`
	CommissionReward float64 `json:"commission_reward"`
	RegisterBonus    float64 `json:"register_bonus"`
	CommissionRate   float64 `json:"commission_rate"`
}

// ReferralRewardResponse represents a referral reward item
type ReferralRewardResponse struct {
	ID           int64   `json:"id"`
	RefereeEmail string  `json:"referee_email"`
	RewardType   string  `json:"reward_type"`
	RewardAmount float64 `json:"reward_amount"`
	SourceAmount float64 `json:"source_amount,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

// GetReferralInfo returns the user's referral information
// GET /api/v1/user/referral
func (h *ReferralHandler) GetReferralInfo(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	ctx := c.Request.Context()
	settings := h.referralService.GetSettings(ctx)

	if !settings.Enabled {
		response.Success(c, ReferralInfoResponse{
			Enabled: false,
		})
		return
	}

	apiBaseURL := h.settingService.GetAPIBaseURL(ctx)
	info, err := h.referralService.GetReferralInfo(ctx, subject.UserID, apiBaseURL)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// Use user's custom commission rate if set, otherwise use global rate
	commissionRate := settings.CommissionRate
	if info.UserCommissionRate != nil {
		commissionRate = *info.UserCommissionRate
	}

	response.Success(c, ReferralInfoResponse{
		Enabled:          true,
		ReferralCode:     info.ReferralCode,
		ReferralLink:     info.ReferralLink,
		TotalInvited:     info.TotalInvited,
		TotalReward:      info.TotalReward,
		RegisterReward:   info.RegisterReward,
		CommissionReward: info.CommissionReward,
		RegisterBonus:    settings.RegisterBonus,
		CommissionRate:   commissionRate,
	})
}

// GetReferralRewards returns the user's referral reward history
// GET /api/v1/user/referral/rewards
func (h *ReferralHandler) GetReferralRewards(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	// Parse pagination params
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	rewards, total, err := h.referralService.GetRewards(c.Request.Context(), subject.UserID, offset, pageSize)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items := make([]ReferralRewardResponse, 0, len(rewards))
	for _, r := range rewards {
		items = append(items, ReferralRewardResponse{
			ID:           r.ID,
			RefereeEmail: r.RefereeEmail,
			RewardType:   r.RewardType,
			RewardAmount: r.RewardAmount,
			SourceAmount: r.SourceAmount,
			CreatedAt:    r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	response.Success(c, gin.H{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetReferralSettings returns the referral system settings (public)
// GET /api/v1/referral/settings
func (h *ReferralHandler) GetReferralSettings(c *gin.Context) {
	settings := h.referralService.GetSettings(c.Request.Context())
	response.Success(c, gin.H{
		"enabled":         settings.Enabled,
		"register_bonus":  settings.RegisterBonus,
		"commission_rate": settings.CommissionRate,
	})
}
