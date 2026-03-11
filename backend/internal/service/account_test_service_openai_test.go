package service

import (
	"context"
	"net/http"
	"testing"
	"time"
)

type accountTestTokenCacheStub struct {
	token string
}

func (s *accountTestTokenCacheStub) GetAccessToken(ctx context.Context, cacheKey string) (string, error) {
	return s.token, nil
}

func (s *accountTestTokenCacheStub) SetAccessToken(ctx context.Context, cacheKey string, token string, ttl time.Duration) error {
	return nil
}

func (s *accountTestTokenCacheStub) DeleteAccessToken(ctx context.Context, cacheKey string) error {
	return nil
}

func (s *accountTestTokenCacheStub) AcquireRefreshLock(ctx context.Context, cacheKey string, ttl time.Duration) (bool, error) {
	return false, nil
}

func (s *accountTestTokenCacheStub) ReleaseRefreshLock(ctx context.Context, cacheKey string) error {
	return nil
}

func TestBuildOpenAITestRequest_UsesTokenProviderForOAuth(t *testing.T) {
	provider := NewOpenAITokenProvider(nil, &accountTestTokenCacheStub{token: "provider-token"}, nil)
	service := NewAccountTestService(nil, nil, nil, provider, nil, nil, nil)
	account := &Account{
		ID:       42,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":       "stale-db-token",
			"chatgpt_account_id": "acct-123",
		},
	}

	req, _, _, err := service.buildOpenAITestRequest(context.Background(), account, "", false)
	if err != nil {
		t.Fatalf("buildOpenAITestRequest returned error: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer provider-token" {
		t.Fatalf("Authorization = %q, want %q", got, "Bearer provider-token")
	}
	if got := req.Header.Get("chatgpt-account-id"); got != "acct-123" {
		t.Fatalf("chatgpt-account-id = %q, want %q", got, "acct-123")
	}
	if req.Method != http.MethodPost {
		t.Fatalf("Method = %q, want %q", req.Method, http.MethodPost)
	}
}
