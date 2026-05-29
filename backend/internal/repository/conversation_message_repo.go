package repository

import (
	"context"

	entsql "entgo.io/ent/dialect/sql"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/conversationmessage"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type messageRepository struct {
	client *dbent.Client
}

func NewMessageRepository(client *dbent.Client) service.MessageRepository {
	return &messageRepository{client: client}
}

func (r *messageRepository) GetByClientID(ctx context.Context, userID, conversationID int64, clientMessageID string) (*service.Message, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.ConversationMessage.Query().
		Where(
			conversationmessage.UserIDEQ(userID),
			conversationmessage.ConversationIDEQ(conversationID),
			conversationmessage.ClientMessageIDEQ(clientMessageID),
		).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrConversationNotFound, nil)
	}
	return messageEntityToService(m), nil
}

func (r *messageRepository) GetByID(ctx context.Context, userID, conversationID, id int64) (*service.Message, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.ConversationMessage.Query().
		Where(
			conversationmessage.IDEQ(id),
			conversationmessage.UserIDEQ(userID),
			conversationmessage.ConversationIDEQ(conversationID),
		).
		Only(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrConversationNotFound, nil)
	}
	return messageEntityToService(m), nil
}

func (r *messageRepository) Create(ctx context.Context, m *service.Message) error {
	client := clientFromContext(ctx, r.client)
	builder := client.ConversationMessage.Create().
		SetConversationID(m.ConversationID).
		SetUserID(m.UserID).
		SetRole(m.Role).
		SetContent(m.Content).
		SetModel(m.Model).
		SetStatus(m.Status).
		SetClientMessageID(m.ClientMessageID)

	if m.ReportedInputTokens != nil {
		builder.SetReportedInputTokens(*m.ReportedInputTokens)
	}
	if m.ReportedOutputTokens != nil {
		builder.SetReportedOutputTokens(*m.ReportedOutputTokens)
	}
	if m.GatewayRequestID != nil {
		builder.SetGatewayRequestID(*m.GatewayRequestID)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		// Unique (conversation_id, client_message_id) violation -> Conflict so the
		// service layer can resolve idempotency.
		return translatePersistenceError(err, nil, service.ErrMessageConflict)
	}
	m.ID = created.ID
	m.CreatedAt = created.CreatedAt
	return nil
}

func (r *messageRepository) List(ctx context.Context, userID, conversationID int64, afterID int64, limit int) ([]service.Message, error) {
	client := clientFromContext(ctx, r.client)
	q := client.ConversationMessage.Query().
		Where(
			conversationmessage.UserIDEQ(userID),
			conversationmessage.ConversationIDEQ(conversationID),
		)
	if afterID > 0 {
		q = q.Where(conversationmessage.IDGT(afterID))
	}
	items, err := q.
		Order(conversationmessage.ByID(entsql.OrderAsc())).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return messageEntitiesToService(items), nil
}

func (r *messageRepository) ListBefore(ctx context.Context, userID, conversationID int64, beforeID int64, limit int) ([]service.Message, error) {
	client := clientFromContext(ctx, r.client)
	q := client.ConversationMessage.Query().
		Where(
			conversationmessage.UserIDEQ(userID),
			conversationmessage.ConversationIDEQ(conversationID),
		)
	if beforeID > 0 {
		q = q.Where(conversationmessage.IDLT(beforeID))
	}
	// Newest-first; the service trims the extra row and reverses to id ASC.
	items, err := q.
		Order(conversationmessage.ByID(entsql.OrderDesc())).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return messageEntitiesToService(items), nil
}

func (r *messageRepository) MaxMessageID(ctx context.Context, userID, conversationID int64) (int64, bool, error) {
	client := clientFromContext(ctx, r.client)
	m, err := client.ConversationMessage.Query().
		Where(
			conversationmessage.UserIDEQ(userID),
			conversationmessage.ConversationIDEQ(conversationID),
		).
		Order(conversationmessage.ByID(entsql.OrderDesc())).
		First(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return m.ID, true, nil
}

func (r *messageRepository) DeleteFrom(ctx context.Context, userID, conversationID, fromID int64) (int, error) {
	client := clientFromContext(ctx, r.client)
	return client.ConversationMessage.Delete().
		Where(
			conversationmessage.UserIDEQ(userID),
			conversationmessage.ConversationIDEQ(conversationID),
			conversationmessage.IDGTE(fromID),
		).
		Exec(ctx)
}

func messageEntityToService(m *dbent.ConversationMessage) *service.Message {
	if m == nil {
		return nil
	}
	out := &service.Message{
		ID:              m.ID,
		ConversationID:  m.ConversationID,
		UserID:          m.UserID,
		Role:            m.Role,
		Content:         m.Content,
		Model:           m.Model,
		Status:          m.Status,
		ClientMessageID: m.ClientMessageID,
		CreatedAt:       m.CreatedAt,
	}
	if m.ReportedInputTokens != nil {
		v := *m.ReportedInputTokens
		out.ReportedInputTokens = &v
	}
	if m.ReportedOutputTokens != nil {
		v := *m.ReportedOutputTokens
		out.ReportedOutputTokens = &v
	}
	if m.GatewayRequestID != nil {
		v := *m.GatewayRequestID
		out.GatewayRequestID = &v
	}
	return out
}

func messageEntitiesToService(models []*dbent.ConversationMessage) []service.Message {
	out := make([]service.Message, 0, len(models))
	for i := range models {
		if s := messageEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
