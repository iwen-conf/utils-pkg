// Package captcha 提供了安全的验证码生成和验证功能。
//
// 本包只负责生成和比较验证码。TTL、存储、限流、尝试次数和成功后消费由下游实现。
package captcha

import (
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"unicode/utf8"
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

	// ErrInvalidCharset 自定义字符集非法
	ErrInvalidCharset = errors.New("invalid captcha charset")
)

// Generate 生成指定长度的数字验证码
func Generate(length int) (string, error) {
	return generateWithReader(length, DigitCharset, rand.Reader)
}

// Generate6 生成推荐的6位数字验证码。
// 只有同时存在短 TTL、尝试次数限制和成功后消费时，6 位才足以限制在线猜测。
func Generate6() (string, error) {
	return Generate(DefaultLength)
}

// Generate4 生成4位数字验证码
func Generate4() (string, error) {
	return Generate(4)
}

// Generate8 生成8位数字验证码
func Generate8() (string, error) {
	return Generate(8)
}

// Validate 比较用户输入与期望验证码。
// 任一 trim 后为空都失败；等长非空值使用恒定时间比较。
func Validate(input, expected string) bool {
	input = strings.TrimSpace(input)
	expected = strings.TrimSpace(expected)
	if input == "" || expected == "" {
		return false
	}
	if len(input) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(input), []byte(expected)) == 1
}

// IsValidFormat 检查字符串是否为有效的数字验证码格式
func IsValidFormat(code string) bool {
	if len(code) < MinLength || len(code) > MaxLength {
		return false
	}
	for _, char := range code {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// GenerateWithCustomCharset 使用自定义字符集按 rune 生成验证码。
func GenerateWithCustomCharset(length int, charset string) (string, error) {
	return generateWithReader(length, charset, rand.Reader)
}

func generateWithReader(length int, charset string, r io.Reader) (string, error) {
	if length < MinLength || length > MaxLength {
		return "", ErrInvalidLength
	}
	runes, err := uniqueRunes(charset)
	if err != nil {
		return "", err
	}
	var builder strings.Builder
	for i := 0; i < length; i++ {
		idx, err := randomIndex(r, len(runes))
		if err != nil {
			return "", err
		}
		builder.WriteRune(runes[idx])
	}
	return builder.String(), nil
}

func uniqueRunes(charset string) ([]rune, error) {
	if charset == "" {
		return nil, fmt.Errorf("%w: charset is empty", ErrInvalidCharset)
	}
	if !utf8.ValidString(charset) {
		return nil, fmt.Errorf("%w: charset is not valid UTF-8", ErrInvalidCharset)
	}
	seen := make(map[rune]struct{}, utf8.RuneCountInString(charset))
	out := make([]rune, 0, utf8.RuneCountInString(charset))
	for _, r := range charset {
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	if len(out) < 2 {
		return nil, fmt.Errorf("%w: need at least two unique runes", ErrInvalidCharset)
	}
	return out, nil
}

func randomIndex(r io.Reader, n int) (int, error) {
	idx, err := rand.Int(r, big.NewInt(int64(n)))
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrGenerationFailed, err)
	}
	return int(idx.Int64()), nil
}
