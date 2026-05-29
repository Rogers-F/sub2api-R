-- 081_add_conversations.sql
-- Persisted multi-conversation chat: conversations + conversation_messages.
-- Both tables are HARD-DELETE (no soft delete). Messages are removed via
-- ON DELETE CASCADE when the parent conversation is deleted.
--
-- Token columns (reported_input_tokens / reported_output_tokens) are
-- client-reported, display-only values. They are NEVER used for billing.

CREATE TABLE IF NOT EXISTS conversations (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    client_conversation_id VARCHAR(64) NOT NULL,
    title VARCHAR(255) NOT NULL DEFAULT '',
    model VARCHAR(100) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_conversations_user_client UNIQUE (user_id, client_conversation_id)
);

CREATE INDEX IF NOT EXISTS idx_conversations_user_updated_id
    ON conversations (user_id, updated_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS conversation_messages (
    id BIGSERIAL PRIMARY KEY,
    conversation_id BIGINT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id),
    role VARCHAR(16) NOT NULL CHECK (role IN ('user', 'assistant')),
    content TEXT NOT NULL DEFAULT '',
    model VARCHAR(100) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL DEFAULT 'complete' CHECK (status IN ('complete', 'error')),
    reported_input_tokens INT NULL CHECK (reported_input_tokens IS NULL OR reported_input_tokens >= 0),
    reported_output_tokens INT NULL CHECK (reported_output_tokens IS NULL OR reported_output_tokens >= 0),
    client_message_id VARCHAR(64) NOT NULL,
    gateway_request_id VARCHAR(128) NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- User messages must always be complete.
    CONSTRAINT chk_conversation_messages_user_complete CHECK (role <> 'user' OR status = 'complete'),
    -- A complete message must carry content.
    CONSTRAINT chk_conversation_messages_complete_content CHECK (status <> 'complete' OR octet_length(content) > 0),
    -- 512KB content cap.
    CONSTRAINT chk_conversation_messages_content_size CHECK (octet_length(content) <= 524288),
    CONSTRAINT uq_conversation_messages_client UNIQUE (conversation_id, client_message_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_messages_user_conv_id
    ON conversation_messages (user_id, conversation_id, id);
