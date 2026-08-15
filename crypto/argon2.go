package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
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

const (
	minArgon2EncodedLength = 1
	maxArgon2EncodedLength = 4096
	minArgon2MemoryKiB     = 8 * 1024
	maxArgon2MemoryKiB     = 256 * 1024
	minArgon2Iterations    = 1
	maxArgon2Iterations    = 10
	minArgon2Parallelism   = 1
	maxArgon2Parallelism   = 16
	minArgon2SaltLength    = 16
	maxArgon2SaltLength    = 64
	minArgon2KeyLength     = 16
	maxArgon2KeyLength     = 64
	argon2Version          = 19
)

// ErrInvalidArgon2Params 表示 Argon2 参数或编码非法。
var ErrInvalidArgon2Params = errors.New("invalid Argon2 params")

// Argon2Params Argon2参数配置
type Argon2Params struct {
	// Memory 是 Argon2 内存参数，单位 KiB。
	Memory uint32
	// Iterations 迭代次数
	Iterations uint32
	// Parallelism 并行度
	Parallelism uint8
	// SaltLength salt 长度
	SaltLength uint32
	// KeyLength 派生密钥长度
	KeyLength uint32
	// Type Argon2 变种
	Type Argon2Type
}

// DefaultArgon2Params 返回推荐的Argon2参数
func DefaultArgon2Params() *Argon2Params {
	return &Argon2Params{
		Memory:      64 * 1024,
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
		Memory:      32 * 1024,
		Iterations:  2,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   32,
		Type:        Argon2id,
	}
}

func copyArgon2Params(params *Argon2Params) Argon2Params {
	if params == nil {
		return *DefaultArgon2Params()
	}
	return *params
}

func validateArgon2Params(params Argon2Params) error {
	if params.Memory < minArgon2MemoryKiB || params.Memory > maxArgon2MemoryKiB {
		return fmt.Errorf("%w: memory must be %d-%d KiB", ErrInvalidArgon2Params, minArgon2MemoryKiB, maxArgon2MemoryKiB)
	}
	if params.Iterations < minArgon2Iterations || params.Iterations > maxArgon2Iterations {
		return fmt.Errorf("%w: iterations must be %d-%d", ErrInvalidArgon2Params, minArgon2Iterations, maxArgon2Iterations)
	}
	if params.Parallelism < minArgon2Parallelism || params.Parallelism > maxArgon2Parallelism {
		return fmt.Errorf("%w: parallelism must be %d-%d", ErrInvalidArgon2Params, minArgon2Parallelism, maxArgon2Parallelism)
	}
	if params.SaltLength < minArgon2SaltLength || params.SaltLength > maxArgon2SaltLength {
		return fmt.Errorf("%w: salt length must be %d-%d", ErrInvalidArgon2Params, minArgon2SaltLength, maxArgon2SaltLength)
	}
	if params.KeyLength < minArgon2KeyLength || params.KeyLength > maxArgon2KeyLength {
		return fmt.Errorf("%w: key length must be %d-%d", ErrInvalidArgon2Params, minArgon2KeyLength, maxArgon2KeyLength)
	}
	if params.Type != Argon2i && params.Type != Argon2id {
		return fmt.Errorf("%w: unsupported Argon2 type", ErrInvalidArgon2Params)
	}
	return nil
}

func computeArgon2(password, salt []byte, params Argon2Params) ([]byte, error) {
	switch params.Type {
	case Argon2i:
		return argon2.Key(password, salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength), nil
	case Argon2id:
		return argon2.IDKey(password, salt, params.Iterations, params.Memory, params.Parallelism, params.KeyLength), nil
	default:
		return nil, fmt.Errorf("%w: unsupported Argon2 type", ErrInvalidArgon2Params)
	}
}

