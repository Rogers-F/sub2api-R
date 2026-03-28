# Sub2API 部署架构总览

本文档记录当前 Sub2API 的双层代理部署架构、数据流、账号路由机制及已知问题。

---

## 1. 整体架构

```
用户 (Claude Code / API 客户端)
       │
       ▼
┌─────────────────────────────┐
│  国内服务器 (阿里云华东)      │
│  Host: iZwz9goue2h7vleeplyzugZ │
│                               │
│  Nginx (宿主机, :80/:443)     │
│       │                       │
│       ▼                       │
│  Sub2API (:8080)              │
│  PostgreSQL + Redis (Docker)  │
│  账号类型: apikey             │
│  (转发到日本)                 │
└──────────────┬────────────────┘
               │ HTTP (apikey 认证)
               ▼
┌─────────────────────────────┐
│  日本服务器 (阿里云日本)      │
│  Host: iZ6we480zudo2slq904es5Z │
│  IP: 47.79.151.22             │
│                               │
│  Sub2API (:8080)              │
│  PostgreSQL + Redis (Docker)  │
│  账号类型: oauth              │
│  (连接 Anthropic API)        │
└──────────────┬────────────────┘
               │ HTTPS (OAuth, 经 SOCKS5 代理)
               ▼
┌─────────────────────────────┐
│  Anthropic API               │
│  api.anthropic.com           │
└──────────────────────────────┘
```

---

## 2. 服务器详情

### 2.1 国内服务器

| 项目 | 值 |
|------|-----|
| 主机名 | iZwz9goue2h7vleeplyzugZ |
| 部署路径 | `~/sub2api/deploy` |
| 容器 | sub2api, sub2api-postgres, sub2api-redis |
| 反向代理 | 宿主机 Nginx + Let's Encrypt SSL |
| 应用端口 | 8080 (内部) |
| 日志路径 | `/var/log/nginx/access.log` (Nginx), `docker logs sub2api` (应用) |

### 2.2 日本服务器

| 项目 | 值 |
|------|-----|
| 主机名 | iZ6we480zudo2slq904es5Z |
| IP | 47.79.151.22 |
| 部署路径 | `~/sub2api/deploy` |
| 容器 | sub2api, sub2api-postgres, sub2api-redis |
| 应用端口 | 8080 |
| 代理出口 | 各 OAuth 账号通过 SOCKS5 代理连接 Anthropic |

---

## 3. 账号路由机制

### 3.1 请求全链路

```
1. 用户发送请求 (带 API Key) → 国内 Nginx
2. Nginx → 国内 Sub2API (:8080)
3. 国内 Sub2API 认证 API Key → 找到对应 Group
4. Group 内选择可用 Account (type=apikey)
5. 用 Account 的 credentials.api_key + credentials.base_url 转发到日本
6. 日本 Sub2API 认证收到的 API Key → 找到对应 Group
7. Group 内选择可用 Account (type=oauth)
8. 用 OAuth token 经 SOCKS5 代理 → Anthropic API
9. 响应原路返回
```

### 3.2 国内侧数据结构

```
users (用户)
  └── user_subscriptions (订阅, 关联 group_id)
        └── groups (分组, 如 "4/四人车")
              └── account_groups (关联表)
                    └── accounts (type=apikey, credentials 含日本的 api_key + base_url)
```

**国内 accounts 表 credentials 示例:**

```json
{
  "api_key": "sk-xxxx",
  "base_url": "http://47.79.151.22:8080/",
  "model_mapping": {
    "claude-opus-4-6": "claude-opus-4-6",
    "claude-sonnet-4-6": "claude-sonnet-4-6",
    "claude-haiku-4-5-20251001": "claude-haiku-4-5-20251001"
  }
}
```

### 3.3 日本侧数据结构

```
api_keys (国内连过来的密钥, 关联 group_id)
  └── groups (分组, 如 "max-4")
        └── account_groups (关联表)
              └── accounts (type=oauth, Anthropic OAuth 账号)
```

### 3.4 当前映射关系 (1:1 架构)

每个国内 apikey 账号对应日本一个 API Key，每个日本 API Key 对应一个 Group，每个 Group 含一个 OAuth 账号:

| 国内账号 | 国内 Group | 日本 API Key | 日本 Group | 日本 OAuth 账号 |
|---------|-----------|-------------|-----------|---------------|
| max-4 (id=10) | 4/四人车 (id=23) | max-4 (sk-3e6adc6...) | max-4 (id=7) | id=9, JoseManhaso252 |
| max-18 (id=33) | 对应 group | max-18 (sk-6055f77...) | max-18 (id=21) | id=25, JorgeDennys745 |
| ... | ... | ... | ... | ... |

**日本侧完整账号池** (截至 2026-02-18):

