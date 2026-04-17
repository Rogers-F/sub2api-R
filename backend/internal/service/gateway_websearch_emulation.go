package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/websearch"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tidwall/gjson"
)

const (
	toolTypeWebSearchPrefix = "web_search"
	toolTypeGoogleSearch    = "google_search"
	toolNameWebSearch       = "web_search"
	toolNameGoogleSearch    = "google_search"
	toolNameWebSearch2025   = "web_search_20250305"

	webSearchDefaultMaxResults = 5
	defaultWebSearchModel      = "claude-sonnet-4-6"
	webSearchMsgIDPrefix       = "msg_ws_"
	webSearchToolUseIDPrefix   = "srvtoolu_ws_"
	tokenEstimateDivisor       = 4

	featureKeyWebSearchEmulation = "web_search_emulation"
)

var webSearchManagerPtr atomic.Pointer[websearch.Manager]

func SetWebSearchManager(manager *websearch.Manager) {
	webSearchManagerPtr.Store(manager)
}

func getWebSearchManager() *websearch.Manager {
	return webSearchManagerPtr.Load()
}

func (s *GatewayService) shouldEmulateWebSearch(ctx context.Context, account *Account, groupID *int64, body []byte) bool {
	if s == nil || s.settingService == nil || account == nil {
		return false
	}
	if getWebSearchManager() == nil {
		return false
	}
	if !isOnlyWebSearchToolInBody(body) {
		return false
	}
	if !s.settingService.IsWebSearchEmulationEnabled(ctx) {
		return false
	}

	switch account.GetWebSearchEmulationMode() {
	case WebSearchModeEnabled:
		return true
	case WebSearchModeDisabled:
		return false
	default:
		if groupID == nil || s.channelService == nil {
			return false
		}
		channel, err := s.channelService.GetChannelForGroup(ctx, *groupID)
		if err != nil || channel == nil {
			return false
		}
		return channel.IsWebSearchEmulationEnabled(account.Platform)
	}
}

func isOnlyWebSearchToolInBody(body []byte) bool {
	tools := gjson.GetBytes(body, "tools")
	if !tools.IsArray() {
		return false
	}
	items := tools.Array()
	if len(items) != 1 {
		return false
	}
	return isWebSearchToolJSON(items[0])
}

func isWebSearchToolJSON(tool gjson.Result) bool {
	toolType := tool.Get("type").String()
	if strings.HasPrefix(toolType, toolTypeWebSearchPrefix) || toolType == toolTypeGoogleSearch {
		return true
	}
	switch tool.Get("name").String() {
	case toolNameWebSearch, toolNameGoogleSearch, toolNameWebSearch2025:
		return true
	default:
		return false
	}
}

func extractSearchQueryFromBody(body []byte) string {
	messages := gjson.GetBytes(body, "messages")
	if !messages.IsArray() {
		return ""
	}
	items := messages.Array()
	if len(items) == 0 {
		return ""
	}
	last := items[len(items)-1]
	if last.Get("role").String() != "user" {
		return ""
	}
	return extractWebSearchTextFromContent(last.Get("content"))
}

func extractWebSearchTextFromContent(content gjson.Result) string {
	if content.Type == gjson.String {
		return content.String()
	}
	if !content.IsArray() {
		return ""
	}
	for _, block := range content.Array() {
		if block.Get("type").String() != "text" {
			continue
		}
		if text := block.Get("text").String(); text != "" {
			return text
		}
	}
	return ""
}

