package service

import (
	"testing"

	"github.com/tidwall/gjson"
)

// --- detectDownstreamRequested1hCache：下游 1h 意图检测 ---

func TestDetectDownstreamRequested1hCache(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{"no ttl field", `{"model":"claude","messages":[{"role":"user","content":"hi"}]}`, false},
		{"system block 1h", `{"system":[{"type":"text","text":"x","cache_control":{"type":"ephemeral","ttl":"1h"}}]}`, true},
		{"system block 5m (default)", `{"system":[{"type":"text","text":"x","cache_control":{"type":"ephemeral","ttl":"5m"}}]}`, false},
		{"system block ephemeral no ttl", `{"system":[{"type":"text","text":"x","cache_control":{"type":"ephemeral"}}]}`, false},
		{"messages content block 1h", `{"messages":[{"role":"user","content":[{"type":"text","text":"x","cache_control":{"type":"ephemeral","ttl":"1h"}}]}]}`, true},
		{"tools block 1h", `{"tools":[{"name":"t","cache_control":{"type":"ephemeral","ttl":"1h"}}]}`, true},
		{"mixed: one 5m one 1h → true", `{"system":[{"cache_control":{"type":"ephemeral","ttl":"5m"}}],"tools":[{"cache_control":{"type":"ephemeral","ttl":"1h"}}]}`, true},
		{"ttl 1h in user text value, not cache_control → false", `{"messages":[{"role":"user","content":[{"type":"text","text":"please cache for 1h"}]}]}`, false},
		{"cache_control ttl 1h but no type=ephemeral → false (not a valid block)", `{"a":{"cache_control":{"ttl":"1h"}}}`, false},
		{"deeply nested valid 1h", `{"a":{"b":[{"c":{"cache_control":{"type":"ephemeral","ttl":"1h"}}}]}}`, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectDownstreamRequested1hCache([]byte(tc.body)); got != tc.want {
				t.Fatalf("detect(%s) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

// --- resolveCacheTTLDecision：决策真值表（分组级路径；account=nil） ---

func TestResolveCacheTTLDecision_GroupLevel(t *testing.T) {
	anthropicOn := &Group{Platform: PlatformAnthropic, ClaudeUnrequested1hCacheAs5m: true}
	anthropicOff := &Group{Platform: PlatformAnthropic, ClaudeUnrequested1hCacheAs5m: false}
	openaiOn := &Group{Platform: "openai", ClaudeUnrequested1hCacheAs5m: true}

	cases := []struct {
		name              string
		group             *Group
		downstreamReq1h   bool
		wantEnabled       bool
		wantTarget        string
	}{
		{"anthropic on, downstream not 1h → 5m", anthropicOn, false, true, "5m"},
		{"anthropic on, downstream requested 1h → off", anthropicOn, true, false, ""},
		{"anthropic off → off", anthropicOff, false, false, ""},
		{"openai on → off (platform gate)", openaiOn, false, false, ""},
		{"nil group → off", nil, false, false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dec := resolveCacheTTLDecision(nil, tc.group, tc.downstreamReq1h)
			if dec.Enabled != tc.wantEnabled || (tc.wantEnabled && dec.Target != tc.wantTarget) {
				t.Fatalf("decision=%+v, want enabled=%v target=%v", dec, tc.wantEnabled, tc.wantTarget)
			}
			if tc.wantEnabled && dec.Source != "group" {
				t.Fatalf("expected source=group, got %q", dec.Source)
			}
		})
	}
}

// --- rewriteSSEDataCacheTTL：SSE data 改写 ---

func TestRewriteSSEDataCacheTTL_5m(t *testing.T) {
	dec := cacheTTLDecision{Enabled: true, Target: "5m"}

	// message_start：1h=80,5m=20 → 全归 5m=100,1h=0
	in := `{"type":"message_start","message":{"usage":{"input_tokens":1,"cache_creation":{"ephemeral_5m_input_tokens":20,"ephemeral_1h_input_tokens":80}}}}`
	out, changed := rewriteSSEDataCacheTTL(in, dec)
	if !changed {
		t.Fatal("expected changed=true")
	}
	if g := gjson.Get(out, "message.usage.cache_creation.ephemeral_5m_input_tokens").Int(); g != 100 {
		t.Fatalf("5m = %d, want 100", g)
	}
	if g := gjson.Get(out, "message.usage.cache_creation.ephemeral_1h_input_tokens").Int(); g != 0 {
		t.Fatalf("1h = %d, want 0", g)
	}
	// 其它字段不动
	if g := gjson.Get(out, "message.usage.input_tokens").Int(); g != 1 {
		t.Fatalf("input_tokens mutated: %d", g)
	}

	// message_delta usage 路径
	inDelta := `{"type":"message_delta","usage":{"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":50}}}`
	outDelta, changedDelta := rewriteSSEDataCacheTTL(inDelta, dec)
	if !changedDelta || gjson.Get(outDelta, "usage.cache_creation.ephemeral_5m_input_tokens").Int() != 50 ||
		gjson.Get(outDelta, "usage.cache_creation.ephemeral_1h_input_tokens").Int() != 0 {
		t.Fatalf("delta rewrite failed: %s", outDelta)
	}
}

func TestRewriteSSEDataCacheTTL_NoopCases(t *testing.T) {
	dec := cacheTTLDecision{Enabled: true, Target: "5m"}
	noops := []string{
		`{"type":"content_block_delta","delta":{"text":"hi"}}`,                            // 非 start/delta
		`{"type":"message_start","message":{"usage":{"input_tokens":5}}}`,                 // 无 cache_creation
		`{"type":"message_start","message":{"usage":{"cache_creation":{"ephemeral_5m_input_tokens":30,"ephemeral_1h_input_tokens":0}}}}`, // 已全 5m
	}
	for _, in := range noops {
		if out, changed := rewriteSSEDataCacheTTL(in, dec); changed || out != in {
			t.Fatalf("expected no-op for %s, got changed=%v out=%s", in, changed, out)
		}
	}
	// 决策未生效
	if _, changed := rewriteSSEDataCacheTTL(`{"type":"message_start","message":{"usage":{"cache_creation":{"ephemeral_1h_input_tokens":9}}}}`, cacheTTLDecision{}); changed {
		t.Fatal("disabled decision must not change data")
	}
}

// --- rewriteNonStreamCacheTTLBody：非流式 body 改写 + usage 同步 ---

func TestRewriteNonStreamCacheTTLBody(t *testing.T) {
	dec := cacheTTLDecision{Enabled: true, Target: "5m"}
	body := []byte(`{"id":"m","usage":{"input_tokens":3,"cache_creation_input_tokens":100,"cache_creation":{"ephemeral_5m_input_tokens":40,"ephemeral_1h_input_tokens":60}}}`)
	usage := &ClaudeUsage{CacheCreation5mTokens: 40, CacheCreation1hTokens: 60, CacheCreationInputTokens: 100}

	nb, changed := rewriteNonStreamCacheTTLBody(body, usage, dec)
	if !changed {
		t.Fatal("expected changed")
	}
	// usage 同步为 5m
	if usage.CacheCreation5mTokens != 100 || usage.CacheCreation1hTokens != 0 {
		t.Fatalf("usage not synced: 5m=%d 1h=%d", usage.CacheCreation5mTokens, usage.CacheCreation1hTokens)
	}
	// body 同步
	if gjson.GetBytes(nb, "usage.cache_creation.ephemeral_5m_input_tokens").Int() != 100 ||
		gjson.GetBytes(nb, "usage.cache_creation.ephemeral_1h_input_tokens").Int() != 0 {
		t.Fatalf("body not synced: %s", nb)
	}
	// flat 总量不变
	if gjson.GetBytes(nb, "usage.cache_creation_input_tokens").Int() != 100 {
		t.Fatalf("flat cache_creation_input_tokens changed")
	}

	// 决策未生效 → 不改
	usage2 := &ClaudeUsage{CacheCreation1hTokens: 60}
	if _, changed := rewriteNonStreamCacheTTLBody(body, usage2, cacheTTLDecision{}); changed {
		t.Fatal("disabled decision must not change body")
	}
}
