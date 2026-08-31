package store

import (
	"fmt"
	"strings"

	"gitsune/internal/model"
)

// GetUserByUsername 按用户名查询用户，不存在返回 sql.ErrNoRows。
func (s *Store) GetUserByUsername(username string) (*model.User, error) {
	var u model.User
	if err := s.db.Get(&u, `SELECT * FROM user WHERE username = ?`, username); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetUserByID 按 ID 查询用户，不存在返回 sql.ErrNoRows。
func (s *Store) GetUserByID(id int64) (*model.User, error) {
	var u model.User
	if err := s.db.Get(&u, `SELECT * FROM user WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &u, nil
}

// CreateUser 创建用户。
func (s *Store) CreateUser(username, passwordHash, role string) (*model.User, error) {
	now := model.NowUTC()
	res, err := s.db.Exec(
		`INSERT INTO user (username, password_hash, role, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		username, passwordHash, role, now, now,
	)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(id)
}

// ListUsers 分页查询用户，按 id 倒序（新用户在前），keyword 模糊匹配用户名。
func (s *Store) ListUsers(page, size int, keyword string) ([]model.User, int, error) {
	where := ""
	args := []any{}
	if keyword != "" {
		where = ` WHERE username LIKE ?`
		args = append(args, "%"+keyword+"%")
	}
	var total int
	if err := s.db.Get(&total, `SELECT COUNT(*) FROM user`+where, args...); err != nil {
		return nil, 0, err
	}
	items := []model.User{}
	query := fmt.Sprintf(`SELECT * FROM user%s ORDER BY id DESC LIMIT ? OFFSET ?`, where)
	args = append(args, size, (page-1)*size)
	if err := s.db.Select(&items, query, args...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// UpdateUserRole 更新用户角色。
func (s *Store) UpdateUserRole(id int64, role string) error {
	_, err := s.db.Exec(`UPDATE user SET role = ?, updated_at = ? WHERE id = ?`, role, model.NowUTC(), id)
	return err
}

// UpdateUserPassword 更新用户密码哈希。
func (s *Store) UpdateUserPassword(id int64, passwordHash string) error {
	_, err := s.db.Exec(`UPDATE user SET password_hash = ?, updated_at = ? WHERE id = ?`, passwordHash, model.NowUTC(), id)
	return err
}

// DeleteUser 删除用户。
func (s *Store) DeleteUser(id int64) error {
	_, err := s.db.Exec(`DELETE FROM user WHERE id = ?`, id)
	return err
}

// normalizePage 规整分页参数。
func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

// likePattern 构造 LIKE 模糊匹配模式。
func likePattern(keyword string) string {
	return "%" + strings.TrimSpace(keyword) + "%"
}
