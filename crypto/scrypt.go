package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/scrypt"
)

// ScryptParams scrypt参数配置
type ScryptParams struct {
	// CPU/内存成本参数
	N int
	// 块大小参数
	R int
	// 并行化参数
	P int
	// Salt长度
	SaltLength int
	// Key长度
	KeyLength int
}

// DefaultScryptParams 返回推荐的scrypt参数
func DefaultScryptParams() *ScryptParams {
	return &ScryptParams{
		N:          32768, // 2^15
		R:          8,     // 块大小
		P:          1,     // 并行化
		SaltLength: 16,    // salt长度
		KeyLength:  32,    // 输出key长度
	}
}

// FastScryptParams 返回快速但安全的scrypt参数
func FastScryptParams() *ScryptParams {
	return &ScryptParams{
		N:          16384, // 2^14
		R:          8,
		P:          1,
		SaltLength: 16,
		KeyLength:  32,
	}
}

// HashWithScrypt 使用scrypt算法哈希密码
func HashWithScrypt(password []byte, params *ScryptParams) (string, error) {
	if params == nil {
		params = DefaultScryptParams()
	}

	// 生成随机salt
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("生成salt失败: %w", err)
	}

	// 使用scrypt生成key
	key, err := scrypt.Key(password, salt, params.N, params.R, params.P, params.KeyLength)
	if err != nil {
		return "", fmt.Errorf("scrypt计算失败: %w", err)
	}

	// 编码格式: $scrypt$N={n},r={r},p={p}${salt}${hash}
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedKey := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$scrypt$N=%d,r=%d,p=%d$%s$%s",
		params.N, params.R, params.P, encodedSalt, encodedKey), nil
}

// VerifyScryptHash 验证scrypt哈希
func VerifyScryptHash(hash, password []byte) (bool, error) {
	// 解析哈希字符串
	parts := strings.Split(string(hash), "$")
	if len(parts) != 5 || parts[1] != "scrypt" {
		return false, errors.New("无效的scrypt哈希格式")
	}

	// 解析参数
	var N, R, P int
	var salt, key []byte

	_, err := fmt.Sscanf(parts[2], "N=%d,r=%d,p=%d", &N, &R, &P)
	if err != nil {
		return false, fmt.Errorf("解析参数失败: %w", err)
	}

	// 解码salt和hash
	salt, err = base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("解码salt失败: %w", err)
	}

	key, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("解码hash失败: %w", err)
	}

	// 使用相同参数重新计算哈希
	computedKey, err := scrypt.Key(password, salt, N, R, P, len(key))
	if err != nil {
		return false, fmt.Errorf("scrypt计算失败: %w", err)
	}

	// 安全比较
	return SecureCompare(key, computedKey), nil
}

// ScryptHasher scrypt哈希器
type ScryptHasher struct {
	params *ScryptParams
}

// NewScryptHasher 创建scrypt哈希器
func NewScryptHasher(params *ScryptParams) *ScryptHasher {
	if params == nil {
		params = DefaultScryptParams()
	}
	return &ScryptHasher{params: params}
}

// Hash 使用scrypt哈希密码
func (s *ScryptHasher) Hash(password []byte) (string, error) {
	return HashWithScrypt(password, s.params)
}

// Verify 验证scrypt哈希
func (s *ScryptHasher) Verify(hash, password []byte) (bool, error) {
	return VerifyScryptHash(hash, password)
}