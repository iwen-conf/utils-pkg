package jwt

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 自定义的 JWT Claims 结构体
type Claims struct {
	jwt.RegisteredClaims
	UserID string                 `json:"user_id"`
	Extra  map[string]interface{} `json:"extra,omitempty"`
}

// JWTManager JWT 管理器
// ClaimValidator 自定义声明验证器函数类型
type ClaimValidator func(claims *Claims) error

// JWTManager JWT 管理器
type JWTManager struct {
	secretKey []byte
	expires   time.Duration
	// 自定义声明验证器
	validators []ClaimValidator
	// token 黑名单
	blacklist     map[string]time.Time
	blacklistLock sync.RWMutex
}

// NewJWTManager 创建新的 JWT 管理器
func NewJWTManager(secretKey string, expires time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:  []byte(secretKey),
		expires:    expires,
		validators: make([]ClaimValidator, 0),
		blacklist:  make(map[string]time.Time),
	}
}

// GenerateToken 生成 JWT token
func (m *JWTManager) GenerateToken(userID string, extra map[string]interface{}, customExpires ...time.Duration) (string, error) {
	// 验证用户ID不能为空
	if userID == "" {
		return "", errors.New("用户ID不能为空")
	}

	// 确定过期时间：如果提供了自定义过期时间，则使用自定义时间，否则使用默认时间
	expires := m.expires
	if len(customExpires) > 0 && customExpires[0] > 0 {
		expires = customExpires[0]
	}

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expires)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
		UserID: userID,
		Extra:  extra,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// ValidateToken 验证并解析 JWT token
func (m *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("意外的签名方法")
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, err
	}

	// 检查是否在黑名单中
	if m.IsBlacklisted(tokenStr) {
		return nil, errors.New("令牌已被列入黑名单")
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// 运行所有自定义验证器
		for _, validator := range m.validators {
			if err := validator(claims); err != nil {
				return nil, err
			}
		}
		return claims, nil
	}

	return nil, errors.New("无效的令牌")
}

// AddValidator 添加自定义声明验证器
func (m *JWTManager) AddValidator(validator ClaimValidator) {
	m.validators = append(m.validators, validator)
}

// AddToBlacklist 将 token 加入黑名单
func (m *JWTManager) AddToBlacklist(tokenStr string, expireAt time.Time) error {
	// 不再验证令牌，直接加入黑名单
	if tokenStr == "" {
		return errors.New("令牌不能为空")
	}

	m.blacklistLock.Lock()
	defer m.blacklistLock.Unlock()

	m.blacklist[tokenStr] = expireAt
	return nil
}

// IsBlacklisted 检查 token 是否在黑名单中
func (m *JWTManager) IsBlacklisted(tokenStr string) bool {
	m.blacklistLock.RLock()
	defer m.blacklistLock.RUnlock()

	expireAt, exists := m.blacklist[tokenStr]
	if !exists {
		return false
	}

	// 如果黑名单过期时间已到，从黑名单中移除
	if time.Now().After(expireAt) {
		delete(m.blacklist, tokenStr)
		return false
	}

	return true
}

// CleanBlacklist 清理过期的黑名单记录
func (m *JWTManager) CleanBlacklist() {
	m.blacklistLock.Lock()
	defer m.blacklistLock.Unlock()

	now := time.Now()
	for token, expireAt := range m.blacklist {
		if now.After(expireAt) {
			delete(m.blacklist, token)
		}
	}
}
