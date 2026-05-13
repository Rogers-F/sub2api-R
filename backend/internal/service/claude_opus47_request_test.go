package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNormalizeClaudeOpus47RequestBody_EffortSuffix(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-7-xhigh","temperature":0.7,"top_p":0.8,"top_k":32,"messages":[]}`)

	normalized, modelID, changed, err := normalizeClaudeOpus47RequestBody(body, "claude-opus-4-7-xhigh")
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, "claude-opus-4-7", modelID)
	assert.Equal(t, "claude-opus-4-7", gjson.GetBytes(normalized, "model").String())
	assert.Equal(t, "adaptive", gjson.GetBytes(normalized, "thinking.type").String())
	assert.Equal(t, "summarized", gjson.GetBytes(normalized, "thinking.display").String())
	assert.Equal(t, "max", gjson.GetBytes(normalized, "output_config.effort").String())
	assert.False(t, gjson.GetBytes(normalized, "temperature").Exists())
	assert.False(t, gjson.GetBytes(normalized, "top_p").Exists())
	assert.False(t, gjson.GetBytes(normalized, "top_k").Exists())
}

func TestNormalizeClaudeOpus47RequestBody_EnabledThinkingAndXHighAreNormalized(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-7","temperature":1,"top_p":0.9,"top_k":64,"thinking":{"type":"enabled","budget_tokens":2048},"output_config":{"effort":"xhigh"},"messages":[]}`)

	normalized, modelID, changed, err := normalizeClaudeOpus47RequestBody(body, "claude-opus-4-7")
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, "claude-opus-4-7", modelID)
	assert.Equal(t, "claude-opus-4-7", gjson.GetBytes(normalized, "model").String())
	assert.Equal(t, "adaptive", gjson.GetBytes(normalized, "thinking.type").String())
	assert.Equal(t, "summarized", gjson.GetBytes(normalized, "thinking.display").String())
	assert.Equal(t, "max", gjson.GetBytes(normalized, "output_config.effort").String())
	assert.False(t, gjson.GetBytes(normalized, "thinking.budget_tokens").Exists())
	assert.False(t, gjson.GetBytes(normalized, "temperature").Exists())
	assert.False(t, gjson.GetBytes(normalized, "top_p").Exists())
	assert.False(t, gjson.GetBytes(normalized, "top_k").Exists())
}

func TestNormalizeClaudeOpus47RequestBody_ClaudeThinkingSuffixMapsToAdaptiveHigh(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-6-thinking","temperature":0.7,"messages":[]}`)

	normalized, modelID, changed, err := normalizeClaudeOpus47RequestBody(body, "claude-sonnet-4-6-thinking")
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, "claude-sonnet-4-6", modelID)
	assert.Equal(t, "claude-sonnet-4-6", gjson.GetBytes(normalized, "model").String())
	assert.Equal(t, "adaptive", gjson.GetBytes(normalized, "thinking.type").String())
	assert.Equal(t, "summarized", gjson.GetBytes(normalized, "thinking.display").String())
	assert.Equal(t, "high", gjson.GetBytes(normalized, "output_config.effort").String())
	assert.False(t, gjson.GetBytes(normalized, "temperature").Exists())
}

func TestNormalizeClaudeOpus47RequestBody_ClaudeEffortSuffixMapsToAdaptiveEffort(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-6-medium","thinking":{"type":"enabled","budget_tokens":4096},"top_p":0.8,"messages":[]}`)

	normalized, modelID, changed, err := normalizeClaudeOpus47RequestBody(body, "claude-opus-4-6-medium")
	require.NoError(t, err)
	require.True(t, changed)
	assert.Equal(t, "claude-opus-4-6", modelID)
	assert.Equal(t, "claude-opus-4-6", gjson.GetBytes(normalized, "model").String())
	assert.Equal(t, "adaptive", gjson.GetBytes(normalized, "thinking.type").String())
	assert.Equal(t, "summarized", gjson.GetBytes(normalized, "thinking.display").String())
	assert.Equal(t, "medium", gjson.GetBytes(normalized, "output_config.effort").String())
	assert.False(t, gjson.GetBytes(normalized, "thinking.budget_tokens").Exists())
	assert.False(t, gjson.GetBytes(normalized, "top_p").Exists())
}
