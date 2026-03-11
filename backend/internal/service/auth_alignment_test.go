//go:build unit

package service

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGatewayService_ShouldRetryUpstreamError_AnthropicOAuthOnlyRetries403(t *testing.T) {
	svc := &GatewayService{}
	account := &Account{
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
	}

	require.False(t, svc.shouldRetryUpstreamError(account, 401))
	require.True(t, svc.shouldRetryUpstreamError(account, 403))
	require.False(t, svc.shouldRetryUpstreamError(account, 429))
}

func TestIsOAuthTokenExpired401(t *testing.T) {
	tests := []struct {
		name       string
		account    *Account
		statusCode int
		body       []byte
		want       bool
	}{
		{
			name: "openai oauth expired",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
			},
			statusCode: http.StatusUnauthorized,
			body:       []byte(`{"type":"error","error":{"type":"authentication_error","message":"OAuth token has expired. Please obtain a new token or refresh your existing token."}}`),
			want:       true,
		},
		{
			name: "anthropic oauth expired",
			account: &Account{
				Platform: PlatformAnthropic,
				Type:     AccountTypeOAuth,
			},
			statusCode: http.StatusUnauthorized,
			body:       []byte(`{"error":{"message":"Access token expired"}}`),
			want:       true,
		},
		{
			name: "oauth permanent 401",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
			},
			statusCode: http.StatusUnauthorized,
			body:       []byte(`{"error":{"message":"Token revoked"}}`),
			want:       false,
		},
		{
			name: "non oauth account",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
			},
			statusCode: http.StatusUnauthorized,
			body:       []byte(`{"error":{"message":"OAuth token has expired"}}`),
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, isOAuthTokenExpired401(tt.account, tt.statusCode, tt.body))
		})
	}
}

func TestShouldUseBackgroundTokenRefresh(t *testing.T) {
	tests := []struct {
		name    string
		account *Account
		want    bool
	}{
		{
			name: "anthropic oauth disabled",
			account: &Account{
				Platform: PlatformAnthropic,
				Type:     AccountTypeOAuth,
			},
			want: false,
		},
		{
			name: "openai oauth disabled",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
			},
			want: false,
		},
		{
			name: "gemini oauth enabled",
			account: &Account{
				Platform: PlatformGemini,
				Type:     AccountTypeOAuth,
			},
			want: true,
		},
		{
			name: "api key disabled",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeAPIKey,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, shouldUseBackgroundTokenRefresh(tt.account))
		})
	}
}

func TestTokenRefreshService_ProcessRefresh_SkipsAnthropicAndOpenAI(t *testing.T) {
	repo := &tokenRefreshProcessRepo{
		accounts: []Account{
			{ID: 1, Name: "claude", Platform: PlatformAnthropic, Type: AccountTypeOAuth, Credentials: map[string]any{"expires_at": time.Now().Add(-time.Minute).Format(time.RFC3339)}},
			{ID: 2, Name: "openai", Platform: PlatformOpenAI, Type: AccountTypeOAuth, Credentials: map[string]any{"expires_at": time.Now().Add(-time.Minute).Format(time.RFC3339)}},
			{ID: 3, Name: "gemini", Platform: PlatformGemini, Type: AccountTypeOAuth, Credentials: map[string]any{"expires_at": time.Now().Add(-time.Minute).Format(time.RFC3339)}},
		},
	}
	cfg := &config.Config{
		TokenRefresh: config.TokenRefreshConfig{
			MaxRetries:           1,
			RetryBackoffSeconds:  0,
			RefreshWindowMinutes: 2,
		},
	}
	service := NewTokenRefreshService(repo, nil, nil, nil, nil, nil, nil, nil, cfg)
	service.refreshers = []TokenRefresher{
		&tokenRefresherStub{
			credentials: map[string]any{
				"access_token": "new-token",
			},
		},
	}

	service.processRefresh()

	require.Equal(t, 1, repo.updateCalls)
	require.NotNil(t, repo.lastAccount)
	require.Equal(t, PlatformGemini, repo.lastAccount.Platform)
}

type tokenRefreshProcessRepo struct {
	mockAccountRepoForGemini
	accounts    []Account
	updateCalls int
	lastAccount *Account
}

func (r *tokenRefreshProcessRepo) ListActive(ctx context.Context) ([]Account, error) {
	return r.accounts, nil
}

func (r *tokenRefreshProcessRepo) Update(ctx context.Context, account *Account) error {
	r.updateCalls++
	r.lastAccount = account
	return nil
}
