package jwt

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var parseWithClaims = jwt.ParseWithClaims

// TokenManager 管理 JWT 令牌签发、按用途隔离验证、刷新轮换及令牌/会话吊销。
type TokenManager struct {
	secretKey              []byte
	revocations            RevocationStore
	refreshStore           RefreshTokenUseStore
	blacklistCleanInterval time.Duration
	clock                  func() time.Time

	cache *validationCache

	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	shutdownOnce  sync.Once

	tokenProfile TokenProfile

	configMu                sync.RWMutex
	cacheSize               int
	cacheTTL                time.Duration
	accessTokenExpiry       time.Duration
	refreshTokenExpiry      time.Duration
	issuer                  string
	audience                []string
	expectedIssuer          string
	expectedAudience        []string
	enableLog               bool
	enableCache             bool
	refreshTokenReuseAction RefreshTokenReuseAction
	errorReporter           func(error)
}

// NewTokenManager 创建新的 JWT 令牌管理器。密钥按 UTF-8 字节计至少 32 字节。
func NewTokenManager(secretKey string, options ...*JWTOptions) (*TokenManager, error) {
	if len([]byte(secretKey)) < 32 {
		return nil, fmt.Errorf("%w: got %d bytes", ErrSecretKeyTooShort, len([]byte(secretKey)))
	}

	opts := DefaultJWTOptions()
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}
	if err := validateJWTOptions(opts); err != nil {
		return nil, err
	}

	clock := opts.Clock
	if clock == nil {
		clock = time.Now
	}

	revocations := opts.RevocationStore
	if revocations == nil {
		revocations = NewMemoryRevocationStoreWithClock(clock)
	}
	refreshStore := opts.RefreshTokenUseStore
	if refreshStore == nil {
		refreshStore = NewMemoryRefreshTokenUseStoreWithClock(clock)
	}
	reporter := opts.ErrorReporter
	if reporter == nil {
		reporter = defaultErrorReporter
	}

	manager := &TokenManager{
		secretKey:               []byte(secretKey),
		revocations:             revocations,
		refreshStore:            refreshStore,
		blacklistCleanInterval:  opts.BlacklistCleanInterval,
		clock:                   clock,
		cache:                   newValidationCache(),
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
		expectedIssuer:          opts.ExpectedIssuer,
		expectedAudience:        cloneStringSlice(opts.ExpectedAudience),
		tokenProfile:            normalizeTokenProfile(opts.TokenProfile),
		errorReporter:           reporter,
	}

	if opts.BlacklistCleanInterval > 0 {
		manager.cleanupTicker = time.NewTicker(opts.BlacklistCleanInterval)
		go manager.startCleanupRoutine()
	}

	return manager, nil
}

// MustNewTokenManager 创建新的 JWT 令牌管理器，配置非法时直接 panic（仅建议在确定性初始化中使用）。
func MustNewTokenManager(secretKey string, options ...*JWTOptions) *TokenManager {
	manager, err := NewTokenManager(secretKey, options...)
	if err != nil {
		panic(err)
	}
	return manager
}

func (m *TokenManager) now() time.Time {
	if m.clock != nil {
		return m.clock()
	}
	return time.Now()
}

// startCleanupRoutine 启动吊销记录、refresh 消费记录和验证缓存后台清理协程。
func (m *TokenManager) startCleanupRoutine() {
	for {
		select {
		case <-m.cleanupTicker.C:
			now := m.now()
			if cleaner, ok := m.revocations.(RevocationStoreCleaner); ok {
				if _, err := cleaner.CleanExpired(context.Background(), now); err != nil {
					m.reportError(fmt.Errorf("clean revocation store: %w", err))
				}
			}
			cleanedCache := m.cache.cleanExpired(now)
			if cleanedCache > 0 {
				m.logf("已清理过期缓存 %d 条，当前缓存大小: %d", cleanedCache, m.cache.size())
			}
			if cleaner, ok := m.refreshStore.(RefreshTokenUseStoreCleaner); ok {
				if _, err := cleaner.CleanExpired(context.Background(), now); err != nil {
					m.reportError(fmt.Errorf("clean refresh jti store: %w", err))
				}
			}
		case <-m.stopCleanup:
			m.cleanupTicker.Stop()
			return
		}
	}
}

