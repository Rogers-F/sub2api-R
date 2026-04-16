package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const applicationJSONContentType = "application/json; charset=utf-8"

func shouldForceApplicationJSONForNonStream(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	return ForceApplicationJSONForNonStreamFromContext(c.Request.Context())
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
