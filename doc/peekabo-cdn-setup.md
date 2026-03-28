# Peekabo CDN 加速接入教程 — xingsuancode.com

> **目标**：通过 Peekabo CDN（三网直连）加速中国用户访问日本服务器 `xingsuancode.com`，确保 SSE 流式 API 正常工作。
>
> **适用场景**：Sub2API 双层代理架构，国内服务器 -> 日本服务器（经 CDN 加速）-> Anthropic/OpenAI
>
> **支付方式**：微信 / 支付宝（信用卡和加密货币即将支持）

---

## 为什么选 Peekabo

| 项目 | 说明 |
|------|------|
| **SSE 验证** | 88code 项目（5万+用户）已在生产环境使用 Peekabo，SSE 流式无问题 |
| **线路质量** | 三网直连：电信 163 + 联通 4837 + 移动 CMI，中国延迟 30-50ms |
| **DDoS 防护** | 内置企业级 DDoS 防护 |
| **即时开通** | 付款后自动配置，无需人工审核 |
| **专注亚太** | 香港/日本节点专门优化中国大陆连接，100G+ 直连带宽 |

---

## 架构变更说明

```
变更前（直连）:
  用户 -> cn.xingsuancode.com（国内）-> 47.79.151.22:8080（日本直连）-> Anthropic

变更后（CDN 加速）:
  用户 -> cn.xingsuancode.com（国内）-> xingsuancode.com（Peekabo CDN）-> 47.79.151.22:443（日本源站）-> Anthropic
                                              |
                                     Peekabo 日本节点（三网直连）加速
```

**预期效果**：中国 -> 日本延迟从 60-120ms 降至 30-50ms，SSE 流式更稳定（88code 生产验证）。

---

## 第 1 步：注册 Peekabo 账号

1. 打开 **https://peekabo.io**
2. 点击右上角 **Register**（或中文界面的"注册"）
3. 填写邮箱、密码，完成注册
4. 验证邮箱

> **备注**：控制面板目前仅支持中文版，英文版即将推出。
> Telegram 客户群：https://t.me/peekabo_cdn（遇到问题可在群里咨询）

---

## 第 2 步：购买套餐

1. 登录后进入首页，选择产品：
   - **Photon CDN - JPN**（日本节点）— 推荐，源站在日本
   - Photon CDN - HKG（香港节点）— 备选

2. 选择套餐层级：

| 套餐 | 价格 | 说明 |
|------|------|------|
| 基础款 | $10/月（$100/年） | 适合起步测试 |
| 更高层级 | 按需选择 | 更多带宽和流量 |

> 所有层级享受相同的防护能力和中国优化服务。

3. 支付方式选择 **微信** 或 **支付宝**
4. 付款后即时开通，无需等待

---

## 第 3 步：添加网站

### 3.1 打开 CDN 面板

1. 登录后进入「产品中心」
2. 找到刚购买的 Photon CDN 产品，点击「检视详情」
3. 点击「打开 CDN 面板」

### 3.2 添加网站（三步向导）

点击「添加网站」，按向导填写：

**第一步：基本信息**

| 字段 | 值 | 说明 |
|------|-----|------|
| **服务计划** | 选择已购买的套餐 | — |
| **协议** | 先启用 HTTP | HTTPS 后续配置 |
| **网站名称** | `xingsuancode` | 自定义标识 |
| **域名** | `xingsuancode.com` | 主域名 |

**第二步：回源配置**

| 字段 | 值 | 说明 |
|------|-----|------|
| **源站 IP** | `47.79.151.22` | 日本服务器 IP |
| **回源协议** | **HTTPS** | 日本 Nginx 已有 Let's Encrypt 证书 |
| **回源端口** | **443** | — |
| **回源主机名策略** | **跟随 CDN 服务** | CDN 域名和源站域名一致，选此项 |

> **为什么选 HTTPS 回源**：日本服务器 Nginx 已配好 Let's Encrypt SSL 证书（:443），
> 使用 HTTPS 回源可以确保 CDN -> 源站链路加密。

**第三步：完成**

系统会自动分配一个 CNAME 目标地址，格式类似 `xxx.peekabo.io` 或 `xxx.sharon.io`。
**记录下这个 CNAME 地址**，下一步要用。

---

## 第 4 步：修改 DNS（CNAME 解析）

到你的域名 DNS 管理面板操作：

### 4.1 删除旧记录

| 操作 | 记录类型 | 主机记录 | 原值 |
|------|---------|---------|------|
| **删除** | A | `@` | `47.79.151.22` |

### 4.2 添加新记录

| 操作 | 记录类型 | 主机记录 | 值 |
|------|---------|---------|-----|
| **添加** | CNAME | `@` | `系统分配的 CNAME 地址` |

> **注意事项**：
> - `cn.xingsuancode.com` 的 DNS **不要改**，保持指向国内服务器
> - 如果使用 Cloudflare DNS，**关闭小黄云**（代理状态设为灰色/DNS only）
> - Cloudflare 用户：根域名 CNAME 可能显示"未解析"，这是正常的（CNAME Flattening 导致），不影响使用

### 4.3 等待 DNS 生效

回到 Peekabo CDN 面板，系统会自动检测 DNS 是否生效。通常 1-5 分钟。

---

## 第 5 步：申请 SSL 证书

### 5.1 创建 ACME 账户

1. 在 CDN 面板左侧菜单进入「证书库」
2. 点击「创建 ACME 账户」
3. 填写你的邮箱地址
4. 完成创建

### 5.2 申请证书