// Shutdown 关闭管理器，停止后台清理协程。幂等且并发安全。
func (m *TokenManager) Shutdown() {
	m.shutdownOnce.Do(func() {
		if m.cleanupTicker != nil {
			close(m.stopCleanup)
		}
	})
}

// EnableLog 动态启用或禁用调试日志。
func (m *TokenManager) EnableLog(enable bool) {
	m.configMu.Lock()
	m.enableLog = enable
	m.configMu.Unlock()
}

// EnableCache 动态启用或禁用验证缓存；禁用时立即清空旧缓存，防止再次启用时复活陈旧条目。
func (m *TokenManager) EnableCache(enable bool) {
	m.configMu.Lock()
	m.enableCache = enable
	m.configMu.Unlock()
	if !enable {
		m.cache.clear()
	}
}

// SetCacheTTL 动态调整缓存最大存活时长，并立即收紧已缓存条目的有效期上限。
func (m *TokenManager) SetCacheTTL(ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidCacheTTL
	}
	m.configMu.Lock()
	m.cacheTTL = ttl
	m.configMu.Unlock()
	m.cache.clampTTL(m.now(), ttl)
	return nil
}

// SetCacheSize 动态调整缓存最大容量，并立即淘汰超出新容量的最旧条目。
func (m *TokenManager) SetCacheSize(size int) error {
	if size <= 0 {
		return ErrInvalidCacheSize
	}
	m.configMu.Lock()
	m.cacheSize = size
	m.configMu.Unlock()
	m.cache.resize(size)
	return nil
}

// SetTokenExpiry 动态设置指定类型令牌的默认有效期。
func (m *TokenManager) SetTokenExpiry(tokenType TokenType, expiry time.Duration) error {
	if expiry <= 0 {
		return ErrInvalidTokenExpiry
	}
	m.configMu.Lock()
	defer m.configMu.Unlock()
	switch tokenType {
	case AccessToken:
		m.accessTokenExpiry = expiry
	case RefreshToken:
		m.refreshTokenExpiry = expiry
	default:
		return ErrUnknownTokenType
	}
	return nil
}

// SetIssuer 设置默认签发者（仅影响后续签发）。
func (m *TokenManager) SetIssuer(issuer string) {
	m.configMu.Lock()
	m.issuer = issuer
	m.configMu.Unlock()
}

// SetAudience 设置默认受众（仅影响后续签发）。
func (m *TokenManager) SetAudience(audience ...string) {
	m.configMu.Lock()
	m.audience = cloneStringSlice(audience)
	m.configMu.Unlock()
}

func (m *TokenManager) snapshotConfig() managerSnapshot {
	m.configMu.RLock()
	defer m.configMu.RUnlock()
	return managerSnapshot{
		enableCache:             m.enableCache,
		enableLog:               m.enableLog,
		cacheSize:               m.cacheSize,
		cacheTTL:                m.cacheTTL,
		accessTokenExpiry:       m.accessTokenExpiry,
		refreshTokenExpiry:      m.refreshTokenExpiry,
		issuer:                  m.issuer,
		audience:                cloneStringSlice(m.audience),
		expectedIssuer:          m.expectedIssuer,
		expectedAudience:        cloneStringSlice(m.expectedAudience),
		refreshTokenReuseAction: m.refreshTokenReuseAction,
		errorReporter:           m.errorReporter,
	}
}

func (m *TokenManager) reportError(err error) {
	if err == nil {
		return
	}
	reporter := m.snapshotConfig().errorReporter
	if reporter == nil {
		reporter = defaultErrorReporter
	}
	reporter(err)
}

func defaultErrorReporter(err error) {
	if err == nil {
		return
	}
	log.Printf("jwt manager error: %s", err.Error())
}

func (m *TokenManager) logf(format string, args ...interface{}) {
	if m.snapshotConfig().enableLog {
		log.Printf(format, args...)
	}
}

