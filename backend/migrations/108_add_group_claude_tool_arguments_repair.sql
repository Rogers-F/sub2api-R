ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS claude_tool_arguments_repair_enabled BOOLEAN NOT NULL DEFAULT FALSE;
