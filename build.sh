#!/usr/bin/env bash
# Build the frontend and embed it into a single backend binary.
set -euo pipefail
cd "$(dirname "$0")"

# Frontend (output: web/dist, embedded via web/embed.go)
PNPM="pnpm"
if ! command -v pnpm >/dev/null 2>&1; then
  PNPM="corepack pnpm"
fi
cd web
$PNPM install
$PNPM build
cd ..

# Backend: -tags embed 内嵌前端产物（web/embed.go）；默认 CGO_ENABLED=0
#（纯 Go modernc.org/sqlite 驱动），本机有 C 工具链时可 CGO_ENABLED=1 ./build.sh。
bin="dist/gitsune"
if [ "$(go env GOOS)" = "windows" ]; then
  bin="dist/gitsune.exe"
fi
mkdir -p dist
CGO_ENABLED="${CGO_ENABLED:-0}" go build -tags embed -trimpath -ldflags "-s -w" -o "$bin" ./cmd/gitsune
echo "Built $bin"
