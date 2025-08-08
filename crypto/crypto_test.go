package crypto

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"testing"
)

func TestNewAESEncryptor(t *testing.T) {
	// 测试有效密钥长度
	validKeys := []int{16, 24, 32}
	for _, length := range validKeys {
		key := make([]byte, length)
		encryptor, err := NewAESEncryptor(key)
		if err != nil {
			t.Errorf("Failed to create encryptor with key length %d: %v", length, err)
		}
		if encryptor == nil {
			t.Errorf("Encryptor is nil for key length %d", length)
		}
	}

	// 测试无效密钥长度
	invalidKey := make([]byte, 20)
	_, err := NewAESEncryptor(invalidKey)
	if err == nil {
		t.Error("Expected error for invalid key size")
	}
}

func TestNewAESEncryptorWithMode(t *testing.T) {
	// 测试不同的加密模式
	modes := []EncryptionMode{ModeCFB, ModeGCM}
	for _, mode := range modes {
		key := make([]byte, 32)
		encryptor, err := NewAESEncryptorWithMode(key, mode)
		if err != nil {
			t.Errorf("Failed to create encryptor with mode %d: %v", mode, err)
		}
		if encryptor == nil {
			t.Errorf("Encryptor is nil for mode %d", mode)
		}
		if encryptor.mode != mode {
			t.Errorf("Expected mode %d but got %d", mode, encryptor.mode)
		}
	}
}

func TestAESEncryptor_EncryptDecrypt(t *testing.T) {
	key := make([]byte, 32)
	// 使用GCM模式（推荐）
	encryptor, _ := NewAESEncryptorWithMode(key, ModeGCM)

	// 测试加密解密
	plaintext := []byte("Hello, World!")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	// 验证密文不等于明文
	if base64.StdEncoding.EncodeToString(plaintext) == ciphertext {
		t.Error("Ciphertext should not equal plaintext")
	}

	// 测试解密
	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	// 验证解密后的文本等于原文
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("Decrypted text does not match original")
	}

	// 测试无效密文
	_, err = encryptor.Decrypt("invalid-base64")
	if err == nil {
		t.Error("Expected error for invalid base64")
	}
}

func TestAESEncryptor_GCM(t *testing.T) {
	key := make([]byte, 32)
	encryptor, _ := NewAESEncryptorWithMode(key, ModeGCM)

	// 测试 GCM 模式加密解密
	plaintext := []byte("Hello, World with GCM mode!")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("GCM encryption failed: %v", err)
	}

	// 测试 GCM 解密
	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("GCM decryption failed: %v", err)
	}

	// 验证解密后的文本等于原文
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("GCM decrypted text does not match original")
	}
}

func TestAESEncryptor_CFB_Deprecated(t *testing.T) {
	key := make([]byte, 32)
	// 测试已弃用的CFB模式以确保向后兼容
	encryptor, _ := NewAESEncryptorWithMode(key, ModeCFB)

	// 测试 CFB 模式加密解密
	plaintext := []byte("Hello, World with deprecated CFB mode!")
	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("CFB encryption failed: %v", err)
	}

	// 测试 CFB 解密
	decrypted, err := encryptor.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("CFB decryption failed: %v", err)
	}

	// 验证解密后的文本等于原文
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("CFB decrypted text does not match original")
	}
}

func TestAESEncryptor_URLSafeEncoding(t *testing.T) {
	key := make([]byte, 32)
	// 使用GCM模式（推荐）
	encryptor, _ := NewAESEncryptorWithMode(key, ModeGCM)

	// 测试 URL 安全编码
	plaintext := []byte("Hello, URL-safe encoding!")
	ciphertext, err := encryptor.EncryptWithOptions(plaintext, EncodingURLSafe)
	if err != nil {
		t.Fatalf("URL-safe encryption failed: %v", err)
	}

	// 验证使用 URL 安全编码
	for _, c := range ciphertext {
		if c == '+' || c == '/' {
			t.Error("URL-safe encoding should not contain '+' or '/'")
			break
		}
	}

	// 测试 URL 安全解密
	decrypted, err := encryptor.DecryptWithOptions(ciphertext, EncodingURLSafe)
	if err != nil {
		t.Fatalf("URL-safe decryption failed: %v", err)
	}

	// 验证解密后的文本等于原文
	if !bytes.Equal(decrypted, plaintext) {
		t.Error("URL-safe decrypted text does not match original")
	}
}

