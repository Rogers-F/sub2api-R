package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/oauth"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"

	"github.com/imroc/req/v3"
)

func NewClaudeOAuthClient(versionService *service.VersionService) service.ClaudeOAuthClient {
	return &claudeOAuthService{
		baseURL:        "https://claude.ai",
		tokenURL:       oauth.TokenURL,
		clientFactory:  createReqClient,
		versionService: versionService,
	}
}

type claudeOAuthService struct {
	baseURL        string
	tokenURL       string
	clientFactory  func(proxyURL string) *req.Client
	versionService *service.VersionService
}

func (s *claudeOAuthService) GetOrganizationUUID(ctx context.Context, sessionKey, proxyURL string) (string, error) {
	client := s.clientFactory(proxyURL)

	var orgs []struct {
		UUID      string  `json:"uuid"`
		Name      string  `json:"name"`
		RavenType *string `json:"raven_type"` // nil for personal, "team" for team organization
	}

	targetURL := s.baseURL + "/api/organizations"
	log.Printf("[OAuth] Step 1: Getting organization UUID from %s", targetURL)

	resp, err := client.R().
		SetContext(ctx).
		SetCookies(&http.Cookie{
			Name:  "sessionKey",
			Value: sessionKey,
		}).
		SetSuccessResult(&orgs).
		Get(targetURL)

	if err != nil {
		log.Printf("[OAuth] Step 1 FAILED - Request error: %v", err)
		return "", fmt.Errorf("request failed: %w", err)
	}

	log.Printf("[OAuth] Step 1 Response - Status: %d", resp.StatusCode)

	if !resp.IsSuccessState() {
		return "", fmt.Errorf("failed to get organizations: status %d, body: %s", resp.StatusCode, resp.String())
	}

	if len(orgs) == 0 {
		return "", fmt.Errorf("no organizations found")
	}

	// 如果只有一个组织，直接使用
	if len(orgs) == 1 {
		log.Printf("[OAuth] Step 1 SUCCESS - Single org found, UUID: %s, Name: %s", orgs[0].UUID, orgs[0].Name)
		return orgs[0].UUID, nil
	}

	// 如果有多个组织，优先选择 raven_type 为 "team" 的组织
	for _, org := range orgs {
		if org.RavenType != nil && *org.RavenType == "team" {
			log.Printf("[OAuth] Step 1 SUCCESS - Selected team org, UUID: %s, Name: %s, RavenType: %s",
				org.UUID, org.Name, *org.RavenType)
			return org.UUID, nil
		}
	}

	// 如果没有 team 类型的组织，使用第一个
	log.Printf("[OAuth] Step 1 SUCCESS - No team org found, using first org, UUID: %s, Name: %s", orgs[0].UUID, orgs[0].Name)
	return orgs[0].UUID, nil
}

