//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func newBalanceNotifyServiceForTest() (*BalanceNotifyService, *mockSettingRepo) {
	repo := newMockSettingRepo()
	email := NewEmailService(repo, nil)
	return NewBalanceNotifyService(email, repo, nil), repo
}

func TestCheckBalanceAfterDeduction_NilUser(t *testing.T) {
	s, _ := newBalanceNotifyServiceForTest()
	s.CheckBalanceAfterDeduction(context.Background(), nil, 100, 50)
}

func TestCheckBalanceAfterDeduction_UserNotifyDisabled(t *testing.T) {
	s, repo := newBalanceNotifyServiceForTest()
	repo.data[SettingKeyBalanceLowNotifyEnabled] = "true"
	repo.data[SettingKeyBalanceLowNotifyThreshold] = "10"
	u := &User{ID: 1, BalanceNotifyEnabled: false}
	s.CheckBalanceAfterDeduction(context.Background(), u, 20, 15)
}

func TestCheckBalanceAfterDeduction_GlobalDisabled(t *testing.T) {
	s, repo := newBalanceNotifyServiceForTest()
	repo.data[SettingKeyBalanceLowNotifyEnabled] = "false"
	u := &User{ID: 1, BalanceNotifyEnabled: true}
	s.CheckBalanceAfterDeduction(context.Background(), u, 20, 15)
}

func TestCheckBalanceAfterDeduction_NoCrossingNotFired(t *testing.T) {
	s, repo := newBalanceNotifyServiceForTest()
	repo.data[SettingKeyBalanceLowNotifyEnabled] = "true"
	repo.data[SettingKeyBalanceLowNotifyThreshold] = "10"
	u := &User{ID: 1, BalanceNotifyEnabled: true}

	s.CheckBalanceAfterDeduction(context.Background(), u, 100, 5)
	s.CheckBalanceAfterDeduction(context.Background(), u, 5, 2)
}

func TestGetBalanceNotifyConfig_AllFields(t *testing.T) {
	s, repo := newBalanceNotifyServiceForTest()
	repo.data[SettingKeyBalanceLowNotifyEnabled] = "true"
	repo.data[SettingKeyBalanceLowNotifyThreshold] = "12.5"
	repo.data[SettingKeyBalanceLowNotifyRechargeURL] = "https://example.com/pay"

	enabled, threshold, url := s.getBalanceNotifyConfig(context.Background())
	require.True(t, enabled)
	require.Equal(t, 12.5, threshold)
	require.Equal(t, "https://example.com/pay", url)
}

func TestIsAccountQuotaNotifyEnabled(t *testing.T) {
	s, repo := newBalanceNotifyServiceForTest()
	require.False(t, s.isAccountQuotaNotifyEnabled(context.Background()))

	repo.data[SettingKeyAccountQuotaNotifyEnabled] = "false"
	require.False(t, s.isAccountQuotaNotifyEnabled(context.Background()))

	repo.data[SettingKeyAccountQuotaNotifyEnabled] = "true"
	require.True(t, s.isAccountQuotaNotifyEnabled(context.Background()))
}

func TestGetSiteName_FallsBackToDefault(t *testing.T) {
	s, _ := newBalanceNotifyServiceForTest()
	name := s.getSiteName(context.Background())
	require.Equal(t, defaultSiteName, name)
}

func TestGetSiteName_Configured(t *testing.T) {
	s, repo := newBalanceNotifyServiceForTest()
	repo.data[SettingKeySiteName] = "My Site"
	require.Equal(t, "My Site", s.getSiteName(context.Background()))
}

func TestCrossedDownward_CrossesBelow(t *testing.T) {
	require.True(t, crossedDownward(100, 5, 10))
}

func TestCrossedDownward_ExactlyAtThreshold(t *testing.T) {
	require.False(t, crossedDownward(100, 10, 10))
}

func TestCrossedDownward_OldExactlyAtThreshold_NewBelow(t *testing.T) {
	require.True(t, crossedDownward(10, 5, 10))
}

func TestCrossedDownward_AlreadyBelow(t *testing.T) {
	require.False(t, crossedDownward(5, 3, 10))
}

func TestCheckQuotaDimCrossings_NoDimensions(t *testing.T) {
	s, _ := newBalanceNotifyServiceForTest()
	account := &Account{ID: 1, Name: "test", Platform: PlatformAnthropic}
	s.checkQuotaDimCrossings(account, nil, 10, []string{"admin@example.com"}, "TestSite")
	s.checkQuotaDimCrossings(account, []quotaDim{}, 10, []string{"admin@example.com"}, "TestSite")
}

func TestCheckQuotaDimCrossings_ZeroThresholdSkipped(t *testing.T) {
	s, _ := newBalanceNotifyServiceForTest()
	account := &Account{ID: 1, Name: "test", Platform: PlatformAnthropic}
	dims := []quotaDim{
		{
			name:          quotaDimDaily,
			enabled:       true,
			threshold:     0,
			thresholdType: thresholdTypeFixed,
			currentUsed:   950,
			limit:         1000,
		},
	}
	s.checkQuotaDimCrossings(account, dims, 50, []string{"admin@example.com"}, "TestSite")
}

func TestCheckQuotaDimCrossings_NoCrossing_BothBelowThreshold(t *testing.T) {
	s, _ := newBalanceNotifyServiceForTest()
	account := &Account{ID: 1, Name: "test", Platform: PlatformAnthropic}
	dims := []quotaDim{
		{
			name:          quotaDimDaily,
			enabled:       true,
			threshold:     400,
			thresholdType: thresholdTypeFixed,
			currentUsed:   300,
			limit:         1000,
		},
	}
	s.checkQuotaDimCrossings(account, dims, 50, []string{"admin@example.com"}, "TestSite")
}

func TestCheckQuotaDimCrossings_PercentageThreshold_NoCrossing(t *testing.T) {
	s, _ := newBalanceNotifyServiceForTest()
	account := &Account{ID: 1, Name: "test", Platform: PlatformAnthropic}
	dims := []quotaDim{
		{
			name:          quotaDimWeekly,
			enabled:       true,
			threshold:     30,
			thresholdType: thresholdTypePercentage,
			currentUsed:   500,
			limit:         1000,
		},
	}
	s.checkQuotaDimCrossings(account, dims, 50, []string{"admin@example.com"}, "TestSite")
}
