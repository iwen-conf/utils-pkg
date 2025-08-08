package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2Type Argon2算法类型
type Argon2Type int

const (
	// Argon2i 优化的对抗侧信道攻击
	Argon2i Argon2Type = iota
	// Argon2id 混合模式，推荐使用
	Argon2id
)

// Argon2Params Argon2参数配置
type Argon2Params struct {
	// 内存大小（字节）
	Memory uint32
	// 迭代次数
	Iterations uint32
	// 并行线程数
	Parallelism uint8
	// Salt长度
	SaltLength uint32
	// Key长度
	KeyLength uint32
	// Argon2类型
	Type Argon2Type
}

// DefaultArgon2Params 返回推荐的Argon2参数
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      64 * 1024, // 64MB
		Iterations:  3,
		Parallelism: 4,
		SaltLength:  16,
		KeyLength:   32,
		Type:        Argon2id,
	}
}

// FastArgon2Params 返回快速但安全的Argon2参数
func FastArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      32 * 1024, // 32MB
		Iterations:  2,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
		Type:        Argon2id,
	}
}

// HashWithArgon2 使用Argon2算法哈希密码
func HashWithArgon2(password []byte, params *Argon2Params) (string, error) {
	if params == nil {
		params = DefaultArgon2Params()
	}

	// 生成随机salt
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	// 根据类型选择Argon2变种
	var hash []byte
	switch params.Type {
	case Argon2i:
		hash = argon2.Key(password, salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	default: // Argon2id
		hash = argon2.IDKey(password, salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength)
	}

	// 编码格式: $argon2{type}$v={version}$m={memory},t={iterations},p={parallelism}${salt}${hash}
	version := 19 // Argon2版本
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	typeStr := "id"
	if params.Type == Argon2i {
		typeStr = "i"
	}

	return fmt.Sprintf("$argon2%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		typeStr, version, params.Memory, params.Iterations, params.Parallelism, encodedSalt, encodedHash), nil
}

// VerifyArgon2Hash 验证Argon2哈希
func VerifyArgon2Hash(hash, password []byte) (bool, error) {
	// 解析哈希字符串
	parts := strings.Split(string(hash), "$")
	if len(parts) != 6 {
		return false, errors.New("invalid Argon2 hash format")
	}

	// 解析参数
	var argonType Argon2Type = Argon2id
	if parts[1] == "argon2i" {
		argonType = Argon2i
	} else if parts[1] != "argon2id" {
		return false, errors.New("unsupported Argon2 type")
	}

	var memory, iterations, parallelism uint32
	var salt, key []byte
	var version int

	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil || version != 19 {
		return false, errors.New("unsupported Argon2 version")
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false, fmt.Errorf("failed to parse parameters: %w", err)
	}

	// 检查参数有效性
	if memory == 0 {
		return false, errors.New("memory size cannot be zero")
	}

	// 解码salt和hash
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	key, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// 使用相同参数重新计算哈希
	var computedHash []byte
	if argonType == Argon2i {
		computedHash = argon2.Key(password, salt, iterations, memory, uint8(parallelism), uint32(len(key)))
	} else {
		computedHash = argon2.IDKey(password, salt, iterations, memory, uint8(parallelism), uint32(len(key)))
	}

	// 安全比较
	return SecureCompare(key, computedHash), nil
}

// Argon2Hasher Argon2哈希器
type Argon2Hasher struct {
	params *Argon2Params
}

// NewArgon2Hasher 创建Argon2哈希器
func NewArgon2Hasher(params *Argon2Params) *Argon2Hasher {
	if params == nil {
		params = DefaultArgon2Params()
	}
	return &Argon2Hasher{params: params}
}

// Hash 使用Argon2哈希密码
func (a *Argon2Hasher) Hash(password []byte) (string, error) {
	return HashWithArgon2(password, a.params)
}

// Verify 验证Argon2哈希
func (a *Argon2Hasher) Verify(hash, password []byte) (bool, error) {
	return VerifyArgon2Hash(hash, password)
}