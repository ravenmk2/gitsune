package store

import (
	"fmt"
	"strings"

	"gitsune/internal/model"
)

// CreateTaskLog 创建一条 running 状态的任务日志，返回 ID。
func (s *Store) CreateTaskLog(typ, triggerMode string) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO task_log (type, status, trigger_mode, started_at) VALUES (?, ?, ?, ?)`,
		typ, model.TaskStatusRunning, triggerMode, model.NowUTC(),
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FinishTaskLog 结束任务日志，写入状态、消息与新增数。
func (s *Store) FinishTaskLog(id int64, status, message string, addedCount int) error {
	_, err := s.db.Exec(
		`UPDATE task_log SET status = ?, message = ?, added_count = ?, finished_at = ? WHERE id = ?`,
		status, message, addedCount, model.NowUTC(), id,
	)
	return err
}

// HasRunningTask 判断指定类型是否存在 running 状态的任务。
func (s *Store) HasRunningTask(typ string) (bool, error) {
	var count int
	if err := s.db.Get(&count, `SELECT COUNT(*) FROM task_log WHERE type = ? AND status = ?`, typ, model.TaskStatusRunning); err != nil {
		return false, err
	}
	return count > 0, nil
}

// FailStaleRunningTasks 启动恢复：把残留 running 记录标记为 failed，返回影响条数。
func (s *Store) FailStaleRunningTasks() (int64, error) {
	res, err := s.db.Exec(
		`UPDATE task_log SET status = ?, message = ?, finished_at = ? WHERE status = ?`,
		model.TaskStatusFailed, "task interrupted by process restart", model.NowUTC(), model.TaskStatusRunning,
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ListTaskLogs 分页查询任务日志，按 id 倒序，支持 type/status/trigger_mode 筛选。
func (s *Store) ListTaskLogs(page, size int, typ, status, triggerMode string) ([]model.TaskLog, int, error) {
	var conds []string
	var args []any
	if typ != "" {
		conds = append(conds, `type = ?`)
		args = append(args, typ)
	}
	if status != "" {
		conds = append(conds, `status = ?`)
		args = append(args, status)
	}
	if triggerMode != "" {
		conds = append(conds, `trigger_mode = ?`)
		args = append(args, triggerMode)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := s.db.Get(&total, `SELECT COUNT(*) FROM task_log`+where, args...); err != nil {
		return nil, 0, err
	}
	items := []model.TaskLog{}
	query := fmt.Sprintf(`SELECT * FROM task_log%s ORDER BY id DESC LIMIT ? OFFSET ?`, where)
	args = append(args, size, (page-1)*size)
	if err := s.db.Select(&items, query, args...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
