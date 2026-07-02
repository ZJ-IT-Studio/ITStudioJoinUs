# IT Studio 校园招新系统

一个可长期复用的校园招新站：公开宣传页、学生报名/查询终端，以及带批次归档、动态表单和审核状态流的管理后台。

## 技术结构

- `web/`：Vue 3、TypeScript、Vite、Pinia、Anime.js，以及按 Vue Bits copy-in 模式本地维护的动态文字/背景组件。
- `server/`：Go、Chi、SQLite；提供 API、认证、邮件、私有图片和前端静态资源。
- `/data`：运行时 SQLite、上传图片和备份目录；容器重建不会丢失。

## 本地开发

需要 Node.js 24+ 和 Go 1.25+。

```powershell
Copy-Item .env.example .env
npm --prefix web install
go run ./server/cmd/server
```

另开终端启动前端（Vite 会把 API 代理到 `localhost:8080`）：

```powershell
npm --prefix web run dev
```

访问 `http://localhost:5173`；后台为 `http://localhost:5173/admin/login`。`.env` 中的 `ADMIN_EMAIL` 和 `ADMIN_PASSWORD` 对应受保护的超级管理员，可在后台创建“可编辑”和“仅查看”用户。示例批次默认处于草稿状态，需要在后台检查表单后手动开放。

未配置 SMTP 时，服务端会把验证码/重置邮件写入日志，验证码也会在开发环境的 API 响应中返回。生产环境务必配置 SMTP，并设置强管理员密码。

## 单容器部署

```powershell
Copy-Item .env.example .env
# 修改 .env，至少设置管理员密码、站点 URL 和 SMTP
docker compose up -d --build
```

访问 `http://localhost:8080`。镜像使用多阶段构建，最终容器只有 Go 服务；Vue 产物嵌入二进制，容器以非 root 用户运行。

建议生产配置：

```dotenv
APP_BASE_URL=https://join.example.edu
COOKIE_SECURE=true
ADMIN_EMAIL=owner@example.edu
ADMIN_PASSWORD=请替换为高强度初始密码
SMTP_HOST=smtp.example.edu
SMTP_PORT=587
SMTP_USER=...
SMTP_PASSWORD=...
SMTP_FROM=IT Studio <noreply@example.edu>
```

反向代理应把真实客户端地址传入 `X-Forwarded-For`，并仅通过 HTTPS 暴露服务。

## 数据备份与恢复

SQLite 使用 WAL 模式。最稳妥的备份方式是在短暂停止容器后复制整个 `data` 目录：

```powershell
docker compose stop
Copy-Item -Recurse data "backup-$(Get-Date -Format yyyyMMdd-HHmmss)"
docker compose start
```

恢复时停止容器，将备份中的 `itstudio.db`、可能存在的 WAL 文件和 `uploads/` 一并还原，再启动容器。不要只备份数据库而遗漏私有图片。

## 业务规则

- 同一时间只能开放一个批次；批次第一次开放后，动态表单永久锁定。
- 学生通过“学号 + 查询密码”查看报名；邮箱只用于首次验证和找回密码。
- 开放期内可以撤回并创建新版本；旧版本和审计记录不会删除。
- 审核状态可自定义；已被报名引用的状态必须先迁移，才能删除。
- 超级管理员由环境变量确定且不可降级，负责后台用户管理；可编辑用户能修改业务数据但不能管理用户；仅查看用户可以查看和导出，但所有写接口都会被服务端拒绝。
- 图片只接受 JPEG、PNG、WebP，最大 5 MB；服务端重新编码并去除原始元数据。

## 验证命令

```powershell
go test ./...
npm --prefix web test
npm --prefix web run build
docker build -t itstudio-join-us .
```
