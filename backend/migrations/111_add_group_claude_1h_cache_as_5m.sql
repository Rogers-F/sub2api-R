-- Migration: 111_add_group_claude_1h_cache_as_5m
-- 分组级开关：仅 anthropic，下游未声明 ttl=1h 时把上游返回的 1h 缓存创建按 5m
-- 计费/展示（响应 usage、自身计费、用量日志一致），差额由本站承担。默认关闭。

ALTER TABLE groups
  ADD COLUMN IF NOT EXISTS claude_unrequested_1h_cache_as_5m BOOLEAN NOT NULL DEFAULT FALSE;
