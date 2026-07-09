package jwt

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
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

// RefreshTokenReuseAction refresh token 被检测到复用(重放)时要执行的动作
// 单实例模式下，该动作会通过黑名单实现“立即失效”。
type RefreshTokenReuseAction int

const (
	// RevokeSessionOnRefreshReuse 表示发现 refresh token 复用时撤销整个 session(sid)
	RevokeSessionOnRefreshReuse RefreshTokenReuseAction = iota
)

// StandardClaims 标准JWT声明结构体
type StandardClaims struct {
	jwt.RegisteredClaims
	TokenType TokenType `json:"type,omitempty"`
	SessionID string    `json:"sid,omitempty"`

	TokenID      string                 `json:"jti,omitempty"`
	CustomClaims map[string]interface{} `json:"-"`
}

var parseWithClaims = jwt.ParseWithClaims

func (c *StandardClaims) MarshalJSON() ([]byte, error) {
	m := jwt.MapClaims{}
	for k, v := range c.CustomClaims {
		m[k] = v
	}

	if c.Issuer != "" {
		m["iss"] = c.Issuer
	}
	if c.Subject != "" {
		m["sub"] = c.Subject
	}
	if len(c.Audience) > 0 {
		m["aud"] = c.Audience
	}
	if c.ExpiresAt != nil {
		m["exp"] = c.ExpiresAt.Unix()
	}
	if c.NotBefore != nil {
		m["nbf"] = c.NotBefore.Unix()
	}
	if c.IssuedAt != nil {
		m["iat"] = c.IssuedAt.Unix()
	}
	if c.TokenID != "" {
		m["jti"] = c.TokenID
	} else if c.ID != "" {
		m["jti"] = c.ID
	}

	if c.TokenType != "" {
		m["type"] = c.TokenType
	}
	if c.SessionID != "" {
		m["sid"] = c.SessionID
	}

	return json.Marshal(m)
}

func (c *StandardClaims) UnmarshalJSON(data []byte) error {
	var m jwt.MapClaims
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	custom := make(map[string]interface{})
	for k, v := range m {
		custom[k] = v
	}

	subject, _ := m.GetSubject()
	issuer, _ := m.GetIssuer()
	aud, _ := m.GetAudience()
	issuedAt, _ := m.GetIssuedAt()
	expiresAt, _ := m.GetExpirationTime()
	notBefore, _ := m.GetNotBefore()

	jti := ""
	if v, ok := m["jti"]; ok {
		if s, ok := v.(string); ok {
			jti = s
		} else if f, ok := v.(float64); ok {
			jti = fmt.Sprintf("%.0f", f)
		}
	}

	c.RegisteredClaims = jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   subject,
		Audience:  aud,
		ExpiresAt: expiresAt,
		NotBefore: notBefore,
		IssuedAt:  issuedAt,
		ID:        jti,
	}
	c.TokenID = jti

	if v, ok := m["type"]; ok {
		if s, ok := v.(string); ok {
			c.TokenType = TokenType(s)
		}
		delete(custom, "type")
	}
	if v, ok := m["sid"]; ok {
		if s, ok := v.(string); ok {
			c.SessionID = s
		}
		delete(custom, "sid")
	}

	for _, k := range []string{"iss", "sub", "aud", "exp", "nbf", "iat", "jti"} {
		delete(custom, k)
	}

	c.CustomClaims = custom
	return nil
}

// TokenOptions JWT令牌选项
type TokenOptions struct {
	// 令牌过期时间，覆盖默认值
	ExpiresIn time.Duration
	// 签发者，覆盖管理器默认值
	Issuer string
	// 受众，覆盖管理器默认值
	Audience []string
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
	// 默认签发者
	Issuer string
	// 默认受众
	Audience []string
	// TokenProfile 定义业务身份字段写入 custom claims 时的标准 key。
	TokenProfile *TokenProfile
	// RefreshTokenReuseAction refresh token 被复用(重放)时的动作
	RefreshTokenReuseAction RefreshTokenReuseAction
}

