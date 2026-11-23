package jwt

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var parseWithClaims = jwt.ParseWithClaims

// TokenManager 管理 JWT 令牌签发、按用途隔离验证、刷新轮换及令牌/会话吊销（构造后配置不可变）。
type TokenManager struct {
	secretKey              []byte
	signingKeyID           string
	verificationKeys       map[string][]byte
	legacyVerificationKey  []byte
	revocations            RevocationStore
	refreshStore           RefreshTokenUseStore
	blacklistCleanInterval time.Duration
	clock                  func() time.Time

	cache *validationCache

	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
	shutdownOnce  sync.Once

	tokenProfile TokenProfile

	cacheSize               int
	cacheTTL                time.Duration
	accessTokenExpiry       time.Duration
	refreshTokenExpiry      time.Duration
	issuer                  string
	audience                []string
	expectedIssuer          string
	expectedAudience        []string
	enableCache             bool
	refreshTokenReuseAction RefreshTokenReuseAction
	errorReporter           func(error)
}

// NewTokenManager 创建新的 JWT 令牌管理器。密钥按 UTF-8 字节计至少 32 字节。
func NewTokenManager(secretKey string, options ...*JWTOptions) (*TokenManager, error) {
	secretBytes := []byte(secretKey)
	if len(secretBytes) < minKeyBytes {
		return nil, fmt.Errorf("%w: got %d bytes", ErrSecretKeyTooShort, len(secretBytes))
	}

	opts := DefaultJWTOptions()
	if len(options) > 0 && options[0] != nil {
		opts = options[0]
	}
	if err := validateJWTOptions(opts); err != nil {
		return nil, err
	}

	signingKeyID, verificationKeys, legacyKey, err := buildKeyRing(secretBytes, opts)
	if err != nil {
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

	manager := &TokenManager{
		secretKey:               secretBytes,
		signingKeyID:            signingKeyID,
		verificationKeys:        verificationKeys,
		legacyVerificationKey:   legacyKey,
		revocations:             revocations,
		refreshStore:            refreshStore,
		blacklistCleanInterval:  opts.BlacklistCleanInterval,
		clock:                   clock,
		cache:                   newValidationCache(),
		cacheSize:               opts.CacheSize,
		cacheTTL:                opts.CacheTTL,
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
		errorReporter:           opts.ErrorReporter,
	}

	_, revCleaner := revocations.(RevocationStoreCleaner)
	_, refreshCleaner := refreshStore.(RefreshTokenUseStoreCleaner)
	if opts.BlacklistCleanInterval > 0 && (revCleaner || refreshCleaner || opts.EnableCache) {
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
			_ = m.cache.cleanExpired(now)
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

func (m *TokenManager) reportError(err error) {
	if err == nil || m.errorReporter == nil {
		return
	}
	m.errorReporter(err)
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

	var expiresIn time.Duration
	switch {
	case opts.ExpiresIn > 0:
		expiresIn = opts.ExpiresIn
	case tokenType == AccessToken:
		expiresIn = m.accessTokenExpiry
	default:
		expiresIn = m.refreshTokenExpiry
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
		issuer = m.issuer
	}
	audience := cloneStringSlice(opts.Audience)
	if len(audience) == 0 {
		audience = cloneStringSlice(m.audience)
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
	if m.signingKeyID != "" {
		token.Header["kid"] = m.signingKeyID
	}
	tokenStr, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", err
	}
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

	revoked, err := m.IsTokenRevoked(ctx, tokenStr)
	if err != nil {
		return nil, fmt.Errorf("check token revocation: %w", err)
	}
	if revoked {
		m.cache.delete(tokenStr)
		return nil, ErrTokenRevoked
	}

	now := m.now()

	if m.enableCache {
		if claims, cloneErr, found := m.cache.get(tokenStr, now); found {
			if cloneErr != nil {
				return nil, cloneErr
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

	if m.enableCache {
		if cacheErr := m.cache.put(tokenStr, claims, now, m.cacheTTL, m.cacheSize); cacheErr != nil {
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

	revoked, err := m.IsTokenRevoked(ctx, tokenStr)
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

func ensureExpectedTokenType(got TokenType, want TokenType) error {
	if got == want {
		return nil
	}
	if !isValidTokenType(got) {
		return fmt.Errorf("%w: %q", ErrUnknownTokenType, got)
	}
	return fmt.Errorf("%w: got %q, want %q", ErrTokenTypeMismatch, got, want)
}

func (m *TokenManager) signingKeyForToken(token *jwt.Token) ([]byte, error) {
	if token.Method == nil || token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
		return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
	}

	rawKID, hasKID := token.Header["kid"]
	if hasKID {
		kid, ok := rawKID.(string)
		if !ok || strings.TrimSpace(kid) == "" {
			return nil, ErrUnknownKeyID
		}
		if m.signingKeyID != "" && kid == m.signingKeyID {
			return m.secretKey, nil
		}
		if key, exists := m.verificationKeys[kid]; exists {
			return key, nil
		}
		return nil, fmt.Errorf("%w: %q", ErrUnknownKeyID, kid)
	}

	if m.signingKeyID == "" {
		return m.secretKey, nil
	}
	if len(m.legacyVerificationKey) > 0 {
		return m.legacyVerificationKey, nil
	}
	return nil, ErrUnknownKeyID
}

func (m *TokenManager) parseTokenClaims(tokenStr string, validateClaims bool) (*StandardClaims, error) {
	parserOptions := []jwt.ParserOption{
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(m.now),
	}
	if validateClaims {
		parserOptions = append(parserOptions, jwt.WithExpirationRequired())
		if m.expectedIssuer != "" {
			parserOptions = append(parserOptions, jwt.WithIssuer(m.expectedIssuer))
		}
		if len(m.expectedAudience) > 0 {
			parserOptions = append(parserOptions, jwt.WithAudience(m.expectedAudience...))
		}
	} else {
		parserOptions = append(parserOptions, jwt.WithoutClaimsValidation())
	}

	token, err := parseWithClaims(tokenStr, &StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		return m.signingKeyForToken(token)
	}, parserOptions...)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*StandardClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}
	if !validateClaims {
		if m.expectedIssuer != "" && claims.Issuer != m.expectedIssuer {
			return nil, fmt.Errorf("%w: issuer mismatch", jwt.ErrTokenInvalidIssuer)
		}
		if len(m.expectedAudience) > 0 && !audienceIntersects(claims.Audience, m.expectedAudience) {
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
	tokenRevoked, err := m.IsTokenRevoked(ctx, refreshTokenStr)
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

	// 在触碰外部消费状态前，先在内存中生成新令牌对，确保参数异常或上下文取消不会白白烧毁旧 refresh token。
	newRefresh, err := m.GenerateToken(claims.Subject, &TokenOptions{
		TokenType:    RefreshToken,
		ExpiresIn:    m.refreshTokenExpiry,
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
		ExpiresIn:    m.accessTokenExpiry,
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

	expireAt := m.now().Add(m.refreshTokenExpiry)
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
		AccessExpiresIn:  int64(m.accessTokenExpiry.Seconds()),
		RefreshExpiresIn: int64(m.refreshTokenExpiry.Seconds()),
	}, nil
}

func (m *TokenManager) handleRefreshReuse(ctx context.Context, claims *StandardClaims) error {
	switch m.refreshTokenReuseAction {
	case RevokeSessionOnRefreshReuse:
		if claims != nil && strings.TrimSpace(claims.SessionID) != "" {
			if err := m.revokeSession(ctx, claims.SessionID, m.sessionRevocationExpiry(claims)); err != nil {
				return fmt.Errorf("revoke replayed refresh token session: %w", err)
			}
		}
	}
	return nil
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
	maxConfigured := m.refreshTokenExpiry
	if m.accessTokenExpiry > maxConfigured {
		maxConfigured = m.accessTokenExpiry
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

// RevokeToken 使用请求上下文幂等吊销单个具体令牌。
func (m *TokenManager) RevokeToken(ctx context.Context, tokenStr string) error {
	ctx = nonNilContext(ctx)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateRawTokenFormat(tokenStr); err != nil {
		return err
	}

	revoked, err := m.IsTokenRevoked(ctx, tokenStr)
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

	if err := m.revocations.Revoke(ctx, tokenRevocationKey(tokenStr), expireTime); err != nil {
		return fmt.Errorf("store token revocation: %w", err)
	}
	m.cache.delete(tokenStr)
	return nil
}

// IsTokenRevoked 查询具体令牌是否已被提前吊销。
func (m *TokenManager) IsTokenRevoked(ctx context.Context, tokenStr string) (bool, error) {
	if tokenStr == "" {
		return false, nil
	}
	if len(tokenStr) > MaxTokenLength {
		return false, fmt.Errorf("%w: %w", ErrInvalidTokenFormat, ErrTokenTooLarge)
	}
	return m.revocations.IsRevoked(nonNilContext(ctx), tokenRevocationKey(tokenStr))
}

// IsSessionRevoked 查询指定会话是否已被提前吊销。
func (m *TokenManager) IsSessionRevoked(ctx context.Context, sessionID string) (bool, error) {
	return m.isSessionRevoked(ctx, sessionID)
}

// RevokeSession 使用请求上下文根据已签名令牌注销整个会话。
func (m *TokenManager) RevokeSession(ctx context.Context, tokenStr string) error {
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
	if m.accessTokenExpiry > m.refreshTokenExpiry {
		return m.accessTokenExpiry
	}
	return m.refreshTokenExpiry
}

// CleanBlacklist 清理支持主动清理能力的吊销存储。
func (m *TokenManager) CleanBlacklist(ctx context.Context) (int, error) {
	cleaner, ok := m.revocations.(RevocationStoreCleaner)
	if !ok {
		return 0, nil
	}
	return cleaner.CleanExpired(nonNilContext(ctx), m.now())
}

// GetBlacklistSize 返回支持统计能力的吊销存储记录数。
func (m *TokenManager) GetBlacklistSize(ctx context.Context) (int, error) {
	sizer, ok := m.revocations.(RevocationStoreSizer)
	if !ok {
		return 0, ErrBlacklistSizeUnsupported
	}
	return sizer.Size(nonNilContext(ctx))
}

// GetCacheSize 返回当前有效解析缓存条目数。
func (m *TokenManager) GetCacheSize() int {
	return m.cache.size()
}

// GetTokenExpiryConfig 返回当前生效配置的非密钥快照副本（不暴露 VerificationKeys 或 LegacyVerificationKey 密钥字节）。
func (m *TokenManager) GetTokenExpiryConfig() *JWTOptions {
	return &JWTOptions{
		BlacklistCleanInterval:  m.blacklistCleanInterval,
		EnableCache:             m.enableCache,
		CacheSize:               m.cacheSize,
		CacheTTL:                m.cacheTTL,
		AccessTokenExpiry:       m.accessTokenExpiry,
		RefreshTokenExpiry:      m.refreshTokenExpiry,
		Issuer:                  m.issuer,
		Audience:                cloneStringSlice(m.audience),
		ExpectedIssuer:          m.expectedIssuer,
		ExpectedAudience:        cloneStringSlice(m.expectedAudience),
		SigningKeyID:            m.signingKeyID,
		TokenProfile:            m.tokenProfile.clone(),
		RefreshTokenReuseAction: m.refreshTokenReuseAction,
		RevocationStore:         m.revocations,
		RefreshTokenUseStore:    m.refreshStore,
		ErrorReporter:           m.errorReporter,
		Clock:                   m.clock,
	}
}

// GetAccessTokenExpiry 返回访问令牌默认过期时间。
func (m *TokenManager) GetAccessTokenExpiry() time.Duration {
	return m.accessTokenExpiry
}

// GetRefreshTokenExpiry 返回刷新令牌默认过期时间。
func (m *TokenManager) GetRefreshTokenExpiry() time.Duration {
	return m.refreshTokenExpiry
}
