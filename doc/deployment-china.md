# Sub2API Fork 版本 - 中国境内部署指南

本文档介绍如何在中国境内服务器上部署 Sub2API Fork 版本（Rogers-F/sub2api-R）。

---

## 目录

- [环境要求](#环境要求)
- [Docker 安装](#docker-安装)
  - [Ubuntu/Debian](#ubuntudebian)
  - [CentOS/AlmaLinux/Rocky Linux](#centosalmalinuxrocky-linux)
- [配置镜像加速](#配置镜像加速)
- [部署服务](#部署服务)
- [域名与 HTTPS](#域名与-https)
- [更新升级](#更新升级)
- [常见问题](#常见问题)

---

## 环境要求

- Ubuntu 20.04+ / Debian 11+ / CentOS 7+ / AlmaLinux 8+ / Rocky Linux 8+
- 1 核 CPU / 1GB 内存（最低）
- 开放端口：80、443、8080

**判断系统类型**：

```bash
cat /etc/os-release
```

---

## Docker 安装

### Ubuntu/Debian

#### 方式一：使用阿里云镜像安装（推荐）

```bash
# 安装依赖
sudo apt update
sudo apt install -y ca-certificates curl gnupg

# 添加阿里云 Docker GPG 密钥
sudo install -m 0755 -d /etc/apt/keyrings
curl -fsSL https://mirrors.aliyun.com/docker-ce/linux/ubuntu/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg
sudo chmod a+r /etc/apt/keyrings/docker.gpg

# 添加阿里云 Docker 仓库
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] https://mirrors.aliyun.com/docker-ce/linux/ubuntu \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

# 安装 Docker
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 将当前用户加入 docker 组
sudo usermod -aG docker $USER

# 重新登录使生效
exit
```

#### 方式二：一键脚本安装

```bash
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER
exit
```

重新 SSH 登录后验证：

```bash
docker --version
docker compose version
```

---

### CentOS/AlmaLinux/Rocky Linux

#### 方式一：使用阿里云镜像安装（推荐）

```bash
# 安装依赖
sudo yum install -y yum-utils

# 添加阿里云 Docker 仓库
sudo yum-config-manager --add-repo https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo

# 替换仓库地址为阿里云
sudo sed -i 's+download.docker.com+mirrors.aliyun.com/docker-ce+' /etc/yum.repos.d/docker-ce.repo

# 安装 Docker
sudo yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin

# 启动 Docker 并设置开机自启
sudo systemctl start docker
sudo systemctl enable docker

# 将当前用户加入 docker 组（如果不是 root）
sudo usermod -aG docker $USER

# 重新登录使生效（非 root 用户）
exit
```

#### 方式二：一键脚本安装

```bash
curl -fsSL https://get.docker.com | sh
sudo systemctl start docker
sudo systemctl enable docker
exit
```

重新 SSH 登录后验证：

```bash
docker --version
docker compose version
```

#### CentOS 7 特殊说明

如果是 CentOS 7，可能需要先更新内核或使用旧版本 Docker：

```bash
# 安装 EPEL 源
sudo yum install -y epel-release

# 更新系统
sudo yum update -y
```

---

## 配置镜像加速

中国境内拉取 ghcr.io 镜像可能较慢或失败，配置镜像加速：

### 方式一：使用南京大学镜像（推荐）

```bash
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json <<EOF
{
  "registry-mirrors": [
    "https://docker.nju.edu.cn"
  ]
}
EOF

# 重启 Docker
sudo systemctl daemon-reload
sudo systemctl restart docker
```

### 方式二：使用其他镜像源

```bash
sudo tee /etc/docker/daemon.json <<EOF
{
  "registry-mirrors": [
    "https://docker.nju.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ]
}
EOF

sudo systemctl daemon-reload
sudo systemctl restart docker
```

### 验证镜像加速

```bash
docker info | grep -A 5 "Registry Mirrors"
```

---

## 部署服务

### 1. 创建部署目录

```bash
mkdir -p ~/sub2api/deploy
cd ~/sub2api/deploy
```

### 2. 创建 docker-compose.yml

```bash
cat > docker-compose.yml <<'EOF'
services:
  sub2api:
    image: ghcr.io/rogers-f/sub2api:v0.2.4
    container_name: sub2api
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - DATABASE_URL=postgres://sub2api:${POSTGRES_PASSWORD}@postgres:5432/sub2api?sslmode=disable
      - REDIS_URL=redis://redis:6379/0
      - JWT_SECRET=${JWT_SECRET}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD}
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy

  postgres:
    image: postgres:16-alpine
    container_name: sub2api-postgres
    restart: unless-stopped
    environment:
      - POSTGRES_USER=sub2api
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
      - POSTGRES_DB=sub2api
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U sub2api"]
      interval: 5s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: sub2api-redis
    restart: unless-stopped
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
  redis_data:
EOF
```

### 3. 创建环境变量文件

```bash
cat > .env <<'EOF'
# 数据库密码（必须修改）
POSTGRES_PASSWORD=your_secure_password_here

# JWT 密钥（必须修改，使用下面命令生成）
# openssl rand -hex 32
JWT_SECRET=your_jwt_secret_here

# 管理员密码（可选，不设置会自动生成）
ADMIN_PASSWORD=your_admin_password
EOF
```

生成安全的密钥：

```bash
# 生成 JWT 密钥
echo "JWT_SECRET=$(openssl rand -hex 32)"

# 生成数据库密码
echo "POSTGRES_PASSWORD=$(openssl rand -hex 16)"
```

编辑 .env 文件填入生成的密钥：

```bash
nano .env
```

### 4. 拉取镜像

如果直接拉取 ghcr.io 失败，使用南京大学镜像：

```bash
# 方式一：直接拉取（配置镜像加速后）
docker compose pull

# 方式二：使用南京大学 ghcr 镜像（如果方式一失败）
docker pull ghcr.nju.edu.cn/rogers-f/sub2api:v0.2.4
docker tag ghcr.nju.edu.cn/rogers-f/sub2api:v0.2.4 ghcr.io/rogers-f/sub2api:v0.2.4
```

### 5. 启动服务

```bash
docker compose up -d
```

### 6. 查看日志

```bash
docker compose logs -f sub2api
```

看到 `Server started on :8080` 表示启动成功。

如果没设置 `ADMIN_PASSWORD`，查找自动生成的密码：

```bash
docker compose logs sub2api | grep -i password
```

### 7. 访问

- 地址：`http://服务器IP:8080`
- 管理员邮箱：`admin@sub2api.local`
- 密码：你设置的或日志中显示的

---

## 域名与 HTTPS

### 1. 安装 Nginx + Certbot

```bash
sudo apt update
sudo apt install -y nginx certbot python3-certbot-nginx
```

### 2. 配置 Nginx

```bash
sudo nano /etc/nginx/sites-available/sub2api
```

粘贴以下内容（将 `example.com` 替换为你的域名）：

```nginx
server {
    listen 80;
    server_name example.com;

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

### 3. 启用配置

```bash
sudo ln -s /etc/nginx/sites-available/sub2api /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

### 4. 申请 SSL 证书

```bash
sudo certbot --nginx -d example.com
```

---

## 更新升级

### 1. 查看当前版本

```bash
cd ~/sub2api/deploy
cat docker-compose.yml | grep image
```

### 2. 修改版本号

```bash
# 将旧版本替换为新版本号
sed -i 's/OLD_VERSION/NEW_VERSION/g' docker-compose.yml

# 例如更新到 v0.2.5
sed -i 's/v0.2.4/v0.2.5/g' docker-compose.yml
```

### 3. 拉取新镜像

```bash
# 方式一：直接拉取
docker compose pull

# 方式二：使用南京大学镜像（如果方式一失败）
docker pull ghcr.nju.edu.cn/rogers-f/sub2api:vNEW_VERSION
docker tag ghcr.nju.edu.cn/rogers-f/sub2api:vNEW_VERSION ghcr.io/rogers-f/sub2api:vNEW_VERSION
```

### 4. 重启服务

```bash
docker compose up -d
```

### 5. 验证更新

```bash
# 确认容器状态为 Recreated（而非 Running）
docker compose ps

# 查看日志
docker compose logs -f sub2api
```

如果显示 `Running` 而非 `Recreated`：

```bash
docker compose up -d --force-recreate
```

---

## 常见问题

### 1. 拉取镜像失败

**错误**：`Error response from daemon: Get "https://ghcr.io/v2/": net/http: request canceled`

**解决**：使用南京大学镜像

```bash
docker pull ghcr.nju.edu.cn/rogers-f/sub2api:v0.2.4
docker tag ghcr.nju.edu.cn/rogers-f/sub2api:v0.2.4 ghcr.io/rogers-f/sub2api:v0.2.4
```

### 2. 容器启动失败

```bash
# 查看详细日志
docker compose logs sub2api

# 检查数据库连接
docker compose exec postgres pg_isready

# 检查 Redis 连接
docker compose exec redis redis-cli ping
```

### 3. 忘记管理员密码

```bash
# 查看启动日志中的密码
docker compose logs sub2api | grep -i password

# 或者重新设置密码后重启
echo "ADMIN_PASSWORD=new_password" >> .env
docker compose up -d --force-recreate
```

### 4. 数据库备份

```bash
docker compose exec postgres pg_dump -U sub2api sub2api > backup_$(date +%Y%m%d).sql
```

### 5. 完全重置

**警告：会删除所有数据**

```bash
docker compose down -v
docker compose up -d
```

---

## 镜像源列表

| 源 | 地址 | 说明 |
|----|------|------|
| 南京大学 | ghcr.nju.edu.cn | ghcr.io 镜像 |
| 南京大学 | docker.nju.edu.cn | Docker Hub 镜像 |
| 网易 | hub-mirror.c.163.com | Docker Hub 镜像 |
| 百度 | mirror.baidubce.com | Docker Hub 镜像 |

---

## 快速命令参考

```bash
# 启动
docker compose up -d

# 停止
docker compose down

# 重启
docker compose restart

# 查看状态
docker compose ps

# 查看日志
docker compose logs -f sub2api

# 强制重建
docker compose up -d --force-recreate

# 拉取镜像
docker compose pull
```
