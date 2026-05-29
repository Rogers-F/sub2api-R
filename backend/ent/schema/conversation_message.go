package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ConversationMessage holds the schema definition for a single message inside a
// conversation.
//
// Delete strategy: hard delete (removed via ON DELETE CASCADE when the parent
// conversation is deleted). This entity only carries a created_at timestamp
// (messages are immutable once written), so TimeMixin is intentionally not used.
//
// Token fields (reported_input_tokens / reported_output_tokens) are
// client-reported, display-only values. They are never used for billing.
//
// The SQL migration (081_add_conversations.sql) is the authoritative source for
// DDL, including CHECK constraints; this schema is kept consistent with it.
type ConversationMessage struct {
	ent.Schema
}

func (ConversationMessage) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "conversation_messages"},
	}
}

func (ConversationMessage) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("conversation_id"),
		// Redundant owner reference for defense-in-depth ownership filtering.
		field.Int64("user_id"),
		field.String("role").
			MaxLen(16).
			Comment("Message role: user, assistant"),
		field.String("content").
			SchemaType(map[string]string{dialect.Postgres: "text"}).
			Default(""),
		field.String("model").
			MaxLen(100).
			Default(""),
		field.String("status").
			MaxLen(16).
			Default("complete").
			Comment("Message status: complete, error"),
		// Client-reported token counts, display-only, never used for billing.
		field.Int("reported_input_tokens").
			Optional().
			Nillable().
			Comment("Client-reported prompt token count (display-only, not for billing)"),
		field.Int("reported_output_tokens").
			Optional().
			Nillable().
			Comment("Client-reported completion token count (display-only, not for billing)"),
		field.String("client_message_id").
			MaxLen(64).
			Comment("Client-generated identifier used for idempotent append"),
		field.String("gateway_request_id").
			MaxLen(128).
			Optional().
			Nillable().
			Comment("Upstream gateway request identifier, when available"),
		field.Time("created_at").
			Immutable().
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (ConversationMessage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", Conversation.Type).
			Ref("messages").
			Field("conversation_id").
			Unique().
			Required(),
		edge.From("user", User.Type).
			Ref("conversation_messages").
			Field("user_id").
			Unique().
			Required(),
	}
}

func (ConversationMessage) Indexes() []ent.Index {
	return []ent.Index{
		// Idempotent append per conversation.
		index.Fields("conversation_id", "client_message_id").Unique(),
		// Ownership-scoped listing within a conversation, ordered by id.
		index.Fields("user_id", "conversation_id", "id"),
	}
}
