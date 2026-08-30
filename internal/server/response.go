package server

import (
	"math"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错误码
const (
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeForbidden           = "FORBIDDEN"
	CodeValidationError     = "VALIDATION_ERROR"
	CodeUsernameExists      = "USERNAME_EXISTS"
	CodeUserNotFound        = "USER_NOT_FOUND"
	CodeRepoNotFound        = "REPO_NOT_FOUND"
	CodePlatformNotSupport  = "PLATFORM_NOT_SUPPORTED"
	CodeTaskAlreadyRunning  = "TASK_ALREADY_RUNNING"
	CodeAdminUserImmutable  = "ADMIN_USER_IMMUTABLE"
	CodeInternalError       = "INTERNAL_ERROR"
)

// ok 返回成功信封。
func ok(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data, "error": nil})
}

// fail 返回失败信封（业务错误统一 HTTP 200）。
func fail(c *gin.Context, code, message string) {
	c.JSON(http.StatusOK, gin.H{
		"success": false,
		"data":    nil,
		"error":   gin.H{"code": code, "message": message},
	})
}

// pageResult 分页结构。
type pageResult struct {
	Items     any `json:"items"`
	Page      int `json:"page"`
	Size      int `json:"size"`
	Total     int `json:"total"`
	PageCount int `json:"page_count"`
}

// page 返回分页信封。
func page(c *gin.Context, items any, pageNum, size, total int) {
	pageCount := 0
	if size > 0 {
		pageCount = int(math.Ceil(float64(total) / float64(size)))
	}
	ok(c, pageResult{Items: items, Page: pageNum, Size: size, Total: total, PageCount: pageCount})
}

// normalizePage 规整分页参数：page 从 1 开始，size 默认 10、上限 100。
func normalizePage(pageNum, size int) (int, int) {
	if pageNum < 1 {
		pageNum = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	return pageNum, size
}
