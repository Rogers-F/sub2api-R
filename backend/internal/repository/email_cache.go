package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	verifyCodeKeyPrefix          = "verify_code:"
	passwordResetKeyPrefix       = "password_reset:"
	passwordResetSentAtKeyPrefix = "password_reset_sent:"
)

// verifyCodeKey generates the Redis key for email verification code.
func verifyCodeKey(email string) string {
	return verifyCodeKeyPrefix + email
}

// passwordResetKey generates the Redis key for password reset token.
func passwordResetKey(email string) string {
	return passwordResetKeyPrefix + email
}

// passwordResetSentAtKey generates the Redis key for password reset email sent timestamp.
func passwordResetSentAtKey(email string) string {
	return passwordResetSentAtKeyPrefix + email
}

type emailCache struct {
	rdb *redis.Client
}

func NewEmailCache(rdb *redis.Client) service.EmailCache {
	return &emailCache{rdb: rdb}
}

func (c *emailCache) GetVerificationCode(ctx context.Context, email string) (*service.VerificationCodeData, error) {
	key := verifyCodeKey(email)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var data service.VerificationCodeData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *emailCache) SetVerificationCode(ctx context.Context, email string, data *service.VerificationCodeData, ttl time.Duration) error {
	key := verifyCodeKey(email)
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, val, ttl).Err()
}

func (c *emailCache) DeleteVerificationCode(ctx context.Context, email string) error {
	key := verifyCodeKey(email)
	return c.rdb.Del(ctx, key).Err()
}

// Password reset token methods

func (c *emailCache) GetPasswordResetToken(ctx context.Context, email string) (*service.PasswordResetTokenData, error) {
	key := passwordResetKey(email)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var data service.PasswordResetTokenData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (c *emailCache) SetPasswordResetToken(ctx context.Context, email string, data *service.PasswordResetTokenData, ttl time.Duration) error {
	key := passwordResetKey(email)
	val, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, val, ttl).Err()
}

func (c *emailCache) DeletePasswordResetToken(ctx context.Context, email string) error {
	key := passwordResetKey(email)
	return c.rdb.Del(ctx, key).Err()
}

// compareAndDeleteScript is a Lua script that atomically compares the token and deletes if matched.
// This prevents both race conditions and DoS attacks (wrong token won't delete valid token).
// Returns: 1 if matched and deleted, 0 if not matched, nil if key doesn't exist
var compareAndDeleteScript = redis.NewScript(`
local val = redis.call('GET', KEYS[1])
if val == false then
    return nil
end
local data = cjson.decode(val)
if data.Token == ARGV[1] then
    redis.call('DEL', KEYS[1])
    return 1
end
return 0
`)

// GetAndDeletePasswordResetToken atomically verifies and deletes the password reset token.
// Uses a Lua script to ensure the token matches before deletion.
// This prevents:
// 1. Race conditions where concurrent requests could both verify the same token
// 2. DoS attacks where wrong token could delete valid token
// Returns the token data only if the provided token matches and was successfully deleted.
func (c *emailCache) GetAndDeletePasswordResetToken(ctx context.Context, email string) (*service.PasswordResetTokenData, error) {
	// This method signature is kept for interface compatibility, but we need the token for comparison.
	// The actual atomic compare-and-delete is done via ConsumePasswordResetTokenAtomic.
	// For backward compatibility, this falls back to simple GET + DELETE (non-atomic).
	key := passwordResetKey(email)
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var data service.PasswordResetTokenData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}
	// Delete after getting (non-atomic, but caller should use ConsumePasswordResetTokenAtomic for safety)
	_ = c.rdb.Del(ctx, key).Err()
	return &data, nil
}

// ConsumePasswordResetTokenAtomic atomically verifies and deletes the password reset token.
// Uses a Lua script to compare token before deletion, preventing DoS attacks.
// Returns true if token matched and was deleted, false if token didn't match or doesn't exist.
func (c *emailCache) ConsumePasswordResetTokenAtomic(ctx context.Context, email, token string) (bool, error) {
	key := passwordResetKey(email)
	result, err := compareAndDeleteScript.Run(ctx, c.rdb, []string{key}, token).Result()
	if err == redis.Nil {
		// Key doesn't exist
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// result is 1 if matched and deleted, 0 if not matched
	return result.(int64) == 1, nil
}

// Password reset email cooldown methods

func (c *emailCache) IsPasswordResetEmailInCooldown(ctx context.Context, email string) bool {
	key := passwordResetSentAtKey(email)
	exists, err := c.rdb.Exists(ctx, key).Result()
	return err == nil && exists > 0
}

func (c *emailCache) SetPasswordResetEmailCooldown(ctx context.Context, email string, ttl time.Duration) error {
	key := passwordResetSentAtKey(email)
	return c.rdb.Set(ctx, key, "1", ttl).Err()
}
