package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/hkdf"
)

// ECIES provides asymmetric encryption to a secp256k1 public key. Anyone can
// encrypt a message to an account's public key; only the holder of the matching
// private key can decrypt it.
//
// Scheme (self-consistent, not wire-compatible with any specific standard):
//
//	ephemeral key E
//	shared = ECDH(E_priv, recipientPub)              // 32-byte X coordinate
//	key    = HKDF-SHA256(shared, info="ecies-secp256k1-aesgcm") -> 32 bytes
//	out    = E_pub_uncompressed(65) || nonce(12) || AES-256-GCM(key, nonce, msg)
//
// The ephemeral public key is authenticated as GCM additional data.
const (
	eciesEphemeralLen = 65 // uncompressed pubkey
	eciesNonceLen     = 12
)

var eciesInfo = []byte("ecies-secp256k1-aesgcm/v1")

// Encrypt encrypts msg to the recipient's public key.
func Encrypt(recipient *PublicKey, msg []byte) ([]byte, error) {
	if recipient == nil {
		return nil, errors.New("nil recipient public key")
	}
	eph, err := secp256k1.GeneratePrivateKey()
	if err != nil {
		return nil, err
	}
	shared := secp256k1.GenerateSharedSecret(eph, recipient)
	gcm, err := eciesGCM(shared)
	if err != nil {
		return nil, err
	}
	ephPub := eph.PubKey().SerializeUncompressed()
	nonce := make([]byte, eciesNonceLen)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ct := gcm.Seal(nil, nonce, msg, ephPub) // ephPub as AAD
	out := make([]byte, 0, eciesEphemeralLen+eciesNonceLen+len(ct))
	out = append(out, ephPub...)
	out = append(out, nonce...)
	out = append(out, ct...)
	return out, nil
}

// Decrypt decrypts a blob produced by Encrypt using the recipient's private key.
func Decrypt(priv *PrivateKey, blob []byte) ([]byte, error) {
	if priv == nil {
		return nil, errors.New("nil private key")
	}
	if len(blob) < eciesEphemeralLen+eciesNonceLen {
		return nil, errors.New("ciphertext too short")
	}
	ephPub := blob[:eciesEphemeralLen]
	nonce := blob[eciesEphemeralLen : eciesEphemeralLen+eciesNonceLen]
	ct := blob[eciesEphemeralLen+eciesNonceLen:]

	pub, err := secp256k1.ParsePubKey(ephPub)
	if err != nil {
		return nil, fmt.Errorf("parse ephemeral pubkey: %w", err)
	}
	shared := secp256k1.GenerateSharedSecret(priv, pub)
	gcm, err := eciesGCM(shared)
	if err != nil {
		return nil, err
	}
	msg, err := gcm.Open(nil, nonce, ct, ephPub)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return msg, nil
}

// eciesGCM derives an AES-256-GCM AEAD from the ECDH shared secret via HKDF.
func eciesGCM(shared []byte) (cipher.AEAD, error) {
	kdf := hkdf.New(sha256.New, shared, nil, eciesInfo)
	key := make([]byte, 32)
	if _, err := io.ReadFull(kdf, key); err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
