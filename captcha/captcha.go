// Package captcha 提供了安全的验证码生成和验证功能。
//
// 本包专注于生成密码学安全的数字验证码，适用于用户身份验证、
// 防止自动化攻击等场景。所有验证码都使用密码学安全伪随机数生成器(CSPRNG)生成。
//
// 主要特性：
//   - 使用crypto/rand确保密码学安全的随机性
//   - 纯数字字符集，避免用户输入混淆
//   - 推荐6位长度，平衡安全性和用户体验
//   - 灵活的长度配置
//   - 简单的验证功能
//
// 使用示例：
//
//	package main
//
//	import (
//		"fmt"
//		"github.com/iwen-conf/utils-pkg/captcha"
//	)
//
//	func main() {
//		// 生成推荐的6位验证码
//		code, err := captcha.Generate6()
//		if err != nil {
//			panic(err)
//		}
//		fmt.Printf("验证码: %s\n", code)
//
//		// 生成自定义长度的验证码
//		code8, err := captcha.Generate(8)
//		if err != nil {
//			panic(err)
//		}
//		fmt.Printf("8位验证码: %s\n", code8)
//
//		// 验证验证码
//		if captcha.Validate(userInput, code) {
//			fmt.Println("验证码正确")
//		} else {
//			fmt.Println("验证码错误")
//		}
//	}
package captcha

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const (
	// DefaultLength 推荐的验证码长度
	DefaultLength = 6
	
	// MinLength 最小验证码长度
	MinLength = 4
	
	// MaxLength 最大验证码长度
	MaxLength = 12
	
	// DigitCharset 数字字符集
	DigitCharset = "0123456789"
)

var (
	// ErrInvalidLength 无效的验证码长度错误
	ErrInvalidLength = errors.New("验证码长度必须在4-12位之间")
	
	// ErrGenerationFailed 验证码生成失败错误
	ErrGenerationFailed = errors.New("验证码生成失败")
)

// Generate 生成指定长度的数字验证码
//
// 参数：
//   - length: 验证码长度，必须在4-12位之间
//
// 返回值：
//   - string: 生成的验证码
//   - error: 错误信息，如果生成成功则为nil
//
// 安全性：
//   - 使用crypto/rand包确保密码学安全的随机性
//   - 每个数字位都是独立随机生成的
//   - 不存在可预测的模式
func Generate(length int) (string, error) {
	// 验证长度参数
	if length < MinLength || length > MaxLength {
		return "", ErrInvalidLength
	}
	
	// 预分配字符串构建器
	var builder strings.Builder
	builder.Grow(length)
	
	// 字符集长度
	charsetLen := big.NewInt(int64(len(DigitCharset)))
	
	// 生成每一位数字
	for i := 0; i < length; i++ {
		// 使用密码学安全的随机数生成器
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrGenerationFailed, err)
		}
		
		// 添加随机字符到结果中
		builder.WriteByte(DigitCharset[randomIndex.Int64()])
	}
	
	return builder.String(), nil
}

// Generate6 生成推荐的6位数字验证码
//
// 这是Generate(6)的便捷函数，6位数字提供了100万种组合(10^6)，
// 对于有时效性和尝试次数限制的验证码来说，这个数量级足以抵御在线暴力破解。
//
// 返回值：
//   - string: 生成的6位验证码
//   - error: 错误信息，如果生成成功则为nil
func Generate6() (string, error) {
	return Generate(DefaultLength)
}

// Generate4 生成4位数字验证码
//
// 4位验证码提供1万种组合(10^4)，安全性较低，
// 建议仅在对安全性要求不高的场景使用。
//
// 返回值：
//   - string: 生成的4位验证码
//   - error: 错误信息，如果生成成功则为nil
func Generate4() (string, error) {
	return Generate(4)
}

// Generate8 生成8位数字验证码
//
// 8位验证码提供1亿种组合(10^8)，安全性很高，
// 适用于对安全性要求极高的场景。
//
// 返回值：
//   - string: 生成的8位验证码
//   - error: 错误信息，如果生成成功则为nil
func Generate8() (string, error) {
	return Generate(8)
}

// Validate 验证验证码是否匹配
//
// 参数：
//   - input: 用户输入的验证码
//   - expected: 期望的验证码
//
// 返回值：
//   - bool: 如果验证码匹配返回true，否则返回false
//
// 注意：
//   - 验证是大小写敏感的（虽然数字验证码不涉及大小写）
//   - 会自动去除输入两端的空白字符
func Validate(input, expected string) bool {
	// 去除两端空白字符
	input = strings.TrimSpace(input)
	expected = strings.TrimSpace(expected)
	
	// 简单的字符串比较
	return input == expected
}

// IsValidFormat 检查字符串是否为有效的验证码格式
//
// 参数：
//   - code: 要检查的验证码字符串
//
// 返回值：
//   - bool: 如果格式有效返回true，否则返回false
//
// 验证规则：
//   - 长度在4-12位之间
//   - 只包含数字字符
func IsValidFormat(code string) bool {
	// 检查长度
	if len(code) < MinLength || len(code) > MaxLength {
		return false
	}
	
	// 检查是否只包含数字
	for _, char := range code {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return true
}

// GenerateWithCustomCharset 使用自定义字符集生成验证码
//
// 参数：
//   - length: 验证码长度
//   - charset: 自定义字符集
//
// 返回值：
//   - string: 生成的验证码
//   - error: 错误信息
//
// 注意：虽然提供了自定义字符集的功能，但强烈建议使用默认的数字字符集
// 以避免用户输入时的混淆（如O和0，l和I和1等）
func GenerateWithCustomCharset(length int, charset string) (string, error) {
	// 验证参数
	if length < MinLength || length > MaxLength {
		return "", ErrInvalidLength
	}
	
	if len(charset) == 0 {
		return "", errors.New("字符集不能为空")
	}
	
	// 预分配字符串构建器
	var builder strings.Builder
	builder.Grow(length)
	
	// 字符集长度
	charsetLen := big.NewInt(int64(len(charset)))
	
	// 生成每一位字符
	for i := 0; i < length; i++ {
		// 使用密码学安全的随机数生成器
		randomIndex, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("%w: %v", ErrGenerationFailed, err)
		}
		
		// 添加随机字符到结果中
		builder.WriteByte(charset[randomIndex.Int64()])
	}
	
	return builder.String(), nil
}
