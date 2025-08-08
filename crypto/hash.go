package crypto

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
)

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

// SecureCompare 使用恒定时间比较两个字节切片
func SecureCompare(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}