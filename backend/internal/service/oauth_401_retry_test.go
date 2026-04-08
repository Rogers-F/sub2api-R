//go:build unit

package service

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	openaipkg "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type openAIForwardRepoStub struct {
	mockAccountRepoForGemini
	account              *Account
	updateCalls          int
	setErrorCalls        int
	tempUnschedCalls     int
	lastErrorMsg         string
	lastTempUnschedUntil time.Time
	lastTempUnschedMsg   string
}

func (r *openAIForwardRepoStub) GetByID(ctx context.Context, id int64) (*Account, error) {
	if r.account != nil && r.account.ID == id {
		return r.account, nil
	}
	return nil, errors.New("account not found")
}

func (r *openAIForwardRepoStub) Update(ctx context.Context, account *Account) error {
	r.updateCalls++
	r.account = account
	return nil
}

func (r *openAIForwardRepoStub) SetError(ctx context.Context, id int64, errorMsg string) error {
	r.setErrorCalls++
	r.lastErrorMsg = errorMsg
	return nil
}

func (r *openAIForwardRepoStub) SetTempUnschedulable(ctx context.Context, id int64, until time.Time, reason string) error {
	r.tempUnschedCalls++
	r.lastTempUnschedUntil = until
	r.lastTempUnschedMsg = reason
	return nil
}

type openAIOAuthClientStub struct {
	refreshResponse *openaipkg.TokenResponse
	refreshErr      error
	refreshCalls    int
}

func (s *openAIOAuthClientStub) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL, clientID string) (*openaipkg.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *openAIOAuthClientStub) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*openaipkg.TokenResponse, error) {
	s.refreshCalls++
	if s.refreshErr != nil {
		return nil, s.refreshErr
	}
	return s.refreshResponse, nil
}

func (s *openAIOAuthClientStub) RefreshTokenWithClientID(ctx context.Context, refreshToken, proxyURL string, clientID string) (*openaipkg.TokenResponse, error) {
	return s.RefreshToken(ctx, refreshToken, proxyURL)
}

type openAIForwardUpstreamStub struct {
	calls     int
	auths     []string
	responder func(call int, req *http.Request) (*http.Response, error)
}

func (s *openAIForwardUpstreamStub) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	s.calls++
	s.auths = append(s.auths, req.Header.Get("authorization"))
	return s.responder(s.calls, req)
}

func (s *openAIForwardUpstreamStub) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, enableTLSFingerprint bool) (*http.Response, error) {
	return s.Do(req, proxyURL, accountID, accountConcurrency)
}

func newOpenAITestResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"x-request-id": []string{"req-test"},
		},
	}
}

func newOpenAITestContext(body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", bytes.NewReader(body))
	c.Request.Header.Set("content-type", "application/json")
	return c, rec
}

func TestOpenAIGatewayService_Forward_ExpiredOAuth401MarksAccountForRefresh(t *testing.T) {
	requestBody := []byte(`{"model":"gpt-4.1","stream":false}`)
	account := &Account{
		ID:       201,
		Name:     "openai-oauth",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":  "stale-token",
			"refresh_token": "refresh-token",
			"expires_at":    time.Now().Add(10 * time.Minute).Format(time.RFC3339),
		},
	}
	repo := &openAIForwardRepoStub{account: account}
	tokenCache := newOpenAITokenCacheStub()
	tokenCache.tokens[OpenAITokenCacheKey(account)] = "stale-token"
	oauthClient := &openAIOAuthClientStub{
		refreshResponse: &openaipkg.TokenResponse{
			AccessToken:  "fresh-token",
			RefreshToken: "fresh-refresh-token",
			ExpiresIn:    3600,
		},
	}
	oauthService := NewOpenAIOAuthService(nil, oauthClient)
	provider := NewOpenAITokenProvider(repo, tokenCache, oauthService)
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitService.SetTokenCacheInvalidator(NewCompositeTokenCacheInvalidator(tokenCache))
	upstream := &openAIForwardUpstreamStub{
		responder: func(call int, req *http.Request) (*http.Response, error) {
			require.Equal(t, 1, call)
			require.Equal(t, "Bearer stale-token", req.Header.Get("authorization"))
			return newOpenAITestResponse(http.StatusUnauthorized, `{"type":"error","error":{"type":"authentication_error","message":"OAuth token has expired. Please obtain a new token or refresh your existing token."}}`), nil
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:         repo,
		cfg:                 &config.Config{},
		httpUpstream:        upstream,
		openAITokenProvider: provider,
		rateLimitService:    rateLimitService,
	}
	c, _ := newOpenAITestContext(requestBody)

	result, err := svc.Forward(context.Background(), c, account, requestBody)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusUnauthorized, failoverErr.StatusCode)
	require.Equal(t, 1, upstream.calls)
	require.Equal(t, []string{"Bearer stale-token"}, upstream.auths)
	require.Equal(t, 0, oauthClient.refreshCalls)
	require.Equal(t, "stale-token", repo.account.GetOpenAIAccessToken())
	require.GreaterOrEqual(t, repo.updateCalls, 1)
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, 1, repo.tempUnschedCalls)
	require.Contains(t, strings.ToLower(repo.lastTempUnschedMsg), "oauth token has expired")
	expiresAt := repo.account.GetCredentialAsTime("expires_at")
	require.NotNil(t, expiresAt)
	require.WithinDuration(t, time.Now(), *expiresAt, 5*time.Second)
	_, cacheStillPresent := tokenCache.tokens[OpenAITokenCacheKey(account)]
	require.False(t, cacheStillPresent)
}

