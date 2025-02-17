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