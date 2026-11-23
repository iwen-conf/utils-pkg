package crypto

import (
	"bytes"
	"errors"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"
)

func TestNewAESEncryptor_ValidationAndEncoding(t *testing.T) {
	for _, n := range []int{0, 15, 17, 23, 25, 31, 33} {
		if _, err := NewAESEncryptor(make([]byte, n)); !errors.Is(err, ErrInvalidAESKeySize) {
			t.Fatalf("key %d: %v", n, err)
		}
	}
	key := bytes.Repeat([]byte{1}, 16)
	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}
	var encIfc Encryptor = enc
	if _, err := encIfc.EncryptWithOptions([]byte("x"), EncodingType(9)); !errors.Is(err, ErrUnsupportedEncoding) {
		t.Fatalf("EncryptWithOptions invalid encoding: %v", err)
	}
	if _, err := encIfc.DecryptWithOptions("YWJj", EncodingType(9)); !errors.Is(err, ErrUnsupportedEncoding) {
		t.Fatalf("DecryptWithOptions invalid encoding: %v", err)
	}
	if _, err := encIfc.EncryptWithAADAndEncoding([]byte("x"), []byte("aad"), EncodingType(9)); !errors.Is(err, ErrUnsupportedEncoding) {
		t.Fatalf("EncryptWithAADAndEncoding invalid encoding: %v", err)
	}
	if _, err := encIfc.DecryptWithAADAndEncoding("YWJj", []byte("aad"), EncodingType(9)); !errors.Is(err, ErrUnsupportedEncoding) {
		t.Fatalf("DecryptWithAADAndEncoding invalid encoding: %v", err)
	}
}

func TestAESEncryptor_TamperAndConcurrent(t *testing.T) {
	key := bytes.Repeat([]byte{7}, 32)
	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatal(err)
	}
	ct, err := enc.Encrypt([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := enc.Decrypt(ct[:len(ct)-1]); err == nil {
		t.Fatalf("truncated ciphertext should fail")
	}
	tampered := []byte(ct)
	tampered[len(tampered)-1] ^= 0xff
	if _, err := enc.Decrypt(string(tampered)); err == nil {
		t.Fatalf("tampered tag should fail")
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c1, err := enc.Encrypt([]byte("abc"))
			if err != nil {
				t.Errorf("encrypt: %v", err)
				return
			}
			c2, err := enc.Encrypt([]byte("abc"))
			if err != nil {
				t.Errorf("encrypt: %v", err)
				return
			}
			if c1 == c2 {
				t.Errorf("nonces should differ")
			}
			pt, err := enc.Decrypt(c1)
			if err != nil || string(pt) != "abc" {
				t.Errorf("decrypt: %v %q", err, pt)
			}
		}()
	}
	wg.Wait()
}

func TestValidateArgon2Params_Bounds(t *testing.T) {
	ok := *DefaultArgon2Params()
	if err := validateArgon2Params(ok); err != nil {
		t.Fatalf("default: %v", err)
	}
	min := Argon2Params{Memory: minArgon2MemoryKiB, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 16, Type: Argon2i}
	if err := validateArgon2Params(min); err != nil {
		t.Fatalf("min: %v", err)
	}
	bad := []Argon2Params{
		{Memory: minArgon2MemoryKiB - 1, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 16, Type: Argon2id},
		{Memory: maxArgon2MemoryKiB + 1, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 16, Type: Argon2id},
		{Memory: minArgon2MemoryKiB, Iterations: 0, Parallelism: 1, SaltLength: 16, KeyLength: 16, Type: Argon2id},
		{Memory: minArgon2MemoryKiB, Iterations: 11, Parallelism: 1, SaltLength: 16, KeyLength: 16, Type: Argon2id},
		{Memory: minArgon2MemoryKiB, Iterations: 1, Parallelism: 0, SaltLength: 16, KeyLength: 16, Type: Argon2id},
		{Memory: minArgon2MemoryKiB, Iterations: 1, Parallelism: 17, SaltLength: 16, KeyLength: 16, Type: Argon2id},
		{Memory: minArgon2MemoryKiB, Iterations: 1, Parallelism: 1, SaltLength: 15, KeyLength: 16, Type: Argon2id},
		{Memory: minArgon2MemoryKiB, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 15, Type: Argon2id},
		{Memory: minArgon2MemoryKiB, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 16, Type: Argon2Type(9)},
	}
	for i, params := range bad {
		if err := validateArgon2Params(params); !errors.Is(err, ErrInvalidArgon2Params) {
			t.Fatalf("case %d: %v", i, err)
		}
	}

	if _, err := VerifyArgon2Hash([]byte(strings.Repeat("a", 4097)), []byte("x")); !errors.Is(err, ErrInvalidArgon2Params) {
		t.Fatalf("long hash: %v", err)
	}
	if _, err := VerifyArgon2Hash([]byte("$argon2id$v=19$m=4294967295,t=3,p=256$aaaa$bbbb"), []byte("x")); !errors.Is(err, ErrInvalidArgon2Params) {
		t.Fatalf("malicious params: %v", err)
	}
}

