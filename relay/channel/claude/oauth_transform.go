package claude

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// maxCacheControlBlocks is the maximum number of cache_control blocks Anthropic allows.
const maxCacheControlBlocks = 4

// claudeCodePromptPrefixes detect an already-present Claude Code system prompt (any
// known variant). Ported verbatim from sub2api; prefixes must not contain one another.
var claudeCodePromptPrefixes = []string{
	"You are Claude Code, Anthropic's official CLI for Claude",
	"You are a Claude agent, built on Anthropic's Claude Agent SDK",
	"You are a file search specialist for Claude Code",
	"You are a helpful AI assistant tasked with summarizing conversations",
}

type anthropicCacheControlPayload struct {
	Type string `json:"type"`
}

type anthropicSystemTextBlockPayload struct {
	Type         string                        `json:"type"`
	Text         string                        `json:"text"`
	CacheControl *anthropicCacheControlPayload `json:"cache_control,omitempty"`
}

type anthropicMetadataPayload struct {
	UserID string `json:"user_id"`
}

type claudeOAuthNormalizeOptions struct {
	injectMetadata          bool
	metadataUserID          string
	stripSystemCacheControl bool
}

// OAuthTransformOptions carries the per-request inputs the transform cannot derive
// from the body alone.
type OAuthTransformOptions struct {
	// MetadataUserID, when non-empty, is injected as metadata.user_id IF the body does
	// not already carry one. Compute it via BuildOAuthMetadataUserID.
	MetadataUserID string
}

// OAuthBodyTransform applies the full Claude-Code "mimic" transform to an outbound
// /v1/messages body for an OAuth subscription account: inject the Claude Code banner
// (non-haiku), normalize the model short->long, ensure tools[], inject metadata.user_id
// when absent, drop temperature/tool_choice, and enforce the cache_control limit.
//
// It is byte-idempotent — re-running it on its own output is a no-op (banner dedup +
// metadata-only-if-absent + identity model mapping) — so it is safe under the relay
// retry loop WITHOUT a context flag.
func OAuthBodyTransform(body []byte, opts OAuthTransformOptions) []byte {
	if len(body) == 0 {
		return body
	}
	reqModel := gjson.GetBytes(body, "model").String()
	isHaiku := strings.Contains(strings.ToLower(reqModel), "haiku")

	// Haiku skips ONLY the banner injection; all other normalization still applies.
	if !isHaiku && !systemIncludesClaudeCodePrompt(body) {
		body = injectClaudeCodePrompt(body)
	}

	nopts := claudeOAuthNormalizeOptions{stripSystemCacheControl: true}
	if strings.TrimSpace(opts.MetadataUserID) != "" {
		nopts.injectMetadata = true
		nopts.metadataUserID = opts.MetadataUserID
	}
	body, _ = normalizeClaudeOAuthRequestBody(body, reqModel, nopts)
	body = enforceCacheControlLimit(body)
	return body
}

// --- system banner ----------------------------------------------------------------

func injectClaudeCodePrompt(body []byte) []byte {
	claudeCodeBlock, err := marshalAnthropicSystemTextBlock(ClaudeCodeSystemPrompt, true)
	if err != nil {
		return body
	}
	prefix := strings.TrimSpace(ClaudeCodeSystemPrompt)
	sys := gjson.GetBytes(body, "system")

	var items [][]byte
	switch {
	case !sys.Exists() || sys.Type == gjson.Null:
		items = [][]byte{claudeCodeBlock}
	case sys.Type == gjson.String:
		v := sys.String()
		if strings.TrimSpace(v) == "" || strings.TrimSpace(v) == prefix {
			items = [][]byte{claudeCodeBlock}
		} else {
			merged := v
			if !strings.HasPrefix(v, prefix) {
				merged = prefix + "\n\n" + v
			}
			nextBlock, buildErr := marshalAnthropicSystemTextBlock(merged, false)
			if buildErr != nil {
				return body
			}
			items = [][]byte{claudeCodeBlock, nextBlock}
		}
	case sys.IsArray():
		items = append(items, claudeCodeBlock)
		prefixedNext := false
		sys.ForEach(func(_, item gjson.Result) bool {
			textResult := item.Get("text")
			if textResult.Exists() && textResult.Type == gjson.String &&
				strings.TrimSpace(textResult.String()) == prefix {
				return true // drop a duplicate banner entry
			}
			raw := []byte(item.Raw)
			// Prefix the first subsequent text block once (matches sub2api/opencode).
			if !prefixedNext && item.Get("type").String() == "text" && textResult.Exists() && textResult.Type == gjson.String {
				text := textResult.String()
				if strings.TrimSpace(text) != "" && !strings.HasPrefix(text, prefix) {
					if next, setErr := sjson.SetBytes(raw, "text", prefix+"\n\n"+text); setErr == nil {
						raw = next
						prefixedNext = true
					}
				}
			}
			items = append(items, raw)
			return true
		})
	default:
		items = [][]byte{claudeCodeBlock}
	}

	result, ok := setJSONRawBytes(body, "system", buildJSONArrayRaw(items))
	if !ok {
		return body
	}
	return result
}