// GenerateToken 签发底层 JWT 令牌。业务侧优先推荐使用 IssueTokenPair / IssueAccessToken。
func (m *TokenManager) GenerateToken(subject string, options ...*TokenOptions) (string, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return "", ErrEmptySubject
	}

	opts := DefaultTokenOptions()
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}

	tokenType := opts.TokenType
	if tokenType == "" {
		tokenType = AccessToken
	}
	if !isValidTokenType(tokenType) {
		return "", fmt.Errorf("%w: %q", ErrUnknownTokenType, tokenType)
	}
	if opts.ExpiresIn < 0 {
		return "", ErrInvalidTokenExpiry
	}
	if err := validateCustomClaimsKeys(opts.CustomClaims); err != nil {
		return "", err
	}

	snap := m.snapshotConfig()
	var expiresIn time.Duration
	switch {
	case opts.ExpiresIn > 0:
		expiresIn = opts.ExpiresIn
	case tokenType == AccessToken:
		expiresIn = snap.accessTokenExpiry
	default:
		expiresIn = snap.refreshTokenExpiry
	}

	tokenID := strings.TrimSpace(opts.TokenID)
	if tokenID == "" {
		generatedID, err := generateRandomHex(16)
		if err != nil {
			return "", fmt.Errorf("failed to generate token id: %w", err)
		}
		tokenID = generatedID
	}

	now := m.now()
	issuer := opts.Issuer
	if issuer == "" {
		issuer = snap.issuer
	}
	audience := cloneStringSlice(opts.Audience)
	if len(audience) == 0 {
		audience = cloneStringSlice(snap.audience)
	}

	claims := &StandardClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			ExpiresAt: jwt.NewNumericDate(now.Add(expiresIn)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ID:        tokenID,
			Issuer:    issuer,
			Audience:  audience,
		},
		TokenType:    tokenType,
		SessionID:    strings.TrimSpace(opts.SessionID),
		TokenID:      tokenID,
		CustomClaims: cloneMapClaims(opts.CustomClaims),
		profile:      m.tokenProfile.clone(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(m.secretKey)
	if err != nil {
		m.logf("令牌签名失败: %v", err)
		return "", err
	}

	m.logf("已生成%s令牌，主题: %s, 过期时间: %v", tokenType, subject, expiresIn)
	return tokenStr, nil
}

// ValidateAccessToken 验证访问令牌（严格要求 TokenType == AccessToken）。
// 若传入 RefreshToken，将返回 ErrTokenTypeMismatch。
func (m *TokenManager) ValidateAccessToken(ctx context.Context, tokenStr string) (*StandardClaims, error) {
	ctx = nonNilContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateRawTokenFormat(tokenStr); err != nil {
		return nil, err
	}

	revoked, err := m.IsTokenRevokedContext(ctx, tokenStr)
	if err != nil {
		return nil, fmt.Errorf("check token revocation: %w", err)
	}
	if revoked {
		m.cache.delete(tokenStr)
		return nil, ErrTokenRevoked
	}

	snap := m.snapshotConfig()
	now := m.now()

	if snap.enableCache {
		if claims, cloneErr, found := m.cache.get(tokenStr, now); found {
			if cloneErr != nil {
				return nil, cloneErr
			}
			if snap.enableLog {
				m.logf("使用缓存结果验证令牌: token_hash=%s", tokenLogID(tokenStr))
			}
			revoked, err := m.isSessionRevoked(ctx, claims.SessionID)
			if err != nil {
				return nil, fmt.Errorf("check session revocation: %w", err)
			}
			if revoked {
				m.cache.delete(tokenStr)
				return nil, ErrSessionRevoked
			}
			return claims, nil
		}
	}

	claims, err := m.parseTokenClaims(tokenStr, true)
	if err != nil {
		return nil, err
	}

	if err := ensureExpectedTokenType(claims.TokenType, AccessToken); err != nil {
		return nil, err
	}

	revoked, err = m.isSessionRevoked(ctx, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("check session revocation: %w", err)
	}
	if revoked {
		return nil, ErrSessionRevoked
	}

	claims.profile = m.tokenProfile.clone()

	if snap.enableCache {
		if cacheErr := m.cache.put(tokenStr, claims, now, snap.cacheTTL, snap.cacheSize); cacheErr != nil {
			return nil, cacheErr
		}
	}
	return claims, nil
}