// DefaultJWTOptions 返回默认的JWT管理器选项
func DefaultJWTOptions() *JWTOptions {
	return &JWTOptions{
		EnableLog:               false,                       // 默认不启用日志
		BlacklistCleanInterval:  10 * time.Minute,            // 默认10分钟清理一次黑名单
		EnableCache:             true,                        // 默认启用缓存
		CacheSize:               1000,                        // 默认缓存1000个结果
		CacheTTL:                5 * time.Minute,             // 默认缓存5分钟
		AccessTokenExpiry:       15 * time.Minute,            // 默认访问令牌15分钟过期
		RefreshTokenExpiry:      24 * time.Hour,              // 默认刷新令牌24小时过期
		RefreshTokenReuseAction: RevokeSessionOnRefreshReuse, // 默认发现复用就封禁整个 session
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
	secretKey              []byte
	blacklist              []map[string]time.Time
	blacklistLock          []*sync.RWMutex
	blacklistSegments      int
	blacklistCleanInterval time.Duration

	// refresh token 复用检测（单实例内存版）
	usedRefreshJTIs     []map[string]time.Time
	usedRefreshJTIsLock []*sync.RWMutex

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

	issuer       string
	audience     []string
	tokenProfile TokenProfile

	// 选项
	enableLog               bool
	enableCache             bool
	refreshTokenReuseAction RefreshTokenReuseAction
}

// NewTokenManager 创建新的JWT令牌管理器
func NewTokenManager(secretKey string, options ...*JWTOptions) (*TokenManager, error) {
	// Validate secret key strength
	if len(secretKey) < 32 {
		return nil, errors.New("JWT secret key must be at least 32 characters long for security")
	}

	// Check key entropy (basic check for character variety)
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, c := range secretKey {
		switch {
		case c >= 'A' && c <= 'Z':
			hasUpper = true
		case c >= 'a' && c <= 'z':
			hasLower = true
		case c >= '0' && c <= '9':
			hasNumber = true
		default:
			hasSpecial = true
		}
	}

	variety := 0
	if hasUpper {
		variety++
	}
	if hasLower {
		variety++
	}
	if hasNumber {
		variety++
	}
	if hasSpecial {
		variety++
	}

	if variety < 3 {
		return nil, errors.New("JWT secret key should contain at least 3 different character types (uppercase, lowercase, numbers, special characters)")
	}

	opts := DefaultJWTOptions()
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}

	const numSegments = 16
	locks := make([]*sync.RWMutex, numSegments)
	blacklist := make([]map[string]time.Time, numSegments)
	usedLocks := make([]*sync.RWMutex, numSegments)
	usedJTIs := make([]map[string]time.Time, numSegments)
	for i := 0; i < numSegments; i++ {
		locks[i] = &sync.RWMutex{}
		blacklist[i] = make(map[string]time.Time)
		usedLocks[i] = &sync.RWMutex{}
		usedJTIs[i] = make(map[string]time.Time)
	}

	manager := &TokenManager{
		secretKey:               []byte(secretKey),
		blacklist:               blacklist,
		blacklistLock:           locks,
		blacklistSegments:       numSegments,
		blacklistCleanInterval:  opts.BlacklistCleanInterval,
		usedRefreshJTIs:         usedJTIs,
		usedRefreshJTIsLock:     usedLocks,
		cache:                   make(map[string]cacheItem),
		cacheSize:               opts.CacheSize,
		cacheTTL:                opts.CacheTTL,
		enableLog:               opts.EnableLog,
		enableCache:             opts.EnableCache,
		refreshTokenReuseAction: opts.RefreshTokenReuseAction,
		stopCleanup:             make(chan struct{}),
		accessTokenExpiry:       opts.AccessTokenExpiry,
		refreshTokenExpiry:      opts.RefreshTokenExpiry,
		issuer:                  opts.Issuer,
		audience:                cloneStringSlice(opts.Audience),
		tokenProfile:            normalizeTokenProfile(opts.TokenProfile),
	}

	if opts.BlacklistCleanInterval > 0 {
		manager.cleanupTicker = time.NewTicker(opts.BlacklistCleanInterval)
		go manager.startCleanupRoutine()
	}

	return manager, nil
}

