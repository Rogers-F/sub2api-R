package middleware

import (
	"log/slog"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/gin-gonic/gin"
)

// Logger 请求日志中间件
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		c.Next()
		latency := time.Since(startTime)

		requestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string)

		attrs := []any{
			"client_request_id", requestID,
			"status", c.Writer.Status(),
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, "error_count", len(c.Errors), "errors", c.Errors.String())
		}
		slog.Info("http_request", attrs...)
	}
}
