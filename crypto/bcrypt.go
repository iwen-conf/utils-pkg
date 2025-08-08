package crypto

import (
	"crypto/rand"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// BcryptCost bcrypt 成本级别
type BcryptCost int

const (
	// BcryptCostLow 低计算成本，适合高性能需求场景
	BcryptCostLow BcryptCost = 4
	// BcryptCostDefault 默认成本
	BcryptCostDefault BcryptCost = 10
	// BcryptCostHigh 高计算成本，适合高安全性需求场景
	BcryptCostHigh BcryptCost = 14
)

// HashPassword 使用 bcrypt 算法对密码进行加密，使用默认成本
func HashPassword(password []byte) ([]byte, error) {
	return HashPasswordWithCost(password, BcryptCostDefault)
}

// HashPasswordWithCost 使用 bcrypt 算法和指定成本对密码进行加密
func HashPasswordWithCost(password []byte, cost BcryptCost) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, int(cost))
}

// CompareHashAndPassword 安全地比较密码和其哈希值
func CompareHashAndPassword(hashedPassword, password []byte) error {
	return bcrypt.CompareHashAndPassword(hashedPassword, password)
}

// GenerateRandomBytes 生成指定长度的随机字节
func GenerateRandomBytes(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// BcryptHasher bcrypt哈希器
type BcryptHasher struct {
	cost BcryptCost
}

// NewBcryptHasher 创建bcrypt哈希器
func NewBcryptHasher(cost BcryptCost) *BcryptHasher {
	return &BcryptHasher{cost: cost}
}

// Hash 使用bcrypt哈希密码
func (b *BcryptHasher) Hash(password []byte) (string, error) {
	hashed, err := HashPasswordWithCost(password, b.cost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// Verify 验证bcrypt哈希
func (b *BcryptHasher) Verify(hash, password []byte) (bool, error) {
	err := CompareHashAndPassword(hash, password)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return false, err
}