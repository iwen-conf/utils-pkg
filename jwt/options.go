package jwt

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

const (
	// MaxTokenLength 限制单个 JWT 字符串最大字节长度，防止超大输入导致哈希/解析/内存放大。
	MaxTokenLength = 8192
	minKeyBytes    = 32
)

// TokenType 定义令牌用途类型。
type TokenType string

const (
	// AccessToken 访问令牌，用于业务 API 鉴权。
	AccessToken TokenType = "access"
	// RefreshToken 刷新令牌，仅用于轮换获取新令牌对，不可用于访问受保护业务接口。
	RefreshToken TokenType = "refresh"
)

func isValidTokenType(t TokenType) bool {
	return t == AccessToken || t == RefreshToken
}

// RefreshTokenReuseAction 定义 refresh token 被检测到复用(重放)时执行的动作。
type RefreshTokenReuseAction int

const (
	// RevokeSessionOnRefreshReuse 表示发现 refresh token 复用时撤销整个 session(sid)。
	RevokeSessionOnRefreshReuse RefreshTokenReuseAction = iota
)

// TokenOptions 控制单次令牌签发选项。
type TokenOptions struct {
	// ExpiresIn 覆盖管理器默认过期时间；必须 >= 0（0 表示使用对应类型的默认过期时间）。
	ExpiresIn time.Duration
	// Issuer 覆盖管理器默认签发者。
	Issuer string
	// Audience 覆盖管理器默认受众。
	Audience []string
	// TokenType 令牌类型，为空时默认 AccessToken。
	TokenType TokenType
	// SessionID 会话标识，可用于会话级注销与重放封禁。
	SessionID string
	// TokenID 令牌唯一标识(jti)，为空时自动使用 CSPRNG 生成。
	TokenID string
	// CustomClaims 附加业务声明；禁止使用 iss/sub/aud/exp/nbf/iat/jti/type/sid 等保留键。
	CustomClaims map[string]interface{}
}

// DefaultTokenOptions 返回默认单令牌选项。
func DefaultTokenOptions() *TokenOptions {
	return &TokenOptions{
		TokenType:    AccessToken,
		CustomClaims: make(map[string]interface{}),
	}
}

// JWTOptions 配置 TokenManager（构造后不可变）。
type JWTOptions struct {
	// BlacklistCleanInterval 默认内存存储和验证缓存的后台清理周期；0 表示禁用后台协程。
	BlacklistCleanInterval time.Duration
	// EnableCache 是否启用有效令牌解析缓存。
	EnableCache bool
	// CacheSize 验证缓存最大条目数（LRU 严格受限）。
	CacheSize int
	// CacheTTL 验证缓存最大存活时间（实际有效期不超过 min(now+CacheTTL, token.exp)）。
	CacheTTL time.Duration
	// AccessTokenExpiry 访问令牌默认有效期。
	AccessTokenExpiry time.Duration
	// RefreshTokenExpiry 刷新令牌默认有效期。
	RefreshTokenExpiry time.Duration
	// Issuer 默认签发者（仅用于签发默认值）。
	Issuer string
	// Audience 默认受众（仅用于签发默认值）。
	Audience []string
	// ExpectedIssuer 验证时要求的签发者；为空表示不校验 iss。
	ExpectedIssuer string
	// ExpectedAudience 验证时要求的受众；为空表示不校验 aud。
	ExpectedAudience []string
	// SigningKeyID 可选；非空时签发令牌会在 JWT Header 写入 kid，并要求验签严格匹配 kid。
	SigningKeyID string
	// VerificationKeys 可选的多密钥验签映射（kid -> HS256 密钥，每把至少 32 字节），用于密钥轮换平滑过渡。
	VerificationKeys map[string][]byte
	// LegacyVerificationKey 可选的旧版无 kid 令牌兜底验签密钥（非空时至少 32 字节），仅在 SigningKeyID 非空时生效。
	LegacyVerificationKey []byte
	// TokenProfile 定义业务 Identity 映射到 CustomClaims 的字段名。
	TokenProfile *TokenProfile
	// RefreshTokenReuseAction 检测到 refresh token 重放时的处理策略。
	RefreshTokenReuseAction RefreshTokenReuseAction
	// RevocationStore 保存 token/session 吊销状态；为空时使用单进程内存实现。
	RevocationStore RevocationStore
	// RefreshTokenUseStore 原子消费 refresh jti；为空时使用单进程内存实现。
	RefreshTokenUseStore RefreshTokenUseStore
	// ErrorReporter 接收后台清理错误；为空时不输出日志。
	ErrorReporter func(error)
	// Clock 可选时间源，用于测试或自定义时钟；为空时使用 time.Now。
	Clock func() time.Time
}

