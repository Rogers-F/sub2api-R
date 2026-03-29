package schema

import (
	"github.com/Wei-Shaw/sub2api/ent/schema/mixins"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type PaygOrder struct {
	ent.Schema
}

func (PaygOrder) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "payg_orders"},
	}
}

func (PaygOrder) Mixin() []ent.Mixin {
	return []ent.Mixin{
		mixins.TimeMixin{},
	}
}

func (PaygOrder) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id"),
		field.String("client_sn").
			MaxLen(64).
			Unique(),
		field.String("sn").
			MaxLen(64).
			Optional().
			Nillable().
			Unique(),
		field.Float("amount_yuan").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.Int64("amount_cent"),
		field.Float("credit_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}),
		field.String("payway").
			MaxLen(32).
			Default(""),
		field.String("payway_name").
			MaxLen(64).
			Default(""),
		field.String("status").
			MaxLen(20).
			Default("PENDING"),
		field.Time("paid_at").
			Optional().
			Nillable(),
	}
}

func (PaygOrder) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("payg_orders").
			Field("user_id").
			Required().
			Unique(),
	}
}

func (PaygOrder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at"),
		index.Fields("status", "created_at"),
	}
}