// HashWithArgon2 使用Argon2算法哈希密码
func HashWithArgon2(password []byte, params *Argon2Params) (string, error) {
	copied := copyArgon2Params(params)
	if err := validateArgon2Params(copied); err != nil {
		return "", err
	}

	salt := make([]byte, copied.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash, err := computeArgon2(password, salt, copied)
	if err != nil {
		return "", err
	}

	typeStr := "id"
	if copied.Type == Argon2i {
		typeStr = "i"
	}
	return fmt.Sprintf("$argon2%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		typeStr, argon2Version, copied.Memory, copied.Iterations, copied.Parallelism,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash)), nil
}

// VerifyArgon2Hash 验证Argon2哈希
func VerifyArgon2Hash(hash, password []byte) (bool, error) {
	if len(hash) < minArgon2EncodedLength || len(hash) > maxArgon2EncodedLength {
		return false, fmt.Errorf("%w: encoded hash length", ErrInvalidArgon2Params)
	}

	parts := strings.Split(string(hash), "$")
	if len(parts) != 6 || parts[0] != "" {
		return false, fmt.Errorf("%w: invalid Argon2 hash format", ErrInvalidArgon2Params)
	}

	var params Argon2Params
	switch parts[1] {
	case "argon2i":
		params.Type = Argon2i
	case "argon2id":
		params.Type = Argon2id
	default:
		return false, fmt.Errorf("%w: unsupported Argon2 type", ErrInvalidArgon2Params)
	}

	version, err := parseExactPrefixedInt(parts[2], "v=")
	if err != nil || version != argon2Version {
		return false, fmt.Errorf("%w: unsupported Argon2 version: %w", ErrInvalidArgon2Params, err)
	}

	memory, iterations, parallelism, err := parseArgon2MTP(parts[3])
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrInvalidArgon2Params, err)
	}
	if parallelism < minArgon2Parallelism || parallelism > maxArgon2Parallelism {
		return false, fmt.Errorf("%w: parallelism must be %d-%d", ErrInvalidArgon2Params, minArgon2Parallelism, maxArgon2Parallelism)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("%w: decode salt: %w", ErrInvalidArgon2Params, err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("%w: decode hash: %w", ErrInvalidArgon2Params, err)
	}

	params.Memory = memory
	params.Iterations = iterations
	params.Parallelism = uint8(parallelism)
	params.SaltLength = uint32(len(salt))
	params.KeyLength = uint32(len(key))
	if err := validateArgon2Params(params); err != nil {
		return false, err
	}

	computed, err := computeArgon2(password, salt, params)
	if err != nil {
		return false, err
	}
	return SecureCompare(key, computed), nil
}

func parseArgon2MTP(raw string) (memory, iterations, parallelism uint32, err error) {
	fields := strings.Split(raw, ",")
	if len(fields) != 3 {
		return 0, 0, 0, errors.New("parameter block must have m,t,p")
	}
	memory64, err := parseExactPrefixedInt(fields[0], "m=")
	if err != nil {
		return 0, 0, 0, err
	}
	iter64, err := parseExactPrefixedInt(fields[1], "t=")
	if err != nil {
		return 0, 0, 0, err
	}
	par64, err := parseExactPrefixedInt(fields[2], "p=")
	if err != nil {
		return 0, 0, 0, err
	}
	return uint32(memory64), uint32(iter64), uint32(par64), nil
}

func parseExactPrefixedInt(raw, prefix string) (int, error) {
	if !strings.HasPrefix(raw, prefix) {
		return 0, fmt.Errorf("missing prefix %q", prefix)
	}
	value, err := strconv.ParseInt(raw[len(prefix):], 10, 32)
	if err != nil {
		return 0, err
	}
	return int(value), nil
}

// Argon2Hasher Argon2哈希器
type Argon2Hasher struct {
	params Argon2Params
}

// NewArgon2Hasher 创建Argon2哈希器，保存参数副本。
func NewArgon2Hasher(params *Argon2Params) (*Argon2Hasher, error) {
	copied := copyArgon2Params(params)
	if err := validateArgon2Params(copied); err != nil {
		return nil, err
	}
	return &Argon2Hasher{params: copied}, nil
}

// Hash 使用Argon2哈希密码
func (a *Argon2Hasher) Hash(password []byte) (string, error) {
	copied := a.params
	return HashWithArgon2(password, &copied)
}

// Verify 验证Argon2哈希
func (a *Argon2Hasher) Verify(hash, password []byte) (bool, error) {
	return VerifyArgon2Hash(hash, password)
}
