package claude

import (
	"testing"

	"github.com/tidwall/gjson"
)

func TestOAuthBodyTransformBannerAndIdempotent(t *testing.T) {
	body := []byte(`{"model":"claude-sonnet-4-5","messages":[{"role":"user","content":"hi"}],"temperature":0.7,"tool_choice":{"type":"auto"}}`)
	opts := OAuthTransformOptions{MetadataUserID: "user_abc_account_x_session_y"}
	out := OAuthBodyTransform(body, opts)

	if got := gjson.GetBytes(out, "system.0.text").String(); got != ClaudeCodeSystemPrompt {
		t.Fatalf("banner not injected as first system block, got %q", got)
	}
	// Faithful to sub2api: the banner is injected WITH cache_control, then
	// stripSystemCacheControl removes cache_control from all system blocks (banner
	// included). So the final banner carries no cache_control.
	if gjson.GetBytes(out, "system.0.cache_control").Exists() {
		t.Fatalf("system cache_control should be stripped on the OAuth path: %s", out)
	}
	if got := gjson.GetBytes(out, "model").String(); got != "claude-sonnet-4-5-20250929" {
		t.Fatalf("model not normalized short->long, got %q", got)
	}
	if !gjson.GetBytes(out, "tools").Exists() {
		t.Fatalf("tools[] not ensured")
	}
	if gjson.GetBytes(out, "temperature").Exists() {
		t.Fatalf("temperature not stripped")
	}
	if gjson.GetBytes(out, "tool_choice").Exists() {
		t.Fatalf("tool_choice not stripped")
	}
	if got := gjson.GetBytes(out, "metadata.user_id").String(); got != "user_abc_account_x_session_y" {
		t.Fatalf("metadata.user_id not injected, got %q", got)
	}

	// Byte-idempotent: re-running on its own output must be a no-op (the §5.10 invariant
	// that lets the relay retry loop re-apply it without a context flag).
	out2 := OAuthBodyTransform(out, opts)
	if string(out2) != string(out) {
		t.Fatalf("transform not idempotent:\n first=%s\nsecond=%s", out, out2)
	}
}

func TestOAuthBodyTransformHaikuSkipsBanner(t *testing.T) {
	body := []byte(`{"model":"claude-haiku-4-5","messages":[{"role":"user","content":"hi"}]}`)
	out := OAuthBodyTransform(body, OAuthTransformOptions{})
	if systemIncludesClaudeCodePrompt(out) {
		t.Fatalf("haiku must not get the Claude Code banner: %s", out)
	}
	if got := gjson.GetBytes(out, "model").String(); got != "claude-haiku-4-5-20251001" {
		t.Fatalf("haiku model not normalized, got %q", got)
	}
	if !gjson.GetBytes(out, "tools").Exists() {
		t.Fatalf("tools[] not ensured for haiku")
	}
}

func TestOAuthBodyTransformPreservesClientMetadata(t *testing.T) {
	body := []byte(`{"model":"claude-opus-4-5","messages":[],"metadata":{"user_id":"client-provided"}}`)
	out := OAuthBodyTransform(body, OAuthTransformOptions{MetadataUserID: "should-not-override"})
	if got := gjson.GetBytes(out, "metadata.user_id").String(); got != "client-provided" {
		t.Fatalf("client metadata.user_id must be preserved, got %q", got)
	}
}

func TestEnforceCacheControlLimit(t *testing.T) {
	// 1 system + 4 message cache_control blocks = 5 > 4 -> one message block removed.
	body := []byte(`{"system":[{"type":"text","text":"x","cache_control":{"type":"ephemeral"}}],` +
		`"messages":[{"role":"user","content":[` +
		`{"type":"text","text":"a","cache_control":{"type":"ephemeral"}},` +
		`{"type":"text","text":"b","cache_control":{"type":"ephemeral"}},` +
		`{"type":"text","text":"c","cache_control":{"type":"ephemeral"}},` +
		`{"type":"text","text":"d","cache_control":{"type":"ephemeral"}}]}]}`)
	out := enforceCacheControlLimit(body)
	count := 0
	gjson.GetBytes(out, "system").ForEach(func(_, item gjson.Result) bool {
		if item.Get("cache_control").Exists() {
			count++
		}
		return true
	})
	gjson.GetBytes(out, "messages").ForEach(func(_, msg gjson.Result) bool {
		msg.Get("content").ForEach(func(_, item gjson.Result) bool {
			if item.Get("cache_control").Exists() {
				count++
			}
			return true
		})
		return true
	})
	if count != 4 {
		t.Fatalf("expected 4 cache_control blocks after limit, got %d: %s", count, out)
	}
}

func TestEnforceCacheControlStripsThinkingCacheControl(t *testing.T) {
	body := []byte(`{"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"...","cache_control":{"type":"ephemeral"}}]}]}`)
	out := enforceCacheControlLimit(body)
	if gjson.GetBytes(out, "messages.0.content.0.cache_control").Exists() {
		t.Fatalf("thinking-block cache_control must be stripped: %s", out)
	}
}
