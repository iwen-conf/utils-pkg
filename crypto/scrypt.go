package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"golang.org/x/crypto/scrypt"
)

const (
	minScryptEncodedLength = 1
	maxScryptEncodedLength = 4096
	minScryptR             = 1
	maxScryptR             = 32
	minScryptP             = 1
	maxScryptP             = 16
	minScryptSaltLength    = 16
	maxScryptSaltLength    = 64
	minScryptKeyLength     = 16
	maxScryptKeyLength     = 64
	maxScryptMemoryBytes   = 256 << 20
	maxScryptRP            = 1 << 30
)

// ErrInvalidScryptParams 表示 scrypt 参数或编码非法。
var ErrInvalidScryptParams = errors.New("invalid scrypt params")

// ScryptParams scrypt参数配置
type ScryptParams struct {
	N          int
	R          int
	P          int
	SaltLength int
	KeyLength  int
}

// DefaultScryptParams 返回推荐的scrypt参数
func DefaultScryptParams() *ScryptParams {
	return &ScryptParams{
		N:          32768,
		R:          8,
		P:          1,
		SaltLength: 16,
		KeyLength:  32,
	}
}

// FastScryptParams 返回快速但安全的scrypt参数
func FastScryptParams() *ScryptParams {
	return &ScryptParams{
		N:          16384,
		R:          8,
		P:          1,
		SaltLength: 16,
		KeyLength:  32,
	}
}

func copyScryptParams(params *ScryptParams) ScryptParams {
	if params == nil {
		return *DefaultScryptParams()
	}
	return *params
}

func mul64(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, true
	}
	if a > math.MaxInt64/b {
		return 0, false
	}
	return a * b, true
}

func validateScryptParams(params ScryptParams) error {
	if params.N <= 1 || params.N&(params.N-1) != 0 {
		return fmt.Errorf("%w: N must be a power of two greater than 1", ErrInvalidScryptParams)
	}
	if params.R < minScryptR || params.R > maxScryptR {
		return fmt.Errorf("%w: r must be %d-%d", ErrInvalidScryptParams, minScryptR, maxScryptR)
	}
	if params.P < minScryptP || params.P > maxScryptP {
		return fmt.Errorf("%w: p must be %d-%d", ErrInvalidScryptParams, minScryptP, maxScryptP)
	}
	if params.SaltLength < minScryptSaltLength || params.SaltLength > maxScryptSaltLength {
		return fmt.Errorf("%w: salt length must be %d-%d", ErrInvalidScryptParams, minScryptSaltLength, maxScryptSaltLength)
	}
	if params.KeyLength < minScryptKeyLength || params.KeyLength > maxScryptKeyLength {
		return fmt.Errorf("%w: key length must be %d-%d", ErrInvalidScryptParams, minScryptKeyLength, maxScryptKeyLength)
	}

	mem, ok := mul64(128, int64(params.N))
	if !ok {
		return fmt.Errorf("%w: memory estimate overflow", ErrInvalidScryptParams)
	}
	mem, ok = mul64(mem, int64(params.R))
	if !ok || mem > maxScryptMemoryBytes {
		return fmt.Errorf("%w: estimated memory exceeds 256MiB", ErrInvalidScryptParams)
	}

	rp, ok := mul64(int64(params.R), int64(params.P))
	if !ok || rp >= maxScryptRP {
		return fmt.Errorf("%w: r*p exceeds scrypt limit", ErrInvalidScryptParams)
	}
	return nil
}

// HashWithScrypt 使用scrypt算法哈希密码
func HashWithScrypt(password []byte, params *ScryptParams) (string, error) {
	copied := copyScryptParams(params)
	if err := validateScryptParams(copied); err != nil {
		return "", err
	}

	salt := make([]byte, copied.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}
	key, err := scrypt.Key(password, salt, copied.N, copied.R, copied.P, copied.KeyLength)
	if err != nil {
		return "", fmt.Errorf("scrypt computation failed: %w", err)
	}
	return fmt.Sprintf("$scrypt$N=%d,r=%d,p=%d$%s$%s",
		copied.N, copied.R, copied.P,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

// VerifyScryptHash 验证scrypt哈希
func VerifyScryptHash(hash, password []byte) (bool, error) {
	if len(hash) < minScryptEncodedLength || len(hash) > maxScryptEncodedLength {
		return false, fmt.Errorf("%w: encoded hash length", ErrInvalidScryptParams)
	}
	parts := strings.Split(string(hash), "$")
	if len(parts) != 5 || parts[0] != "" || parts[1] != "scrypt" {
		return false, fmt.Errorf("%w: invalid scrypt hash format", ErrInvalidScryptParams)
	}

	params, err := parseScryptNRP(parts[2])
	if err != nil {
		return false, fmt.Errorf("%w: %w", ErrInvalidScryptParams, err)
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("%w: decode salt: %w", ErrInvalidScryptParams, err)
	}
	key, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("%w: decode hash: %w", ErrInvalidScryptParams, err)
	}
	params.SaltLength = len(salt)
	params.KeyLength = len(key)
	if err := validateScryptParams(params); err != nil {
		return false, err
	}

	computed, err := scrypt.Key(password, salt, params.N, params.R, params.P, params.KeyLength)
	if err != nil {
		return false, fmt.Errorf("scrypt computation failed: %w", err)
	}
	return SecureCompare(key, computed), nil
}

func parseScryptNRP(raw string) (ScryptParams, error) {
	fields := strings.Split(raw, ",")
	if len(fields) != 3 {
		return ScryptParams{}, errors.New("parameter block must have N,r,p")
	}
	n, err := parseExactPrefixedInt64(fields[0], "N=")
	if err != nil {
		return ScryptParams{}, err
	}
	r, err := parseExactPrefixedInt64(fields[1], "r=")
	if err != nil {
		return ScryptParams{}, err
	}
	p, err := parseExactPrefixedInt64(fields[2], "p=")
	if err != nil {
		return ScryptParams{}, err
	}
	return ScryptParams{N: int(n), R: int(r), P: int(p)}, nil
}

func parseExactPrefixedInt64(raw, prefix string) (int64, error) {
	if !strings.HasPrefix(raw, prefix) {
		return 0, fmt.Errorf("missing prefix %q", prefix)
	}
	return strconv.ParseInt(raw[len(prefix):], 10, 64)
}

// ScryptHasher scrypt哈希器
type ScryptHasher struct {
	params ScryptParams
}

// NewScryptHasher 创建scrypt哈希器，保存参数副本。
func NewScryptHasher(params *ScryptParams) (*ScryptHasher, error) {
	copied := copyScryptParams(params)
	if err := validateScryptParams(copied); err != nil {
		return nil, err
	}
	return &ScryptHasher{params: copied}, nil
}

// Hash 使用scrypt哈希密码
func (s *ScryptHasher) Hash(password []byte) (string, error) {
	copied := s.params
	return HashWithScrypt(password, &copied)
}

// Verify 验证scrypt哈希
func (s *ScryptHasher) Verify(hash, password []byte) (bool, error) {
	return VerifyScryptHash(hash, password)
}
