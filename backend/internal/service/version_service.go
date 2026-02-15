package service

import (
	"context"
	"math/rand"
)

// CLI version pool (aligned with 88code)
var cliVersionPool = []string{
	"2.0.65",
	"2.0.66",
	"2.0.67",
}

// Node.js LTS version pool (real versions from 88code)
var nodeVersionPool = []string{
	"v18.12.0", "v18.12.1", "v18.13.0", "v18.14.0", "v18.14.1", "v18.14.2",
	"v18.15.0", "v18.16.0", "v18.16.1", "v18.17.0", "v18.17.1", "v18.18.0",
	"v18.18.1", "v18.18.2", "v18.19.0", "v18.19.1", "v18.20.0",
	"v20.9.0", "v20.10.0", "v20.11.0", "v20.11.1", "v20.12.0", "v20.12.1",
	"v20.12.2", "v20.13.0", "v20.13.1", "v20.14.0",
}

// VersionCache defines cache operations for version persistence
type VersionCache interface {
	GetCLIVersion(ctx context.Context, key string) (string, error)
	SetCLIVersion(ctx context.Context, key string, version string) error
	GetNodeVersion(ctx context.Context, key string) (string, error)
	SetNodeVersion(ctx context.Context, key string, version string) error
}

// VersionService manages per-account CLI and Node.js version assignments
type VersionService struct {
	cache VersionCache
}

// NewVersionService creates a new VersionService
func NewVersionService(cache VersionCache) *VersionService {
	return &VersionService{cache: cache}
}

// GetOrCreateCLIVersion returns the cached CLI version for the given key,
// or picks a random one from the pool and caches it.
func (s *VersionService) GetOrCreateCLIVersion(ctx context.Context, accountKey string) string {
	cached, err := s.cache.GetCLIVersion(ctx, accountKey)
	if err == nil && cached != "" {
		return cached
	}

	version := cliVersionPool[rand.Intn(len(cliVersionPool))]
	_ = s.cache.SetCLIVersion(ctx, accountKey, version)
	return version
}

// GetOrCreateNodeVersion returns the cached Node version for the given key,
// or picks a random one from the pool and caches it.
func (s *VersionService) GetOrCreateNodeVersion(ctx context.Context, accountKey string) string {
	cached, err := s.cache.GetNodeVersion(ctx, accountKey)
	if err == nil && cached != "" {
		return cached
	}

	version := nodeVersionPool[rand.Intn(len(nodeVersionPool))]
	_ = s.cache.SetNodeVersion(ctx, accountKey, version)
	return version
}
