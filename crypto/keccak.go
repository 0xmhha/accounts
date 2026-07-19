// Package crypto provides the SDK's cryptographic primitives: Keccak-256
// hashing and secp256k1 key/signature operations. It is a clean-room layer
// (ADR-0001) built on permissively licensed libraries (golang.org/x/crypto,
// github.com/decred/dcrd/dcrec/secp256k1) — no LGPL/GPL sources are used.
package crypto

import "golang.org/x/crypto/sha3"

// Keccak256 returns the Keccak-256 (legacy, pre-NIST) hash of the concatenation
// of the given byte slices. This is the hash used throughout Ethereum/StableNet
// for sighashes and address derivation.
func Keccak256(data ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	for _, d := range data {
		h.Write(d)
	}
	return h.Sum(nil)
}
