package utils

import (
	"github.com/golang-jwt/jwt/v5"
	"time"
)

var JWTSecret = []byte("cdq123") // JWT 密钥

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
