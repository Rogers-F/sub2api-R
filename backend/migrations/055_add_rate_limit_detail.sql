ALTER TABLE accounts ADD COLUMN IF NOT EXISTS rate_limit_window_type VARCHAR(20);
ALTER TABLE accounts ADD COLUMN IF NOT EXISTS rate_limit_detail TEXT;
