package jwt

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType 定义令牌类型
type TokenType string

const (
	// AccessToken 访问令牌
	AccessToken TokenType = "access"
	// RefreshToken 刷新令牌
	RefreshToken TokenType = "refresh"
)

// StandardClaims 标准JWT声明结构体
type StandardClaims struct {
	jwt.RegisteredClaims
	// 用户标识符
	Subject string `json:"sub,omitempty"`
	// 令牌类型 (access/refresh)
	TokenType TokenType `json:"type,omitempty"`
	// 唯一会话ID
	SessionID string `json:"sid,omitempty"`
	// 令牌ID
	TokenID string `json:"jti,omitempty"`
}

// TokenOptions JWT令牌选项
type TokenOptions struct {
	// 令牌过期时间，覆盖默认值
	ExpiresIn time.Duration
	// 令牌类型
	TokenType TokenType
	// 会话ID
	SessionID string
	// 令牌ID，默认会自动生成
	TokenID string
	// 其他自定义声明
	CustomClaims map[string]interface{}
}

// DefaultTokenOptions 返回默认令牌选项
func DefaultTokenOptions() *TokenOptions {
	return &TokenOptions{
		TokenType:    AccessToken,
		CustomClaims: make(map[string]interface{}),
	}
}

// JWTOptions JWT管理器选项
type JWTOptions struct {
	// 是否启用日志
	EnableLog bool
	// 黑名单清理间隔
	BlacklistCleanInterval time.Duration
	// 启用结果缓存
	EnableCache bool
	// 缓存大小限制
	CacheSize int
	// 缓存过期时间
	CacheTTL time.Duration
	// 访问令牌默认过期时间
	AccessTokenExpiry time.Duration
	// 刷新令牌默认过期时间
	RefreshTokenExpiry time.Duration
}

// DefaultJWTOptions 返回默认的JWT管理器选项
func DefaultJWTOptions() *JWTOptions {
	return &JWTOptions{
		EnableLog:              false,            // 默认不启用日志
		BlacklistCleanInterval: 10 * time.Minute, // 默认10分钟清理一次黑名单
		EnableCache:            true,             // 默认启用缓存
		CacheSize:              1000,             // 默认缓存1000个结果
		CacheTTL:               5 * time.Minute,  // 默认缓存5分钟
		AccessTokenExpiry:      15 * time.Minute, // 默认访问令牌15分钟过期
		RefreshTokenExpiry:     24 * time.Hour,   // 默认刷新令牌24小时过期
	}
}

// cacheItem 缓存项结构
type cacheItem struct {
	claims    *StandardClaims
	err       error
	timestamp time.Time
}

// TokenManager JWT 令牌管理器
type TokenManager struct {
	secretKey []byte
	// token 黑名单 - 使用分段锁减少竞争
	blacklist         map[string]time.Time
	blacklistLock     []*sync.RWMutex // 分段锁数组
	blacklistSegments int             // 分段数量

	// 令牌验证结果缓存
	cache     map[string]cacheItem
	cacheLock sync.RWMutex
	cacheSize int
	cacheTTL  time.Duration

	// 清理黑名单的定时器
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}

	// 令牌过期时间设置
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration

	// 选项
	enableLog   bool
	enableCache bool
}

// NewTokenManager 创建新的JWT令牌管理器
func NewTokenManager(secretKey string, options ...*JWTOptions) *TokenManager {
	opts := DefaultJWTOptions()
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}

	// 创建分段锁，减少并发写入的锁竞争
	const numSegments = 16 // 16个分段
	locks := make([]*sync.RWMutex, numSegments)
	for i := 0; i < numSegments; i++ {
		locks[i] = &sync.RWMutex{}
	}

	manager := &TokenManager{
		secretKey:          []byte(secretKey),
		blacklist:          make(map[string]time.Time),
		blacklistLock:      locks,
		blacklistSegments:  numSegments,
		cache:              make(map[string]cacheItem),
		cacheSize:          opts.CacheSize,
		cacheTTL:           opts.CacheTTL,
		enableLog:          opts.EnableLog,
		enableCache:        opts.EnableCache,
		stopCleanup:        make(chan struct{}),
		accessTokenExpiry:  opts.AccessTokenExpiry,
		refreshTokenExpiry: opts.RefreshTokenExpiry,
	}

	// 启动黑名单自动清理
	if opts.BlacklistCleanInterval > 0 {
		manager.cleanupTicker = time.NewTicker(opts.BlacklistCleanInterval)
		go manager.startCleanupRoutine()
	}

	return manager
}

