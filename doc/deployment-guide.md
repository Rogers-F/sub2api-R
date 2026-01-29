# Sub2API Ubuntu 部署指南

本文档介绍如何在 Ubuntu 云服务器上部署 Sub2API，包括 Docker 部署、域名绑定、SSL 配置及日常运维。

---

## 目录

- [环境要求](#环境要求)
- [Docker 部署](#docker-部署)
- [域名绑定与 HTTPS](#域名绑定与-https)
- [日常运维](#日常运维)
- [更新升级](#更新升级)
- [故障排查](#故障排查)

---

## 环境要求

- Ubuntu 20.04+ / Debian 11+
- 1 核 CPU / 1GB 内存（最低）
- 开放端口：80、443、8080（8080 仅内部使用，可不对外开放）

---

## Docker 部署

### 1. 安装 Docker

```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# 安装 Docker Compose 插件
sudo apt update && sudo apt install -y docker-compose-plugin

# 重新登录使 docker 组生效
exit
# 重新 SSH 登录后继续
```

### 2. 克隆仓库

```bash
git clone https://github.com/Wei-Shaw/sub2api.git
cd sub2api/deploy
```

### 3. 配置环境变量

```bash
cp .env.example .env
nano .env
```

**必须修改的配置项：**

```bash
# 数据库密码（必须设置）
POSTGRES_PASSWORD=你的安全密码

# JWT 密钥（强烈建议设置，防止容器重启后登录失效）
# 生成命令：openssl rand -hex 32
JWT_SECRET=生成的64位十六进制字符串

# 管理员密码（可选，不设置会自动生成并显示在日志中）
ADMIN_PASSWORD=你的管理员密码
```

保存退出：`Ctrl+O` 回车保存，`Ctrl+X` 退出

### 4. 启动服务

```bash
docker compose up -d
```

### 5. 查看启动日志

```bash
docker compose logs -f sub2api
```

看到 `Server started on :8080` 表示启动成功。

如果没设置 `ADMIN_PASSWORD`，在日志中查找自动生成的密码：

```bash
docker compose logs sub2api | grep -i password
```

### 6. 访问

- 地址：`http://服务器IP:8080`
- 管理员邮箱：`admin@sub2api.local`
- 密码：你设置的或日志中显示的

---

## 域名绑定与 HTTPS

### 1. DNS 解析

在域名服务商控制台添加 A 记录：

| 记录类型 | 主机记录 | 记录值 |
|---------|---------|--------|
| A | @ | 服务器公网 IP |
| A | www | 服务器公网 IP |

### 2. 安装 Nginx + Certbot

```bash
sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx
```

### 3. 配置 Nginx

```bash
sudo nano /etc/nginx/sites-available/sub2api
```

粘贴以下内容（将 `example.com` 替换为你的域名）：

```nginx
server {
    listen 80;
    server_name example.com www.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE 流式响应支持
        proxy_http_version 1.1;
        proxy_set_header Connection '';
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400;
    }
}
```

保存退出。

### 4. 启用站点配置

```bash
# 创建软链接
sudo ln -s /etc/nginx/sites-available/sub2api /etc/nginx/sites-enabled/

# 测试配置
sudo nginx -t

# 重载 Nginx
sudo systemctl reload nginx
```

### 5. 申请 SSL 证书

```bash
sudo certbot --nginx -d example.com -d www.example.com
```

按提示操作：
1. 输入邮箱（用于证书到期提醒）
2. 同意条款：`Y`
3. 是否分享邮箱：`N`
4. HTTP 重定向到 HTTPS：选 `2`

### 6. 开放防火墙端口

```bash
# UFW 防火墙
sudo ufw allow 80
sudo ufw allow 443

# 云服务器还需在控制台安全组放行 80、443 端口
```

### 7. 验证

访问 `https://你的域名` 确认正常。

---

## 日常运维

### 服务管理

| 操作 | 命令 |
|------|------|
| 启动服务 | `docker compose up -d` |
| 停止服务 | `docker compose down` |
| 重启服务 | `docker compose restart` |
| 查看状态 | `docker compose ps` |
| 查看日志 | `docker compose logs -f sub2api` |
| 查看最近日志 | `docker compose logs --tail=100 sub2api` |

### 数据库操作

```bash
# 进入 PostgreSQL
docker compose exec postgres psql -U sub2api -d sub2api

# 备份数据库
docker compose exec postgres pg_dump -U sub2api sub2api > backup_$(date +%Y%m%d).sql

# 恢复数据库
cat backup.sql | docker compose exec -T postgres psql -U sub2api -d sub2api
```

### Redis 操作

```bash
# 进入 Redis CLI
docker compose exec redis redis-cli

# 查看键值
docker compose exec redis redis-cli keys '*'
```

### SSL 证书管理

```bash
# 查看证书状态
sudo certbot certificates

# 手动续期测试
sudo certbot renew --dry-run

# 强制续期
sudo certbot renew --force-renewal
```

证书会自动续期，无需手动操作。

---

## 更新升级

### 方式一：指定版本号更新（推荐）

```bash
cd ~/sub2api/deploy

# 1. 查看当前镜像版本
cat docker-compose.yml | grep image

# 2. 修改镜像版本号（将 OLD_VERSION 替换为旧版本，NEW_VERSION 替换为新版本）
sed -i 's/OLD_VERSION/NEW_VERSION/g' docker-compose.yml

# 例如：从 0.1.63-fork 更新到 0.1.65-fork
sed -i 's/0.1.63-fork/0.1.65-fork/g' docker-compose.yml

# 3. 拉取新镜像
docker compose pull

# 4. 重启服务
docker compose up -d

# 5. 查看日志确认启动成功
docker compose logs -f sub2api
```

### 方式二：拉取 latest 镜像

如果 `docker-compose.yml` 中配置的是 `latest` 标签：

```bash
cd ~/sub2api/deploy

# 拉取最新镜像
docker compose pull

# 重启服务（自动使用新镜像）
docker compose up -d

# 查看日志确认启动成功
docker compose logs -f sub2api
```

### 方式三：更新代码后重建

```bash
cd ~/sub2api

# 拉取最新代码
git pull origin main

# 进入部署目录
cd deploy

# 重新拉取镜像并启动
docker compose pull
docker compose up -d
```

### 常见问题：容器没有更新

如果执行 `docker compose up -d` 后显示 `Running` 而非 `Recreated`，说明镜像没变：

```bash
# 强制重建容器
docker compose up -d --force-recreate

# 或者先停止再启动
docker compose down
docker compose up -d
```

### 更新注意事项

1. **备份数据**：重大更新前建议备份数据库
   ```bash
   docker compose exec postgres pg_dump -U sub2api sub2api > backup_$(date +%Y%m%d).sql
   ```

2. **查看更新日志**：访问 [GitHub Releases](https://github.com/Wei-Shaw/sub2api/releases) 了解版本变更

3. **配置迁移**：如果 `.env.example` 有新增配置项，需同步到 `.env`

---

## 故障排查

### 常见问题

#### 1. 容器无法启动

```bash
# 查看详细日志
docker compose logs sub2api

# 检查端口占用
sudo lsof -i :8080
```

#### 2. 数据库连接失败

```bash
# 检查 PostgreSQL 状态
docker compose ps postgres

# 检查数据库连接
docker compose exec postgres pg_isready
```

#### 3. Redis 连接失败

```bash
# 检查 Redis 状态
docker compose ps redis

# 测试连接
docker compose exec redis redis-cli ping
```

#### 4. 域名无法访问

```bash
# 检查 Nginx 状态
sudo systemctl status nginx

# 检查 Nginx 配置
sudo nginx -t

# 查看 Nginx 错误日志
sudo tail -f /var/log/nginx/error.log
```

#### 5. SSL 证书问题

```bash
# 查看证书状态
sudo certbot certificates

# 重新申请证书
sudo certbot --nginx -d example.com
```

### 重置服务

如需完全重置（**会删除所有数据**）：

```bash
cd ~/sub2api/deploy

# 停止并删除所有容器和数据卷
docker compose down -v

# 重新启动
docker compose up -d
```

---

## 附录

### 目录结构

```
~/sub2api/
├── deploy/
│   ├── docker-compose.yml    # Docker 编排文件
│   ├── .env                  # 环境配置（你创建的）
│   └── .env.example          # 配置模板
└── ...

/etc/nginx/sites-available/
└── sub2api                   # Nginx 站点配置

/etc/letsencrypt/live/域名/
├── fullchain.pem             # SSL 证书
└── privkey.pem               # SSL 私钥
```

### 常用端口

| 端口 | 服务 | 说明 |
|------|------|------|
| 80 | Nginx | HTTP（自动跳转 HTTPS） |
| 443 | Nginx | HTTPS |
| 8080 | Sub2API | 应用服务（内部） |
| 5432 | PostgreSQL | 数据库（内部） |
| 6379 | Redis | 缓存（内部） |
