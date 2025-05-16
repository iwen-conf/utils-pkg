package crypto

import (
	"bytes"
	"encoding/base64"
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
	encryptor, _ := NewAESEncryptor(key)

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

func TestAESEncryptor_URLSafeEncoding(t *testing.T) {
	key := make([]byte, 32)
	encryptor, _ := NewAESEncryptor(key)

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
	encryptor, _ := NewAESEncryptor(key)
	plaintext := []byte("Hello, World!")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = encryptor.Encrypt(plaintext)
	}
}

func BenchmarkAESEncryptor_Decrypt(b *testing.B) {
	key := make([]byte, 32)
	encryptor, _ := NewAESEncryptor(key)
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
