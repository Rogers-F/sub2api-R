package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// Conversation lifecycle status values.
const (
	ConversationStatusActive   = "active"
	ConversationStatusArchived = "archived"
)

// Message role values. Only user/assistant are accepted; system is rejected.
const (
	MessageRoleUser      = "user"
	MessageRoleAssistant = "assistant"
)

// Message status values.
const (
	MessageStatusComplete = "complete"
	MessageStatusError    = "error"
)

// MaxMessageContentBytes is the 512KB content cap enforced both in the service
// layer and by a DB CHECK constraint.
const MaxMessageContentBytes = 512 * 1024

// Errors returned by the conversation service. Mapped to HTTP status codes via
// the standard ApplicationError envelope.
var (
	ErrConversationNotFound    = infraerrors.NotFound("CONVERSATION_NOT_FOUND", "conversation not found")
	ErrConversationInvalid     = infraerrors.BadRequest("CONVERSATION_INVALID", "invalid conversation request")
	ErrMessageInvalid          = infraerrors.BadRequest("MESSAGE_INVALID", "invalid message request")
	ErrMessageConflict         = infraerrors.Conflict("MESSAGE_CONFLICT", "message with this client_message_id already exists with a different payload")
	ErrMessageDuplicateInBatch = infraerrors.BadRequest("MESSAGE_DUPLICATE_IN_BATCH", "duplicate client_message_id within the same batch")
)

// Conversation is the service-layer representation of a chat conversation.
type Conversation struct {
	ID                   int64     `json:"id"`
	UserID               int64     `json:"user_id"`
	ClientConversationID string    `json:"client_conversation_id"`
	Title                string    `json:"title"`
	Model                string    `json:"model"`
	Status               string    `json:"status"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	// LastMessageAt is the dedicated list-ordering key (see schema). Advanced
	// only on message append/replace, never on rename.
	LastMessageAt time.Time `json:"last_message_at"`
}

// Message is the service-layer representation of a conversation message.
//
// ReportedInputTokens / ReportedOutputTokens are client-reported, display-only
// values. They are never used for billing.
type Message struct {
	ID                   int64     `json:"id"`
	ConversationID       int64     `json:"conversation_id"`
	UserID               int64     `json:"user_id"`
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

// MessageInput is the service-layer input for appending a single message.
type MessageInput struct {
	Role                 string
	Content              string
	Model                string
	Status               string
	ReportedInputTokens  *int
	ReportedOutputTokens *int
	ClientMessageID      string
	GatewayRequestID     *string
}

// ConversationCursor is the composite cursor for listing conversations,
// ordered by (last_message_at DESC, id DESC).
type ConversationCursor struct {
	LastMessageAt time.Time
	ID            int64
}

// ConversationRepository persists conversation metadata.
//
// All methods that take an id MUST also filter by user_id to prevent IDOR.
type ConversationRepository interface {
	// Create inserts a new conversation. UNIQUE (user_id, client_conversation_id)
	// is enforced at the DB level.
	Create(ctx context.Context, c *Conversation) error
	// GetByClientID returns the conversation for (user_id, client_conversation_id),
	// or ErrConversationNotFound when missing.
	GetByClientID(ctx context.Context, userID int64, clientConversationID string) (*Conversation, error)
	// GetByID returns the conversation for (id, user_id), or ErrConversationNotFound.
	GetByID(ctx context.Context, userID, id int64) (*Conversation, error)
	// GetByIDForUpdate locks and returns the conversation for (id, user_id).
	// Must be called inside a transaction.
	GetByIDForUpdate(ctx context.Context, userID, id int64) (*Conversation, error)
	// List returns conversations for a user ordered by (updated_at DESC, id DESC),
	// starting after the optional cursor, up to limit items.
	List(ctx context.Context, userID int64, cursor *ConversationCursor, limit int) ([]Conversation, error)
	// UpdateTitle sets the title for (id, user_id). Returns ErrConversationNotFound
	// when no rows match. Does NOT advance last_message_at (rename must not
	// re-order the list).
	UpdateTitle(ctx context.Context, userID, id int64, title string) error
	// Touch sets last_message_at for (id, user_id) to the given time. Used after
	// appending/replacing messages; the value is the newest message's created_at.
	Touch(ctx context.Context, userID, id int64, at time.Time) error
	// Delete hard-deletes (id, user_id). Cascade removes messages. Returns
	// ErrConversationNotFound when no rows match.
	Delete(ctx context.Context, userID, id int64) error
}

// MessageRepository persists conversation messages.
//
// All methods that take an id MUST also filter by user_id to prevent IDOR.
type MessageRepository interface {
	// GetByClientID returns a message by (conversation_id, client_message_id)
	// scoped to user_id, or ErrConversationNotFound when missing.
	GetByClientID(ctx context.Context, userID, conversationID int64, clientMessageID string) (*Message, error)
	// GetByID returns a message by (id) scoped to (user_id, conversation_id), or
	// ErrConversationNotFound when missing. Used to resolve/validate a cutoff.
	GetByID(ctx context.Context, userID, conversationID, id int64) (*Message, error)
	// Create inserts a single message.
	Create(ctx context.Context, m *Message) error
	// List returns messages for (user_id, conversation_id) ordered by id ASC,
	// starting after the optional id cursor (id > afterID), up to limit items.
	// Forward pagination retained for backward compatibility.
	List(ctx context.Context, userID, conversationID int64, afterID int64, limit int) ([]Message, error)
	// ListBefore returns messages for (user_id, conversation_id) with id < beforeID
	// (or the newest messages when beforeID <= 0), ordered id DESC (newest first),
	// up to limit items. Backward pagination; the service reverses to id ASC.
	ListBefore(ctx context.Context, userID, conversationID int64, beforeID int64, limit int) ([]Message, error)
	// MaxMessageID returns the largest message id within (user_id, conversation_id),
	// and false when the conversation has no messages.
	MaxMessageID(ctx context.Context, userID, conversationID int64) (int64, bool, error)
	// DeleteFrom hard-deletes messages in (user_id, conversation_id) with id >= fromID,
	// returning the number of rows removed.
	DeleteFrom(ctx context.Context, userID, conversationID, fromID int64) (int, error)
}