func TestHashFunctions(t *testing.T) {
	data := []byte("test data")

	// 测试 SHA256
	sha256Hash := HashSHA256(data)
	if len(sha256Hash) != 32 { // SHA256 produces 32 bytes
		t.Error("SHA256 hash length incorrect")
	}

	// 测试 SHA512
	sha512Hash := HashSHA512(data)
	if len(sha512Hash) != 64 { // SHA512 produces 64 bytes
		t.Error("SHA512 hash length incorrect")
	}

	// 测试 MD5
	md5Hash := HashMD5(data)
	if len(md5Hash) != 16 { // MD5 produces 16 bytes
		t.Error("MD5 hash length incorrect")
	}

	// 验证相同输入产生相同哈希
	if !bytes.Equal(HashSHA256(data), HashSHA256(data)) {
		t.Error("SHA256 hash not consistent")
	}
}

func TestPasswordPolicy(t *testing.T) {
	policy := NewDefaultPasswordPolicy()

	// 测试有效密码
	validPassword := "Test123!@#"
	if err := policy.ValidatePassword(validPassword); err != nil {
		t.Errorf("Valid password rejected: %v", err)
	}

	// 测试密码长度
	shortPassword := "Abc123!"
	if err := policy.ValidatePassword(shortPassword); err == nil {
		t.Error("Short password should be rejected")
	}

	// 测试缺少必需字符
	testCases := []struct {
		password string
		name     string
	}{
		{"abcdefgh123!", "无大写字母"},
		{"ABCDEFGH123!", "无小写字母"},
		{"Abcdefghijk!", "无数字"},
		{"Abcdefgh123", "无特殊字符"},
	}

	for _, tc := range testCases {
		if err := policy.ValidatePassword(tc.password); err == nil {
			t.Errorf("密码应该被拒绝（%s）: %s", tc.name, tc.password)
		}
	}

	// 测试禁用词
	passwordWithDisallowedWord := "Password123!"
	if err := policy.ValidatePassword(passwordWithDisallowedWord); err == nil {
		t.Error("包含禁用词的密码应该被拒绝")
	}

	// 测试设置新的禁用词
	customWords := []string{"testword", "example"}
	policy.SetDisallowWords(customWords)

	passwordWithCustomDisallowedWord := "Testword123!"
	if err := policy.ValidatePassword(passwordWithCustomDisallowedWord); err == nil {
		t.Error("包含自定义禁用词的密码应该被拒绝")
	}

	passwordWithOldDisallowedWord := "Password123!"
	if err := policy.ValidatePassword(passwordWithOldDisallowedWord); err != nil {
		t.Error("旧禁用词应该已被移除")
	}
}

func TestPasswordHashing(t *testing.T) {
	password := []byte("test-password")

	// 测试密码哈希
	hashedPassword, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Password hashing failed: %v", err)
	}

	// 验证哈希密码不等于原始密码
	if bytes.Equal(hashedPassword, password) {
		t.Error("Hashed password should not equal original password")
	}

	// 测试密码验证
	if err := CompareHashAndPassword(hashedPassword, password); err != nil {
		t.Error("Password verification failed")
	}

	// 测试错误密码
	wrongPassword := []byte("wrong-password")
	if err := CompareHashAndPassword(hashedPassword, wrongPassword); err == nil {
		t.Error("Wrong password should not verify")
	}
}

