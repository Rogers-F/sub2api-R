package claude

// OAuth subscription (auth_mode="oauth") relay constants, ported faithfully from
// sub2api so that requests forwarded with a Claude.ai OAuth access token are accepted
// by api.anthropic.com and the underlying subscription account is not correlation-flagged.
// These apply ONLY on the OAuth path; the legacy x-api-key path is unchanged.

// ClaudeCodeSystemPrompt is the exact banner that must be the first system block for a
// Claude-Code-scoped OAuth token to be accepted. Keep EXACT (no trailing whitespace);
// separators are added as "\n\n" at concatenation time.
const ClaudeCodeSystemPrompt = "You are Claude Code, Anthropic's official CLI for Claude."

// anthropic-beta tokens used on the OAuth path. The claude-code beta is deliberately
// NOT included: the /v1/messages mimic branch drops it to match real CLI traffic.
const (
	BetaOAuth               = "oauth-2025-04-20"
	BetaInterleavedThinking = "interleaved-thinking-2025-05-14"
)

// HaikuBetaHeader is the anthropic-beta value for haiku models (same set as the mimic
// branch: oauth + interleaved-thinking, claude-code dropped).
const HaikuBetaHeader = BetaOAuth + "," + BetaInterleavedThinking

// MimicMessageBetas: the /v1/messages mimic branch sets exactly these, with the
// claude-code beta deliberately dropped to match real CLI traffic.
var MimicMessageBetas = []string{BetaOAuth, BetaInterleavedThinking}

// DefaultHeaders is the Claude CLI / Stainless fingerprint applied to OAuth requests
// (verbatim from sub2api). On the mimic path these are force-set; otherwise fill-if-empty.
var DefaultHeaders = map[string]string{
	"User-Agent":                                "claude-cli/2.1.22 (external, cli)",
	"X-Stainless-Lang":                          "js",
	"X-Stainless-Package-Version":               "0.70.0",
	"X-Stainless-OS":                            "Linux",
	"X-Stainless-Arch":                          "arm64",
	"X-Stainless-Runtime":                       "node",
	"X-Stainless-Runtime-Version":               "v24.13.0",
	"X-Stainless-Retry-Count":                   "0",
	"X-Stainless-Timeout":                       "600",
	"X-App":                                     "cli",
	"Anthropic-Dangerous-Direct-Browser-Access": "true",
}

// OAuthAllowedHeaders is the lowercase allowlist of inbound client headers permitted to
// reach the OAuth upstream. The final outbound sanitizer strips everything not in this
// set (including x-api-key, authorization, cookie, host). "authorization" is deliberately
// ABSENT: it is set explicitly to the fresh Bearer token, never passed through.
var OAuthAllowedHeaders = map[string]bool{
	"accept":                                    true,
	"x-stainless-retry-count":                   true,
	"x-stainless-timeout":                       true,
	"x-stainless-lang":                          true,
	"x-stainless-package-version":               true,
	"x-stainless-os":                            true,
	"x-stainless-arch":                          true,
	"x-stainless-runtime":                       true,
	"x-stainless-runtime-version":               true,
	"x-stainless-helper-method":                 true,
	"anthropic-dangerous-direct-browser-access": true,
	"anthropic-version":                         true,
	"x-app":                                     true,
	"anthropic-beta":                            true,
	"accept-language":                           true,
	"sec-fetch-mode":                            true,
	"user-agent":                                true,
	"content-type":                              true,
}

// ModelIDOverrides is the short->long OAuth model normalization. REQUIRED for OAuth
// (a Claude-Code-scoped token expects dated model IDs); the legacy apikey path does NOT
// normalize. Only the three dated short aliases are mapped; full IDs are identity.
var ModelIDOverrides = map[string]string{
	"claude-sonnet-4-5": "claude-sonnet-4-5-20250929",
	"claude-opus-4-5":   "claude-opus-4-5-20251101",
	"claude-haiku-4-5":  "claude-haiku-4-5-20251001",
}

// NormalizeModelID maps a short OAuth model alias to its dated upstream ID; identity otherwise.
func NormalizeModelID(id string) string {
	if id == "" {
		return id
	}
	if mapped, ok := ModelIDOverrides[id]; ok {
		return mapped
	}
	return id
}
