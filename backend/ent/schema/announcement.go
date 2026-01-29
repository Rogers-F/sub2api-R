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

// Announcement holds the schema definition for the Announcement entity.
//
// 系统公告：管理员发布的公告信息
// 用户登录时会检查是否有未读公告并弹窗显示
type Announcement struct {
	ent.Schema
}

func (Announcement) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "announcements"},
	}
}

func (Announcement) Fields() []ent.Field {
	return []ent.Field{
		field.String("title").
			MaxLen(255).
			NotEmpty().
			Comment("公告标题"),
		field.String("content").
			Comment("公告内容"),
		field.String("content_type").
			MaxLen(20).
			Default("markdown").
			Comment("内容类型：markdown/html/url"),
		field.Int("priority").
			Default(0).
			Comment("排序优先级，越大越靠前"),
		field.String("status").
			MaxLen(20).
			Default("active").
			Comment("状态：active=启用，inactive=禁用"),
		field.Time("published_at").
			Optional().
			Nillable().
			Comment("发布时间，NULL表示立即发布"),
		field.Time("expires_at").
			Optional().
			Nillable().
			Comment("过期时间，NULL表示永不过期"),
		field.Time("created_at").
			Default(time.Now).
			Immutable().
			Comment("创建时间"),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now).
			Comment("更新时间"),
	}
}

func (Announcement) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("reads", AnnouncementRead.Type),
	}
}

func (Announcement) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
		index.Fields("priority"),
		index.Fields("published_at"),
		index.Fields("expires_at"),
	}
}
