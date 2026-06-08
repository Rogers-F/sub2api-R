package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

// ClaudeOAuthKey is the OAuth subscription credential stored verbatim in
// Channel.Key when a Claude(Anthropic) channel uses auth_mode="oauth". Anthropic
// OAuth is plain OAuth2 (access + refresh), NOT OIDC — there is deliberately no
// id_token field.
//
// It lives in package service (not relay/channel/claude) on purpose: the
// request-path lazy refresh forces claude -> service, and the refresh coordinator
// (also service) needs this type; keeping it in claude would create a
// service <-> claude import cycle. This mirrors how CodexOAuthKey lives here too.
type ClaudeOAuthKey struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`   // unix seconds
	AccountUUID  string `json:"account_uuid,omitempty"` // stable per-subscription identity; keys the refresh lock/cache
	OrgUUID      string `json:"org_uuid,omitempty"`
	Email        string `json:"email,omitempty"`
	Type         string `json:"type,omitempty"`           // "claude"
	LastRefresh  string `json:"last_refresh,omitempty"`   // RFC3339
	TokenVersion int64  `json:"_token_version,omitempty"` // epoch millis; logical version (CAS uses exact-key compare)
}

// ParseClaudeOAuthKey parses the JSON blob stored in Channel.Key.
func ParseClaudeOAuthKey(raw string) (*ClaudeOAuthKey, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("claude oauth: empty key")
	}
	var k ClaudeOAuthKey
	if err := common.Unmarshal([]byte(raw), &k); err != nil {
		return nil, errors.New("claude oauth: invalid key json")
	}
	return &k, nil
}

// RedactedSummary returns a secret-free summary of the credential (no access/refresh
// tokens) for control-plane responses that must never leak the token material. This is
// the single source of truth for "what is safe to expose about an OAuth credential".
func (k *ClaudeOAuthKey) RedactedSummary() map[string]any {
	if k == nil {
		return map[string]any{}
	}
	last4 := ""
	if n := len(k.AccessToken); n >= 4 {
		last4 = k.AccessToken[n-4:]
	}
	return map[string]any{
		"email":              k.Email,
		"account_uuid":       k.AccountUUID,
		"expires_at":         k.ExpiresAt,
		"access_token_last4": last4,
	}
}

// NeedsRefresh reports whether the access token is missing or expiring within skew.
func (k *ClaudeOAuthKey) NeedsRefresh(skew time.Duration) bool {
	if k == nil || k.AccessToken == "" || k.ExpiresAt <= 0 {
		return true
	}
	return time.Until(time.Unix(k.ExpiresAt, 0)) <= skew
}

// LockAccountID returns the stable identity used to key the refresh lock and token
// cache for this credential. account_uuid is preferred (siblings sharing one
// subscription must serialize); when absent it falls back to a hash of the refresh
// token (NEVER the channel id, which would re-open the rotation race).
func (k *ClaudeOAuthKey) LockAccountID() string {
	if k == nil {
		return ""
	}
	if k.AccountUUID != "" {
		return k.AccountUUID
	}
	if k.RefreshToken != "" {
		sum := sha256.Sum256([]byte(k.RefreshToken))
		return "rt:" + hex.EncodeToString(sum[:8])
	}
	return ""
}

// LooksLikeOAuthBlob reports whether a raw Channel.Key is an OAuth credential blob
// (a JSON object carrying an access_token or refresh_token), as opposed to a static
// api key. Used to reject OAuth credentials on control-plane paths that lack channel
// context (e.g. POST /api/channel/fetch_models).
func LooksLikeOAuthBlob(raw string) bool {
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "{") {
		return false
	}
	var probe struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return false
	}
	return probe.AccessToken != "" || probe.RefreshToken != ""
}

// MergeRefreshedTokens applies refreshed token values onto the existing raw blob while
// PRESERVING any unknown/sibling fields (map-preserve). refreshToken is written only
// when non-empty: Anthropic omits it when not rotating, and the prior value MUST survive
// (writing an empty string would brick the account). Returns the new Channel.Key string.
func MergeRefreshedTokens(raw, accessToken, refreshToken string, expiresAt, tokenVersion int64, lastRefresh string) (string, error) {
	m := map[string]json.RawMessage{}
	raw = strings.TrimSpace(raw)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			return "", errors.New("claude oauth: cannot merge into invalid key json")
		}
	}
	set := func(key string, val any) error {
		b, err := json.Marshal(val)
		if err != nil {
			return err
		}
		m[key] = b
		return nil
	}
	if err := set("access_token", accessToken); err != nil {
		return "", err
	}
	if refreshToken != "" {
		if err := set("refresh_token", refreshToken); err != nil {
			return "", err
		}
	}
	if err := set("expires_at", expiresAt); err != nil {
		return "", err
	}
	if err := set("_token_version", tokenVersion); err != nil {
		return "", err
	}
	if lastRefresh != "" {
		if err := set("last_refresh", lastRefresh); err != nil {
			return "", err
		}
	}
	out, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(out), nil
}