func (s *GatewayService) handleWebSearchEmulation(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	parsed *ParsedRequest,
) (*ForwardResult, error) {
	startTime := time.Now()
	if parsed.OnUpstreamAccepted != nil {
		parsed.OnUpstreamAccepted()
	}

	query := extractSearchQueryFromBody(parsed.Body)
	if query == "" {
		return nil, fmt.Errorf("web search emulation: no query found in messages")
	}

	slog.Info("web search emulation: executing search",
		"account_id", account.ID,
		"account_name", account.Name,
		"query", query,
	)

	resp, providerName, err := doWebSearch(ctx, account, query)
	if err != nil {
		if errors.Is(err, websearch.ErrProxyUnavailable) {
			return nil, &UpstreamFailoverError{
				StatusCode:   http.StatusBadGateway,
				ResponseBody: []byte(err.Error()),
			}
		}
		return nil, err
	}

	slog.Info("web search emulation: search completed",
		"provider", providerName,
		"results_count", len(resp.Results),
	)

	model := parsed.Model
	if model == "" {
		model = defaultWebSearchModel
	}
	if parsed.Stream {
		return writeWebSearchStreamResponse(c, query, resp, model, startTime)
	}
	return writeWebSearchNonStreamResponse(c, query, resp, model, startTime)
}

func doWebSearch(ctx context.Context, account *Account, query string) (*websearch.SearchResponse, string, error) {
	manager := getWebSearchManager()
	if manager == nil {
		return nil, "", fmt.Errorf("web search emulation: manager not initialized")
	}
	proxyURL := resolveAccountProxyURL(account)
	resp, providerName, err := manager.SearchWithBestProvider(ctx, websearch.SearchRequest{
		Query:      query,
		MaxResults: webSearchDefaultMaxResults,
		ProxyURL:   proxyURL,
	})
	if err != nil {
		slog.Error("web search emulation: search failed", "error", err)
		return nil, "", fmt.Errorf("web search emulation: %w", err)
	}
	return resp, providerName, nil
}

func resolveAccountProxyURL(account *Account) string {
	if account != nil && account.ProxyID != nil && account.Proxy != nil {
		return account.Proxy.URL()
	}
	return ""
}

func writeWebSearchStreamResponse(
	c *gin.Context,
	query string,
	resp *websearch.SearchResponse,
	model string,
	startTime time.Time,
) (*ForwardResult, error) {
	msgID := webSearchMsgIDPrefix + uuid.New().String()
	toolUseID := webSearchToolUseIDPrefix + uuid.New().String()[:16]
	textSummary := buildTextSummary(query, resp.Results)

	setWebSearchSSEHeaders(c)
	writer := c.Writer
	for _, fn := range []func() error{
		func() error { return writeSSEMessageStart(writer, msgID, model) },
		func() error { return writeSSEServerToolUse(writer, toolUseID, query, 0) },
		func() error { return writeSSEToolResult(writer, toolUseID, resp.Results, 1) },
		func() error { return writeSSETextBlock(writer, textSummary, 2) },
		func() error { return writeSSEMessageEnd(writer, len(textSummary)/tokenEstimateDivisor) },
	} {
		if err := fn(); err != nil {
			slog.Warn("web search emulation: SSE write failed, stopping", "error", err)
			break
		}
	}
	writer.Flush()

	return &ForwardResult{Model: model, Duration: time.Since(startTime), Usage: ClaudeUsage{}}, nil
}

func setWebSearchSSEHeaders(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)
}

func writeSSEMessageStart(w http.ResponseWriter, msgID, model string) error {
	payload := map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            msgID,
			"type":          "message",
			"role":          "assistant",
			"model":         model,
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage":         map[string]int{"input_tokens": 0, "output_tokens": 0},
		},
	}
	return flushSSEJSON(w, "message_start", payload)
}

