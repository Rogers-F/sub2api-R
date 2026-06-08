package controller

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// Persisted multi-conversation chat endpoints (ported from sub2api). Mounted
// under /api/conversations with UserAuth; userId comes from the session
// (c.GetInt("id")), never the body, to prevent IDOR.
//
// Timestamps are stored as int64 unix seconds but emitted to the (star Vue)
// frontend as RFC3339 strings, matching the existing types/chat.ts contract.

// --- API DTOs (response shapes the frontend consumes) ---

type conversationDTO struct {
	Id                   int    `json:"id"`
	ClientConversationId string `json:"client_conversation_id"`
	Title                string `json:"title"`
	Model                string `json:"model"`
	Status               string `json:"status"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
	LastMessageAt        string `json:"last_message_at"`
}

type messageDTO struct {
	Id                   int     `json:"id"`
	ConversationId       int     `json:"conversation_id"`
	Role                 string  `json:"role"`
	Content              string  `json:"content"`
	Model                string  `json:"model"`
	Status               string  `json:"status"`
	ReportedInputTokens  *int    `json:"reported_input_tokens,omitempty"`
	ReportedOutputTokens *int    `json:"reported_output_tokens,omitempty"`
	ClientMessageId      string  `json:"client_message_id"`
	GatewayRequestId     *string `json:"gateway_request_id,omitempty"`
	CreatedAt            string  `json:"created_at"`
}

type conversationListResponse struct {
	Items      []conversationDTO `json:"items"`
	NextCursor *string           `json:"next_cursor"`
}

type messageListResponse struct {
	Items      []messageDTO `json:"items"`
	NextCursor *string      `json:"next_cursor"`
}

// --- request bodies ---

type createConversationRequest struct {
	ClientConversationId string `json:"client_conversation_id" binding:"required"`
	Title                string `json:"title"`
	Model                string `json:"model"`
}

type updateConversationRequest struct {
	Title string `json:"title"`
}

type messageInputRequest struct {
	Role                 string  `json:"role" binding:"required"`
	Content              string  `json:"content"`
	Model                string  `json:"model"`
	Status               string  `json:"status"`
	ReportedInputTokens  *int    `json:"reported_input_tokens"`
	ReportedOutputTokens *int    `json:"reported_output_tokens"`
	ClientMessageId      string  `json:"client_message_id" binding:"required"`
	GatewayRequestId     *string `json:"gateway_request_id"`
}

type appendMessagesRequest struct {
	Messages []messageInputRequest `json:"messages" binding:"required,min=1"`
}

type replaceMessageRequest struct {
	FromID              int                 `json:"from_id"`
	FromClientMessageID string              `json:"from_client_message_id"`
	Message             messageInputRequest `json:"message" binding:"required"`
}

func unixToISO(ts int64) string {
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func toConversationDTO(c *model.Conversation) conversationDTO {
	return conversationDTO{
		Id:                   c.Id,
		ClientConversationId: c.ClientConversationId,
		Title:                c.Title,
		Model:                c.Model,
		Status:               c.Status,
		CreatedAt:            unixToISO(c.CreatedAt),
		UpdatedAt:            unixToISO(c.UpdatedAt),
		LastMessageAt:        unixToISO(c.LastMessageAt),
	}
}

func toMessageDTO(m *model.ConversationMessage) messageDTO {
	return messageDTO{
		Id:                   m.Id,
		ConversationId:       m.ConversationId,
		Role:                 m.Role,
		Content:              m.Content,
		Model:                m.Model,
		Status:               m.Status,
		ReportedInputTokens:  m.ReportedInputTokens,
		ReportedOutputTokens: m.ReportedOutputTokens,
		ClientMessageId:      m.ClientMessageId,
		GatewayRequestId:     m.GatewayRequestId,
		CreatedAt:            unixToISO(m.CreatedAt),
	}
}

func messageInputToModel(in *messageInputRequest) model.MessageInput {
	return model.MessageInput{
		Role:                 in.Role,
		Content:              in.Content,
		Model:                in.Model,
		Status:               in.Status,
		ReportedInputTokens:  in.ReportedInputTokens,
		ReportedOutputTokens: in.ReportedOutputTokens,
		ClientMessageId:      in.ClientMessageId,
		GatewayRequestId:     in.GatewayRequestId,
	}
}

// conversationError maps model errors to the new-api {success:false} envelope
// (HTTP 200 per new-api convention; the frontend keys off the success field).
func conversationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, model.ErrConversationNotFound):
		common.ApiErrorMsg(c, "conversation not found")
	case errors.Is(err, model.ErrMessageConflict):
		common.ApiErrorMsg(c, "message conflict")
	case errors.Is(err, model.ErrConversationInvalid),
		errors.Is(err, model.ErrMessageInvalid),
		errors.Is(err, model.ErrMessageDuplicateInBatch):
		common.ApiErrorMsg(c, err.Error())
	default:
		common.ApiError(c, err)
	}
}

func parseConversationPathID(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id <= 0 {
		common.ApiErrorMsg(c, "invalid conversation id")
		return 0, false
	}
	return id, true
}

func parseLimit(v string) int {
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

// GetAllConversations handles GET /api/conversations.
func GetAllConversations(c *gin.Context) {
	userID := c.GetInt("id")

	cursor, err := model.DecodeConvCursor(c.Query("cursor"))
	if err != nil {
		common.ApiErrorMsg(c, "invalid cursor")
		return
	}
	items, next, err := model.ListConversations(userID, cursor, parseLimit(c.Query("limit")))
	if err != nil {
		conversationError(c, err)
		return
	}

	resp := conversationListResponse{Items: make([]conversationDTO, 0, len(items))}
	for i := range items {
		resp.Items = append(resp.Items, toConversationDTO(&items[i]))
	}
	if next != nil {
		s := model.EncodeConvCursor(next)
		resp.NextCursor = &s
	}
	common.ApiSuccess(c, resp)
}

// CreateConversation handles POST /api/conversations. Idempotent on client_conversation_id.
func CreateConversation(c *gin.Context) {
	userID := c.GetInt("id")
	var req createConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	created, err := model.CreateConversation(userID, req.ClientConversationId, req.Title, req.Model)
	if err != nil {
		conversationError(c, err)
		return
	}
	common.ApiSuccess(c, toConversationDTO(created))
}

// GetConversation handles GET /api/conversations/:id.
func GetConversation(c *gin.Context) {
	userID := c.GetInt("id")
	id, ok := parseConversationPathID(c)
	if !ok {
		return
	}
	conv, err := model.GetConversation(userID, id)
	if err != nil {
		conversationError(c, err)
		return
	}
	common.ApiSuccess(c, toConversationDTO(conv))
}

// UpdateConversation handles PATCH /api/conversations/:id.
func UpdateConversation(c *gin.Context) {
	userID := c.GetInt("id")
	id, ok := parseConversationPathID(c)
	if !ok {
		return
	}
	var req updateConversationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	updated, err := model.UpdateConversationTitle(userID, id, req.Title)
	if err != nil {
		conversationError(c, err)
		return
	}
	common.ApiSuccess(c, toConversationDTO(updated))
}

// DeleteConversation handles DELETE /api/conversations/:id.
func DeleteConversation(c *gin.Context) {
	userID := c.GetInt("id")
	id, ok := parseConversationPathID(c)
	if !ok {
		return
	}
	if err := model.DeleteConversation(userID, id); err != nil {
		conversationError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{})
}

// GetConversationMessages handles GET /api/conversations/:id/messages.
// Backward (before_id) and forward (cursor=afterID) pagination are mutually
// exclusive; messages are always returned id ASC.
func GetConversationMessages(c *gin.Context) {
	userID := c.GetInt("id")
	id, ok := parseConversationPathID(c)
	if !ok {
		return
	}
	limit := parseLimit(c.Query("limit"))

	beforeRaw, hasBefore := c.GetQuery("before_id")
	_, hasCursor := c.GetQuery("cursor")
	if hasBefore && hasCursor {
		common.ApiErrorMsg(c, "cursor and before_id are mutually exclusive")
		return
	}

	var (
		items    []model.ConversationMessage
		hasMore  bool
		err      error
		backward bool
	)
	if hasBefore {
		beforeID, perr := strconv.Atoi(strings.TrimSpace(beforeRaw))
		if perr != nil || beforeID < 0 {
			common.ApiErrorMsg(c, "invalid before_id")
			return
		}
		backward = true
		items, hasMore, err = model.ListMessagesBefore(userID, id, beforeID, limit)
	} else {
		afterID := 0
		if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
			afterID, err = strconv.Atoi(raw)
			if err != nil || afterID < 0 {
				common.ApiErrorMsg(c, "invalid cursor")
				return
			}
		}
		items, hasMore, err = model.ListMessages(userID, id, afterID, limit)
	}
	if err != nil {
		conversationError(c, err)
		return
	}

	resp := messageListResponse{Items: make([]messageDTO, 0, len(items))}
	for i := range items {
		resp.Items = append(resp.Items, toMessageDTO(&items[i]))
	}
	// Items are id ASC. Emit next_cursor only when another page exists:
	// backward -> smallest id (oldest, next older page); forward -> largest id.
	if hasMore && len(items) > 0 {
		var s string
		if backward {
			s = strconv.Itoa(items[0].Id)
		} else {
			s = strconv.Itoa(items[len(items)-1].Id)
		}
		resp.NextCursor = &s
	}
	common.ApiSuccess(c, resp)
}

// AppendConversationMessages handles POST /api/conversations/:id/messages.
func AppendConversationMessages(c *gin.Context) {
	userID := c.GetInt("id")
	id, ok := parseConversationPathID(c)
	if !ok {
		return
	}
	var req appendMessagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	inputs := make([]model.MessageInput, 0, len(req.Messages))
	for i := range req.Messages {
		inputs = append(inputs, messageInputToModel(&req.Messages[i]))
	}
	created, err := model.AppendMessages(userID, id, inputs)
	if err != nil {
		conversationError(c, err)
		return
	}
	out := messageListResponse{Items: make([]messageDTO, 0, len(created))}
	for i := range created {
		out.Items = append(out.Items, toMessageDTO(&created[i]))
	}
	common.ApiSuccess(c, out)
}

// ReplaceConversationMessage handles POST /api/conversations/:id/messages/replace.
func ReplaceConversationMessage(c *gin.Context) {
	userID := c.GetInt("id")
	id, ok := parseConversationPathID(c)
	if !ok {
		return
	}
	var req replaceMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	replaced, err := model.ReplaceMessageFrom(userID, id, req.FromID, req.FromClientMessageID, messageInputToModel(&req.Message))
	if err != nil {
		conversationError(c, err)
		return
	}
	common.ApiSuccess(c, toMessageDTO(replaced))
}
