// Package web 内嵌前端构建产物。
// dist 由 web 前端构建生成（pnpm build）；仓库中保留 dist/.gitkeep 占位，
// 使未构建前端时 go build 仍可编译。
package web

import "embed"

// DistFS 内嵌的前端静态资源（SPA，入口为 dist/index.html）。
//
//go:embed all:dist
var DistFS embed.FS
