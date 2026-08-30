# Frontend build
FROM node:20-alpine AS web
WORKDIR /build/web
COPY web/package.json web/pnpm-lock.yaml* web/pnpm-workspace.yaml* ./
# pnpm 11.24's sqlite store index hits disk I/O errors on overlayfs; pin pnpm 10
RUN npm install -g pnpm@10 && (pnpm install --frozen-lockfile || pnpm install)
COPY web/ ./
RUN pnpm build && pnpm gen-licenses

# Backend build with CGO (gcc included) -> mattn/go-sqlite3 driver.
# The frontend bundle is embedded via web/embed.go, so dist is copied from the web stage.
# Third-party license texts are generated during the build and shipped in the image.
FROM golang:1.27-bookworm AS backend
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY go.mod go.sum LICENSE ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/embed.go web/
COPY --from=web /build/web/dist/ web/dist/
COPY --from=web /build/web/licenses.txt third_party/web.txt
RUN CGO_ENABLED=1 go run github.com/google/go-licenses/v2@latest save ./... --save_path=third_party/go --force \
    && CGO_ENABLED=1 go build -trimpath -ldflags "-s -w" -o /out/gitsune ./cmd/gitsune

# Runtime (/app layout, container user 1000:1000)
FROM debian:bookworm-slim
ARG VERSION=dev
ARG REVISION=unknown
ARG CREATED=""
LABEL org.opencontainers.image.title="Gitsune" \
      org.opencontainers.image.description="Self-hosted Git repo collection tool: collect GitHub/GitLab/Gitee repos and daily GitHub Trending / Gitee GVP charts, with a web UI" \
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
COPY --from=backend --chown=1000:1000 /out/gitsune /app/gitsune
COPY --from=backend --chown=1000:1000 /build/third_party/ /app/third_party/
WORKDIR /app
USER 1000:1000
ENV GITSUNE_LISTEN_ADDR=:8080 \
    GITSUNE_DATA_PATH=/app/data
EXPOSE 8080
VOLUME ["/app/data"]
ENTRYPOINT ["/app/gitsune"]
