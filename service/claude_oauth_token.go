package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// Anthropic (Claude.ai) OAuth subscription endpoints + client fingerprint. These
// MUST match the real Claude CLI exactly or platform.claude.com rejects the refresh
// / the subscription account gets correlation-flagged. Unlike Codex, Anthropic's
// token endpoint takes a JSON body (not form-encoded) and expects the axios UA.
const (
	claudeOAuthClientID      = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	claudeOAuthTokenURL      = "https://platform.claude.com/v1/oauth/token"
	claudeOAuthRefreshUA     = "axios/1.8.4"
	claudeOAuthRefreshAccept = "application/json, text/plain, */*"
)

// ClaudeOAuthTokenResult is the normalized result of a token refresh / code exchange.
// RefreshToken may be empty: Anthropic omits it when it does not rotate, and the
// caller MUST preserve the prior refresh_token in that case (see MergeRefreshedTokens).
type ClaudeOAuthTokenResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	AccountUUID  string
	OrgUUID      string
	Email        string
}

// RefreshClaudeOAuthTokenWithProxy refreshes an Anthropic OAuth access token,
// optionally through the channel's proxy (the same egress the relay uses).
func RefreshClaudeOAuthTokenWithProxy(ctx context.Context, refreshToken string, proxyURL string) (*ClaudeOAuthTokenResult, error) {
	rt := strings.TrimSpace(refreshToken)
	if rt == "" {
		return nil, errors.New("claude oauth: empty refresh_token")
	}
	client, err := getClaudeOAuthHTTPClient(proxyURL)
	if err != nil {
		return nil, err
	}

	// Anthropic's token endpoint takes a JSON body (NOT x-www-form-urlencoded).
	bodyBytes, err := json.Marshal(map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": rt,
		"client_id":     claudeOAuthClientID,
	})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeOAuthTokenURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", claudeOAuthRefreshAccept)
	req.Header.Set("User-Agent", claudeOAuthRefreshUA)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Check status BEFORE decoding: a non-2xx body is an error payload, not the success
	// shape, so decoding it first would lose the status semantics.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("claude oauth refresh failed: status=%d", resp.StatusCode)
	}
	var out struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		Organization struct {
			UUID string `json:"uuid"`
		} `json:"organization"`
		Account struct {
			UUID         string `json:"uuid"`
			EmailAddress string `json:"email_address"`
		} `json:"account"`
	}
	if err := common.DecodeJson(resp.Body, &out); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.AccessToken) == "" || out.ExpiresIn <= 0 {
		return nil, errors.New("claude oauth refresh response missing fields")
	}

	return &ClaudeOAuthTokenResult{
		AccessToken:  strings.TrimSpace(out.AccessToken),
		RefreshToken: strings.TrimSpace(out.RefreshToken),
		ExpiresAt:    time.Now().Add(time.Duration(out.ExpiresIn) * time.Second),
		AccountUUID:  strings.TrimSpace(out.Account.UUID),
		OrgUUID:      strings.TrimSpace(out.Organization.UUID),
		Email:        strings.TrimSpace(out.Account.EmailAddress),
	}, nil
}

// NewClaudeOAuthKeyJSON builds the initial Channel.Key blob from a successful token
// refresh (used by the admin import flow). fallbackRefresh preserves the pasted
// refresh_token when Anthropic did not rotate it in the response.
func NewClaudeOAuthKeyJSON(res *ClaudeOAuthTokenResult, fallbackRefresh string) (string, error) {
	if res == nil {
		return "", errors.New("claude oauth: nil token result")
	}
	refresh := strings.TrimSpace(res.RefreshToken)
	if refresh == "" {
		refresh = strings.TrimSpace(fallbackRefresh)
	}
	k := ClaudeOAuthKey{
		AccessToken:  res.AccessToken,
		RefreshToken: refresh,
		ExpiresAt:    res.ExpiresAt.Unix(),
		AccountUUID:  res.AccountUUID,
		OrgUUID:      res.OrgUUID,
		Email:        res.Email,
		Type:         "claude",
		LastRefresh:  time.Now().Format(time.RFC3339),
		TokenVersion: time.Now().UnixMilli(),
	}
	b, err := common.Marshal(&k)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func getClaudeOAuthHTTPClient(proxyURL string) (*http.Client, error) {
	baseClient, err := GetHttpClientWithProxy(strings.TrimSpace(proxyURL))
	if err != nil {
		return nil, err
	}
	if baseClient == nil {
		return &http.Client{Timeout: defaultHTTPTimeout}, nil
	}
	clientCopy := *baseClient
	clientCopy.Timeout = defaultHTTPTimeout
	return &clientCopy, nil
}
