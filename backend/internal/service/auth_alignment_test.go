//go:build unit

package service

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// oauthExpired401Markers lists known OAuth token expiry message fragments.
var oauthExpired401Markers = []string{
	"oauth token has expired",
	"token has expired",
	"access token expired",
	"expired token",
	"token expired",
	"session expired",
}

func isOAuthTokenExpired401(account *Account, statusCode int, responseBody []byte) bool {
	if account == nil || account.Type != AccountTypeOAuth || statusCode != http.StatusUnauthorized {
		return false
	}

	msg := strings.TrimSpace(extractUpstreamErrorMessage(responseBody))
	if msg == "" {
		msg = strings.TrimSpace(string(responseBody))
	}
	return isOAuth401ExpiredMessage(msg)
}

func isOAuth401ExpiredMessage(msg string) bool {
	normalized := strings.ToLower(strings.TrimSpace(sanitizeUpstreamErrorMessage(msg)))
	if normalized == "" {
		return false
	}

	for _, marker := range oauthExpired401Markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}

	return strings.Contains(normalized, "expired") &&
		(strings.Contains(normalized, "token") || strings.Contains(normalized, "session"))
}

func TestShouldRetryUpstreamError_AnthropicOAuthOnlyRetries403(t *testing.T) {
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
			MaxRetries:          1,
			RetryBackoffSeconds: 0,
		},
	}
	service := NewTokenRefreshService(repo, nil, nil, nil, nil, nil, nil, cfg, nil)
	service.refreshers = []TokenRefresher{
		&platformTokenRefresherStub{
			tokenRefresherStub: tokenRefresherStub{
				credentials: map[string]any{
					"access_token": "new-token",
				},
			},
			platform: PlatformGemini,
		},
	}

	service.processRefresh()

	require.Equal(t, 1, repo.updateCalls)
	require.NotNil(t, repo.lastAccount)
	require.Equal(t, PlatformGemini, repo.lastAccount.Platform)
}

type platformTokenRefresherStub struct {
	tokenRefresherStub
	platform string
}

func (r *platformTokenRefresherStub) CanRefresh(account *Account) bool {
	return account != nil && account.Platform == r.platform
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