func systemIncludesClaudeCodePrompt(body []byte) bool {
	sys := gjson.GetBytes(body, "system")
	switch {
	case sys.Type == gjson.String:
		return hasClaudeCodePrefix(sys.String())
	case sys.IsArray():
		found := false
		sys.ForEach(func(_, item gjson.Result) bool {
			t := item.Get("text")
			if t.Exists() && t.Type == gjson.String && hasClaudeCodePrefix(t.String()) {
				found = true
				return false
			}
			return true
		})
		return found
	}
	return false
}

func hasClaudeCodePrefix(text string) bool {
	for _, prefix := range claudeCodePromptPrefixes {
		if strings.HasPrefix(text, prefix) {
			return true
		}
	}
	return false
}

// sanitizeSystemText rewrites only the fixed OpenCode identity sentence (if present) to
// the canonical Claude Code banner; it never does broad keyword replacement.
func sanitizeSystemText(text string) string {
	if text == "" {
		return text
	}
	return strings.ReplaceAll(
		text,
		"You are OpenCode, the best coding agent on the planet.",
		strings.TrimSpace(ClaudeCodeSystemPrompt),
	)
}

// --- body normalization ------------------------------------------------------------

func normalizeClaudeOAuthRequestBody(body []byte, modelID string, opts claudeOAuthNormalizeOptions) ([]byte, string) {
	if len(body) == 0 {
		return body, modelID
	}
	out := body
	modified := false

	if next, changed := normalizeClaudeOAuthSystemBody(out, opts); changed {
		out = next
		modified = true
	}

	rawModel := gjson.GetBytes(out, "model")
	if rawModel.Exists() && rawModel.Type == gjson.String {
		normalized := NormalizeModelID(rawModel.String())
		if normalized != rawModel.String() {
			if next, ok := setJSONValueBytes(out, "model", normalized); ok {
				out = next
				modified = true
			}
			modelID = normalized
		}
	}

	// Ensure tools exists (even as an empty array) — a Claude-Code-scoped token expects it.
	if !gjson.GetBytes(out, "tools").Exists() {
		if next, ok := setJSONRawBytes(out, "tools", []byte("[]")); ok {
			out = next
			modified = true
		}
	}

	if opts.injectMetadata && opts.metadataUserID != "" {
		if next, changed := ensureClaudeOAuthMetadataUserID(out, opts.metadataUserID); changed {
			out = next
			modified = true
		}
	}

	if gjson.GetBytes(out, "temperature").Exists() {
		if next, ok := deleteJSONPathBytes(out, "temperature"); ok {
			out = next
			modified = true
		}
	}
	if gjson.GetBytes(out, "tool_choice").Exists() {
		if next, ok := deleteJSONPathBytes(out, "tool_choice"); ok {
			out = next
			modified = true
		}
	}

	if !modified {
		return body, modelID
	}
	return out, modelID
}

func normalizeClaudeOAuthSystemBody(body []byte, opts claudeOAuthNormalizeOptions) ([]byte, bool) {
	sys := gjson.GetBytes(body, "system")
	if !sys.Exists() {
		return body, false
	}
	out := body
	modified := false

	switch {
	case sys.Type == gjson.String:
		sanitized := sanitizeSystemText(sys.String())
		if sanitized != sys.String() {
			if next, ok := setJSONValueBytes(out, "system", sanitized); ok {
				out = next
				modified = true
			}
		}
	case sys.IsArray():
		index := 0
		sys.ForEach(func(_, item gjson.Result) bool {
			if item.Get("type").String() == "text" {
				textResult := item.Get("text")
				if textResult.Exists() && textResult.Type == gjson.String {
					text := textResult.String()
					sanitized := sanitizeSystemText(text)
					if sanitized != text {
						if next, ok := setJSONValueBytes(out, fmt.Sprintf("system.%d.text", index), sanitized); ok {
							out = next
							modified = true
						}
					}
				}
			}
			if opts.stripSystemCacheControl && item.Get("cache_control").Exists() {
				if next, ok := deleteJSONPathBytes(out, fmt.Sprintf("system.%d.cache_control", index)); ok {
					out = next
					modified = true
				}
			}
			index++
			return true
		})
	}

	return out, modified
}

func ensureClaudeOAuthMetadataUserID(body []byte, userID string) ([]byte, bool) {
	if strings.TrimSpace(userID) == "" {
		return body, false
	}
	metadata := gjson.GetBytes(body, "metadata")
	if !metadata.Exists() || metadata.Type == gjson.Null {
		raw, err := marshalAnthropicMetadata(userID)
		if err != nil {
			return body, false
		}
		return setJSONRawBytes(body, "metadata", raw)
	}

	trimmedRaw := strings.TrimSpace(metadata.Raw)
	if strings.HasPrefix(trimmedRaw, "{") {
		existing := metadata.Get("user_id")
		if existing.Exists() && existing.Type == gjson.String && existing.String() != "" {
			return body, false
		}
		return setJSONValueBytes(body, "metadata.user_id", userID)
	}

	raw, err := marshalAnthropicMetadata(userID)
	if err != nil {
		return body, false
	}
	return setJSONRawBytes(body, "metadata", raw)
}

