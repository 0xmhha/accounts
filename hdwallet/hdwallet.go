// Package hdwallet provides BIP-39 mnemonic + BIP-32/BIP-44 hierarchical
// deterministic derivation of StableNet (Ethereum-style, secp256k1) accounts.
//
// BIP-39 (mnemonic <-> seed) uses the permissive tyler-smith/go-bip39. BIP-32
// key derivation is implemented here on top of the permissive decred secp256k1
// scalar arithmetic (no LGPL/GPL sources; ADR-0001).
package hdwallet

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/0xmhha/accounts/account"
	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	bip39 "github.com/tyler-smith/go-bip39"
)

const hardened uint32 = 0x80000000

// Wallet is a BIP-32 master key from which accounts are derived.
type Wallet struct {
	seed []byte
	root extKey
}

// NewMnemonic returns a fresh BIP-39 mnemonic with the given entropy bits
// (128 => 12 words, 256 => 24 words).
func NewMnemonic(bits int) (string, error) {
	entropy, err := bip39.NewEntropy(bits)
	if err != nil {
		return "", err
	}
	return bip39.NewMnemonic(entropy)
}

// FromMnemonic builds a wallet from a (validated) BIP-39 mnemonic and optional
// passphrase.
func FromMnemonic(mnemonic, passphrase string) (*Wallet, error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, passphrase)
	if err != nil {
		return nil, fmt.Errorf("invalid mnemonic: %w", err)
	}
	return FromSeed(seed)
}

// FromSeed builds a wallet directly from a BIP-32 seed.
func FromSeed(seed []byte) (*Wallet, error) {
	root, err := masterKey(seed)
	if err != nil {
		return nil, err
	}
	return &Wallet{seed: seed, root: root}, nil
}

// DeriveEthereum derives the account at the standard Ethereum BIP-44 path
// m/44'/60'/0'/0/index.
func (w *Wallet) DeriveEthereum(index uint32) (*account.Account, error) {
	return w.Derive(fmt.Sprintf("m/44'/60'/0'/0/%d", index))
}

// Derive derives the account at an absolute BIP-32 path such as
// "m/44'/60'/0'/0/0".
func (w *Wallet) Derive(path string) (*account.Account, error) {
	indices, err := parsePath(path)
	if err != nil {
		return nil, err
	}
	k := w.root
	for _, idx := range indices {
		k, err = k.child(idx)
		if err != nil {
			return nil, err
		}
	}
	return account.FromPrivateKeyBytes(k.key)
}

// --- BIP-32 ------------------------------------------------------------------

type extKey struct {
	key       []byte // 32-byte private key
	chainCode []byte // 32-byte chain code
}

func masterKey(seed []byte) (extKey, error) {
	if len(seed) < 16 {
		return extKey{}, errors.New("seed too short")
	}
	I := hmac512([]byte("Bitcoin seed"), seed)
	k := extKey{key: I[:32], chainCode: I[32:]}
	var s secp256k1.ModNScalar
	if overflow := s.SetByteSlice(k.key); overflow || s.IsZero() {
		return extKey{}, errors.New("invalid master key")
	}
	return k, nil
}

func (k extKey) child(index uint32) (extKey, error) {
	var data []byte
	if index >= hardened {
		data = append([]byte{0x00}, k.key...)
	} else {
		priv := secp256k1.PrivKeyFromBytes(k.key)
		data = priv.PubKey().SerializeCompressed() // 33 bytes
	}
	data = append(data, ser32(index)...)

	I := hmac512(k.chainCode, data)
	il, ir := I[:32], I[32:]

	var ilScalar, parentScalar secp256k1.ModNScalar
	if overflow := ilScalar.SetByteSlice(il); overflow {
		return extKey{}, errors.New("derived IL >= n; pick another index")
	}
	parentScalar.SetByteSlice(k.key)
	ilScalar.Add(&parentScalar)
	if ilScalar.IsZero() {
		return extKey{}, errors.New("derived zero key; pick another index")
	}
	childKey := ilScalar.Bytes()
	return extKey{key: childKey[:], chainCode: ir}, nil
}

func parsePath(path string) ([]uint32, error) {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, "m") {
		return nil, fmt.Errorf("path must start with m: %q", path)
	}
	parts := strings.Split(path, "/")
	out := make([]uint32, 0, len(parts)-1)
	for _, p := range parts[1:] {
		if p == "" {
			continue
		}
		h := strings.HasSuffix(p, "'") || strings.HasSuffix(p, "h")
		p = strings.TrimRight(p, "'h")
		n, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("bad path element %q: %w", p, err)
		}
		idx := uint32(n)
		if h {
			idx += hardened
		}
		out = append(out, idx)
	}
	return out, nil
}

func hmac512(key, data []byte) []byte {
	h := hmac.New(sha512.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func ser32(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}
