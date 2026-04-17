//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveBalanceThreshold_Fixed(t *testing.T) {
	require.Equal(t, 10.0, resolveBalanceThreshold(10, thresholdTypeFixed, 1000))
	require.Equal(t, 10.0, resolveBalanceThreshold(10, thresholdTypeFixed, 0))
	require.Equal(t, 0.0, resolveBalanceThreshold(0, thresholdTypeFixed, 1000))
}

func TestResolveBalanceThreshold_Percentage(t *testing.T) {
	require.Equal(t, 100.0, resolveBalanceThreshold(10, thresholdTypePercentage, 1000))
	require.Equal(t, 100.0, resolveBalanceThreshold(50, thresholdTypePercentage, 200))
}

func TestResolveBalanceThreshold_PercentageZeroRecharged(t *testing.T) {
	require.Equal(t, 10.0, resolveBalanceThreshold(10, thresholdTypePercentage, 0))
}

func TestResolveBalanceThreshold_EmptyType(t *testing.T) {
	require.Equal(t, 10.0, resolveBalanceThreshold(10, "", 1000))
}

func TestResolvedThreshold_FixedNormal(t *testing.T) {
	d := quotaDim{threshold: 400, thresholdType: thresholdTypeFixed, limit: 1000}
	require.Equal(t, 600.0, d.resolvedThreshold())
}

func TestResolvedThreshold_PercentageNormal(t *testing.T) {
	d := quotaDim{threshold: 30, thresholdType: thresholdTypePercentage, limit: 1000}
	require.InDelta(t, 700.0, d.resolvedThreshold(), 0.001)
}

func TestResolvedThreshold_ZeroLimit(t *testing.T) {
	d := quotaDim{threshold: 100, thresholdType: thresholdTypeFixed, limit: 0}
	require.Equal(t, 0.0, d.resolvedThreshold())
}

func TestSanitizeEmailHeader_CRLF(t *testing.T) {
	require.Equal(t, "Subject injected", sanitizeEmailHeader("Subject\r\n injected"))
}

func TestSanitizeEmailHeader_Clean(t *testing.T) {
	require.Equal(t, "Sub2API", sanitizeEmailHeader("Sub2API"))
}

func TestBuildQuotaDims_AllDimensionsReturned(t *testing.T) {
	a := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"quota_notify_daily_enabled":         true,
			"quota_notify_daily_threshold":       100.0,
			"quota_notify_daily_threshold_type":  thresholdTypeFixed,
			"quota_notify_weekly_enabled":        true,
			"quota_notify_weekly_threshold":      20.0,
			"quota_notify_weekly_threshold_type": thresholdTypePercentage,
			"quota_notify_total_enabled":         false,
			"quota_daily_limit":                  500.0,
			"quota_weekly_limit":                 2000.0,
			"quota_limit":                        10000.0,
			"quota_daily_used":                   50.0,
			"quota_weekly_used":                  300.0,
			"quota_used":                         1000.0,
		},
	}

	dims := buildQuotaDims(a)
	require.Len(t, dims, 3)
	require.Equal(t, quotaDimDaily, dims[0].name)
	require.True(t, dims[0].enabled)
	require.Equal(t, 100.0, dims[0].threshold)
	require.Equal(t, thresholdTypeFixed, dims[0].thresholdType)
	require.Equal(t, 500.0, dims[0].limit)
	require.Equal(t, 50.0, dims[0].currentUsed)
}

func TestBuildQuotaDimsFromState_UsesStateValues(t *testing.T) {
	a := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeAPIKey,
		Extra: map[string]any{
			"quota_notify_daily_enabled":   true,
			"quota_notify_daily_threshold": 100.0,
			"quota_daily_used":             999.0,
			"quota_daily_limit":            999.0,
		},
	}
	state := &AccountQuotaState{
		DailyUsed:   77.0,
		DailyLimit:  500.0,
		WeeklyUsed:  88.0,
		WeeklyLimit: 2000.0,
		TotalUsed:   99.0,
		TotalLimit:  10000.0,
	}
	dims := buildQuotaDimsFromState(a, state)
	require.Len(t, dims, 3)
	require.True(t, dims[0].enabled)
	require.Equal(t, 100.0, dims[0].threshold)
	require.Equal(t, 77.0, dims[0].currentUsed)
	require.Equal(t, 500.0, dims[0].limit)
}

func TestCollectBalanceNotifyRecipients_FiltersDisabledAndUnverified(t *testing.T) {
	s := &BalanceNotifyService{}
	u := &User{
		BalanceNotifyExtraEmails: []NotifyEmailEntry{
			{Email: "a@example.com", Verified: true, Disabled: false},
			{Email: "b@example.com", Verified: true, Disabled: true},
			{Email: "c@example.com", Verified: false, Disabled: false},
			{Email: "d@example.com", Verified: true, Disabled: false},
		},
	}
	got := s.collectBalanceNotifyRecipients(u)
	require.Equal(t, []string{"a@example.com", "d@example.com"}, got)
}

func TestCollectBalanceNotifyRecipients_DeduplicatesCaseInsensitive(t *testing.T) {
	s := &BalanceNotifyService{}
	u := &User{
		BalanceNotifyExtraEmails: []NotifyEmailEntry{
			{Email: "User@Example.com", Verified: true},
			{Email: "user@example.com", Verified: true},
			{Email: "USER@EXAMPLE.COM", Verified: true},
		},
	}
	got := s.collectBalanceNotifyRecipients(u)
	require.Len(t, got, 1)
	require.Equal(t, "User@Example.com", got[0])
}
