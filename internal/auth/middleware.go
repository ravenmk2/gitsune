package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"gitsune/internal/model"
)

// 上下文键
const (
	CtxUserID   = "user_id"
	CtxUsername = "username"
	CtxRole     = "role"
)

func abort(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"success": false,
		"data":    nil,
		"error":   gin.H{"code": code, "message": message},
	})
}

// Middleware 校验 Authorization: Bearer <token>，失败返回 401 UNAUTHORIZED。
func (s *Service) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok {
			abort(c, http.StatusUnauthorized, "UNAUTHORIZED", "not logged in or session expired")
			return
		}
		claims, err := s.ParseToken(strings.TrimSpace(token))
		if err != nil {
			abort(c, http.StatusUnauthorized, "UNAUTHORIZED", "not logged in or session expired")
			return
		}
		c.Set(CtxUserID, claims.UserID)
		c.Set(CtxUsername, claims.Username)
		c.Set(CtxRole, claims.Role)
		c.Next()
	}
}

// AdminRequired 校验当前用户为 admin，否则返回 403 FORBIDDEN。
func AdminRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetString(CtxRole) != model.RoleAdmin {
			abort(c, http.StatusForbidden, "FORBIDDEN", "admin permission required")
			return
		}
		c.Next()
	}
}
