package claude

import "strings"

const ThinkingDisplaySummarized = "summarized"

func IsOpus47Model(model string) bool {
	value := strings.ToLower(strings.TrimSpace(model))
	return value == "claude-opus-4-7" || strings.HasPrefix(value, "claude-opus-4-7-")
}

func IsClaudeAdaptiveThinkingModel(model string) bool {
	value := strings.ToLower(strings.TrimSpace(model))
	return value == "claude-opus-4" ||
		strings.HasPrefix(value, "claude-opus-4-") ||
		value == "claude-sonnet-4" ||
		strings.HasPrefix(value, "claude-sonnet-4-")
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
	return ParseClaudeThinkingAliasModel(model)
}

func ParseClaudeThinkingAliasModel(model string) (baseModel, effort string, thinkingAlias bool, ok bool) {
	value := strings.ToLower(strings.TrimSpace(model))
	if value == "" {
		return "", "", false, false
	}

	suffixes := []struct {
		suffix        string
		effort        string
		thinkingAlias bool
	}{
		{suffix: "-thinking", effort: "high", thinkingAlias: true},
		{suffix: "-xhigh", effort: "max"},
		{suffix: "-max", effort: "max"},
		{suffix: "-high", effort: "high"},
		{suffix: "-medium", effort: "medium"},
		{suffix: "-low", effort: "low"},
	}
	for _, item := range suffixes {
		if strings.HasSuffix(value, item.suffix) {
			base := strings.TrimSuffix(value, item.suffix)
			if IsClaudeAdaptiveThinkingModel(base) {
				return base, item.effort, item.thinkingAlias, true
			}
		}
	}

	return "", "", false, false
}
