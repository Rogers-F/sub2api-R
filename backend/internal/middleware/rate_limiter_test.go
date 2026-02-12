package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestWindowTTLMillis(t *testing.T) {
	require.Equal(t, int64(1), windowTTLMillis(500*time.Microsecond))
	require.Equal(t, int64(1), windowTTLMillis(1500*time.Microsecond))
	require.Equal(t, int64(2), windowTTLMillis(2500*time.Microsecond))
}

func TestRateLimiterFailureModes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  50 * time.Millisecond,
		ReadTimeout:  50 * time.Millisecond,
		WriteTimeout: 50 * time.Millisecond,
	})
	t.Cleanup(func() {
		_ = rdb.Close()
	})

	limiter := NewRateLimiter(rdb)

	// fail-open: Redis 故障时放行，不写限流头
	failOpenRouter := gin.New()
	failOpenRouter.Use(limiter.Limit("test", 1, time.Second))
	failOpenRouter.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	recorder := httptest.NewRecorder()
	failOpenRouter.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Header().Get("X-RateLimit-Limit"), "fail-open should not write rate limit headers")

	// fail-close: Redis 故障时返回 429，写降级头
	failCloseRouter := gin.New()
	failCloseRouter.Use(limiter.LimitWithOptions("test", 1, time.Second, RateLimitOptions{
		FailureMode: RateLimitFailClose,
	}))
	failCloseRouter.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	recorder = httptest.NewRecorder()
	failCloseRouter.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "1", recorder.Header().Get("X-RateLimit-Limit"), "fail-close should write degraded headers")
	require.Equal(t, "0", recorder.Header().Get("X-RateLimit-Remaining"))
	require.NotEmpty(t, recorder.Header().Get("Retry-After"), "fail-close should include Retry-After")
}

func TestRateLimiterSuccessAndLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalRun := rateLimitRun
	counts := []int64{1, 2}
	callIndex := 0
	rateLimitRun = func(ctx context.Context, client *redis.Client, key string, windowMillis int64) (int64, bool, int64, error) {
		if callIndex >= len(counts) {
			return counts[len(counts)-1], false, 30000, nil
		}
		value := counts[callIndex]
		callIndex++
		return value, false, 30000, nil
	}
	t.Cleanup(func() {
		rateLimitRun = originalRun
	})

	limiter := NewRateLimiter(redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"}))

	router := gin.New()
	router.Use(limiter.Limit("test", 1, time.Second))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	// 第一次请求：放行，写限流头
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, "1", recorder.Header().Get("RateLimit-Limit"))
	require.Equal(t, "0", recorder.Header().Get("RateLimit-Remaining"))
	require.NotEmpty(t, recorder.Header().Get("RateLimit-Reset"))
	require.Equal(t, "1", recorder.Header().Get("X-RateLimit-Limit"))
	require.Equal(t, "0", recorder.Header().Get("X-RateLimit-Remaining"))
	require.NotEmpty(t, recorder.Header().Get("X-RateLimit-Reset"))
	require.Empty(t, recorder.Header().Get("Retry-After"), "successful request should not have Retry-After")

	// 第二次请求：超限，返回 429 + Retry-After
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	require.Equal(t, http.StatusTooManyRequests, recorder.Code)
	require.Equal(t, "1", recorder.Header().Get("RateLimit-Limit"))
	require.Equal(t, "0", recorder.Header().Get("RateLimit-Remaining"))
	require.NotEmpty(t, recorder.Header().Get("Retry-After"), "429 response must include Retry-After")
	require.NotEmpty(t, recorder.Header().Get("X-RateLimit-Reset"))
}

func TestCeilDiv(t *testing.T) {
	require.Equal(t, int64(1), ceilDiv(1, 1000))
	require.Equal(t, int64(1), ceilDiv(999, 1000))
	require.Equal(t, int64(1), ceilDiv(1000, 1000))
	require.Equal(t, int64(2), ceilDiv(1001, 1000))
	require.Equal(t, int64(60), ceilDiv(60000, 1000))
}
