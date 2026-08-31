package server

import (
	"slices"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitsune/internal/model"
	"gitsune/internal/task"
)

// getSetting POST /api/setting/get（admin）：github_token 掩码返回；tasks 为各任务类型的启用状态与生效 cron。
func (s *Server) getSetting(c *gin.Context) {
	token := s.store.GetSettingOr("github_token", "")
	tasks := gin.H{}
	for _, typ := range model.TaskTypes {
		enabled, cronExpr := s.scheduler.TaskConfig(typ)
		tasks[typ] = gin.H{"enabled": enabled, "cron": cronExpr}
	}
	ok(c, gin.H{
		"github_token": maskToken(token),
		"tasks":        tasks,
	})
}

type updateTaskConfig struct {
	Enabled *bool   `json:"enabled"`
	Cron    *string `json:"cron"`
}

type updateSettingRequest struct {
	GitHubToken *string                      `json:"github_token"`
	Tasks       map[string]updateTaskConfig `json:"tasks"`
}

// updateSetting POST /api/setting/update（admin）：仅更新传入的 key。
func (s *Server) updateSetting(c *gin.Context) {
	var req updateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	if len(req.Tasks) > 0 {
		for typ, cfg := range req.Tasks {
			if !slices.Contains(model.TaskTypes, typ) {
				fail(c, CodeValidationError, "unknown task type: "+typ)
				return
			}
			if cfg.Cron != nil {
				if err := task.ValidateCronExpr(*cfg.Cron); err != nil {
					fail(c, CodeValidationError, "invalid cron expression")
					return
				}
				if err := s.store.SetSetting(typ+"_cron", *cfg.Cron); err != nil {
					fail(c, CodeInternalError, "failed to save settings")
					return
				}
			}
			if cfg.Enabled != nil {
				if err := s.store.SetSetting(typ+"_enabled", strconv.FormatBool(*cfg.Enabled)); err != nil {
					fail(c, CodeInternalError, "failed to save settings")
					return
				}
			}
		}
		if err := s.scheduler.Reload(); err != nil {
			logrus.WithError(err).Error("failed to rebuild scheduler")
			fail(c, CodeInternalError, "failed to rebuild scheduler")
			return
		}
	}
	if req.GitHubToken != nil {
		// 空串表示清除
		if err := s.store.SetSetting("github_token", *req.GitHubToken); err != nil {
			fail(c, CodeInternalError, "failed to save settings")
			return
		}
	}
	ok(c, nil)
}

// maskToken 掩码 token："****"+末 4 位。
func maskToken(token string) string {
	if token == "" {
		return ""
	}
	if len(token) <= 4 {
		return "****"
	}
	return "****" + token[len(token)-4:]
}
