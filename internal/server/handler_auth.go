package server

import (
	"database/sql"
	"strings"

	"github.com/gin-gonic/gin"

	"gitsune/internal/auth"
)

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// login POST /api/auth/login → {token, user:{id, username, role}}
func (s *Server) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		fail(c, CodeValidationError, "username and password are required")
		return
	}
	user, err := s.store.GetUserByUsername(req.Username)
	if err == sql.ErrNoRows || (err == nil && !auth.CheckPassword(user.PasswordHash, req.Password)) {
		c.JSON(401, gin.H{
			"success": false,
			"data":    nil,
			"error":   gin.H{"code": CodeUnauthorized, "message": "invalid username or password"},
		})
		return
	}
	if err != nil {
		fail(c, CodeInternalError, "login failed")
		return
	}
	token, err := s.auth.GenerateToken(user)
	if err != nil {
		fail(c, CodeInternalError, "failed to issue token")
		return
	}
	ok(c, gin.H{
		"token": token,
		"user":  gin.H{"id": user.ID, "username": user.Username, "role": user.Role},
	})
}

// logout POST /api/auth/logout → data: null（JWT 无状态，仅返回成功）
func (s *Server) logout(c *gin.Context) {
	ok(c, nil)
}

// me POST /api/me → {id, username, role}
func (s *Server) me(c *gin.Context) {
	ok(c, gin.H{
		"id":       c.GetInt64(auth.CtxUserID),
		"username": c.GetString(auth.CtxUsername),
		"role":     c.GetString(auth.CtxRole),
	})
}
