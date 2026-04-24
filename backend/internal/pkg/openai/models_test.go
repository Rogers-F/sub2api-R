package openai

import "testing"

func TestDefaultModels_ContainsGPTImage2(t *testing.T) {
	t.Parallel()

	byID := make(map[string]Model, len(DefaultModels))
	for _, model := range DefaultModels {
		byID[model.ID] = model
	}

	model, ok := byID["gpt-image-2"]
	if !ok {
		t.Fatalf("expected curated OpenAI model %q to exist", "gpt-image-2")
	}
	if model.DisplayName == "" {
		t.Fatalf("expected curated OpenAI model %q to have a display name", "gpt-image-2")
	}
}
