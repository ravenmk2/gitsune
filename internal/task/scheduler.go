package task

import (
	"sync"

	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"

	"gitsune/internal/model"
	"gitsune/internal/store"
)

// DefaultCronExpr 默认定时调度表达式（每 6 小时一次）。
const DefaultCronExpr = "0 */6 * * *"

// Scheduler 定时调度，支持按 setting 中的 cron 表达式重建。
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

// TaskConfig 返回任务的启用状态与生效 cron 表达式。
// 启用开关读 <type>_enabled（默认 true）；cron 读 <type>_cron，
// github_trending 额外兜底旧的统一设置项 cron（再兜底默认表达式）。
func (s *Scheduler) TaskConfig(typ string) (enabled bool, cronExpr string) {
	enabled = s.store.GetSettingOr(typ+"_enabled", "true") == "true"
	def := DefaultCronExpr
	if typ == model.TaskTypeGitHubTrending {
		def = s.store.GetSettingOr("cron", DefaultCronExpr)
	}
	return enabled, s.store.GetSettingOr(typ+"_cron", def)
}

func (s *Scheduler) startLocked() error {
	c := cron.New()
	for _, typ := range model.TaskTypes {
		enabled, expr := s.TaskConfig(typ)
		if !enabled {
			logrus.Infof("task %s: disabled, skipping schedule", typ)
			continue
		}
		typ := typ
		if _, err := c.AddFunc(expr, func() { s.runScheduled(typ) }); err != nil {
			return err
		}
		logrus.Infof("task %s scheduled: %s", typ, expr)
	}
	c.Start()
	s.cron = c
	return nil
}

// runScheduled 定时回调：执行指定任务，写 task_log。
func (s *Scheduler) runScheduled(typ string) {
	logrus.Infof("scheduled task %s started", typ)
	s.runner.RunSync(typ)
	logrus.Infof("scheduled task %s finished", typ)
}

// ValidateCronExpr 校验 cron 表达式（5 段标准格式）。
func ValidateCronExpr(expr string) error {
	_, err := cron.ParseStandard(expr)
	return err
}
