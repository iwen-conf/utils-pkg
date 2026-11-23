package jwt

import "errors"

var (
	// ErrSecretKeyTooShort 表示密钥不足 32 字节。
	ErrSecretKeyTooShort = errors.New("JWT secret key must be at least 32 bytes")
	// ErrInvalidAccessTokenExpiry 表示访问令牌过期时间非法。
	ErrInvalidAccessTokenExpiry = errors.New("access token expiry must be positive")
	// ErrInvalidRefreshTokenExpiry 表示刷新令牌过期时间非法。
	ErrInvalidRefreshTokenExpiry = errors.New("refresh token expiry must be positive")
	// ErrInvalidCacheSize 表示启用缓存时容量非法。
	ErrInvalidCacheSize = errors.New("cache size must be positive when cache is enabled")
	// ErrInvalidCacheTTL 表示启用缓存时 TTL 非法。
	ErrInvalidCacheTTL = errors.New("cache TTL must be positive when cache is enabled")
	// ErrInvalidCleanupInterval 表示清理间隔为负数。
	ErrInvalidCleanupInterval = errors.New("blacklist clean interval must be non-negative")
	// ErrUnknownRefreshReuseAction 表示未知的 refresh 复用动作。
	ErrUnknownRefreshReuseAction = errors.New("unknown refresh token reuse action")
	// ErrUnknownTokenType 表示未知或缺失的令牌类型。
	ErrUnknownTokenType = errors.New("unknown token type")
	// ErrTokenTypeMismatch 表示令牌类型与当前操作要求的用途不匹配（如将 refresh token 当作 access token 验证）。
	ErrTokenTypeMismatch = errors.New("unexpected token type")
	// ErrInvalidTokenExpiry 表示动态或单次设置的过期时间非法。
	ErrInvalidTokenExpiry = errors.New("token expiry must be positive")
	// ErrBlacklistSizeUnsupported 表示吊销存储不支持统计。
	ErrBlacklistSizeUnsupported = errors.New("revocation store does not support size")
	// ErrEmptyRefreshJTI 表示 refresh 消费缺少 jti。
	ErrEmptyRefreshJTI = errors.New("refresh jti cannot be empty")
	// ErrRefreshJTINotFuture 表示 refresh jti 过期时间不在未来。
	ErrRefreshJTINotFuture = errors.New("refresh jti expiry must be in the future")
	// ErrUnsupportedClaimType 表示缓存无法安全克隆的可变 claim 类型。
	ErrUnsupportedClaimType = errors.New("unsupported mutable claim type")
	// ErrRefreshReuseDetected 表示 refresh token 被重复消费。
	ErrRefreshReuseDetected = errors.New("refresh token reuse detected")
	// ErrInvalidClaimType 表示注册 claim 类型非法。
	ErrInvalidClaimType = errors.New("invalid registered claim type")
	// ErrEmptySubject 表示签发令牌时 subject 为空或仅空白字符。
	ErrEmptySubject = errors.New("subject cannot be empty")
	// ErrEmptyToken 表示传入令牌为空字符串。
	ErrEmptyToken = errors.New("token cannot be empty")
	// ErrInvalidTokenFormat 表示令牌不满足三段式 JWT 基本格式或超过最大长度。
	ErrInvalidTokenFormat = errors.New("invalid token format")
	// ErrTokenTooLarge 表示令牌长度超过安全上限。
	ErrTokenTooLarge = errors.New("token exceeds maximum length")
	// ErrInvalidToken 表示底层解析器返回了无效令牌对象。
	ErrInvalidToken = errors.New("invalid token")
	// ErrReservedClaimKey 表示自定义声明或身份配置占用了 JWT 保留字段。
	ErrReservedClaimKey = errors.New("custom claim key is reserved")
	// ErrDuplicateClaimKey 表示自定义声明或身份配置存在重复冲突字段。
	ErrDuplicateClaimKey = errors.New("duplicate claim key")
	// ErrMissingTokenID 表示需要一次性消费的令牌缺少 jti。
	ErrMissingTokenID = errors.New("refresh token missing jti")
	// ErrInvalidKeyID 表示配置中的 kid 为空或仅空白字符。
	ErrInvalidKeyID = errors.New("invalid JWT key id")
	// ErrDuplicateKeyID 表示配置中存在重复或冲突的 kid。
	ErrDuplicateKeyID = errors.New("duplicate JWT key id with conflicting secret")
	// ErrUnknownKeyID 表示令牌缺少必需的 kid 或引用了未知的 kid。
	ErrUnknownKeyID = errors.New("unknown or missing JWT key id")
)
