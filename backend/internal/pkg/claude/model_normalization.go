package claude

import "strings"

const ThinkingDisplaySummarized = "summarized"

func IsOpus47Model(model string) bool {
	value := strings.ToLower(strings.TrimSpace(model))
	return value == "claude-opus-4-7" || strings.HasPrefix(value, "claude-opus-4-7-")
}

func NormalizeOutputEffort(raw string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "low", "medium", "high", "max":
		return value, true
	case "xhigh":
		return "max", true
	default:
		return "", false
	}
}

func ParseOpus47AliasModel(model string) (baseModel, effort string, thinkingAlias bool, ok bool) {
	value := strings.ToLower(strings.TrimSpace(model))
	if value == "" {
		return "", "", false, false
	}

	if value == "claude-opus-4-7-thinking" {
		return "claude-opus-4-7", "high", true, true
	}

	suffixes := []struct {
		suffix string
		effort string
	}{
		{suffix: "-xhigh", effort: "max"},
		{suffix: "-max", effort: "max"},
		{suffix: "-high", effort: "high"},
		{suffix: "-medium", effort: "medium"},
		{suffix: "-low", effort: "low"},
	}
	for _, item := range suffixes {
		if strings.HasSuffix(value, item.suffix) {
			base := strings.TrimSuffix(value, item.suffix)
			if base == "claude-opus-4-7" {
				return base, item.effort, false, true
			}
		}
	}

	return "", "", false, false
}
