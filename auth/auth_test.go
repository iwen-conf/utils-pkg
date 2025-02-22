package auth

import (
	"testing"
	"time"
)

func TestNewAuthManager(t *testing.T) {
	secretKey := "test-secret"
	accessExpires := time.Hour
	refreshExpires := 24 * time.Hour

	manager := NewAuthManager(secretKey, accessExpires, refreshExpires)

	if manager.accessExpires != accessExpires {
		t.Errorf("Expected access expires duration %v, got %v", accessExpires, manager.accessExpires)
	}
	if manager.refreshExpires != refreshExpires {
		t.Errorf("Expected refresh expires duration %v, got %v", refreshExpires, manager.refreshExpires)
	}
	if len(manager.refreshTokens) != 0 {
		t.Error("Expected empty refresh tokens map")
	}
}

func TestAuthManager_GenerateTokenPair(t *testing.T) {
	manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)
	userID := "123"
	extra := map[string]interface{}{"role": "admin"}

	// 生成令牌对
	tokenPair, err := manager.GenerateTokenPair(userID, extra)
	if err != nil {
		t.Fatalf("Failed to generate token pair: %v", err)
	}

	// 验证访问令牌
	claims, err := manager.ValidateAccessToken(tokenPair.AccessToken)
	if err != nil {
		t.Fatalf("Failed to validate access token: %v", err)
	}

	// 验证令牌内容
	if claims.UserID != userID {
		t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
	}
	
	if claims.Extra["role"] != "admin" {
		t.Errorf("Expected role admin, got %v", claims.Extra["role"])
	}
}

func TestAuthManager_RefreshAccessToken(t *testing.T) {
	manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)

	// 生成初始令牌对
	initialPair, _ := manager.GenerateTokenPair("123", nil)

	// 使用刷新令牌获取新的令牌对
	newPair, err := manager.RefreshAccessToken(initialPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to refresh access token: %v", err)
	}

	// 验证新的访问令牌
	_, err = manager.ValidateAccessToken(newPair.AccessToken)
	if err != nil {
		t.Errorf("New access token should be valid: %v", err)
	}

	// 验证新旧访问令牌不同
	if newPair.AccessToken == initialPair.AccessToken {
		t.Error("New access token should be different from the initial one")
	}
}

func TestAuthManager_RevokeRefreshToken(t *testing.T) {
	manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)

	// 生成令牌对
	tokenPair, _ := manager.GenerateTokenPair("123", nil)

	// 撤销刷新令牌
	err := manager.RevokeRefreshToken(tokenPair.RefreshToken)
	if err != nil {
		t.Fatalf("Failed to revoke refresh token: %v", err)
	}

	// 尝试使用已撤销的刷新令牌
	_, err = manager.RefreshAccessToken(tokenPair.RefreshToken)
	if err == nil {
		t.Error("Should not be able to use revoked refresh token")
	}
}

func BenchmarkAuthManager_GenerateTokenPair(b *testing.B) {
	manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)
	userID := "123"
	extra := map[string]interface{}{"role": "admin"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateTokenPair(userID, extra)
	}
}

func BenchmarkAuthManager_ValidateAccessToken(b *testing.B) {
	manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)
	tokenPair, _ := manager.GenerateTokenPair("123", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.ValidateAccessToken(tokenPair.AccessToken)
	}
}

func TestAuthManager_GenerateTokenPair_Parallel(t *testing.T) {
	tests := []struct {
		name    string
		userID  string
		extra   map[string]interface{}
		wantErr bool
	}{
		{
			name:    "正常用户",
			userID:  "123",
			extra:   map[string]interface{}{"role": "user"},
			wantErr: false,
		},
		{
			name:    "管理员用户",
			userID:  "456",
			extra:   map[string]interface{}{"role": "admin"},
			wantErr: false,
		},
		{
			name:    "空用户ID",
			userID:  "",
			extra:   nil,
			wantErr: true,
		},
	}

	manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)

	for _, tt := range tests {
		tt := tt // 捕获循环变量
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := manager.GenerateTokenPair(tt.userID, tt.extra)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateTokenPair() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}