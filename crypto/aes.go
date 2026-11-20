package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// EncodingType 定义编码类型
type EncodingType int

const (
	// EncodingStandard 使用标准 Base64 编码
	EncodingStandard EncodingType = iota
	// EncodingURLSafe 使用对 URL 安全的 Base64 编码
	EncodingURLSafe
)

// EncryptionMode 定义加密模式
type EncryptionMode int

const (
	// ModeGCM AES-GCM 认证加密模式
	ModeGCM EncryptionMode = iota
	// ModeCFB 已不再支持；构造会失败。
	ModeCFB
)

var (
	// ErrUnsupportedEncryptionMode 表示请求了未实现的加密模式。
	ErrUnsupportedEncryptionMode = errors.New("unsupported encryption mode")
	// ErrUnsupportedEncoding 表示未知编码类型。
	ErrUnsupportedEncoding = errors.New("unsupported encoding")
	// ErrInvalidAESKeySize 表示密钥长度不是 16/24/32。
	ErrInvalidAESKeySize = errors.New("invalid key size: must be 16, 24, or 32 bytes")
	// ErrCiphertextTooShort 表示密文短于 nonce。
	ErrCiphertextTooShort = errors.New("ciphertext is too short")
	// ErrUninitializedEncryptor 表示对 nil 或未通过构造函数初始化的 AESEncryptor 调用加解密。
	ErrUninitializedEncryptor = errors.New("AES encryptor is not initialized")
)

// Encryptor 加密器接口。
type Encryptor interface {
	Encrypt(plaintext []byte) (string, error)
	Decrypt(ciphertext string) ([]byte, error)
	EncryptWithOptions(plaintext []byte, encoding EncodingType) (string, error)
	DecryptWithOptions(ciphertext string, encoding EncodingType) ([]byte, error)
}

// AESEncryptor 提供并发安全的 AES-GCM 认证加密。
type AESEncryptor struct {
	gcm cipher.AEAD
}

// NewAESEncryptor 创建 AES-GCM 加密器。密钥必须为 16/24/32 字节且来自 CSPRNG 或 KMS。
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	return NewAESEncryptorWithMode(key, ModeGCM)
}

// NewAESEncryptorWithMode 仅接受 ModeGCM。
func NewAESEncryptorWithMode(key []byte, mode EncryptionMode) (*AESEncryptor, error) {
	if mode != ModeGCM {
		return nil, ErrUnsupportedEncryptionMode
	}
	switch len(key) {
	case 16, 24, 32:
	default:
		return nil, ErrInvalidAESKeySize
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	return &AESEncryptor{gcm: gcm}, nil
}

func (e *AESEncryptor) validate() error {
	if e == nil || e.gcm == nil {
		return ErrUninitializedEncryptor
	}
	return nil
}

func encoderFor(encodingType EncodingType) (*base64.Encoding, error) {
	switch encodingType {
	case EncodingStandard:
		return base64.StdEncoding, nil
	case EncodingURLSafe:
		return base64.URLEncoding, nil
	default:
		return nil, ErrUnsupportedEncoding
	}
}

// EncryptWithAADAndEncoding 使用指定 Base64 编码及附加认证数据（AAD）执行 AES-GCM 加密。
// additionalData 参与完整性校验但不被加密，可用于将密文绑定至用户 ID、记录主键等业务上下文。
func (e *AESEncryptor) EncryptWithAADAndEncoding(plaintext, additionalData []byte, encoding EncodingType) (string, error) {
	if err := e.validate(); err != nil {
		return "", err
	}
	enc, err := encoderFor(encoding)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, additionalData)
	return enc.EncodeToString(ciphertext), nil
}

// EncryptWithAAD 使用标准 Base64 编码及附加认证数据（AAD）执行 AES-GCM 加密。
func (e *AESEncryptor) EncryptWithAAD(plaintext, additionalData []byte) (string, error) {
	return e.EncryptWithAADAndEncoding(plaintext, additionalData, EncodingStandard)
}

// EncryptWithOptions 使用指定编码和 AES-GCM 加密。
func (e *AESEncryptor) EncryptWithOptions(plaintext []byte, encoding EncodingType) (string, error) {
	return e.EncryptWithAADAndEncoding(plaintext, nil, encoding)
}

// Encrypt 使用标准 Base64 编码加密。
func (e *AESEncryptor) Encrypt(plaintext []byte) (string, error) {
	return e.EncryptWithAADAndEncoding(plaintext, nil, EncodingStandard)
}

// DecryptWithAADAndEncoding 使用指定编码及附加认证数据（AAD）执行 AES-GCM 解密与完整性验证。
func (e *AESEncryptor) DecryptWithAADAndEncoding(ciphertext string, additionalData []byte, encoding EncodingType) ([]byte, error) {
	if err := e.validate(); err != nil {
		return nil, err
	}
	enc, err := encoderFor(encoding)
	if err != nil {
		return nil, err
	}
	data, err := enc.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}
	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize+e.gcm.Overhead() {
		return nil, ErrCiphertextTooShort
	}
	plaintext, err := e.gcm.Open(nil, data[:nonceSize], data[nonceSize:], additionalData)
	if err != nil {
		return nil, fmt.Errorf("open GCM: %w", err)
	}
	return plaintext, nil
}

// DecryptWithAAD 使用标准 Base64 编码及附加认证数据（AAD）执行 AES-GCM 解密。
func (e *AESEncryptor) DecryptWithAAD(ciphertext string, additionalData []byte) ([]byte, error) {
	return e.DecryptWithAADAndEncoding(ciphertext, additionalData, EncodingStandard)
}

// DecryptWithOptions 使用指定编码和 AES-GCM 解密。
func (e *AESEncryptor) DecryptWithOptions(ciphertext string, encoding EncodingType) ([]byte, error) {
	return e.DecryptWithAADAndEncoding(ciphertext, nil, encoding)
}

// Decrypt 使用标准 Base64 编码解密。
func (e *AESEncryptor) Decrypt(ciphertext string) ([]byte, error) {
	return e.DecryptWithAADAndEncoding(ciphertext, nil, EncodingStandard)
}
