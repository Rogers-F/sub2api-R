package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AnnouncementRead holds the schema definition for the AnnouncementRead entity.
//
// 公告已读记录：记录用户已读的公告
type AnnouncementRead struct {
	ent.Schema
}

func (AnnouncementRead) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "announcement_reads"},
	}
}

func (AnnouncementRead) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("user_id").
			Comment("用户ID"),
		field.Int64("announcement_id").
			Comment("公告ID"),
		field.Time("read_at").
			Default(time.Now).
			Immutable().
			Comment("阅读时间"),
	}
}

func (AnnouncementRead) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("announcement_reads").
			Field("user_id").
			Required().
			Unique(),
		edge.From("announcement", Announcement.Type).
			Ref("reads").
			Field("announcement_id").
			Required().
			Unique(),
	}
}

func (AnnouncementRead) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("announcement_id"),
		index.Fields("user_id", "announcement_id").Unique(),
	}
}
