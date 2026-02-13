package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/redis/go-redis/v9"
)

const (
	oauthSessionKeyPrefix = "oauth:session:"
)

// redisSessionStore implements oauth.SessionStore backed by Redis
type redisSessionStore struct {
	rdb *redis.Client
}

// NewRedisSessionStore creates a Redis-backed OAuth session store
func NewRedisSessionStore(rdb *redis.Client) oauth.SessionStore {
	return &redisSessionStore{rdb: rdb}
}

func (s *redisSessionStore) Set(ctx context.Context, sessionID string, session *oauth.OAuthSession) error {
	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	key := oauthSessionKeyPrefix + sessionID
	return s.rdb.Set(ctx, key, data, oauth.SessionTTL).Err()
}

func (s *redisSessionStore) Get(ctx context.Context, sessionID string) (*oauth.OAuthSession, error) {
	key := oauthSessionKeyPrefix + sessionID
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, oauth.ErrSessionNotFound
		}
		return nil, fmt.Errorf("redis get session: %w", err)
	}
	var session oauth.OAuthSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	return &session, nil
}

func (s *redisSessionStore) Delete(ctx context.Context, sessionID string) error {
	key := oauthSessionKeyPrefix + sessionID
	return s.rdb.Del(ctx, key).Err()
}

func (s *redisSessionStore) Stop() {
	// no-op: Redis handles TTL expiration automatically
}