func (s *claudeOAuthService) GetAuthorizationCode(ctx context.Context, sessionKey, orgUUID, scope, codeChallenge, state, proxyURL string) (string, error) {
	client := s.clientFactory(proxyURL)

	authURL := fmt.Sprintf("%s/v1/oauth/%s/authorize", s.baseURL, orgUUID)

	reqBody := map[string]any{
		"response_type":         "code",
		"client_id":             oauth.ClientID,
		"organization_uuid":     orgUUID,
		"redirect_uri":          oauth.RedirectURI,
		"scope":                 scope,
		"state":                 state,
		"code_challenge":        codeChallenge,
		"code_challenge_method": "S256",
	}

	log.Printf("[OAuth] Step 2: Getting authorization code from %s", authURL)
	reqBodyJSON, _ := json.Marshal(logredact.RedactMap(reqBody))
	log.Printf("[OAuth] Step 2 Request Body: %s", string(reqBodyJSON))

	var result struct {
		RedirectURI string `json:"redirect_uri"`
	}

	resp, err := client.R().
		SetContext(ctx).
		SetCookies(&http.Cookie{
			Name:  "sessionKey",
			Value: sessionKey,
		}).
		SetHeader("Accept", "application/json").
		SetHeader("Accept-Language", "en-US,en;q=0.9").
		SetHeader("Cache-Control", "no-cache").
		SetHeader("Origin", "https://claude.ai").
		SetHeader("Referer", "https://claude.ai/new").
		SetHeader("Content-Type", "application/json").
		SetBody(reqBody).
		SetSuccessResult(&result).
		Post(authURL)

	if err != nil {
		log.Printf("[OAuth] Step 2 FAILED - Request error: %v", err)
		return "", fmt.Errorf("request failed: %w", err)
	}

	log.Printf("[OAuth] Step 2 Response - Status: %d, Body: %s", resp.StatusCode, logredact.RedactJSON(resp.Bytes()))

	if !resp.IsSuccessState() {
		return "", fmt.Errorf("failed to get authorization code: status %d, body: %s", resp.StatusCode, resp.String())
	}

	if result.RedirectURI == "" {
		return "", fmt.Errorf("no redirect_uri in response")
	}

	parsedURL, err := url.Parse(result.RedirectURI)
	if err != nil {
		return "", fmt.Errorf("failed to parse redirect_uri: %w", err)
	}

	queryParams := parsedURL.Query()
	authCode := queryParams.Get("code")
	responseState := queryParams.Get("state")

	if authCode == "" {
		return "", fmt.Errorf("no authorization code in redirect_uri")
	}

	fullCode := authCode
	if responseState != "" {
		fullCode = authCode + "#" + responseState
	}

	log.Printf("[OAuth] Step 2 SUCCESS - Got authorization code")
	return fullCode, nil
}

func (s *claudeOAuthService) ExchangeCodeForToken(ctx context.Context, code, codeVerifier, state, proxyURL string, isSetupToken bool) (*oauth.TokenResponse, error) {
	s.sendPreOAuthHelloRequests(ctx, proxyURL)

	client := s.clientFactory(proxyURL)

	// Parse code which may contain state in format "authCode#state"
	authCode := code
	codeState := ""
	if idx := strings.Index(code, "#"); idx != -1 {
		authCode = code[:idx]
		codeState = code[idx+1:]
	}

	reqBody := map[string]any{
		"code":          authCode,
		"grant_type":    "authorization_code",
		"client_id":     oauth.ClientID,
		"redirect_uri":  oauth.RedirectURI,
		"code_verifier": codeVerifier,
	}

	if codeState != "" {
		reqBody["state"] = codeState
	}

	// Setup token requires longer expiration (1 year)
	if isSetupToken {
		reqBody["expires_in"] = 31536000 // 365 * 24 * 60 * 60 seconds
	}

	log.Printf("[OAuth] Step 3: Exchanging code for token at %s", s.tokenURL)
	reqBodyJSON, _ := json.Marshal(logredact.RedactMap(reqBody))
	log.Printf("[OAuth] Step 3 Request Body: %s", string(reqBodyJSON))

	var tokenResp oauth.TokenResponse

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json, text/plain, */*").
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", "axios/1.8.4").
		SetBody(reqBody).
		SetSuccessResult(&tokenResp).
		Post(s.tokenURL)

	if err != nil {
		log.Printf("[OAuth] Step 3 FAILED - Request error: %v", err)
		return nil, fmt.Errorf("request failed: %w", err)
	}

	log.Printf("[OAuth] Step 3 Response - Status: %d, Body: %s", resp.StatusCode, logredact.RedactJSON(resp.Bytes()))

	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("token exchange failed: status %d, body: %s", resp.StatusCode, resp.String())
	}

	log.Printf("[OAuth] Step 3 SUCCESS - Got access token")
	return &tokenResp, nil
}

func (s *claudeOAuthService) RefreshToken(ctx context.Context, refreshToken, proxyURL string) (*oauth.TokenResponse, error) {
	client := s.clientFactory(proxyURL)

	reqBody := map[string]any{
		"grant_type":    "refresh_token",
		"refresh_token": refreshToken,
		"client_id":     oauth.ClientID,
		"scope":         oauth.ScopeAPI,
	}

	var tokenResp oauth.TokenResponse

	resp, err := client.R().
		SetContext(ctx).
		SetHeader("Accept", "application/json, text/plain, */*").
		SetHeader("Content-Type", "application/json").
		SetHeader("User-Agent", "axios/1.8.4").
		SetBody(reqBody).
		SetSuccessResult(&tokenResp).
		Post(s.tokenURL)

	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if !resp.IsSuccessState() {
		return nil, fmt.Errorf("token refresh failed: status %d, body: %s", resp.StatusCode, resp.String())
	}

	return &tokenResp, nil
}

