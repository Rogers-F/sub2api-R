package service

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildNoAvailableAccountsEmptyPoolError_KnownWindowOmitsUpstreamDetail(t *testing.T) {
	resetAt := time.Now().Add(2 * time.Hour)
	diag := &accountFilterStats{
		Total:                   1,
		RateLimited:             1,
		RateLimited5h:           1,
		EarliestRateLimitReset:  &resetAt,
		EarliestRateLimitDetail: `{"type":"error","error":{"message":"weekly limit exceeded"}}`,
	}

	err := buildNoAvailableAccountsEmptyPoolError(false, PlatformAnthropic, diag)

	require.Error(t, err)
	require.Contains(t, err.Error(), "5小时窗口")
	require.NotContains(t, err.Error(), "上游：")
}

func TestBuildNoAvailableAccountsFilterError_UnknownWindowIncludesSanitizedDetail(t *testing.T) {
	resetAt := time.Now().Add(24 * time.Hour)
	st := accountFilterStats{
		Total:                   1,
		RateLimited:             1,
		RateLimitedOther:        1,
		EarliestRateLimitReset:  &resetAt,
		EarliestRateLimitDetail: `{"type":"error","error":{"message":"weekly limit exceeded request_id=req_abc123 access_token=super-secret ops@example.com https://secret.example.com/limit?token=abc"}}`,
	}

	err := buildNoAvailableAccountsFilterError("", st)

	require.Error(t, err)
	require.Contains(t, err.Error(), "未知窗口")
	require.Contains(t, err.Error(), "上游：weekly limit exceeded")
	require.NotContains(t, err.Error(), "req_abc123")
	require.NotContains(t, err.Error(), "super-secret")
	require.NotContains(t, err.Error(), "ops@example.com")
	require.NotContains(t, err.Error(), "https://secret.example.com")
}

func TestSummarizeRateLimitDetailForUser_TruncatesLongDetail(t *testing.T) {
	longDetail := strings.Repeat("weekly limit exceeded due to upstream quota exhaustion ", 8)

	got := summarizeRateLimitDetailForUser(longDetail)

	require.True(t, strings.HasSuffix(got, "..."), "expected ellipsis, got %q", got)
	require.LessOrEqual(t, len([]rune(got)), rateLimitDetailMaxLen)
}

func TestSummarizeRateLimitDetailForUser_RedactsSensitiveValues(t *testing.T) {
	detail := "weekly limit exceeded request_id=req_abc123 authorization=Bearer sk-secret token=abc access_token=xyz ops@example.com https://secret.example.com/path"

	got := summarizeRateLimitDetailForUser(detail)

	require.Contains(t, got, "[redacted-request-id]")
	require.Contains(t, got, "authorization=[redacted]")
	require.Contains(t, got, "token=[redacted]")
	require.Contains(t, got, "access_token=[redacted]")
	require.Contains(t, got, "[redacted-email]")
	require.Contains(t, got, "[redacted-url]")
	require.NotContains(t, got, "req_abc123")
	require.NotContains(t, got, "sk-secret")
	require.NotContains(t, got, "ops@example.com")
	require.NotContains(t, got, "https://secret.example.com")
}
