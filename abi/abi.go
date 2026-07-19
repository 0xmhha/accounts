// Package abi is a minimal Ethereum ABI encoder/decoder for calling contracts
// with static-typed arguments (address, uintN, bool, bytesN). It covers the
// ERC-20 / EIP-2612 surface the SDK needs; it is not a full ABI codec.
package abi

import (
	"fmt"
	"math/big"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/types"
)

// Selector returns the 4-byte function selector keccak256(signature)[:4], e.g.
// Selector("transfer(address,uint256)").
func Selector(signature string) []byte {
	return crypto.Keccak256([]byte(signature))[:4]
}

// Pack ABI-encodes a call: selector(signature) || 32-byte-packed static args.
// Supported arg types: types.Address, *big.Int, uint64, int, bool, [32]byte,
// []byte (must be exactly 32 bytes).
func Pack(signature string, args ...interface{}) ([]byte, error) {
	out := Selector(signature)
	for i, a := range args {
		word, err := packArg(a)
		if err != nil {
			return nil, fmt.Errorf("arg %d: %w", i, err)
		}
		out = append(out, word...)
	}
	return out, nil
}

func packArg(v interface{}) ([]byte, error) {
	word := make([]byte, 32)
	switch x := v.(type) {
	case types.Address:
		copy(word[12:], x.Bytes())
	case *big.Int:
		if x.Sign() < 0 {
			return nil, fmt.Errorf("negative integers not supported")
		}
		x.FillBytes(word)
	case uint64:
		new(big.Int).SetUint64(x).FillBytes(word)
	case int:
		if x < 0 {
			return nil, fmt.Errorf("negative int not supported")
		}
		big.NewInt(int64(x)).FillBytes(word)
	case bool:
		if x {
			word[31] = 1
		}
	case [32]byte:
		copy(word, x[:])
	case []byte:
		if len(x) != 32 {
			return nil, fmt.Errorf("only 32-byte []byte supported, got %d", len(x))
		}
		copy(word, x)
	default:
		return nil, fmt.Errorf("unsupported arg type %T", v)
	}
	return word, nil
}

// DecodeUint256 reads a uint256 from the first 32 bytes of data.
func DecodeUint256(data []byte) *big.Int {
	if len(data) < 32 {
		return new(big.Int).SetBytes(data)
	}
	return new(big.Int).SetBytes(data[:32])
}

// DecodeBool reads a bool from the first 32 bytes of data.
func DecodeBool(data []byte) bool {
	return DecodeUint256(data).Sign() != 0
}

// DecodeAddress reads an address from the first 32 bytes of data.
func DecodeAddress(data []byte) types.Address {
	if len(data) < 32 {
		return types.BytesToAddress(data)
	}
	return types.BytesToAddress(data[12:32])
}
