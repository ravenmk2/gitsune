package store

import (
	"fmt"
	"strings"

	"gitsune/internal/model"
)

// UpsertRepo 按 (platform, owner, name) 插入或更新仓库，返回仓库 ID 与是否为新增。
func (s *Store) UpsertRepo(r *model.Repo) (int64, bool, error) {
	now := model.NowUTC()
	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO repo
		 (platform, owner, name, url, description, language, stars, forks, license, source, created_at, last_synced_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Platform, r.Owner, r.Name, r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source, now, now,
	)
	if err != nil {
		return 0, false, err
	}
	if n, _ := res.RowsAffected(); n == 1 {
		id, err := res.LastInsertId()
		return id, true, err
	}
	if _, err := s.db.Exec(
		`UPDATE repo SET url = ?, description = ?, language = ?, stars = ?, forks = ?, license = ?, source = ?, last_synced_at = ?
		 WHERE platform = ? AND owner = ? AND name = ?`,
		r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source, now,
		r.Platform, r.Owner, r.Name,
	); err != nil {
		return 0, false, err
	}
	var id int64
	if err := s.db.Get(&id, `SELECT id FROM repo WHERE platform = ? AND owner = ? AND name = ?`, r.Platform, r.Owner, r.Name); err != nil {
		return 0, false, err
	}
	return id, false, nil
}

// GetRepoByID 按 ID 查询仓库，不存在返回 sql.ErrNoRows。
func (s *Store) GetRepoByID(id int64) (*model.Repo, error) {
	var r model.Repo
	if err := s.db.Get(&r, `SELECT * FROM repo WHERE id = ?`, id); err != nil {
		return nil, err
	}
	return &r, nil
}

// DeleteRepo 删除仓库。
func (s *Store) DeleteRepo(id int64) error {
	_, err := s.db.Exec(`DELETE FROM repo WHERE id = ?`, id)
	return err
}

// ListRepos 分页查询仓库，按 stars 倒序；keyword 模糊匹配 owner/name/description。
func (s *Store) ListRepos(page, size int, platform, keyword, language, source string) ([]model.Repo, int, error) {
	var conds []string
	var args []any
	if platform != "" {
		conds = append(conds, `platform = ?`)
		args = append(args, platform)
	}
	if language != "" {
		conds = append(conds, `language = ?`)
		args = append(args, language)
	}
	if source != "" {
		conds = append(conds, `source = ?`)
		args = append(args, source)
	}
	if keyword != "" {
		conds = append(conds, `(owner LIKE ? OR name LIKE ? OR description LIKE ?)`)
		kw := likePattern(keyword)
		args = append(args, kw, kw, kw)
	}
	where := ""
	if len(conds) > 0 {
		where = " WHERE " + strings.Join(conds, " AND ")
	}
	var total int
	if err := s.db.Get(&total, `SELECT COUNT(*) FROM repo`+where, args...); err != nil {
		return nil, 0, err
	}
	items := []model.Repo{}
	query := fmt.Sprintf(`SELECT * FROM repo%s ORDER BY stars DESC, id ASC LIMIT ? OFFSET ?`, where)
	args = append(args, size, (page-1)*size)
	if err := s.db.Select(&items, query, args...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
