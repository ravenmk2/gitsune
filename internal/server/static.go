package server

import (
	"io/fs"
	"net/http"
	"strings"

	"gitsune/web"

	"github.com/gin-gonic/gin"
)

// registerStatic 挂载静态资源：/api 之外的 GET 路径先尝试静态文件，未命中回退 index.html（SPA）。
// 前端产物由 gitsune/web 包内嵌（web/dist，见 web/embed.go）。
func (s *Server) registerStatic() {
	sub, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	s.engine.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api") {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"data":    nil,
				"error":   gin.H{"code": "NOT_FOUND", "message": "api not found"},
			})
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"data":    nil,
				"error":   gin.H{"code": "NOT_FOUND", "message": "resource not found"},
			})
			return
		}
		name := strings.TrimPrefix(p, "/")
		if name != "" && !strings.Contains(name, "..") {
			if f, err := sub.Open(name); err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		// SPA 回退
		data, err := fs.ReadFile(sub, "index.html")
		if err != nil {
			c.String(http.StatusNotFound, "frontend assets not built")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})
}
