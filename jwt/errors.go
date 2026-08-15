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
	// ErrUnknownTokenType 表示未知令牌类型。
	ErrUnknownTokenType = errors.New("unknown token type")
	// ErrInvalidTokenExpiry 表示动态设置的过期时间非法。
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
)