// ValidateRefreshToken 验证刷新令牌（严格要求 TokenType == RefreshToken，且未被吊销或消费）。
func (m *TokenManager) ValidateRefreshToken(ctx context.Context, tokenStr string) (*StandardClaims, error) {
	ctx = nonNilContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateRawTokenFormat(tokenStr); err != nil {
		return nil, err
	}

	revoked, err := m.IsTokenRevokedContext(ctx, tokenStr)
	if err != nil {
		return nil, fmt.Errorf("check token revocation: %w", err)
	}
	if revoked {
		return nil, ErrTokenRevoked
	}

	claims, err := m.parseTokenClaims(tokenStr, true)
	if err != nil {
		return nil, err
	}

	if err := ensureExpectedTokenType(claims.TokenType, RefreshToken); err != nil {
		return nil, err
	}

	revoked, err = m.isSessionRevoked(ctx, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("check session revocation: %w", err)
	}
	if revoked {
		return nil, ErrSessionRevoked
	}

	if jti := claims.effectiveJTI(); jti != "" {
		consumed, err := m.refreshStore.IsConsumed(ctx, jti)
		if err != nil {
			return nil, fmt.Errorf("check refresh jti consumption: %w", err)
		}
		if consumed {
			return nil, ErrTokenRevoked
		}
	}

	claims.profile = m.tokenProfile.clone()
	return claims, nil
}

// ValidateToken 验证访问令牌并返回声明（等价于 ValidateAccessToken(context.Background(), tokenStr)）。
// 为防止 RefreshToken 被误用为业务接口凭证，此接口仅接受 AccessToken。
func (m *TokenManager) ValidateToken(tokenStr string) (*StandardClaims, error) {
	return m.ValidateAccessToken(context.Background(), tokenStr)
}

// ValidateTokenContext 使用请求上下文验证访问令牌（仅接受 AccessToken）。
// 如需验证或检视 RefreshToken，请显式调用 ValidateRefreshToken。
func (m *TokenManager) ValidateTokenContext(ctx context.Context, tokenStr string) (*StandardClaims, error) {
	return m.ValidateAccessToken(ctx, tokenStr)
}

func ensureExpectedTokenType(got TokenType, want TokenType) error {
	if got == want {
		return nil
	}
	if !isValidTokenType(got) {
		return fmt.Errorf("%w: %q", ErrUnknownTokenType, got)
	}
	return fmt.Errorf("%w: got %q, want %q", ErrTokenTypeMismatch, got, want)
}

func (m *TokenManager) parseTokenClaims(tokenStr string, validateClaims bool) (*StandardClaims, error) {
	snap := m.snapshotConfig()
	parserOptions := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(m.now),
	}
	if validateClaims {
		parserOptions = append(parserOptions, jwt.WithExpirationRequired())
		if snap.expectedIssuer != "" {
			parserOptions = append(parserOptions, jwt.WithIssuer(snap.expectedIssuer))
		}
		if len(snap.expectedAudience) > 0 {
			parserOptions = append(parserOptions, jwt.WithAudience(snap.expectedAudience...))
		}
	} else {
		parserOptions = append(parserOptions, jwt.WithoutClaimsValidation())
	}

	token, err := parseWithClaims(tokenStr, &StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	}, parserOptions...)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*StandardClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if !validateClaims {
		if snap.expectedIssuer != "" && claims.Issuer != snap.expectedIssuer {
			return nil, fmt.Errorf("%w: issuer mismatch", jwt.ErrTokenInvalidIssuer)
		}
		if len(snap.expectedAudience) > 0 && !audienceIntersects(claims.Audience, snap.expectedAudience) {
			return nil, fmt.Errorf("%w: audience mismatch", jwt.ErrTokenInvalidAudience)
		}
	}
	claims.profile = m.tokenProfile.clone()
	return claims, nil
}

func audienceIntersects(tokenAud []string, expected []string) bool {
	for _, want := range expected {
		for _, got := range tokenAud {
			if got == want {
				return true
			}
		}
	}
	return false
}

