package store

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"gitsune/internal/model"
)

// InsertRepo 按 (platform, owner, name) 插入仓库，已存在则忽略（不覆盖任何字段），返回仓库 ID 与是否为新增。
func (s *Store) InsertRepo(r *model.Repo) (int64, bool, error) {
	now := model.NowUTC()
	res, err := s.db.Exec(
		`INSERT OR IGNORE INTO repo
		 (platform, owner, name, url, description, language, stars, forks, license, source, created_at, refreshed_at)
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
	var id int64
	if err := s.db.Get(&id, `SELECT id FROM repo WHERE platform = ? AND owner = ? AND name = ?`, r.Platform, r.Owner, r.Name); err != nil {
		return 0, false, err
	}
	return id, false, nil
}

// UpsertRepo 按 (platform, owner, name) 插入或更新仓库，返回仓库 ID 与是否为新增。
func (s *Store) UpsertRepo(r *model.Repo) (int64, bool, error) {
	id, created, err := s.InsertRepo(r)
	if err != nil || created {
		return id, created, err
	}
	now := model.NowUTC()
	if _, err := s.db.Exec(
		`UPDATE repo SET url = ?, description = ?, language = ?, stars = ?, forks = ?, license = ?, source = ?, refreshed_at = ?
		 WHERE platform = ? AND owner = ? AND name = ?`,
		r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source, now,
		r.Platform, r.Owner, r.Name,
	); err != nil {
		return 0, false, err
	}
	return id, false, nil
}

// GetRepoByKey 按 (platform, owner, name) 查询仓库，不存在返回 sql.ErrNoRows。
func (s *Store) GetRepoByKey(platform, owner, name string) (*model.Repo, error) {
	var r model.Repo
	if err := s.db.Get(&r, `SELECT * FROM repo WHERE platform = ? AND owner = ? AND name = ?`, platform, owner, name); err != nil {
		return nil, err
	}
	return &r, nil
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

// ListRepos 分页查询仓库，默认按 id 倒序，支持按 id/stars/forks 升降序；keyword 模糊匹配 owner/name/description。
func (s *Store) ListRepos(page, size int, platform, keyword, language, source, sortBy, sortOrder string) ([]model.Repo, int, error) {
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
	orderBy := repoOrderBy(sortBy, sortOrder)
	query := fmt.Sprintf(`SELECT * FROM repo%s ORDER BY %s LIMIT ? OFFSET ?`, where, orderBy)
	args = append(args, size, (page-1)*size)
	if err := s.db.Select(&items, query, args...); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// repoOrderBy 生成排序子句：列名白名单防注入，默认 id DESC；非 id 排序时以 id ASC 兜底保证稳定。
func repoOrderBy(sortBy, sortOrder string) string {
	col := "id"
	switch sortBy {
	case "stars", "forks":
		col = sortBy
	}
	dir := "DESC"
	if sortOrder == "asc" {
		dir = "ASC"
	}
	if col == "id" {
		return "id " + dir
	}
	return col + " " + dir + ", id ASC"
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

// ListStaleRepos 查询刷新时间老于 cutoff（RFC3339 UTC）的仓库，按 id 升序；空串视为最旧。
func (s *Store) ListStaleRepos(cutoff string) ([]model.Repo, error) {
	items := []model.Repo{}
	if err := s.db.Select(&items, `SELECT * FROM repo WHERE refreshed_at < ? ORDER BY id`, cutoff); err != nil {
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
		 (platform, owner, name, url, description, language, stars, forks, license, source, created_at, refreshed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = insert.Close() }()
	// refreshed_at 只在数据实际变化时随更新写入，不参与变化判定
	update, err := tx.Preparex(
		`UPDATE repo SET url = ?, description = ?, language = ?, stars = ?, forks = ?, license = ?, source = ?, refreshed_at = ?
		 WHERE platform = ? AND owner = ? AND name = ?
		   AND (url != ? OR description != ? OR language != ? OR stars != ? OR forks != ? OR license != ? OR source != ?)`)
	if err != nil {
		return 0, 0, err
	}
	defer func() { _ = update.Close() }()

	for i := range repos {
		r := &repos[i]
		res, err := insert.Exec(
			r.Platform, r.Owner, r.Name, r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source, now, now,
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
			r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source, now,
			r.Platform, r.Owner, r.Name,
			r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source,
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
		 (platform, owner, name, url, description, language, stars, forks, license, source, created_at, refreshed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer func() { _ = insert.Close() }()

	now := model.NowUTC()
	for i := range repos {
		r := &repos[i]
		if _, err := insert.Exec(
			r.Platform, r.Owner, r.Name, r.URL, r.Description, r.Language, r.Stars, r.Forks, r.License, r.Source, now, now,
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
