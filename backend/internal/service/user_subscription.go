package service

import (
	"math"
	"time"
)

const (
	DailyWindowSize   = 24 * time.Hour
	WeeklyWindowSize  = 7 * 24 * time.Hour
	MonthlyWindowSize = 30 * 24 * time.Hour
)

// WindowStart calculates the current tumbling window start anchored to the subscription's StartsAt.
// Formula: anchor + floor((now - anchor) / windowSize) * windowSize
func WindowStart(anchor, now time.Time, windowSize time.Duration) time.Time {
	elapsed := now.Sub(anchor)
	if elapsed < 0 {
		return anchor
	}
	periods := math.Floor(float64(elapsed) / float64(windowSize))
	return anchor.Add(time.Duration(periods) * windowSize)
}

// WindowEnd returns the end of the current tumbling window.
func WindowEnd(anchor, now time.Time, windowSize time.Duration) time.Time {
	return WindowStart(anchor, now, windowSize).Add(windowSize)
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

// NeedsDailyReset reports whether the stored daily window start differs from
// the correct anchored window start (or is nil, meaning first activation).
func (s *UserSubscription) NeedsDailyReset() bool {
	now := time.Now()
	correct := WindowStart(s.StartsAt, now, DailyWindowSize)
	return s.DailyWindowStart == nil || !s.DailyWindowStart.Equal(correct)
}

// NeedsWeeklyReset reports whether the stored weekly window start differs from
// the correct anchored window start (or is nil).
func (s *UserSubscription) NeedsWeeklyReset() bool {
	now := time.Now()
	correct := WindowStart(s.StartsAt, now, WeeklyWindowSize)
	return s.WeeklyWindowStart == nil || !s.WeeklyWindowStart.Equal(correct)
}

// NeedsMonthlyReset reports whether the stored monthly window start differs from
// the correct anchored window start (or is nil).
func (s *UserSubscription) NeedsMonthlyReset() bool {
	now := time.Now()
	correct := WindowStart(s.StartsAt, now, MonthlyWindowSize)
	return s.MonthlyWindowStart == nil || !s.MonthlyWindowStart.Equal(correct)
}

func (s *UserSubscription) DailyResetTime() *time.Time {
	if s.DailyWindowStart == nil {
		return nil
	}
	t := WindowEnd(s.StartsAt, time.Now(), DailyWindowSize)
	return &t
}

func (s *UserSubscription) WeeklyResetTime() *time.Time {
	if s.WeeklyWindowStart == nil {
		return nil
	}
	t := WindowEnd(s.StartsAt, time.Now(), WeeklyWindowSize)
	return &t
}

func (s *UserSubscription) MonthlyResetTime() *time.Time {
	if s.MonthlyWindowStart == nil {
		return nil
	}
	t := WindowEnd(s.StartsAt, time.Now(), MonthlyWindowSize)
	return &t
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
