# 旧 sub2api 部署资料 —— 请勿使用 (LEGACY — DO NOT USE)

本目录是融合前「星算 / sub2api」时代的部署脚本与 runbook(Caddyfile、`docker-deploy.sh`、`build_image.sh` 等)。

融合(方案 B,new-api 为唯一后端)之后:
- 仓库根的 `Dockerfile` 已替换为 **new-api 后端**的构建配方;
- `deploy/build_image.sh` 会用根 `Dockerfile` 构建并打 `sub2api:latest` 标签 —— 这会把 new-api 镜像**错误地伪装成 sub2api**。

因此:
- **禁止执行 `deploy/` 下任何脚本**,也不要据此目录上线;
- 统一的融合部署(单镜像)将在 **P7「打包」** 阶段重做。

此目录暂作历史资料保留(避免破坏现有 runbook 引用),**不代表当前部署方式**。
