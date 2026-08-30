package server

import (
	"database/sql"
	"strings"

	"github.com/gin-gonic/gin"

	"gitsune/internal/auth"
	"gitsune/internal/model"
)

// builtInAdmin 内置管理员用户名。
const builtInAdmin = "admin"

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// createUser POST /api/user/create（admin）
func (s *Server) createUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		fail(c, CodeValidationError, "username and password are required")
		return
	}
	if req.Role != model.RoleAdmin && req.Role != model.RoleUser {
		fail(c, CodeValidationError, "role must be admin or user")
		return
	}
	if _, err := s.store.GetUserByUsername(req.Username); err == nil {
		fail(c, CodeUsernameExists, "username already exists")
		return
	} else if err != sql.ErrNoRows {
		fail(c, CodeInternalError, "failed to create user")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		fail(c, CodeInternalError, "failed to create user")
		return
	}
	user, err := s.store.CreateUser(req.Username, hash, req.Role)
	if err != nil {
		fail(c, CodeInternalError, "failed to create user")
		return
	}
	ok(c, user)
}

type listUsersRequest struct {
	Page    int    `json:"page"`
	Size    int    `json:"size"`
	Keyword string `json:"keyword"`
}

// listUsers POST /api/user/list（admin）
func (s *Server) listUsers(c *gin.Context) {
	var req listUsersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	pageNum, size := normalizePage(req.Page, req.Size)
	items, total, err := s.store.ListUsers(pageNum, size, req.Keyword)
	if err != nil {
		fail(c, CodeInternalError, "failed to query users")
		return
	}
	page(c, items, pageNum, size, total)
}

type updateUserRequest struct {
	ID   int64  `json:"id"`
	Role string `json:"role"`
}

// updateUser POST /api/user/update（admin）
func (s *Server) updateUser(c *gin.Context) {
	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	if req.Role != model.RoleAdmin && req.Role != model.RoleUser {
		fail(c, CodeValidationError, "role must be admin or user")
		return
	}
	user, err := s.store.GetUserByID(req.ID)
	if err == sql.ErrNoRows {
		fail(c, CodeUserNotFound, "user not found")
		return
	}
	if err != nil {
		fail(c, CodeInternalError, "failed to update user")
		return
	}
	if user.Username == builtInAdmin && req.Role != model.RoleAdmin {
		fail(c, CodeAdminUserImmutable, "built-in admin role cannot be changed")
		return
	}
	if err := s.store.UpdateUserRole(req.ID, req.Role); err != nil {
		fail(c, CodeInternalError, "failed to update user")
		return
	}
	user, _ = s.store.GetUserByID(req.ID)
	ok(c, user)
}

type userIDRequest struct {
	ID int64 `json:"id"`
}

// deleteUser POST /api/user/delete（admin）
func (s *Server) deleteUser(c *gin.Context) {
	var req userIDRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	user, err := s.store.GetUserByID(req.ID)
	if err == sql.ErrNoRows {
		fail(c, CodeUserNotFound, "user not found")
		return
	}
	if err != nil {
		fail(c, CodeInternalError, "failed to delete user")
		return
	}
	if user.Username == builtInAdmin {
		fail(c, CodeAdminUserImmutable, "built-in admin cannot be deleted")
		return
	}
	if err := s.store.DeleteUser(req.ID); err != nil {
		fail(c, CodeInternalError, "failed to delete user")
		return
	}
	ok(c, nil)
}

type resetPasswordRequest struct {
	ID       int64  `json:"id"`
	Password string `json:"password"`
}

// resetPassword POST /api/user/reset-password（admin）
func (s *Server) resetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	if req.Password == "" {
		fail(c, CodeValidationError, "password is required")
		return
	}
	if _, err := s.store.GetUserByID(req.ID); err == sql.ErrNoRows {
		fail(c, CodeUserNotFound, "user not found")
		return
	} else if err != nil {
		fail(c, CodeInternalError, "failed to reset password")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		fail(c, CodeInternalError, "failed to reset password")
		return
	}
	if err := s.store.UpdateUserPassword(req.ID, hash); err != nil {
		fail(c, CodeInternalError, "failed to reset password")
		return
	}
	ok(c, nil)
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// changePassword POST /api/user/change-password（任意登录用户改自己密码）
func (s *Server) changePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, CodeValidationError, "invalid request body")
		return
	}
	if req.NewPassword == "" {
		fail(c, CodeValidationError, "new password is required")
		return
	}
	user, err := s.store.GetUserByID(c.GetInt64(auth.CtxUserID))
	if err != nil {
		fail(c, CodeUserNotFound, "user not found")
		return
	}
	if !auth.CheckPassword(user.PasswordHash, req.OldPassword) {
		fail(c, CodeValidationError, "incorrect old password")
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		fail(c, CodeInternalError, "failed to change password")
		return
	}
	if err := s.store.UpdateUserPassword(user.ID, hash); err != nil {
		fail(c, CodeInternalError, "failed to change password")
		return
	}
	ok(c, nil)
}
