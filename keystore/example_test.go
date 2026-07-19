package keystore_test

import (
	"encoding/hex"
	"fmt"

	"github.com/0xmhha/accounts/keystore"
)

// Encrypt a private key with a password and decrypt it back. The keystore-v3
// output is compatible with go-ethereum / go-stablenet keystores.
func ExampleEncrypt() {
	priv, _ := hex.DecodeString("4646464646464646464646464646464646464646464646464646464646464646")

	// LightScryptN/P are for tests; use StandardScryptN/P for real keys.
	doc, _ := keystore.Encrypt(priv, "password", keystore.LightScryptN, keystore.LightScryptP)

	back, _ := keystore.Decrypt(doc, "password")
	fmt.Println(hex.EncodeToString(back) == hex.EncodeToString(priv))
	// Output: true
}
