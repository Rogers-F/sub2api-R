package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	// defaultConversationListLimit is used when the caller does not specify one.
	defaultConversationListLimit = 20
	// maxConversationListLimit caps page size to bound query cost.
	maxConversationListLimit = 100
	// maxMessageBatchSize caps the number of messages appended in one request.
	maxMessageBatchSize = 50
)

// ConversationService implements persisted multi-conversation chat operations.
//
// This layer does NOT touch billing and does NOT call the gateway. Token fields
// carried on messages are client-reported and display-only.
type ConversationService struct {
	entClient        *dbent.Client
	conversationRepo ConversationRepository
	messageRepo      MessageRepository
}

func NewConversationService(
	entClient *dbent.Client,
	conversationRepo ConversationRepository,
	messageRepo MessageRepository,
) *ConversationService {
	return &ConversationService{
		entClient:        entClient,
		conversationRepo: conversationRepo,
		messageRepo:      messageRepo,
	}
}

// ConversationList is the result of ListConversations: a page of items plus an
// optional cursor pointing at the next page.
type ConversationList struct {
	Items      []Conversation
	NextCursor *ConversationCursor
}

// CreateConversation is idempotent on (user_id, client_conversation_id): if a
// conversation already exists for the pair, the existing one is returned.
func (s *ConversationService) CreateConversation(
	ctx context.Context,
	userID int64,
	clientConversationID, title, model string,
) (*Conversation, error) {
	if userID <= 0 {
		return nil, ErrConversationInvalid
	}
	clientConversationID = strings.TrimSpace(clientConversationID)
	if clientConversationID == "" || len(clientConversationID) > 64 {
		return nil, ErrConversationInvalid
	}
	title = strings.TrimSpace(title)
	if len(title) > 255 {
		return nil, ErrConversationInvalid
	}
	if len(model) > 100 {
		return nil, ErrConversationInvalid
	}

	// Fast path: return the existing conversation if present.
	existing, err := s.conversationRepo.GetByClientID(ctx, userID, clientConversationID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, ErrConversationNotFound) {
		return nil, err
	}

	c := &Conversation{
		UserID:               userID,
		ClientConversationID: clientConversationID,
		Title:                title,
		Model:                model,
		Status:               ConversationStatusActive,
	}
	if err := s.conversationRepo.Create(ctx, c); err != nil {
		// Concurrent creation: the unique constraint may have fired between the
		// lookup above and the insert. Resolve idempotently by re-reading.
		if infraerrors.IsConflict(err) {
			if existing, getErr := s.conversationRepo.GetByClientID(ctx, userID, clientConversationID); getErr == nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("create conversation: %w", err)
	}
	return c, nil
}

// ListConversations returns a page of conversations ordered by
// (updated_at DESC, id DESC) with a composite cursor.
func (s *ConversationService) ListConversations(
	ctx context.Context,
	userID int64,
	cursor *ConversationCursor,
	limit int,
) (*ConversationList, error) {
	if userID <= 0 {
		return nil, ErrConversationInvalid
	}
	limit = normalizeConversationLimit(limit, defaultConversationListLimit, maxConversationListLimit)

	// Fetch one extra row to detect whether another page exists.
	items, err := s.conversationRepo.List(ctx, userID, cursor, limit+1)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}

	out := &ConversationList{}
	if len(items) > limit {
		last := items[limit-1]
		out.NextCursor = &ConversationCursor{LastMessageAt: last.LastMessageAt, ID: last.ID}
		items = items[:limit]
	}
	out.Items = items
	return out, nil
}

// GetConversation returns conversation metadata for (id, user_id), or
// ErrConversationNotFound if not owned/missing.
func (s *ConversationService) GetConversation(ctx context.Context, userID, id int64) (*Conversation, error) {
	if userID <= 0 || id <= 0 {
		return nil, ErrConversationNotFound
	}
	return s.conversationRepo.GetByID(ctx, userID, id)
}

// UpdateTitle sets the title for (id, user_id); 0 rows updated -> not found.
func (s *ConversationService) UpdateTitle(ctx context.Context, userID, id int64, title string) (*Conversation, error) {
	if userID <= 0 || id <= 0 {
		return nil, ErrConversationNotFound
	}
	title = strings.TrimSpace(title)
	if len(title) > 255 {
		return nil, ErrConversationInvalid
	}
	if err := s.conversationRepo.UpdateTitle(ctx, userID, id, title); err != nil {
		return nil, err
	}
	return s.conversationRepo.GetByID(ctx, userID, id)
}

