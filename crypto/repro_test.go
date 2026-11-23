package crypto

import (
	"bytes"
	"errors"
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

func TestRepro_FailFastHasherValidation(t *testing.T) {
	longPwd := bytes.Repeat([]byte("a"), 80)
	if _, err := HashPasswordWithCost(longPwd, BcryptCostLow); err == nil {
		t.Fatalf("expected bcrypt >72B error")
	}
	if _, err := NewBcryptHasher(BcryptCost(99)); !errors.Is(err, ErrInvalidBcryptCost) {
		t.Fatalf("expected ErrInvalidBcryptCost, got %v", err)
	}
	var nilBcrypt *BcryptHasher
	if _, err := nilBcrypt.Hash([]byte("x")); !errors.Is(err, ErrUninitializedHasher) {
		t.Fatalf("expected ErrUninitializedHasher for nil BcryptHasher, got %v", err)
	}
	var nilArgon *Argon2Hasher
	if _, err := nilArgon.Hash([]byte("x")); !errors.Is(err, ErrUninitializedHasher) {
		t.Fatalf("expected ErrUninitializedHasher for nil Argon2Hasher, got %v", err)
	}
	var nilScrypt *ScryptHasher
	if _, err := nilScrypt.Hash([]byte("x")); !errors.Is(err, ErrUninitializedHasher) {
		t.Fatalf("expected ErrUninitializedHasher for nil ScryptHasher, got %v", err)
	}
}