func writeSSEServerToolUse(w http.ResponseWriter, toolUseID, query string, index int) error {
	start := map[string]any{
		"type":  "content_block_start",
		"index": index,
		"content_block": map[string]any{
			"type":  "server_tool_use",
			"id":    toolUseID,
			"name":  toolNameWebSearch,
			"input": map[string]string{"query": query},
		},
	}
	if err := flushSSEJSON(w, "content_block_start", start); err != nil {
		return err
	}
	return flushSSEJSON(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": index})
}

func writeSSEToolResult(w http.ResponseWriter, toolUseID string, results []websearch.SearchResult, index int) error {
	start := map[string]any{
		"type":  "content_block_start",
		"index": index,
		"content_block": map[string]any{
			"type":        "web_search_tool_result",
			"tool_use_id": toolUseID,
			"content":     buildSearchResultBlocks(results),
		},
	}
	if err := flushSSEJSON(w, "content_block_start", start); err != nil {
		return err
	}
	return flushSSEJSON(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": index})
}

func writeSSETextBlock(w http.ResponseWriter, text string, index int) error {
	if err := flushSSEJSON(w, "content_block_start", map[string]any{
		"type":  "content_block_start",
		"index": index,
		"content_block": map[string]any{
			"type": "text",
			"text": "",
		},
	}); err != nil {
		return err
	}
	if err := flushSSEJSON(w, "content_block_delta", map[string]any{
		"type":  "content_block_delta",
		"index": index,
		"delta": map[string]string{
			"type": "text_delta",
			"text": text,
		},
	}); err != nil {
		return err
	}
	return flushSSEJSON(w, "content_block_stop", map[string]any{"type": "content_block_stop", "index": index})
}

func writeSSEMessageEnd(w http.ResponseWriter, outputTokens int) error {
	if err := flushSSEJSON(w, "message_delta", map[string]any{
		"type":  "message_delta",
		"delta": map[string]any{"stop_reason": "end_turn", "stop_sequence": nil},
		"usage": map[string]int{"output_tokens": outputTokens},
	}); err != nil {
		return err
	}
	return flushSSEJSON(w, "message_stop", map[string]string{"type": "message_stop"})
}

func flushSSEJSON(w http.ResponseWriter, event string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, payload); err != nil {
		return fmt.Errorf("write: %w", err)
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
	return nil
}

func writeWebSearchNonStreamResponse(
	c *gin.Context,
	query string,
	resp *websearch.SearchResponse,
	model string,
	startTime time.Time,
) (*ForwardResult, error) {
	msgID := webSearchMsgIDPrefix + uuid.New().String()
	toolUseID := webSearchToolUseIDPrefix + uuid.New().String()[:16]
	textSummary := buildTextSummary(query, resp.Results)

	payload := map[string]any{
		"id":    msgID,
		"type":  "message",
		"role":  "assistant",
		"model": model,
		"content": []any{
			map[string]any{
				"type":  "server_tool_use",
				"id":    toolUseID,
				"name":  toolNameWebSearch,
				"input": map[string]string{"query": query},
			},
			map[string]any{
				"type":        "web_search_tool_result",
				"tool_use_id": toolUseID,
				"content":     buildSearchResultBlocks(resp.Results),
			},
			map[string]any{"type": "text", "text": textSummary},
		},
		"stop_reason":   "end_turn",
		"stop_sequence": nil,
		"usage":         map[string]int{"input_tokens": 0, "output_tokens": len(textSummary) / tokenEstimateDivisor},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("web search emulation: marshal response: %w", err)
	}
	c.Data(http.StatusOK, "application/json", body)
	return &ForwardResult{Model: model, Duration: time.Since(startTime), Usage: ClaudeUsage{}}, nil
}

func buildSearchResultBlocks(results []websearch.SearchResult) []map[string]string {
	blocks := make([]map[string]string, 0, len(results))
	for _, result := range results {
		block := map[string]string{
			"type":  "web_search_result",
			"url":   result.URL,
			"title": result.Title,
		}
		if result.Snippet != "" {
			block["page_content"] = result.Snippet
		}
		if result.PageAge != "" {
			block["page_age"] = result.PageAge
		}
		blocks = append(blocks, block)
	}
	return blocks
}

func buildTextSummary(query string, results []websearch.SearchResult) string {
	if len(results) == 0 {
		return "No search results found for: " + query
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Here are the search results for \"%s\":\n\n", query)
	for i, result := range results {
		fmt.Fprintf(&sb, "%d. **%s**\n   %s\n   %s\n\n", i+1, result.Title, result.URL, result.Snippet)
	}
	return sb.String()
}