func TestNewArgon2Hasher_CopiesParams(t *testing.T) {
	params := FastArgon2Params()
	hasher, err := NewArgon2Hasher(params)
	if err != nil {
		t.Fatal(err)
	}
	params.Memory = 1
	if hasher.params.Memory == 1 {
		t.Fatalf("hasher must copy params")
	}
}

func TestValidateScryptParams_Bounds(t *testing.T) {
	if err := validateScryptParams(*DefaultScryptParams()); err != nil {
		t.Fatal(err)
	}
	min := ScryptParams{N: 2, R: 1, P: 1, SaltLength: 16, KeyLength: 16}
	if err := validateScryptParams(min); err != nil {
		t.Fatalf("min: %v", err)
	}
	// 128*N*r == 256MiB
	atLimit := ScryptParams{N: 1 << 16, R: 32, P: 1, SaltLength: 16, KeyLength: 16}
	if err := validateScryptParams(atLimit); err != nil {
		t.Fatalf("at limit: %v", err)
	}
	over := atLimit
	over.N = 1 << 17
	if err := validateScryptParams(over); !errors.Is(err, ErrInvalidScryptParams) {
		t.Fatalf("over memory: %v", err)
	}
	for i, params := range []ScryptParams{
		{N: 0, R: 1, P: 1, SaltLength: 16, KeyLength: 16},
		{N: 1, R: 1, P: 1, SaltLength: 16, KeyLength: 16},
		{N: 3, R: 1, P: 1, SaltLength: 16, KeyLength: 16},
		{N: 4, R: 0, P: 1, SaltLength: 16, KeyLength: 16},
		{N: 4, R: 33, P: 1, SaltLength: 16, KeyLength: 16},
		{N: 4, R: 1, P: 0, SaltLength: 16, KeyLength: 16},
		{N: 4, R: 1, P: 17, SaltLength: 16, KeyLength: 16},
	} {
		if err := validateScryptParams(params); !errors.Is(err, ErrInvalidScryptParams) {
			t.Fatalf("case %d: %v", i, err)
		}
	}
	if _, err := VerifyScryptHash([]byte(strings.Repeat("a", 4097)), []byte("x")); !errors.Is(err, ErrInvalidScryptParams) {
		t.Fatalf("long hash: %v", err)
	}
}

func TestPasswordPolicy_ConfigAndUnicode(t *testing.T) {
	cfg := PasswordPolicyConfig{MinLength: 4, MaxLength: 4, RequireUpper: true, RequireLower: false, RequireNumber: true, RequireSpecial: true}
	policy, err := NewPasswordPolicy(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := policy.ValidatePassword("中A1!"); err != nil {
		t.Fatalf("4-rune password: %v", err)
	}
	if err := policy.ValidatePassword("中A!"); !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("short rune: %v", err)
	}
	if utf8.RuneCountInString("中A1!") != 4 {
		t.Fatalf("fixture length")
	}

	got, err := policy.Config()
	if err != nil {
		t.Fatal(err)
	}
	got.DisallowWords = []string{"x"}
	if err := policy.SetDisallowWords([]string{"secret"}); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPasswordPolicy(PasswordPolicyConfig{MinLength: 0, MaxLength: 8}); !errors.Is(err, ErrInvalidPasswordPolicy) {
		t.Fatalf("min 0: %v", err)
	}
	if err := policy.SetDisallowWords([]string{""}); !errors.Is(err, ErrInvalidPasswordPolicy) {
		t.Fatalf("empty word: %v", err)
	}
	var nilPolicy *PasswordPolicy
	if _, err := nilPolicy.Config(); !errors.Is(err, ErrInvalidPasswordPolicy) {
		t.Fatalf("nil config: %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = policy.SetConfig(cfg)
			_ = policy.ValidatePassword("中A1!")
		}()
	}
	wg.Wait()
}

func TestGenerateRandomBytes_Bounds(t *testing.T) {
	if _, err := GenerateRandomBytes(0); !errors.Is(err, ErrInvalidRandomLength) {
		t.Fatalf("0: %v", err)
	}
	if _, err := GenerateRandomBytes(-1); !errors.Is(err, ErrInvalidRandomLength) {
		t.Fatalf("-1: %v", err)
	}
	if _, err := GenerateRandomBytes(MaxRandomBytesLength + 1); !errors.Is(err, ErrInvalidRandomLength) {
		t.Fatalf("max+1: %v", err)
	}
	b, err := GenerateRandomBytes(1)
	if err != nil || len(b) != 1 {
		t.Fatalf("len 1: %v %d", err, len(b))
	}
}
