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

// TestAuthManager_RefreshAccessToken_Enhanced 增强的刷新令牌测试
func TestAuthManager_RefreshAccessToken_Enhanced(t *testing.T) {
	t.Run("测试正常刷新流程", func(t *testing.T) {
		manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)
		userID := "123"
		extra := map[string]interface{}{"role": "admin"}

		// 生成初始令牌对
		initialPair, _ := manager.GenerateTokenPair(userID, extra)

		// 使用刷新令牌获取新的令牌对
		newPair, err := manager.RefreshAccessToken(initialPair.RefreshToken)
		if err != nil {
			t.Fatalf("Failed to refresh access token: %v", err)
		}

		// 验证新的访问令牌
		claims, err := manager.ValidateAccessToken(newPair.AccessToken)
		if err != nil {
			t.Errorf("New access token should be valid: %v", err)
		}

		// 验证用户ID保持不变
		if claims.UserID != userID {
			t.Errorf("Expected user ID %s, got %s", userID, claims.UserID)
		}

		// 验证额外信息正确传递
		if claims.Extra["role"] != "admin" {
			t.Errorf("Expected role admin, got %v", claims.Extra["role"])
		}

		// 确保新的访问令牌不包含 token_type 字段
		if _, exists := claims.Extra["token_type"]; exists {
			t.Errorf("New access token should not contain token_type field")
		}
	})

	t.Run("测试旧刷新令牌不能再次使用", func(t *testing.T) {
		manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)

		// 生成初始令牌对
		initialPair, _ := manager.GenerateTokenPair("123", nil)

		// 第一次刷新
		_, err := manager.RefreshAccessToken(initialPair.RefreshToken)
		if err != nil {
			t.Fatalf("First refresh should succeed: %v", err)
		}

		// 尝试再次使用同一个刷新令牌
		_, err = manager.RefreshAccessToken(initialPair.RefreshToken)
		if err == nil {
			t.Error("Should not be able to use the same refresh token twice")
		}
	})

	t.Run("测试刷新令牌不在存储中", func(t *testing.T) {
		manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)

		// 生成令牌对但不保存到存储中
		tokenStr, _ := manager.jwtManager.GenerateToken("123", map[string]interface{}{
			"token_type": "refresh",
		}, manager.refreshExpires)

		// 尝试使用未存储的刷新令牌
		_, err := manager.RefreshAccessToken(tokenStr)
		if err == nil {
			t.Error("Should not be able to use a refresh token that is not in storage")
		}
	})

	t.Run("测试刷新令牌中缺少token_type", func(t *testing.T) {
		manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)

		// 生成没有token_type的令牌
		tokenStr, _ := manager.jwtManager.GenerateToken("123", nil, manager.refreshExpires)

		// 手动添加到刷新令牌存储中
		manager.refreshTokensLock.Lock()
		manager.refreshTokens[tokenStr] = "123"
		manager.refreshTokensLock.Unlock()

		// 尝试使用缺少token_type的刷新令牌
		_, err := manager.RefreshAccessToken(tokenStr)
		if err == nil {
			t.Error("Should not be able to use a refresh token without token_type")
		}
	})

	t.Run("测试刷新令牌过期", func(t *testing.T) {
		// 创建一个短期过期的管理器
		manager := NewAuthManager("test-secret", time.Hour, 1*time.Millisecond)

		// 生成初始令牌对
		initialPair, _ := manager.GenerateTokenPair("123", nil)

		// 等待令牌过期
		time.Sleep(10 * time.Millisecond)

		// 尝试使用过期的刷新令牌
		_, err := manager.RefreshAccessToken(initialPair.RefreshToken)
		if err == nil {
			t.Error("Should not be able to use an expired refresh token")
		}
	})

	t.Run("测试Extra为nil的情况", func(t *testing.T) {
		manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)

		// 生成一个Extra为nil的令牌，但手动添加token_type
		tokenStr, _ := manager.jwtManager.GenerateToken("123", map[string]interface{}{
			"token_type": "refresh",
		}, manager.refreshExpires)

		// 手动添加到刷新令牌存储中
		manager.refreshTokensLock.Lock()
		manager.refreshTokens[tokenStr] = "123"
		manager.refreshTokensLock.Unlock()

		// 应该能成功刷新
		_, err := manager.RefreshAccessToken(tokenStr)
		if err != nil {
			t.Errorf("Should be able to refresh with minimal Extra data: %v", err)
		}
	})
}

// TestRefreshTokenBlacklistIssue 测试登录后 refresh_token 是否立即被拉黑
func TestRefreshTokenBlacklistIssue(t *testing.T) {
	manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)
	userID := "123"
	extra := map[string]interface{}{"role": "admin"}

	// 生成令牌对，模拟用户登录
	tokenPair, err := manager.GenerateTokenPair(userID, extra)
	if err != nil {
		t.Fatalf("生成令牌对失败: %v", err)
	}

	// 检查 refresh_token 是否已经在黑名单中
	isBlacklisted := manager.jwtManager.IsBlacklisted(tokenPair.RefreshToken)
	if isBlacklisted {
		t.Error("登录后的 refresh_token 不应该在黑名单中，但现在却是")
	}

	// 验证 refresh_token 是否可以使用
	_, err = manager.RefreshAccessToken(tokenPair.RefreshToken)
	if err != nil {
		t.Errorf("无法使用登录后的 refresh_token: %v", err)
	}
}