// DefaultJWTOptions 返回推荐的默认配置。
func DefaultJWTOptions() *JWTOptions {
	return &JWTOptions{
		BlacklistCleanInterval:  10 * time.Minute,
		EnableCache:             true,
		CacheSize:               1000,
		CacheTTL:                5 * time.Minute,
		AccessTokenExpiry:       15 * time.Minute,
		RefreshTokenExpiry:      24 * time.Hour,
		RefreshTokenReuseAction: RevokeSessionOnRefreshReuse,
	}
}

func validateJWTOptions(opts *JWTOptions) error {
	if opts.AccessTokenExpiry <= 0 {
		return ErrInvalidAccessTokenExpiry
	}
	if opts.RefreshTokenExpiry <= 0 {
		return ErrInvalidRefreshTokenExpiry
	}
	if opts.EnableCache {
		if opts.CacheSize <= 0 {
			return ErrInvalidCacheSize
		}
		if opts.CacheTTL <= 0 {
			return ErrInvalidCacheTTL
		}
	}
	if opts.BlacklistCleanInterval < 0 {
		return ErrInvalidCleanupInterval
	}
	switch opts.RefreshTokenReuseAction {
	case RevokeSessionOnRefreshReuse:
	default:
		return ErrUnknownRefreshReuseAction
	}
	profile := normalizeTokenProfile(opts.TokenProfile)
	if err := validateTokenProfile(profile); err != nil {
		return err
	}
	return nil
}

func buildKeyRing(secretKey []byte, opts *JWTOptions) (string, map[string][]byte, []byte, error) {
	signingKeyID := strings.TrimSpace(opts.SigningKeyID)
	if opts.SigningKeyID != "" && signingKeyID == "" {
		return "", nil, nil, ErrInvalidKeyID
	}

	var verificationKeys map[string][]byte
	if len(opts.VerificationKeys) > 0 {
		verificationKeys = make(map[string][]byte, len(opts.VerificationKeys))
		for rawKID, key := range opts.VerificationKeys {
			kid := strings.TrimSpace(rawKID)
			if kid == "" {
				return "", nil, nil, ErrInvalidKeyID
			}
			if len(key) < minKeyBytes {
				return "", nil, nil, fmt.Errorf("%w: verification key %q got %d bytes", ErrSecretKeyTooShort, kid, len(key))
			}
			if _, exists := verificationKeys[kid]; exists {
				return "", nil, nil, fmt.Errorf("%w: %q", ErrDuplicateKeyID, kid)
			}
			if signingKeyID != "" && kid == signingKeyID && !bytes.Equal(key, secretKey) {
				return "", nil, nil, fmt.Errorf("%w: %q conflicts with signing secret", ErrDuplicateKeyID, kid)
			}
			verificationKeys[kid] = bytes.Clone(key)
		}
	}

	var legacyKey []byte
	if len(opts.LegacyVerificationKey) > 0 {
		if len(opts.LegacyVerificationKey) < minKeyBytes {
			return "", nil, nil, fmt.Errorf("%w: legacy verification key got %d bytes", ErrSecretKeyTooShort, len(opts.LegacyVerificationKey))
		}
		legacyKey = bytes.Clone(opts.LegacyVerificationKey)
	}

	return signingKeyID, verificationKeys, legacyKey, nil
}

func validateTokenProfile(p TokenProfile) error {
	keys := []struct {
		field string
		claim string
	}{
		{"ActorTypeClaim", strings.TrimSpace(p.ActorTypeClaim)},
		{"ActorIDClaim", strings.TrimSpace(p.ActorIDClaim)},
		{"TenantIDClaim", strings.TrimSpace(p.TenantIDClaim)},
		{"RolesClaim", strings.TrimSpace(p.RolesClaim)},
		{"ScopesClaim", strings.TrimSpace(p.ScopesClaim)},
	}
	seen := make(map[string]string, len(keys))
	for _, item := range keys {
		if item.claim == "" {
			return fmt.Errorf("%w: %s cannot be empty", ErrDuplicateClaimKey, item.field)
		}
		if isReservedClaimKey(item.claim) {
			return fmt.Errorf("%w: TokenProfile.%s cannot use reserved claim %q", ErrReservedClaimKey, item.field, item.claim)
		}
		if prevField, exists := seen[item.claim]; exists {
			return fmt.Errorf("%w: TokenProfile.%s and %s share claim %q", ErrDuplicateClaimKey, prevField, item.field, item.claim)
		}
		seen[item.claim] = item.field
	}
	return nil
}
