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

// ReferralReward holds the schema definition for the ReferralReward entity.
//
// 邀请奖励记录：记录邀请系统产生的所有奖励
// - register: 注册奖励，邀请人和被邀请人各获得奖励
// - commission: 返利奖励，被邀请人消费时邀请人获得返利
type ReferralReward struct {
	ent.Schema
}

func (ReferralReward) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "referral_rewards"},
	}
}

func (ReferralReward) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("referrer_id").
			Comment("邀请人ID"),
		field.Int64("referee_id").
			Comment("被邀请人ID"),
		field.String("reward_type").
			MaxLen(20).
			Comment("奖励类型：register=注册奖励，commission=返利"),
		field.String("source_type").
			MaxLen(20).
			Optional().
			Nillable().
			Comment("来源类型：redeem_code=兑换码"),
		field.Int64("source_id").
			Optional().
			Nillable().
			Comment("来源ID"),
		field.Float("source_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Default(0).
			Comment("来源金额"),
		field.Float("reward_amount").
			SchemaType(map[string]string{dialect.Postgres: "decimal(20,8)"}).
			Comment("奖励金额"),
		field.Float("commission_rate").
			SchemaType(map[string]string{dialect.Postgres: "decimal(5,4)"}).
			Optional().
			Nillable().
			Comment("返利比例（仅 commission 类型）"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
	}
}

func (ReferralReward) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("referrer", User.Type).
			Ref("referral_rewards_given").
			Field("referrer_id").
			Required().
			Unique(),
		edge.From("referee", User.Type).
			Ref("referral_rewards_received").
			Field("referee_id").
			Required().
			Unique(),
	}
}

func (ReferralReward) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("referrer_id"),
		index.Fields("referee_id"),
		index.Fields("reward_type"),
		index.Fields("created_at"),
	}
}
