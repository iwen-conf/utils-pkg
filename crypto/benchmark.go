package crypto

import (
	"errors"
	"fmt"
	"time"
)

// ErrInvalidBenchmarkIterations 表示基准迭代次数必须为正整数。
var ErrInvalidBenchmarkIterations = errors.New("benchmark iterations must be positive")

// BenchmarkPasswordHashersChecked 对各密码哈希算法运行指定次数基准测量，并在任一算法出错时返回错误。
func BenchmarkPasswordHashersChecked(password []byte, iterations int) (map[string]time.Duration, error) {
	if iterations <= 0 {
		return nil, ErrInvalidBenchmarkIterations
	}

	results := make(map[string]time.Duration, 5)
	argonParams := DefaultArgon2Params()
	fastArgonParams := FastArgon2Params()
	scryptParams := DefaultScryptParams()
	fastScryptParams := FastScryptParams()

	runners := []struct {
		name string
		fn   func() error
	}{
		{
			name: "bcrypt",
			fn: func() error {
				_, err := HashPasswordWithCost(password, BcryptCostDefault)
				return err
			},
		},
		{
			name: "argon2",
			fn: func() error {
				_, err := HashWithArgon2(password, argonParams)
				return err
			},
		},
		{
			name: "argon2-fast",
			fn: func() error {
				_, err := HashWithArgon2(password, fastArgonParams)
				return err
			},
		},
		{
			name: "scrypt",
			fn: func() error {
				_, err := HashWithScrypt(password, scryptParams)
				return err
			},
		},
		{
			name: "scrypt-fast",
			fn: func() error {
				_, err := HashWithScrypt(password, fastScryptParams)
				return err
			},
		},
	}

	for _, runner := range runners {
		start := time.Now()
		for range iterations {
			if err := runner.fn(); err != nil {
				return nil, fmt.Errorf("benchmark %s: %w", runner.name, err)
			}
		}
		results[runner.name] = time.Since(start)
	}
	return results, nil
}

// BenchmarkPasswordHashers 测量不同密码哈希算法的耗时。
// 若某个算法对输入报错（例如密码超过 bcrypt 72 字节上限），则跳过该算法，绝不写入虚假的失败耗时。
func BenchmarkPasswordHashers(password []byte, iterations int) map[string]time.Duration {
	results := make(map[string]time.Duration, 5)
	if iterations <= 0 {
		return results
	}

	argonParams := DefaultArgon2Params()
	fastArgonParams := FastArgon2Params()
	scryptParams := DefaultScryptParams()
	fastScryptParams := FastScryptParams()

	measure := func(name string, fn func() error) {
		start := time.Now()
		for range iterations {
			if err := fn(); err != nil {
				return
			}
		}
		results[name] = time.Since(start)
	}

	measure("bcrypt", func() error {
		_, err := HashPasswordWithCost(password, BcryptCostDefault)
		return err
	})
	measure("argon2", func() error {
		_, err := HashWithArgon2(password, argonParams)
		return err
	})
	measure("argon2-fast", func() error {
		_, err := HashWithArgon2(password, fastArgonParams)
		return err
	})
	measure("scrypt", func() error {
		_, err := HashWithScrypt(password, scryptParams)
		return err
	})
	measure("scrypt-fast", func() error {
		_, err := HashWithScrypt(password, fastScryptParams)
		return err
	})

	return results
}
