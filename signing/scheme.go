// Package signing defines the SigningScheme abstraction (spec:
// docs/spec/protocol/v0/signing.md). Isolating signing behind this interface
// keeps the (security-critical) key/signature surface small and lets the
// signature algorithm evolve via new scheme implementations without touching
// higher layers.
package signing

import (
	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/types"
)

// Scheme signs and recovers over precomputed 32-byte hashes. Per-transaction
// sighash construction lives in the tx package; a Scheme only touches raw keys
// and signatures, which is the only cryptographic surface to audit.
type Scheme interface {
	// ID returns the versioned scheme identifier, e.g. "secp256k1@1".
	ID() string
	// Sign returns a recoverable signature [R || S || V] over hash.
	Sign(hash []byte, priv *crypto.PrivateKey) ([]byte, error)
	// Recover returns the signer address for a signature over hash.
	Recover(hash, sig []byte) (types.Address, error)
}

// Secp256k1 is the only scheme in protocol/v0: standard Ethereum secp256k1
// signatures. go-stablenet uses secp256k1 exclusively for transaction signing.
type Secp256k1 struct{}

// ID implements Scheme.
func (Secp256k1) ID() string { return "secp256k1@1" }

// Sign implements Scheme.
func (Secp256k1) Sign(hash []byte, priv *crypto.PrivateKey) ([]byte, error) {
	return crypto.Sign(hash, priv)
}

// Recover implements Scheme.
func (Secp256k1) Recover(hash, sig []byte) (types.Address, error) {
	return crypto.Recover(hash, sig)
}

// Default is the scheme used unless overridden.
var Default Scheme = Secp256k1{}
