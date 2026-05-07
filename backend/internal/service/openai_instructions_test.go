package service

import (
	"testing"

	openaipkg "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/stretchr/testify/require"
)

func TestOpenAIInstructionsForRequest_UsesPrettyAPIEnableCodexPresetSwitch(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Extra: map[string]any{
			"enable_codex_preset": true,
		},
	}

	gpt55Instructions, ok := openaipkg.GetInstructionsForModel("gpt-5.5")
	require.True(t, ok)
	gpt53CodexInstructions, ok := openaipkg.GetInstructionsForModel("gpt-5.3-codex")
	require.True(t, ok)

	require.Equal(t, gpt55Instructions, openAIInstructionsForRequest(account, "gpt-5.5", "gpt-5.5"))
	require.Equal(t, gpt53CodexInstructions, openAIInstructionsForRequest(account, "gpt-5.3-codex", "gpt-5.3-codex"))
}

func TestOpenAIInstructionsForRequest_DoesNotUseCodexPresetForNonMatchingModel(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Extra: map[string]any{
			"enable_codex_preset": true,
		},
	}

	require.Equal(t, openAIDefaultCompatInstructions, openAIInstructionsForRequest(account, "gpt-4.1", "gpt-4.1"))
}

func TestOpenAIInstructionsForRequest_KeepsLegacySwitchAsAlias(t *testing.T) {
	account := &Account{
		Platform: PlatformOpenAI,
		Extra: map[string]any{
			"openai_codex_preset_instructions": true,
		},
	}

	instructions, ok := openaipkg.GetInstructionsForModel("gpt-5.5")
	require.True(t, ok)
	require.Equal(t, instructions, openAIInstructionsForRequest(account, "gpt-5.5", "gpt-5.5"))
}
