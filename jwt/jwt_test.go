package jwt

import (
	"testing"
	"time"
)

func TestNewTokenManager(t *testing.T) {
	secretKey := "test-secret"

	manager := NewTokenManager(secretKey)

	if string(manager.secretKey) != secretKey {
		t.Errorf("Expected secret key %s, got %s", secretKey, string(manager.secretKey))
	}
	if manager.accessTokenExpiry != DefaultJWTOptions().AccessTokenExpiry {
		t.Errorf("Expected default access token expiry %v, got %v", DefaultJWTOptions().AccessTokenExpiry, manager.accessTokenExpiry)
	}
	if manager.refreshTokenExpiry != DefaultJWTOptions().RefreshTokenExpiry {
		t.Errorf("Expected default refresh token expiry %v, got %v", DefaultJWTOptions().RefreshTokenExpiry, manager.refreshTokenExpiry)
	}
	if len(manager.blacklist) != 0 {
		t.Error("Expected empty blacklist map")
	}
}

func TestTokenManager_GenerateToken(t *testing.T) {
	manager := NewTokenManager("test-secret")
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

	// 验证刷新令牌
	claims, err := manager.ValidateToken(refreshToken)
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

func TestTokenManager_ValidateToken(t *testing.T) {
	manager := NewTokenManager("test-secret")
	subject := "123"

	options := &TokenOptions{
		TokenType: AccessToken,
		SessionID: "test-session",
		CustomClaims: map[string]interface{}{
			"role": "admin",
		},
	}

	// 生成有效 token
	token, err := manager.GenerateToken(subject, options)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// 验证有效 token
	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// 验证 claims 内容
	if claims.Subject != subject {
		t.Errorf("Expected subject %s, got %s", subject, claims.Subject)
	}
	if claims.TokenType != AccessToken {
		t.Errorf("Expected token type %s, got %s", AccessToken, claims.TokenType)
	}
	if claims.SessionID != "test-session" {
		t.Errorf("Expected session ID %s, got %s", "test-session", claims.SessionID)
	}

	// 测试无效 token
	_, err = manager.ValidateToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestTokenManager_RefreshToken(t *testing.T) {
	manager := NewTokenManager("test-secret")
	subject := "123"
	sessionID := "test-session"

	// 生成刷新令牌
	refreshOptions := &TokenOptions{
		TokenType: RefreshToken,
		SessionID: sessionID,
	}
	refreshToken, err := manager.GenerateToken(subject, refreshOptions)
	if err != nil {
		t.Fatalf("Failed to generate refresh token: %v", err)
	}

	// 使用刷新令牌获取新的访问令牌
	accessToken, newRefreshToken, err := manager.RefreshToken(refreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh token: %v", err)
	}

	// 验证返回的刷新令牌是否与原始刷新令牌相同
	if newRefreshToken != refreshToken {
		t.Errorf("Expected returned refresh token to match original, got different token")
	}

	// 验证新的访问令牌
	claims, err := manager.ValidateToken(accessToken)
	if err != nil {
		t.Fatalf("Failed to validate new access token: %v", err)
	}

	// 验证访问令牌包含正确信息
	if claims.Subject != subject {
		t.Errorf("Expected subject %s, got %s", subject, claims.Subject)
	}
	if claims.TokenType != AccessToken {
		t.Errorf("Expected token type %s, got %s", AccessToken, claims.TokenType)
	}
	if claims.SessionID != sessionID {
		t.Errorf("Expected session ID %s, got %s", sessionID, claims.SessionID)
	}

	// 尝试使用访问令牌作为刷新令牌 (应该失败)
	_, _, err = manager.RefreshToken(accessToken)
	if err == nil {
		t.Error("Using access token as refresh token should fail")
	}
}

func TestTokenManager_RevokeToken(t *testing.T) {
	manager := NewTokenManager("test-secret")
	subject := "123"
	token, _ := manager.GenerateToken(subject)

	// 测试撤销令牌
	err := manager.RevokeToken(token)
	if err != nil {
		t.Fatalf("Failed to revoke token: %v", err)
	}

	// 验证已撤销的令牌在黑名单中
	if !manager.IsBlacklisted(token) {
		t.Error("Token should be blacklisted")
	}

	// 验证已撤销的令牌不能通过验证
	_, err = manager.ValidateToken(token)
	if err == nil || err.Error() != "令牌已被撤销" {
		t.Errorf("Validation should fail for revoked token, got: %v", err)
	}

	// 测试清理过期的黑名单记录
	lockIndex := manager.getLockIndex(token)
	manager.blacklistLock[lockIndex].Lock()
	manager.blacklist[token] = time.Now().Add(-time.Hour) // 设置为过期时间
	manager.blacklistLock[lockIndex].Unlock()

	manager.CleanBlacklist()
	if manager.IsBlacklisted(token) {
		t.Error("Expired token should be removed from blacklist")
	}
}

func BenchmarkTokenManager_GenerateToken(b *testing.B) {
	manager := NewTokenManager("test-secret")
	subject := "123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateToken(subject)
	}
}

func BenchmarkTokenManager_ValidateToken(b *testing.B) {
	manager := NewTokenManager("test-secret")
	token, _ := manager.GenerateToken("123")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ValidateToken(token)
	}
}

func BenchmarkTokenManager_IsBlacklisted(b *testing.B) {
	manager := NewTokenManager("test-secret")
	token, _ := manager.GenerateToken("123")
	_ = manager.RevokeToken(token)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.IsBlacklisted(token)
	}
}

func TestTokenManager_ValidateToken_Parallel(t *testing.T) {
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
				CustomClaims: map[string]interface{}{"role": "user"},
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

	manager := NewTokenManager("test-secret")

	for _, tt := range tests {
		tt := tt // 捕获循环变量
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.subject == "" {
				// 空主题的情况，直接检查生成令牌是否失败
				_, err := manager.GenerateToken(tt.subject, tt.options)
				if (err != nil) != tt.wantErr {
					t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}

			token, _ := manager.GenerateToken(tt.subject, tt.options)
			if tt.name == "黑名单令牌" {
				_ = manager.RevokeToken(token)
			}
			_, err := manager.ValidateToken(token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestTokenManager_LogControl 测试日志控制功能
func TestTokenManager_LogControl(t *testing.T) {
	// 创建一个默认的令牌管理器（默认禁用日志）
	manager1 := NewTokenManager("test-secret")

	// 通过选项启用日志的令牌管理器
	options := DefaultJWTOptions()
	options.EnableLog = true
	manager2 := NewTokenManager("test-secret", options)

	// 使用EnableLog方法启用或禁用日志
	manager3 := NewTokenManager("test-secret")
	manager3.EnableLog(true)

	// 测试默认值
	if manager1.enableLog {
		t.Error("默认应该禁用日志")
	}

	// 测试通过选项启用日志
	if !manager2.enableLog {
		t.Error("使用选项应成功启用日志")
	}

	// 测试使用方法启用日志
	if !manager3.enableLog {
		t.Error("使用EnableLog方法应成功启用日志")
	}
}