// RefreshTokenPair 是推荐的刷新入口：验证并一次性消费旧 RefreshToken，返回轮换后的完整 TokenPair。
//
// 状态转换保证：
//  1. 仅接受有效且未过期的 RefreshToken（AccessToken 直接拒绝且不污染消费状态）。
//  2. 新令牌对在提交外部存储前预先在内存中完成签发校验；签发参数错误绝不消耗旧令牌。
//  3. 通过 RefreshTokenUseStore.Consume 原子抢占首次消费权；并发重放只会有唯一调用者成功，其余触发重放处理。
//  4. 若后续写入 RevocationStore 瞬时失败，支持 Release 的消费存储会自动回滚，保证客户端安全重试而不会误封会话。
func (m *TokenManager) RefreshTokenPair(ctx context.Context, refreshTokenStr string) (*TokenPair, error) {
	ctx = nonNilContext(ctx)
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := validateRawTokenFormat(refreshTokenStr); err != nil {
		return nil, fmt.Errorf("刷新令牌验证失败: %w", err)
	}

	claims, err := m.parseTokenClaims(refreshTokenStr, true)
	if err != nil {
		return nil, fmt.Errorf("刷新令牌验证失败: %w", err)
	}
	if typeErr := ensureExpectedTokenType(claims.TokenType, RefreshToken); typeErr != nil {
		return nil, fmt.Errorf("刷新令牌验证失败: %w", typeErr)
	}

	jti := claims.effectiveJTI()
	if jti == "" {
		return nil, ErrMissingTokenID
	}

	sessionRevoked, err := m.isSessionRevoked(ctx, claims.SessionID)
	if err != nil {
		return nil, fmt.Errorf("check session revocation: %w", err)
	}
	if sessionRevoked {
		return nil, fmt.Errorf("刷新令牌验证失败: %w", ErrSessionRevoked)
	}

	// Happens-before 顺序保证：轮换提交时先执行 refreshStore.Consume，再执行 revocations.Revoke；
	// 因此读取时必须先查 IsTokenRevoked，命中后再查 IsConsumed，绝不会出现看到已吊销却漏检已消费的并发窗口。
	tokenRevoked, err := m.IsTokenRevokedContext(ctx, refreshTokenStr)
	if err != nil {
		return nil, fmt.Errorf("check token revocation: %w", err)
	}
	if tokenRevoked {
		consumed, storeErr := m.refreshStore.IsConsumed(ctx, jti)
		if storeErr != nil {
			return nil, fmt.Errorf("check refresh jti consumption: %w", storeErr)
		}
		if consumed {
			if err := m.handleRefreshReuse(ctx, claims); err != nil {
				return nil, err
			}
			return nil, ErrRefreshReuseDetected
		}
		return nil, fmt.Errorf("刷新令牌验证失败: %w", ErrTokenRevoked)
	}

	snap := m.snapshotConfig()

	// 在触碰外部消费状态前，先在内存中生成新令牌对，确保参数异常或上下文取消不会白白烧毁旧 refresh token。
	newRefresh, err := m.GenerateToken(claims.Subject, &TokenOptions{
		TokenType:    RefreshToken,
		ExpiresIn:    snap.refreshTokenExpiry,
		Issuer:       claims.Issuer,
		Audience:     cloneStringSlice(claims.Audience),
		SessionID:    claims.SessionID,
		CustomClaims: cloneMapClaims(claims.CustomClaims),
	})
	if err != nil {
		return nil, fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	newAccess, err := m.GenerateToken(claims.Subject, &TokenOptions{
		TokenType:    AccessToken,
		ExpiresIn:    snap.accessTokenExpiry,
		Issuer:       claims.Issuer,
		Audience:     cloneStringSlice(claims.Audience),
		SessionID:    claims.SessionID,
		CustomClaims: cloneMapClaims(claims.CustomClaims),
	})
	if err != nil {
		return nil, fmt.Errorf("生成访问令牌失败: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	expireAt := m.now().Add(snap.refreshTokenExpiry)
	if claims.ExpiresAt != nil {
		expireAt = claims.ExpiresAt.Time
	}

	first, err := m.refreshStore.Consume(ctx, jti, expireAt)
	if err != nil {
		return nil, fmt.Errorf("consume refresh jti: %w", err)
	}
	if !first {
		if err := m.handleRefreshReuse(ctx, claims); err != nil {
			return nil, err
		}
		return nil, ErrRefreshReuseDetected
	}

	if err := m.revocations.Revoke(ctx, tokenRevocationKey(refreshTokenStr), expireAt); err != nil {
		if releaser, ok := m.refreshStore.(RefreshTokenUseReleaser); ok {
			_ = releaser.Release(context.Background(), jti)
		}
		return nil, fmt.Errorf("revoke rotated refresh token: %w", err)
	}
	m.cache.delete(refreshTokenStr)

	return &TokenPair{
		AccessToken:      newAccess,
		RefreshToken:     newRefresh,
		TokenType:        "Bearer",
		SessionID:        claims.SessionID,
		AccessExpiresIn:  int64(snap.accessTokenExpiry.Seconds()),
		RefreshExpiresIn: int64(snap.refreshTokenExpiry.Seconds()),
	}, nil
}

func (m *TokenManager) handleRefreshReuse(ctx context.Context, claims *StandardClaims) error {
	snap := m.snapshotConfig()
	switch snap.refreshTokenReuseAction {
	case RevokeSessionOnRefreshReuse:
		if claims != nil && strings.TrimSpace(claims.SessionID) != "" {
			if err := m.revokeSession(ctx, claims.SessionID, m.sessionRevocationExpiry(claims)); err != nil {
				return fmt.Errorf("revoke replayed refresh token session: %w", err)
			}
		}
	}
	return nil
}

// RefreshToken 刷新访问令牌并返回新的访问令牌与新的刷新令牌（兼容入口，推荐使用 RefreshTokenPair）。
func (m *TokenManager) RefreshToken(refreshTokenStr string) (accessToken string, refreshToken string, err error) {
	return m.RefreshTokenContext(context.Background(), refreshTokenStr)
}

// RefreshTokenContext 使用请求上下文刷新访问令牌并轮换刷新令牌（兼容入口，推荐使用 RefreshTokenPair）。
func (m *TokenManager) RefreshTokenContext(ctx context.Context, refreshTokenStr string) (accessToken string, refreshToken string, err error) {
	pair, err := m.RefreshTokenPair(ctx, refreshTokenStr)
	if err != nil {
		return "", "", err
	}
	return pair.AccessToken, pair.RefreshToken, nil
}

func (m *TokenManager) revokeSession(ctx context.Context, sessionID string, expiresAt time.Time) error {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ErrSessionIDMissing
	}
	return m.revocations.Revoke(nonNilContext(ctx), sessionRevocationKey(sessionID), expiresAt)
}

func (m *TokenManager) isSessionRevoked(ctx context.Context, sessionID string) (bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false, nil
	}
	return m.revocations.IsRevoked(nonNilContext(ctx), sessionRevocationKey(sessionID))
}

func (m *TokenManager) sessionRevocationExpiry(claims *StandardClaims) time.Time {
	snap := m.snapshotConfig()
	maxConfigured := snap.refreshTokenExpiry
	if snap.accessTokenExpiry > maxConfigured {
		maxConfigured = snap.accessTokenExpiry
	}
	expiresAt := m.now().Add(maxConfigured)
	if claims != nil && claims.ExpiresAt != nil && claims.ExpiresAt.Time.After(expiresAt) {
		expiresAt = claims.ExpiresAt.Time
	}
	return expiresAt
}

func validateRawTokenFormat(tokenStr string) error {
	if tokenStr == "" {
		return ErrEmptyToken
	}
	if len(tokenStr) > MaxTokenLength {
		return fmt.Errorf("%w: %w", ErrInvalidTokenFormat, ErrTokenTooLarge)
	}
	if !isTokenFormatValid(tokenStr) {
		return ErrInvalidTokenFormat
	}
	return nil
}

func (m *TokenManager) isTokenFormatValid(tokenStr string) bool {
	return isTokenFormatValid(tokenStr)
}

func isTokenFormatValid(tokenStr string) bool {
	if len(tokenStr) < 10 || len(tokenStr) > MaxTokenLength {
		return false
	}
	firstDot := -1
	secondDot := -1
	for i := 0; i < len(tokenStr); i++ {
		c := tokenStr[i]
		if c == '.' {
			if firstDot == -1 {
				firstDot = i
			} else if secondDot == -1 {
				secondDot = i
			} else {
				return false
			}
		}
	}
	return firstDot > 0 && secondDot > firstDot+1 && secondDot < len(tokenStr)-1
}

// RevokeToken 幂等吊销单个具体令牌。
func (m *TokenManager) RevokeToken(tokenStr string) error {
	return m.RevokeTokenContext(context.Background(), tokenStr)
}

// RevokeTokenContext 使用请求上下文幂等吊销单个具体令牌。
func (m *TokenManager) RevokeTokenContext(ctx context.Context, tokenStr string) error {
	ctx = nonNilContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateRawTokenFormat(tokenStr); err != nil {
		return err
	}

	revoked, err := m.IsTokenRevokedContext(ctx, tokenStr)
	if err != nil {
		return fmt.Errorf("check token revocation: %w", err)
	}
	if revoked {
		m.cache.delete(tokenStr)
		return nil
	}

	claims, err := m.parseTokenClaims(tokenStr, false)
	if err != nil {
		return fmt.Errorf("无法撤销无效的令牌: %w", err)
	}

	now := m.now()
	expireTime := now.Add(24 * time.Hour)
	if claims.ExpiresAt != nil {
		expireTime = claims.ExpiresAt.Time
	}
	if !expireTime.After(now) {
		m.cache.delete(tokenStr)
		return nil
	}

	m.logf("撤销令牌: token_hash=%s, 过期时间: %v", tokenLogID(tokenStr), expireTime)
	if err := m.revocations.Revoke(ctx, tokenRevocationKey(tokenStr), expireTime); err != nil {
		return fmt.Errorf("store token revocation: %w", err)
	}
	m.cache.delete(tokenStr)
	return nil
}

// IsBlacklisted 检查令牌是否在吊销列表中。
//
// Deprecated: 请使用 IsTokenRevokedContext。存储错误会被传入 ErrorReporter 并按失败关闭返回 true。
func (m *TokenManager) IsBlacklisted(tokenStr string) bool {
	revoked, err := m.IsTokenRevokedContext(context.Background(), tokenStr)
	if err != nil {
		m.reportError(fmt.Errorf("check token revocation: %w", err))
		return true
	}
	return revoked
}

// IsTokenRevokedContext 查询具体令牌是否已被提前吊销。
func (m *TokenManager) IsTokenRevokedContext(ctx context.Context, tokenStr string) (bool, error) {
	if tokenStr == "" {
		return false, nil
	}
	if len(tokenStr) > MaxTokenLength {
		return false, fmt.Errorf("%w: %w", ErrInvalidTokenFormat, ErrTokenTooLarge)
	}
	return m.revocations.IsRevoked(nonNilContext(ctx), tokenRevocationKey(tokenStr))
}

// IsSessionRevokedContext 查询指定会话是否已被提前吊销。
func (m *TokenManager) IsSessionRevokedContext(ctx context.Context, sessionID string) (bool, error) {
	return m.isSessionRevoked(ctx, sessionID)
}

// RevokeSession 使用任一带 sid 的已签名令牌注销整个会话。
func (m *TokenManager) RevokeSession(tokenStr string) error {
	return m.RevokeSessionContext(context.Background(), tokenStr)
}

// RevokeSessionContext 使用请求上下文根据已签名令牌注销整个会话。
func (m *TokenManager) RevokeSessionContext(ctx context.Context, tokenStr string) error {
	ctx = nonNilContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateRawTokenFormat(tokenStr); err != nil {
		return err
	}

	claims, err := m.parseTokenClaims(tokenStr, false)
	if err != nil {
		return fmt.Errorf("无法撤销无效令牌的会话: %w", err)
	}
	if strings.TrimSpace(claims.SessionID) == "" {
		return ErrSessionIDMissing
	}

	revoked, err := m.isSessionRevoked(ctx, claims.SessionID)
	if err != nil {
		return fmt.Errorf("check session revocation: %w", err)
	}
	if revoked {
		m.cache.delete(tokenStr)
		return nil
	}
	if err := m.revokeSession(ctx, claims.SessionID, m.sessionRevocationExpiry(claims)); err != nil {
		return fmt.Errorf("store session revocation: %w", err)
	}
	m.cache.delete(tokenStr)
	return nil
}

// RevokeSessionByID 直接按会话 ID 吊销整个会话（适用于后台强制下线、修改密码踢出全部设备等无原始 JWT 场景）。
// ttl 为可选吊销保留时长；不传时默认取 max(AccessTokenExpiry, RefreshTokenExpiry)。
func (m *TokenManager) RevokeSessionByID(ctx context.Context, sessionID string, ttl ...time.Duration) error {
	ctx = nonNilContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return ErrSessionIDMissing
	}

	duration := m.sessionRevocationDefaultTTL()
	if len(ttl) > 0 {
		if ttl[0] <= 0 {
			return ErrInvalidTokenExpiry
		}
		duration = ttl[0]
	}

	if err := m.revokeSession(ctx, sessionID, m.now().Add(duration)); err != nil {
		return fmt.Errorf("store session revocation: %w", err)
	}
	return nil
}

