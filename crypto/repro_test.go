package crypto

import (
	"bytes"
	"testing"
)

func TestRepro_AESNilAndZeroReceiverDoNotPanic(t *testing.T) {
	var nilEnc *AESEncryptor
	if _, err := nilEnc.Encrypt([]byte("hi")); err == nil {
		t.Fatalf("expected error on nil AESEncryptor.Encrypt")
	}
	var zeroEnc AESEncryptor
	if _, err := zeroEnc.Decrypt("YWJj"); err == nil {
		t.Fatalf("expected error on zero AESEncryptor.Decrypt")
	}
}

func TestRepro_BenchmarkPasswordHashersHandlesBcryptOver72Bytes(t *testing.T) {
	longPwd := bytes.Repeat([]byte("a"), 80)
	// bcrypt rejects >72 bytes; BenchmarkPasswordHashers must not silently pretend bcrypt succeeded!
	res := BenchmarkPasswordHashers(longPwd, 1)
	if d, exists := res["bcrypt"]; exists && d > 0 {
		t.Fatalf("BUG: BenchmarkPasswordHashers silently ignored bcrypt >72B error and reported valid timing %v", d)
	}
	if _, err := BenchmarkPasswordHashersChecked(longPwd, 1); err == nil {
		t.Fatalf("expected BenchmarkPasswordHashersChecked to return bcrypt >72B error")
	}
	if _, err := BenchmarkPasswordHashersChecked([]byte("ok"), 0); err != ErrInvalidBenchmarkIterations {
		t.Fatalf("expected ErrInvalidBenchmarkIterations, got %v", err)
	}
	if _, err := NewBcryptHasherChecked(BcryptCost(99)); err == nil {
		t.Fatalf("expected invalid bcrypt cost error")
	}
	var nilBcrypt *BcryptHasher
	if _, err := nilBcrypt.Hash([]byte("x")); err != ErrUninitializedHasher {
		t.Fatalf("expected ErrUninitializedHasher for nil BcryptHasher, got %v", err)
	}
	var nilArgon *Argon2Hasher
	if _, err := nilArgon.Hash([]byte("x")); err != ErrUninitializedHasher {
		t.Fatalf("expected ErrUninitializedHasher for nil Argon2Hasher, got %v", err)
	}
	var nilScrypt *ScryptHasher
	if _, err := nilScrypt.Hash([]byte("x")); err != ErrUninitializedHasher {
		t.Fatalf("expected ErrUninitializedHasher for nil ScryptHasher, got %v", err)
	}
}
