-- 082_chat_phase2.sql
-- Chat phase 2: introduce conversations.last_message_at as the dedicated list
-- ordering key, decoupled from updated_at.
--
-- Why: updated_at is auto-bumped on every row update (including title rename),
-- which incorrectly re-orders the conversation list. last_message_at is only
-- advanced when a message is appended (or atomically replaced), so renaming a
-- conversation no longer moves it to the top.
--
-- Invariant: last_message_at == MAX(conversation_messages.created_at) for the
-- conversation; for a conversation with no messages it equals created_at.
--
-- This migration runs inside a transaction (the default for non *_notx.sql
-- files). The matching index is created concurrently in
-- 083_chat_phase2_index_notx.sql. The old (user_id, updated_at, id) index is
-- intentionally retained for rollback safety and dropped in a later release.

-- Bound how long the table-rewriting / lock-taking steps may block, so a stuck
-- lock fails the migration fast instead of stalling startup.
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

-- Add nullable first, backfill, then set default + not-null. This avoids a full
-- table rewrite driven by a volatile default (NOW()).
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMPTZ;

UPDATE conversations c
SET last_message_at = COALESCE(
        (SELECT MAX(m.created_at) FROM conversation_messages m WHERE m.conversation_id = c.id),
        c.created_at)
WHERE last_message_at IS NULL;

ALTER TABLE conversations ALTER COLUMN last_message_at SET DEFAULT NOW();
ALTER TABLE conversations ALTER COLUMN last_message_at SET NOT NULL;
