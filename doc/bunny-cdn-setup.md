# Bunny.net CDN 加速接入教程 — xingsuancode.com

> **目标**：通过 Bunny.net CDN 加速中国用户访问日本服务器 `xingsuancode.com`，同时确保 SSE 流式 API 正常工作。
>
> **适用场景**：Sub2API 双层代理架构，国内服务器 -> 日本服务器（经 CDN 加速）-> Anthropic/OpenAI
>
> **支付方式**：Visa / MasterCard / PayPal（首次注册赠送 14 天免费试用 + 1TB 流量）

---

## 风险提示（请先阅读）

Bunny.net **没有官方 SSE 流式支持文档**。以下结论基于调研推断：

| 项目 | 状态 | 说明 |
|------|------|------|
| SSE 流式透传 | 理论可行，需实测 | Bunny 底层是 Nginx，Sub2API 已发送 `X-Accel-Buffering: no` 头 |
| 长连接超时 | 未知 | Bunny 未公开 `proxy_read_timeout`，Opus 长对话（>60s）可能被切断 |
| Request Coalescing | **必须关闭** | 不关会导致多用户数据串联（安全问题） |
| 生产验证 | 无 | 不同于 Peekabo（88code 已生产验证），Bunny 无 AI API 代理先例 |

**建议策略**：利用 14 天免费试用测试，不满意可 5 分钟回滚。

---

## 架构变更说明

```
变更前（直连）:
  用户 -> cn.xingsuancode.com（国内）-> 47.79.151.22:8080（日本直连）-> Anthropic

变更后（CDN 加速）:
  用户 -> cn.xingsuancode.com（国内）-> xingsuancode.com（Bunny CDN）-> 47.79.151.22:443（日本源站）-> Anthropic
                                              |
                                     Bunny 东京/香港节点加速
```

**预期效果**：中国 -> 日本延迟从 60-120ms 降至约 70-100ms（Bunny 东京/香港节点），SSE 流式响应更稳定（待验证）。

---

## 第 1 步：注册 Bunny.net 账号

1. 打开 **https://bunny.net** ，点击右上角 **Start Free Trial**
2. 填写邮箱、密码，完成注册
3. 验证邮箱（查收激活邮件）
4. 登录后进入 Dashboard

> **免费试用**：14 天 + 1TB 流量，无需预先绑卡。试用结束后按量计费。

---

## 第 2 步：绑定支付方式

1. 左侧菜单 -> **Billing**
2. 点击 **Add Payment Method**
3. 选择 **Credit Card**（Visa / MasterCard）或 **PayPal**
4. 填写卡号信息，保存

> **计费模式**：按流量计费，亚太区 $0.03/GB。预充值制，最低充值 $10。
>
> **费用估算**（5 万用户）：
> - API 请求流量（纯文本为主）：约 50-200 GB/月 -> $1.5-6/月
> - 网页流量：约 10-50 GB/月 -> $0.3-1.5/月
> - **合计约 $2-8/月**

---

## 第 3 步：创建 Pull Zone

Pull Zone 是 Bunny.net 的核心概念，相当于一个 CDN 加速配置。

1. 左侧菜单 -> **CDN** -> **Pull Zones**
2. 点击 **Add Pull Zone**
3. 填写配置：

| 字段 | 值 | 说明 |
|------|-----|------|
| **Pull Zone Name** | `xingsuancode` | 自定义名称，仅用于管理标识 |
| **Origin Type** | `Origin URL` | 从源站拉取内容 |
| **Origin URL** | `https://47.79.151.22` | 日本服务器 IP（HTTPS） |

4. **Pricing Tier** 选择 **Standard**（亚太区覆盖东京、香港等节点）
5. 点击 **Add Pull Zone** 完成创建

---

## 第 4 步：配置 Pull Zone（关键）

创建完成后进入 Pull Zone 设置页面，依次配置以下选项：

### 4.1 General -> Origin

确认源站配置：

