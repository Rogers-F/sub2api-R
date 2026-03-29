CREATE TABLE IF NOT EXISTS payg_orders (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    client_sn VARCHAR(64) NOT NULL,
    sn VARCHAR(64),
    amount_yuan DECIMAL(20, 8) NOT NULL,
    amount_cent BIGINT NOT NULL,
    credit_amount DECIMAL(20, 8) NOT NULL,
    payway VARCHAR(32) NOT NULL DEFAULT '',
    payway_name VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    paid_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_payg_orders_client_sn ON payg_orders (client_sn);
CREATE UNIQUE INDEX IF NOT EXISTS uq_payg_orders_sn ON payg_orders (sn) WHERE sn IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payg_orders_user_id_created_at ON payg_orders (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_payg_orders_status_created_at ON payg_orders (status, created_at DESC);
