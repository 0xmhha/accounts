// Package types defines primitive value types (Address, Hash) shared across the
// SDK. These mirror standard Ethereum sizes; go-stablenet does not diverge here.
package types

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// AddressLength is the length of an account address in bytes.
const AddressLength = 20

// HashLength is the length of a 32-byte hash.
const HashLength = 32

// Address is a 20-byte account address.
type Address [AddressLength]byte

// Hash is a 32-byte hash.
type Hash [HashLength]byte

// BytesToAddress returns an Address from a byte slice, left-truncating or
// left-padding to AddressLength (rightmost bytes are significant).
func BytesToAddress(b []byte) Address {
	var a Address
	if len(b) > AddressLength {
		b = b[len(b)-AddressLength:]
	}
	copy(a[AddressLength-len(b):], b)
	return a
}

// HexToAddress parses a hex string (with optional 0x prefix) into an Address.
func HexToAddress(s string) (Address, error) {
	s = strings.TrimPrefix(s, "0x")
	b, err := hex.DecodeString(s)
	if err != nil {
		return Address{}, fmt.Errorf("invalid address hex: %w", err)
	}
	if len(b) != AddressLength {
		return Address{}, fmt.Errorf("invalid address length %d, want %d", len(b), AddressLength)
	}
	return BytesToAddress(b), nil
}

// Bytes returns the address as a byte slice.
func (a Address) Bytes() []byte { return a[:] }

// Hex returns the lowercase 0x-prefixed hex encoding of the address.
func (a Address) Hex() string { return "0x" + hex.EncodeToString(a[:]) }

// String implements fmt.Stringer.
func (a Address) String() string { return a.Hex() }

// MarshalJSON encodes the address as a 0x-prefixed hex string.
func (a Address) MarshalJSON() ([]byte, error) { return json.Marshal(a.Hex()) }

// Bytes returns the hash as a byte slice.
func (h Hash) Bytes() []byte { return h[:] }

// Hex returns the lowercase 0x-prefixed hex encoding of the hash.
func (h Hash) Hex() string { return "0x" + hex.EncodeToString(h[:]) }

// String implements fmt.Stringer.
func (h Hash) String() string { return h.Hex() }

// BytesToHash returns a Hash from a byte slice (left-padded/truncated).
func BytesToHash(b []byte) Hash {
	var h Hash
	if len(b) > HashLength {
		b = b[len(b)-HashLength:]
	}
	copy(h[HashLength-len(b):], b)
	return h
}
