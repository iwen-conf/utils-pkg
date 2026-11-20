package crypto

import (
	"bytes"
	"errors"
	"testing"
)

func TestAESEncryptor_AADAndCiphertextBounds(t *testing.T) {
	key := bytes.Repeat([]byte{0x42}, 32)
	enc, err := NewAESEncryptor(key)
	if err != nil {
		t.Fatalf("NewAESEncryptor: %v", err)
	}

	// Standard Base64 + AAD
	aad := []byte("tenant:t1|user:u9")
	ct, err := enc.EncryptWithAAD([]byte("secret-value"), aad)
	if err != nil {
		t.Fatalf("EncryptWithAAD: %v", err)
	}
	pt, err := enc.DecryptWithAAD(ct, aad)
	if err != nil || string(pt) != "secret-value" {
		t.Fatalf("DecryptWithAAD: pt=%q err=%v", pt, err)
	}
	if _, err := enc.DecryptWithAAD(ct, []byte("tenant:t1|user:u8")); err == nil {
		t.Fatalf("mismatched AAD must fail GCM authentication")
	}
	if _, err := enc.Decrypt(ct); err == nil {
		t.Fatalf("missing AAD must fail GCM authentication")
	}

	// URL-safe Base64 + AAD + empty plaintext
	ctURL, err := enc.EncryptWithAADAndEncoding(nil, aad, EncodingURLSafe)
	if err != nil {
		t.Fatalf("EncryptWithAADAndEncoding empty: %v", err)
	}
	ptEmpty, err := enc.DecryptWithAADAndEncoding(ctURL, aad, EncodingURLSafe)
	if err != nil || len(ptEmpty) != 0 {
		t.Fatalf("DecryptWithAADAndEncoding empty: pt=%v err=%v", ptEmpty, err)
	}

	// Ciphertext shorter than nonce + tag returns ErrCiphertextTooShort
	if _, err := enc.Decrypt("YWJjZA=="); !errors.Is(err, ErrCiphertextTooShort) {
		t.Fatalf("expected ErrCiphertextTooShort, got %v", err)
	}
}
