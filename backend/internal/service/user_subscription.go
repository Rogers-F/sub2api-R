package service

import "time"

// WindowResetInfo describes the reset status of a single usage window (daily/weekly/monthly).
type WindowResetInfo struct {
	Status  string     // one of WindowResetStatus* constants
	ResetAt *time.Time // countdown target (window end or subscription expiry)
}

type UserSubscription struct {
	ID      int64
	UserID  int64
	GroupID int64

	StartsAt  time.Time
	ExpiresAt time.Time
	Status    string

	DailyWindowStart   *time.Time
	WeeklyWindowStart  *time.Time
	MonthlyWindowStart *time.Time

	DailyUsageUSD   float64
	WeeklyUsageUSD  float64
	MonthlyUsageUSD float64

	DailyResetInfo   WindowResetInfo
	WeeklyResetInfo  WindowResetInfo
	MonthlyResetInfo WindowResetInfo

	AssignedBy *int64
	AssignedAt time.Time
	Notes      string

	CreatedAt time.Time
	UpdatedAt time.Time

	User           *User
	Group          *Group
	AssignedByUser *User
}

func (s *UserSubscription) IsActive() bool {
	return s.Status == SubscriptionStatusActive && time.Now().Before(s.ExpiresAt)
}

func (s *UserSubscription) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *UserSubscription) DaysRemaining() int {
	if s.IsExpired() {
		return 0
	}
	return int(time.Until(s.ExpiresAt).Hours() / 24)
}

func (s *UserSubscription) IsWindowActivated() bool {
	return s.DailyWindowStart != nil || s.WeeklyWindowStart != nil || s.MonthlyWindowStart != nil
}

func (s *UserSubscription) NeedsDailyReset() bool {
	if s.DailyWindowStart == nil {
		return false
	}
	return time.Since(*s.DailyWindowStart) >= 24*time.Hour
}

func (s *UserSubscription) NeedsWeeklyReset() bool {
	if s.WeeklyWindowStart == nil {
		return false
	}
	return time.Since(*s.WeeklyWindowStart) >= 7*24*time.Hour
}

func (s *UserSubscription) NeedsMonthlyReset() bool {
	if s.MonthlyWindowStart == nil {
		return false
	}
	return time.Since(*s.MonthlyWindowStart) >= 30*24*time.Hour
}

func (s *UserSubscription) DailyResetTime() *time.Time {
	if s.DailyWindowStart == nil {
		return nil
	}
	t := s.DailyWindowStart.Add(24 * time.Hour)
	return &t
}

func (s *UserSubscription) WeeklyResetTime() *time.Time {
	if s.WeeklyWindowStart == nil {
		return nil
	}
	t := s.WeeklyWindowStart.Add(7 * 24 * time.Hour)
	return &t
}

func (s *UserSubscription) MonthlyResetTime() *time.Time {
	if s.MonthlyWindowStart == nil {
		return nil
	}
	t := s.MonthlyWindowStart.Add(30 * 24 * time.Hour)
	return &t
}

// ShouldAllowWindowReset 仅当订阅剩余时间 >= 窗口时长时才允许重置
func (s *UserSubscription) ShouldAllowWindowReset(windowDuration time.Duration) bool {
	return time.Until(s.ExpiresAt) >= windowDuration
}

func (s *UserSubscription) CanResetDailyWindow() bool {
	return s.NeedsDailyReset() && s.ShouldAllowWindowReset(24*time.Hour)
}

func (s *UserSubscription) CanResetWeeklyWindow() bool {
	return s.NeedsWeeklyReset() && s.ShouldAllowWindowReset(7*24*time.Hour)
}

func (s *UserSubscription) CanResetMonthlyWindow() bool {
	return s.NeedsMonthlyReset() && s.ShouldAllowWindowReset(30*24*time.Hour)
}

func (s *UserSubscription) CheckDailyLimit(group *Group, additionalCost float64) bool {
	if !group.HasDailyLimit() {
		return true
	}
	return s.DailyUsageUSD+additionalCost <= *group.DailyLimitUSD
}

func (s *UserSubscription) CheckWeeklyLimit(group *Group, additionalCost float64) bool {
	if !group.HasWeeklyLimit() {
		return true
	}
	return s.WeeklyUsageUSD+additionalCost <= *group.WeeklyLimitUSD
}

func (s *UserSubscription) CheckMonthlyLimit(group *Group, additionalCost float64) bool {
	if !group.HasMonthlyLimit() {
		return true
	}
	return s.MonthlyUsageUSD+additionalCost <= *group.MonthlyLimitUSD
}

func (s *UserSubscription) CheckAllLimits(group *Group, additionalCost float64) (daily, weekly, monthly bool) {
	daily = s.CheckDailyLimit(group, additionalCost)
	weekly = s.CheckWeeklyLimit(group, additionalCost)
	monthly = s.CheckMonthlyLimit(group, additionalCost)
	return
}

// ComputeWindowResetStatus determines the reset status for a single usage window.
func (s *UserSubscription) ComputeWindowResetStatus(
	now time.Time,
	hasLimit bool,
	windowStart *time.Time,
	windowDuration time.Duration,
) WindowResetInfo {
	if !hasLimit {
		return WindowResetInfo{Status: WindowResetStatusNoLimit}
	}
	if now.After(s.ExpiresAt) {
		return WindowResetInfo{Status: WindowResetStatusExpiredSubscription}
	}
	if windowStart == nil {
		return WindowResetInfo{Status: WindowResetStatusAwaitingFirstUse}
	}
	windowEnd := windowStart.Add(windowDuration)

	if now.Before(windowEnd) {
		// Window is still active — check if after it ends, subscription has enough time for another window
		if s.ExpiresAt.Sub(windowEnd) < windowDuration {
			t := s.ExpiresAt
			return WindowResetInfo{Status: WindowResetStatusActiveFinalWindow, ResetAt: &t}
		}
		return WindowResetInfo{Status: WindowResetStatusActive, ResetAt: &windowEnd}
	}
	// Window expired — check if subscription has enough remaining time to reset
	if s.ExpiresAt.Sub(now) >= windowDuration {
		return WindowResetInfo{Status: WindowResetStatusExpiredWillReset}
	}
	t := s.ExpiresAt
	return WindowResetInfo{Status: WindowResetStatusActiveFinalWindow, ResetAt: &t}
}
