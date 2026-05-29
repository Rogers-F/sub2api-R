-- Migration: 110_seed_channel_monitor_templates
-- 内置渠道监控请求模板（合并上游 129 Claude Code 伪装 + 139 OpenAI 模板组）。
--
-- ON CONFLICT (provider, name) DO NOTHING：已部署/重复执行不覆盖运维已编辑的模板。
-- 用户可自行编辑覆盖此 seed；上游升级新模板时另起 migration，不动用户旧模板。

-- 1) Anthropic：Claude Code 伪装
INSERT INTO channel_monitor_request_templates (
    name, provider, description, extra_headers, body_override_mode, body_override
)
VALUES (
    'Claude Code 伪装',
    'anthropic',
    '完整模拟 Claude Code 2.1.114 客户端：UA + anthropic-beta + system + metadata.user_id 全部对齐，绕过 Anthropic 上游 ''Claude Code only'' 限制（如 Max 套餐）。',
    '{
        "User-Agent": "claude-cli/2.1.114 (external, sdk-cli)",
        "X-App": "cli",
        "anthropic-version": "2023-06-01",
        "anthropic-beta": "claude-code-20250219,interleaved-thinking-2025-05-14,context-management-2025-06-27,prompt-caching-scope-2026-01-05,advisor-tool-2026-03-01",
        "anthropic-dangerous-direct-browser-access": "true"
    }'::jsonb,
    'merge',
    '{
        "system": [
            {
                "type": "text",
                "text": "You are Claude Code, Anthropic''s official CLI for Claude."
            }
        ],
        "metadata": {
            "user_id": "user_0000000000000000000000000000000000000000000000000000000000000000_account_00000000-0000-0000-0000-000000000000_session_00000000-0000-0000-0000-000000000000"
        }
    }'::jsonb
)
ON CONFLICT (provider, name) DO NOTHING;

-- 2) OpenAI：Chat Completions / Responses 模板组
INSERT INTO channel_monitor_request_templates (
    name, provider, api_mode, description, extra_headers, body_override_mode, body_override
)
VALUES
(
    'OpenAI Compatible 默认检测',
    'openai',
    'chat_completions',
    '适用于大多数 OpenAI-compatible 上游：POST /v1/chat/completions，后端自动生成 messages 数学 challenge。',
    '{}'::jsonb,
    'off',
    NULL
),
(
    'OpenAI Compatible 低 token 检测',
    'openai',
    'chat_completions',
    '仍走 /v1/chat/completions，仅把 max_tokens 调低；model/messages/stream 由后端保护，避免误伤 challenge。',
    '{}'::jsonb,
    'merge',
    '{"max_tokens": 20}'::jsonb
),
(
    'OpenAI Responses / 本站自检',
    'openai',
    'responses',
    '适用于本站或原生 Responses API：POST /v1/responses，默认 payload 自动带 instructions 与 input，避免 Instructions are required。',
    '{}'::jsonb,
    'off',
    NULL
),
(
    'OpenAI Responses 低 token 检测',
    'openai',
    'responses',
    '仍走 /v1/responses，仅把 max_output_tokens 调低；instructions/input/model/stream 由后端保护。',
    '{}'::jsonb,
    'merge',
    '{"max_output_tokens": 20}'::jsonb
)
ON CONFLICT (provider, name) DO NOTHING;
