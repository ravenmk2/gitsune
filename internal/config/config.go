// Package config 从环境变量加载服务配置。
package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
)

// Config 服务运行配置。
type Config struct {
	Addr          string // GITSUNE_LISTEN_ADDR，默认 :8080
	DataPath      string // GITSUNE_DATA_PATH，数据目录，默认 ./data
	AdminPassword string // GITSUNE_ADMIN_PASSWORD，内置 admin 初始密码
	GitHubToken   string // GITSUNE_GITHUB_TOKEN，播种到 setting
	Cron          string // GITSUNE_CRON，播种到 setting
	LogLevel      string // GITSUNE_LOG_LEVEL，默认 info
	JWTSecret     string // 数据目录 jwt_secret 文件，首次启动生成
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Load 读取环境变量并返回配置。
func Load() *Config {
	return &Config{
		Addr:          env("GITSUNE_LISTEN_ADDR", ":8080"),
		DataPath:      env("GITSUNE_DATA_PATH", "./data"),
		AdminPassword: os.Getenv("GITSUNE_ADMIN_PASSWORD"),
		GitHubToken:   os.Getenv("GITSUNE_GITHUB_TOKEN"),
		Cron:          os.Getenv("GITSUNE_CRON"),
		LogLevel:      env("GITSUNE_LOG_LEVEL", "info"),
	}
}

// DBPath 返回 SQLite 数据库文件路径（数据目录下的 gitsune.db）。
func (c *Config) DBPath() string {
	return filepath.Join(c.DataPath, "gitsune.db")
}

// jwtSecretFile 是数据目录下持久化 JWT secret 的文件名。
const jwtSecretFile = "jwt_secret"

// ResolveJWTSecret 确定 JWT secret：读取数据目录下的 jwt_secret 文件，
// 文件不存在时生成随机 secret 并写入（0600），保证重启后 token 仍然有效。
// 数据目录不可写时回退为仅内存的随机 secret 并告警。
func (c *Config) ResolveJWTSecret() {
	path := filepath.Join(c.DataPath, jwtSecretFile)
	if data, err := os.ReadFile(path); err == nil && len(data) > 0 {
		c.JWTSecret = string(data)
		return
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		logrus.Fatalf("failed to generate random JWT secret: %v", err)
	}
	c.JWTSecret = hex.EncodeToString(buf)
	if err := os.MkdirAll(c.DataPath, 0o755); err != nil {
		logrus.Warnf("failed to create data directory; JWT secret valid for this run only: %v", err)
		return
	}
	if err := os.WriteFile(path, []byte(c.JWTSecret), 0o600); err != nil {
		logrus.Warnf("failed to persist JWT secret; secret valid for this run only: %v", err)
		return
	}
	logrus.Infof("generated random JWT secret and saved to %s", path)
}
