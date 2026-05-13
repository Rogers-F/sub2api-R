ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS bedrock_thinking_signature_compat_enabled BOOLEAN NOT NULL DEFAULT FALSE;
