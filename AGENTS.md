# Gitsune 项目指南

Git 仓库收录工具：收录 GitHub / GitLab / Gitee 仓库，定时（默认每 6 小时）或手动采集 GitHub Trending，带多用户网页管理界面。后端 Go（Gin + sqlx + SQLite + Logrus），前端 Vue3 + Element Plus + Vite，构建为内嵌前端的单二进制，支持 Docker 部署。

## 目录结构

```
cmd/gitsune/main.go    入口：config → logrus → store → registry → runner/scheduler → gin
internal/config/       环境变量加载（GITSUNE_*）
internal/model/        数据模型（db tag + json tag，时间为 RFC3339 UTC 字符串）
internal/store/        sqlx + 手写 SQL；schema.sql 经 go:embed 在启动时建表
internal/auth/         JWT HS256、bcrypt、Gin 中间件（登录校验、AdminRequired）
internal/platform/     平台采集器：Collector 接口 + Registry；github/gitlab/gitee 实现
internal/task/         任务 runner（per-type 互斥锁）+ cron 调度器
internal/server/       Gin 路由、响应信封 helpers、各模块 handler、SPA 静态服务
web/                   前端源码；embed.go（-tags embed 时）用 //go:embed all:dist 内嵌构建产物，embed_stub.go 为默认空占位
docs/conventions/      API 设计规范（必须遵守）
```

## 构建与验证

```bash
# 一键构建（前端 + 单二进制，等价于下面两步）
./build.sh

# 前端（产物输出 web/dist，正式发布构建经 -tags embed 内嵌进二进制）
cd web && pnpm install && pnpm build && cd ..

# 后端
go vet ./...
go build -tags embed -o dist/gitsune ./cmd/gitsune  # 内嵌前端的正式二进制；本机有 gcc 时走 mattn/go-sqlite3
CGO_ENABLED=0 go build ./...                        # 无 CGO 时自动回退 modernc.org/sqlite

# 说明：web/embed.go 带 //go:build embed 约束，只有 -tags embed 才编译它（要求 web/dist 已构建）；
# 默认编译走 web/embed_stub.go 空占位，未构建前端也能编译，页面返回 "frontend assets not built"，API 正常。

# Docker
docker build -t gitsune .
```

CI：`.github/workflows/build-image.yml`，push 到 master/main 时先跑测试（前端构建 → go vet + CGO=0 `-tags embed` 构建）再构建镜像推送 `ghcr.io/ravenmk2/gitsune:dev`；打 SemVer 版本 tag（`vX.Y.Z`，如 `v1.4.2`）时推送 `ghcr.io/ravenmk2/gitsune:vX.Y.Z` 与 `:latest`。

改动后必须跑 `go vet ./...` 和 `CGO_ENABLED=0 go build ./...`；涉及 API 的改动需启动服务用 curl 实测对应端点。

## 许可证与第三方声明

项目以 Apache-2.0 发布（根目录 `LICENSE`）。第三方依赖的许可证文本**不进仓库**（`.gitignore` 已忽略 `third_party/` 与 `web/licenses.txt`），由 Dockerfile 在构建时生成并随镜像分发到 `/app/third_party/`：

- 后端：backend 阶段跑 `go-licenses save ./... --save_path=third_party/go`（CGO=1，与镜像二进制依赖树一致）
- 前端：web 阶段跑 `pnpm gen-licenses`（`web/scripts/gen-licenses.cjs` 遍历 pnpm 虚拟仓库收集全部依赖的 LICENSE 文本，输出 `web/licenses.txt` → `third_party/web.txt`）

本地如需检查依赖许可证，用 `go-licenses check --disallowed_types=forbidden,restricted,reciprocal ./...` 与 `pnpm gen-licenses` 即可。


## 后端约定

