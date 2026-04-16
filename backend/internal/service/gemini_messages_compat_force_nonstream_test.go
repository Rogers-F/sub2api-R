package service

import (
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

func TestGeminiMessagesCompatServiceForward_GroupFlagForcesNonStreaming(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.Group, &Group{
		ID:                               401,
		Name:                             "gemini-force-json",
		Platform:                         PlatformGemini,
		Status:                           StatusActive,
		Hydrated:                         true,
		ForceApplicationJSONForNonStream: true,
	}))
	c.Request = req

	httpStub := &geminiCompatHTTPUpstreamStub{
		response: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}, "x-request-id": []string{"gemini-force-json-1"}},
			Body: io.NopCloser(strings.NewReader(
				`{"candidates":[{"content":{"parts":[{"text":"hello"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5}}`,
			)),
		},
	}

	svc := &GeminiMessagesCompatService{
		cfg:          &config.Config{},
		httpUpstream: httpStub,
	}
	account := &Account{
		ID:   1,
		Name: "gemini-apikey",
		Type: AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "test-key",
		},
	}

	body := []byte(`{"model":"gemini-2.5-pro","stream":true,"max_tokens":32,"messages":[{"role":"user","content":"hello"}]}`)
	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.False(t, result.Stream)
	require.NotNil(t, httpStub.lastReq)
	require.Contains(t, httpStub.lastReq.URL.String(), ":generateContent")
	require.NotContains(t, httpStub.lastReq.URL.String(), "streamGenerateContent")
	require.NotContains(t, httpStub.lastReq.URL.String(), "alt=sse")
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}