// MustNewTokenManager creates a new JWT token manager and panics on error
// This function maintains backward compatibility with existing code
func MustNewTokenManager(secretKey string, options ...*JWTOptions) *TokenManager {
	manager, err := NewTokenManager(secretKey, options...)
	if err != nil {
		panic(err)
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

func (m *TokenManager) getJTIIndex(jti string) int {
	// 复用 blacklistSegments 的分段数
	var sum int
	for i := 0; i < len(jti) && i < 10; i++ {
		sum += int(jti[i])
	}
	return sum % m.blacklistSegments
}

// consumeRefreshJTI 标记 refresh token 的 jti 为已使用。
// 返回 true 表示首次使用；false 表示已经用过（疑似重放）。
func (m *TokenManager) consumeRefreshJTI(jti string, expireAt time.Time) bool {
	idx := m.getJTIIndex(jti)
	m.usedRefreshJTIsLock[idx].Lock()
	defer m.usedRefreshJTIsLock[idx].Unlock()

	if t, ok := m.usedRefreshJTIs[idx][jti]; ok {
		// 已记录过的 jti，如果记录已过期则视为可再次使用（兜底）。
		if time.Now().After(t) {
			m.usedRefreshJTIs[idx][jti] = expireAt
			return true
		}
		return false
	}
	m.usedRefreshJTIs[idx][jti] = expireAt
	return true
}

func (m *TokenManager) cleanUsedRefreshJTIs() {
	if m.cacheTTL <= 0 {
		// 允许 ttl<=0 的情况仍然清理 jti 表
	}
	now := time.Now()
	cleaned := 0
	for i := 0; i < m.blacklistSegments; i++ {
		m.usedRefreshJTIsLock[i].Lock()
		for jti, expireAt := range m.usedRefreshJTIs[i] {
			if now.After(expireAt) {
				delete(m.usedRefreshJTIs[i], jti)
				cleaned++
			}
		}
		m.usedRefreshJTIsLock[i].Unlock()
	}
	if m.enableLog && cleaned > 0 {
		m.logf("已清理 %d 条过期的 refresh jti 记录", cleaned)
	}
}

func (m *TokenManager) revokeSession(sessionID string) {
	if sessionID == "" {
		return
	}

	// 单实例实现：把当前缓存/黑名单机制提升到 session 级别，通过撤销 session 标记来立即拒绝。
	// 这里采用一个约定 key："sid:<sessionID>"，并存入 blacklist，过期时间取 refreshTokenExpiry。
	key := "sid:" + sessionID
	expireAt := time.Now().Add(m.refreshTokenExpiry)
	idx := m.getLockIndex(key)
	m.blacklistLock[idx].Lock()
	m.blacklist[idx][key] = expireAt
	m.blacklistLock[idx].Unlock()
}

func (m *TokenManager) isSessionRevoked(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	key := "sid:" + sessionID
	return m.IsBlacklisted(key)
}

// startCleanupRoutine 启动黑名单自动清理例程
func (m *TokenManager) startCleanupRoutine() {
	for {
		select {
		case <-m.cleanupTicker.C:
			m.CleanBlacklist()
			m.cleanCache()           // 同时清理过期缓存
			m.cleanUsedRefreshJTIs() // 同时清理过期的 refresh jti 记录
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

// SetIssuer 设置默认 JWT 签发者。
func (m *TokenManager) SetIssuer(issuer string) {
	m.issuer = issuer
}

// SetAudience 设置默认 JWT 受众。
func (m *TokenManager) SetAudience(audience ...string) {
	m.audience = cloneStringSlice(audience)
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
		return "", errors.New("subject cannot be empty")
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

	// 生成唯一令牌ID，如果没有提供。
	// 使用 CSPRNG，避免可预测的时间戳 ID。
	tokenID := opts.TokenID
	if tokenID == "" {
		generatedID, err := generateRandomHex(16)
		if err != nil {
			return "", fmt.Errorf("failed to generate token id: %w", err)
		}
		tokenID = generatedID
	}

	// 构建基本声明
	now := time.Now()
	issuer := opts.Issuer
	if issuer == "" {
		issuer = m.issuer
	}
	audience := cloneStringSlice(opts.Audience)
	if len(audience) == 0 {
		audience = cloneStringSlice(m.audience)
	}
	claims := &StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
			Issuer:    issuer,
			Audience:  audience,
		},
		TokenType: tokenType,
		SessionID: opts.SessionID,
		TokenID:   tokenID,
	}

	claims.RegisteredClaims.Subject = subject
	claims.CustomClaims = opts.CustomClaims

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

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
			m.cacheResult(tokenStr, nil, errors.New("token has been revoked"))
		}
		return nil, errors.New("token has been revoked")
	}

	// 进行预检查，避免解析无效token
	if !m.isTokenFormatValid(tokenStr) {
		if m.enableCache {
			m.cacheResult(tokenStr, nil, errors.New("invalid token format"))
		}
		return nil, errors.New("invalid token format")
	}

	// 解析并验证令牌
	token, err := parseWithClaims(tokenStr, &StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
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
		// session 级撤销：一旦 session 被标记撤销，则所有该 session 的 token 都应拒绝
		if m.isSessionRevoked(claims.SessionID) {
			err := errors.New("session has been revoked")
			if m.enableCache {
				m.cacheResult(tokenStr, nil, err)
			}
			return nil, err
		}

		// 缓存验证成功的结果
		if m.enableCache {
			m.cacheResult(tokenStr, claims, nil)
		}
		return claims, nil
	}

	// 缓存无效令牌结果
	if m.enableCache {
		m.cacheResult(tokenStr, nil, errors.New("invalid token"))
	}
	return nil, errors.New("invalid token")
}

// RefreshToken 刷新访问令牌并返回新的访问令牌和新的刷新令牌（rotation）。
// 当检测到 refresh token 复用(重放)时，会按配置执行动作（默认封禁整个 session）。
func (m *TokenManager) RefreshToken(refreshTokenStr string) (accessToken string, refreshToken string, err error) {
	// 验证刷新令牌
	claims, err := m.ValidateToken(refreshTokenStr)
	if err != nil {
		return "", "", fmt.Errorf("刷新令牌验证失败: %w", err)
	}

	// 如果 session 已被撤销，直接拒绝
	if m.isSessionRevoked(claims.SessionID) {
		return "", "", errors.New("session has been revoked")
	}

	// 确保是刷新令牌类型
	if claims.TokenType != RefreshToken {
		return "", "", errors.New("provided token is not a valid refresh token")
	}

	// refresh token 必须带 jti，用于一次性消费
	if claims.TokenID == "" {
		return "", "", errors.New("refresh token missing jti")
	}

	expireAt := time.Now().Add(m.refreshTokenExpiry)
	if claims.ExpiresAt != nil {
		expireAt = claims.ExpiresAt.Time
	}

	// 原子消费 jti：如果第二次使用，则判定为复用/重放
	if first := m.consumeRefreshJTI(claims.TokenID, expireAt); !first {
		switch m.refreshTokenReuseAction {
		case RevokeSessionOnRefreshReuse:
			m.revokeSession(claims.SessionID)
		}
		return "", "", errors.New("refresh token reuse detected")
	}

	// rotation：生成新的 refresh token（同 sid）
	newRefresh, err := m.GenerateToken(claims.RegisteredClaims.Subject, &TokenOptions{
		TokenType:    RefreshToken,
		Issuer:       claims.Issuer,
		Audience:     cloneStringSlice(claims.Audience),
		SessionID:    claims.SessionID,
		CustomClaims: cloneMapClaims(claims.CustomClaims),
	})
	if err != nil {
		return "", "", fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	// 生成新的 access token
	newAccess, err := m.GenerateToken(claims.RegisteredClaims.Subject, &TokenOptions{
		TokenType:    AccessToken,
		Issuer:       claims.Issuer,
		Audience:     cloneStringSlice(claims.Audience),
		SessionID:    claims.SessionID,
		CustomClaims: cloneMapClaims(claims.CustomClaims),
	})
	if err != nil {
		return "", "", fmt.Errorf("生成访问令牌失败: %w", err)
	}

	// 旧 refresh 立即作废
	_ = m.RevokeToken(refreshTokenStr)

	return newAccess, newRefresh, nil
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
		prefixLen := 10
		if len(tokenStr) < prefixLen {
			prefixLen = len(tokenStr)
		}
		m.logf("使用缓存结果验证令牌: %s...", tokenStr[:prefixLen])
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
	if count <= 0 || len(m.cache) == 0 {
		return
	}

	type cacheAge struct {
		token string
		time  time.Time
	}

	items := make([]cacheAge, 0, len(m.cache))
	for token, item := range m.cache {
		items = append(items, cacheAge{token: token, time: item.timestamp})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].time.Before(items[j].time)
	})

	if count > len(items) {
		count = len(items)
	}
	for i := 0; i < count; i++ {
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
		return errors.New("token cannot be empty")
	}

	claims, err := m.ValidateToken(tokenStr)
	if err != nil {
		return fmt.Errorf("无法撤销无效的令牌: %w", err)
	}

	var expireTime time.Time
	if claims.ExpiresAt != nil {
		expireTime = claims.ExpiresAt.Time
	} else {
		expireTime = time.Now().Add(24 * time.Hour)
	}

	if m.enableLog && len(tokenStr) > 10 {
		m.logf("撤销令牌: %s..., 过期时间: %v", tokenStr[:10], expireTime)
	}

	idx := m.getLockIndex(tokenStr)
	m.blacklistLock[idx].Lock()
	m.blacklist[idx][tokenStr] = expireTime
	m.blacklistLock[idx].Unlock()

	if m.enableCache {
		m.cacheLock.Lock()
		delete(m.cache, tokenStr)
		m.cacheLock.Unlock()
	}

	return nil
}

