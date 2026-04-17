//go:build unit

package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildBalanceLowEmailBody_ContainsRequiredFields(t *testing.T) {
	s := &BalanceNotifyService{}
	body := s.buildBalanceLowEmailBody("Alice", 3.14, 10.0, "MySite", "")

	require.Contains(t, body, "MySite")
	require.Contains(t, body, "Alice")
	require.Contains(t, body, "$3.14")
	require.Contains(t, body, "$10.00")
	require.NotContains(t, body, "%!")
	require.NotContains(t, body, "MISSING")
	require.NotContains(t, body, "EXTRA")
}

func TestBuildBalanceLowEmailBody_WithRechargeURL(t *testing.T) {
	s := &BalanceNotifyService{}
	body := s.buildBalanceLowEmailBody("Bob", 5.0, 20.0, "Site", "https://example.com/pay")

	require.Contains(t, body, `href="https://example.com/pay"`)
	require.Contains(t, body, "立即充值")
	require.NotContains(t, body, "%!")
}

func TestBuildBalanceLowEmailBody_RechargeURLEscaped(t *testing.T) {
	s := &BalanceNotifyService{}
	body := s.buildBalanceLowEmailBody("u", 1.0, 5.0, "Site", `https://example.com/?a=1&b=<script>`)

	require.Contains(t, body, "&amp;")
	require.Contains(t, body, "&lt;script&gt;")
	require.NotContains(t, body, "<script>")
}

func TestBuildBalanceLowEmailBody_NoRechargeURLOmitsButton(t *testing.T) {
	s := &BalanceNotifyService{}
	body := s.buildBalanceLowEmailBody("u", 1.0, 5.0, "Site", "")
	require.NotContains(t, body, `<a href`)
	require.NotContains(t, body, "立即充值")
}

func TestBuildQuotaAlertEmailBody_AllFieldsPresent(t *testing.T) {
	s := &BalanceNotifyService{}
	body := s.buildQuotaAlertEmailBody(42, "acc-foo", "anthropic", "日限额 / Daily", 750.50, 1000.0, 249.50, "$249.50", "MySite")

	require.Contains(t, body, "MySite")
	require.Contains(t, body, "#42")
	require.Contains(t, body, "acc-foo")
	require.Contains(t, body, "anthropic")
	require.Contains(t, body, "Daily")
	require.Contains(t, body, "$750.50")
	require.Contains(t, body, "$1000.00")
	require.Contains(t, body, "$249.50")
	require.NotContains(t, body, "%!")
	require.NotContains(t, body, "MISSING")
	require.NotContains(t, body, "EXTRA")
}

func TestBuildQuotaAlertEmailBody_UnlimitedDisplay(t *testing.T) {
	s := &BalanceNotifyService{}
	body := s.buildQuotaAlertEmailBody(1, "n", "p", "dim", 100.0, 0.0, 0.0, "30%", "Site")
	require.Contains(t, body, "无限制")
	require.Contains(t, body, "Unlimited")
}

func TestBuildBalanceLowEmailBody_NoCSSFormatError(t *testing.T) {
	s := &BalanceNotifyService{}
	body := s.buildBalanceLowEmailBody("u", 1.0, 5.0, "Site", "")
	require.True(t, strings.Contains(body, "0%") && strings.Contains(body, "100%"))
}

func TestBuildQuotaAlertEmailBody_NoCSSFormatError(t *testing.T) {
	s := &BalanceNotifyService{}
	body := s.buildQuotaAlertEmailBody(1, "n", "p", "d", 0, 0, 0, "$0.00", "Site")
	require.True(t, strings.Contains(body, "0%") && strings.Contains(body, "100%"))
}
