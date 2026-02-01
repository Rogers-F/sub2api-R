-- Add commission_rate field to users table
-- NULL means use global setting, non-NULL means custom rate for this user
-- This rate is used as the referrer's commission rate when their referees make purchases

ALTER TABLE users ADD COLUMN IF NOT EXISTS commission_rate DECIMAL(5,4);

-- Add check constraint to ensure rate is between 0 and 1 (idempotent)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'check_commission_rate_range'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT check_commission_rate_range
            CHECK (commission_rate IS NULL OR (commission_rate >= 0 AND commission_rate <= 1));
    END IF;
END $$;

-- Add comment for documentation
COMMENT ON COLUMN users.commission_rate IS 'Custom commission rate for this user as a referrer (NULL = use global rate, 0-1 = custom rate)';
