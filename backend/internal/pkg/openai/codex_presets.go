package openai

import (
	"embed"
	"strings"
)

//go:embed codex_prompts/*.md
var codexPresetFiles embed.FS

type codexPresetRule struct {
	Prefixes []string
	File     string
}

var codexPresetRules = []codexPresetRule{
	{Prefixes: []string{"gpt-5.4"}, File: "gpt-5.4_prompt.md"},
	{Prefixes: []string{"gpt-5.5-pro"}, File: "gpt-5.4_prompt.md"},
	{Prefixes: []string{"gpt-5.5"}, File: "gpt-5.4_prompt.md"},
	{Prefixes: []string{"gpt-5.3-codex"}, File: "gpt-5.3-codex_prompt.md"},
	{Prefixes: []string{"gpt-5.2-codex", "bengalfox"}, File: "gpt-5.2-codex_prompt.md"},
	{Prefixes: []string{"gpt-5.1-codex-max"}, File: "gpt-5.1-codex-max_prompt.md"},
	{Prefixes: []string{"gpt-5-codex", "gpt-5.1-codex", "codex-"}, File: "gpt_5_codex_prompt.md"},
	{Prefixes: []string{"gpt-5.2", "boomslang"}, File: "gpt_5_2_prompt.md"},
	{Prefixes: []string{"gpt-5.1"}, File: "gpt_5_1_prompt.md"},
}

const codexFallbackPresetFile = "prompt.md"

// MatchCodexPreset returns the pretty-api compatible preset filename for model.
func MatchCodexPreset(model string) (string, bool) {
	modelLower := strings.ToLower(strings.TrimSpace(model))
	if modelLower == "" {
		return "", false
	}
	if strings.Contains(modelLower, "/") {
		parts := strings.Split(modelLower, "/")
		modelLower = parts[len(parts)-1]
	}
	for _, rule := range codexPresetRules {
		for _, prefix := range rule.Prefixes {
			if strings.HasPrefix(modelLower, prefix) {
				return rule.File, true
			}
		}
	}
	return "", false
}

// GetInstructionsForModel returns Codex preset instructions for a matching model.
func GetInstructionsForModel(model string) (string, bool) {
	presetFile, ok := MatchCodexPreset(model)
	if !ok {
		return "", false
	}
	data, err := codexPresetFiles.ReadFile("codex_prompts/" + presetFile)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func GetFallbackInstructions() (string, bool) {
	data, err := codexPresetFiles.ReadFile("codex_prompts/" + codexFallbackPresetFile)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func IsCodexPresetModel(model string) bool {
	modelLower := strings.ToLower(strings.TrimSpace(model))
	if modelLower == "" {
		return false
	}
	if strings.Contains(modelLower, "/") {
		parts := strings.Split(modelLower, "/")
		modelLower = parts[len(parts)-1]
	}
	for _, prefix := range []string{"gpt-5", "codex-", "bengalfox", "boomslang"} {
		if strings.HasPrefix(modelLower, prefix) {
			return true
		}
	}
	return false
}
