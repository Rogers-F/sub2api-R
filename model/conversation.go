package model

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Persisted multi-conversation chat. Ported from the legacy sub2api
// ConversationService (ent) to new-api (GORM). This layer does NOT touch billing
// and does NOT call the relay. Token fields on messages are client-reported and
// display-only. All id-scoped methods filter by user_id to prevent IDOR.
//
// new-api adaptations vs the source: ids are int (new-api User.Id is int);
// timestamps are int64 unix seconds (common.GetTimestamp); row locks use
// clause.Locking{Strength:"UPDATE"} (GORM v2 FOR UPDATE) on Postgres only.

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

const (
	maxConversationClientIDLen = 64
	maxConversationTitleLen    = 255
	maxConversationModelLen    = 100
	maxMessageClientIDLen      = 64
	maxMessageGatewayReqIDLen  = 128

	defaultConversationListLimit = 20
	maxConversationListLimit     = 100
	maxMessageBatchSize          = 50

	conversationCursorVersion = "v2"
)

// Errors returned by the conversation model layer. The controller maps these to
// the new-api {success,message} envelope / HTTP codes.
var (
	ErrConversationNotFound    = errors.New("conversation not found")
	ErrConversationInvalid     = errors.New("invalid conversation request")
	ErrMessageInvalid          = errors.New("invalid message request")
	ErrMessageConflict         = errors.New("message conflict")
	ErrMessageDuplicateInBatch = errors.New("duplicate client_message_id within the same batch")
	errInvalidCursor           = errors.New("invalid cursor")
)

// Conversation is a persisted chat conversation.
type Conversation struct {
	Id                   int    `json:"id" gorm:"primaryKey;autoIncrement;index:idx_conversations_user_lastmsg,priority:3,sort:desc"`
	UserId               int    `json:"user_id" gorm:"not null;uniqueIndex:uq_conversations_user_client,priority:1;index:idx_conversations_user_lastmsg,priority:1"`
	ClientConversationId string `json:"client_conversation_id" gorm:"type:varchar(64);not null;uniqueIndex:uq_conversations_user_client,priority:2"`
	Title                string `json:"title" gorm:"type:varchar(255);not null;default:''"`
	Model                string `json:"model" gorm:"type:varchar(100);not null;default:''"`
	Status               string `json:"status" gorm:"type:varchar(20);not null;default:'active'"`
	CreatedAt            int64  `json:"created_at" gorm:"bigint;not null"`
	UpdatedAt            int64  `json:"updated_at" gorm:"bigint;not null"`
	// LastMessageAt is the dedicated list-ordering key. Advanced only on message
	// append/replace, never on rename, so last_message_at == MAX(message.created_at).
	LastMessageAt int64 `json:"last_message_at" gorm:"bigint;not null;index:idx_conversations_user_lastmsg,priority:2,sort:desc"`
}

func (Conversation) TableName() string { return "conversations" }

// ConvCursor is the composite keyset cursor for listing conversations, ordered
// by (last_message_at DESC, id DESC). Exported so the controller can base64
// encode/decode it.
type ConvCursor struct {
	LastMessageAt int64
	ID            int
}

// normalizeConversationLimit clamps a requested limit into [1, max], defaulting
// when <= 0.
func normalizeConversationLimit(limit int) int {
	if limit <= 0 {
		return defaultConversationListLimit
	}
	if limit > maxConversationListLimit {
		return maxConversationListLimit
	}
	return limit
}

// usePGRowLock reports whether SELECT ... FOR UPDATE should be emitted. Postgres
// (production) supports it; SQLite (unit tests) does not.
func usePGRowLock() bool {
	return common.UsingPostgreSQL
}

// getConversationByClientID returns the conversation for (user_id,
// client_conversation_id), or ErrConversationNotFound when missing.
func getConversationByClientID(userID int, clientConversationID string) (*Conversation, error) {
	var c Conversation
	err := DB.Where("user_id = ? AND client_conversation_id = ?", userID, clientConversationID).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	return &c, nil
}

// CreateConversation is idempotent on (user_id, client_conversation_id): if a
// conversation already exists for the pair, the existing one is returned.
func CreateConversation(userID int, clientConversationID, title, model string) (*Conversation, error) {
	if userID <= 0 {
		return nil, ErrConversationInvalid
	}
	clientConversationID = strings.TrimSpace(clientConversationID)
	if clientConversationID == "" || len(clientConversationID) > maxConversationClientIDLen {
		return nil, ErrConversationInvalid
	}
	title = strings.TrimSpace(title)
	if len(title) > maxConversationTitleLen {
		return nil, ErrConversationInvalid
	}
	if len(model) > maxConversationModelLen {
		return nil, ErrConversationInvalid
	}

	// Fast path: return the existing conversation if present.
	if existing, err := getConversationByClientID(userID, clientConversationID); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrConversationNotFound) {
		return nil, err
	}

	// Pin created_at/updated_at/last_message_at to one instant so the
	// "no messages -> last_message_at == created_at" invariant holds exactly.
	now := common.GetTimestamp()
	c := &Conversation{
		UserId:               userID,
		ClientConversationId: clientConversationID,
		Title:                title,
		Model:                model,
		Status:               ConversationStatusActive,
		CreatedAt:            now,
		UpdatedAt:            now,
		LastMessageAt:        now,
	}
	if err := DB.Create(c).Error; err != nil {
		// Concurrent insert may trip the unique (user_id, client_conversation_id)
		// index between the lookup above and here. Resolve idempotently by
		// re-reading rather than relying on translated duplicate-key errors
		// (new-api does not enable gorm TranslateError).
		if existing, getErr := getConversationByClientID(userID, clientConversationID); getErr == nil {
			return existing, nil
		}
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	return c, nil
}

