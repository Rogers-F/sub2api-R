package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Conversation is the API representation of a chat conversation.
type Conversation struct {
	ID                   int64     `json:"id"`
	ClientConversationID string    `json:"client_conversation_id"`
	Title                string    `json:"title"`
	Model                string    `json:"model"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	// LastMessageAt is the list-ordering key (advanced on append/replace only).
	LastMessageAt time.Time `json:"last_message_at"`
}

// Message is the API representation of a conversation message.
//
// ReportedInputTokens / ReportedOutputTokens are client-reported, display-only
// values. They are never used for billing.
type Message struct {
	ID                   int64     `json:"id"`
	ConversationID       int64     `json:"conversation_id"`
	Role                 string    `json:"role"`
	Content              string    `json:"content"`
	Model                string    `json:"model"`
	Status               string    `json:"status"`
	ReportedInputTokens  *int      `json:"reported_input_tokens,omitempty"`
	ReportedOutputTokens *int      `json:"reported_output_tokens,omitempty"`
	ClientMessageID      string    `json:"client_message_id"`
	GatewayRequestID     *string   `json:"gateway_request_id,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

// CreateConversationRequest is the body for POST /conversations.
type CreateConversationRequest struct {
	ClientConversationID string `json:"client_conversation_id" binding:"required"`
	Title                string `json:"title"`
	Model                string `json:"model"`
}

// UpdateConversationRequest is the body for PATCH /conversations/:id.
type UpdateConversationRequest struct {
	Title string `json:"title"`
}

// MessageInputRequest is a single message in an append request.
//
// ReportedInputTokens / ReportedOutputTokens are client-reported, display-only
// values. They are never used for billing.
type MessageInputRequest struct {
	Role                 string  `json:"role" binding:"required"`
	Content              string  `json:"content"`
	Model                string  `json:"model"`
	Status               string  `json:"status"`
	ReportedInputTokens  *int    `json:"reported_input_tokens"`
	ReportedOutputTokens *int    `json:"reported_output_tokens"`
	ClientMessageID      string  `json:"client_message_id" binding:"required"`
	GatewayRequestID     *string `json:"gateway_request_id"`
}

// AppendMessagesRequest is the body for POST /conversations/:id/messages.
type AppendMessagesRequest struct {
	Messages []MessageInputRequest `json:"messages" binding:"required,min=1"`
}

// ReplaceMessageRequest is the body for POST /conversations/:id/messages/replace.
//
// Exactly one of FromID / FromClientMessageID identifies the cutoff (the
// current trailing assistant message being regenerated). Message is the new
// assistant reply to insert in its place.
type ReplaceMessageRequest struct {
	FromID              int64               `json:"from_id"`
	FromClientMessageID string              `json:"from_client_message_id"`
	Message             MessageInputRequest `json:"message" binding:"required"`
}

// ConversationListResponse is the paginated list of conversations with a cursor.
type ConversationListResponse struct {
	Items      []Conversation `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// MessageListResponse is the paginated list of messages with an id cursor.
type MessageListResponse struct {
	Items      []Message `json:"items"`
	NextCursor string    `json:"next_cursor,omitempty"`
}

// ConversationFromService converts a service Conversation into its DTO.
func ConversationFromService(c *service.Conversation) *Conversation {
	if c == nil {
		return nil
	}
	return &Conversation{
		ID:                   c.ID,
		ClientConversationID: c.ClientConversationID,
		Title:                c.Title,
		Model:                c.Model,
		Status:               c.Status,
		CreatedAt:            c.CreatedAt,
		UpdatedAt:            c.UpdatedAt,
		LastMessageAt:        c.LastMessageAt,
	}
}

// MessageFromService converts a service Message into its DTO.
func MessageFromService(m *service.Message) *Message {
	if m == nil {
		return nil
	}
	return &Message{
		ID:                   m.ID,
		ConversationID:       m.ConversationID,
		Role:                 m.Role,
		Content:              m.Content,
		Model:                m.Model,
		Status:               m.Status,
		ReportedInputTokens:  m.ReportedInputTokens,
		ReportedOutputTokens: m.ReportedOutputTokens,
		ClientMessageID:      m.ClientMessageID,
		GatewayRequestID:     m.GatewayRequestID,
		CreatedAt:            m.CreatedAt,
	}
}

// MessageInputToService converts an API message input into the service input.
func MessageInputToService(in *MessageInputRequest) service.MessageInput {
	return service.MessageInput{
		Role:                 in.Role,
		Content:              in.Content,
		Model:                in.Model,
		Status:               in.Status,
		ReportedInputTokens:  in.ReportedInputTokens,
		ReportedOutputTokens: in.ReportedOutputTokens,
		ClientMessageID:      in.ClientMessageID,
		GatewayRequestID:     in.GatewayRequestID,
	}
}
