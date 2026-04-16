ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS force_application_json_for_non_stream BOOLEAN NOT NULL DEFAULT FALSE;
