package account_test

import (
	"encoding/hex"
	"fmt"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/keystore"
	"github.com/0xmhha/accounts/signing"
)

// Create an account from an existing private key and read its address.
func ExampleFromPrivateKeyBytes() {
	priv, _ := hex.DecodeString("4646464646464646464646464646464646464646464646464646464646464646")
	acct, _ := account.FromPrivateKeyBytes(priv)
	fmt.Println(acct.Address())
	// Output: 0x9d8a62f656a8d1615c1294fd71e9cfb3e4855a4f
}

// Generate a fresh account and sign a message hash; the signature recovers back
// to the account address.
func ExampleAccount_Sign() {
	acct, _ := account.Generate()
	hash := crypto.Keccak256([]byte("hello stablenet"))
	sig, _ := acct.Sign(hash)

	signer, _ := crypto.Recover(hash, sig)
	fmt.Println(signer == acct.Address())
	// Output: true
}

// Encrypt an account to a keystore-v3 file and load it back.
func ExampleAccount_ToKeystore() {
	acct, _ := account.Generate()

	// Encrypt (use keystore.StandardScryptN/P in production).
	doc, _ := acct.ToKeystore("my-password", keystore.LightScryptN, keystore.LightScryptP)

	// Later: decrypt.
	loaded, _ := account.FromKeystore(doc, "my-password")
	fmt.Println(loaded.Address() == acct.Address())
	// Output: true
}

// Sign a message with EIP-191 personal_sign framing; the signature recovers to
// the signer.
func ExampleAccount_SignPersonal() {
	priv, _ := hex.DecodeString("4646464646464646464646464646464646464646464646464646464646464646")
	acct, _ := account.FromPrivateKeyBytes(priv)

	sig, _ := acct.SignPersonal([]byte("Sign in @ 1700000000"))
	signer, _ := crypto.Recover(signing.EIP191Hash([]byte("Sign in @ 1700000000")), sig)
	fmt.Println(signer == acct.Address())
	// Output: true
}

// Encrypt data to an account's public key (ECIES) and decrypt it with the
// account's private key.
func ExampleEncrypt() {
	recipient, _ := account.Generate()

	ciphertext, _ := account.Encrypt(recipient.PublicKey(), []byte("secret memo"))
	plaintext, _ := recipient.Decrypt(ciphertext)
	fmt.Printf("%s\n", plaintext)
	// Output: secret memo
}
