package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestResolveClientStreamingPreference_GroupFlagForcesNonStream(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.Group, &Group{
		ID:                               301,
		Name:                             "force-json-group",
		Platform:                         PlatformAnthropic,
		Status:                           StatusActive,
		Hydrated:                         true,
		ForceApplicationJSONForNonStream: true,
	}))
	c.Request = req

	require.False(t, resolveClientStreamingPreference(c, true))
	require.False(t, resolveClientStreamingPreference(c, false))
}

func TestEnforceNonStreamingRequestBody_GroupFlagSetsStreamFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	req = req.WithContext(context.WithValue(req.Context(), ctxkey.Group, &Group{
		ID:                               302,
		Name:                             "force-json-group",
		Platform:                         PlatformOpenAI,
		Status:                           StatusActive,
		Hydrated:                         true,
		ForceApplicationJSONForNonStream: true,
	}))
	c.Request = req

	body := []byte(`{"model":"gpt-5.2","stream":true,"input":"hello"}`)
	next, changed, err := enforceNonStreamingRequestBody(c, body)
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, gjson.GetBytes(next, "stream").Bool())
}
