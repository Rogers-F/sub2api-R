package model

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConversationMessage is a single persisted message within a conversation.
//
// ReportedInputTokens / ReportedOutputTokens are client-reported, display-only
// values. They are NEVER used for billing.
type ConversationMessage struct {
	Id                   int     `json:"id" gorm:"primaryKey;autoIncrement;index:idx_conversation_messages_user_conv_id,priority:3"`
	ConversationId       int     `json:"conversation_id" gorm:"not null;uniqueIndex:uq_conversation_messages_client,priority:1;index:idx_conversation_messages_user_conv_id,priority:2"`
	UserId               int     `json:"user_id" gorm:"not null;index:idx_conversation_messages_user_conv_id,priority:1"`
	Role                 string  `json:"role" gorm:"type:varchar(16);not null"`
	Content              string  `json:"content" gorm:"type:text;not null;default:''"`
	Model                string  `json:"model" gorm:"type:varchar(100);not null;default:''"`
	Status               string  `json:"status" gorm:"type:varchar(16);not null;default:'complete'"`
	ReportedInputTokens  *int    `json:"reported_input_tokens,omitempty" gorm:"column:reported_input_tokens;type:integer"`
	ReportedOutputTokens *int    `json:"reported_output_tokens,omitempty" gorm:"column:reported_output_tokens;type:integer"`
	ClientMessageId      string  `json:"client_message_id" gorm:"type:varchar(64);not null;uniqueIndex:uq_conversation_messages_client,priority:2"`
	GatewayRequestId     *string `json:"gateway_request_id,omitempty" gorm:"type:varchar(128)"`
	CreatedAt            int64   `json:"created_at" gorm:"bigint;not null"`
}

func (ConversationMessage) TableName() string { return "conversation_messages" }

// MessageInput is the input for appending/replacing a single message.
type MessageInput struct {
	Role                 string
	Content              string
	Model                string
	Status               string
	ReportedInputTokens  *int
	ReportedOutputTokens *int
	ClientMessageId      string
	GatewayRequestId     *string
}

// validateMessageInput enforces the service-layer rules before persistence and
// trims/normalizes role, status and client_message_id in place.
func validateMessageInput(in *MessageInput) error {
	in.Role = strings.TrimSpace(in.Role)
	switch in.Role {
	case MessageRoleUser, MessageRoleAssistant:
	default:
		// system and any other role are rejected.
		return ErrMessageInvalid
	}

	in.Status = strings.TrimSpace(in.Status)
	if in.Status == "" {
		in.Status = MessageStatusComplete
	}
	switch in.Status {
	case MessageStatusComplete, MessageStatusError:
	default:
		return ErrMessageInvalid
	}

	if in.Role == MessageRoleUser && in.Status != MessageStatusComplete {
		return ErrMessageInvalid
	}
	if in.Status == MessageStatusComplete && strings.TrimSpace(in.Content) == "" {
		return ErrMessageInvalid
	}
	if len(in.Content) > MaxMessageContentBytes {
		return ErrMessageInvalid
	}

	in.ClientMessageId = strings.TrimSpace(in.ClientMessageId)
	if in.ClientMessageId == "" || len(in.ClientMessageId) > maxMessageClientIDLen {
		return ErrMessageInvalid
	}
	if len(in.Model) > maxConversationModelLen {
		return ErrMessageInvalid
	}
	if in.GatewayRequestId != nil && len(*in.GatewayRequestId) > maxMessageGatewayReqIDLen {
		return ErrMessageInvalid
	}
	// Reported tokens map to a Postgres INT (int4) column: enforce [0, MaxInt32].
	if in.ReportedInputTokens != nil && (*in.ReportedInputTokens < 0 || *in.ReportedInputTokens > math.MaxInt32) {
		return ErrMessageInvalid
	}
	if in.ReportedOutputTokens != nil && (*in.ReportedOutputTokens < 0 || *in.ReportedOutputTokens > math.MaxInt32) {
		return ErrMessageInvalid
	}
	return nil
}

