// Gitsune 入口：初始化配置/日志/存储/调度器/HTTP 服务，支持优雅关闭。
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"gitsune/internal/auth"
	"gitsune/internal/config"
	"gitsune/internal/platform"
	"gitsune/internal/server"
	"gitsune/internal/store"
	"gitsune/internal/task"
)

func main() {
	cfg := config.Load()
	initLog(cfg.LogLevel)

	// JWT secret：从数据目录读取，首次运行生成并持久化
	cfg.ResolveJWTSecret()

	st, err := store.Open(cfg.DBPath())
	if err != nil {
		logrus.Fatalf("failed to open database: %v", err)
	}
	defer st.Close()

	if err := st.SeedAdmin(cfg.AdminPassword); err != nil {
		logrus.Fatalf("failed to seed built-in admin: %v", err)
	}
	seedSettings(st, cfg)

	// 启动恢复：残留 running 任务标记 failed
	if n, err := st.FailStaleRunningTasks(); err != nil {
		logrus.WithError(err).Error("failed to mark stale running tasks as failed")
	} else if n > 0 {
		logrus.Warnf("marked %d stale running task(s) as failed", n)
	}

	// 平台采集器：GitHub token 从 setting 动态读取
	registry := platform.NewRegistry()
	githubCollector := platform.NewGitHub(func() string { return st.GetSettingOr("github_token", "") })
	giteeCollector := platform.NewGitee()
	registry.Register(githubCollector)
	registry.Register(platform.NewGitLab())
	registry.Register(giteeCollector)

	runner := task.NewRunner(st, githubCollector, registry)
	scheduler := task.NewScheduler(st, runner)
	if err := scheduler.Start(); err != nil {
		logrus.WithError(err).Error("failed to start scheduler")
	}
	defer scheduler.Stop()

	srv := server.New(st, auth.NewService(cfg.JWTSecret), registry, runner, scheduler)
	httpSrv := &http.Server{Addr: cfg.Addr, Handler: srv.Handler()}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logrus.Infof("Gitsune started, listening on %s", cfg.Addr)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logrus.Fatalf("HTTP server failed: %v", err)
		}
	}()

	<-ctx.Done()
	logrus.Info("shutdown signal received, shutting down gracefully")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logrus.WithError(err).Error("HTTP server shutdown error")
	}
	logrus.Info("shutdown complete")
}

// initLog 初始化 logrus。
func initLog(level string) {
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		lvl = logrus.InfoLevel
	}
	logrus.SetLevel(lvl)
}

// seedSettings 播种设置项：env 有值且 setting 未设置过时写入。
func seedSettings(st *store.Store, cfg *config.Config) {
	if cfg.GitHubToken != "" && !st.HasSetting("github_token") {
		if err := st.SetSetting("github_token", cfg.GitHubToken); err != nil {
			logrus.WithError(err).Error("failed to seed github_token setting")
		} else {
			logrus.Info("seeded github_token setting from GITSUNE_GITHUB_TOKEN")
		}
	}
	if !st.HasSetting("cron") {
		expr := cfg.Cron
		if expr == "" {
			expr = task.DefaultCronExpr
		}
		if err := task.ValidateCronExpr(expr); err != nil {
			logrus.WithError(err).Warnf("invalid GITSUNE_CRON expression %q, falling back to default %q", expr, task.DefaultCronExpr)
			expr = task.DefaultCronExpr
		}
		if err := st.SetSetting("cron", expr); err != nil {
			logrus.WithError(err).Error("failed to seed cron setting")
		}
	}
}
