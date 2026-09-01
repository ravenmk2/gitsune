// Package server Gin 路由、中间件挂载与内嵌静态资源。
package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gitsune/internal/auth"
	"gitsune/internal/platform"
	"gitsune/internal/store"
	"gitsune/internal/task"
)

// Server HTTP 服务。
type Server struct {
	engine    *gin.Engine
	store     *store.Store
	auth      *auth.Service
	registry  *platform.Registry
	runner    *task.Runner
	scheduler *task.Scheduler
}

// New 创建服务并注册路由。
func New(st *store.Store, authSvc *auth.Service, registry *platform.Registry, runner *task.Runner, scheduler *task.Scheduler) *Server {
	gin.SetMode(gin.ReleaseMode)
	s := &Server{
		engine:    gin.New(),
		store:     st,
		auth:      authSvc,
		registry:  registry,
		runner:    runner,
		scheduler: scheduler,
	}
	s.engine.Use(gin.Recovery())
	s.routes()
	return s
}

// Handler 返回 HTTP Handler。
func (s *Server) Handler() http.Handler {
	return s.engine
}

func (s *Server) routes() {
	// 公开端点
	s.engine.GET("/api/health", s.health)
	s.engine.POST("/api/auth/login", s.login)

	// 需要登录
	authed := s.engine.Group("/api", s.auth.Middleware())
	{
		authed.POST("/auth/logout", s.logout)
		authed.POST("/me", s.me)
		authed.POST("/user/change-password", s.changePassword)

		authed.POST("/repo/collect", s.collectRepo)
		authed.POST("/repo/list", s.listRepos)
		authed.POST("/repo/get", s.getRepo)
		authed.POST("/repo/export", s.exportRepos)

		authed.POST("/stats/overview", s.statsOverview)
	}

	// 管理员专属
	admin := s.engine.Group("/api", s.auth.Middleware(), auth.AdminRequired())
	{
		admin.POST("/user/create", s.createUser)
		admin.POST("/user/list", s.listUsers)
		admin.POST("/user/update", s.updateUser)
		admin.POST("/user/delete", s.deleteUser)
		admin.POST("/user/reset-password", s.resetPassword)

		admin.POST("/repo/refresh", s.refreshRepo)
		admin.POST("/repo/delete", s.deleteRepo)
		admin.POST("/repo/import", s.importRepos)

		admin.POST("/task/start", s.startTask)
		admin.POST("/task-log/list", s.listTaskLogs)

		admin.POST("/setting/get", s.getSetting)
		admin.POST("/setting/update", s.updateSetting)
	}

	s.registerStatic()
}

func (s *Server) health(c *gin.Context) {
	ok(c, gin.H{"status": "ok"})
}
