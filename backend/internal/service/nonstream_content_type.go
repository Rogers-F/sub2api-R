package service

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

const applicationJSONContentType = "application/json; charset=utf-8"

func shouldForceApplicationJSONForNonStream(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	return ForceApplicationJSONForNonStreamFromContext(c.Request.Context())
}

func resolveClientStreamingPreference(c *gin.Context, requested bool) bool {
	if shouldForceApplicationJSONForNonStream(c) {
		return false
	}
	return requested
}

func enforceNonStreamingRequestBody(c *gin.Context, body []byte) ([]byte, bool, error) {
	if !shouldForceApplicationJSONForNonStream(c) || len(body) == 0 {
		return body, false, nil
	}

	stream := gjson.GetBytes(body, "stream")
	if !stream.Exists() || stream.Type == gjson.False {
		return body, false, nil
	}

	next, err := sjson.SetBytes(body, "stream", false)
	if err != nil {
		return body, false, err
	}
	return next, true, nil
}

func applyForcedApplicationJSONHeaderForNonStream(c *gin.Context) {
	if shouldForceApplicationJSONForNonStream(c) {
		c.Writer.Header().Set("Content-Type", applicationJSONContentType)
	}
}

func resolveNonStreamJSONContentType(c *gin.Context, upstreamType, fallback string) string {
	if shouldForceApplicationJSONForNonStream(c) {
		return applicationJSONContentType
	}
	if trimmed := strings.TrimSpace(upstreamType); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(fallback); trimmed != "" {
		return trimmed
	}
	return applicationJSONContentType
}
