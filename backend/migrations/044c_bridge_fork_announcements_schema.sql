-- Bridge migration: align fork's announcements schema with upstream.
-- Only runs ALTER TABLE if the table already exists (fork database).
-- On fresh installs, the table doesn't exist yet so this is a no-op;
-- upstream's 045_add_announcements.sql will create it with the correct schema.

DO $$ BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'announcements') THEN
    ALTER TABLE announcements ADD COLUMN IF NOT EXISTS targeting JSONB NOT NULL DEFAULT '{}'::jsonb;
    ALTER TABLE announcements ADD COLUMN IF NOT EXISTS starts_at TIMESTAMPTZ DEFAULT NULL;
    ALTER TABLE announcements ADD COLUMN IF NOT EXISTS ends_at TIMESTAMPTZ DEFAULT NULL;
    ALTER TABLE announcements ADD COLUMN IF NOT EXISTS created_by BIGINT DEFAULT NULL REFERENCES users(id) ON DELETE SET NULL;
    ALTER TABLE announcements ADD COLUMN IF NOT EXISTS updated_by BIGINT DEFAULT NULL REFERENCES users(id) ON DELETE SET NULL;
  END IF;

  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'announcement_reads') THEN
    ALTER TABLE announcement_reads ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
  END IF;
END $$;