// IsBlacklisted 检查令牌是否在黑名单中
func (m *TokenManager) IsBlacklisted(tokenStr string) bool {
	idx := m.getLockIndex(tokenStr)
	m.blacklistLock[idx].RLock()
	expireAt, exists := m.blacklist[idx][tokenStr]
	m.blacklistLock[idx].RUnlock()

	if !exists {
		return false
	}

	now := time.Now()
	if now.After(expireAt) {
		if m.enableLog && len(tokenStr) > 10 {
			m.logf("令牌在黑名单中但已过期，移除: %s...", tokenStr[:10])
		}

		m.blacklistLock[idx].Lock()
		if expireAt2, exists2 := m.blacklist[idx][tokenStr]; exists2 && now.After(expireAt2) {
			delete(m.blacklist[idx], tokenStr)
		}
		m.blacklistLock[idx].Unlock()
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

	for i := 0; i < m.blacklistSegments; i++ {
		m.blacklistLock[i].Lock()
		for token, expireAt := range m.blacklist[i] {
			if now.After(expireAt) {
				delete(m.blacklist[i], token)
				cleaned++
			}
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
		total += len(m.blacklist[i])
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

// GetTokenExpiryConfig 返回当前配置的JWT选项
func (m *TokenManager) GetTokenExpiryConfig() *JWTOptions {
	return &JWTOptions{
		EnableLog:               m.enableLog,
		BlacklistCleanInterval:  m.blacklistCleanInterval,
		EnableCache:             m.enableCache,
		CacheSize:               m.cacheSize,
		CacheTTL:                m.cacheTTL,
		AccessTokenExpiry:       m.accessTokenExpiry,
		RefreshTokenExpiry:      m.refreshTokenExpiry,
		Issuer:                  m.issuer,
		Audience:                cloneStringSlice(m.audience),
		TokenProfile:            m.tokenProfile.clone(),
		RefreshTokenReuseAction: m.refreshTokenReuseAction,
	}
}

// GetAccessTokenExpiry 返回访问令牌过期时间
func (m *TokenManager) GetAccessTokenExpiry() time.Duration {
	return m.accessTokenExpiry
}

// GetRefreshTokenExpiry 返回刷新令牌过期时间
func (m *TokenManager) GetRefreshTokenExpiry() time.Duration {
	return m.refreshTokenExpiry
}