func (m *TokenManager) sessionRevocationDefaultTTL() time.Duration {
	snap := m.snapshotConfig()
	if snap.accessTokenExpiry > snap.refreshTokenExpiry {
		return snap.accessTokenExpiry
	}
	return snap.refreshTokenExpiry
}

// CleanBlacklistContext 清理支持主动清理能力的吊销存储。
func (m *TokenManager) CleanBlacklistContext(ctx context.Context) (int, error) {
	cleaner, ok := m.revocations.(RevocationStoreCleaner)
	if !ok {
		return 0, nil
	}
	return cleaner.CleanExpired(nonNilContext(ctx), m.now())
}

// CleanBlacklist 清理支持主动清理能力的吊销存储。
//
// Deprecated: 请使用 CleanBlacklistContext。
func (m *TokenManager) CleanBlacklist() {
	cleaned, err := m.CleanBlacklistContext(context.Background())
	if err != nil {
		m.reportError(fmt.Errorf("clean revocation store: %w", err))
		return
	}
	if m.snapshotConfig().enableLog && cleaned > 0 {
		m.logf("已清理 %d 条过期的黑名单记录", cleaned)
	}
}

// GetBlacklistSizeContext 返回支持统计能力的吊销存储记录数。
func (m *TokenManager) GetBlacklistSizeContext(ctx context.Context) (int, error) {
	sizer, ok := m.revocations.(RevocationStoreSizer)
	if !ok {
		return 0, ErrBlacklistSizeUnsupported
	}
	return sizer.Size(nonNilContext(ctx))
}

