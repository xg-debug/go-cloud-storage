package utils

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"time"
)

var JWTSecret []byte

// InitJWTSecret 初始化 JWT 签名密钥。
// 优先级：配置文件 jwt.secret > 环境变量 JWT_SECRET。
// 配置的密钥必须 >= 32 字节，否则拒绝启动（防止弱密钥被伪造 token）。
// 两者都未配置时（仅限开发），生成随机临时密钥并告警——重启后所有 token 失效。
func InitJWTSecret(cfgSecret string) {
	if cfgSecret != "" {
		if len(cfgSecret) < 32 {
			panic("jwt secret too weak: configure a secret of at least 32 bytes (jwt.secret or GCS_JWT_SECRET)")
		}
		JWTSecret = []byte(cfgSecret)
		return
	}
	if envSecret := os.Getenv("JWT_SECRET"); len(envSecret) >= 32 {
		JWTSecret = []byte(envSecret)
		return
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("failed to generate ephemeral jwt secret: " + err.Error())
	}
	JWTSecret = []byte(hex.EncodeToString(buf))
	slog.Warn("JWT secret not configured; generated an ephemeral random secret. " +
		"All tokens will be invalidated on restart. Set GCS_JWT_SECRET or jwt.secret (>=32 bytes) in production.")
}

type Claims struct {
	UserId int    `json:"userId"`
	Type   string `json:"type"` // token类型：access/refresh
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成访问令牌（短期有效）
func GenerateAccessToken(userId int, expireTime time.Duration) (string, error) {
	return generateToken(userId, "access", expireTime)
}

// GenerateRefreshToken 生成刷新令牌（长期有效）
func GenerateRefreshToken(userId int, expireTime time.Duration) (string, error) {
	return generateToken(userId, "refresh", expireTime)
}

// GenerateToken 通用生成 JWT Token
func generateToken(userId int, tokenType string, expireTime time.Duration) (string, error) {
	claims := &Claims{
		UserId: userId,
		Type:   tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expireTime)), // 过期时间
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "go-cloud-storage", // 签发者
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// ParseToken 解析JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return JWTSecret, nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}

// GenerateResetToken 生成密码重置令牌
func GenerateResetToken(userId int, expireTime time.Duration) (string, error) {
	return generateToken(userId, "reset", expireTime)
}

// ParseTokenWithType 解析并验证特定类型的Token
func ParseTokenWithType(tokenString string, tokenType string) (*Claims, error) {
	claims, err := ParseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.Type != tokenType {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
