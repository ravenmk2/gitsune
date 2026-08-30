// Package store 基于 sqlx + 手写 SQL 的 SQLite 数据访问层。
package store

import (
	_ "embed"
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"

	"gitsune/internal/model"
)

//go:embed schema.sql
var schema string

// Store 封装数据库连接。
type Store struct {
	db *sqlx.DB
}

// Open 打开数据库并执行建表 DDL。
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sqlx.Open(driverName, path)
	if err != nil {
		return nil, err
	}
	// SQLite 单写者，串行化连接避免 SQLITE_BUSY
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close 关闭数据库连接。
func (s *Store) Close() error {
	return s.db.Close()
}

// SeedAdmin 播种内置管理员（username=admin），已存在则跳过。
func (s *Store) SeedAdmin(password string) error {
	if password == "" {
		password = "admin123"
	}
	var count int
	if err := s.db.Get(&count, `SELECT COUNT(*) FROM user WHERE username = ?`, "admin"); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	now := model.NowUTC()
	_, err = s.db.Exec(
		`INSERT INTO user (username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		"admin", string(hash), model.RoleAdmin, now, now,
	)
	if err == nil {
		logrus.Info("created built-in admin account: admin")
	}
	return err
}
