package service

import (
	"strings"
	"testing"
	"time"
)

// TestClaudeOAuthLockTTLExceedsRefreshTimeout guards the §5.10 invariant that the
// cross-instance refresh lock cannot expire mid-refresh (which would let a peer instance
// double-refresh and race the refresh_token rotation).
func TestClaudeOAuthLockTTLExceedsRefreshTimeout(t *testing.T) {
	const safetyMargin = 30 * time.Second
	if claudeOAuthLockTTL <= claudeOAuthRefreshTime+safetyMargin {
		t.Fatalf("lock TTL (%s) must exceed refresh timeout (%s) by at least %s",
			claudeOAuthLockTTL, claudeOAuthRefreshTime, safetyMargin)
	}
}

// TestMergeRefreshedTokensPreservesRefreshTokenWhenEmpty is the brick-prevention
// invariant: when Anthropic does not rotate (empty refresh_token in the response) the
// prior refresh_token, account_uuid and any unknown fields MUST survive.
func TestMergeRefreshedTokensPreservesRefreshTokenWhenEmpty(t *testing.T) {
	raw := `{"access_token":"A0","refresh_token":"R0","account_uuid":"acct-1","custom":"keep"}`
	out, err := MergeRefreshedTokens(raw, "A1", "", 123, 456, "2026-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	k, err := ParseClaudeOAuthKey(out)
	if err != nil {
		t.Fatal(err)
	}
	if k.AccessToken != "A1" {
		t.Fatalf("access_token not updated: %q", k.AccessToken)
	}
	if k.RefreshToken != "R0" {
		t.Fatalf("refresh_token must be preserved when not rotated, got %q", k.RefreshToken)
	}
	if k.AccountUUID != "acct-1" {
		t.Fatalf("account_uuid lost: %q", k.AccountUUID)
	}
	if k.TokenVersion != 456 {
		t.Fatalf("token version not set: %d", k.TokenVersion)
	}
	if !strings.Contains(out, `"custom":"keep"`) {
		t.Fatalf("unknown field not preserved: %s", out)
	}
}

func TestMergeRefreshedTokensRotatesWhenPresent(t *testing.T) {
	out, err := MergeRefreshedTokens(`{"access_token":"A0","refresh_token":"R0"}`, "A1", "R1", 1, 2, "")
	if err != nil {
		t.Fatal(err)
	}
	k, _ := ParseClaudeOAuthKey(out)
	if k.RefreshToken != "R1" {
		t.Fatalf("refresh_token must rotate when present, got %q", k.RefreshToken)
	}
}

func TestLooksLikeOAuthBlob(t *testing.T) {
	cases := map[string]bool{
		`{"access_token":"x"}`:  true,
		`{"refresh_token":"x"}`: true,
		`sk-ant-static-key`:     false,
		`{"foo":"bar"}`:         false,
		``:                      false,
	}
	for in, want := range cases {
		if got := LooksLikeOAuthBlob(in); got != want {
			t.Fatalf("LooksLikeOAuthBlob(%q)=%v want %v", in, got, want)
		}
	}
}
