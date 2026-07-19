package crypto

import (
	"errors"
	"fmt"

	"github.com/0xmhha/accounts/types"
	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

// PrivateKey is a secp256k1 private key. It aliases the underlying permissive
// library type so callers depend only on this package.
type PrivateKey = secp256k1.PrivateKey

// PublicKey is a secp256k1 public key.
type PublicKey = secp256k1.PublicKey

// SignatureLength is the byte length of a recoverable signature [R || S || V].
const SignatureLength = 65

// compactMagicOffset is the "+27" recovery-code offset used by the compact
// signature format (see decred SignCompact).
const compactMagicOffset = 27

// GenerateKey generates a new random secp256k1 private key.
func GenerateKey() (*PrivateKey, error) {
	return secp256k1.GeneratePrivateKey()
}

// PrivKeyFromBytes returns a private key from a 32-byte big-endian scalar.
func PrivKeyFromBytes(b []byte) (*PrivateKey, error) {
	if len(b) != 32 {
		return nil, fmt.Errorf("private key must be 32 bytes, got %d", len(b))
	}
	return secp256k1.PrivKeyFromBytes(b), nil
}

// Sign signs a 32-byte hash with priv and returns a 65-byte recoverable
// signature in Ethereum layout [R(32) || S(32) || V(1)] where V is the recovery
// id (0 or 1). The signature is deterministic (RFC 6979) and canonical (low-S).
func Sign(hash []byte, priv *PrivateKey) ([]byte, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash must be 32 bytes, got %d", len(hash))
	}
	if priv == nil {
		return nil, errors.New("nil private key")
	}
	// Compact layout: [27+recID || R || S] (uncompressed pubkey).
	compact := ecdsa.SignCompact(priv, hash, false)
	if len(compact) != SignatureLength {
		return nil, fmt.Errorf("unexpected compact signature length %d", len(compact))
	}
	recID := compact[0] - compactMagicOffset
	sig := make([]byte, SignatureLength)
	copy(sig[0:32], compact[1:33])   // R
	copy(sig[32:64], compact[33:65]) // S
	sig[64] = recID                  // V (0/1)
	return sig, nil
}

// RecoverPubkey recovers the public key that produced sig over hash. sig must be
// in Ethereum layout [R || S || V].
func RecoverPubkey(hash, sig []byte) (*PublicKey, error) {
	if len(hash) != 32 {
		return nil, fmt.Errorf("hash must be 32 bytes, got %d", len(hash))
	}
	if len(sig) != SignatureLength {
		return nil, fmt.Errorf("signature must be %d bytes, got %d", SignatureLength, len(sig))
	}
	v := sig[64]
	if v >= 4 {
		return nil, fmt.Errorf("invalid recovery id %d", v)
	}
	// Convert to compact layout [27+V || R || S].
	compact := make([]byte, SignatureLength)
	compact[0] = compactMagicOffset + v
	copy(compact[1:33], sig[0:32])
	copy(compact[33:65], sig[32:64])
	pub, _, err := ecdsa.RecoverCompact(compact, hash)
	if err != nil {
		return nil, fmt.Errorf("recover: %w", err)
	}
	return pub, nil
}

// Recover recovers the signer address from a signature over hash.
func Recover(hash, sig []byte) (types.Address, error) {
	pub, err := RecoverPubkey(hash, sig)
	if err != nil {
		return types.Address{}, err
	}
	return PubkeyToAddress(pub), nil
}

// PubkeyToAddress derives the 20-byte account address from a public key:
// keccak256(uncompressed_pubkey[1:])[12:].
func PubkeyToAddress(pub *PublicKey) types.Address {
	uncompressed := pub.SerializeUncompressed() // 0x04 || X(32) || Y(32)
	h := Keccak256(uncompressed[1:])
	return types.BytesToAddress(h[12:])
}

// PrivKeyToAddress derives the account address for a private key.
func PrivKeyToAddress(priv *PrivateKey) types.Address {
	return PubkeyToAddress(priv.PubKey())
}