// sendPreOAuthHelloRequests sends hello requests before token exchange to mimic CLI behavior.
func (s *claudeOAuthService) sendPreOAuthHelloRequests(ctx context.Context, proxyURL string) {
	if s.versionService == nil {
		return
	}
	cliVersion := s.versionService.GetOrCreateCLIVersion(ctx, "__hello__")
	userAgent := fmt.Sprintf("claude-cli/%s (external, cli)", cliVersion)

	helloCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client := s.clientFactory(proxyURL)

	helloURLs := []string{
		"https://console.anthropic.com/v1/oauth/hello",
		"https://api.anthropic.com/api/hello",
	}

	for _, u := range helloURLs {
		resp, err := client.R().
			SetContext(helloCtx).
			SetHeader("Accept", "application/json, text/plain, */*").
			SetHeader("User-Agent", userAgent).
			SetHeader("Accept-Encoding", "gzip, compress, deflate, br").
			Get(u)
		if err != nil {
			log.Printf("[OAuth] Pre-hello %s failed: %v", u, err)
			continue
		}
		log.Printf("[OAuth] Pre-hello %s -> %d", u, resp.StatusCode)
	}
}

// SendPostOAuthRequests sends simulation requests after token exchange to mimic CLI behavior.
func (s *claudeOAuthService) SendPostOAuthRequests(ctx context.Context, accessToken, accountUUID, proxyURL string) {
	if s.versionService == nil {
		return
	}
	cliVersion := s.versionService.GetOrCreateCLIVersion(ctx, accountUUID)

	postCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	client := s.clientFactory(proxyURL)

	type postReq struct {
		url         string
		headerStyle string // "axios" or "claude-code"
		contentType bool
	}

	requests := []postReq{
		{"https://api.anthropic.com/api/oauth/profile", "axios", true},
		{"https://api.anthropic.com/api/oauth/claude_cli/roles", "axios", false},
		{"https://api.anthropic.com/api/organization/claude_code_first_token_date", "claude-code", false},
		{"https://api.anthropic.com/api/claude_code/organizations/metrics_enabled", "claude-code", true},
	}

	for _, r := range requests {
		req := client.R().
			SetContext(postCtx).
			SetHeader("Accept", "application/json, text/plain, */*").
			SetHeader("Accept-Encoding", "gzip, compress, deflate, br").
			SetHeader("Authorization", "Bearer "+accessToken)

		if r.headerStyle == "axios" {
			req.SetHeader("User-Agent", "axios/1.8.4")
		} else {
			req.SetHeader("User-Agent", fmt.Sprintf("claude-code/%s", cliVersion))
			req.SetHeader("anthropic-beta", "oauth-2025-04-20")
		}

		if r.contentType {
			req.SetHeader("Content-Type", "application/json")
		}

		resp, err := req.Get(r.url)
		if err != nil {
			log.Printf("[OAuth] Post-OAuth %s failed: %v", r.url, err)
			continue
		}
		log.Printf("[OAuth] Post-OAuth %s -> %d", r.url, resp.StatusCode)
	}
}

func createReqClient(proxyURL string) *req.Client {
	// 禁用 CookieJar，确保每次授权都是干净的会话
	client := req.C().
		SetTimeout(60 * time.Second).
		ImpersonateChrome().
		SetCookieJar(nil) // 禁用 CookieJar

	if strings.TrimSpace(proxyURL) != "" {
		client.SetProxyURL(strings.TrimSpace(proxyURL))
	}

	return client
}
