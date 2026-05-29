package handler

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

// errInvalidCursor is returned when a pagination cursor token is malformed.
var errInvalidCursor = errors.New("invalid cursor")

// ConversationHandler handles persisted multi-conversation chat operations for
// the authenticated user.
type ConversationHandler struct {
	conversationService *service.ConversationService
}

// NewConversationHandler creates a new conversation handler.
func NewConversationHandler(conversationService *service.ConversationService) *ConversationHandler {
	return &ConversationHandler{conversationService: conversationService}
}

// List handles GET /conversations.
func (h *ConversationHandler) List(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	limit := parseLimitQuery(c.Query("limit"))
	cursor, err := decodeConversationCursor(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "Invalid cursor")
		return
	}

	result, err := h.conversationService.ListConversations(c.Request.Context(), subject.UserID, cursor, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	items := make([]dto.Conversation, 0, len(result.Items))
	for i := range result.Items {
		items = append(items, *dto.ConversationFromService(&result.Items[i]))
	}
	resp := dto.ConversationListResponse{Items: items}
	if result.NextCursor != nil {
		resp.NextCursor = encodeConversationCursor(result.NextCursor)
	}
	response.Success(c, resp)
}

// Create handles POST /conversations. Idempotent on client_conversation_id.
func (h *ConversationHandler) Create(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	var req dto.CreateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	created, err := h.conversationService.CreateConversation(
		c.Request.Context(),
		subject.UserID,
		req.ClientConversationID,
		req.Title,
		req.Model,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ConversationFromService(created))
}

// GetByID handles GET /conversations/:id.
func (h *ConversationHandler) GetByID(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	id, ok := parseConversationID(c)
	if !ok {
		return
	}

	item, err := h.conversationService.GetConversation(c.Request.Context(), subject.UserID, id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ConversationFromService(item))
}

// Update handles PATCH /conversations/:id.
func (h *ConversationHandler) Update(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	id, ok := parseConversationID(c)
	if !ok {
		return
	}

	var req dto.UpdateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	updated, err := h.conversationService.UpdateTitle(c.Request.Context(), subject.UserID, id, req.Title)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.ConversationFromService(updated))
}

// Delete handles DELETE /conversations/:id.
func (h *ConversationHandler) Delete(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	id, ok := parseConversationID(c)
	if !ok {
		return
	}

	if err := h.conversationService.DeleteConversation(c.Request.Context(), subject.UserID, id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Conversation deleted successfully"})
}

// ListMessages handles GET /conversations/:id/messages.
func (h *ConversationHandler) ListMessages(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	id, ok := parseConversationID(c)
	if !ok {
		return
	}

	limit := parseLimitQuery(c.Query("limit"))

	// Detect presence by key (not value), so an empty value like "?before_id="
	// is treated as a malformed backward request rather than silently falling
	// back to the legacy forward path.
	beforeRaw, hasBefore := c.GetQuery("before_id")
	_, hasCursor := c.GetQuery("cursor")
	// The two pagination modes are mutually exclusive.
	if hasBefore && hasCursor {
		response.BadRequest(c, "cursor and before_id are mutually exclusive")
		return
	}

	var (
		result *service.MessageList
		err    error
		// backward mode emits next_cursor = smallest id of the page (older page).
		backward bool
	)
	if hasBefore {
		// Backward (newest-first) pagination. before_id == 0 means "latest page".
		beforeID, perr := strconv.ParseInt(strings.TrimSpace(beforeRaw), 10, 64)
		if perr != nil || beforeID < 0 {
			response.BadRequest(c, "Invalid before_id")
			return
		}
		backward = true
		result, err = h.conversationService.ListMessagesBefore(c.Request.Context(), subject.UserID, id, beforeID, limit)
	} else {
		// Legacy forward pagination (cursor = afterID), retained for compatibility.
		afterID, perr := parseAfterIDQuery(c.Query("cursor"))
		if perr != nil {
			response.BadRequest(c, "Invalid cursor")
			return
		}
		result, err = h.conversationService.ListMessages(c.Request.Context(), subject.UserID, id, afterID, limit)
	}
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.Message, 0, len(result.Items))
	for i := range result.Items {
		out = append(out, *dto.MessageFromService(&result.Items[i]))
	}
	resp := dto.MessageListResponse{Items: out}
	// Items are always id ASC. Emit next_cursor only when another page exists:
	// backward -> smallest id (oldest, for the next older page);
	// forward  -> largest id (newest, for the next forward page).
	if result.HasMore && len(result.Items) > 0 {
		if backward {
			resp.NextCursor = strconv.FormatInt(result.Items[0].ID, 10)
		} else {
			resp.NextCursor = strconv.FormatInt(result.Items[len(result.Items)-1].ID, 10)
		}
	}
	response.Success(c, resp)
}

