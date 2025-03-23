package main

import (
	"fmt"
	"time"

	"github.com/iwen-conf/utils-pkg/auth"
)

func main() {
	// 示例1：使用默认选项（禁用日志）
	fmt.Println("示例1：使用默认选项（禁用日志）")
	authManager1 := auth.NewAuthManager(
		"your-secret-key",
		1*time.Hour,  // 访问令牌有效期1小时
		24*time.Hour, // 刷新令牌有效期24小时
	)

	// 生成令牌对
	tokenPair1, err := authManager1.GenerateTokenPair("123", map[string]interface{}{
		"role": "admin",
	})
	if err != nil {
		fmt.Printf("生成令牌失败: %v\n", err)
		return
	}

	fmt.Printf("访问令牌: %s...\n", tokenPair1.AccessToken[:20])
	fmt.Printf("刷新令牌: %s...\n", tokenPair1.RefreshToken[:20])
	fmt.Println("注意：默认情况下没有日志输出")
	fmt.Println()

	// 示例2：使用选项启用日志
	fmt.Println("示例2：使用选项启用日志")
	options := auth.DefaultAuthOptions()
	options.EnableLog = true

	authManager2 := auth.NewAuthManager(
		"your-secret-key",
		1*time.Hour,
		24*time.Hour,
		options,
	)

	// 生成令牌对
	tokenPair2, err := authManager2.GenerateTokenPair("456", map[string]interface{}{
		"role": "user",
	})
	if err != nil {
		fmt.Printf("生成令牌失败: %v\n", err)
		return
	}

	fmt.Printf("访问令牌: %s...\n", tokenPair2.AccessToken[:20])
	fmt.Printf("刷新令牌: %s...\n", tokenPair2.RefreshToken[:20])
	fmt.Println("注意：上面应该有日志输出")
	fmt.Println()

	// 示例3：动态开关日志
	fmt.Println("示例3：动态开关日志")
	authManager3 := auth.NewAuthManager(
		"your-secret-key",
		1*time.Hour,
		24*time.Hour,
	)

	fmt.Println("默认情况（禁用日志）:")
	authManager3.GenerateTokenPair("789", nil)

	fmt.Println("\n启用日志后:")
	authManager3.EnableLog(true)
	tokenPair3, _ := authManager3.GenerateTokenPair("789", nil)

	fmt.Println("\n禁用日志后:")
	authManager3.EnableLog(false)
	authManager3.RefreshAccessToken(tokenPair3.RefreshToken)
}
