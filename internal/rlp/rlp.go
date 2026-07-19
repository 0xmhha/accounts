// Package rlp is a minimal, original RLP (Recursive Length Prefix) encoder
// sufficient for encoding StableNet transaction envelopes and signing
// preimages. It is a clean-room implementation (ADR-0001) — it does not derive
// from any LGPL/GPL source. Only encoding is provided; decoding is not required
// by the SDK's signing paths.
//
// Reference: https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/
package rlp

import "math/big"

// EncodeBytes RLP-encodes a byte string.
func EncodeBytes(b []byte) []byte {
	// Single byte in [0x00, 0x7f] encodes as itself.
	if len(b) == 1 && b[0] <= 0x7f {
		return []byte{b[0]}
	}
	return append(encodeLength(len(b), 0x80), b...)
}

// EncodeUint RLP-encodes an unsigned integer as a minimal big-endian byte
// string (no leading zero bytes). Zero encodes as the empty string (0x80).
func EncodeUint(x uint64) []byte {
	return EncodeBytes(bigEndianMinimal(x))
}

// EncodeBig RLP-encodes a non-negative big.Int as a minimal big-endian byte
// string. A nil or zero value encodes as the empty string (0x80).
func EncodeBig(x *big.Int) []byte {
	if x == nil || x.Sign() == 0 {
		return EncodeBytes(nil)
	}
	return EncodeBytes(x.Bytes())
}

// EncodeList wraps already-encoded items into an RLP list. Each argument MUST
// already be a valid RLP encoding (e.g. from EncodeBytes/EncodeUint/EncodeList).
func EncodeList(items ...[]byte) []byte {
	var payload []byte
	for _, it := range items {
		payload = append(payload, it...)
	}
	return append(encodeLength(len(payload), 0xc0), payload...)
}

// Concat concatenates already-encoded items without a list header. Useful for
// composing a list payload incrementally before a single EncodeListRaw.
func Concat(items ...[]byte) []byte {
	var out []byte
	for _, it := range items {
		out = append(out, it...)
	}
	return out
}

// EncodeListRaw wraps a pre-concatenated RLP payload into a list.
func EncodeListRaw(payload []byte) []byte {
	return append(encodeLength(len(payload), 0xc0), payload...)
}

// encodeLength produces the RLP length prefix for a payload of length n using
// the given base offset (0x80 for strings, 0xc0 for lists).
func encodeLength(n int, offset byte) []byte {
	if n < 56 {
		return []byte{offset + byte(n)}
	}
	lenBytes := bigEndianMinimal(uint64(n))
	prefix := make([]byte, 0, 1+len(lenBytes))
	prefix = append(prefix, offset+55+byte(len(lenBytes)))
	return append(prefix, lenBytes...)
}

// bigEndianMinimal returns the minimal big-endian byte representation of x with
// no leading zero bytes. Zero returns an empty slice.
func bigEndianMinimal(x uint64) []byte {
	if x == 0 {
		return nil
	}
	var buf [8]byte
	i := 8
	for x > 0 {
		i--
		buf[i] = byte(x)
		x >>= 8
	}
	return buf[i:]
}
