package task

import (
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"gitsune/internal/model"
	"gitsune/internal/store"
)

// DefaultCronExpr 默认每日调度表达式。
const DefaultCronExpr = "0 7 * * *"

// Scheduler 每日定时调度，支持按 setting 中的 cron 表达式重建。
type Scheduler struct {
	store  *store.Store
	runner *Runner

	mu   sync.Mutex
	cron *cron.Cron
}

// NewScheduler 创建调度器。
func NewScheduler(st *store.Store, runner *Runner) *Scheduler {
	return &Scheduler{store: st, runner: runner}
}

// Start 读取 setting 中的 cron 表达式并启动调度。
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startLocked()
}

// Reload 停止当前调度并按最新 setting 重建。
func (s *Scheduler) Reload() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron != nil {
		s.cron.Stop()
		s.cron = nil
	}
	return s.startLocked()
}

// Stop 停止调度。
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cron != nil {
		s.cron.Stop()
		s.cron = nil
	}
}

func (s *Scheduler) startLocked() error {
	expr := s.store.GetSettingOr("cron", DefaultCronExpr)
	c := cron.New()
	if _, err := c.AddFunc(expr, s.runDaily); err != nil {
		return err
	}
	c.Start()
	s.cron = c
	logrus.Infof("daily collection task scheduled: %s", expr)
	return nil
}

// runDaily 顺序执行 github_trending 再 gitee_gvp，各自独立写 task_log。
func (s *Scheduler) runDaily() {
	logrus.Info("scheduled collection started")
	s.runner.RunSync(model.TaskTypeGitHubTrending)
	s.runner.RunSync(model.TaskTypeGiteeGVP)
	logrus.Info("scheduled collection finished")
}

// ValidateCronExpr 校验 cron 表达式（5 段标准格式）。
func ValidateCronExpr(expr string) error {
	_, err := cron.ParseStandard(expr)
	return err
}
