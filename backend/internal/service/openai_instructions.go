package service

import (
	"strings"

	openaipkg "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

const openAIDefaultCompatInstructions = "You are a helpful coding assistant."

type codexOAuthTransformOptions struct {
	UseCodexPresetInstructions bool
}

func applyOpenAIDefaultInstructions(reqBody map[string]any, account *Account, requestedModel string, mappedModel string) bool {
	if !isInstructionsEmpty(reqBody) {
		return false
	}
	reqBody["instructions"] = openAIInstructionsForRequest(account, requestedModel, mappedModel)
	return true
}

func openAIInstructionsForRequest(account *Account, requestedModel string, mappedModel string) string {
	if shouldUseOpenAICodexPresetInstructions(account, requestedModel, mappedModel) {
		if instructions, ok := openAICodexPresetInstructionsForModels(requestedModel, mappedModel); ok {
			return instructions
		}
	}
	return openAIDefaultCompatInstructions
}

func shouldUseOpenAICodexPresetInstructions(account *Account, requestedModel string, mappedModel string) bool {
	return account != nil &&
		account.IsOpenAICodexPresetEnabled() &&
		isOpenAICodexPresetInstructionsModel(requestedModel, mappedModel)
}

func isOpenAICodexPresetInstructionsModel(models ...string) bool {
	_, ok := openAICodexPresetInstructionsForModels(models...)
	return ok
}

func openAICodexPresetInstructionsForModels(models ...string) (string, bool) {
	for _, model := range models {
		model = normalizeModelIDForInstructionMatch(model)
		if model == "" {
			continue
		}
		if instructions, ok := openaipkg.GetInstructionsForModel(model); ok && strings.TrimSpace(instructions) != "" {
			return instructions, true
		}
	}
	return "", false
}

func normalizeModelIDForInstructionMatch(model string) string {
	model = strings.TrimSpace(strings.ToLower(model))
	if model == "" {
		return ""
	}
	if strings.Contains(model, "/") {
		parts := strings.Split(model, "/")
		model = parts[len(parts)-1]
	}
	return model
}

func removeCodexPresetChatSystemInput(reqBody map[string]any) bool {
	input, ok := reqBody["input"].([]any)
	if !ok || len(input) == 0 {
		return false
	}

	filtered := make([]any, 0, len(input))
	removed := false
	for _, item := range input {
		m, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		role, _ := m["role"].(string)
		role = strings.ToLower(strings.TrimSpace(role))
		if role == "system" || role == "developer" {
			removed = true
			continue
		}
		filtered = append(filtered, item)
	}
	if removed {
		reqBody["input"] = filtered
	}
	return removed
}
