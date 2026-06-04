// Package newapi is a thin client for the external new-api admin API used by
// the same-origin chat relay (BFF). It provisions and manages one relay token
// per platform user.
//
// Security: the admin access token and any per-user relay key are secrets.
// This package never logs them and never returns the access token to callers.
package newapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// defaultTimeout bounds every admin API call. Chat streaming does NOT go
// through this client (the relay handler proxies that directly), so a short
// timeout is appropriate here.
const defaultTimeout = 15 * time.Second

// maxAdminResponseBytes caps admin API response bodies to avoid unbounded reads.
const maxAdminResponseBytes = 1 << 20 // 1MB

// Config carries the connection parameters for the new-api admin API.
type Config struct {
	BaseURL      string
	AccessToken  string
	AdminUserID  int
	DefaultGroup string
}

// Client talks to the new-api admin API.
type Client struct {
	baseURL      string
	accessToken  string
	adminUserID  int
	defaultGroup string
	httpClient   *http.Client
}

// NewClient builds a Client. A nil httpClient falls back to a default with a
// bounded timeout.
func NewClient(cfg Config, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	group := strings.TrimSpace(cfg.DefaultGroup)
	if group == "" {
		group = "default"
	}
	return &Client{
		baseURL:      strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"),
		accessToken:  strings.TrimSpace(cfg.AccessToken),
		adminUserID:  cfg.AdminUserID,
		defaultGroup: group,
		httpClient:   httpClient,
	}
}

// Configured reports whether the client has the minimum config to operate.
func (c *Client) Configured() bool {
	return c.baseURL != "" && c.accessToken != ""
}

// TokenItem is a single token entry from the list endpoint. The key returned
// by the list endpoint is masked and unusable for chat.
type TokenItem struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Key  string `json:"key"`
}

// envelope is the common new-api response wrapper.
type envelope struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

// apiStatusError carries the upstream HTTP status for non-2xx responses so
// callers can treat specific codes (e.g. 404 on delete) as idempotent.
type apiStatusError struct{ code int }

func (e *apiStatusError) Error() string { return fmt.Sprintf("newapi returned status %d", e.code) }

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	// Auth headers required on EVERY admin call.
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("New-Api-User", strconv.Itoa(c.adminUserID))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// do executes the request and decodes the standard envelope. It deliberately
// never includes the response body verbatim in errors that might echo secrets;
// admin endpoints used here do not return secrets in error paths, but we still
// keep error text minimal.
func (c *Client) do(req *http.Request) (*envelope, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("newapi request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxAdminResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("read newapi response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &apiStatusError{code: resp.StatusCode}
	}

	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("decode newapi response: %w", err)
	}
	if !env.Success {
		msg := strings.TrimSpace(env.Message)
		if msg == "" {
			msg = "request not successful"
		}
		return nil, fmt.Errorf("newapi: %s", msg)
	}
	return &env, nil
}

// CreateToken creates a new unlimited, non-expiring token named `name`.
// new-api does NOT return the key or id from this call.
func (c *Client) CreateToken(ctx context.Context, name string) error {
	body := map[string]any{
		"name":            name,
		"expired_time":    -1,
		"unlimited_quota": true,
		"group":           c.defaultGroup,
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/api/token/", body)
	if err != nil {
		return err
	}
	if _, err := c.do(req); err != nil {
		return err
	}
	return nil
}

// FindTokenIDByName scans the token list (all pages) and returns the id of the
// one matching `name`. Returns (0, nil) when no token with that name exists.
// Paginating fully matters: the deterministic per-user token may not be on the
// first page, so a single-page scan could miss an existing/just-created token
// and break the "reuse by name" idempotency guarantee.
func (c *Client) FindTokenIDByName(ctx context.Context, name string) (int, error) {
	const pageSize = 100
	const maxPages = 1000 // hard backstop against an unbounded loop
	for page := 1; page <= maxPages; page++ {
		req, err := c.newRequest(ctx, http.MethodGet,
			"/api/token/?p="+strconv.Itoa(page)+"&page_size="+strconv.Itoa(pageSize), nil)
		if err != nil {
			return 0, err
		}
		env, err := c.do(req)
		if err != nil {
			return 0, err
		}
		var payload struct {
			Items []TokenItem `json:"items"`
		}
		if len(env.Data) > 0 {
			if err := json.Unmarshal(env.Data, &payload); err != nil {
				return 0, fmt.Errorf("decode token list: %w", err)
			}
		}
		for i := range payload.Items {
			if payload.Items[i].Name == name {
				return payload.Items[i].ID, nil
			}
		}
		// Last page reached (fewer than a full page, or empty).
		if len(payload.Items) < pageSize {
			break
		}
	}
	return 0, nil
}

// GetTokenKey fetches the full plaintext key body for the token id and returns
// the usable API key prefixed with "sk-". The caller must treat it as a secret.
func (c *Client) GetTokenKey(ctx context.Context, id int) (string, error) {
	if id <= 0 {
		return "", errors.New("invalid token id")
	}
	req, err := c.newRequest(ctx, http.MethodPost, "/api/token/"+strconv.Itoa(id)+"/key", nil)
	if err != nil {
		return "", err
	}
	env, err := c.do(req)
	if err != nil {
		return "", err
	}
	var payload struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(env.Data, &payload); err != nil {
		return "", fmt.Errorf("decode token key: %w", err)
	}
	key := strings.TrimSpace(payload.Key)
	if key == "" {
		return "", errors.New("newapi returned empty token key")
	}
	return "sk-" + key, nil
}

// DeleteToken removes a token by id.
func (c *Client) DeleteToken(ctx context.Context, id int) error {
	if id <= 0 {
		return errors.New("invalid token id")
	}
	req, err := c.newRequest(ctx, http.MethodDelete, "/api/token/"+strconv.Itoa(id), nil)
	if err != nil {
		return err
	}
	if _, err := c.do(req); err != nil {
		// A 404 means the token is already gone -> treat delete as idempotent
		// success rather than a revoke failure.
		var se *apiStatusError
		if errors.As(err, &se) && se.code == http.StatusNotFound {
			return nil
		}
		return err
	}
	return nil
}
