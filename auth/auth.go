package auth

import (
	"errors"
	"sync"
	"time"

	"github.com/iwen-conf/utils-pkg/jwt"
)

// TokenPair 包含访问令牌和刷新令牌
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthManager 认证管理器
type AuthManager struct {
	jwtManager     *jwt.JWTManager
	accessExpires  time.Duration
	refreshExpires time.Duration
	// 用于存储刷新令牌的映射关系
	refreshTokens     map[string]string // refreshToken -> userID
	refreshTokensLock sync.RWMutex
}

// NewAuthManager 创建新的认证管理器
func NewAuthManager(secretKey string, accessExpires, refreshExpires time.Duration) *AuthManager {
	return &AuthManager{
		jwtManager:     jwt.NewJWTManager(secretKey, accessExpires),
		accessExpires:  accessExpires,
		refreshExpires: refreshExpires,
		refreshTokens:  make(map[string]string),
	}
}

// GenerateTokenPair 生成访问令牌和刷新令牌对
func (m *AuthManager) GenerateTokenPair(userID string, extra map[string]interface{}) (*TokenPair, error) {
	// 验证userID不能为空
	if userID == "" {
		return nil, errors.New("userID cannot be empty")
	}

	// 生成访问令牌
	accessToken, err := m.jwtManager.GenerateToken(userID, extra)
	if err != nil {
		return nil, err
	}

	// 生成刷新令牌
	refreshExtra := map[string]interface{}{
		"token_type": "refresh",
	}
	refreshToken, err := m.jwtManager.GenerateToken(userID, refreshExtra, m.refreshExpires)
	if err != nil {
		return nil, err
	}

	// 存储刷新令牌
	m.refreshTokensLock.Lock()
	m.refreshTokens[refreshToken] = userID
	m.refreshTokensLock.Unlock()

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// RefreshAccessToken 使用刷新令牌获取新的访问令牌
func (m *AuthManager) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	// 验证刷新令牌
	claims, err := m.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		return nil, err
	}

	// 确保是刷新令牌
	if claims.Extra == nil || claims.Extra["token_type"] != "refresh" {
		return nil, errors.New("invalid refresh token")
	}

	// 检查刷新令牌是否在存储中
	m.refreshTokensLock.RLock()
	userID, exists := m.refreshTokens[refreshToken]
	m.refreshTokensLock.RUnlock()

	if !exists {
		return nil, errors.New("refresh token not found")
	}

	// 创建新的额外信息，不包含token_type
	userExtra := make(map[string]interface{})
	for k, v := range claims.Extra {
		if k != "token_type" {
			userExtra[k] = v
		}
	}

	// 生成新的令牌对
	newTokenPair, err := m.GenerateTokenPair(userID, userExtra)
	if err != nil {
		return nil, err
	}

	// 可选：撤销旧的刷新令牌
	m.refreshTokensLock.Lock()
	delete(m.refreshTokens, refreshToken)
	m.refreshTokensLock.Unlock()

	return newTokenPair, nil
}

// RevokeRefreshToken 撤销刷新令牌
func (m *AuthManager) RevokeRefreshToken(refreshToken string) error {
	m.refreshTokensLock.Lock()
	defer m.refreshTokensLock.Unlock()

	// 检查令牌是否存在
	if _, exists := m.refreshTokens[refreshToken]; !exists {
		return errors.New("refresh token not found")
	}

	// 从存储中删除刷新令牌
	delete(m.refreshTokens, refreshToken)

	// 将访问令牌加入黑名单
	return m.jwtManager.AddToBlacklist(refreshToken, time.Now().Add(m.refreshExpires))
}

// ValidateAccessToken 验证访问令牌
func (m *AuthManager) ValidateAccessToken(accessToken string) (*jwt.Claims, error) {
	return m.jwtManager.ValidateToken(accessToken)
}