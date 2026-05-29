-- Migration: 109_channel_monitor_tables
-- 渠道监控功能建表（最终态合并：等价于上游 125 建表 + 126 聚合/watermark + 127 去软删
-- + 128 请求模板/快照字段 + 138 OpenAI api_mode）。
--
-- 说明：
--   - 本 fork 生产库从未创建过监控表，故直接建“最终态”而非 create-then-alter 串联，
--     减少上游历史迁移噪音。
--   - 仍保留防御式 IF NOT EXISTS / IF EXISTS / DO 块判定，兼容曾手工试装过上游迁移的
--     dev/staging 库：缺列补列、残留 deleted_at 去除、缺约束/外键补齐，全部幂等。
--   - 无 CONCURRENTLY，整文件在单事务内执行（见 migrations_runner）。

-- ===========================================================================
-- 1) 请求模板表（被 channel_monitors.template_id 外键引用，需先建）
-- ===========================================================================
CREATE TABLE IF NOT EXISTS channel_monitor_request_templates (
    id            BIGSERIAL    PRIMARY KEY,
    name          VARCHAR(100) NOT NULL,
    provider      VARCHAR(20)  NOT NULL,
    api_mode      VARCHAR(32)  NOT NULL DEFAULT 'chat_completions',
    description   VARCHAR(500) NOT NULL DEFAULT '',
    extra_headers JSONB        NOT NULL DEFAULT '{}'::jsonb,
    body_override_mode VARCHAR(10) NOT NULL DEFAULT 'off',
    body_override JSONB        NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT channel_monitor_request_templates_provider_check
        CHECK (provider IN ('openai', 'anthropic', 'gemini')),
    CONSTRAINT channel_monitor_request_templates_body_mode_check
        CHECK (body_override_mode IN ('off', 'merge', 'replace'))
);

-- 防御：旧库可能缺 api_mode 列与其约束
ALTER TABLE channel_monitor_request_templates
    ADD COLUMN IF NOT EXISTS api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'channel_monitor_request_templates_api_mode_check'
          AND table_name = 'channel_monitor_request_templates'
    ) THEN
        ALTER TABLE channel_monitor_request_templates
            ADD CONSTRAINT channel_monitor_request_templates_api_mode_check
            CHECK (api_mode IN ('chat_completions', 'responses'));
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS channel_monitor_request_templates_provider_name
    ON channel_monitor_request_templates (provider, name);
CREATE INDEX IF NOT EXISTS idx_channel_monitor_templates_provider_api_mode
    ON channel_monitor_request_templates (provider, api_mode);

-- ===========================================================================
-- 2) 渠道监控主表
-- ===========================================================================
CREATE TABLE IF NOT EXISTS channel_monitors (
    id                 BIGSERIAL    PRIMARY KEY,
    name               VARCHAR(100) NOT NULL,
    provider           VARCHAR(20)  NOT NULL,    -- openai / anthropic / gemini
    api_mode           VARCHAR(32)  NOT NULL DEFAULT 'chat_completions',
    endpoint           VARCHAR(500) NOT NULL,    -- base origin
    api_key_encrypted  TEXT         NOT NULL,    -- AES-256-GCM (base64)
    primary_model      VARCHAR(200) NOT NULL,
    extra_models       JSONB        NOT NULL DEFAULT '[]'::jsonb,
    group_name         VARCHAR(100) NOT NULL DEFAULT '',
    enabled            BOOLEAN      NOT NULL DEFAULT TRUE,
    interval_seconds   INT          NOT NULL,
    last_checked_at    TIMESTAMPTZ,
    created_by         BIGINT       NOT NULL,
    template_id        BIGINT       NULL,
    extra_headers      JSONB        NOT NULL DEFAULT '{}'::jsonb,
    body_override_mode VARCHAR(10)  NOT NULL DEFAULT 'off',
    body_override      JSONB        NULL,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT channel_monitors_provider_check CHECK (provider IN ('openai', 'anthropic', 'gemini')),
    CONSTRAINT channel_monitors_interval_check CHECK (interval_seconds BETWEEN 15 AND 3600)
);

-- 防御：旧库（仅跑过上游 125）可能缺以下列
ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';
ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS template_id BIGINT NULL;
ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS extra_headers JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS body_override_mode VARCHAR(10) NOT NULL DEFAULT 'off';
ALTER TABLE channel_monitors
    ADD COLUMN IF NOT EXISTS body_override JSONB NULL;

