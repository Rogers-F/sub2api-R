package service

import (
	"context"
	"sync/atomic"
	"time"
)

type channelCache struct {
	channelByGroupID map[int64]*Channel
	byID             map[int64]*Channel
	groupPlatform    map[int64]string
	loadedAt         time.Time
}

type ChannelService struct {
	cache atomic.Value // *channelCache
}

func (s *ChannelService) loadCache(_ context.Context) (*channelCache, error) {
	if cached, ok := s.cache.Load().(*channelCache); ok && cached != nil {
		return cached, nil
	}
	return &channelCache{
		channelByGroupID: make(map[int64]*Channel),
		byID:             make(map[int64]*Channel),
		groupPlatform:    make(map[int64]string),
		loadedAt:         time.Now(),
	}, nil
}

func (s *ChannelService) GetChannelForGroup(ctx context.Context, groupID int64) (*Channel, error) {
	cache, err := s.loadCache(ctx)
	if err != nil {
		return nil, err
	}
	ch, ok := cache.channelByGroupID[groupID]
	if !ok || !ch.IsActive() {
		return nil, nil
	}
	return ch.Clone(), nil
}

func (s *ChannelService) GetGroupPlatform(ctx context.Context, groupID int64) string {
	cache, err := s.loadCache(ctx)
	if err != nil {
		return ""
	}
	return cache.groupPlatform[groupID]
}