// --- cache_control limit -----------------------------------------------------------

func collectCacheControlPaths(body []byte) (invalidThinking []string, messagePaths []string, systemPaths []string) {
	system := gjson.GetBytes(body, "system")
	if system.IsArray() {
		sysIndex := 0
		system.ForEach(func(_, item gjson.Result) bool {
			if item.Get("cache_control").Exists() {
				path := fmt.Sprintf("system.%d.cache_control", sysIndex)
				if item.Get("type").String() == "thinking" {
					invalidThinking = append(invalidThinking, path)
				} else {
					systemPaths = append(systemPaths, path)
				}
			}
			sysIndex++
			return true
		})
	}

	messages := gjson.GetBytes(body, "messages")
	if messages.IsArray() {
		msgIndex := 0
		messages.ForEach(func(_, msg gjson.Result) bool {
			content := msg.Get("content")
			if content.IsArray() {
				contentIndex := 0
				content.ForEach(func(_, item gjson.Result) bool {
					if item.Get("cache_control").Exists() {
						path := fmt.Sprintf("messages.%d.content.%d.cache_control", msgIndex, contentIndex)
						if item.Get("type").String() == "thinking" {
							invalidThinking = append(invalidThinking, path)
						} else {
							messagePaths = append(messagePaths, path)
						}
					}
					contentIndex++
					return true
				})
			}
			msgIndex++
			return true
		})
	}

	return invalidThinking, messagePaths, systemPaths
}

// enforceCacheControlLimit caps cache_control blocks at maxCacheControlBlocks, removing
// from messages first (preferring to keep system caches) and stripping the illegal
// cache_control field from thinking blocks (which do not support it).
func enforceCacheControlLimit(body []byte) []byte {
	if len(body) == 0 {
		return body
	}

	invalidThinking, messagePaths, systemPaths := collectCacheControlPaths(body)
	out := body
	modified := false

	for _, path := range invalidThinking {
		if !gjson.GetBytes(out, path).Exists() {
			continue
		}
		if next, ok := deleteJSONPathBytes(out, path); ok {
			out = next
			modified = true
		}
	}

	count := len(messagePaths) + len(systemPaths)
	if count <= maxCacheControlBlocks {
		if modified {
			return out
		}
		return body
	}

	remaining := count - maxCacheControlBlocks
	for _, path := range messagePaths {
		if remaining <= 0 {
			break
		}
		if !gjson.GetBytes(out, path).Exists() {
			continue
		}
		if next, ok := deleteJSONPathBytes(out, path); ok {
			out = next
			modified = true
			remaining--
		}
	}
	for i := len(systemPaths) - 1; i >= 0 && remaining > 0; i-- {
		path := systemPaths[i]
		if !gjson.GetBytes(out, path).Exists() {
			continue
		}
		if next, ok := deleteJSONPathBytes(out, path); ok {
			out = next
			modified = true
			remaining--
		}
	}

	if modified {
		return out
	}
	return body
}

// --- json byte helpers -------------------------------------------------------------

func marshalAnthropicSystemTextBlock(text string, includeCacheControl bool) ([]byte, error) {
	block := anthropicSystemTextBlockPayload{Type: "text", Text: text}
	if includeCacheControl {
		block.CacheControl = &anthropicCacheControlPayload{Type: "ephemeral"}
	}
	return json.Marshal(block)
}

func marshalAnthropicMetadata(userID string) ([]byte, error) {
	return json.Marshal(anthropicMetadataPayload{UserID: userID})
}

func buildJSONArrayRaw(items [][]byte) []byte {
	if len(items) == 0 {
		return []byte("[]")
	}
	total := 2
	for _, item := range items {
		total += len(item)
	}
	total += len(items) - 1

	buf := make([]byte, 0, total)
	buf = append(buf, '[')
	for i, item := range items {
		if i > 0 {
			buf = append(buf, ',')
		}
		buf = append(buf, item...)
	}
	buf = append(buf, ']')
	return buf
}

func setJSONValueBytes(body []byte, path string, value any) ([]byte, bool) {
	next, err := sjson.SetBytes(body, path, value)
	if err != nil {
		return body, false
	}
	return next, true
}

func setJSONRawBytes(body []byte, path string, raw []byte) ([]byte, bool) {
	next, err := sjson.SetRawBytes(body, path, raw)
	if err != nil {
		return body, false
	}
	return next, true
}

func deleteJSONPathBytes(body []byte, path string) ([]byte, bool) {
	next, err := sjson.DeleteBytes(body, path)
	if err != nil {
		return body, false
	}
	return next, true
}
