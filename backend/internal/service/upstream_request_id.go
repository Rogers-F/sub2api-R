package service

import (
	"strings"

	"github.com/gin-gonic/gin"
)

const upstreamRequestIDContextKey = "upstream_request_id"

func SetUpstreamRequestID(c *gin.Context, requestID string) {
	if c == nil {
		return
	}
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return
	}
	c.Set(upstreamRequestIDContextKey, requestID)
}

func GetUpstreamRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v, ok := c.Get(upstreamRequestIDContextKey); ok {
		if requestID, ok := v.(string); ok {
			return strings.TrimSpace(requestID)
		}
	}
	return ""
}
