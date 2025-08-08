package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"sync"
)

// EncodingType 定义编码类型
type EncodingType int

const (
	// EncodingStandard 使用标准 Base64 编码
	EncodingStandard EncodingType = iota
	// EncodingURLSafe 使用对 URL 安全的 Base64 编码
	EncodingURLSafe
)

// Encryptor 加密器接口。
// 注意：此实现强制使用 AES-GCM 进行认证加密。
type Encryptor interface {
	Encrypt(plaintext []byte) (string, error)
	Decrypt(ciphertext string) ([]byte, error)
	EncryptWithOptions(plaintext []byte, encoding EncodingType) (string, error)
	DecryptWithOptions(ciphertext string, encoding EncodingType) ([]byte, error)
}

// AESEncryptor 提供使用 AES-GCM 的加密实现。
// AES-GCM 是一种认证加密模式，能同时提供保密性、完整性和真实性，是现代应用的首选。
type AESEncryptor struct {
	key        []byte
	block      cipher.Block
	blockMutex sync.RWMutex
}

// NewAESEncryptor 创建新的 AES-GCM 加密器。
// key 的长度必须是 16, 24, 或 32 字节，分别对应 AES-128, AES-192, AES-256。
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	keySize := len(key)
	if keySize != 16 && keySize != 24 && keySize != 32 {
		return nil, errors.New("invalid key size: must be 16, 24, or 32 bytes")
	}

	// 预先创建 cipher.Block 以提高效率。
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &AESEncryptor{
		key:   key,
		block: block,
	}, nil
}

// getBlock 获取缓存的加密块。
func (e *AESEncryptor) getBlock() (cipher.Block, error) {
	e.blockMutex.RLock()
	// block 在构造函数中已初始化，不应为 nil。
	block := e.block
	e.blockMutex.RUnlock()
	return block, nil
}

// getEncoder 根据编码类型获取编码器。
func getEncoder(encodingType EncodingType) *base64.Encoding {
	if encodingType == EncodingURLSafe {
		return base64.URLEncoding
	}
	return base64.StdEncoding
}

// EncryptWithOptions 使用指定的编码方式和 AES-GCM 模式加密数据。
func (e *AESEncryptor) EncryptWithOptions(plaintext []byte, encoding EncodingType) (string, error) {
	block, err := e.getBlock()
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成一个随机的 Nonce。对于同一个密钥，每次加密的 Nonce 都必须是唯一的。
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 从池中获取一个临时缓冲区。
	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr) // 确保缓冲区在使用后归还。

	// GCM 的输出是 nonce || ciphertext || tag
	// Seal 函数会将密文和认证标签追加到其第一个参数（dst）中。
	// 我们将 nonce 作为 dst 的初始内容，Seal 会在它后面追加数据。
	ciphertext := aesGCM.Seal(nonce, nonce, plaintext, nil)

	// 将完整的结果（nonce + 密文 + 认证标签）编码为 Base64 字符串。
	return getEncoder(encoding).EncodeToString(ciphertext), nil
}

// Encrypt 使用标准 Base64 编码加密数据。
func (e *AESEncryptor) Encrypt(plaintext []byte) (string, error) {
	return e.EncryptWithOptions(plaintext, EncodingStandard)
}

// DecryptWithOptions 使用指定的编码方式和 AES-GCM 模式解密数据。
func (e *AESEncryptor) DecryptWithOptions(ciphertext string, encoding EncodingType) ([]byte, error) {
	data, err := getEncoder(encoding).DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}

	block, err := e.getBlock()
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext is too short")
	}

	// 从数据开头提取 Nonce。
	nonce, encryptedData := data[:nonceSize], data[nonceSize:]

	// 解密并验证认证标签。如果标签无效，Open 会返回错误。
	plaintext, err := aesGCM.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// Decrypt 使用标准 Base64 编码解密数据。
func (e *AESEncryptor) Decrypt(ciphertext string) ([]byte, error) {
	return e.DecryptWithOptions(ciphertext, EncodingStandard)
}
