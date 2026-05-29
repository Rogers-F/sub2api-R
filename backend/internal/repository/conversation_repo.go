package repository

import (
	"context"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/conversation"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// supportsRowLock reports whether the client's dialect supports
// SELECT ... FOR UPDATE. Production uses Postgres; SQLite (unit tests) does not.
func supportsRowLock(client *dbent.Client) bool {
	drv, ok := client.Driver().(interface{ Dialect() string })
	if !ok {
		return false
	}
	return drv.Dialect() == dialect.Postgres
}

type conversationRepository struct {
	client *dbent.Client
}

func NewConversationRepository(client *dbent.Client) service.ConversationRepository {
	return &conversationRepository{client: client}
}

func (r *conversationRepository) Create(ctx context.Context, c *service.Conversation) error {
	client := clientFromContext(ctx, r.client)
	// Pin created_at/updated_at/last_message_at to a single instant so the
	// "no messages -> last_message_at == created_at" invariant holds exactly
	// (the per-field ent defaults would each call time.Now() separately).
	now := time.Now()
	created, err := client.Conversation.Create().
		SetUserID(c.UserID).
		SetClientConversationID(c.ClientConversationID).
		SetTitle(c.Title).
		SetModel(c.Model).
		SetStatus(c.Status).
		SetCreatedAt(now).
		SetUpdatedAt(now).
		SetLastMessageAt(now).
		Save(ctx)
	if err != nil {
		// Unique (user_id, client_conversation_id) violation -> Conflict so the
		// service layer can resolve idempotently.
		return translatePersistenceError(err, nil, service.ErrMessageConflict)
	}
	applyConversationEntityToService(c, created)
	return nil
}

func (r *conversationRepository) GetByClientID(ctx context.Context, userID int64, clientConversationID string) (*service.Conversation, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.Conversation.Query().
		Where(
			conversation.UserIDEQ(userID),
			conversation.ClientConversationIDEQ(clientConversationID),
		).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrConversationNotFound, nil)
	}
	return conversationEntityToService(m), nil
}

func (r *conversationRepository) GetByID(ctx context.Context, userID, id int64) (*service.Conversation, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.Conversation.Query().
		Where(
			conversation.IDEQ(id),
			conversation.UserIDEQ(userID),
		).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrConversationNotFound, nil)
	}
	return conversationEntityToService(m), nil
}

func (r *conversationRepository) GetByIDForUpdate(ctx context.Context, userID, id int64) (*service.Conversation, error) {
	client := clientFromContext(ctx, r.client)
	q := client.Conversation.Query().
		Where(
			conversation.IDEQ(id),
			conversation.UserIDEQ(userID),
		)
	// Row-level locking is only emitted on dialects that support SELECT ... FOR
	// UPDATE (Postgres in production). Other dialects (e.g. SQLite used by unit
	// tests) do not support it, so the lock clause is skipped there.
	if supportsRowLock(client) {
		q = q.ForUpdate()
	}
	m, err := q.Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrConversationNotFound, nil)
	}
	return conversationEntityToService(m), nil
}

func (r *conversationRepository) List(ctx context.Context, userID int64, cursor *service.ConversationCursor, limit int) ([]service.Conversation, error) {
	client := clientFromContext(ctx, r.client)
	q := client.Conversation.Query().
		Where(conversation.UserIDEQ(userID))

	// Composite keyset cursor: rows strictly after (last_message_at DESC, id DESC).
	// (last_message_at < c.LastMessageAt) OR (last_message_at = c.LastMessageAt AND id < c.ID)
	if cursor != nil {
		q = q.Where(
			conversation.Or(
				conversation.LastMessageAtLT(cursor.LastMessageAt),
				conversation.And(
					conversation.LastMessageAtEQ(cursor.LastMessageAt),
					conversation.IDLT(cursor.ID),
				),
			),
		)
	}

	items, err := q.
		Order(
			conversation.ByLastMessageAt(entsql.OrderDesc()),
			conversation.ByID(entsql.OrderDesc()),
		).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return conversationEntitiesToService(items), nil
}

func (r *conversationRepository) UpdateTitle(ctx context.Context, userID, id int64, title string) error {
	client := clientFromContext(ctx, r.client)
	affected, err := client.Conversation.Update().
		Where(
			conversation.IDEQ(id),
			conversation.UserIDEQ(userID),
		).
		SetTitle(title).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrConversationNotFound
	}
	return nil
}

func (r *conversationRepository) Touch(ctx context.Context, userID, id int64, at time.Time) error {
	client := clientFromContext(ctx, r.client)
	// Advance the dedicated ordering key only. updated_at is auto-bumped by the
	// TimeMixin and is intentionally not used for list ordering.
	affected, err := client.Conversation.Update().
		Where(
			conversation.IDEQ(id),
			conversation.UserIDEQ(userID),
		).
		SetLastMessageAt(at).
		Save(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrConversationNotFound
	}
	return nil
}

func (r *conversationRepository) Delete(ctx context.Context, userID, id int64) error {
	client := clientFromContext(ctx, r.client)
	affected, err := client.Conversation.Delete().
		Where(
			conversation.IDEQ(id),
			conversation.UserIDEQ(userID),
		).
		Exec(ctx)
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrConversationNotFound
	}
	return nil
}

func applyConversationEntityToService(dst *service.Conversation, src *dbent.Conversation) {
	if dst == nil || src == nil {
		return
	}
	dst.ID = src.ID
	dst.CreatedAt = src.CreatedAt
	dst.UpdatedAt = src.UpdatedAt
	dst.LastMessageAt = src.LastMessageAt
}

func conversationEntityToService(m *dbent.Conversation) *service.Conversation {
	if m == nil {
		return nil
	}
	return &service.Conversation{
		ID:                   m.ID,
		UserID:               m.UserID,
		ClientConversationID: m.ClientConversationID,
		Title:                m.Title,
		Model:                m.Model,
		Status:               m.Status,
		CreatedAt:            m.CreatedAt,
		UpdatedAt:            m.UpdatedAt,
		LastMessageAt:        m.LastMessageAt,
	}
}

func conversationEntitiesToService(models []*dbent.Conversation) []service.Conversation {
	out := make([]service.Conversation, 0, len(models))
	for i := range models {
		if s := conversationEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
