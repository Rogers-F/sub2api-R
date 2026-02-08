-- Bridge migration: align fork's announcements schema with upstream.
-- Safe to run on both old fork databases and fresh installs (uses IF NOT EXISTS).

-- Add missing columns to announcements table
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS targeting JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS starts_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS ends_at TIMESTAMPTZ DEFAULT NULL;
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS created_by BIGINT DEFAULT NULL REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE announcements ADD COLUMN IF NOT EXISTS updated_by BIGINT DEFAULT NULL REFERENCES users(id) ON DELETE SET NULL;

-- Add missing column to announcement_reads table
ALTER TABLE announcement_reads ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Ensure all upstream indexes exist
CREATE INDEX IF NOT EXISTS idx_announcements_status ON announcements(status);
CREATE INDEX IF NOT EXISTS idx_announcements_starts_at ON announcements(starts_at);
CREATE INDEX IF NOT EXISTS idx_announcements_ends_at ON announcements(ends_at);
CREATE INDEX IF NOT EXISTS idx_announcements_created_at ON announcements(created_at);
CREATE INDEX IF NOT EXISTS idx_announcement_reads_announcement_id ON announcement_reads(announcement_id);
CREATE INDEX IF NOT EXISTS idx_announcement_reads_user_id ON announcement_reads(user_id);
CREATE INDEX IF NOT EXISTS idx_announcement_reads_read_at ON announcement_reads(read_at);
