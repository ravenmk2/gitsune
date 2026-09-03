//go:build !embed

// Package web 的前端产物空占位：未传 -tags embed 的默认编译不内嵌前端，
// 使未构建 web/dist 时 go build / go vet / gopls 仍可工作；
// 此时访问页面返回 "frontend assets not built"（internal/server/static.go 兜底），API 不受影响。
package web

import "embed"

// DistFS 空 FS，正式版本由 embed.go（-tags embed）提供内嵌产物。
var DistFS embed.FS
