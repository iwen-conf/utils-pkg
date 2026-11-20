package crypto_test

import (
	"fmt"

	"github.com/iwen-conf/utils-pkg/crypto"
)

func ExampleAESEncryptor_EncryptWithAAD() {
	key := []byte("0123456789abcdef0123456789abcdef")
	enc, err := crypto.NewAESEncryptor(key)
	if err != nil {
		panic(err)
	}

	// Bind ciphertext to user:1001 so it cannot be transplanted to another user's record.
	aad := []byte("user:1001")
	ciphertext, err := enc.EncryptWithAAD([]byte("6222-0000-1234-5678"), aad)
	if err != nil {
		panic(err)
	}

	plaintext, err := enc.DecryptWithAAD(ciphertext, aad)
	if err != nil {
		panic(err)
	}
	fmt.Printf("decrypted: %s\n", plaintext)

	_, err = enc.DecryptWithAAD(ciphertext, []byte("user:9999"))
	fmt.Printf("wrong AAD rejected: %v\n", err != nil)

	// Output:
	// decrypted: 6222-0000-1234-5678
	// wrong AAD rejected: true
}

func ExampleNewArgon2Hasher() {
	hasher, err := crypto.NewArgon2Hasher(crypto.FastArgon2Params())
	if err != nil {
		panic(err)
	}

	encoded, err := hasher.Hash([]byte("CorrectHorseBatteryStaple!9"))
	if err != nil {
		panic(err)
	}

	ok, err := hasher.Verify([]byte(encoded), []byte("CorrectHorseBatteryStaple!9"))
	if err != nil {
		panic(err)
	}
	fmt.Printf("verified: %v\n", ok)

	// Output:
	// verified: true
}