func messageFromInput(conversationID, userID int, in *MessageInput) *ConversationMessage {
	return &ConversationMessage{
		ConversationId:       conversationID,
		UserId:               userID,
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

// messageFingerprintEqual compares the idempotency-relevant fields: role,
// content, model, status, reported tokens, gateway_request_id.
func messageFingerprintEqual(a, b *ConversationMessage) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Role == b.Role &&
		a.Content == b.Content &&
		a.Model == b.Model &&
		a.Status == b.Status &&
		ptrEqual(a.ReportedInputTokens, b.ReportedInputTokens) &&
		ptrEqual(a.ReportedOutputTokens, b.ReportedOutputTokens) &&
		ptrEqual(a.GatewayRequestId, b.GatewayRequestId)
}

// ptrEqual reports whether two pointers reference equal values, treating two nil
// pointers as equal and a nil/non-nil pair as unequal.
func ptrEqual[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// --- message persistence helpers (all scoped by user_id to prevent IDOR) ---

func getMessageByClientID(db *gorm.DB, userID, conversationID int, clientMessageID string) (*ConversationMessage, error) {
	var m ConversationMessage
	err := db.Where("user_id = ? AND conversation_id = ? AND client_message_id = ?",
		userID, conversationID, clientMessageID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	return &m, nil
}

func getMessageByID(db *gorm.DB, userID, conversationID, id int) (*ConversationMessage, error) {
	var m ConversationMessage
	err := db.Where("id = ? AND user_id = ? AND conversation_id = ?",
		id, userID, conversationID).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	return &m, nil
}

func maxMessageID(db *gorm.DB, userID, conversationID int) (int, bool, error) {
	var m ConversationMessage
	err := db.Where("user_id = ? AND conversation_id = ?", userID, conversationID).
		Order("id DESC").First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return m.Id, true, nil
}

func deleteMessagesFrom(db *gorm.DB, userID, conversationID, fromID int) (int64, error) {
	res := db.Where("user_id = ? AND conversation_id = ? AND id >= ?",
		userID, conversationID, fromID).Delete(&ConversationMessage{})
	return res.RowsAffected, res.Error
}

// lockConversation loads the parent conversation for (id, user_id), taking a
// row-level lock on Postgres (skipped on SQLite). Must run inside a transaction.
func lockConversation(tx *gorm.DB, userID, id int) (*Conversation, error) {
	q := tx.Where("id = ? AND user_id = ?", id, userID)
	if usePGRowLock() {
		// clause.Locking is the GORM v2 row-lock API and renders "FOR UPDATE" on
		// Postgres. The legacy Set("gorm:query_option","FOR UPDATE") escape hatch
		// is a silent no-op on this GORM version (verified via DryRun), so it must
		// NOT be used for the append/replace serialization.
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var c Conversation
	if err := q.First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConversationNotFound
		}
		return nil, err
	}
	return &c, nil
}

// ListMessages returns a page of messages (id ASC) using forward pagination
// (id > afterID). Retained for backward compatibility with the legacy cursor.
// hasMore reports whether a further (newer) page exists.
func ListMessages(userID, conversationID, afterID, limit int) ([]ConversationMessage, bool, error) {
	if userID <= 0 || conversationID <= 0 {
		return nil, false, ErrConversationNotFound
	}
	if _, err := GetConversation(userID, conversationID); err != nil {
		return nil, false, err
	}
	limit = normalizeConversationLimit(limit)
	q := DB.Where("user_id = ? AND conversation_id = ?", userID, conversationID)
	if afterID > 0 {
		q = q.Where("id > ?", afterID)
	}
	var items []ConversationMessage
	if err := q.Order("id ASC").Limit(limit + 1).Find(&items).Error; err != nil {
		return nil, false, fmt.Errorf("list messages: %w", err)
	}
	hasMore := false
	if len(items) > limit {
		hasMore = true
		items = items[:limit]
	}
	return items, hasMore, nil
}

// ListMessagesBefore returns a page of messages using backward (newest-first)
// pagination: id < beforeID (or the newest page when beforeID <= 0), returned
// id ASC for display. hasMore reports whether still-older messages exist.
func ListMessagesBefore(userID, conversationID, beforeID, limit int) ([]ConversationMessage, bool, error) {
	if userID <= 0 || conversationID <= 0 {
		return nil, false, ErrConversationNotFound
	}
	if beforeID < 0 {
		return nil, false, ErrMessageInvalid
	}
	if _, err := GetConversation(userID, conversationID); err != nil {
		return nil, false, err
	}
	limit = normalizeConversationLimit(limit)
	q := DB.Where("user_id = ? AND conversation_id = ?", userID, conversationID)
	if beforeID > 0 {
		q = q.Where("id < ?", beforeID)
	}
	var rows []ConversationMessage
	if err := q.Order("id DESC").Limit(limit + 1).Find(&rows).Error; err != nil {
		return nil, false, fmt.Errorf("list messages before: %w", err)
	}
	hasMore := false
	if len(rows) > limit {
		hasMore = true
		rows = rows[:limit]
	}
	// Reverse id DESC -> id ASC for display.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	return rows, hasMore, nil
}

// AppendMessages inserts a batch of messages in one transaction: lock the parent
// (verifying ownership), insert messages, and advance last_message_at to the
// newest inserted message's created_at — but only when at least one new message
// was actually inserted (pure idempotent retries must not re-order the list).
//
// Idempotency per message is keyed on (conversation_id, client_message_id):
// identical existing payload returns the existing row; different payload -> 409.
// Duplicate client_message_id within the same batch -> 400.
func AppendMessages(userID, conversationID int, inputs []MessageInput) ([]ConversationMessage, error) {
	if userID <= 0 || conversationID <= 0 {
		return nil, ErrConversationNotFound
	}
	if len(inputs) == 0 || len(inputs) > maxMessageBatchSize {
		return nil, ErrMessageInvalid
	}

	seen := make(map[string]struct{}, len(inputs))
	for i := range inputs {
		if err := validateMessageInput(&inputs[i]); err != nil {
			return nil, err
		}
		if _, dup := seen[inputs[i].ClientMessageId]; dup {
			return nil, ErrMessageDuplicateInBatch
		}
		seen[inputs[i].ClientMessageId] = struct{}{}
	}

	var results []ConversationMessage
	err := DB.Transaction(func(tx *gorm.DB) error {
		if _, err := lockConversation(tx, userID, conversationID); err != nil {
			return err
		}
		results = make([]ConversationMessage, 0, len(inputs))
		inserted := false
		var maxCreatedAt int64
		for i := range inputs {
			m, created, err := appendOne(tx, userID, conversationID, &inputs[i])
			if err != nil {
				return err
			}
			if created {
				inserted = true
				if m.CreatedAt > maxCreatedAt {
					maxCreatedAt = m.CreatedAt
				}
			}
			results = append(results, *m)
		}
		if inserted {
			return touchConversationAt(tx, userID, conversationID, maxCreatedAt)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// appendOne inserts a single message under the parent FOR UPDATE lock. It does a
// read-then-insert: if a row already exists for (conversation_id,
// client_message_id), compare the fingerprint (equal -> idempotent return;
// differ -> 409); otherwise insert. The second return reports whether a new row
// was inserted. A failed insert (genuine race despite the lock) is reported as a
// conflict rather than re-read, because on Postgres the failed INSERT aborts the
// surrounding transaction.
func appendOne(tx *gorm.DB, userID, conversationID int, input *MessageInput) (*ConversationMessage, bool, error) {
	m := messageFromInput(conversationID, userID, input)

	existing, getErr := getMessageByClientID(tx, userID, conversationID, input.ClientMessageId)
	if getErr == nil {
		if !messageFingerprintEqual(existing, m) {
			return nil, false, ErrMessageConflict
		}
		return existing, false, nil
	}
	if !errors.Is(getErr, ErrConversationNotFound) {
		return nil, false, getErr
	}

	m.CreatedAt = common.GetTimestamp()
	if err := tx.Create(m).Error; err != nil {
		return nil, false, ErrMessageConflict
	}
	return m, true, nil
}

// ReplaceMessageFrom atomically replaces a conversation's trailing assistant
// message: it deletes the cutoff (and anything after it) and inserts a new
// assistant message in a single transaction, advancing last_message_at.
//
// Safety: exactly one of cutoffID / cutoffClientID identifies the cutoff, which
// must resolve within (user_id, conversation_id); it must be an assistant message
// AND the conversation's current last message (cutoff.id == MAX(id)), otherwise a
// concurrent append changed the tail and we refuse (409). Idempotent on
// newMsg.ClientMessageId: a retry after success returns the existing replacement;
// same-id-different-payload or a still-present cutoff is a 409.
func ReplaceMessageFrom(userID, conversationID int, cutoffID int, cutoffClientID string, newMsg MessageInput) (*ConversationMessage, error) {
	if userID <= 0 || conversationID <= 0 {
		return nil, ErrConversationNotFound
	}
	if cutoffID < 0 {
		return nil, ErrMessageInvalid
	}
	cutoffClientID = strings.TrimSpace(cutoffClientID)
	// Exactly one cutoff identifier must be supplied.
	if (cutoffID > 0) == (cutoffClientID != "") {
		return nil, ErrMessageInvalid
	}
	if err := validateMessageInput(&newMsg); err != nil {
		return nil, err
	}
	// Replacement is always a completed assistant message.
	if newMsg.Role != MessageRoleAssistant || newMsg.Status != MessageStatusComplete {
		return nil, ErrMessageInvalid
	}

	var out *ConversationMessage
	err := DB.Transaction(func(tx *gorm.DB) error {
		if _, err := lockConversation(tx, userID, conversationID); err != nil {
			return err
		}

		candidate := messageFromInput(conversationID, userID, &newMsg)

		// Idempotency: has the replacement already been persisted?
		existingNew, getErr := getMessageByClientID(tx, userID, conversationID, newMsg.ClientMessageId)
		if getErr == nil {
			if !messageFingerprintEqual(existingNew, candidate) {
				return ErrMessageConflict
			}
			// Same payload: a retry is a success only once the cutoff is gone.
			stillExists, err := cutoffStillExists(tx, userID, conversationID, cutoffID, cutoffClientID)
			if err != nil {
				return err
			}
			if stillExists {
				return ErrMessageConflict
			}
			out = existingNew
			return nil
		}
		if !errors.Is(getErr, ErrConversationNotFound) {
			return getErr
		}

		// Resolve and validate the cutoff.
		cutoff, err := resolveCutoff(tx, userID, conversationID, cutoffID, cutoffClientID)
		if err != nil {
			return err // ErrConversationNotFound -> 404
		}
		if cutoff.Role != MessageRoleAssistant {
			return ErrMessageConflict
		}
		maxID, ok, err := maxMessageID(tx, userID, conversationID)
		if err != nil {
			return fmt.Errorf("max message id: %w", err)
		}
		if !ok || cutoff.Id != maxID {
			return ErrMessageConflict
		}

		if _, err := deleteMessagesFrom(tx, userID, conversationID, cutoff.Id); err != nil {
			return fmt.Errorf("delete suffix: %w", err)
		}
		// Reuse candidate (built above for the fingerprint check) as the new row.
		candidate.CreatedAt = common.GetTimestamp()
		if err := tx.Create(candidate).Error; err != nil {
			return ErrMessageConflict
		}
		if err := touchConversationAt(tx, userID, conversationID, candidate.CreatedAt); err != nil {
			return err
		}
		out = candidate
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func resolveCutoff(db *gorm.DB, userID, conversationID, cutoffID int, cutoffClientID string) (*ConversationMessage, error) {
	if cutoffID > 0 {
		return getMessageByID(db, userID, conversationID, cutoffID)
	}
	return getMessageByClientID(db, userID, conversationID, cutoffClientID)
}

func cutoffStillExists(db *gorm.DB, userID, conversationID, cutoffID int, cutoffClientID string) (bool, error) {
	_, err := resolveCutoff(db, userID, conversationID, cutoffID, cutoffClientID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrConversationNotFound) {
		return false, nil
	}
	return false, err
}
