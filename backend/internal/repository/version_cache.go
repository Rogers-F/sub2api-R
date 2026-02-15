package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	cliVersionKeyPrefix  = "claude:version:cli:"
	nodeVersionKeyPrefix = "claude:version:node:"
	cliVersionTTL        = 30 * 24 * time.Hour // 30 days
	// Node version: TTL 0 = permanent
)

type versionCache struct {
	rdb *redis.Client
}

// NewVersionCache creates a Redis-backed VersionCache
func NewVersionCache(rdb *redis.Client) service.VersionCache {
	return &versionCache{rdb: rdb}
}

func (c *versionCache) GetCLIVersion(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, fmt.Sprintf("%s%s", cliVersionKeyPrefix, key)).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func (c *versionCache) SetCLIVersion(ctx context.Context, key string, version string) error {
	return c.rdb.Set(ctx, fmt.Sprintf("%s%s", cliVersionKeyPrefix, key), version, cliVersionTTL).Err()
}

func (c *versionCache) GetNodeVersion(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, fmt.Sprintf("%s%s", nodeVersionKeyPrefix, key)).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

func (c *versionCache) SetNodeVersion(ctx context.Context, key string, version string) error {
	return c.rdb.Set(ctx, fmt.Sprintf("%s%s", nodeVersionKeyPrefix, key), version, 0).Err()
}
