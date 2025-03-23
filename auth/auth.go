package auth

import (
	"errors"
	"fmt"
	"log"
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
		return nil, errors.New("用户ID不能为空")
	}

	// 打印生成令牌的用户信息
	log.Printf("为用户 %s 生成令牌对", userID)

	// 生成访问令牌
	accessToken, err := m.jwtManager.GenerateToken(userID, extra)
	if err != nil {
		log.Printf("生成访问令牌失败: %v", err)
		return nil, err
	}

	// 确保refreshExtra是一个新的map，避免引用相同的内存
	refreshExtra := make(map[string]interface{})
	// 复制原始extra中的信息
	if extra != nil {
		for k, v := range extra {
			if k != "token_type" && k != "nonce" {
				refreshExtra[k] = v
			}
		}
	}

	// 添加token_type和时间戳
	refreshExtra["token_type"] = "refresh"
	refreshExtra["nonce"] = fmt.Sprintf("%d", time.Now().UnixNano())

	// 打印用于刷新令牌的额外信息
	log.Printf("刷新令牌额外信息: %+v", refreshExtra)

	refreshToken, err := m.jwtManager.GenerateToken(userID, refreshExtra, m.refreshExpires)
	if err != nil {
		log.Printf("生成刷新令牌失败: %v", err)
		return nil, err
	}

	if len(refreshToken) > 10 {
		log.Printf("已生成刷新令牌: %s...", refreshToken[:10])
	}

	// 确保令牌不在黑名单中
	if m.jwtManager.IsBlacklisted(refreshToken) {
		log.Printf("警告：新生成的刷新令牌错误地在黑名单中！")
	}

	// 存储刷新令牌
	m.refreshTokensLock.Lock()
	m.refreshTokens[refreshToken] = userID
	m.refreshTokensLock.Unlock()
	log.Printf("已将刷新令牌存储到 refreshTokens map 中")

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// RefreshAccessToken 使用刷新令牌获取新的访问令牌
func (m *AuthManager) RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	// 输出令牌前缀以便调试
	if len(refreshToken) > 10 {
		log.Printf("正在刷新的令牌前缀: %s...", refreshToken[:10])
	}

	// 首先检查令牌是否为空
	if refreshToken == "" {
		return nil, errors.New("刷新令牌不能为空")
	}

	// 检查刷新令牌是否在黑名单中
	if m.jwtManager.IsBlacklisted(refreshToken) {
		log.Printf("令牌在黑名单中: %s...", refreshToken[:10])
		return nil, errors.New("刷新令牌已被列入黑名单")
	}

	// 检查刷新令牌是否在存储中
	m.refreshTokensLock.RLock()
	userID, exists := m.refreshTokens[refreshToken]
	m.refreshTokensLock.RUnlock()

	if !exists {
		log.Printf("令牌不在存储中: %s...", refreshToken[:10])
		return nil, errors.New("未找到刷新令牌")
	} else {
		log.Printf("令牌在存储中，对应用户ID: %s", userID)
	}

	// 验证刷新令牌
	claims, err := m.jwtManager.ValidateToken(refreshToken)
	if err != nil {
		log.Printf("令牌验证失败: %v", err)
		// 如果验证失败，确保从存储中删除并加入黑名单
		m.refreshTokensLock.Lock()
		delete(m.refreshTokens, refreshToken)
		m.refreshTokensLock.Unlock()
		_ = m.jwtManager.AddToBlacklist(refreshToken, time.Now().Add(m.refreshExpires))
		return nil, err
	}

	log.Printf("令牌验证成功，用户ID: %s", claims.UserID)

	// 确保是刷新令牌
	if claims.Extra == nil || claims.Extra["token_type"] != "refresh" {
		log.Printf("令牌不是刷新令牌类型")
		// 如果不是刷新令牌，从存储中删除并加入黑名单
		m.refreshTokensLock.Lock()
		delete(m.refreshTokens, refreshToken)
		m.refreshTokensLock.Unlock()
		// 将无效令牌加入黑名单
		_ = m.jwtManager.AddToBlacklist(refreshToken, time.Now().Add(m.refreshExpires))
		return nil, errors.New("无效的刷新令牌")
	}

	// 创建新的额外信息，不包含token_type
	userExtra := make(map[string]interface{})
	for k, v := range claims.Extra {
		if k != "token_type" && k != "nonce" && k != "jti" {
			userExtra[k] = v
		}
	}

	log.Printf("准备创建新的令牌对，用户ID: %s", userID)

	// 生成新的令牌对（在撤销旧令牌之前）
	accessToken, err := m.jwtManager.GenerateToken(userID, userExtra)
	if err != nil {
		log.Printf("生成访问令牌失败: %v", err)
		return nil, err
	}

	// 生成新的刷新令牌
	refreshExtra := make(map[string]interface{})
	refreshExtra["token_type"] = "refresh"
	refreshExtra["nonce"] = fmt.Sprintf("%d", time.Now().UnixNano())
	// 复制原始extra中的其他信息
	for k, v := range userExtra {
		refreshExtra[k] = v
	}

	newRefreshToken, err := m.jwtManager.GenerateToken(userID, refreshExtra, m.refreshExpires)
	if err != nil {
		log.Printf("生成刷新令牌失败: %v", err)
		return nil, err
	}

	// 存储新的刷新令牌（在撤销旧令牌之前）
	m.refreshTokensLock.Lock()
	// 先添加新令牌再删除旧令牌，避免临时状态下两个令牌都不可用
	m.refreshTokens[newRefreshToken] = userID

	// 然后删除旧令牌
	delete(m.refreshTokens, refreshToken)
	m.refreshTokensLock.Unlock()

	// 将旧令牌加入黑名单
	_ = m.jwtManager.AddToBlacklist(refreshToken, time.Now().Add(m.refreshExpires))
	log.Printf("已将旧令牌添加到黑名单: %s...", refreshToken[:10])

	// 验证新令牌不在黑名单中
	if m.jwtManager.IsBlacklisted(newRefreshToken) {
		log.Printf("警告：新生成的刷新令牌被错误地加入黑名单！")
	} else {
		log.Printf("新生成的刷新令牌不在黑名单中，符合预期")
	}

	// 验证新令牌在存储中
	m.refreshTokensLock.RLock()
	_, newExists := m.refreshTokens[newRefreshToken]
	m.refreshTokensLock.RUnlock()
	if !newExists {
		log.Printf("警告：新生成的刷新令牌不在存储中！")
	} else {
		log.Printf("新生成的刷新令牌在存储中，符合预期")
	}

	// 对比新旧令牌
	if refreshToken == newRefreshToken {
		log.Printf("警告：新旧刷新令牌相同！这不应该发生！")
	} else {
		log.Printf("新旧刷新令牌不同，符合预期")
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
	}, nil
}

// RevokeRefreshToken 撤销刷新令牌
func (m *AuthManager) RevokeRefreshToken(refreshToken string) error {
	m.refreshTokensLock.Lock()
	defer m.refreshTokensLock.Unlock()

	// 检查令牌是否存在
	if _, exists := m.refreshTokens[refreshToken]; !exists {
		return errors.New("未找到刷新令牌")
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