func TestHashPasswordWithCost(t *testing.T) {
	password := []byte("test-password")
	costs := []BcryptCost{BcryptCostLow, BcryptCostDefault, BcryptCostHigh}

	for _, cost := range costs {
		hashedPassword, err := HashPasswordWithCost(password, cost)
		if err != nil {
			t.Fatalf("Password hashing with cost %d failed: %v", cost, err)
		}

		// 验证密码哈希
		if err := CompareHashAndPassword(hashedPassword, password); err != nil {
			t.Errorf("Password verification failed for cost %d", cost)
		}
	}
}

func TestSecureCompare(t *testing.T) {
	a := []byte("test-string")
	b := []byte("test-string")
	c := []byte("different-string")

	// 测试相同的字节切片
	if !SecureCompare(a, b) {
		t.Error("Identical byte slices should compare equal")
	}

	// 测试不同的字节切片
	if SecureCompare(a, c) {
		t.Error("Different byte slices should not compare equal")
	}
}

func TestGenerateRandomBytes(t *testing.T) {
	length := 32

	// 生成随机字节
	b1, err := GenerateRandomBytes(length)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}

	// 验证长度
	if len(b1) != length {
		t.Errorf("Expected length %d, got %d", length, len(b1))
	}

	// 验证两次生成的随机字节不相同
	b2, _ := GenerateRandomBytes(length)
	if bytes.Equal(b1, b2) {
		t.Error("Generated random bytes should not be equal")
	}
}

func BenchmarkAESEncryptor_Encrypt(b *testing.B) {
	key := make([]byte, 32)
	encryptor, _ := NewAESEncryptorWithMode(key, ModeGCM)
	plaintext := []byte("Hello, World!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Encrypt(plaintext)
	}
}

func BenchmarkAESEncryptor_Decrypt(b *testing.B) {
	key := make([]byte, 32)
	encryptor, _ := NewAESEncryptorWithMode(key, ModeGCM)
	plaintext := []byte("Hello, World!")
	ciphertext, _ := encryptor.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Decrypt(ciphertext)
	}
}

func BenchmarkAESEncryptor_GCM_Encrypt(b *testing.B) {
	key := make([]byte, 32)
	encryptor, _ := NewAESEncryptorWithMode(key, ModeGCM)
	plaintext := []byte("Hello, World!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Encrypt(plaintext)
	}
}

func BenchmarkAESEncryptor_GCM_Decrypt(b *testing.B) {
	key := make([]byte, 32)
	encryptor, _ := NewAESEncryptorWithMode(key, ModeGCM)
	plaintext := []byte("Hello, World!")
	ciphertext, _ := encryptor.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Decrypt(ciphertext)
	}
}

func BenchmarkHashFunctions(b *testing.B) {
	data := []byte("test data")

	b.Run("SHA256", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = HashSHA256(data)
		}
	})

	b.Run("SHA512", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = HashSHA512(data)
		}
	})

	b.Run("MD5", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = HashMD5(data)
		}
	})
}

func BenchmarkPasswordHashingWithCost(b *testing.B) {
	password := []byte("test-password")

	b.Run("LowCost", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = HashPasswordWithCost(password, BcryptCostLow)
		}
	})

	b.Run("DefaultCost", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = HashPasswordWithCost(password, BcryptCostDefault)
		}
	})

	// 高成本不适合在基准测试中运行，太慢
}