| 设置项 | 值 |
|--------|-----|
| **Origin URL** | `https://47.79.151.22` |
| **Origin Host Header** | `xingsuancode.com` |
| **Forward Host Header** | 开启 |

> **重要**：`Origin Host Header` 必须设置为 `xingsuancode.com`，否则日本服务器 Nginx 无法识别虚拟主机，会返回错误。

### 4.2 General -> Routing

选择 CDN 节点区域：

- **Asia & Oceania**（必选，包含东京、香港节点）
- 其他区域按需选择（如果只服务中国用户，只选亚太即可省流量费）

### 4.3 Caching -> General

| 设置项 | 值 | 说明 |
|--------|-----|------|
| **Smart Cache** | 保持开启 | `application/json` 和 `text/event-stream` 不在缓存列表中，默认不缓存 |
| **Cache Expiration Time** | `Respect Origin Cache-Control` | 尊重源站缓存头 |
| **Browser Cache Expiration Time** | `Match Server Cache Expiration` | 与服务端一致 |
| **Strip Response Cookies** | 关闭 | 保留响应 Cookie |

### 4.4 Caching -> Request Coalescing（安全关键）

| 设置项 | 值 | 说明 |
|--------|-----|------|
| **Request Coalescing** | **关闭** | **必须关闭！** |

> **为什么必须关闭**：Request Coalescing 会把同时到达的相同 URL 请求合并为一个源站请求。
> 对于 API 场景，两个用户同时请求 `/v1/messages` 会被合并 -> **一个用户会收到另一个用户的 AI 回复**。
> 这是数据泄露/安全问题，不是性能问题。Bunny 官方也警告：
> "If your origin returns different responses based on authentication or user context, enabling this feature could cause personal information to be shared between users."

### 4.5 Headers

| 设置项 | 值 | 说明 |
|--------|-----|------|
| **Add CORS Headers** | 开启 | API 跨域需要 |
| **Enable Access Control (Allow Origin)** | `*` | 或填你的前端域名 |

### 4.6 Edge Rules（SSE 流式关键配置）

进入 **Edge Rules** 页面，添加以下规则：

**规则 1：API 路径禁用缓存 + 防缓冲**

```
Rule Name:    Bypass Cache for API
Condition:    If URL matches /v1/*
Actions:
  1. Override Cache Time -> 0 (秒)
  2. Override Browser Cache Time -> 0 (秒)
  3. Set Response Header -> X-Accel-Buffering: no
  4. Set Response Header -> Cache-Control: no-cache, no-store
```

**规则 2：健康检查路径禁用缓存**

```
Rule Name:    Bypass Cache for Health
Condition:    If URL matches /health
Actions:
  1. Override Cache Time -> 0 (秒)
```

> **注意**：经调研确认，Bunny Edge Rules **不支持按 HTTP Method（POST）匹配**。
> 只能通过 URL 路径匹配。好在 Sub2API 的 API 路径都在 `/v1/` 下，路径匹配已足够覆盖。
>
> `X-Accel-Buffering: no` 是关键头 —— Bunny 底层是 Nginx，该头会告诉 Nginx 不要缓冲响应，
> 直接逐块转发给客户端。Sub2API 源站本身也会发送此头（代码中 12 处设置），Edge Rule 作为双保险。

---

## 第 5 步：绑定自定义域名 + HTTPS

### 5.1 添加自定义域名

1. Pull Zone 设置 -> **Hostnames**
2. 点击 **Add Hostname**
3. 输入 `xingsuancode.com`
4. 点击 **Add**

### 5.2 修改 DNS

到你的域名 DNS 管理面板（阿里云 / Cloudflare / 其他），修改 `xingsuancode.com` 的解析：

| 操作 | 记录类型 | 主机记录 | 值 |
|------|---------|---------|-----|
| 删除旧记录 | ~~A~~ | `@` | ~~47.79.151.22~~ |
| 添加新记录 | **CNAME** | `@` | `xingsuancode.b-cdn.net` |

