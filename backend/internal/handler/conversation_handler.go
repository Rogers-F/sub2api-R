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
	afterID, err := parseAfterIDQuery(c.Query("cursor"))
	if err != nil {
		response.BadRequest(c, "Invalid cursor")
		return
	}

	result, err := h.conversationService.ListMessages(c.Request.Context(), subject.UserID, id, afterID, limit)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	out := make([]dto.Message, 0, len(result.Items))
	for i := range result.Items {
		out = append(out, *dto.MessageFromService(&result.Items[i]))
	}
	resp := dto.MessageListResponse{Items: out}
	// id ASC cursor: emit a cursor only when another page exists.
	if result.HasMore && len(result.Items) > 0 {
		resp.NextCursor = strconv.FormatInt(result.Items[len(result.Items)-1].ID, 10)
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

// encodeConversationCursor encodes (updated_at_unix_nanos:id) as a base64 token.
func encodeConversationCursor(cur *service.ConversationCursor) string {
	raw := strconv.FormatInt(cur.UpdatedAt.UTC().UnixNano(), 10) + ":" + strconv.FormatInt(cur.ID, 10)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// decodeConversationCursor decodes a cursor token. An empty token yields a nil
// cursor (first page). A malformed token yields an error.
func decodeConversationCursor(token string) (*service.ConversationCursor, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return nil, errInvalidCursor
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, err
	}
	return &service.ConversationCursor{
		UpdatedAt: time.Unix(0, nanos).UTC(),
		ID:        id,
	}, nil
}
