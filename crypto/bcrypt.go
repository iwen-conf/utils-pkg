package crypto

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MaxRandomBytesLength 是 GenerateRandomBytes 允许的最大长度。
	MaxRandomBytesLength = 1 << 20
)

var (
	// ErrInvalidRandomLength 表示随机字节长度非法。
	ErrInvalidRandomLength = errors.New("invalid random bytes length")
	// ErrInvalidBcryptCost 表示 bcrypt 成本参数超出标准允许范围。
	ErrInvalidBcryptCost = errors.New("invalid bcrypt cost")
	// ErrUninitializedHasher 表示对 nil 哈希器调用方法。
	ErrUninitializedHasher = errors.New("password hasher is not initialized")
)

// BcryptCost bcrypt 成本级别
type BcryptCost int

const (
	// BcryptCostLow 最小计算成本（仅用于测试，严禁用于生产环境密码存储）
	BcryptCostLow BcryptCost = BcryptCost(bcrypt.MinCost)
	// BcryptCostDefault 默认推荐成本
	BcryptCostDefault BcryptCost = BcryptCost(bcrypt.DefaultCost)
	// BcryptCostHigh 高计算成本，适合高安全等级场景
	BcryptCostHigh BcryptCost = 14
)

func validateBcryptCost(cost BcryptCost) error {
	if int(cost) < bcrypt.MinCost || int(cost) > bcrypt.MaxCost {
		return fmt.Errorf("%w: must be %d-%d, got %d", ErrInvalidBcryptCost, bcrypt.MinCost, bcrypt.MaxCost, cost)
	}
	return nil
}

// HashPassword 使用 bcrypt 算法对密码进行哈希（默认成本 10；注意 bcrypt 限制密码不超过 72 字节）。
func HashPassword(password []byte) ([]byte, error) {
	return HashPasswordWithCost(password, BcryptCostDefault)
}

// HashPasswordWithCost 使用 bcrypt 算法和指定成本对密码进行哈希。
func HashPasswordWithCost(password []byte, cost BcryptCost) ([]byte, error) {
	if err := validateBcryptCost(cost); err != nil {
		return nil, err
	}
	return bcrypt.GenerateFromPassword(password, int(cost))
}

// CompareHashAndPassword 安全地比较密码和其 bcrypt 哈希值。
func CompareHashAndPassword(hashedPassword, password []byte) error {
	return bcrypt.CompareHashAndPassword(hashedPassword, password)
}

// GenerateRandomBytes 使用 CSPRNG 生成指定长度的随机字节（1 ~ 1MiB）。
func GenerateRandomBytes(length int) ([]byte, error) {
	if length <= 0 || length > MaxRandomBytesLength {
		return nil, fmt.Errorf("%w: must be 1-%d", ErrInvalidRandomLength, MaxRandomBytesLength)
	}
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("read random bytes: %w", err)
	}
	return b, nil
}

// BcryptHasher bcrypt 哈希器
type BcryptHasher struct {
	cost BcryptCost
}

// NewBcryptHasher 创建并校验成本参数的 bcrypt 哈希器。
func NewBcryptHasher(cost BcryptCost) (*BcryptHasher, error) {
	if err := validateBcryptCost(cost); err != nil {
		return nil, err
	}
	return &BcryptHasher{cost: cost}, nil
}

// Hash 使用 bcrypt 哈希密码
func (b *BcryptHasher) Hash(password []byte) (string, error) {
	if b == nil {
		return "", ErrUninitializedHasher
	}
	hashed, err := HashPasswordWithCost(password, b.cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// Verify 验证 bcrypt 哈希
func (b *BcryptHasher) Verify(hash, password []byte) (bool, error) {
	if b == nil {
		return false, ErrUninitializedHasher
	}
	err := CompareHashAndPassword(hash, password)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return false, err
}
