package jwt

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMustNewTokenManager(t *testing.T) {
	secretKey := "this-is-a-very-secure-jwt-secret-key-32bytes!"

	manager := MustNewTokenManager(secretKey)
	defer manager.Shutdown()

	if string(manager.secretKey) != secretKey {
		t.Errorf("Expected secret key %s, got %s", secretKey, string(manager.secretKey))
	}
	if manager.accessTokenExpiry != DefaultJWTOptions().AccessTokenExpiry {
		t.Errorf("Expected default access token expiry %v, got %v", DefaultJWTOptions().AccessTokenExpiry, manager.accessTokenExpiry)
	}
	if manager.refreshTokenExpiry != DefaultJWTOptions().RefreshTokenExpiry {
		t.Errorf("Expected default refresh token expiry %v, got %v", DefaultJWTOptions().RefreshTokenExpiry, manager.refreshTokenExpiry)
	}
	size, err := manager.GetBlacklistSize(context.Background())
	if err != nil || size != 0 {
		t.Errorf("Expected empty blacklist, got size=%d err=%v", size, err)
	}
}

func TestTokenManager_GenerateToken(t *testing.T) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	ctx := context.Background()
	subject := "123"

	// 测试默认访问令牌
	token1, err := manager.GenerateToken(subject)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	if token1 == "" {
		t.Error("Generated token is empty")
	}

	// 测试自定义过期时间
	customExpires := 2 * time.Hour
	options := &TokenOptions{
		ExpiresIn: customExpires,
	}
	token2, err := manager.GenerateToken(subject, options)
	if err != nil {
		t.Fatalf("Failed to generate token with custom expires: %v", err)
	}
	if token2 == "" {
		t.Error("Generated token with custom expires is empty")
	}
	if token1 == token2 {
		t.Error("Tokens should be different")
	}

	// 测试刷新令牌
	refreshOptions := &TokenOptions{
		TokenType: RefreshToken,
		SessionID: "test-session",
	}
	refreshToken, err := manager.GenerateToken(subject, refreshOptions)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	// 验证刷新令牌不能被通用接口当作访问令牌通过，必须通过 ValidateRefreshToken 验证
	if _, err := manager.ValidateAccessToken(ctx, refreshToken); !errors.Is(err, ErrTokenTypeMismatch) {
		t.Fatalf("Expected ErrTokenTypeMismatch when passing refresh token to ValidateAccessToken, got %v", err)
	}
	claims, err := manager.ValidateRefreshToken(ctx, refreshToken)
	if err != nil {
		t.Fatalf("Failed to validate refresh token: %v", err)
	}
	if claims.TokenType != RefreshToken {
		t.Errorf("Expected token type %s, got %s", RefreshToken, claims.TokenType)
	}
	if claims.SessionID != "test-session" {
		t.Errorf("Expected session ID %s, got %s", "test-session", claims.SessionID)
	}
}

func TestTokenManager_ValidateAccessToken(t *testing.T) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	ctx := context.Background()
	subject := "123"

	options := &TokenOptions{
		TokenType: AccessToken,
		SessionID: "test-session",
		CustomClaims: map[string]interface{}{
			"department": "engineering",
		},
	}

	token, err := manager.GenerateToken(subject, options)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	claims, err := manager.ValidateAccessToken(ctx, token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.Subject != subject {
		t.Errorf("Expected subject %s, got %s", subject, claims.Subject)
	}
	if claims.TokenType != AccessToken {
		t.Errorf("Expected token type %s, got %s", AccessToken, claims.TokenType)
	}
	if claims.SessionID != "test-session" {
		t.Errorf("Expected session ID %s, got %s", "test-session", claims.SessionID)
	}

	if v, ok := claims.CustomClaims["department"]; !ok || v != "engineering" {
		t.Errorf("Expected custom claim department=engineering, got %v (exists=%v)", v, ok)
	}

	_, err = manager.ValidateAccessToken(ctx, "invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestStandardClaims_UnmarshalJSON_JTINumber(t *testing.T) {
	var c StandardClaims
	err := c.UnmarshalJSON([]byte(`{"sub":"u-1","jti":123,"type":"access","sid":"s-1","foo":"bar"}`))
	if !errors.Is(err, ErrInvalidClaimType) {
		t.Fatalf("Expected ErrInvalidClaimType for numeric jti, got %v", err)
	}
}

func TestTokenManager_RefreshTokenPair(t *testing.T) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	ctx := context.Background()
	subject := "123"
	sessionID := "test-session"

	refreshToken, err := manager.GenerateToken(subject, &TokenOptions{
		TokenType: RefreshToken,
		SessionID: sessionID,
	})
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	pair, err := manager.RefreshTokenPair(ctx, refreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	if pair.RefreshToken == refreshToken {
		t.Errorf("Expected rotated refresh token to be different from original")
	}

	claims, err := manager.ValidateAccessToken(ctx, pair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate new access token: %v", err)
	}

	if claims.Subject != subject {
		t.Errorf("Expected subject %s, got %s", subject, claims.Subject)
	}
	if claims.TokenType != AccessToken {
		t.Errorf("Expected token type %s, got %s", AccessToken, claims.TokenType)
	}
	if claims.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, claims.SessionID)
	}

	// rotation：旧 refresh token 应被作废（再次使用应失败）
	if _, err := manager.RefreshTokenPair(ctx, refreshToken); !errors.Is(err, ErrRefreshReuseDetected) {
		t.Errorf("Reusing old refresh token should fail with ErrRefreshReuseDetected, got %v", err)
	}
	if _, err := manager.ValidateAccessToken(ctx, pair.AccessToken); !errors.Is(err, ErrSessionRevoked) {
		t.Fatalf("replayed refresh token should revoke session, ValidateAccessToken() error = %v", err)
	}

	// 尝试使用访问令牌作为刷新令牌 (应该失败)
	if _, err := manager.RefreshTokenPair(ctx, pair.AccessToken); !errors.Is(err, ErrTokenTypeMismatch) {
		t.Errorf("Using access token as refresh token should fail with ErrTokenTypeMismatch, got %v", err)
	}
}

