# ==============================================================================
# Stage: web — 前端构建（产物 web/dist + 依赖许可证文本）
# ==============================================================================
FROM node:20-alpine AS web
WORKDIR /build/web
COPY web/package.json web/pnpm-lock.yaml* web/pnpm-workspace.yaml* ./
# pnpm 11.24's sqlite store index hits disk I/O errors on overlayfs; pin pnpm 10
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    npm install -g pnpm@10 && (pnpm install --frozen-lockfile || pnpm install)
COPY web/ ./
RUN pnpm build && pnpm gen-licenses

# ==============================================================================
# Stage: backend — Go 后端构建（CGO，mattn/go-sqlite3 驱动）
# 前端产物经 web/embed.go（-tags embed）内嵌，dist 从 web 阶段拷贝；
# 第三方许可证文本在构建时生成并随镜像分发。
# ==============================================================================
FROM golang:1.27-bookworm AS backend
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY go.mod go.sum LICENSE ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/
COPY --from=web /build/web/dist/ web/dist/
COPY --from=web /build/web/licenses.txt third_party/web.txt
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=1 go run github.com/google/go-licenses/v2@v2.0.1 save ./... --save_path=third_party/go --force \
    && CGO_ENABLED=1 go build -tags embed -trimpath -ldflags "-s -w" -o /out/gitsune ./cmd/gitsune

# ==============================================================================
# Stage: runtime — 运行镜像（/app 布局，容器内用户 1000:1000）
# ==============================================================================
FROM debian:bookworm-slim
ARG VERSION=dev
ARG REVISION=unknown
ARG CREATED=""
LABEL org.opencontainers.image.title="Gitsune" \
      org.opencontainers.image.description="Self-hosted Git repo collection tool: collect GitHub/GitLab/Gitee repos and GitHub Trending chart, with a web UI" \
      org.opencontainers.image.url="https://github.com/ravenmk2/gitsune" \
      org.opencontainers.image.source="https://github.com/ravenmk2/gitsune" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.created="${CREATED}" \
      org.opencontainers.image.licenses="Apache-2.0"
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -g 1000 gitsune \
    && useradd -u 1000 -g gitsune -s /usr/sbin/nologin gitsune \
    && mkdir -p /app/data /app/conf \
    && chown -R 1000:1000 /app
# third_party only changes with the dependency tree; copy it before the
# binary (which changes on every commit) so it stays cached across code pushes.
COPY --from=backend --chown=1000:1000 /build/third_party/ /app/third_party/
COPY --from=backend --chown=1000:1000 /out/gitsune /app/gitsune
WORKDIR /app
USER 1000:1000
ENV GITSUNE_LISTEN_ADDR=:8080 \
    GITSUNE_DATA_PATH=/app/data
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/app/gitsune"]