-- 约束 + 外键（DO 块 IF NOT EXISTS，幂等）
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'channel_monitors_body_mode_check'
          AND table_name = 'channel_monitors'
    ) THEN
        ALTER TABLE channel_monitors
            ADD CONSTRAINT channel_monitors_body_mode_check
            CHECK (body_override_mode IN ('off', 'merge', 'replace'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'channel_monitors_api_mode_check'
          AND table_name = 'channel_monitors'
    ) THEN
        ALTER TABLE channel_monitors
            ADD CONSTRAINT channel_monitors_api_mode_check
            CHECK (api_mode IN ('chat_completions', 'responses'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'channel_monitors_template_id_fkey'
          AND table_name = 'channel_monitors'
    ) THEN
        ALTER TABLE channel_monitors
            ADD CONSTRAINT channel_monitors_template_id_fkey
            FOREIGN KEY (template_id)
            REFERENCES channel_monitor_request_templates (id)
            ON DELETE SET NULL;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_enabled_last_checked
    ON channel_monitors (enabled, last_checked_at);
CREATE INDEX IF NOT EXISTS idx_channel_monitors_provider
    ON channel_monitors (provider);
CREATE INDEX IF NOT EXISTS idx_channel_monitors_group_name
    ON channel_monitors (group_name);
CREATE INDEX IF NOT EXISTS idx_channel_monitors_provider_api_mode
    ON channel_monitors (provider, api_mode);
CREATE INDEX IF NOT EXISTS idx_channel_monitors_template_id
    ON channel_monitors (template_id)
    WHERE template_id IS NOT NULL;

-- ===========================================================================
-- 3) 检测历史明细表（最终态：无 deleted_at，物理删由 ops cleanup 分批跑）
-- ===========================================================================
CREATE TABLE IF NOT EXISTS channel_monitor_histories (
    id              BIGSERIAL PRIMARY KEY,
    monitor_id      BIGINT      NOT NULL REFERENCES channel_monitors(id) ON DELETE CASCADE,
    model           VARCHAR(200) NOT NULL,
    status          VARCHAR(20)  NOT NULL,
    latency_ms      INT,
    ping_latency_ms INT,
    message         VARCHAR(500) NOT NULL DEFAULT '',
    checked_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT channel_monitor_histories_status_check
        CHECK (status IN ('operational', 'degraded', 'failed', 'error'))
);

-- 防御：旧库（跑过上游 126 软删）可能残留 deleted_at 列与索引
DROP INDEX IF EXISTS idx_channel_monitor_histories_deleted_at;
ALTER TABLE channel_monitor_histories DROP COLUMN IF EXISTS deleted_at;

CREATE INDEX IF NOT EXISTS idx_channel_monitor_histories_monitor_model_checked
    ON channel_monitor_histories (monitor_id, model, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_channel_monitor_histories_checked_at
    ON channel_monitor_histories (checked_at);

-- ===========================================================================
-- 4) 日聚合表（最终态：无 deleted_at）
-- ===========================================================================
CREATE TABLE IF NOT EXISTS channel_monitor_daily_rollups (
    id                    BIGSERIAL PRIMARY KEY,
    monitor_id            BIGINT       NOT NULL REFERENCES channel_monitors(id) ON DELETE CASCADE,
    model                 VARCHAR(200) NOT NULL,
    bucket_date           DATE         NOT NULL,
    total_checks          INT          NOT NULL DEFAULT 0,
    ok_count              INT          NOT NULL DEFAULT 0,
    operational_count     INT          NOT NULL DEFAULT 0,
    degraded_count        INT          NOT NULL DEFAULT 0,
    failed_count          INT          NOT NULL DEFAULT 0,
    error_count           INT          NOT NULL DEFAULT 0,
    sum_latency_ms        BIGINT       NOT NULL DEFAULT 0,
    count_latency         INT          NOT NULL DEFAULT 0,
    sum_ping_latency_ms   BIGINT       NOT NULL DEFAULT 0,
    count_ping_latency    INT          NOT NULL DEFAULT 0,
    computed_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- 防御：旧库可能残留 deleted_at 列与索引
DROP INDEX IF EXISTS idx_channel_monitor_daily_rollups_deleted_at;
ALTER TABLE channel_monitor_daily_rollups DROP COLUMN IF EXISTS deleted_at;

CREATE UNIQUE INDEX IF NOT EXISTS idx_channel_monitor_daily_rollups_unique
    ON channel_monitor_daily_rollups (monitor_id, model, bucket_date);
CREATE INDEX IF NOT EXISTS idx_channel_monitor_daily_rollups_bucket
    ON channel_monitor_daily_rollups (bucket_date);

-- ===========================================================================
-- 5) 聚合 watermark 单行表（id=1）
-- ===========================================================================
CREATE TABLE IF NOT EXISTS channel_monitor_aggregation_watermark (
    id                   INT          PRIMARY KEY DEFAULT 1,
    last_aggregated_date DATE,
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT channel_monitor_aggregation_watermark_singleton CHECK (id = 1)
);

INSERT INTO channel_monitor_aggregation_watermark (id, last_aggregated_date, updated_at)
VALUES (1, NULL, NOW())
ON CONFLICT (id) DO NOTHING;