// TestConsecutiveRefreshes 测试连续多次刷新令牌的场景
func TestConsecutiveRefreshes(t *testing.T) {
	manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)
	userID := "123"
	extra := map[string]interface{}{"role": "admin"}

	// 生成初始令牌对
	initialPair, err := manager.GenerateTokenPair(userID, extra)
	if err != nil {
		t.Fatalf("生成初始令牌对失败: %v", err)
	}

	// 保存初始刷新令牌，以便后续测试
	initialRefreshToken := initialPair.RefreshToken

	// 检查初始刷新令牌是否在黑名单中
	if manager.jwtManager.IsBlacklisted(initialRefreshToken) {
		t.Error("登录后的 refresh_token 不应该在黑名单中，但它在")
	} else {
		t.Log("登录后的 refresh_token 正确地不在黑名单中")
	}

	// 第一次刷新
	firstRefresh, err := manager.RefreshAccessToken(initialRefreshToken)
	if err != nil {
		t.Fatalf("第一次刷新失败: %v", err)
	} else {
		t.Log("第一次刷新成功")
	}

	// 验证初始刷新令牌已经不能再使用
	if manager.jwtManager.IsBlacklisted(initialRefreshToken) {
		t.Log("初始刷新令牌已正确地加入黑名单")
	} else {
		t.Error("初始刷新令牌应该在黑名单中，但它不在")
	}

	_, err = manager.RefreshAccessToken(initialRefreshToken)
	if err == nil {
		t.Error("初始刷新令牌在第一次刷新后仍然可用，这不符合预期")
	} else {
		t.Logf("初始刷新令牌正确失效: %v", err)
	}

	// 检查新生成的刷新令牌是否在黑名单中
	firstRefreshToken := firstRefresh.RefreshToken
	if manager.jwtManager.IsBlacklisted(firstRefreshToken) {
		t.Error("第一次刷新生成的 refresh_token 不应该在黑名单中，但它在")
	} else {
		t.Log("第一次刷新生成的 refresh_token 正确地不在黑名单中")
	}

	// 检查 refreshTokens map 中是否包含新生成的令牌
	manager.refreshTokensLock.RLock()
	_, exists := manager.refreshTokens[firstRefreshToken]
	manager.refreshTokensLock.RUnlock()
	if !exists {
		t.Error("第一次刷新生成的 refresh_token 应该在存储中，但它不在")
	} else {
		t.Log("第一次刷新生成的 refresh_token 正确地存在于存储中")
	}

	// 第二次刷新
	secondRefresh, err := manager.RefreshAccessToken(firstRefreshToken)
	if err != nil {
		t.Fatalf("使用第一次刷新后的令牌进行第二次刷新失败: %v", err)
	} else {
		t.Log("第二次刷新成功")
	}

	// 验证第一次生成的刷新令牌已经不能再使用
	if manager.jwtManager.IsBlacklisted(firstRefreshToken) {
		t.Log("第一次刷新生成的令牌已正确地加入黑名单")
	} else {
		t.Error("第一次刷新生成的令牌应该在黑名单中，但它不在")
	}

	// 第三次刷新
	_, err = manager.RefreshAccessToken(secondRefresh.RefreshToken)
	if err != nil {
		t.Fatalf("第三次刷新失败: %v", err)
	} else {
		t.Log("第三次刷新成功")
	}
}

// TestRefreshTokenIdentity 测试刷新令牌的值是否有变化
func TestRefreshTokenIdentity(t *testing.T) {
	manager := NewAuthManager("test-secret", time.Hour, 24*time.Hour)
	userID := "123"
	extra := map[string]interface{}{"role": "admin"}

	// 生成初始令牌对
	initialPair, _ := manager.GenerateTokenPair(userID, extra)
	initialRefreshToken := initialPair.RefreshToken

	// 打印前10个字符做比较
	t.Logf("初始 refresh_token 前缀: %s...", initialRefreshToken[:10])

	// 第一次刷新
	firstRefresh, _ := manager.RefreshAccessToken(initialRefreshToken)
	firstRefreshToken := firstRefresh.RefreshToken

	// 打印前10个字符做比较
	t.Logf("第一次刷新生成的 refresh_token 前缀: %s...", firstRefreshToken[:10])

	// 比较两个令牌是否相同
	if initialRefreshToken == firstRefreshToken {
		t.Error("初始 refresh_token 和刷新后的 refresh_token 不应该相同，但它们相同")
	} else {
		t.Log("初始 refresh_token 和刷新后的 refresh_token 正确地不同")
	}

	// 检查旧令牌是否在黑名单中
	if manager.jwtManager.IsBlacklisted(initialRefreshToken) {
		t.Log("初始 refresh_token 正确地在黑名单中")
	} else {
		t.Error("初始 refresh_token 应该在黑名单中，但它不在")
	}

	// 检查新令牌是否在黑名单中
	if manager.jwtManager.IsBlacklisted(firstRefreshToken) {
		t.Error("刷新后的 refresh_token 不应该在黑名单中，但它在")
	} else {
		t.Log("刷新后的 refresh_token 正确地不在黑名单中")
	}
}

// TestAuthManager_LogControl 测试日志控制功能
func TestAuthManager_LogControl(t *testing.T) {
	// 创建一个默认的Auth管理器（默认禁用日志）
	manager1 := NewAuthManager("test-secret", time.Hour, 24*time.Hour)

	// 通过选项启用日志的Auth管理器
	options := DefaultAuthOptions()
	options.EnableLog = true
	manager2 := NewAuthManager("test-secret", time.Hour, 24*time.Hour, options)

	// 使用EnableLog方法启用或禁用日志
	manager3 := NewAuthManager("test-secret", time.Hour, 24*time.Hour)
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

	// 测试使用方法禁用日志
	manager3.EnableLog(false)
	if manager3.enableLog {
		t.Error("使用EnableLog方法应成功禁用日志")
	}
}
