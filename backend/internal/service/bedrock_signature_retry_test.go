package service

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGatewayService_ExecuteBedrockUpstream_RetriesThinkingSignatureErrorWithFilteredBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := context.WithValue(context.Background(), ctxkey.Group, &Group{
		ID:                                    901,
		Platform:                              PlatformAnthropic,
		Status:                                StatusActive,
		Hydrated:                              true,
		ThinkingSignatureCompatEnabled:        true,
		BedrockThinkingSignatureCompatEnabled: true,
		StrongSafetyModeEnabled:               true,
		ClaudePromptCachingEnabled:            true,
		ClaudeToolUseRepairEnabled:            true,
		ForceApplicationJSONForNonStream:      false,
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	body := []byte(`{
		"anthropic_version":"bedrock-2023-05-31",
		"thinking":{"type":"enabled","budget_tokens":1024},
		"messages":[
			{"role":"user","content":[{"type":"text","text":"hello"}]},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"private","signature":"bad-signature"},
				{"type":"text","text":"visible"}
			]},
			{"role":"user","content":[{"type":"text","text":"again"}]}
		]
	}`)

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"x-amzn-requestid": []string{"bedrock-rid-1"}},
				Body: io.NopCloser(strings.NewReader(`{
					"message":"ValidationException: Thinking signature verification failed: the signature on a thinking block in messages[1].content is invalid."
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"id":"msg_1","type":"message","usage":{"input_tokens":1,"output_tokens":1}}`))),
			},
		},
	}

	svc := &GatewayService{
		cfg:          &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          501,
		Name:        "bedrock-signature-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeBedrock,
		Concurrency: 1,
		Credentials: map[string]any{
			"auth_mode": "apikey",
			"api_key":   "bedrock-key",
			"region":    "us-east-1",
		},
	}

	resp, err := svc.executeBedrockUpstream(ctx, c, account, body, "us.anthropic.claude-sonnet-4-6", "us-east-1", false, nil, "bedrock-key", "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, upstream.requestBodies, 2)
	require.Contains(t, string(upstream.requestBodies[0]), `"type":"thinking"`)

	retryBody := string(upstream.requestBodies[1])
	require.NotContains(t, retryBody, `"type":"thinking"`)
	require.NotContains(t, retryBody, `"bad-signature"`)
	require.NotContains(t, retryBody, `"thinking":{"type":"enabled"`)
	require.Contains(t, retryBody, `"text":"visible"`)
}

func TestGatewayService_ExecuteBedrockUpstream_DoesNotRetryThinkingSignatureErrorWithoutBedrockSwitch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctx := context.WithValue(context.Background(), ctxkey.Group, &Group{
		ID:                             901,
		Platform:                       PlatformAnthropic,
		Status:                         StatusActive,
		Hydrated:                       true,
		ThinkingSignatureCompatEnabled: true,
		StrongSafetyModeEnabled:        true,
		ClaudePromptCachingEnabled:     true,
	})

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	body := []byte(`{
		"anthropic_version":"bedrock-2023-05-31",
		"thinking":{"type":"enabled","budget_tokens":1024},
		"messages":[
			{"role":"user","content":[{"type":"text","text":"hello"}]},
			{"role":"assistant","content":[
				{"type":"thinking","thinking":"private","signature":"bad-signature"},
				{"type":"text","text":"visible"}
			]}
		]
	}`)

	upstream := &queuedHTTPUpstreamStub{
		responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Header:     http.Header{"x-amzn-requestid": []string{"bedrock-rid-1"}},
				Body: io.NopCloser(strings.NewReader(`{
					"message":"ValidationException: Thinking signature verification failed: the signature on a thinking block in messages[1].content is invalid."
				}`)),
			},
		},
	}

	svc := &GatewayService{
		cfg:          &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}},
		httpUpstream: upstream,
	}
	account := &Account{
		ID:          501,
		Name:        "bedrock-signature-test",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeBedrock,
		Concurrency: 1,
		Credentials: map[string]any{
			"auth_mode": "apikey",
			"api_key":   "bedrock-key",
			"region":    "us-east-1",
		},
	}

	resp, err := svc.executeBedrockUpstream(ctx, c, account, body, "us.anthropic.claude-sonnet-4-6", "us-east-1", false, nil, "bedrock-key", "")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
	require.Len(t, upstream.requestBodies, 1)
}
