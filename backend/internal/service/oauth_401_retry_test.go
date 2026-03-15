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
	account       *Account
	updateCalls   int
	setErrorCalls int
	lastErrorMsg  string
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

type openAIOAuthClientStub struct {
	refreshResponse *openaipkg.TokenResponse
	refreshErr      error
	refreshCalls    int
}

func (s *openAIOAuthClientStub) ExchangeCode(ctx context.Context, code, codeVerifier, redirectURI, proxyURL string) (*openaipkg.TokenResponse, error) {
	return nil, errors.New("not implemented")
}

func (s *openAIOAuthClientStub) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*openaipkg.TokenResponse, error) {
	s.refreshCalls++
	if s.refreshErr != nil {
		return nil, s.refreshErr
	}
	return s.refreshResponse, nil
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

func TestOpenAIGatewayService_Forward_ExpiredOAuth401RefreshesAndRetries(t *testing.T) {
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
	oauthClient := &openAIOAuthClientStub{
		refreshResponse: &openaipkg.TokenResponse{
			AccessToken:  "fresh-token",
			RefreshToken: "fresh-refresh-token",
			ExpiresIn:    3600,
		},
	}
	oauthService := NewOpenAIOAuthService(nil, oauthClient)
	provider := NewOpenAITokenProvider(repo, tokenCache, oauthService)
	upstream := &openAIForwardUpstreamStub{
		responder: func(call int, req *http.Request) (*http.Response, error) {
			switch call {
			case 1:
				require.Equal(t, "Bearer stale-token", req.Header.Get("authorization"))
				return newOpenAITestResponse(http.StatusUnauthorized, `{"type":"error","error":{"type":"authentication_error","message":"OAuth token has expired. Please obtain a new token or refresh your existing token."}}`), nil
			case 2:
				require.Equal(t, "Bearer fresh-token", req.Header.Get("authorization"))
				return newOpenAITestResponse(http.StatusOK, `{"usage":{"input_tokens":3,"output_tokens":5,"input_tokens_details":{"cached_tokens":1}}}`), nil
			default:
				return nil, errors.New("unexpected call")
			}
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:         repo,
		cfg:                 &config.Config{},
		httpUpstream:        upstream,
		openAITokenProvider: provider,
		rateLimitService:    NewRateLimitService(repo, nil, &config.Config{}, nil, nil),
	}
	c, rec := newOpenAITestContext(requestBody)

	result, err := svc.Forward(context.Background(), c, account, requestBody)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 2, upstream.calls)
	require.Equal(t, []string{"Bearer stale-token", "Bearer fresh-token"}, upstream.auths)
	require.Equal(t, 1, oauthClient.refreshCalls)
	require.Equal(t, "fresh-token", repo.account.GetOpenAIAccessToken())
	// 委托给 TokenProvider 后，updateCalls 可能为 2（provider 内部 + Forward 流程各一次）
	require.GreaterOrEqual(t, repo.updateCalls, 1)
	require.Equal(t, 200, rec.Code)
	require.Equal(t, 3, result.Usage.InputTokens)
	require.Equal(t, 5, result.Usage.OutputTokens)
	// 通过 TokenProvider 刷新后 token 会被缓存（预期行为变化）
	require.Equal(t, "fresh-token", tokenCache.tokens[OpenAITokenCacheKey(account)])
}

func TestOpenAIGatewayService_Forward_ExpiredOAuth401FinalFailoverDoesNotSetError(t *testing.T) {
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
			switch call {
			case 1:
				return newOpenAITestResponse(http.StatusUnauthorized, `{"type":"error","error":{"type":"authentication_error","message":"OAuth token has expired. Please obtain a new token or refresh your existing token."}}`), nil
			case 2:
				require.Equal(t, "Bearer fresh-token", req.Header.Get("authorization"))
				return newOpenAITestResponse(http.StatusUnauthorized, `{"type":"error","error":{"type":"authentication_error","message":"Access token expired"}}`), nil
			default:
				return nil, errors.New("unexpected call")
			}
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
	require.Equal(t, 2, upstream.calls)
	require.Equal(t, 1, oauthClient.refreshCalls)
	require.Equal(t, 0, repo.setErrorCalls)
	require.GreaterOrEqual(t, repo.updateCalls, 2)
	require.Len(t, invalidator.accounts, 1)
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