> **注意**：
> - CNAME 值格式为 `你的pullzone名.b-cdn.net`，在 Hostnames 页面会显示具体值
> - 如果你的 DNS 不支持根域名 CNAME（部分老旧 DNS），可以使用 Cloudflare（支持根域名 CNAME Flattening）
> - **`cn.xingsuancode.com` 的 DNS 不要改**，保持指向国内服务器

### 5.3 启用 HTTPS（Free SSL）

DNS 生效后（通常 1-5 分钟）：

1. 回到 **Hostnames** 页面
2. 找到 `xingsuancode.com`，点击右侧 **Add Free SSL Certificate**
3. Bunny.net 会自动签发 Let's Encrypt 证书（1-2 分钟）
4. 勾选 **Force SSL**（强制 HTTPS）

> 日本源站的 Let's Encrypt 证书保持不变，用于 CDN -> 源站的 HTTPS 回源。
> Bunny 证书自动续期，无需手动操作。

---

## 第 6 步：验证 SSE 流式（先验证再改 base_url）

> **重要**：不要急着改国内 accounts 表。先在国内服务器上手动测试 CDN 链路是否正常。

### 6.1 基础连通性测试

```bash
# SSH 到国内服务器后执行

# 测试 CDN 链路
curl -w "DNS: %{time_namelookup}s | Connect: %{time_connect}s | TLS: %{time_appconnect}s | Total: %{time_total}s\n" \
  -o /dev/null -s https://xingsuancode.com/health

# 对比直连（参考基线）
curl -w "DNS: %{time_namelookup}s | Connect: %{time_connect}s | TLS: %{time_appconnect}s | Total: %{time_total}s\n" \
  -o /dev/null -s -k https://47.79.151.22/health -H "Host: xingsuancode.com"
```

确认返回 200 再继续。

### 6.2 确认 Bunny CDN 节点

```bash
curl -sI https://xingsuancode.com/health | grep -i "cdn\|server\|cache"

# 应包含 Bunny.net 标识头，例如：
# CDN-PullZone: xingsuancode
# CDN-Uid: xxxxxx
# CDN-RequestCountryCode: CN
```

### 6.3 SSE 流式验证（核心测试）

```bash
# 通过 CDN 链路测试 SSE 流式
# 替换 sk-your-key 为日本服务器上有效的 API Key
curl -N --no-buffer https://xingsuancode.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: sk-your-key" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-6",
    "max_tokens": 200,
    "stream": true,
    "messages": [{"role": "user", "content": "Count from 1 to 20, one number per line, slowly."}]
  }'
```

**判断标准**：

| 现象 | 结论 | 后续 |
|------|------|------|
| 数据逐行实时输出（event: content_block_delta 逐个出现） | SSE 正常 | 继续第 7 步 |
| 长时间无输出，然后一次性涌出全部内容 | SSE 被缓冲 | 见下方"缓冲问题排查" |
| 超时断开（约 60s 后连接中断） | 超时限制 | 见下方"超时问题排查" |
| 返回错误 | 配置问题 | 检查 Origin Host Header 和源站 |

### 6.4 缓冲问题排查

如果 SSE 被缓冲（数据攒批返回），按顺序尝试：

1. **确认 Edge Rule 生效**：
```bash
curl -sI https://xingsuancode.com/v1/test | grep -i "x-accel-buffering\|cache-control"
# 应看到：
# X-Accel-Buffering: no
# Cache-Control: no-cache, no-store
```

2. **检查源站头是否被覆盖**：如果 Edge Rule 的 `Set Response Header` 没生效，可能是 Bunny 边缘节点在 Edge Rule 之前就做了缓冲处理。

3. **联系 Bunny 客服**：提交 ticket 问 "Does your CDN proxy support streaming `text/event-stream` responses without buffering?"

4. **如果无法解决** -> 回滚（第 9 步），考虑 Peekabo 或其他方案。

### 6.5 超时问题排查

如果长对话被切断（连接在 60-120 秒后中断）：