// AppendMessages handles POST /conversations/:id/messages.
func (h *ConversationHandler) AppendMessages(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	id, ok := parseConversationID(c)
	if !ok {
		return
	}

	var req dto.AppendMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	inputs := make([]service.MessageInput, 0, len(req.Messages))
	for i := range req.Messages {
		inputs = append(inputs, dto.MessageInputToService(&req.Messages[i]))
	}

	created, err := h.conversationService.AppendMessages(c.Request.Context(), subject.UserID, id, inputs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.Message, 0, len(created))
	for i := range created {
		out = append(out, *dto.MessageFromService(&created[i]))
	}
	response.Success(c, dto.MessageListResponse{Items: out})
}

// Replace handles POST /conversations/:id/messages/replace.
//
// Atomically replaces the conversation's trailing assistant message (the cutoff,
// identified by exactly one of from_id / from_client_message_id) with a new
// assistant reply. Used by "regenerate".
func (h *ConversationHandler) Replace(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not found in context")
		return
	}

	id, ok := parseConversationID(c)
	if !ok {
		return
	}

	var req dto.ReplaceMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	newMsg := dto.MessageInputToService(&req.Message)
	replaced, err := h.conversationService.ReplaceMessageFrom(
		c.Request.Context(),
		subject.UserID,
		id,
		req.FromID,
		req.FromClientMessageID,
		newMsg,
	)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.MessageFromService(replaced))
}

// parseConversationID parses and validates the :id path parameter, writing a
// 400 response and returning false on failure.
func parseConversationID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid conversation ID")
		return 0, false
	}
	return id, true
}

func parseLimitQuery(v string) int {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// parseAfterIDQuery parses the message id cursor. An empty token yields 0 (first
// page). A malformed or negative token yields an error so the handler can return
// 400 instead of silently resetting to the first page.
func parseAfterIDQuery(v string) (int64, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, errInvalidCursor
	}
	return n, nil
}

// conversationCursorVersion prefixes the cursor token. The v2 cursor keys on
// last_message_at (v1 keyed on updated_at); the prefix forces rejection of any
// stale v1 token so it is never silently misinterpreted across a deploy.
const conversationCursorVersion = "v2"

// encodeConversationCursor encodes (v2:last_message_at_unix_nanos:id) as base64.
func encodeConversationCursor(cur *service.ConversationCursor) string {
	raw := conversationCursorVersion + ":" +
		strconv.FormatInt(cur.LastMessageAt.UTC().UnixNano(), 10) + ":" +
		strconv.FormatInt(cur.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeConversationCursor decodes a cursor token. An empty token yields a nil
// cursor (first page). A malformed or non-v2 token yields an error.
func decodeConversationCursor(token string) (*service.ConversationCursor, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(decoded), ":", 3)
	if len(parts) != 3 || parts[0] != conversationCursorVersion {
		return nil, errInvalidCursor
	}
	nanos, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, err
	}
	return &service.ConversationCursor{
		LastMessageAt: time.Unix(0, nanos).UTC(),
		ID:            id,
	}, nil
}
