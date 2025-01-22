package jwt

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"time"
)

// Claims 自定义的 JWT Claims 结构体
type Claims struct {
	jwt.RegisteredClaims
	UserID   string                 `json:"user_id"`
	Username string                 `json:"username"`
	Extra    map[string]interface{} `json:"extra,omitempty"`
}

// JWTManager JWT 管理器
type JWTManager struct {
	secretKey []byte
	expires   time.Duration
}

// NewJWTManager 创建新的 JWT 管理器
func NewJWTManager(secretKey string, expires time.Duration) *JWTManager {
	return &JWTManager{
		secretKey: []byte(secretKey),
		expires:   expires,
	}
}

// GenerateToken 生成 JWT token
func (m *JWTManager) GenerateToken(userID, username string, extra map[string]interface{}) (string, error) {
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.expires)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		UserID:   userID,
		Username: username,
		Extra:    extra,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ValidateToken 验证并解析 JWT token
func (m *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}