func TestTokenManager_RevokeToken(t *testing.T) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	ctx := context.Background()
	subject := "123"
	token, _ := manager.GenerateToken(subject)

	if err := manager.RevokeToken(ctx, token); err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	revoked, err := manager.IsTokenRevoked(ctx, token)
	if err != nil || !revoked {
		t.Errorf("Token should be revoked, revoked=%v err=%v", revoked, err)
	}

	if _, err := manager.ValidateAccessToken(ctx, token); !errors.Is(err, ErrTokenRevoked) {
		t.Errorf("Validation should fail for revoked token, got: %v", err)
	}

	store := manager.revocations.(*MemoryRevocationStore)
	store.mu.Lock()
	store.entries[tokenRevocationKey(token)] = time.Now().Add(-time.Hour)
	store.mu.Unlock()

	if _, err := manager.CleanBlacklist(ctx); err != nil {
		t.Fatalf("CleanBlacklist: %v", err)
	}
	revoked, err = manager.IsTokenRevoked(ctx, token)
	if err != nil || revoked {
		t.Errorf("Expired token should be removed from blacklist, revoked=%v err=%v", revoked, err)
	}
}

func BenchmarkTokenManager_GenerateToken(b *testing.B) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	subject := "123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateToken(subject)
	}
}

func BenchmarkTokenManager_ValidateAccessToken(b *testing.B) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	ctx := context.Background()
	token, _ := manager.GenerateToken("123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ValidateAccessToken(ctx, token)
	}
}

func BenchmarkTokenManager_IsTokenRevoked(b *testing.B) {
	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	ctx := context.Background()
	token, _ := manager.GenerateToken("123")
	_ = manager.RevokeToken(ctx, token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.IsTokenRevoked(ctx, token)
	}
}