func TestPasswordPolicy_Parallel(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{
			name:     "有效密码",
			password: "Test123!@#",
			wantErr:  false,
		},
		{
			name:     "密码过短",
			password: "Abc123!",
			wantErr:  true,
		},
		{
			name:     "缺少大写字母",
			password: "abcdefgh123!",
			wantErr:  true,
		},
		{
			name:     "缺少特殊字符",
			password: "Abcdefgh123",
			wantErr:  true,
		},
	}

	policy := NewDefaultPasswordPolicy()

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := policy.ValidatePassword(tc.password)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestArgon2Params(t *testing.T) {
	// 测试默认参数
	defaultParams := DefaultArgon2Params()
	if defaultParams.Memory != 64*1024 {
		t.Errorf("Expected default memory 64KB, got %d", defaultParams.Memory)
	}
	if defaultParams.Iterations != 3 {
		t.Errorf("Expected default iterations 3, got %d", defaultParams.Iterations)
	}
	if defaultParams.Parallelism != 4 {
		t.Errorf("Expected default parallelism 4, got %d", defaultParams.Parallelism)
	}
	if defaultParams.Type != Argon2id {
		t.Errorf("Expected default type Argon2id, got %d", defaultParams.Type)
	}

	// 测试快速参数
	fastParams := FastArgon2Params()
	if fastParams.Memory != 32*1024 {
		t.Errorf("Expected fast memory 32KB, got %d", fastParams.Memory)
	}
	if fastParams.Iterations != 2 {
		t.Errorf("Expected fast iterations 2, got %d", fastParams.Iterations)
	}
}

func TestHashWithArgon2(t *testing.T) {
	password := []byte("test-password-123")

	tests := []struct {
		name   string
		params *Argon2Params
	}{
		{
			name:   "DefaultParams",
			params: DefaultArgon2Params(),
		},
		{
			name:   "FastParams",
			params: FastArgon2Params(),
		},
		{
			name:   "NilParams",
			params: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashWithArgon2(password, tt.params)
			if err != nil {
				t.Fatalf("HashWithArgon2() error = %v", err)
			}

			// 验证哈希格式
			if len(hash) == 0 {
				t.Error("Hash is empty")
			}

			// 验证能够正确验证
			valid, err := VerifyArgon2Hash([]byte(hash), password)
			if err != nil {
				t.Fatalf("VerifyArgon2Hash() error = %v", err)
			}
			if !valid {
				t.Error("Hash verification failed")
			}

			// 验证错误密码失败
			wrongPassword := []byte("wrong-password")
			valid, err = VerifyArgon2Hash([]byte(hash), wrongPassword)
			if err != nil {
				t.Fatalf("VerifyArgon2Hash() error = %v", err)
			}
			if valid {
				t.Error("Hash verification should fail for wrong password")
			}
		})
	}
}

func TestVerifyArgon2Hash_InvalidFormat(t *testing.T) {
	password := []byte("test-password")
	
	// 测试无效格式
	invalidHashes := []string{
		"",
		"invalid-format",
		"$argon2x$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG",
		"$argon2id$v=18$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG",
		"$argon2id$v=19$m=0,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG",
	}

	for _, hash := range invalidHashes {
		_, err := VerifyArgon2Hash([]byte(hash), password)
		if err == nil {
			t.Errorf("VerifyArgon2Hash() should return error for invalid hash: %s", hash)
		}
	}
}

func TestScryptParams(t *testing.T) {
	// 测试默认参数
	defaultParams := DefaultScryptParams()
	if defaultParams.N != 32768 {
		t.Errorf("Expected default N 32768, got %d", defaultParams.N)
	}
	if defaultParams.R != 8 {
		t.Errorf("Expected default R 8, got %d", defaultParams.R)
	}
	if defaultParams.P != 1 {
		t.Errorf("Expected default P 1, got %d", defaultParams.P)
	}

	// 测试快速参数
	fastParams := FastScryptParams()
	if fastParams.N != 16384 {
		t.Errorf("Expected fast N 16384, got %d", fastParams.N)
	}
}

