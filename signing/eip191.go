package signing

import (
	"fmt"

	"github.com/0xmhha/accounts/crypto"
)

// EIP191Hash returns the EIP-191 "personal_sign" digest of msg:
//
//	keccak256("\x19Ethereum Signed Message:\n" + len(msg) + msg)
//
// go-stablenet uses the standard Ethereum prefix (accounts/accounts.go), so a
// signature over this digest is recoverable by eth/personal_ecRecover on the
// node.
func EIP191Hash(msg []byte) []byte {
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(msg))
	return crypto.Keccak256([]byte(prefix), msg)
}
