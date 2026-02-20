package service

import (
	"fmt"
	"strings"
	"time"
)

const (
	rateLimit5hBase      = 5 * time.Hour
	rateLimit7dBase      = 7 * 24 * time.Hour
	rateLimit5hTolerance = 20 * time.Minute
	rateLimit7dTolerance = 6 * time.Hour
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
		switch inferRateLimitWindowType(acc.RateLimitedAt, acc.RateLimitResetAt) {
		case "5h window":
			st.RateLimited5h++
		case "7d window":
			st.RateLimited7d++
		default:
			st.RateLimitedOther++
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