func TestOpenAIGatewayService_Forward_ExpiredOAuth401DoesNotSetError(t *testing.T) {
	requestBody := []byte(`{"model":"gpt-4.1","stream":false}`)
	account := &Account{
		ID:       202,
		Name:     "openai-oauth",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":  "stale-token",
			"refresh_token": "refresh-token",
			"expires_at":    time.Now().Add(10 * time.Minute).Format(time.RFC3339),
		},
	}
	repo := &openAIForwardRepoStub{account: account}
	tokenCache := newOpenAITokenCacheStub()
	oauthClient := &openAIOAuthClientStub{
		refreshResponse: &openaipkg.TokenResponse{
			AccessToken:  "fresh-token",
			RefreshToken: "fresh-refresh-token",
			ExpiresIn:    3600,
		},
	}
	oauthService := NewOpenAIOAuthService(nil, oauthClient)
	provider := NewOpenAITokenProvider(repo, tokenCache, oauthService)
	invalidator := &tokenCacheInvalidatorRecorder{}
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitService.SetTokenCacheInvalidator(invalidator)
	upstream := &openAIForwardUpstreamStub{
		responder: func(call int, req *http.Request) (*http.Response, error) {
			require.Equal(t, 1, call)
			require.Equal(t, "Bearer stale-token", req.Header.Get("authorization"))
			return newOpenAITestResponse(http.StatusUnauthorized, `{"type":"error","error":{"type":"authentication_error","message":"OAuth token has expired. Please obtain a new token or refresh your existing token."}}`), nil
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:         repo,
		cfg:                 &config.Config{},
		httpUpstream:        upstream,
		openAITokenProvider: provider,
		rateLimitService:    rateLimitService,
	}
	c, _ := newOpenAITestContext(requestBody)

	result, err := svc.Forward(context.Background(), c, account, requestBody)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, http.StatusUnauthorized, failoverErr.StatusCode)
	require.Equal(t, 1, upstream.calls)
	require.Equal(t, 0, oauthClient.refreshCalls)
	require.Equal(t, 0, repo.setErrorCalls)
	require.GreaterOrEqual(t, repo.updateCalls, 1)
	require.Equal(t, 1, repo.tempUnschedCalls)
	require.Len(t, invalidator.accounts, 1)
	require.Equal(t, account.ID, invalidator.accounts[0].ID)
}

func TestOpenAIGatewayService_Forward_PermanentOAuth401DoesNotRefresh(t *testing.T) {
	requestBody := []byte(`{"model":"gpt-4.1","stream":false}`)
	account := &Account{
		ID:       203,
		Name:     "openai-oauth",
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":  "stale-token",
			"refresh_token": "refresh-token",
			"expires_at":    time.Now().Add(10 * time.Minute).Format(time.RFC3339),
		},
	}
	repo := &openAIForwardRepoStub{account: account}
	tokenCache := newOpenAITokenCacheStub()
	oauthClient := &openAIOAuthClientStub{
		refreshResponse: &openaipkg.TokenResponse{
			AccessToken:  "fresh-token",
			RefreshToken: "fresh-refresh-token",
			ExpiresIn:    3600,
		},
	}
	oauthService := NewOpenAIOAuthService(nil, oauthClient)
	provider := NewOpenAITokenProvider(repo, tokenCache, oauthService)
	invalidator := &tokenCacheInvalidatorRecorder{}
	rateLimitService := NewRateLimitService(repo, nil, &config.Config{}, nil, nil)
	rateLimitService.SetTokenCacheInvalidator(invalidator)
	upstream := &openAIForwardUpstreamStub{
		responder: func(call int, req *http.Request) (*http.Response, error) {
			require.Equal(t, "Bearer stale-token", req.Header.Get("authorization"))
			return newOpenAITestResponse(http.StatusUnauthorized, `{"error":{"message":"Token revoked"}}`), nil
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:         repo,
		cfg:                 &config.Config{},
		httpUpstream:        upstream,
		openAITokenProvider: provider,
		rateLimitService:    rateLimitService,
	}
	c, _ := newOpenAITestContext(requestBody)

	result, err := svc.Forward(context.Background(), c, account, requestBody)

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr)
	require.Equal(t, 1, upstream.calls)
	require.Equal(t, 0, oauthClient.refreshCalls)
	// OAuth 账号的 401 不再调用 SetError（对齐 88code：从不因 401 禁用 OAuth 账号）
	require.Equal(t, 0, repo.setErrorCalls)
	require.Equal(t, 1, repo.updateCalls)
	require.Len(t, invalidator.accounts, 1)
}
