package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"sync/atomic"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/websearch"
	"golang.org/x/sync/singleflight"
)

const (
	WebSearchModeDefault  = "default"
	WebSearchModeEnabled  = "enabled"
	WebSearchModeDisabled = "disabled"
)

type WebSearchEmulationConfig struct {
	Enabled   bool                      `json:"enabled"`
	Providers []WebSearchProviderConfig `json:"providers"`
}

type WebSearchProviderConfig struct {
	Type             string `json:"type"`
	APIKey           string `json:"api_key,omitempty"`
	APIKeyConfigured bool   `json:"api_key_configured"`
	QuotaLimit       *int64 `json:"quota_limit"`
	SubscribedAt     *int64 `json:"subscribed_at,omitempty"`
	QuotaUsed        int64  `json:"quota_used,omitempty"`
	ProxyID          *int64 `json:"proxy_id"`
	ExpiresAt        *int64 `json:"expires_at,omitempty"`
}

const (
	maxWebSearchProviders       = 10
	sfKeyWebSearchConfig        = "web_search_emulation_config"
	webSearchEmulationCacheTTL  = 60 * time.Second
	webSearchEmulationErrorTTL  = 5 * time.Second
	webSearchEmulationDBTimeout = 5 * time.Second
)

var (
	validWebSearchProviderTypes = []string{
		websearch.ProviderTypeBrave,
		websearch.ProviderTypeTavily,
	}

	webSearchEmulationCache atomic.Value // *cachedWebSearchEmulationConfig
	webSearchEmulationSF    singleflight.Group
)

type cachedWebSearchEmulationConfig struct {
	config    *WebSearchEmulationConfig
	expiresAt int64
}

func validateWebSearchConfig(cfg *WebSearchEmulationConfig) error {
	if cfg == nil {
		return nil
	}
	if len(cfg.Providers) > maxWebSearchProviders {
		return fmt.Errorf("too many providers (max %d)", maxWebSearchProviders)
	}

	seen := make(map[string]struct{}, len(cfg.Providers))
	for i, provider := range cfg.Providers {
		if !slices.Contains(validWebSearchProviderTypes, provider.Type) {
			return fmt.Errorf("provider[%d]: invalid type %q", i, provider.Type)
		}
		if provider.QuotaLimit != nil && *provider.QuotaLimit < 0 {
			return fmt.Errorf("provider[%d]: quota_limit must be > 0 or null", i)
		}
		if _, exists := seen[provider.Type]; exists {
			return fmt.Errorf("provider[%d]: duplicate type %q", i, provider.Type)
		}
		seen[provider.Type] = struct{}{}
	}
	return nil
}

func parseWebSearchConfigJSON(raw string) *WebSearchEmulationConfig {
	cfg := &WebSearchEmulationConfig{}
	if raw == "" {
		return cfg
	}
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		slog.Warn("websearch: failed to parse config JSON", "error", err)
		return &WebSearchEmulationConfig{}
	}
	return cfg
}

func (s *SettingService) GetWebSearchEmulationConfig(ctx context.Context) (*WebSearchEmulationConfig, error) {
	if cached := webSearchEmulationCache.Load(); cached != nil {
		if cfg, ok := cached.(*cachedWebSearchEmulationConfig); ok && cfg != nil && time.Now().UnixNano() < cfg.expiresAt {
			return cfg.config, nil
		}
	}

	result, err, _ := webSearchEmulationSF.Do(sfKeyWebSearchConfig, func() (any, error) {
		return s.loadWebSearchConfigFromDB()
	})
	if err != nil {
		return &WebSearchEmulationConfig{}, err
	}
	cfg, _ := result.(*WebSearchEmulationConfig)
	if cfg == nil {
		return &WebSearchEmulationConfig{}, nil
	}
	return cfg, nil
}

func (s *SettingService) loadWebSearchConfigFromDB() (*WebSearchEmulationConfig, error) {
	dbCtx, cancel := context.WithTimeout(context.Background(), webSearchEmulationDBTimeout)
	defer cancel()

	raw, err := s.settingRepo.GetValue(dbCtx, SettingKeyWebSearchEmulationConfig)
	if err != nil {
		webSearchEmulationCache.Store(&cachedWebSearchEmulationConfig{
			config:    &WebSearchEmulationConfig{},
			expiresAt: time.Now().Add(webSearchEmulationErrorTTL).UnixNano(),
		})
		return &WebSearchEmulationConfig{}, err
	}

	cfg := parseWebSearchConfigJSON(raw)
	webSearchEmulationCache.Store(&cachedWebSearchEmulationConfig{
		config:    cfg,
		expiresAt: time.Now().Add(webSearchEmulationCacheTTL).UnixNano(),
	})
	return cfg, nil
}

func (s *SettingService) SaveWebSearchEmulationConfig(ctx context.Context, cfg *WebSearchEmulationConfig) error {
	if err := validateWebSearchConfig(cfg); err != nil {
		return infraerrors.BadRequest("INVALID_WEB_SEARCH_CONFIG", err.Error())
	}
	s.mergeExistingWebSearchAPIKeys(ctx, cfg)

	if cfg != nil && cfg.Enabled {
		for _, provider := range cfg.Providers {
			if provider.APIKey == "" {
				return infraerrors.BadRequest(
					"MISSING_API_KEY",
					fmt.Sprintf("provider %s has no API key configured", provider.Type),
				)
			}
		}
	}

	payload, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("websearch: marshal config: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyWebSearchEmulationConfig, string(payload)); err != nil {
		return fmt.Errorf("websearch: save config: %w", err)
	}

	webSearchEmulationSF.Forget(sfKeyWebSearchConfig)
	webSearchEmulationCache.Store(&cachedWebSearchEmulationConfig{
		config:    cfg,
		expiresAt: time.Now().Add(webSearchEmulationCacheTTL).UnixNano(),
	})
	s.rebuildWebSearchManager(ctx)
	return nil
}

