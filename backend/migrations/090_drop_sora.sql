-- Migration: 090_drop_sora
-- Remove all Sora-related database objects.
-- Drops tables: sora_tasks, sora_generations, sora_accounts
-- Drops columns from: groups, users, usage_logs

DROP TABLE IF EXISTS sora_tasks;
DROP TABLE IF EXISTS sora_generations;
DROP TABLE IF EXISTS sora_accounts;

ALTER TABLE groups
    DROP COLUMN IF EXISTS sora_image_price_360,
    DROP COLUMN IF EXISTS sora_image_price_540,
    DROP COLUMN IF EXISTS sora_video_price_per_request,
    DROP COLUMN IF EXISTS sora_video_price_per_request_hd,
    DROP COLUMN IF EXISTS sora_storage_quota_bytes;

ALTER TABLE users
    DROP COLUMN IF EXISTS sora_storage_quota_bytes,
    DROP COLUMN IF EXISTS sora_storage_used_bytes;

ALTER TABLE usage_logs
    DROP COLUMN IF EXISTS media_type;
