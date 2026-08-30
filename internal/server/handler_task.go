package server

import (
	"errors"

	"github.com/gin-gonic/gin"

	"gitsune/internal/model"
	"gitsune/internal/task"
)

type startTaskRequest struct {
	Type string `json:"type"`
}

// startTask POST /api/task/start（admin）：异步执行，立即返回任务日志 id。
func (s *Server) startTask(c *gin.Context) {
	var req startTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	id, err := s.runner.Start(req.Type, model.TriggerManual)
	if errors.Is(err, task.ErrInvalidTaskType) {
		fail(c, CodeValidationError, "type must be github_trending or gitee_gvp")
		return
	}
	if errors.Is(err, task.ErrTaskAlreadyRunning) {
		fail(c, CodeTaskAlreadyRunning, "a task of the same type is already running")
		return
	}
	if err != nil {
		fail(c, CodeInternalError, "failed to start task")
		return
	}
	ok(c, gin.H{"id": id})
}

type listTaskLogsRequest struct {
	Page        int    `json:"page"`
	Size        int    `json:"size"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	TriggerMode string `json:"trigger_mode"`
}

// listTaskLogs POST /api/task-log/list（admin）：分页，按 id 倒序。
func (s *Server) listTaskLogs(c *gin.Context) {
	var req listTaskLogsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	pageNum, size := normalizePage(req.Page, req.Size)
	items, total, err := s.store.ListTaskLogs(pageNum, size, req.Type, req.Status, req.TriggerMode)
	if err != nil {
		fail(c, CodeInternalError, "failed to query task logs")
		return
	}
	page(c, items, pageNum, size, total)
}
