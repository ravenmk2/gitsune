//go:build embed

// Package web 内嵌前端构建产物。
// 本文件仅在显式传入 -tags embed 时参与编译（正式发布构建：build.sh / Dockerfile / CI），
// go:embed 要求此时 web/dist 已构建；默认编译走 embed_stub.go 的空占位。
package web

import "embed"

// DistFS 内嵌的前端静态资源（SPA，入口为 dist/index.html）。
//
//go:embed all:dist
var DistFS embed.FS
