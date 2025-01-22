package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"regexp"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// Encryptor 加密器接口
type Encryptor interface {
	Encrypt(plaintext []byte) (string, error)
	Decrypt(ciphertext string) ([]byte, error)
}

// AESEncryptor AES 加密实现
type AESEncryptor struct {
	key []byte
}

// NewAESEncryptor 创建新的 AES 加密器
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("invalid key size: must be 16, 24, or 32 bytes")
	}
	return &AESEncryptor{key: key}, nil
}

// Encrypt 加密数据
func (e *AESEncryptor) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", err
	}

	// 创建随机 IV
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], plaintext)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密数据
func (e *AESEncryptor) Decrypt(ciphertext string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	if len(data) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(data, data)

	return data, nil
}

// HashSHA256 计算 SHA256 哈希
func HashSHA256(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// HashSHA512 计算 SHA512 哈希
func HashSHA512(data []byte) []byte {
	hash := sha512.Sum512(data)
	return hash[:]
}

// HashMD5 计算 MD5 哈希（不推荐用于安全场景）
func HashMD5(data []byte) []byte {
	hash := md5.Sum(data)
	return hash[:]
}

// PasswordPolicy 密码策略结构体
type PasswordPolicy struct {
	MinLength      int
	MaxLength      int
	RequireUpper   bool
	RequireLower   bool
	RequireNumber  bool
	RequireSpecial bool
	DisallowWords  []string
}

// NewDefaultPasswordPolicy 创建默认密码策略
func NewDefaultPasswordPolicy() *PasswordPolicy {
	return &PasswordPolicy{
		MinLength:      8,
		MaxLength:      32,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumber:  true,
		RequireSpecial: true,
		DisallowWords:  []string{"password", "123456", "qwerty"},
	}
}

// ValidatePassword 验证密码是否符合策略
func (p *PasswordPolicy) ValidatePassword(password string) error {
	if len(password) < p.MinLength {
		return fmt.Errorf("密码长度不能小于 %d 个字符", p.MinLength)
	}
	if len(password) > p.MaxLength {
		return fmt.Errorf("密码长度不能大于 %d 个字符", p.MaxLength)
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if p.RequireUpper && !hasUpper {
		return errors.New("密码必须包含大写字母")
	}
	if p.RequireLower && !hasLower {
		return errors.New("密码必须包含小写字母")
	}
	if p.RequireNumber && !hasNumber {
		return errors.New("密码必须包含数字")
	}
	if p.RequireSpecial && !hasSpecial {
		return errors.New("密码必须包含特殊字符")
	}

	// 检查禁用词
	passwordLower := regexp.MustCompile(`[^a-zA-Z0-9]`).ReplaceAllString(password, "")
	for _, word := range p.DisallowWords {
		if regexp.MustCompile(`(?i)` + word).MatchString(passwordLower) {
			return fmt.Errorf("密码不能包含常见词汇: %s", word)
		}
	}

	return nil
}

// HashPassword 使用 bcrypt 算法对密码进行加密
func HashPassword(password []byte) ([]byte, error) {
	return bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
}

// CompareHashAndPassword 安全地比较密码和其哈希值
func CompareHashAndPassword(hashedPassword, password []byte) error {
	return bcrypt.CompareHashAndPassword(hashedPassword, password)
}

// SecureCompare 使用恒定时间比较两个字节切片
func SecureCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
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
