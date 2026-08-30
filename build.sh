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
# vite build empties outDir; keep the placeholder so fresh clones can compile (go:embed)
touch dist/.gitkeep
cd ..

# Backend: default to CGO_ENABLED=0 (pure-Go modernc.org/sqlite driver),
# override with e.g. CGO_ENABLED=1 ./build.sh when a C toolchain is available.
bin="dist/gitsune"
if [ "$(go env GOOS)" = "windows" ]; then
  bin="dist/gitsune.exe"
fi
mkdir -p dist
CGO_ENABLED="${CGO_ENABLED:-0}" go build -trimpath -ldflags "-s -w" -o "$bin" ./cmd/gitsune
echo "Built $bin"
