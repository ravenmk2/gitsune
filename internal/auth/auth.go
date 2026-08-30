// Package auth 提供 JWT 签发/校验、bcrypt 密码哈希与 Gin 认证中间件。
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"gitsune/internal/model"
)

// tokenTTL Token 有效期 7 天。
const tokenTTL = 7 * 24 * time.Hour

// Claims JWT 载荷。
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Service 认证服务。
type Service struct {
	secret []byte
}

// NewService 创建认证服务。
func NewService(secret string) *Service {
	return &Service{secret: []byte(secret)}
}

// HashPassword 生成 bcrypt 密码哈希。
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

// CheckPassword 校验密码。
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateToken 为用户签发 JWT。
func (s *Service) GenerateToken(u *model.User) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
}

// ParseToken 解析并校验 JWT。
func (s *Service) ParseToken(tokenStr string) (*Claims, error) {
	if tokenStr == "" {
		return nil, errors.New("empty token")
	}
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, err
	}
	return claims, nil
}