func TestTokenManager_ValidateAccessToken_Parallel(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		options *TokenOptions
		wantErr bool
	}{
		{
			name:    "正常令牌",
			subject: "123",
			options: &TokenOptions{
				TokenType:    AccessToken,
				CustomClaims: map[string]interface{}{"department": "sales"},
			},
			wantErr: false,
		},
		{
			name:    "空用户ID",
			subject: "",
			options: nil,
			wantErr: true,
		},
		{
			name:    "黑名单令牌",
			subject: "456",
			options: nil,
			wantErr: true,
		},
	}

	manager := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager.Shutdown()
	ctx := context.Background()

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.subject == "" {
				_, err := manager.GenerateToken(tt.subject, tt.options)
				if (err != nil) != tt.wantErr {
					t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			token, _ := manager.GenerateToken(tt.subject, tt.options)
			if tt.name == "黑名单令牌" {
				_ = manager.RevokeToken(ctx, token)
			}
			_, err := manager.ValidateAccessToken(ctx, token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAccessToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewTokenManager_InvalidKey(t *testing.T) {
	if _, err := NewTokenManager("short"); err == nil {
		t.Error("Expected error for short secret key")
	}
	if _, err := NewTokenManager("onlylowercaseletters"); err == nil {
		t.Error("Expected error for key shorter than 32 bytes")
	}
	tm, err := NewTokenManager("this-is-a-very-secure-secret-key-with-mix3d-chars!")
	if err != nil {
		t.Errorf("Unexpected error for valid key: %v", err)
	}
	tm.Shutdown()
}

func TestTokenManager_GetTokenExpiryConfig(t *testing.T) {
	manager1 := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager1.Shutdown()
	config1 := manager1.GetTokenExpiryConfig()

	defaultOptions := DefaultJWTOptions()

	if config1.AccessTokenExpiry != defaultOptions.AccessTokenExpiry {
		t.Errorf("期望访问令牌过期时间为 %v，实际得到 %v", defaultOptions.AccessTokenExpiry, config1.AccessTokenExpiry)
	}
	if config1.RefreshTokenExpiry != defaultOptions.RefreshTokenExpiry {
		t.Errorf("期望刷新令牌过期时间为 %v，实际得到 %v", defaultOptions.RefreshTokenExpiry, config1.RefreshTokenExpiry)
	}
	if config1.EnableCache != defaultOptions.EnableCache {
		t.Errorf("期望EnableCache为 %v，实际得到 %v", defaultOptions.EnableCache, config1.EnableCache)
	}
	if config1.CacheSize != defaultOptions.CacheSize {
		t.Errorf("期望CacheSize为 %v，实际得到 %v", defaultOptions.CacheSize, config1.CacheSize)
	}
	if config1.CacheTTL != defaultOptions.CacheTTL {
		t.Errorf("期望CacheTTL为 %v，实际得到 %v", defaultOptions.CacheTTL, config1.CacheTTL)
	}

	customOptions := &JWTOptions{
		EnableCache:            false,
		CacheSize:              500,
		CacheTTL:               3 * time.Minute,
		AccessTokenExpiry:      30 * time.Minute,
		RefreshTokenExpiry:     7 * 24 * time.Hour,
		BlacklistCleanInterval: 5 * time.Minute,
		SigningKeyID:           "k-2026",
		VerificationKeys:       map[string][]byte{"k-old": []byte("0123456789abcdef0123456789abcdef")},
		LegacyVerificationKey:  []byte("fedcba9876543210fedcba9876543210"),
	}
	manager2 := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!", customOptions)
	defer manager2.Shutdown()
	config2 := manager2.GetTokenExpiryConfig()

	if config2.AccessTokenExpiry != customOptions.AccessTokenExpiry {
		t.Errorf("期望访问令牌过期时间为 %v，实际得到 %v", customOptions.AccessTokenExpiry, config2.AccessTokenExpiry)
	}
	if config2.RefreshTokenExpiry != customOptions.RefreshTokenExpiry {
		t.Errorf("期望刷新令牌过期时间为 %v，实际得到 %v", customOptions.RefreshTokenExpiry, config2.RefreshTokenExpiry)
	}
	if config2.EnableCache != customOptions.EnableCache {
		t.Errorf("期望EnableCache为 %v，实际得到 %v", customOptions.EnableCache, config2.EnableCache)
	}
	if config2.CacheSize != customOptions.CacheSize {
		t.Errorf("期望CacheSize为 %v，实际得到 %v", customOptions.CacheSize, config2.CacheSize)
	}
	if config2.CacheTTL != customOptions.CacheTTL {
		t.Errorf("期望CacheTTL为 %v，实际得到 %v", customOptions.CacheTTL, config2.CacheTTL)
	}
	if config2.SigningKeyID != "k-2026" {
		t.Errorf("期望SigningKeyID为 k-2026，实际得到 %q", config2.SigningKeyID)
	}
	if config2.VerificationKeys != nil || config2.LegacyVerificationKey != nil {
		t.Errorf("GetTokenExpiryConfig must never expose verification secret bytes")
	}
}

func TestTokenManager_GetAccessAndRefreshTokenExpiry(t *testing.T) {
	manager1 := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!")
	defer manager1.Shutdown()

	defaultOptions := DefaultJWTOptions()
	if manager1.GetAccessTokenExpiry() != defaultOptions.AccessTokenExpiry {
		t.Errorf("unexpected default access expiry: %v", manager1.GetAccessTokenExpiry())
	}
	if manager1.GetRefreshTokenExpiry() != defaultOptions.RefreshTokenExpiry {
		t.Errorf("unexpected default refresh expiry: %v", manager1.GetRefreshTokenExpiry())
	}

	customOptions := DefaultJWTOptions()
	customOptions.AccessTokenExpiry = 45 * time.Minute
	customOptions.RefreshTokenExpiry = 3 * 24 * time.Hour
	manager2 := MustNewTokenManager("this-is-a-very-secure-jwt-secret-key-32bytes!", customOptions)
	defer manager2.Shutdown()

	if manager2.GetAccessTokenExpiry() != customOptions.AccessTokenExpiry {
		t.Errorf("unexpected custom access expiry: %v", manager2.GetAccessTokenExpiry())
	}
	if manager2.GetRefreshTokenExpiry() != customOptions.RefreshTokenExpiry {
		t.Errorf("unexpected custom refresh expiry: %v", manager2.GetRefreshTokenExpiry())
	}
}
