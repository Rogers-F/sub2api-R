-- 109_add_enterprises.sql
-- Adds enterprise management as an independent admin grouping layer for accounts.

CREATE TABLE IF NOT EXISTS enterprises (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    notes TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS enterprises_status_idx
    ON enterprises(status)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS enterprises_deleted_at_idx
    ON enterprises(deleted_at);

CREATE UNIQUE INDEX IF NOT EXISTS enterprises_name_unique_active
    ON enterprises(LOWER(name))
    WHERE deleted_at IS NULL;

ALTER TABLE accounts
    ADD COLUMN IF NOT EXISTS enterprise_id BIGINT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'accounts_enterprise_id_fkey'
    ) THEN
        ALTER TABLE accounts
            ADD CONSTRAINT accounts_enterprise_id_fkey
            FOREIGN KEY (enterprise_id)
            REFERENCES enterprises(id)
            ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS accounts_enterprise_id_idx
    ON accounts(enterprise_id)
    WHERE deleted_at IS NULL;
