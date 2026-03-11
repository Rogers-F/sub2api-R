package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	rateLimit5hBase       = 5 * time.Hour
	rateLimit7dBase       = 7 * 24 * time.Hour
	rateLimit5hTolerance  = 20 * time.Minute
	rateLimit7dTolerance  = 6 * time.Hour
	rateLimitDetailMaxLen = 160
)

var (
	rateLimitDetailWhitespaceRegex     = regexp.MustCompile(`\s+`)
	rateLimitDetailBearerRegex         = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._\-+/=]+\b`)
	rateLimitDetailSecretFieldRegex    = regexp.MustCompile(`(?i)\b(authorization|access_token|refresh_token|id_token|api[_-]?key|token)\b\s*[:=]\s*("[^"]*"|'[^']*'|Bearer\s+[^\s,;]+|[^\s,;]+)`)
	rateLimitDetailRequestIDFieldRegex = regexp.MustCompile(`(?i)\b(x-request-id|request[ _-]?id)\b\s*[:=]\s*("[^"]*"|'[^']*'|[A-Za-z0-9._:-]+)`)
	rateLimitDetailReqIDRegex          = regexp.MustCompile(`\breq_[A-Za-z0-9]+\b`)
	rateLimitDetailUUIDRegex           = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\b`)
	rateLimitDetailEmailRegex          = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	rateLimitDetailURLRegex            = regexp.MustCompile(`(?i)\bhttps?://[^\s"'<>]+`)
)

// inferRateLimitWindowType 根据 RateLimitedAt 和 RateLimitResetAt 推断限流窗口类型。
func inferRateLimitWindowType(rateLimitedAt, rateLimitResetAt *time.Time) string {
	if rateLimitedAt == nil || rateLimitResetAt == nil {
		return "unknown window"
	}
	if !rateLimitResetAt.After(*rateLimitedAt) {
		return "unknown window"
	}
	d := rateLimitResetAt.Sub(*rateLimitedAt)
	switch {
	case durationWithin(d, rateLimit5hBase, rateLimit5hTolerance):
		return "5h window"
	case durationWithin(d, rateLimit7dBase, rateLimit7dTolerance):
		return "7d window"
	default:
		return "unknown window"
	}
}

func durationWithin(actual, target, tolerance time.Duration) bool {
	diff := actual - target
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

// dominantRateLimitWindow 返回统计中占主导的限流窗口类型描述。
func dominantRateLimitWindow(st *accountFilterStats) string {
	if st == nil || st.RateLimited <= 0 {
		return "unknown window"
	}
	switch {
	case st.RateLimited5h == st.RateLimited:
		return "5h window"
	case st.RateLimited7d == st.RateLimited:
		return "7d window"
	case st.RateLimited5h == 0 && st.RateLimited7d == 0:
		return "unknown window"
	default:
		return "mixed windows"
	}
}

// formatRecoveryHint 格式化恢复时间提示。
func formatRecoveryHint(resetAt *time.Time) string {
	if resetAt == nil {
		return "recovery time unknown"
	}
	at := resetAt.UTC().Format(time.RFC3339)
	d := time.Until(*resetAt)
	if d < 0 {
		d = 0
	}
	return fmt.Sprintf("earliest recovery at %s, in %s", at, compactDuration(d))
}

// compactDuration 将 Duration 格式化为紧凑的人类可读形式。
func compactDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	if d <= 0 {
		return "<1m"
	}
	days := int(d.Hours()) / 24
	h := int(d.Hours()) % 24
	m := int(d.Minutes()) % 60

	switch {
	case days > 0 && h > 0:
		return fmt.Sprintf("%dd%dh", days, h)
	case days > 0:
		return fmt.Sprintf("%dd", days)
	case h > 0 && m > 0:
		return fmt.Sprintf("%dh%dm", h, m)
	case h > 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dm", m)
	}
}

