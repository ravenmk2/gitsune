package server

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"gitsune/internal/task"
)

// getSetting POST /api/setting/get（admin）：github_token 掩码返回。
func (s *Server) getSetting(c *gin.Context) {
	token := s.store.GetSettingOr("github_token", "")
	ok(c, gin.H{
		"github_token": maskToken(token),
		"cron":         s.store.GetSettingOr("cron", task.DefaultCronExpr),
	})
}

type updateSettingRequest struct {
	GitHubToken *string `json:"github_token"`
	Cron        *string `json:"cron"`
}

// updateSetting POST /api/setting/update（admin）：仅更新传入的 key。
func (s *Server) updateSetting(c *gin.Context) {
	var req updateSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	if req.Cron != nil {
		if err := task.ValidateCronExpr(*req.Cron); err != nil {
			fail(c, CodeValidationError, "invalid cron expression")
			return
		}
		if err := s.store.SetSetting("cron", *req.Cron); err != nil {
			fail(c, CodeInternalError, "failed to save settings")
			return
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