| ID | 名称 | 类型 | 状态 |
|----|------|------|------|
| 1 | qq | apikey | active |
| 2 | vrhcp35443571@outlook.com | oauth | error |
| 5 | 2/4codex---DavidScott5079 | oauth | active |
| 9 | max--4----JoseManhaso252 | oauth | active |
| 10 | Max-3-2--DanielThiemel150 | oauth | active |
| 11-26 | (共 16 个 max 系列 OAuth 账号) | oauth | active |

---

## 4. 调度机制

### 4.1 账号选择流程

```
请求进入 → 确定 Group → 调度器选择账号

调度器过滤链 (gateway_service.go):
  1. isExcluded()                          → 排除列表
  2. IsSchedulable()                       → 状态检查 (active/限流/过载)
  3. isAccountAllowedForPlatform()         → 平台匹配
  4. isModelSupportedByAccountWithContext() → model_mapping 白名单
  5. IsSchedulableForModelWithContext()     → 模型维度限流
  6. isAccountSchedulableForWindowCost()    → 窗口成本限制
```

### 4.2 model_mapping 白名单机制

**关键代码**: `account.go:385-403`

- `model_mapping` 为空 → 允许所有模型
- `model_mapping` 非空 → **仅允许 mapping keys 中存在的模型** (精确匹配 + 通配符)
- `apikey` 类型账号 **不做模型名标准化** (不会自动把 `claude-haiku-4-5` 转换为 `claude-haiku-4-5-20251001`)

### 4.3 限流处理

| 上游状态码 | 处理 | 冷却时间 |
|-----------|------|---------|
| 429 (有 reset header) | SetRateLimited | 按 header 指定时间 |
| 429 (无 reset header) | SetRateLimited | **默认 5 分钟** |
| 529 | SetOverloaded | 配置项, 默认 10 分钟 |
| 401/402/403 | SetError | 永久禁用, 需手动恢复 |

**限流级联问题**: 日本返回 429/502 → 国内也标记对应账号冷却 → 全链路不可用。

### 4.4 自动恢复

- 限流冷却: 到期自动恢复 (time-based)
- Error 状态: 自动恢复服务尝试 3 次连通性测试, 成功则 ClearError()
- 手动测试连通性: 成功后调用 ClearError() 重置 Error 状态

---

## 5. Group 与用户管理

### 5.1 用户访问控制

用户通过 **订阅 (user_subscriptions)** 关联到 Group:

```sql
SELECT us.user_id, u.email, us.group_id, g.name, us.status, us.expires_at
FROM user_subscriptions us
JOIN users u ON us.user_id = u.id
JOIN groups g ON us.group_id = g.id
WHERE us.status = 'active' AND us.deleted_at IS NULL;
```

### 5.2 Group 配置要点

| 字段 | 说明 |
|------|------|
| supported_model_scopes | 支持的模型范围, 如 `["claude", "gemini_text", "gemini_image"]` |
| daily/weekly/monthly_limit_usd | 用量限额 |
| subscription_type | 订阅类型 (subscription) |
| is_exclusive | 是否独占 |
| model_routing_enabled | 是否启用模型路由 |

### 5.3 国内 Group 23 ("4/四人车") 示例

- 用户: 282568694@qq.com, jaymie9019@gmail.com, guoheboke@gmail.com, 2371834550@qq.com
- 账号: max-4 (id=10)
- 限额: 日 $80 / 周 $240 / 月 $800

---

## 6. 已知问题与解决方案

### 6.1 model_mapping 短别名缺失

**问题**: `credentials.model_mapping` 充当白名单, 但只配了完整模型名 (如 `claude-haiku-4-5-20251001`), 缺少短别名 (如 `claude-haiku-4-5`)。Claude Code 客户端使用短别名发请求, 导致 503。

**影响**: 所有使用该账号的用户发送短别名模型请求时返回 `503 No available accounts`。

**修复**: 在 model_mapping 中补充短别名:

```sql
UPDATE accounts SET credentials = jsonb_set(
  credentials, '{model_mapping}',
  credentials->'model_mapping' || '{
    "claude-haiku-4-5": "claude-haiku-4-5",
    "claude-sonnet-4-5": "claude-sonnet-4-5",
    "claude-opus-4": "claude-opus-4",
    "claude-opus-4-1": "claude-opus-4-1",
    "claude-opus-4-5": "claude-opus-4-5",
    "claude-sonnet-4": "claude-sonnet-4"
  }'::jsonb
) WHERE id = <account_id>;
```

### 6.2 日本侧 1:1 映射无 Failover

**问题**: 每个日本 Group 只有 1 个 OAuth 账号, 该账号被限流时整条链路中断。

**影响**: 单点故障, 限流期间 (默认 5 分钟) 用户完全不可用。