1. 用短对话测试（`max_tokens: 50`）确认短请求正常
2. 用长对话测试（`max_tokens: 4096`，用 Opus 模型）确认超时阈值
3. Bunny 不支持自定义 `proxy_read_timeout`，如果超时无法接受 -> 回滚

---

## 第 7 步：更新国内服务器 base_url

**仅在第 6 步全部通过后执行此步。**

### 7.1 SSH 到国内服务器

```bash
docker exec -i $(docker ps -q -f name=postgres) psql -U postgres -d sub2api
```

### 7.2 更新 accounts 表

```sql
-- 先查看当前配置
SELECT id, name, credentials->'base_url' AS base_url
FROM accounts
WHERE type = 'apikey' AND deleted_at IS NULL;

-- 更新为 CDN 加速地址
UPDATE accounts
SET credentials = jsonb_set(credentials, '{base_url}', '"https://xingsuancode.com/"')
WHERE type = 'apikey' AND deleted_at IS NULL;

-- 确认更新结果
SELECT id, name, credentials->'base_url' AS base_url
FROM accounts
WHERE type = 'apikey' AND deleted_at IS NULL;
```

### 7.3 端到端验证

```bash
# 用真实用户路径测试：用户 -> 国内 -> CDN -> 日本 -> Anthropic
curl -N --no-buffer https://cn.xingsuancode.com/v1/messages \
  -H "Content-Type: application/json" \
  -H "x-api-key: sk-your-user-key" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-6",
    "max_tokens": 100,
    "stream": true,
    "messages": [{"role": "user", "content": "Say hello"}]
  }'
```

---

## 第 8 步：多地域测试（可选）

使用在线工具测试全球各地访问速度：
- **https://check-host.net/check-ping** — 输入 `xingsuancode.com` 测试多地 ping
- **https://tools.bunny.net/latency-test** — Bunny 官方延迟测试

---

## 第 9 步：回滚方案

如果 SSE 不工作、延迟不理想、或出现任何问题，可以 5 分钟回滚：

### 9.1 恢复 DNS

将 `xingsuancode.com` 的 CNAME 改回 A 记录：

| 记录类型 | 主机记录 | 值 |
|---------|---------|-----|
| A | `@` | `47.79.151.22` |

### 9.2 恢复 base_url

```sql
UPDATE accounts
SET credentials = jsonb_set(credentials, '{base_url}', '"http://47.79.151.22:8080/"')
WHERE type = 'apikey' AND deleted_at IS NULL;
```

### 9.3 等待 DNS 传播（5-30 分钟）

> 回滚后一切恢复原状。用户 base_url 始终是 `https://cn.xingsuancode.com/`，全程对用户无感知。

---

## 费用总结

| 项目 | 费用 |
|------|------|
| **14 天试用** | 免费（1TB 流量） |
| **亚太区流量** | $0.03/GB |
| **月预估（5 万用户）** | $2-8/月 |
| **SSL 证书** | 免费（内置 Let's Encrypt） |
| **DDoS 防护** | 免费（内置基础防护） |

---

## 与 Peekabo 对比

| 维度 | Bunny.net | Peekabo |
|------|-----------|---------|
| **中国延迟** | 70-100ms（东京/香港普通线路） | 30-50ms（CN2 GIA 三网直连） |
| **SSE 验证** | 未知，需实测 | 88code 生产验证 |
| **月费** | $2-8（按量） | 70-100 RMB（固定） |
| **支付方式** | Visa / PayPal | 微信 / 支付宝 |
| **免费试用** | 14 天 + 1TB | 无 |
| **控制台** | 英文 | 中文 |
| **超时可控** | 不可控 | 可联系客服调整 |

> **建议**：先用 Bunny.net 14 天免费试用验证 SSE。如果 SSE 不工作或延迟不满意，切换到 Peekabo（找人代付微信/支付宝）。
> 两者切换只需改 DNS CNAME，对用户完全无感。

---

*教程版本：v2.0 | 适用于 Sub2API 双层代理架构 | 2026-03-09*