// GetConversation returns conversation metadata for (id, user_id), or
// ErrConversationNotFound if not owned/missing.
func GetConversation(userID, id int) (*Conversation, error) {
	if userID <= 0 || id <= 0 {
		return nil, ErrConversationNotFound
	}
	var c Conversation
	err := DB.Where("id = ? AND user_id = ?", id, userID).First(&c).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	return &c, nil
}

// UpdateConversationTitle sets the title for (id, user_id); 0 rows updated ->
// ErrConversationNotFound. Does NOT advance last_message_at (rename must not
// re-order the list).
func UpdateConversationTitle(userID, id int, title string) (*Conversation, error) {
	if userID <= 0 || id <= 0 {
		return nil, ErrConversationNotFound
	}
	title = strings.TrimSpace(title)
	if len(title) > maxConversationTitleLen {
		return nil, ErrConversationInvalid
	}
	res := DB.Model(&Conversation{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{"title": title, "updated_at": common.GetTimestamp()})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrConversationNotFound
	}
	return GetConversation(userID, id)
}

// DeleteConversation hard-deletes (id, user_id). On Postgres the FK ON DELETE
// CASCADE removes child messages; SQLite (tests) has no such FK, so children are
// removed explicitly (idempotent/harmless on Postgres).
func DeleteConversation(userID, id int) error {
	if userID <= 0 || id <= 0 {
		return ErrConversationNotFound
	}
	res := DB.Where("id = ? AND user_id = ?", id, userID).Delete(&Conversation{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConversationNotFound
	}
	// On Postgres the ON DELETE CASCADE FK already removed child messages.
	// SQLite/MySQL get no such FK (it is added Postgres-only), so delete the
	// children explicitly there.
	if !common.UsingPostgreSQL {
		if err := DB.Where("conversation_id = ? AND user_id = ?", id, userID).Delete(&ConversationMessage{}).Error; err != nil {
			return err
		}
	}
	return nil
}

// ListConversations returns a page of conversations ordered by
// (last_message_at DESC, id DESC) with a composite keyset cursor. It fetches one
// extra row to decide whether a further page exists.
func ListConversations(userID int, cursor *ConvCursor, limit int) ([]Conversation, *ConvCursor, error) {
	if userID <= 0 {
		return nil, nil, ErrConversationInvalid
	}
	limit = normalizeConversationLimit(limit)

	q := DB.Where("user_id = ?", userID)
	if cursor != nil {
		// rows strictly after (last_message_at DESC, id DESC).
		q = q.Where("last_message_at < ? OR (last_message_at = ? AND id < ?)",
			cursor.LastMessageAt, cursor.LastMessageAt, cursor.ID)
	}

	var items []Conversation
	if err := q.Order("last_message_at DESC, id DESC").Limit(limit + 1).Find(&items).Error; err != nil {
		return nil, nil, fmt.Errorf("list conversations: %w", err)
	}

	var next *ConvCursor
	if len(items) > limit {
		last := items[limit-1]
		next = &ConvCursor{LastMessageAt: last.LastMessageAt, ID: last.Id}
		items = items[:limit]
	}
	return items, next, nil
}

// touchConversationAt sets last_message_at for (id, user_id) to the newest
// inserted message's created_at, keeping last_message_at == MAX(message.created_at).
// Called inside the append/replace transaction; updated_at is bumped too.
func touchConversationAt(tx *gorm.DB, userID, id int, at int64) error {
	res := tx.Model(&Conversation{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]interface{}{"last_message_at": at, "updated_at": common.GetTimestamp()})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrConversationNotFound
	}
	return nil
}

// EncodeConvCursor encodes (v2:last_message_at_seconds:id) as base64.
func EncodeConvCursor(cur *ConvCursor) string {
	raw := conversationCursorVersion + ":" +
		strconv.FormatInt(cur.LastMessageAt, 10) + ":" +
		strconv.Itoa(cur.ID)
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeConvCursor decodes a cursor token. An empty token yields a nil cursor
// (first page); a malformed or non-v2 token yields errInvalidCursor.
func DecodeConvCursor(token string) (*ConvCursor, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, errInvalidCursor
	}
	parts := strings.SplitN(string(decoded), ":", 3)
	if len(parts) != 3 || parts[0] != conversationCursorVersion {
		return nil, errInvalidCursor
	}
	lma, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, errInvalidCursor
	}
	id, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, errInvalidCursor
	}
	return &ConvCursor{LastMessageAt: lma, ID: id}, nil
}
