package mixins

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/mixin"
)

// TimeMixin provides created_at and updated_at fields compatible with the existing schema.
type TimeMixin struct {
	mixin.Schema
}

func (TimeMixin) Fields() []ent.Field {
	return timeFields(time.Now)
}

// BeijingTimeMixin provides timestamps that are always stored with Beijing location.
type BeijingTimeMixin struct {
	mixin.Schema
}

func (BeijingTimeMixin) Fields() []ent.Field {
	return timeFields(timezone.BeijingNow)
}

func timeFields(now func() time.Time) []ent.Field {
	return []ent.Field{
		field.Time("created_at").
			Immutable().
			Default(now).
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
		field.Time("updated_at").
			Default(now).
			UpdateDefault(now).
			SchemaType(map[string]string{
				dialect.Postgres: "timestamptz",
			}),
	}
}
