# Gitsune

Gitsune 是一个自托管的 Git 仓库收藏夹：把散落在 GitHub、GitLab、Gitee 上的仓库收录到一处统一管理，还能定时自动收录 GitHub Trending 榜单上的热门项目，配网页管理界面，支持多人使用。

## 功能一览

- **手动收录**：粘贴仓库链接，自动识别平台并抓取描述、语言、star、fork、许可证等信息
- **榜单采集**：定时收录 GitHub Trending（日 / 周 / 月三榜，默认每 6 小时一次），也可随时手动触发，执行记录可查
- **自动刷新**：定时重新抓取超过 7 天未刷新的仓库，保持数据新鲜
- **多用户**：管理员与普通用户两种角色；管理员负责用户管理、删除仓库、触发采集与系统设置
- **筛选检索**：按平台、语言、来源、关键词筛选，按 star 数排序

## 快速开始

### Docker（推荐）

直接使用 GitHub Packages 上的预构建镜像（`dev` 跟踪 master/main 最新构建，发布版本用 SemVer 版本号 tag `vX.Y.Z` 或 `latest`）：

```bash
docker run -d --name gitsune \
  -p 8080:8080 \
  -v gitsune-data:/app/data \
  -e GITSUNE_ADMIN_PASSWORD=your-password \
  ghcr.io/ravenmk2/gitsune:latest
```

也可以本地构建：

```bash
docker build -t gitsune .
# 可选：写入 OCI 标签的版本信息
docker build -t gitsune:1.0.0 \
  --build-arg VERSION=1.0.0 \
  --build-arg REVISION=$(git rev-parse --short HEAD) \
  --build-arg CREATED=$(date -u +%Y-%m-%dT%H:%M:%SZ) .
docker run -d --name gitsune \
  -p 8080:8080 \
  -v gitsune-data:/app/data \
  -e GITSUNE_ADMIN_PASSWORD=your-password \
  gitsune
```

打开 `http://localhost:8080`，用 `admin` / 你设置的密码登录。

### 从源码构建

需要 Go 1.25+ 和 Node 22+（pnpm）：

```bash
./build.sh        # 构建前端并打出内嵌前端的单二进制 dist/gitsune
./dist/gitsune
```

也可以手动分步执行：

```bash
cd web && pnpm install && pnpm build && cd ..
go build -o dist/gitsune ./cmd/gitsune
./dist/gitsune
```

构建产物是单个可执行文件，前端页面已内嵌其中，拷贝到任何机器即可运行。

## 使用说明

- **收录仓库**：仓库页点"收录仓库"，粘贴如 `https://github.com/gin-gonic/gin` 的链接即可；已收录的仓库不会被覆盖，点行内"刷新"按钮才会重新抓取更新数据
- **榜单任务**：任务记录页可手动触发"GitHub Trending"采集（只新增不覆盖已有仓库）或"Repo Refresh"（重新抓取超过 7 天未刷新的仓库），并查看每次执行的状态与影响数量
- **用户管理**：管理员在"用户管理"页创建账号、重置密码、调整角色；内置 `admin` 账号不可删除或降级
- **系统设置**：
  - *GitHub Token*：填入 personal access token 可提高 GitHub API 调用限额，页面只显示掩码
  - *Scheduled Tasks*：每种定时任务可单独启用/停用并配置 cron（标准 5 段表达式，默认 `0 */6 * * *`），保存后立即生效

## 配置项

均通过环境变量设置：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `GITSUNE_LISTEN_ADDR` | `:8080` | HTTP 监听地址 |
| `GITSUNE_DATA_PATH` | `./data`（Docker 中 `/app/data`） | 数据目录，数据库为其下 `gitsune.db` |
| `GITSUNE_ADMIN_PASSWORD` | `admin123` | 首次启动的内置管理员密码，请尽快修改 |
| `GITSUNE_GITHUB_TOKEN` | 空 | 初始 GitHub Token（之后可在设置页修改） |
| `GITSUNE_CRON` | `0 */6 * * *` | 定时采集周期（5 段 cron，默认每 6 小时） |
| `GITSUNE_LOG_LEVEL` | `info` | 日志级别 |

登录令牌（JWT）密钥无需配置：首次启动自动生成随机值并保存到数据目录的 `jwt_secret` 文件，重启后已签发的令牌仍然有效。

Docker 镜像内数据存放在 `/app/data`（已声明为数据卷），配置目录为 `/app/conf`，容器以非 root 用户运行。

## API

提供 RPC 风格的 HTTP API（规范见 `docs/conventions/api-design.md`）：全部 POST、`/api` 前缀、统一 `{success, data, error}` 响应。除健康检查与登录外，均需携带 `Authorization: Bearer <token>`。

## License

本项目以 [Apache License 2.0](LICENSE) 发布。第三方依赖的许可证文本不收录在仓库中，而是在 Docker 构建时自动生成并随镜像分发，镜像内位于 `/app/third_party/`。
