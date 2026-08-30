// Package task 采集任务执行器与定时调度。
package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"gitsune/internal/model"
	"gitsune/internal/platform"
	"gitsune/internal/store"
)

// ErrTaskAlreadyRunning 同类型任务正在运行。
var ErrTaskAlreadyRunning = errors.New("task already running")

// ErrInvalidTaskType 未知任务类型。
var ErrInvalidTaskType = errors.New("invalid task type")

// taskExecTimeout 单个任务最长执行时间。
const taskExecTimeout = 10 * time.Minute

// Runner 任务执行器，手动触发与定时调度共用。
type Runner struct {
	store  *store.Store
	github *platform.GitHubCollector
	gitee  *platform.GiteeCollector
	locks  sync.Map // type -> *sync.Mutex，per-type 互斥锁双保险
}

// NewRunner 创建任务执行器。
func NewRunner(st *store.Store, github *platform.GitHubCollector, gitee *platform.GiteeCollector) *Runner {
	return &Runner{store: st, github: github, gitee: gitee}
}

func (r *Runner) lockFor(typ string) *sync.Mutex {
	v, _ := r.locks.LoadOrStore(typ, &sync.Mutex{})
	return v.(*sync.Mutex)
}

// Start 手动触发：创建 running 日志并异步执行，立即返回日志 ID。
func (r *Runner) Start(typ, triggerMode string) (int64, error) {
	if typ != model.TaskTypeGitHubTrending && typ != model.TaskTypeGiteeGVP {
		return 0, ErrInvalidTaskType
	}
	running, err := r.store.HasRunningTask(typ)
	if err != nil {
		return 0, err
	}
	if running {
		return 0, ErrTaskAlreadyRunning
	}
	id, err := r.store.CreateTaskLog(typ, triggerMode)
	if err != nil {
		return 0, err
	}
	go r.execute(id, typ)
	return id, nil
}

// RunSync 定时调度同步执行（cron 回调中顺序调用）。
func (r *Runner) RunSync(typ string) {
	running, err := r.store.HasRunningTask(typ)
	if err != nil {
		logrus.WithError(err).Errorf("task %s: failed to query running status", typ)
		return
	}
	if running {
		logrus.Warnf("task %s: already running, skipping this scheduled run", typ)
		return
	}
	id, err := r.store.CreateTaskLog(typ, model.TriggerAuto)
	if err != nil {
		logrus.WithError(err).Errorf("task %s: failed to create task log", typ)
		return
	}
	r.execute(id, typ)
}

// execute 执行任务并把结果写入 task_log，panic 不影响后续任务。
func (r *Runner) execute(id int64, typ string) {
	mu := r.lockFor(typ)
	mu.Lock()
	defer mu.Unlock()
	defer func() {
		if rec := recover(); rec != nil {
			msg := fmt.Sprintf("task panicked: %v", rec)
			logrus.Errorf("task %s (id=%d): %s", typ, id, msg)
			if err := r.store.FinishTaskLog(id, model.TaskStatusFailed, msg, 0); err != nil {
				logrus.WithError(err).Errorf("task %s (id=%d): failed to update task log", typ, id)
			}
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), taskExecTimeout)
	defer cancel()

	logrus.Infof("task %s (id=%d): started", typ, id)
	var added int
	var err error
	switch typ {
	case model.TaskTypeGitHubTrending:
		added, err = r.runGitHubTrending(ctx)
	case model.TaskTypeGiteeGVP:
		added, err = r.runGiteeGVP(ctx)
	default:
		err = ErrInvalidTaskType
	}

	if err != nil {
		logrus.WithError(err).Errorf("task %s (id=%d): execution failed", typ, id)
		if uerr := r.store.FinishTaskLog(id, model.TaskStatusFailed, err.Error(), added); uerr != nil {
			logrus.WithError(uerr).Errorf("task %s (id=%d): failed to update task log", typ, id)
		}
		return
	}
	if uerr := r.store.FinishTaskLog(id, model.TaskStatusSuccess, "", added); uerr != nil {
		logrus.WithError(uerr).Errorf("task %s (id=%d): failed to update task log", typ, id)
	}
	logrus.Infof("task %s (id=%d): succeeded, %d new repo(s) added", typ, id, added)
}

// runGitHubTrending 依次抓 daily/weekly/monthly 三榜，合并后 upsert，返回新增条数。
func (r *Runner) runGitHubTrending(ctx context.Context) (int, error) {
	seen := map[string]bool{}
	added := 0
	for _, since := range []string{"daily", "weekly", "monthly"} {
		repos, err := r.github.FetchTrending(ctx, since)
		if err != nil {
			return added, fmt.Errorf("failed to fetch trending (%s): %w", since, err)
		}
		for _, ri := range repos {
			key := ri.Owner + "/" + ri.Name
			if seen[key] {
				continue
			}
			seen[key] = true
			created, err := r.upsert(ri, model.SourceTrending)
			if err != nil {
				return added, err
			}
			if created {
				added++
			}
		}
	}
	return added, nil
}

// runGiteeGVP 抓取 Gitee GVP 列表并 upsert，返回新增条数。
func (r *Runner) runGiteeGVP(ctx context.Context) (int, error) {
	repos, err := r.gitee.FetchGVP(ctx)
	if err != nil {
		return 0, err
	}
	added := 0
	for _, ri := range repos {
		created, err := r.upsert(ri, model.SourceGVP)
		if err != nil {
			return added, err
		}
		if created {
			added++
		}
	}
	return added, nil
}

func (r *Runner) upsert(info *platform.RepoInfo, source string) (bool, error) {
	_, created, err := r.store.UpsertRepo(&model.Repo{
		Platform:    info.Platform,
		Owner:       info.Owner,
		Name:        info.Name,
		URL:         info.URL,
		Description: info.Description,
		Language:    info.Language,
		Stars:       info.Stars,
		Forks:       info.Forks,
		License:     info.License,
		Source:      source,
	})
	if err != nil {
		return false, fmt.Errorf("failed to upsert %s/%s: %w", info.Owner, info.Name, err)
	}
	return created, nil
}
