-- 084_newapi_user_tokens.sql
-- Per-user mapping to an externally provisioned relay token.
-- We persist ONLY the upstream token id (integer) plus lifecycle status.
-- The plaintext relay key is NEVER stored here -- it is fetched on demand
-- and held in process memory only.
--
-- status values:
--   active         -- mapping is live and usable
--   revoked        -- upstream token deleted, mapping retired
--   revoke_failed  -- upstream deletion failed, see last_error for triage

CREATE TABLE IF NOT EXISTS newapi_user_tokens (
    user_id BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    newapi_token_id INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ
);
