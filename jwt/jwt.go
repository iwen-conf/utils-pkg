package jwt

import (
	"errors"
	"fmt"
	"log"
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

// ClaimValidator JWTManager JWT 管理器
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

	// 添加调试日志
	log.Printf("为用户 %s 生成令牌，过期时间: %v", userID, expires)
	if extra != nil {
		log.Printf("令牌额外信息: %+v", extra)
	}

	// 确保 extra 是一个新的 map，避免引用相同的内存
	tokenExtra := make(map[string]interface{})
	for k, v := range extra {
		tokenExtra[k] = v
	}

	// 始终添加一个随机的 jti (JWT ID) 来确保每个令牌都是唯一的
	tokenExtra["jti"] = fmt.Sprintf("%d", time.Now().UnixNano())

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expires)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			ID:        tokenExtra["jti"].(string), // 设置唯一的令牌ID
		},
		UserID: userID,
		Extra:  tokenExtra,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.secretKey)
	if err != nil {
		log.Printf("令牌签名失败: %v", err)
		return "", err
	}

	if len(tokenStr) > 10 {
		log.Printf("已生成令牌: %s...", tokenStr[:10])
	}

	return tokenStr, nil
}

// ValidateToken 验证 JWT token 的有效性
func (m *JWTManager) ValidateToken(tokenStr string) (*Claims, error) {
	// 首先检查是否在黑名单中 - 这样可以避免解析无效令牌
	if m.IsBlacklisted(tokenStr) {
		return nil, errors.New("token is blacklisted")
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("意外的签名方法")
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, err
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

	if len(tokenStr) > 10 {
		log.Printf("添加令牌到黑名单: %s..., 过期时间: %v", tokenStr[:10], expireAt)
	}

	m.blacklistLock.Lock()
	defer m.blacklistLock.Unlock()

	m.blacklist[tokenStr] = expireAt
	return nil
}

// IsBlacklisted 检查 token 是否在黑名单中
func (m *JWTManager) IsBlacklisted(tokenStr string) bool {
	m.blacklistLock.RLock()

	// 显示完整的黑名单内容以便调试
	log.Printf("当前黑名单包含 %d 个令牌", len(m.blacklist))
	for token, expireAt := range m.blacklist {
		if len(token) > 10 {
			log.Printf("黑名单中的令牌: %s..., 过期时间: %v", token[:10], expireAt)
		}
	}

	expireAt, exists := m.blacklist[tokenStr]

	// 先释放读锁
	m.blacklistLock.RUnlock()

	if !exists {
		if len(tokenStr) > 10 {
			log.Printf("令牌不在黑名单中: %s...", tokenStr[:10])
		}
		return false
	}

	// 如果黑名单过期时间已到，从黑名单中移除
	if time.Now().After(expireAt) {
		if len(tokenStr) > 10 {
			log.Printf("令牌在黑名单中但已过期，移除: %s...", tokenStr[:10])
		}

		// 获取写锁删除过期条目
		m.blacklistLock.Lock()
		delete(m.blacklist, tokenStr)
		m.blacklistLock.Unlock()

		return false
	}

	if len(tokenStr) > 10 {
		log.Printf("令牌在黑名单中: %s..., 将在 %v 过期", tokenStr[:10], expireAt)
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
