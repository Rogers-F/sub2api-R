-- 083_chat_phase2_index_notx.sql
-- Concurrent index build for conversations.last_message_at ordering.
--
-- Runs OUTSIDE a transaction (the _notx.sql suffix): CREATE INDEX CONCURRENTLY
-- cannot run inside a transaction block.
--
-- DROP CONCURRENTLY before CREATE CONCURRENTLY: a concurrent index build that
-- fails midway can leave an INVALID index of the same name behind; a plain
-- CREATE ... IF NOT EXISTS would then skip it and leave the table effectively
-- un-indexed. Dropping first self-heals any such leftover on a re-run.
--
-- The old idx_conversations_user_updated_id index is intentionally kept for
-- rollback safety and removed in a later release once this is stable.

DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_user_lastmsg_id;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversations_user_lastmsg_id
    ON conversations (user_id, last_message_at DESC, id DESC);
