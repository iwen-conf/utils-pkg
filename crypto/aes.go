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
	// ModeGCM AES-GCM 模式
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
)

// Encryptor 加密器接口。
type Encryptor interface {
	Encrypt(plaintext []byte) (string, error)
	Decrypt(ciphertext string) ([]byte, error)
	EncryptWithOptions(plaintext []byte, encoding EncodingType) (string, error)
	DecryptWithOptions(ciphertext string, encoding EncodingType) ([]byte, error)
}

// AESEncryptor 只提供 AES-GCM 认证加密。
type AESEncryptor struct {
	block cipher.Block
}

// NewAESEncryptor 创建 AES-GCM 加密器。密钥必须来自 CSPRNG 或 KMS。
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
	return &AESEncryptor{block: block}, nil
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

// EncryptWithOptions 使用指定编码和 AES-GCM 加密。
func (e *AESEncryptor) EncryptWithOptions(plaintext []byte, encoding EncodingType) (string, error) {
	enc, err := encoderFor(encoding)
	if err != nil {
		return "", err
	}
	aesGCM, err := cipher.NewGCM(e.block)
	if err != nil {
		return "", fmt.Errorf("create GCM: %w", err)
	}
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)
	return enc.EncodeToString(ciphertext), nil
}

// Encrypt 使用标准 Base64 编码加密。
func (e *AESEncryptor) Encrypt(plaintext []byte) (string, error) {
	return e.EncryptWithOptions(plaintext, EncodingStandard)
}

// DecryptWithOptions 使用指定编码和 AES-GCM 解密。
func (e *AESEncryptor) DecryptWithOptions(ciphertext string, encoding EncodingType) ([]byte, error) {
	enc, err := encoderFor(encoding)
	if err != nil {
		return nil, err
	}
	data, err := enc.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}
	aesGCM, err := cipher.NewGCM(e.block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}
	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrCiphertextTooShort
	}
	plaintext, err := aesGCM.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("open GCM: %w", err)
	}
	return plaintext, nil
}

// Decrypt 使用标准 Base64 编码解密。
func (e *AESEncryptor) Decrypt(ciphertext string) ([]byte, error) {
	return e.DecryptWithOptions(ciphertext, EncodingStandard)
}
