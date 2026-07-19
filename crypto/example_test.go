package crypto_test

import (
	"encoding/hex"
	"fmt"

	"github.com/0xmhha/accounts/crypto"
)

// Keccak-256 hashing (the hash used for sighashes and addresses).
func ExampleKeccak256() {
	fmt.Println(hex.EncodeToString(crypto.Keccak256([]byte("abc"))))
	// Output: 4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45
}

// Encrypt data to a public key and decrypt with the private key (ECIES).
func ExampleEncrypt() {
	priv, _ := crypto.GenerateKey()

	ciphertext, _ := crypto.Encrypt(priv.PubKey(), []byte("hello"))
	plaintext, _ := crypto.Decrypt(priv, ciphertext)
	fmt.Printf("%s\n", plaintext)
	// Output: hello
}
