package openai

import (
	"strings"
	"testing"
)

func TestMatchCodexPresetMatchesPrettyAPIRules(t *testing.T) {
	tests := []struct {
		model     string
		wantFile  string
		wantMatch bool
	}{
		{"gpt-5.5", "gpt-5.4_prompt.md", true},
		{"gpt-5.5-pro", "gpt-5.4_prompt.md", true},
		{"gpt-5.3-codex", "gpt-5.3-codex_prompt.md", true},
		{"gpt-5.2-codex", "gpt-5.2-codex_prompt.md", true},
		{"bengalfox-2024", "gpt-5.2-codex_prompt.md", true},
		{"gpt-5.1-codex-max", "gpt-5.1-codex-max_prompt.md", true},
		{"gpt-5-codex", "gpt_5_codex_prompt.md", true},
		{"codex-mini-latest", "gpt_5_codex_prompt.md", true},
		{"gpt-5.2", "gpt_5_2_prompt.md", true},
		{"boomslang", "gpt_5_2_prompt.md", true},
		{"gpt-5.1", "gpt_5_1_prompt.md", true},
		{"gpt-4o", "", false},
		{"", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			gotFile, gotMatch := MatchCodexPreset(tt.model)
			if gotMatch != tt.wantMatch {
				t.Fatalf("match = %v, want %v", gotMatch, tt.wantMatch)
			}
			if gotFile != tt.wantFile {
				t.Fatalf("file = %q, want %q", gotFile, tt.wantFile)
			}
		})
	}
}

func TestGetInstructionsForModelMatchesPrettyAPIRules(t *testing.T) {
	instructions, ok := GetInstructionsForModel("gpt-5.5")
	if !ok {
		t.Fatal("expected gpt-5.5 to match")
	}
	if !strings.Contains(instructions, "GPT-5") {
		t.Fatalf("expected gpt-5.5 instructions to contain GPT-5")
	}

	instructions, ok = GetInstructionsForModel("gpt-4o")
	if ok || instructions != "" {
		t.Fatalf("expected gpt-4o to be unmatched")
	}
}
