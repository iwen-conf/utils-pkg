package jwt

import (
	"testing"
	"time"
	"github.com/golang-jwt/jwt/v5"
)

func TestNewJWTManager(t *testing.T) {
	secretKey := "test-secret"
	expires := 24 * time.Hour

	manager := NewJWTManager(secretKey, expires)

	if string(manager.secretKey) != secretKey {
		t.Errorf("Expected secret key %s, got %s", secretKey, string(manager.secretKey))
	}
	if manager.expires != expires {
		t.Errorf("Expected expires duration %v, got %v", expires, manager.expires)
	}
	if len(manager.validators) != 0 {
		t.Error("Expected empty validators slice")
	}
	if len(manager.blacklist) != 0 {
		t.Error("Expected empty blacklist map")
	}
}

func TestJWTManager_GenerateToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)
	userID := "123"
	extra := map[string]interface{}{"role": "admin"}

	// 测试默认过期时间
	token1, err := manager.GenerateToken(userID, extra)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	if token1 == "" {
		t.Error("Generated token is empty")
	}

	// 测试自定义过期时间
	customExpires := 2 * time.Hour
	token2, err := manager.GenerateToken(userID, extra, customExpires)
	if err != nil {
		t.Fatalf("Failed to generate token with custom expires: %v", err)
	}
	if token2 == "" {
		t.Error("Generated token with custom expires is empty")
	}
	if token1 == token2 {
		t.Error("Tokens with different expires should be different")
	}
}

func TestJWTManager_ValidateToken(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)
	userID := "123"
	extra := map[string]interface{}{"role": "admin"}

	// 生成有效 token
	token, err := manager.GenerateToken(userID, extra)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// 验证有效 token
	claims, err := manager.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// 验证 claims 内容
	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}
	if claims.Extra["role"] != "admin" {
		t.Errorf("Expected role admin, got %v", claims.Extra["role"])
	}

	// 测试无效 token
	_, err = manager.ValidateToken("invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
	}
}

func TestJWTManager_AddValidator(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)

	// 添加自定义验证器
	validator := func(claims *Claims) error {
		if claims.Extra["role"] == "blocked" {
			return jwt.ErrTokenInvalidClaims
		}
		return nil
	}
	manager.AddValidator(validator)

	// 测试正常用户
	token1, _ := manager.GenerateToken("123", map[string]interface{}{"role": "normal"})
	_, err := manager.ValidateToken(token1)
	if err != nil {
		t.Errorf("Validation should pass for normal user: %v", err)
	}

	// 测试被阻止的用户
	token2, _ := manager.GenerateToken("456", map[string]interface{}{"role": "blocked"})
	_, err = manager.ValidateToken(token2)
	if err != jwt.ErrTokenInvalidClaims {
		t.Error("Validation should fail for blocked user")
	}
}

func TestJWTManager_Blacklist(t *testing.T) {
	manager := NewJWTManager("test-secret", time.Hour)
	token, _ := manager.GenerateToken("123", nil)

	// 测试添加到黑名单
	expireAt := time.Now().Add(time.Hour)
	err := manager.AddToBlacklist(token, expireAt)
	if err != nil {
		t.Fatalf("Failed to add token to blacklist: %v", err)
	}

	// 验证黑名单中的 token
	if !manager.IsBlacklisted(token) {
		t.Error("Token should be blacklisted")
	}

	// 验证黑名单中的 token 不能通过验证
	_, err = manager.ValidateToken(token)
	if err == nil || err.Error() != "token is blacklisted" {
		t.Error("Validation should fail for blacklisted token")
	}

	// 测试清理过期的黑名单记录
	manager.blacklist[token] = time.Now().Add(-time.Hour) // 设置为过期时间
	manager.CleanBlacklist()
	if manager.IsBlacklisted(token) {
		t.Error("Expired token should be removed from blacklist")
	}
}

func BenchmarkJWTManager_GenerateToken(b *testing.B) {
	manager := NewJWTManager("test-secret", time.Hour)
	userID := "123"
	extra := map[string]interface{}{"role": "admin"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateToken(userID, extra)
	}
}

func BenchmarkJWTManager_ValidateToken(b *testing.B) {
	manager := NewJWTManager("test-secret", time.Hour)
	token, _ := manager.GenerateToken("123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ValidateToken(token)
	}
}

func BenchmarkJWTManager_IsBlacklisted(b *testing.B) {
	manager := NewJWTManager("test-secret", time.Hour)
	token, _ := manager.GenerateToken("123", nil)
	_ = manager.AddToBlacklist(token, time.Now().Add(time.Hour))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.IsBlacklisted(token)
	}
}

func TestJWTManager_ValidateToken_Parallel(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		extra   map[string]interface{}
		wantErr bool
	}{
		{
			name:    "正常令牌",
			userID:  "123",
			extra:   map[string]interface{}{"role": "user"},
			wantErr: false,
		},
		{
			name:    "空用户ID",
			userID:  "",
			extra:   nil,
			wantErr: true,
		},
		{
			name:    "黑名单令牌",
			userID:  "456",
			extra:   nil,
			wantErr: true,
		},
	}

	manager := NewJWTManager("test-secret", time.Hour)

	for _, tt := range tests {
		tt := tt // 捕获循环变量
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			token, _ := manager.GenerateToken(tt.userID, tt.extra)
			if tt.name == "黑名单令牌" {
				_ = manager.AddToBlacklist(token, time.Now().Add(time.Hour))
			}
			_, err := manager.ValidateToken(token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}