package schema

import (
	"time"

	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Conversation holds the schema definition for a persisted chat conversation.
//
// Delete strategy: hard delete (messages are removed via ON DELETE CASCADE).
// The SQL migration (081_add_conversations.sql) is the authoritative source for
// DDL; this schema is kept consistent with it.
type Conversation struct {
	ent.Schema
}

func (Conversation) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "conversations"},
	}
}

// Mixin: only TimeMixin (created_at + updated_at). No SoftDeleteMixin: hard delete.
func (Conversation) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (Conversation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("client_conversation_id").
			MaxLen(64).
			Comment("Client-generated identifier used for idempotent creation"),
		field.String("title").
			MaxLen(255).
			Default(""),
		field.String("model").
			MaxLen(100).
			Default(""),
		field.String("status").
			MaxLen(20).
			Default("active").
			Comment("Lifecycle status: active, archived"),
		// Dedicated list-ordering key, decoupled from updated_at (which is
		// auto-bumped on every update, including rename). Advanced only when a
		// message is appended or atomically replaced.
		// Invariant: == MAX(message.created_at), or created_at when no messages.
		field.Time("last_message_at").
			Default(time.Now).
			SchemaType(map[string]string{dialect.Postgres: "timestamptz"}),
	}
}

func (Conversation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("conversations").
			Field("user_id").
			Unique().
			Required(),
		// Match the authoritative migration: deleting a conversation cascades to
		// its messages (ON DELETE CASCADE).
		edge.To("messages", ConversationMessage.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (Conversation) Indexes() []ent.Index {
	return []ent.Index{
		// Idempotent creation per user.
		index.Fields("user_id", "client_conversation_id").Unique(),
		// Current cursor pagination ordering (last_message_at DESC, id DESC).
		index.Fields("user_id", "last_message_at", "id"),
		// Legacy ordering index, retained for rollback safety (dropped later).
		index.Fields("user_id", "updated_at", "id"),
	}
}