// 获取令牌对应的锁索引
func (m *TokenManager) getLockIndex(token string) int {
	// 简单哈希函数，将令牌映射到锁索引
	var sum int
	for i := 0; i < len(token) && i < 10; i++ {
		sum += int(token[i])
	}
	return sum % m.blacklistSegments
}

// startCleanupRoutine 启动黑名单自动清理例程
func (m *TokenManager) startCleanupRoutine() {
	for {
		select {
		case <-m.cleanupTicker.C:
			m.CleanBlacklist()
			m.cleanCache() // 同时清理过期缓存
		case <-m.stopCleanup:
			m.cleanupTicker.Stop()
			return
		}
	}
}

// Shutdown 关闭管理器，停止所有后台任务
func (m *TokenManager) Shutdown() {
	if m.cleanupTicker != nil {
		close(m.stopCleanup)
	}
}

// EnableLog 启用日志记录
func (m *TokenManager) EnableLog(enable bool) {
	m.enableLog = enable
}

// EnableCache 启用缓存
func (m *TokenManager) EnableCache(enable bool) {
	m.enableCache = enable
}

// SetCacheTTL 设置缓存过期时间
func (m *TokenManager) SetCacheTTL(ttl time.Duration) {
	m.cacheTTL = ttl
}

// SetCacheSize 设置缓存大小
func (m *TokenManager) SetCacheSize(size int) {
	m.cacheSize = size
}

// SetTokenExpiry 设置令牌过期时间
func (m *TokenManager) SetTokenExpiry(tokenType TokenType, expiry time.Duration) {
	if tokenType == AccessToken {
		m.accessTokenExpiry = expiry
	} else if tokenType == RefreshToken {
		m.refreshTokenExpiry = expiry
	}
}

// logf 内部日志记录函数
func (m *TokenManager) logf(format string, args ...interface{}) {
	if m.enableLog {
		log.Printf(format, args...)
	}
}

// GenerateToken 生成JWT令牌
func (m *TokenManager) GenerateToken(subject string, options ...*TokenOptions) (string, error) {
	// 验证subject不能为空
	if subject == "" {
		return "", errors.New("主题(subject)不能为空")
	}

	// 使用默认选项或者用户提供的选项
	opts := DefaultTokenOptions()
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}

	// 确定令牌类型和对应的过期时间
	tokenType := opts.TokenType
	if tokenType == "" {
		tokenType = AccessToken
	}

	var expiresIn time.Duration
	if opts.ExpiresIn > 0 {
		expiresIn = opts.ExpiresIn
	} else if tokenType == AccessToken {
		expiresIn = m.accessTokenExpiry
	} else {
		expiresIn = m.refreshTokenExpiry
	}

	// 生成唯一令牌ID，如果没有提供
	tokenID := opts.TokenID
	if tokenID == "" {
		tokenID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// 构建基本声明
	now := time.Now()
	claims := &StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
		},
		Subject:   subject,
		TokenType: tokenType,
		SessionID: opts.SessionID,
		TokenID:   tokenID,
	}

	// 添加自定义声明
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	if opts.CustomClaims != nil {
		for k, v := range opts.CustomClaims {
			// 不能直接将StandardClaims转为MapClaims
			// 使用RegisteredClaims的私有字段存储自定义声明
			if mapClaims, ok := token.Claims.(jwt.MapClaims); ok {
				mapClaims[k] = v
			}
		}
	}

	// 签名生成令牌
	tokenStr, err := token.SignedString(m.secretKey)
	if err != nil {
		m.logf("令牌签名失败: %v", err)
		return "", err
	}

	if m.enableLog {
		m.logf("已生成%s令牌，主题: %s, 过期时间: %v",
			tokenType, subject, expiresIn)
	}

	return tokenStr, nil
}