// GetBlacklistSize 返回支持统计能力的吊销存储记录数。
//
// Deprecated: 请使用 GetBlacklistSizeContext。
func (m *TokenManager) GetBlacklistSize() int {
	size, err := m.GetBlacklistSizeContext(context.Background())
	if err != nil {
		m.reportError(fmt.Errorf("stat revocation store: %w", err))
		return 0
	}
	return size
}

// GetCacheSize 返回当前有效解析缓存条目数。
func (m *TokenManager) GetCacheSize() int {
	return m.cache.size()
}

// GetTokenExpiryConfig 返回当前生效配置的快照副本。
func (m *TokenManager) GetTokenExpiryConfig() *JWTOptions {
	snap := m.snapshotConfig()
	return &JWTOptions{
		EnableLog:               snap.enableLog,
		BlacklistCleanInterval:  m.blacklistCleanInterval,
		EnableCache:             snap.enableCache,
		CacheSize:               snap.cacheSize,
		CacheTTL:                snap.cacheTTL,
		AccessTokenExpiry:       snap.accessTokenExpiry,
		RefreshTokenExpiry:      snap.refreshTokenExpiry,
		Issuer:                  snap.issuer,
		Audience:                cloneStringSlice(snap.audience),
		ExpectedIssuer:          snap.expectedIssuer,
		ExpectedAudience:        cloneStringSlice(snap.expectedAudience),
		TokenProfile:            m.tokenProfile.clone(),
		RefreshTokenReuseAction: snap.refreshTokenReuseAction,
		RevocationStore:         m.revocations,
		RefreshTokenUseStore:    m.refreshStore,
		ErrorReporter:           snap.errorReporter,
		Clock:                   m.clock,
	}
}

// GetAccessTokenExpiry 返回访问令牌默认过期时间。
func (m *TokenManager) GetAccessTokenExpiry() time.Duration {
	return m.snapshotConfig().accessTokenExpiry
}

// GetRefreshTokenExpiry 返回刷新令牌默认过期时间。
func (m *TokenManager) GetRefreshTokenExpiry() time.Duration {
	return m.snapshotConfig().refreshTokenExpiry
}