func summarizeRateLimitDetailForUser(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return ""
	}

	if msg := strings.TrimSpace(extractUpstreamErrorMessage([]byte(detail))); msg != "" {
		detail = msg
	}
	detail = sanitizeUpstreamErrorMessage(detail)
	detail = strings.ReplaceAll(detail, "\\n", " ")
	detail = strings.ReplaceAll(detail, "\\r", " ")
	detail = strings.ReplaceAll(detail, "\n", " ")
	detail = strings.ReplaceAll(detail, "\r", " ")
	detail = redactRateLimitDetail(detail)
	detail = rateLimitDetailWhitespaceRegex.ReplaceAllString(detail, " ")
	detail = strings.Trim(detail, " \t,;")
	if detail == "" {
		return ""
	}

	runes := []rune(detail)
	if len(runes) <= rateLimitDetailMaxLen {
		return detail
	}
	return strings.TrimSpace(string(runes[:rateLimitDetailMaxLen-3])) + "..."
}

func redactRateLimitDetail(detail string) string {
	detail = rateLimitDetailBearerRegex.ReplaceAllString(detail, "Bearer [redacted-token]")
	detail = rateLimitDetailSecretFieldRegex.ReplaceAllString(detail, `$1=[redacted]`)
	detail = rateLimitDetailRequestIDFieldRegex.ReplaceAllString(detail, `$1=[redacted-request-id]`)
	detail = rateLimitDetailReqIDRegex.ReplaceAllString(detail, "[redacted-request-id]")
	detail = rateLimitDetailUUIDRegex.ReplaceAllString(detail, "[redacted-id]")
	detail = rateLimitDetailEmailRegex.ReplaceAllString(detail, "[redacted-email]")
	detail = rateLimitDetailURLRegex.ReplaceAllString(detail, "[redacted-url]")
	return detail
}

func recordEarliestTime(dst **time.Time, candidate time.Time) {
	if *dst == nil || candidate.Before(**dst) {
		v := candidate
		*dst = &v
	}
}

func recordEarliestTimePtr(dst **time.Time, candidate *time.Time) {
	if candidate == nil {
		return
	}
	recordEarliestTime(dst, *candidate)
}

// classifyUnschedulableAccount 将一个不可调度的账号归类到 filterStats 的子原因中。
func classifyUnschedulableAccount(acc *Account, st *accountFilterStats) {
	now := time.Now()
	switch {
	case acc.IsRateLimited():
		st.RateLimited++
		recordEarliestTimePtr(&st.EarliestRateLimitReset, acc.RateLimitResetAt)

		// Use stored window type; fall back to inference for old data
		wt := acc.RateLimitWindowType
		if wt == "" {
			inferred := inferRateLimitWindowType(acc.RateLimitedAt, acc.RateLimitResetAt)
			if strings.HasPrefix(inferred, "5h") {
				wt = "5h"
			} else if strings.HasPrefix(inferred, "7d") {
				wt = "7d"
			}
		}
		switch wt {
		case "5h":
			st.RateLimited5h++
		case "7d":
			st.RateLimited7d++
		default:
			st.RateLimitedOther++
		}

		// Track detail of earliest-recovery account
		if acc.RateLimitResetAt != nil && st.EarliestRateLimitReset != nil &&
			acc.RateLimitResetAt.Equal(*st.EarliestRateLimitReset) {
			if detail := strings.TrimSpace(acc.RateLimitDetail); detail != "" || st.EarliestRateLimitDetail == "" {
				st.EarliestRateLimitDetail = detail
			}
		}
	case acc.IsOverloaded():
		st.Overloaded++
		recordEarliestTimePtr(&st.EarliestOverloadUntil, acc.OverloadUntil)
	case acc.TempUnschedulableUntil != nil && now.Before(*acc.TempUnschedulableUntil):
		st.TempUnschedulable++
		recordEarliestTimePtr(&st.EarliestTempUntil, acc.TempUnschedulableUntil)
		if st.TempUnschedReason == "" {
			st.TempUnschedReason = strings.TrimSpace(acc.TempUnschedulableReason)
		}
	}
}

// collectEmptyPoolDiagnostics 在 listSchedulableAccounts 返回空时，
// 对活跃账号做诊断分类，返回限流详情统计。
func collectEmptyPoolDiagnostics(allAccounts []Account, platform string) accountFilterStats {
	var st accountFilterStats
	for i := range allAccounts {
		acc := &allAccounts[i]
		if !acc.IsActive() {
			continue
		}
		if platform != "" && acc.Platform != platform {
			continue
		}
		st.Total++
		if !acc.IsSchedulable() {
			st.Unschedulable++
			classifyUnschedulableAccount(acc, &st)
		}
	}
	return st
}