// ValidateToken 验证JWT令牌并返回声明
func (m *TokenManager) ValidateToken(tokenStr string) (*StandardClaims, error) {
	// 先检查缓存以提高性能
	if m.enableCache {
		if claims, err, found := m.checkCache(tokenStr); found {
			return claims, err
		}
	}

	// 快速检查是否在黑名单中
	if m.IsBlacklisted(tokenStr) {
		if m.enableCache {
			m.cacheResult(tokenStr, nil, errors.New("令牌已被撤销"))
		}
		return nil, errors.New("令牌已被撤销")
	}

	// 进行预检查，避免解析无效token
	if !m.isTokenFormatValid(tokenStr) {
		if m.enableCache {
			m.cacheResult(tokenStr, nil, errors.New("令牌格式无效"))
		}
		return nil, errors.New("令牌格式无效")
	}

	// 解析并验证令牌
	token, err := jwt.ParseWithClaims(tokenStr, &StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	// 如果解析出错
	if err != nil {
		if m.enableCache {
			m.cacheResult(tokenStr, nil, err)
		}
		return nil, err
	}

	// 如果验证通过
	if claims, ok := token.Claims.(*StandardClaims); ok && token.Valid {
		// 缓存验证成功的结果
		if m.enableCache {
			m.cacheResult(tokenStr, claims, nil)
		}
		return claims, nil
	}

	// 缓存无效令牌结果
	if m.enableCache {
		m.cacheResult(tokenStr, nil, errors.New("无效的令牌"))
	}
	return nil, errors.New("无效的令牌")
}

// RefreshToken 刷新访问令牌并返回访问令牌和刷新令牌
func (m *TokenManager) RefreshToken(refreshTokenStr string) (accessToken string, refreshToken string, err error) {
	// 验证刷新令牌
	claims, err := m.ValidateToken(refreshTokenStr)
	if err != nil {
		return "", "", fmt.Errorf("刷新令牌验证失败: %w", err)
	}

	// 确保是刷新令牌类型
	if claims.TokenType != RefreshToken {
		return "", "", errors.New("提供的不是有效的刷新令牌")
	}

	// 创建新的访问令牌
	options := &TokenOptions{
		TokenType: AccessToken,
		SessionID: claims.SessionID,
	}

	accessToken, err = m.GenerateToken(claims.Subject, options)
	if err != nil {
		return "", "", fmt.Errorf("生成访问令牌失败: %w", err)
	}

	return accessToken, refreshTokenStr, nil
}

// 检查令牌格式是否有效（快速预检查）
func (m *TokenManager) isTokenFormatValid(tokenStr string) bool {
	// 检查令牌最小长度
	if len(tokenStr) < 10 {
		return false
	}

	// 检查JWT的基本格式：确保有两个点分隔的三部分
	parts := 0
	for _, c := range tokenStr {
		if c == '.' {
			parts++
		}
	}
	return parts == 2
}

// 检查缓存中是否有验证结果
func (m *TokenManager) checkCache(tokenStr string) (*StandardClaims, error, bool) {
	m.cacheLock.RLock()
	item, exists := m.cache[tokenStr]
	m.cacheLock.RUnlock()

	if !exists {
		return nil, nil, false
	}

	// 检查缓存项是否过期
	if time.Since(item.timestamp) > m.cacheTTL {
		// 缓存已过期，移除它
		m.cacheLock.Lock()
		delete(m.cache, tokenStr)
		m.cacheLock.Unlock()
		return nil, nil, false
	}

	// 返回缓存的结果
	if m.enableLog {
		m.logf("使用缓存结果验证令牌: %s...", tokenStr[:10])
	}
	return item.claims, item.err, true
}

// 缓存验证结果
func (m *TokenManager) cacheResult(tokenStr string, claims *StandardClaims, err error) {
	m.cacheLock.Lock()
	defer m.cacheLock.Unlock()

	// 如果缓存已满，先清理部分旧数据
	if len(m.cache) >= m.cacheSize {
		m.evictOldestCache(m.cacheSize / 5) // 清理20%的缓存
	}

	// 添加到缓存
	m.cache[tokenStr] = cacheItem{
		claims:    claims,
		err:       err,
		timestamp: time.Now(),
	}
}

// 清理最旧的部分缓存
func (m *TokenManager) evictOldestCache(count int) {
	// 找出最老的缓存项进行清理
	if count <= 0 || len(m.cache) == 0 {
		return
	}

	type cacheAge struct {
		token string
		time  time.Time
	}

	// 收集所有缓存项的时间戳
	items := make([]cacheAge, 0, len(m.cache))
	for token, item := range m.cache {
		items = append(items, cacheAge{token, item.timestamp})
	}

	// 按时间戳排序（不需要完全排序，找到n个最老的即可）
	// 使用简单的选择排序，只排序需要的部分
	for i := 0; i < count && i < len(items); i++ {
		oldest := i
		for j := i + 1; j < len(items); j++ {
			if items[j].time.Before(items[oldest].time) {
				oldest = j
			}
		}
		// 交换
		if oldest != i {
			items[i], items[oldest] = items[oldest], items[i]
		}

		// 删除这个最老的项
		delete(m.cache, items[i].token)
	}
}

// 清理过期的缓存
func (m *TokenManager) cleanCache() {
	m.cacheLock.Lock()
	defer m.cacheLock.Unlock()

	now := time.Now()
	expiredTime := now.Add(-m.cacheTTL)

	// 删除所有过期的缓存项
	for token, item := range m.cache {
		if item.timestamp.Before(expiredTime) {
			delete(m.cache, token)
		}
	}

	if m.enableLog {
		m.logf("已清理过期缓存，当前缓存大小: %d", len(m.cache))
	}
}

// RevokeToken 撤销令牌（加入黑名单）
func (m *TokenManager) RevokeToken(tokenStr string) error {
	if tokenStr == "" {
		return errors.New("令牌不能为空")
	}

	// 先验证令牌以获取过期时间
	claims, err := m.ValidateToken(tokenStr)
	if err != nil {
		return fmt.Errorf("无法撤销无效的令牌: %w", err)
	}

	// 获取令牌过期时间，确保黑名单条目不会永久保留
	var expireTime time.Time
	if claims.ExpiresAt != nil {
		expireTime = claims.ExpiresAt.Time
	} else {
		// 如果没有过期时间，使用默认的24小时
		expireTime = time.Now().Add(24 * time.Hour)
	}

	if m.enableLog && len(tokenStr) > 10 {
		m.logf("撤销令牌: %s..., 过期时间: %v", tokenStr[:10], expireTime)
	}

	// 使用分段锁减少锁竞争
	lockIndex := m.getLockIndex(tokenStr)
	m.blacklistLock[lockIndex].Lock()
	defer m.blacklistLock[lockIndex].Unlock()

	m.blacklist[tokenStr] = expireTime

	// 从缓存中移除该令牌的验证结果（如果有）
	if m.enableCache {
		m.cacheLock.Lock()
		delete(m.cache, tokenStr)
		m.cacheLock.Unlock()
	}

	return nil
}

// IsBlacklisted 检查令牌是否在黑名单中
func (m *TokenManager) IsBlacklisted(tokenStr string) bool {
	// 找到对应的分段锁
	lockIndex := m.getLockIndex(tokenStr)
	m.blacklistLock[lockIndex].RLock()
	expireAt, exists := m.blacklist[tokenStr]
	m.blacklistLock[lockIndex].RUnlock()

	if !exists {
		return false
	}

	// 如果黑名单过期时间已到，从黑名单中移除
	now := time.Now()
	if now.After(expireAt) {
		if m.enableLog && len(tokenStr) > 10 {
			m.logf("令牌在黑名单中但已过期，移除: %s...", tokenStr[:10])
		}

		// 获取写锁删除过期条目
		m.blacklistLock[lockIndex].Lock()
		delete(m.blacklist, tokenStr)
		m.blacklistLock[lockIndex].Unlock()

		return false
	}

	if m.enableLog && len(tokenStr) > 10 {
		m.logf("令牌在黑名单中: %s..., 将在 %v 过期", tokenStr[:10], expireAt)
	}
	return true
}

// CleanBlacklist 清理过期的黑名单记录
func (m *TokenManager) CleanBlacklist() {
	now := time.Now()
	cleaned := 0

	// 逐个分段清理，减少锁持有时间
	for i := 0; i < m.blacklistSegments; i++ {
		m.blacklistLock[i].Lock()

		// 收集当前分段中的过期令牌
		var expiredTokens []string
		for token, expireAt := range m.blacklist {
			// 计算锁索引，确保只清理当前分段的令牌
			if m.getLockIndex(token) == i && now.After(expireAt) {
				expiredTokens = append(expiredTokens, token)
			}
		}

		// 删除收集到的过期令牌
		for _, token := range expiredTokens {
			delete(m.blacklist, token)
			cleaned++
		}

		m.blacklistLock[i].Unlock()
	}

	if m.enableLog && cleaned > 0 {
		m.logf("已清理 %d 条过期的黑名单记录", cleaned)
	}
}

// GetBlacklistSize 返回黑名单大小
func (m *TokenManager) GetBlacklistSize() int {
	total := 0
	for i := 0; i < m.blacklistSegments; i++ {
		m.blacklistLock[i].RLock()
		// 只统计当前分段负责的令牌数量
		for token := range m.blacklist {
			if m.getLockIndex(token) == i {
				total++
			}
		}
		m.blacklistLock[i].RUnlock()
	}
	return total
}

// GetCacheSize 返回缓存大小
func (m *TokenManager) GetCacheSize() int {
	m.cacheLock.RLock()
	defer m.cacheLock.RUnlock()
	return len(m.cache)
}
