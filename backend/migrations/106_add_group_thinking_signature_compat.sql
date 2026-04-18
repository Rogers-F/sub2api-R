ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS thinking_signature_compat_enabled BOOLEAN NOT NULL DEFAULT FALSE;