func (s *SettingService) mergeExistingWebSearchAPIKeys(ctx context.Context, cfg *WebSearchEmulationConfig) {
	if cfg == nil {
		return
	}
	existing, err := s.getWebSearchEmulationConfigRaw(ctx)
	if err != nil || existing == nil {
		return
	}
	existingKeys := make(map[string]string, len(existing.Providers))
	for _, provider := range existing.Providers {
		if provider.APIKey != "" {
			existingKeys[provider.Type] = provider.APIKey
		}
	}
	for i := range cfg.Providers {
		if cfg.Providers[i].APIKey == "" {
			if apiKey, ok := existingKeys[cfg.Providers[i].Type]; ok {
				cfg.Providers[i].APIKey = apiKey
			}
		}
	}
}

func (s *SettingService) getWebSearchEmulationConfigRaw(ctx context.Context) (*WebSearchEmulationConfig, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyWebSearchEmulationConfig)
	if err != nil {
		return nil, err
	}
	return parseWebSearchConfigJSON(raw), nil
}

func (s *SettingService) IsWebSearchEmulationEnabled(ctx context.Context) bool {
	cfg, err := s.GetWebSearchEmulationConfig(ctx)
	if err != nil || cfg == nil {
		return false
	}
	return cfg.Enabled && len(cfg.Providers) > 0
}

func (s *SettingService) SetWebSearchManagerBuilder(ctx context.Context, builder WebSearchManagerBuilder) {
	s.webSearchManagerBuilder = builder
	s.rebuildWebSearchManager(ctx)
}

func (s *SettingService) rebuildWebSearchManager(ctx context.Context) {
	if s == nil || s.webSearchManagerBuilder == nil {
		return
	}
	cfg, err := s.GetWebSearchEmulationConfig(ctx)
	if err != nil {
		SetWebSearchManager(nil)
		return
	}
	s.webSearchManagerBuilder(cfg, s.resolveProviderProxyURLs(ctx, cfg))
}

func (s *SettingService) resolveProviderProxyURLs(ctx context.Context, cfg *WebSearchEmulationConfig) map[int64]string {
	if s == nil || s.proxyRepo == nil || cfg == nil {
		return nil
	}

	ids := make([]int64, 0, len(cfg.Providers))
	seen := make(map[int64]struct{}, len(cfg.Providers))
	for _, provider := range cfg.Providers {
		if provider.ProxyID == nil || *provider.ProxyID <= 0 {
			continue
		}
		if _, ok := seen[*provider.ProxyID]; ok {
			continue
		}
		seen[*provider.ProxyID] = struct{}{}
		ids = append(ids, *provider.ProxyID)
	}
	if len(ids) == 0 {
		return nil
	}

	proxies, err := s.proxyRepo.ListByIDs(ctx, ids)
	if err != nil {
		slog.Warn("websearch: failed to resolve proxy URLs", "error", err)
		return nil
	}

	result := make(map[int64]string, len(proxies))
	for _, proxy := range proxies {
		result[proxy.ID] = proxy.URL()
	}
	return result
}

type WebSearchTestResult struct {
	Provider string                   `json:"provider"`
	Results  []websearch.SearchResult `json:"results"`
	Query    string                   `json:"query"`
}

const testSearchTimeout = 15 * time.Second

func TestWebSearch(ctx context.Context, query string) (*WebSearchTestResult, error) {
	manager := getWebSearchManager()
	if manager == nil {
		return nil, fmt.Errorf("web search: manager not initialized, save config first")
	}

	testCtx, cancel := context.WithTimeout(ctx, testSearchTimeout)
	defer cancel()

	resp, providerName, err := manager.TestSearch(testCtx, websearch.SearchRequest{
		Query:      query,
		MaxResults: webSearchDefaultMaxResults,
	})
	if err != nil {
		return nil, err
	}
	return &WebSearchTestResult{
		Provider: providerName,
		Results:  resp.Results,
		Query:    resp.Query,
	}, nil
}

func SanitizeWebSearchConfig(_ context.Context, cfg *WebSearchEmulationConfig) *WebSearchEmulationConfig {
	if cfg == nil {
		return nil
	}
	out := *cfg
	out.Providers = make([]WebSearchProviderConfig, len(cfg.Providers))
	for i, provider := range cfg.Providers {
		out.Providers[i] = provider
		out.Providers[i].APIKeyConfigured = provider.APIKey != ""
		out.Providers[i].APIKey = ""
	}
	return &out
}

func PopulateWebSearchUsage(ctx context.Context, cfg *WebSearchEmulationConfig) *WebSearchEmulationConfig {
	if cfg == nil {
		return nil
	}
	out := *cfg
	out.Providers = make([]WebSearchProviderConfig, len(cfg.Providers))
	manager := getWebSearchManager()
	for i, provider := range cfg.Providers {
		out.Providers[i] = provider
		out.Providers[i].APIKeyConfigured = provider.APIKey != ""
		if manager != nil {
			used, _ := manager.GetUsage(ctx, provider.Type)
			out.Providers[i].QuotaUsed = used
		}
	}
	return &out
}

func ResetWebSearchUsage(ctx context.Context, providerType string) error {
	manager := getWebSearchManager()
	if manager == nil {
		return fmt.Errorf("web search manager not initialized")
	}
	return manager.ResetUsage(ctx, providerType)
}