- **API 规范**：严格遵循 `docs/conventions/api-design.md`。一律 POST、`/api` 前缀、kebab-case 路径、统一 HTTP 200 + `{success, data, error}` 信封、错误码 UPPER_SNAKE_CASE、分页结构 `{items, page, size, total, page_count}`、时间 RFC3339 UTC。新增端点挂在 `internal/server/server.go` 的 `routes()` 中相应分组（公开 / 登录 / admin）
- **数据库**：sqlx + 手写 SQL，不用 ORM。表名单数；新增/变更表结构改 `internal/store/schema.sql`（`CREATE TABLE IF NOT EXISTS`，无迁移机制，老库需手工 `ALTER TABLE` 或删库重建）并在 `internal/store/` 对应文件加查询函数。SQLite 驱动按 build tag 二选一：`sqlite_cgo.go`（mattn/go-sqlite3）/ `sqlite_nocgo.go`（modernc.org/sqlite），新增驱动相关代码两个文件都要考虑
- **命名**：采集"任务"统一用 task；执行历史表 `task_log`；任务类型值 `github_trending` / `repo_refresh`；避免使用 SQL 保留字做列名（如 `trigger`，用 `trigger_mode`）
- **权限**：内置 admin（username == "admin"）不可删除、不可降级，违反返回 `ADMIN_USER_IMMUTABLE`；admin 接口挂 `AdminRequired()` 分组
- **平台扩展**：实现 `platform.Collector` 接口（`Name`/`Match`/`FetchRepo`）并在 registry 注册即可支持新平台；HTTP 请求统一带 1 分钟超时，UA 使用 Chrome 浏览器标识（见 `newRequest`）
- **任务执行**：新任务类型在 `internal/task/runner.go` 注册执行函数并加入 `model.TaskTypes`；同类型并发触发必须返回 `TASK_ALREADY_RUNNING`；panic 必须 recover 并落 `task_log.message`；github_trending 抓取遇网络连接类错误原地重试最多 1000 次（间隔 5s，受任务超时约束）；采集任务与手动录入（repo/collect）只新增仓库、不覆盖已有记录，已有数据的更新只走 repo/refresh 与 repo_refresh 任务（刷新 `refreshed_at` 老于 7 天的仓库）
- **任务调度**：每种任务独立配置，设置项为 `<type>_enabled`（默认 true）与 `<type>_cron`（默认 `0 */6 * * *`；github_trending 额外兜底旧的统一设置项 `cron`）；调度器按 `model.TaskTypes` 逐个建 cron entry，写冲突靠单连接串行化（`SetMaxOpenConns(1)`）与 per-type 互斥锁规避，采集只增、刷新只改，无逻辑冲突
- **日志**：统一用 logrus，不用标准库 log / fmt.Print

## 前端约定

- axios 实例在 `web/src/api/index.js`：baseURL `/api`、Bearer token（localStorage `gitsune_token`）、401 清 token 跳登录、`success===false` 统一 `ElMessage.error`；新接口在这里加函数
- 路由守卫在 `web/src/router/index.js`：admin 页面加 `meta.requiresAdmin`
- 时间显示用 dayjs 转本地时区
- 开发调试：`pnpm dev`，vite 已代理 `/api` → `http://localhost:8080`
- **前端内嵌走 build tag**：`-tags embed` 才编译 `web/embed.go`（`go:embed all:dist`，要求前端已构建），默认编译用 `web/embed_stub.go` 空占位；发布构建（build.sh / Dockerfile / CI 镜像）必须带 `-tags embed`，否则会产出无前端的二进制

## 配置（环境变量）

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `GITSUNE_LISTEN_ADDR` | `:8080` | HTTP 监听地址 |
| `GITSUNE_DATA_PATH` | `./data`（Docker 中 `/app/data`） | 数据目录，SQLite 为其下 `gitsune.db` |
| `GITSUNE_ADMIN_PASSWORD` | `admin123` | 首次启动播种的内置 admin 密码 |
| `GITSUNE_GITHUB_TOKEN` | 空 | 播种到 setting，提高 GitHub API 限额 |
| `GITSUNE_CRON` | `0 */6 * * *` | 播种到旧版统一设置项 `cron`，作为 github_trending 的默认 cron（各任务实际调度以设置页 `<type>_cron` / `<type>_enabled` 为准） |
| `GITSUNE_LOG_LEVEL` | `info` | 日志级别 |

JWT 密钥不走环境变量：首次启动生成随机值并持久化到数据目录 `jwt_secret` 文件（0600），重启后 token 不失效；逻辑在 `config.ResolveJWTSecret()`。

Docker 镜像：backend 阶段 `golang:1.27-bookworm` + `CGO_ENABLED=1`（mattn/go-sqlite3；镜像内 Go 版本领先 go.mod 声明属正常，新工具链向下兼容），运行镜像 `debian:bookworm-slim`（含 ca-certificates）。布局：`/app/gitsune`（二进制）、`/app/data`（数据卷）、`/app/conf`，以 `1000:1000` 用户运行；改 Dockerfile 时保持该布局与用户。
