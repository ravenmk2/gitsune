package store

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

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

// ListAll 查询仓库全量列表（platforms 为空=全部），按 platform,owner,name 排序。
func (s *Store) ListAll(platforms []string) ([]model.Repo, error) {
	items := []model.Repo{}
	query := `SELECT * FROM repo`
	var args []any
	if len(platforms) > 0 {
		q, a, err := sqlx.In(`SELECT * FROM repo WHERE platform IN (?)`, platforms)
		if err != nil {
			return nil, err
		}
		query, args = q, a
	}
	query += ` ORDER BY platform, owner, name`
	if err := s.db.Select(&items, query, args...); err != nil {
		return nil, err
	}
	return items, nil
}

// UpsertMany 单事务内按 (platform, owner, name) 批量 upsert 仓库，返回新增与更新条数。
func (s *Store) UpsertMany(repos []model.Repo) (added, updated int, err error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = tx.Rollback() }()

	now := model.NowUTC()
	insert, err := tx.Preparex(
		`INSERT OR IGNORE INTO repo
		 (platform, owner, name, url, description, language, stars, forks, license, source, created_at, last_synced_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = insert.Close() }()
	update, err := tx.Preparex(
		`UPDATE repo SET url = ?, description = ?, language = ?, stars = ?, forks = ?, license = ?, source = ?, last_synced_at = ?
		 WHERE platform = ? AND owner = ? AND name = ?
		   AND (url != ? OR description != ? OR language != ? OR stars != ? OR forks != ? OR license != ? OR source != ? OR last_synced_at != ?)`)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = update.Close() }()

	for i := range repos {
		r := &repos[i]
		res, err := insert.Exec(
			r.Platform, r.Owner, r.Name, r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source, now, r.LastSyncedAt,
		)
		if err != nil {
			return 0, 0, err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			added++
			continue
		}
		// 仅元数据实际变化时计为 updated
		res, err = update.Exec(
			r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source, r.LastSyncedAt,
			r.Platform, r.Owner, r.Name,
			r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source, r.LastSyncedAt,
		)
		if err != nil {
			return 0, 0, err
		}
		if n, _ := res.RowsAffected(); n == 1 {
			updated++
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return added, updated, nil
}

// ReplaceAll 单事务内清空 repo 表并插入全部仓库（失败回滚），返回插入条数。
func (s *Store) ReplaceAll(repos []model.Repo) (added int, err error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM repo`); err != nil {
		return 0, err
	}
	insert, err := tx.Preparex(
		`INSERT INTO repo
		 (platform, owner, name, url, description, language, stars, forks, license, source, created_at, last_synced_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer func() { _ = insert.Close() }()

	now := model.NowUTC()
	for i := range repos {
		r := &repos[i]
		if _, err := insert.Exec(
			r.Platform, r.Owner, r.Name, r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source, now, r.LastSyncedAt,
		); err != nil {
			return 0, err
		}
		added++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return added, nil
}
