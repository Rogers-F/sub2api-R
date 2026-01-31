-- Migration: Add quota_usd and used_usd columns to api_keys table
-- Purpose: Allow per-API-Key usage quota limits (soft limit)

ALTER TABLE api_keys
  ADD COLUMN quota_usd DECIMAL(20,10) NULL,
  ADD COLUMN used_usd DECIMAL(20,10) NOT NULL DEFAULT 0;

COMMENT ON COLUMN api_keys.quota_usd IS 'API Key usage quota in USD, NULL means unlimited';
COMMENT ON COLUMN api_keys.used_usd IS 'Accumulated actual cost for this API Key';