1. 点击「申请证书」
2. 选择域名 `xingsuancode.com`
3. 建议开启自动续期
4. 等待证书签发（通常 1-2 分钟）
5. 确认证书状态为「已签发」

---

## 第 6 步：启用 HTTPS

1. 回到网站详情页面
2. 进入「HTTPS」选项卡
3. **开启 HTTPS 开关**
4. 选择刚申请好的 SSL 证书
5. 点击「保存更改」
6. 切换到「HTTP」选项卡
7. **开启「强制 HTTPS 跳转」**

> 至此，用户访问 `http://xingsuancode.com` 会自动跳转到 `https://xingsuancode.com`。

---

## 第 7 步：验证 CDN 生效（先验证再改 base_url）

> **重要**：不要急着改国内 accounts 表。先手动测试 CDN 链路。

### 7.1 基础连通性测试

```bash
# SSH 到国内服务器后执行

# 测试 CDN 链路
curl -w "DNS: %{time_namelookup}s | Connect: %{time_connect}s | TLS: %{time_appconnect}s | Total: %{time_total}s\n" \
  -o /dev/null -s https://xingsuancode.com/health

# 对比直连（参考基线）
curl -w "DNS: %{time_namelookup}s | Connect: %{time_connect}s | TLS: %{time_appconnect}s | Total: %{time_total}s\n" \
  -o /dev/null -s -k https://47.79.151.22/health -H "Host: xingsuancode.com"
```

预期 CDN 链路比直连快（尤其是 Connect 和 TLS 阶段）。

### 7.2 SSE 流式验证

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
| 数据逐行实时输出 | SSE 正常 | 继续第 8 步 |
| 长时间无输出后一次性涌出 | SSE 被缓冲 | 联系 Peekabo 客服（Telegram 群） |
| 超时断开 | 超时配置问题 | 联系 Peekabo 客服调整 |

> **Peekabo 大概率不会出现缓冲问题**——88code 项目已在生产验证 SSE 正常。
> 但仍建议测试确认后再改 base_url。

### 7.3 延迟对比

```bash
# 多次测试取平均值
for i in {1..5}; do
  echo "--- Test $i ---"
  curl -w "Total: %{time_total}s\n" -o /dev/null -s https://xingsuancode.com/health
  sleep 1
done
```

预期延迟：30-50ms（vs 直连 60-120ms）。

---

## 第 8 步：更新国内服务器 base_url

**仅在第 7 步全部通过后执行。**

### 8.1 SSH 到国内服务器

```bash
docker exec -i $(docker ps -q -f name=postgres) psql -U postgres -d sub2api
```

### 8.2 更新 accounts 表

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

### 8.3 端到端验证

```bash
# 完整用户路径：用户 -> 国内 -> CDN -> 日本 -> Anthropic
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

确认流式正常后，CDN 接入完成。

---

## 第 9 步：回滚方案

如出现任何问题，可以 5 分钟回滚：

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
| **Photon CDN - JPN 基础套餐** | $10/月（$100/年） |
| **SSL 证书** | 免费（内置 ACME / Let's Encrypt） |
| **DDoS 防护** | 免费（内置企业级防护） |

---

## 与 Bunny.net 对比

| 维度 | Peekabo | Bunny.net |
|------|---------|-----------|
| **中国延迟** | 30-50ms（三网直连） | 70-100ms（东京/香港普通线路） |
| **SSE 验证** | 88code 生产验证 | 未知，需实测 |
| **月费** | $10 起（固定套餐） | $2-8（按量计费） |
| **支付方式** | 微信 / 支付宝 | Visa / PayPal |
| **免费试用** | 无 | 14 天 + 1TB |
| **控制台** | 中文 | 英文 |
| **超时可控** | 可联系客服调整 | 不可控 |
| **线路质量** | 电信163 + 联通4837 + 移动CMI | 普通国际线路 |

> **总结**：Peekabo 在延迟和 SSE 可靠性上明显优于 Bunny.net，但不支持 Visa 支付。
> 如果能用微信/支付宝付款，Peekabo 是首选方案。

---

## 常见问题

### Q: Cloudflare DNS 显示"未解析"怎么办？
A: 正常现象。Cloudflare 对根域名 CNAME 会做 Flattening（展平），DNS 检测工具会显示 A 记录而非 CNAME，但实际解析正常，不影响使用。

### Q: 回源主机名策略选哪个？
A: 如果 CDN 域名就是 `xingsuancode.com`（和源站域名一致），选「跟随 CDN 服务」。如果用子域名（如 `cdn.xingsuancode.com`）接入 CDN，选「跟随源站」。

### Q: 日本源站的 SSL 证书需要换吗？
A: 不需要。日本 Nginx 保留原有的 Let's Encrypt 证书，用于 CDN -> 源站的 HTTPS 回源。用户侧的证书由 Peekabo 管理（第 5 步申请的证书）。

### Q: 可以同时给 API 和网页加速吗？
A: 可以。CDN 加速的是整个 `xingsuancode.com` 域名，网页和 API 请求都会走 CDN 节点。

### Q: 后续会支持 CN2 GIA 线路吗？
A: Peekabo 官方已预告将推出「三网 CN2 + 10099 + CMIN2」顶级优化线路产品，届时延迟可进一步降低到 20-40ms。

---

## 参考资源

- Peekabo 官网：https://peekabo.io
- Telegram 客户群：https://t.me/peekabo_cdn
- 社区教程（MISAKA）：https://misaka.es/archives/57.html
- 社区教程（Lumos）：https://lomus.cc/archives/555

---

*教程版本：v1.0 | 适用于 Sub2API 双层代理架构 | 2026-03-09*
