package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestHandleChatBufferedStreamingResponse_GroupFlagForcesApplicationJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.Group, &Group{
		ID:                               101,
		Name:                             "chat-group",
		Platform:                         PlatformOpenAI,
		Status:                           StatusActive,
		Hydrated:                         true,
		ForceApplicationJSONForNonStream: true,
	}))
	c.Request = req

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: ioNopCloserString(strings.Join([]string{
			`data: {"type":"response.completed","response":{"id":"resp_chat_1","model":"gpt-5.4","usage":{"input_tokens":7,"output_tokens":9,"input_tokens_details":{"cached_tokens":1}}}}`,
			`data: [DONE]`,
		}, "\n")),
	}

	svc := &OpenAIGatewayService{}
	result, err := svc.handleChatBufferedStreamingResponse(resp, c, "gpt-5.4", "gpt-5.4", time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

func TestHandleAnthropicBufferedStreamingResponse_GroupFlagForcesApplicationJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.Group, &Group{
		ID:                               202,
		Name:                             "claude-group",
		Platform:                         PlatformAnthropic,
		Status:                           StatusActive,
		Hydrated:                         true,
		ForceApplicationJSONForNonStream: true,
	}))
	c.Request = req

	resp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body: ioNopCloserString(strings.Join([]string{
			`data: {"type":"response.completed","response":{"id":"resp_msg_1","model":"gpt-5.4","usage":{"input_tokens":5,"output_tokens":6,"input_tokens_details":{"cached_tokens":0}}}}`,
			`data: [DONE]`,
		}, "\n")),
	}

	svc := &OpenAIGatewayService{}
	result, err := svc.handleAnthropicBufferedStreamingResponse(resp, c, "claude-opus-4-6", "gpt-5.4", time.Now())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

func ioNopCloserString(body string) *readCloserString {
	return &readCloserString{Reader: strings.NewReader(body)}
}

type readCloserString struct {
	*strings.Reader
}

func (r *readCloserString) Close() error {
	return nil
}
