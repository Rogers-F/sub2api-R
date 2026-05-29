-- 082_chat_phase2.sql
-- Chat phase 2: add conversations.last_message_at as the dedicated list
-- ordering key, decoupled from updated_at. updated_at is auto-bumped on every
-- update (including rename) and would otherwise re-order the list; the new
-- column is advanced only when a message is appended or atomically replaced.
-- The matching index is added in 083 (non-transactional). The legacy
-- (user_id, updated_at, id) index is kept for a safe downgrade path.
--
-- Invariant: last_message_at equals MAX(conversation_messages.created_at),
-- or created_at when the conversation has no messages.
--
-- Runs inside a transaction (the default). Bound lock/statement time so a
-- stuck lock fails fast instead of stalling startup.
SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '60s';

ALTER TABLE conversations ADD COLUMN IF NOT EXISTS last_message_at TIMESTAMPTZ;

UPDATE conversations c
SET last_message_at = COALESCE(
        (SELECT MAX(m.created_at) FROM conversation_messages m WHERE m.conversation_id = c.id),
        c.created_at)
WHERE last_message_at IS NULL;

ALTER TABLE conversations ALTER COLUMN last_message_at SET DEFAULT NOW();
ALTER TABLE conversations ALTER COLUMN last_message_at SET NOT NULL;