// DeleteConversation hard-deletes (id, user_id); cascade removes messages.
func (s *ConversationService) DeleteConversation(ctx context.Context, userID, id int64) error {
	if userID <= 0 || id <= 0 {
		return ErrConversationNotFound
	}
	return s.conversationRepo.Delete(ctx, userID, id)
}

// MessageList is the result of ListMessages: a page of messages plus a flag
// indicating whether a further page exists (so the handler only emits a cursor
// when there is more to fetch).
type MessageList struct {
	Items   []Message
	HasMore bool
}

// ListMessages returns a page of messages within a conversation owned by the
// user, ordered by id ASC.
func (s *ConversationService) ListMessages(
	ctx context.Context,
	userID, conversationID int64,
	afterID int64,
	limit int,
) (*MessageList, error) {
	if userID <= 0 || conversationID <= 0 {
		return nil, ErrConversationNotFound
	}
	// Verify parent ownership before listing children.
	if _, err := s.conversationRepo.GetByID(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	limit = normalizeConversationLimit(limit, defaultConversationListLimit, maxConversationListLimit)
	// Fetch one extra row to detect whether another page exists.
	items, err := s.messageRepo.List(ctx, userID, conversationID, afterID, limit+1)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	out := &MessageList{}
	if len(items) > limit {
		out.HasMore = true
		items = items[:limit]
	}
	out.Items = items
	return out, nil
}

// ListMessagesBefore returns a page of messages within a conversation owned by
// the user using backward (newest-first) pagination: messages with id < beforeID
// (or the newest page when beforeID <= 0), returned id ASC for display.
// HasMore reports whether still-older messages exist.
func (s *ConversationService) ListMessagesBefore(
	ctx context.Context,
	userID, conversationID int64,
	beforeID int64,
	limit int,
) (*MessageList, error) {
	if userID <= 0 || conversationID <= 0 {
		return nil, ErrConversationNotFound
	}
	if beforeID < 0 {
		return nil, ErrMessageInvalid
	}
	// Verify parent ownership before listing children.
	if _, err := s.conversationRepo.GetByID(ctx, userID, conversationID); err != nil {
		return nil, err
	}
	limit = normalizeConversationLimit(limit, defaultConversationListLimit, maxConversationListLimit)
	// Fetch one extra row (oldest of the page) to detect whether more exist.
	rows, err := s.messageRepo.ListBefore(ctx, userID, conversationID, beforeID, limit+1)
	if err != nil {
		return nil, fmt.Errorf("list messages before: %w", err)
	}

	out := &MessageList{}
	if len(rows) > limit {
		out.HasMore = true
		// rows are id DESC; the extra row is the oldest -> drop the tail.
		rows = rows[:limit]
	}
	// Reverse id DESC -> id ASC for display.
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
	out.Items = rows
	return out, nil
}

// AppendMessages inserts a batch of messages into a conversation in a single
// transaction: lock the parent conversation FOR UPDATE (verifying ownership),
// insert messages, and bump the conversation's updated_at.
//
// Idempotency per message is keyed on (conversation_id, client_message_id):
//   - identical existing payload -> the existing message is returned (no error)
//   - different existing payload -> ErrMessageConflict (409)
//
// Duplicate client_message_id within the same batch is rejected (400).
func (s *ConversationService) AppendMessages(
	ctx context.Context,
	userID, conversationID int64,
	inputs []MessageInput,
) ([]Message, error) {
	if userID <= 0 || conversationID <= 0 {
		return nil, ErrConversationNotFound
	}
	if len(inputs) == 0 {
		return nil, ErrMessageInvalid
	}
	if len(inputs) > maxMessageBatchSize {
		return nil, ErrMessageInvalid
	}

	// Validate each input and detect intra-batch duplicate client_message_id.
	seen := make(map[string]struct{}, len(inputs))
	for i := range inputs {
		if err := validateMessageInput(&inputs[i]); err != nil {
			return nil, err
		}
		if _, dup := seen[inputs[i].ClientMessageID]; dup {
			return nil, ErrMessageDuplicateInBatch
		}
		seen[inputs[i].ClientMessageID] = struct{}{}
	}

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin conversation transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	txCtx := dbent.NewTxContext(ctx, tx)

	// Lock parent and verify ownership.
	if _, err := s.conversationRepo.GetByIDForUpdate(txCtx, userID, conversationID); err != nil {
		return nil, err
	}

	results := make([]Message, 0, len(inputs))
	inserted := false
	var maxCreatedAt time.Time
	for i := range inputs {
		m, created, err := s.appendOne(txCtx, userID, conversationID, &inputs[i])
		if err != nil {
			return nil, err
		}
		if created {
			inserted = true
			if m.CreatedAt.After(maxCreatedAt) {
				maxCreatedAt = m.CreatedAt
			}
		}
		results = append(results, *m)
	}

	// Only advance last_message_at when at least one new message was actually
	// inserted, and set it to the newest inserted message's created_at so that
	// last_message_at == MAX(message.created_at) holds exactly. Pure idempotent
	// retries must not reorder the conversation list.
	if inserted {
		if err := s.conversationRepo.Touch(txCtx, userID, conversationID, maxCreatedAt); err != nil {
			return nil, fmt.Errorf("touch conversation: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit conversation transaction: %w", err)
	}
	committed = true
	return results, nil
}

// appendOne inserts a single message, resolving idempotency conflicts.
//
// The parent conversation row is held FOR UPDATE by the caller, so concurrent
// appends to the same conversation are serialized. This lets us safely
// "read-then-insert": look up an existing row by (conversation_id,
// client_message_id) first, then insert only when absent. This avoids relying
// on a failed INSERT inside the transaction, which on PostgreSQL would abort
// the whole transaction and break the conflict-then-reread path.
//
// The second return value reports whether a new row was inserted.
func (s *ConversationService) appendOne(
	ctx context.Context,
	userID, conversationID int64,
	input *MessageInput,
) (*Message, bool, error) {
	m := messageFromInput(conversationID, userID, input)

	// Idempotency check first: if a row already exists for this
	// (conversation_id, client_message_id), compare the fingerprint.
	existing, getErr := s.messageRepo.GetByClientID(ctx, userID, conversationID, input.ClientMessageID)
	if getErr == nil {
		if !messageFingerprintEqual(existing, m) {
			return nil, false, ErrMessageConflict
		}
		return existing, false, nil
	}
	if !errors.Is(getErr, ErrConversationNotFound) {
		return nil, false, getErr
	}

	// No existing row: insert. A unique violation here can only happen under a
	// genuine race despite the parent lock; treat it as a conflict to reread.
	if err := s.messageRepo.Create(ctx, m); err != nil {
		if infraerrors.IsConflict(err) {
			return nil, false, ErrMessageConflict
		}
		return nil, false, fmt.Errorf("create message: %w", err)
	}
	return m, true, nil
}

// ReplaceMessageFrom atomically replaces a conversation's trailing assistant
// message: it deletes the cutoff message (and anything after it) and inserts a
// new assistant message, in a single transaction, advancing last_message_at.
//
// Used by "regenerate", which streams the new reply first and only calls this
// once the new reply reaches a terminal state — so a failed/aborted stream
// leaves the server untouched (no irreversible intermediate state).
//
// Safety:
//   - Exactly one of cutoffID / cutoffClientID identifies the cutoff; it must
//     resolve within (user_id, conversation_id) (IDOR + no foreign/garbage id).
//   - The cutoff must be an assistant message AND the conversation's current
//     last message (cutoff.id == MAX(id)); otherwise a concurrent append has
//     changed the tail and we refuse (409) rather than delete someone else's
//     newer message.
//   - Idempotent on newMsg.ClientMessageID: a retry after success returns the
//     existing replacement; a same-id-different-payload or still-present-cutoff
//     state is a 409.
func (s *ConversationService) ReplaceMessageFrom(
	ctx context.Context,
	userID, conversationID int64,
	cutoffID int64,
	cutoffClientID string,
	newMsg MessageInput,
) (*Message, error) {
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

	tx, err := s.entClient.Tx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin replace transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	txCtx := dbent.NewTxContext(ctx, tx)

	// Lock parent (serializes against AppendMessages / concurrent replaces).
	if _, err := s.conversationRepo.GetByIDForUpdate(txCtx, userID, conversationID); err != nil {
		return nil, err
	}

	candidate := messageFromInput(conversationID, userID, &newMsg)

	// Idempotency: has the replacement already been persisted?
	existingNew, getErr := s.messageRepo.GetByClientID(txCtx, userID, conversationID, newMsg.ClientMessageID)
	if getErr == nil {
		if !messageFingerprintEqual(existingNew, candidate) {
			return nil, ErrMessageConflict
		}
		// Same payload: a retry is a success only once the cutoff is gone (i.e.
		// the replace already happened). If the cutoff still exists, the state
		// is inconsistent (e.g. id reused via append) -> conflict.
		cutoffExists, err := s.cutoffExists(txCtx, userID, conversationID, cutoffID, cutoffClientID)
		if err != nil {
			return nil, err
		}
		if cutoffExists {
			return nil, ErrMessageConflict
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit replace transaction: %w", err)
		}
		committed = true
		return existingNew, nil
	}
	if !errors.Is(getErr, ErrConversationNotFound) {
		return nil, getErr
	}

	// Resolve and validate the cutoff.
	cutoff, err := s.resolveCutoff(txCtx, userID, conversationID, cutoffID, cutoffClientID)
	if err != nil {
		return nil, err // ErrConversationNotFound -> 404
	}
	if cutoff.Role != MessageRoleAssistant {
		return nil, ErrMessageConflict
	}
	maxID, ok, err := s.messageRepo.MaxMessageID(txCtx, userID, conversationID)
	if err != nil {
		return nil, fmt.Errorf("max message id: %w", err)
	}
	// The cutoff must be the current last message; otherwise the tail changed.
	if !ok || cutoff.ID != maxID {
		return nil, ErrMessageConflict
	}

	if _, err := s.messageRepo.DeleteFrom(txCtx, userID, conversationID, cutoff.ID); err != nil {
		return nil, fmt.Errorf("delete suffix: %w", err)
	}

	inserted := messageFromInput(conversationID, userID, &newMsg)
	if err := s.messageRepo.Create(txCtx, inserted); err != nil {
		if infraerrors.IsConflict(err) {
			return nil, ErrMessageConflict
		}
		return nil, fmt.Errorf("insert replacement: %w", err)
	}
	// The new message id is the largest -> last_message_at = its created_at,
	// preserving last_message_at == MAX(message.created_at).
	if err := s.conversationRepo.Touch(txCtx, userID, conversationID, inserted.CreatedAt); err != nil {
		return nil, fmt.Errorf("touch conversation: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit replace transaction: %w", err)
	}
	committed = true
	return inserted, nil
}

// resolveCutoff loads the cutoff message identified by exactly one of (id,
// clientID), scoped to (user_id, conversation_id). Returns ErrConversationNotFound
// when it does not exist in this conversation.
func (s *ConversationService) resolveCutoff(
	ctx context.Context,
	userID, conversationID, cutoffID int64,
	cutoffClientID string,
) (*Message, error) {
	if cutoffID > 0 {
		return s.messageRepo.GetByID(ctx, userID, conversationID, cutoffID)
	}
	return s.messageRepo.GetByClientID(ctx, userID, conversationID, cutoffClientID)
}

// cutoffExists reports whether the cutoff message still exists in (user_id,
// conversation_id).
func (s *ConversationService) cutoffExists(
	ctx context.Context,
	userID, conversationID, cutoffID int64,
	cutoffClientID string,
) (bool, error) {
	_, err := s.resolveCutoff(ctx, userID, conversationID, cutoffID, cutoffClientID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, ErrConversationNotFound) {
		return false, nil
	}
	return false, err
}

// messageFromInput builds a *Message from a validated MessageInput.
func messageFromInput(conversationID, userID int64, in *MessageInput) *Message {
	return &Message{
		ConversationID:       conversationID,
		UserID:               userID,
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

// validateMessageInput enforces service-layer rules before persistence.
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

	// User messages must be complete.
	if in.Role == MessageRoleUser && in.Status != MessageStatusComplete {
		return ErrMessageInvalid
	}

	// A complete message must carry non-blank content.
	if in.Status == MessageStatusComplete && strings.TrimSpace(in.Content) == "" {
		return ErrMessageInvalid
	}

	if len(in.Content) > MaxMessageContentBytes {
		return ErrMessageInvalid
	}

	in.ClientMessageID = strings.TrimSpace(in.ClientMessageID)
	if in.ClientMessageID == "" || len(in.ClientMessageID) > 64 {
		return ErrMessageInvalid
	}

	if len(in.Model) > 100 {
		return ErrMessageInvalid
	}
	if in.GatewayRequestID != nil && len(*in.GatewayRequestID) > 128 {
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

// messageFingerprintEqual compares the idempotency-relevant fields of two
// messages: role, content, model, status, reported tokens, gateway_request_id.
func messageFingerprintEqual(a, b *Message) bool {
	if a == nil || b == nil {
		return false
	}
	return a.Role == b.Role &&
		a.Content == b.Content &&
		a.Model == b.Model &&
		a.Status == b.Status &&
		ptrEqual(a.ReportedInputTokens, b.ReportedInputTokens) &&
		ptrEqual(a.ReportedOutputTokens, b.ReportedOutputTokens) &&
		ptrEqual(a.GatewayRequestID, b.GatewayRequestID)
}

// ptrEqual reports whether two pointers reference equal values, treating two
// nil pointers as equal and a nil/non-nil pair as unequal.
func ptrEqual[T comparable](a, b *T) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// normalizeConversationLimit clamps a requested limit into [1, max], defaulting
// when <= 0.
func normalizeConversationLimit(limit, def, max int) int {
	if limit <= 0 {
		return def
	}
	if limit > max {
		return max
	}
	return limit
}
