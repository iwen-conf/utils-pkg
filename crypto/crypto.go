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
	"strings"
	"sync"
	"unicode"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/scrypt"
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
		return "", fmt.Errorf("生成salt失败: %w", err)
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
		return false, errors.New("无效的Argon2哈希格式")
	}

	// 解析参数
	var argonType Argon2Type = Argon2id
	if parts[1] == "argon2i" {
		argonType = Argon2i
	}

	var memory, iterations, parallelism uint32
	var salt, key []byte
	var version int

	_, err := fmt.Sscanf(parts[2], "v=%d", &version)
	if err != nil || version != 19 {
		return false, errors.New("不支持的Argon2版本")
	}

	_, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &iterations, &parallelism)
	if err != nil {
		return false, fmt.Errorf("解析参数失败: %w", err)
	}

	// 解码salt和hash
	salt, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("解码salt失败: %w", err)
	}

	key, err = base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("解码hash失败: %w", err)
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

// ScryptParams scrypt参数配置
type ScryptParams struct {
	// CPU/内存成本参数
	N int
	// 块大小参数
	R int
	// 并行化参数
	P int
	// Salt长度
	SaltLength int
	// Key长度
	KeyLength int
}

// DefaultScryptParams 返回推荐的scrypt参数
func DefaultScryptParams() *ScryptParams {
	return &ScryptParams{
		N:          32768, // 2^15
		R:          8,     // 块大小
		P:          1,     // 并行化
		SaltLength: 16,    // salt长度
		KeyLength:  32,    // 输出key长度
	}
}

// FastScryptParams 返回快速但安全的scrypt参数
func FastScryptParams() *ScryptParams {
	return &ScryptParams{
		N:          16384, // 2^14
		R:          8,
		P:          1,
		SaltLength: 16,
		KeyLength:  32,
	}
}

// HashWithScrypt 使用scrypt算法哈希密码
func HashWithScrypt(password []byte, params *ScryptParams) (string, error) {
	if params == nil {
		params = DefaultScryptParams()
	}

	// 生成随机salt
	salt := make([]byte, params.SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("生成salt失败: %w", err)
	}

	// 使用scrypt生成key
	key, err := scrypt.Key(password, salt, params.N, params.R, params.P, params.KeyLength)
	if err != nil {
		return "", fmt.Errorf("scrypt计算失败: %w", err)
	}

	// 编码格式: $scrypt$N={n},r={r},p={p}${salt}${hash}
	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedKey := base64.RawStdEncoding.EncodeToString(key)

	return fmt.Sprintf("$scrypt$N=%d,r=%d,p=%d$%s$%s",
		params.N, params.R, params.P, encodedSalt, encodedKey), nil
}

// VerifyScryptHash 验证scrypt哈希
func VerifyScryptHash(hash, password []byte) (bool, error) {
	// 解析哈希字符串
	parts := strings.Split(string(hash), "$")
	if len(parts) != 5 || parts[1] != "scrypt" {
		return false, errors.New("无效的scrypt哈希格式")
	}

	// 解析参数
	var N, R, P int
	var salt, key []byte

	_, err := fmt.Sscanf(parts[2], "N=%d,r=%d,p=%d", &N, &R, &P)
	if err != nil {
		return false, fmt.Errorf("解析参数失败: %w", err)
	}

	// 解码salt和hash
	salt, err = base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("解码salt失败: %w", err)
	}

	key, err = base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("解码hash失败: %w", err)
	}

	// 使用相同参数重新计算哈希
	computedKey, err := scrypt.Key(password, salt, N, R, P, len(key))
	if err != nil {
		return false, fmt.Errorf("scrypt计算失败: %w", err)
	}

	// 安全比较
	return SecureCompare(key, computedKey), nil
}

// PasswordHasher 密码哈希器接口
type PasswordHasher interface {
	Hash(password []byte) (string, error)
	Verify(hash, password []byte) (bool, error)
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

// ScryptHasher scrypt哈希器
type ScryptHasher struct {
	params *ScryptParams
}

// NewScryptHasher 创建scrypt哈希器
func NewScryptHasher(params *ScryptParams) *ScryptHasher {
	if params == nil {
		params = DefaultScryptParams()
	}
	return &ScryptHasher{params: params}
}

// Hash 使用scrypt哈希密码
func (s *ScryptHasher) Hash(password []byte) (string, error) {
	return HashWithScrypt(password, s.params)
}

// Verify 验证scrypt哈希
func (s *ScryptHasher) Verify(hash, password []byte) (bool, error) {
	return VerifyScryptHash(hash, password)
}
