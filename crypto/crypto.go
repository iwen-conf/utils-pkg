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
	"sync"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

var (
	// 预编译正则表达式
	nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]`)

	// 对象池，用于加密和解密操作的缓冲区
	bufferPool = sync.Pool{
		New: func() interface{} {
			// 默认分配 1KB 缓冲区
			buf := make([]byte, 0, 1024)
			return &buf
		},
	}
)

// EncryptionMode 定义加密模式
type EncryptionMode int

const (
	// ModeCFB 使用 CFB 模式
	ModeCFB EncryptionMode = iota
	// ModeGCM 使用 GCM 模式（更高性能）
	ModeGCM
)

// EncodingType 定义编码类型
type EncodingType int

const (
	// EncodingStandard 使用标准 Base64 编码
	EncodingStandard EncodingType = iota
	// EncodingURLSafe 使用对 URL 安全的 Base64 编码
	EncodingURLSafe
)

// Encryptor 加密器接口
type Encryptor interface {
	Encrypt(plaintext []byte) (string, error)
	Decrypt(ciphertext string) ([]byte, error)
	EncryptWithOptions(plaintext []byte, encoding EncodingType) (string, error)
	DecryptWithOptions(ciphertext string, encoding EncodingType) ([]byte, error)
}

// AESEncryptor AES 加密实现
type AESEncryptor struct {
	key        []byte
	block      cipher.Block
	mode       EncryptionMode
	blockMutex sync.RWMutex
}

// NewAESEncryptor 创建新的 AES 加密器
func NewAESEncryptor(key []byte) (*AESEncryptor, error) {
	return NewAESEncryptorWithMode(key, ModeCFB)
}

// NewAESEncryptorWithMode 创建指定模式的 AES 加密器
func NewAESEncryptorWithMode(key []byte, mode EncryptionMode) (*AESEncryptor, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, errors.New("invalid key size: must be 16, 24, or 32 bytes")
	}

	// 预先创建 block
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &AESEncryptor{
		key:   key,
		block: block,
		mode:  mode,
	}, nil
}

// getBlock 获取缓存的加密块
func (e *AESEncryptor) getBlock() (cipher.Block, error) {
	e.blockMutex.RLock()
	block := e.block
	e.blockMutex.RUnlock()

	if block != nil {
		return block, nil
	}

	// 如果 block 为 nil（不应该发生，但以防万一）
	e.blockMutex.Lock()
	defer e.blockMutex.Unlock()

	// 再次检查，避免并发创建
	if e.block != nil {
		return e.block, nil
	}

	var err error
	e.block, err = aes.NewCipher(e.key)
	return e.block, err
}

// getEncoder 根据编码类型获取编码器
func getEncoder(encodingType EncodingType) *base64.Encoding {
	if encodingType == EncodingURLSafe {
		return base64.URLEncoding
	}
	return base64.StdEncoding
}

// EncryptWithOptions 使用指定的编码方式加密数据
func (e *AESEncryptor) EncryptWithOptions(plaintext []byte, encoding EncodingType) (string, error) {
	block, err := e.getBlock()
	if err != nil {
		return "", err
	}

	// 根据模式使用不同的加密方法
	switch e.mode {
	case ModeGCM:
		return e.encryptGCM(block, plaintext, encoding)
	default: // ModeCFB
		return e.encryptCFB(block, plaintext, encoding)
	}
}

// Encrypt 加密数据
func (e *AESEncryptor) Encrypt(plaintext []byte) (string, error) {
	return e.EncryptWithOptions(plaintext, EncodingStandard)
}

// encryptCFB 使用 CFB 模式加密
func (e *AESEncryptor) encryptCFB(block cipher.Block, plaintext []byte, encoding EncodingType) (string, error) {
	// 获取临时缓冲区
	bufPtr := bufferPool.Get().(*[]byte)
	buf := *bufPtr

	// 确保缓冲区有足够的容量
	requiredSize := aes.BlockSize + len(plaintext)
	if cap(buf) < requiredSize {
		buf = make([]byte, 0, requiredSize)
	}

	// 重置长度
	buf = buf[:requiredSize]

	// 创建随机 IV
	iv := buf[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		bufferPool.Put(bufPtr) // 释放缓冲区
		return "", err
	}

	// 加密数据
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(buf[aes.BlockSize:], plaintext)

	// 编码结果
	result := getEncoder(encoding).EncodeToString(buf)

	// 返回缓冲区到池
	bufferPool.Put(bufPtr)

	return result, nil
}

// encryptGCM 使用 GCM 模式加密（更安全和更快）
func (e *AESEncryptor) encryptGCM(block cipher.Block, plaintext []byte, encoding EncodingType) (string, error) {
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 创建随机 nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 获取临时缓冲区
	bufPtr := bufferPool.Get().(*[]byte)
	buf := *bufPtr

	// 确保缓冲区有足够的容量
	requiredSize := len(nonce) + len(plaintext) + aesGCM.Overhead()
	if cap(buf) < requiredSize {
		buf = make([]byte, 0, requiredSize)
	}

	// 重置长度并预分配容量
	buf = buf[:len(nonce)]
	copy(buf, nonce)

	// 使用 GCM 加密并认证
	buf = aesGCM.Seal(buf, nonce, plaintext, nil)

	// 编码结果
	result := getEncoder(encoding).EncodeToString(buf)

	// 返回缓冲区到池
	bufferPool.Put(bufPtr)

	return result, nil
}

// DecryptWithOptions 使用指定的编码方式解密数据
func (e *AESEncryptor) DecryptWithOptions(ciphertext string, encoding EncodingType) ([]byte, error) {
	data, err := getEncoder(encoding).DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}

	block, err := e.getBlock()
	if err != nil {
		return nil, err
	}

	// 根据模式使用不同的解密方法
	switch e.mode {
	case ModeGCM:
		return e.decryptGCM(block, data)
	default: // ModeCFB
		return e.decryptCFB(block, data)
	}
}

// Decrypt 解密数据
func (e *AESEncryptor) Decrypt(ciphertext string) ([]byte, error) {
	return e.DecryptWithOptions(ciphertext, EncodingStandard)
}

// decryptCFB 使用 CFB 模式解密
func (e *AESEncryptor) decryptCFB(block cipher.Block, data []byte) ([]byte, error) {
	if len(data) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]

	// 获取临时缓冲区
	bufPtr := bufferPool.Get().(*[]byte)
	buf := *bufPtr

	// 确保缓冲区有足够的容量
	if cap(buf) < len(data) {
		buf = make([]byte, len(data))
	} else {
		buf = buf[:len(data)]
	}

	// 拷贝数据，避免原地解密导致的问题
	copy(buf, data)

	// 解密数据
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(buf, buf)

	// 创建结果
	result := make([]byte, len(buf))
	copy(result, buf)

	// 返回缓冲区到池
	bufferPool.Put(bufPtr)

	return result, nil
}

// decryptGCM 使用 GCM 模式解密
func (e *AESEncryptor) decryptGCM(block cipher.Block, data []byte) ([]byte, error) {
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(data) < aesGCM.NonceSize() {
		return nil, errors.New("ciphertext too short")
	}

	nonce := data[:aesGCM.NonceSize()]
	ciphertext := data[aesGCM.NonceSize():]

	return aesGCM.Open(nil, nonce, ciphertext, nil)
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
	// 添加预编译正则表达式缓存
	disallowRegexes []*regexp.Regexp
	regexMutex      sync.RWMutex
}

// NewDefaultPasswordPolicy 创建默认密码策略
func NewDefaultPasswordPolicy() *PasswordPolicy {
	policy := &PasswordPolicy{
		MinLength:      8,
		MaxLength:      32,
		RequireUpper:   true,
		RequireLower:   true,
		RequireNumber:  true,
		RequireSpecial: true,
		DisallowWords:  []string{"password", "123456", "qwerty"},
	}

	// 预编译禁用词正则表达式
	policy.compileRegexes()

	return policy
}

// compileRegexes 预编译禁用词正则表达式
func (p *PasswordPolicy) compileRegexes() {
	p.regexMutex.Lock()
	defer p.regexMutex.Unlock()

	p.disallowRegexes = make([]*regexp.Regexp, len(p.DisallowWords))
	for i, word := range p.DisallowWords {
		p.disallowRegexes[i] = regexp.MustCompile(`(?i)` + regexp.QuoteMeta(word))
	}
}

// SetDisallowWords 设置禁用词并重新编译正则表达式
func (p *PasswordPolicy) SetDisallowWords(words []string) {
	p.regexMutex.Lock()
	p.DisallowWords = words
	p.regexMutex.Unlock()

	p.compileRegexes()
}

// ValidatePassword 验证密码是否符合策略
func (p *PasswordPolicy) ValidatePassword(password string) error {
	// 检查长度
	if len(password) < p.MinLength {
		return fmt.Errorf("密码长度不能小于 %d 个字符", p.MinLength)
	}
	if len(password) > p.MaxLength {
		return fmt.Errorf("密码长度不能大于 %d 个字符", p.MaxLength)
	}

	// 使用一次遍历检查所有字符类型
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

		// 如果所有必需条件都满足，提前退出循环
		if (!p.RequireUpper || hasUpper) &&
			(!p.RequireLower || hasLower) &&
			(!p.RequireNumber || hasNumber) &&
			(!p.RequireSpecial || hasSpecial) {
			break
		}
	}

	// 检查字符类型
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
	p.regexMutex.RLock()
	regexes := p.disallowRegexes
	p.regexMutex.RUnlock()

	// 去除所有非字母数字字符
	passwordLower := nonAlphanumericRegex.ReplaceAllString(password, "")

	// 使用预编译的正则表达式
	for i, regex := range regexes {
		if regex.MatchString(passwordLower) {
			return fmt.Errorf("密码不能包含常见词汇: %s", p.DisallowWords[i])
		}
	}

	return nil
}

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
