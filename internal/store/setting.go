package store

import "database/sql"

// GetSetting 读取设置项，不存在返回 sql.ErrNoRows。
func (s *Store) GetSetting(key string) (string, error) {
	var value string
	if err := s.db.Get(&value, `SELECT value FROM setting WHERE key = ?`, key); err != nil {
		return "", err
	}
	return value, nil
}

// GetSettingOr 读取设置项，不存在时返回默认值。
func (s *Store) GetSettingOr(key, def string) string {
	value, err := s.GetSetting(key)
	if err == sql.ErrNoRows {
		return def
	}
	if err != nil {
		return def
	}
	return value
}

// HasSetting 判断设置项是否存在。
func (s *Store) HasSetting(key string) bool {
	var count int
	if err := s.db.Get(&count, `SELECT COUNT(*) FROM setting WHERE key = ?`, key); err != nil {
		return false
	}
	return count > 0
}

// SetSetting 写入设置项（upsert）。
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO setting (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	return err
}