**解决方案**:
- **方案 A**: 给日本 Group 添加多个 OAuth 账号 (推荐)
- **方案 B**: 国内侧做多上游 failover
- **方案 C**: 合并日本 Group 为共享池 (会失去用户隔离)

### 6.3 OAuth 账号旧模型 404

**问题**: Anthropic OAuth 已下线旧模型 (如 `claude-3-5-haiku-20241022`), 请求返回 404, 日本 Sub2API 转为 502 返回国内。

**修复**: 在日本侧对应账号的 model_mapping 中映射旧模型到新模型:

```sql
UPDATE accounts SET extra = jsonb_set(
  COALESCE(extra, '{}'), '{model_mapping}', '{
    "claude-3-5-haiku-20241022": "claude-haiku-4-5-20251001",
    "claude-3-5-sonnet-20241022": "claude-sonnet-4-5-20250929"
  }'
) WHERE id = <account_id>;
```

### 6.4 限流级联

**问题**: Anthropic 429 → 日本返回错误 → 国内标记自身账号冷却 5 分钟 → 期间所有请求 503。

**缓解**: 增加日本侧 OAuth 账号池深度, 减少单账号触发限流的概率。

---

## 7. 运维速查

### 7.1 常用诊断命令

```bash
# === 国内服务器 ===

# 查看账号状态
docker exec -i $(docker ps -q -f name=postgres) psql -U postgres -d sub2api \
  -c "SELECT id, name, status, rate_limited_at, rate_limit_reset_at FROM accounts WHERE deleted_at IS NULL;"

# 查看用户订阅
docker exec -i $(docker ps -q -f name=postgres) psql -U postgres -d sub2api \
  -c "SELECT us.user_id, u.email, us.group_id, g.name, us.status
      FROM user_subscriptions us
      JOIN users u ON us.user_id = u.id
      JOIN groups g ON us.group_id = g.id
      WHERE us.deleted_at IS NULL;"

# 查看最近请求
docker exec -i $(docker ps -q -f name=postgres) psql -U postgres -d sub2api \
  -c "SELECT user_id, model, account_id, total_cost, ip_address, user_agent, created_at
      FROM usage_logs ORDER BY created_at DESC LIMIT 20;"

# 查看 503 错误
docker logs sub2api --tail 500 2>&1 | grep '503.*POST.*messages'

# 查看调度失败
docker logs sub2api --tail 500 2>&1 | grep 'no available accounts'

# Nginx 访问日志 (真实 IP)
cat /var/log/nginx/access.log | grep 'messages' | grep -v '200' | tail -20

# === 日本服务器 ===

# 查看上游错误
docker logs sub2api --tail 500 2>&1 | grep -i 'error\|404\|429\|502'

# 查看账号池
docker exec -i $(docker ps -q -f name=postgres) psql -U postgres -d sub2api \
  -c "SELECT id, name, type, status, rate_limited_at FROM accounts WHERE deleted_at IS NULL ORDER BY id;"

# 查看 API Key 到 Group 的映射
docker exec -i $(docker ps -q -f name=postgres) psql -U postgres -d sub2api \
  -c "SELECT ak.id, ak.name, ak.group_id, g.name as group_name
      FROM api_keys ak JOIN groups g ON ak.group_id = g.id
      WHERE ak.deleted_at IS NULL ORDER BY ak.id;"

# 查看 Group 到 Account 的映射
docker exec -i $(docker ps -q -f name=postgres) psql -U postgres -d sub2api \
  -c "SELECT ag.group_id, g.name as group_name, ag.account_id, a.name as account_name
      FROM account_groups ag
      JOIN groups g ON ag.group_id = g.id
      JOIN accounts a ON ag.account_id = a.id
      ORDER BY ag.group_id;"
```

### 7.2 关键数据库表

| 表名 | 用途 |
|------|------|
| accounts | 上游账号 (国内=apikey转发, 日本=OAuth) |
| account_groups | 账号与分组关联 |
| groups | 分组配置 (模型范围、限额) |
| api_keys | 客户端认证密钥 (日本侧) |
| users | 用户 |
| user_subscriptions | 用户订阅 (关联 group) |
| usage_logs | 请求日志 (含 user_id, model, cost, ip, user_agent) |

---

## 8. 架构图 (数据库关系)

```
国内服务器:
  users ──(1:N)── user_subscriptions ──(N:1)── groups ──(N:M)── accounts
                                                                  │
                                               credentials.api_key + base_url
                                                                  │
                                                                  ▼
日本服务器:
  api_keys ──(N:1)── groups ──(N:M)── accounts (OAuth)
                                          │
                                    OAuth token + SOCKS5 proxy
                                          │
                                          ▼
                                    Anthropic API
```

---

*最后更新: 2026-02-19*
