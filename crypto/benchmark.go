package crypto

import "time"

// BenchmarkPasswordHashers 性能测试不同的密码哈希算法
// 返回每个算法的耗时（纳秒）
func BenchmarkPasswordHashers(password []byte, iterations int) map[string]time.Duration {
	results := make(map[string]time.Duration)

	// 测试bcrypt
	start := time.Now()
	for range iterations {
		HashPasswordWithCost(password, BcryptCostDefault)
	}
	results["bcrypt"] = time.Since(start)

	// 测试Argon2
	argonParams := DefaultArgon2Params()
	start = time.Now()
	for range iterations {
		HashWithArgon2(password, argonParams)
	}
	results["argon2"] = time.Since(start)

	// 测试快速Argon2
	fastArgonParams := FastArgon2Params()
	start = time.Now()
	for range iterations {
		HashWithArgon2(password, fastArgonParams)
	}
	results["argon2-fast"] = time.Since(start)

	// 测试scrypt
	scryptParams := DefaultScryptParams()
	start = time.Now()
	for range iterations {
		HashWithScrypt(password, scryptParams)
	}
	results["scrypt"] = time.Since(start)

	// 测试快速scrypt
	fastScryptParams := FastScryptParams()
	start = time.Now()
	for range iterations {
		HashWithScrypt(password, fastScryptParams)
	}
	results["scrypt-fast"] = time.Since(start)

	return results
}
