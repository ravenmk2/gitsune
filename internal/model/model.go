// Package model 定义数据模型。时间字段以 RFC3339 UTC 字符串存储与输出。
package model

import "time"

// 用户角色
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// 仓库来源
const (
	SourceManual   = "manual"
	SourceTrending = "trending"
	SourceGVP      = "gvp"
)

// 任务类型
const (
	TaskTypeGitHubTrending = "github_trending"
	TaskTypeGiteeGVP       = "gitee_gvp"
)

// 任务状态
const (
	TaskStatusRunning = "running"
	TaskStatusSuccess = "success"
	TaskStatusFailed  = "failed"
)

// 任务触发方式
const (
	TriggerAuto   = "auto"
	TriggerManual = "manual"
)

// NowUTC 返回当前时间的 RFC3339 UTC 字符串。
func NowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// User 用户表。
type User struct {
	ID           int64  `db:"id" json:"id"`
	Username     string `db:"username" json:"username"`
	PasswordHash string `db:"password_hash" json:"-"`
	Role         string `db:"role" json:"role"`
	CreatedAt    string `db:"created_at" json:"created_at"`
	UpdatedAt    string `db:"updated_at" json:"updated_at"`
}

// Repo 仓库表。
type Repo struct {
	ID           int64  `db:"id" json:"id"`
	Platform     string `db:"platform" json:"platform"`
	Owner        string `db:"owner" json:"owner"`
	Name         string `db:"name" json:"name"`
	URL          string `db:"url" json:"url"`
	Description  string `db:"description" json:"description"`
	Language     string `db:"language" json:"language"`
	Stars        int    `db:"stars" json:"stars"`
	Forks        int    `db:"forks" json:"forks"`
	License      string `db:"license" json:"license"`
	Source       string `db:"source" json:"source"`
	CreatedAt    string `db:"created_at" json:"created_at"`
	LastSyncedAt string `db:"last_synced_at" json:"last_synced_at"`
}

// TaskLog 任务执行历史表。
type TaskLog struct {
	ID          int64  `db:"id" json:"id"`
	Type        string `db:"type" json:"type"`
	Status      string `db:"status" json:"status"`
	TriggerMode string `db:"trigger_mode" json:"trigger_mode"`
	Message     string `db:"message" json:"message"`
	AddedCount  int    `db:"added_count" json:"added_count"`
	StartedAt   string `db:"started_at" json:"started_at"`
	FinishedAt  string `db:"finished_at" json:"finished_at"`
}