func TestHashWithScrypt(t *testing.T) {
	password := []byte("test-password-123")

	tests := []struct {
		name   string
		params *ScryptParams
	}{
		{
			name:   "DefaultParams",
			params: DefaultScryptParams(),
		},
		{
			name:   "FastParams",
			params: FastScryptParams(),
		},
		{
			name:   "NilParams",
			params: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashWithScrypt(password, tt.params)
			if err != nil {
				t.Fatalf("HashWithScrypt() error = %v", err)
			}

			// 验证哈希格式
			if len(hash) == 0 {
				t.Error("Hash is empty")
			}

			// 验证能够正确验证
			valid, err := VerifyScryptHash([]byte(hash), password)
			if err != nil {
				t.Fatalf("VerifyScryptHash() error = %v", err)
			}
			if !valid {
				t.Error("Hash verification failed")
			}

			// 验证错误密码失败
			wrongPassword := []byte("wrong-password")
			valid, err = VerifyScryptHash([]byte(hash), wrongPassword)
			if err != nil {
				t.Fatalf("VerifyScryptHash() error = %v", err)
			}
			if valid {
				t.Error("Hash verification should fail for wrong password")
			}
		})
	}
}

func TestVerifyScryptHash_InvalidFormat(t *testing.T) {
	password := []byte("test-password")
	
	// 测试无效格式
	invalidHashes := []string{
		"",
		"invalid-format",
		"$other$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG",
		"$scrypt$N=0,r=8,p=1$c29tZXNhbHQ$RdescudvJCsgt3ub+b+dWRWJTmaaJObG",
	}

	for _, hash := range invalidHashes {
		_, err := VerifyScryptHash([]byte(hash), password)
		if err == nil {
			t.Errorf("VerifyScryptHash() should return error for invalid hash: %s", hash)
		}
	}
}

func TestPasswordHasherInterface(t *testing.T) {
	password := []byte("test-password-123")

	hashers := []PasswordHasher{
		NewBcryptHasher(BcryptCostDefault),
		NewArgon2Hasher(DefaultArgon2Params()),
		NewScryptHasher(DefaultScryptParams()),
	}

	for _, hasher := range hashers {
		t.Run(fmt.Sprintf("%T", hasher), func(t *testing.T) {
			// 测试哈希
			hash, err := hasher.Hash(password)
			if err != nil {
				t.Fatalf("Hash() error = %v", err)
			}

			// 测试验证
			valid, err := hasher.Verify([]byte(hash), password)
			if err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if !valid {
				t.Error("Verify() should return true for correct password")
			}

			// 测试错误密码
			wrongPassword := []byte("wrong-password")
			valid, err = hasher.Verify([]byte(hash), wrongPassword)
			if err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if valid {
				t.Error("Verify() should return false for wrong password")
			}
		})
	}
}

func TestBenchmarkPasswordHashers(t *testing.T) {
	password := []byte("test-password")
	iterations := 5 // 减少迭代次数以避免测试运行时间过长

	results := BenchmarkPasswordHashers(password, iterations)

	// 验证结果包含所有算法
	expectedAlgorithms := []string{"bcrypt", "argon2", "argon2-fast", "scrypt", "scrypt-fast"}
	for _, algo := range expectedAlgorithms {
		if _, exists := results[algo]; !exists {
			t.Errorf("Benchmark results missing algorithm: %s", algo)
		}
		if results[algo] <= 0 {
			t.Errorf("Benchmark duration for %s should be positive", algo)
		}
	}
}

func BenchmarkArgon2Hashing(b *testing.B) {
	password := []byte("test-password")

	b.Run("Default", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = HashWithArgon2(password, DefaultArgon2Params())
		}
	})

	b.Run("Fast", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = HashWithArgon2(password, FastArgon2Params())
		}
	})
}

func BenchmarkScryptHashing(b *testing.B) {
	password := []byte("test-password")

	b.Run("Default", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = HashWithScrypt(password, DefaultScryptParams())
		}
	})

	b.Run("Fast", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = HashWithScrypt(password, FastScryptParams())
		}
	})
}

func BenchmarkArgon2Verification(b *testing.B) {
	password := []byte("test-password")
	hash, _ := HashWithArgon2(password, DefaultArgon2Params())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = VerifyArgon2Hash([]byte(hash), password)
	}
}

func BenchmarkScryptVerification(b *testing.B) {
	password := []byte("test-password")
	hash, _ := HashWithScrypt(password, DefaultScryptParams())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = VerifyScryptHash([]byte(hash), password)
	}
}
