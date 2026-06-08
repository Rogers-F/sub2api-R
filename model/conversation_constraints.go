package model

import "gorm.io/gorm"

// migrateConversationConstraints adds the CHECK constraints and the ON DELETE
// CASCADE foreign key that GORM AutoMigrate does not generate from struct tags.
//
// It is Postgres-only and idempotent: each statement is guarded against
// pg_constraint so re-running migrateDB is safe. SQLite (unit tests) skips this
// entirely and relies on the service-layer validation as the safety net.
//
// The unique indexes (uq_conversations_user_client, uq_conversation_messages_client)
// are created by AutoMigrate from the struct uniqueIndex tags, not here.
//
// Only the conversation_id -> conversations(id) cascade FK is enforced (it backs
// the cascade delete of messages). user_id is scoped in every query at the app
// layer; no FK to users is added, to avoid coupling to that table's migration.
func migrateConversationConstraints(db *gorm.DB) error {
	stmts := []string{
		// conversations.status enum
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_conversations_status') THEN
				ALTER TABLE conversations ADD CONSTRAINT chk_conversations_status
					CHECK (status IN ('active', 'archived'));
			END IF;
		END $$;`,
		// conversation_messages.role enum
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_conversation_messages_role') THEN
				ALTER TABLE conversation_messages ADD CONSTRAINT chk_conversation_messages_role
					CHECK (role IN ('user', 'assistant'));
			END IF;
		END $$;`,
		// conversation_messages.status enum
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_conversation_messages_status') THEN
				ALTER TABLE conversation_messages ADD CONSTRAINT chk_conversation_messages_status
					CHECK (status IN ('complete', 'error'));
			END IF;
		END $$;`,
		// user messages must be complete
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_conversation_messages_user_complete') THEN
				ALTER TABLE conversation_messages ADD CONSTRAINT chk_conversation_messages_user_complete
					CHECK (role <> 'user' OR status = 'complete');
			END IF;
		END $$;`,
		// a complete message must carry content
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_conversation_messages_complete_content') THEN
				ALTER TABLE conversation_messages ADD CONSTRAINT chk_conversation_messages_complete_content
					CHECK (status <> 'complete' OR octet_length(content) > 0);
			END IF;
		END $$;`,
		// 512KB content cap
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_conversation_messages_content_size') THEN
				ALTER TABLE conversation_messages ADD CONSTRAINT chk_conversation_messages_content_size
					CHECK (octet_length(content) <= 524288);
			END IF;
		END $$;`,
		// reported tokens are display-only and non-negative when present
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_conversation_messages_in_tokens') THEN
				ALTER TABLE conversation_messages ADD CONSTRAINT chk_conversation_messages_in_tokens
					CHECK (reported_input_tokens IS NULL OR reported_input_tokens >= 0);
			END IF;
		END $$;`,
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_conversation_messages_out_tokens') THEN
				ALTER TABLE conversation_messages ADD CONSTRAINT chk_conversation_messages_out_tokens
					CHECK (reported_output_tokens IS NULL OR reported_output_tokens >= 0);
			END IF;
		END $$;`,
		// cascade delete of messages when the parent conversation is removed
		`DO $$ BEGIN
			IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'fk_conversation_messages_conversation') THEN
				ALTER TABLE conversation_messages ADD CONSTRAINT fk_conversation_messages_conversation
					FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE;
			END IF;
		END $$;`,
	}
	for _, s := range stmts {
		if err := db.Exec(s).Error; err != nil {
			return err
		}
	}
	return nil
}
