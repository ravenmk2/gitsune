package store

import (
	"time"

	"gitsune/internal/model"
)

// NameCount 通用"名称-数量"统计行（平台/语言分布等）。
type NameCount struct {
	Name  string `db:"name" json:"name"`
	Count int    `db:"count" json:"count"`
}

// RepoTotals 仓库总数与 stars 总和。
func (s *Store) RepoTotals() (total int, totalStars int, err error) {
	var row struct {
		Total      int `db:"total"`
		TotalStars int `db:"total_stars"`
	}
	err = s.db.Get(&row, `SELECT COUNT(*) AS total, COALESCE(SUM(stars), 0) AS total_stars FROM repo`)
	return row.Total, row.TotalStars, err
}

// CountReposSince 统计 created_at 不早于 cutoff（RFC3339 UTC）的仓库数（近 N 天新增）。
func (s *Store) CountReposSince(cutoff string) (int, error) {
	var n int
	err := s.db.Get(&n, `SELECT COUNT(*) FROM repo WHERE created_at >= ?`, cutoff)
	return n, err
}

// CountReposByPlatform 按平台统计仓库数，按数量降序。
func (s *Store) CountReposByPlatform() ([]NameCount, error) {
	rows := []NameCount{}
	err := s.db.Select(&rows, `SELECT platform AS name, COUNT(*) AS count FROM repo GROUP BY platform ORDER BY count DESC`)
	return rows, err
}

// CountReposByLanguage 按语言统计仓库数（忽略空语言），按数量降序取前 limit 名。
func (s *Store) CountReposByLanguage(limit int) ([]NameCount, error) {
	rows := []NameCount{}
	err := s.db.Select(&rows,
		`SELECT language AS name, COUNT(*) AS count FROM repo WHERE language != '' GROUP BY language ORDER BY count DESC, name LIMIT ?`, limit)
	return rows, err
}

// ListLatestRepos 最近收录的仓库（按收录时间倒序）。
func (s *Store) ListLatestRepos(limit int) ([]model.Repo, error) {
	items := []model.Repo{}
	err := s.db.Select(&items, `SELECT * FROM repo ORDER BY created_at DESC, id DESC LIMIT ?`, limit)
	return items, err
}

// ListTopRepos stars 最高的仓库（同星按 id 升序兜底）。
func (s *Store) ListTopRepos(limit int) ([]model.Repo, error) {
	items := []model.Repo{}
	err := s.db.Select(&items, `SELECT * FROM repo ORDER BY stars DESC, id ASC LIMIT ?`, limit)
	return items, err
}

// RecentCutoff 返回 days 天前的 RFC3339 UTC 时间，用于"近 N 天"统计。
func RecentCutoff(days int) string {
	return time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
}
