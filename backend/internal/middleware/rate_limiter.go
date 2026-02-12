package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitFailureMode Redis 故障策略
type RateLimitFailureMode int

const (
	RateLimitFailOpen RateLimitFailureMode = iota
	RateLimitFailClose
)

// RateLimitOptions 限流可选配置
type RateLimitOptions struct {
	FailureMode RateLimitFailureMode
}

var rateLimitScript = redis.NewScript(`
local current = redis.call('INCR', KEYS[1])
local ttl = redis.call('PTTL', KEYS[1])
local repaired = 0
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
  ttl = tonumber(ARGV[1])
elseif ttl == -1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
  ttl = tonumber(ARGV[1])
  repaired = 1
end
return {current, repaired, ttl}
`)

// rateLimitRun 允许测试覆写脚本执行逻辑
var rateLimitRun = func(ctx context.Context, client *redis.Client, key string, windowMillis int64) (int64, bool, int64, error) {
	values, err := rateLimitScript.Run(ctx, client, []string{key}, windowMillis).Slice()
	if err != nil {
		return 0, false, 0, err
	}
	if len(values) < 3 {
		return 0, false, 0, fmt.Errorf("rate limit script returned %d values", len(values))
	}
	count, err := parseInt64(values[0])
	if err != nil {
		return 0, false, 0, err
	}
	repaired, err := parseInt64(values[1])
	if err != nil {
		return 0, false, 0, err
	}
	ttlMs, err := parseInt64(values[2])
	if err != nil {
		return 0, false, 0, err
	}
	return count, repaired == 1, ttlMs, nil
}

// RateLimiter Redis 速率限制器
type RateLimiter struct {
	redis  *redis.Client
	prefix string
}

// NewRateLimiter 创建速率限制器实例
func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	return &RateLimiter{
		redis:  redisClient,
		prefix: "rate_limit:",
	}
}

// Limit 返回速率限制中间件
// key: 限制类型标识
// limit: 时间窗口内最大请求数
// window: 时间窗口
func (r *RateLimiter) Limit(key string, limit int, window time.Duration) gin.HandlerFunc {
	return r.LimitWithOptions(key, limit, window, RateLimitOptions{})
}

// LimitWithOptions 返回速率限制中间件（带可选配置）
func (r *RateLimiter) LimitWithOptions(key string, limit int, window time.Duration, opts RateLimitOptions) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		redisKey := r.prefix + key + ":" + ip

		ctx := c.Request.Context()

		windowMillis := windowTTLMillis(window)

		// 使用 Lua 脚本原子操作增加计数并设置过期
		count, repaired, ttlMs, err := rateLimitRun(ctx, r.redis, redisKey, windowMillis)
		if err != nil {
			log.Printf("[RateLimit] redis error: key=%s mode=%s err=%v", redisKey, failureModeLabel(opts.FailureMode), err)
			if opts.FailureMode == RateLimitFailClose {
				// fail-close: 写降级头（remaining=0, reset=window）
				resetSec := ceilDiv(windowMillis, 1000)
				writeRateLimitHeaders(c, int64(limit), 0, resetSec, true)
				abortRateLimit(c)
				return
			}
			// fail-open: 放行且不写限流头（避免伪数据）
			c.Next()
			return
		}
		if repaired {
			log.Printf("[RateLimit] ttl repaired: key=%s window_ms=%d", redisKey, windowMillis)
		}

		remaining := int64(limit) - count
		if remaining < 0 {
			remaining = 0
		}
		resetSec := ceilDiv(ttlMs, 1000)
		if resetSec < 1 {
			resetSec = 1
		}

		// 超过限制
		if count > int64(limit) {
			writeRateLimitHeaders(c, int64(limit), remaining, resetSec, true)
			abortRateLimit(c)
			return
		}

		// 正常请求也写限流头，让客户端提前感知剩余配额
		writeRateLimitHeaders(c, int64(limit), remaining, resetSec, false)
		c.Next()
	}
}

// writeRateLimitHeaders 写入标准限流响应头
// 同时输出 RateLimit-* (IETF draft) 和 X-RateLimit-* (de facto) 两套头
func writeRateLimitHeaders(c *gin.Context, limit, remaining, resetSec int64, includeRetryAfter bool) {
	limitStr := strconv.FormatInt(limit, 10)
	remainingStr := strconv.FormatInt(remaining, 10)
	resetStr := strconv.FormatInt(resetSec, 10)
	resetUnix := strconv.FormatInt(time.Now().Unix()+resetSec, 10)

	// IETF draft-ietf-httpapi-ratelimit-headers (delta seconds)
	c.Header("RateLimit-Limit", limitStr)
	c.Header("RateLimit-Remaining", remainingStr)
	c.Header("RateLimit-Reset", resetStr)

	// De facto standard (Unix timestamp)
	c.Header("X-RateLimit-Limit", limitStr)
	c.Header("X-RateLimit-Remaining", remainingStr)
	c.Header("X-RateLimit-Reset", resetUnix)

	if includeRetryAfter {
		c.Header("Retry-After", resetStr)
	}
}

func windowTTLMillis(window time.Duration) int64 {
	ttl := window.Milliseconds()
	if ttl < 1 {
		return 1
	}
	return ttl
}

func ceilDiv(a, b int64) int64 {
	return (a + b - 1) / b
}

func abortRateLimit(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error":   "rate limit exceeded",
		"message": "Too many requests, please try again later",
	})
}

func failureModeLabel(mode RateLimitFailureMode) string {
	if mode == RateLimitFailClose {
		return "fail-close"
	}
	return "fail-open"
}

func parseInt64(value any) (int64, error) {
	switch v := value.(type) {
	case int64:
		return v, nil
	case int:
		return int64(v), nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		return 0, fmt.Errorf("unexpected value type %T", value)
	}
}
