-- 083_chat_phase2_index_notx.sql
-- Build the conversations.last_message_at ordering index outside a
-- transaction (the _notx suffix is required for CONCURRENTLY). Drop then
-- create so a half-built invalid index left by a failed prior run is
-- self-healed on the next attempt. The legacy updated_at index is kept for a
-- safe downgrade path and removed in a later release once this is stable.

DROP INDEX CONCURRENTLY IF EXISTS idx_conversations_user_lastmsg_id;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_conversations_user_lastmsg_id
    ON conversations (user_id, last_message_at DESC, id DESC);
