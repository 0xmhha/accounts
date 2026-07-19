// Package tx builds, signs, and encodes StableNet transactions.
//
// Spec: docs/spec/protocol/v0/transactions.md. All standard Ethereum tx types
// (0x00–0x04) are supported plus the StableNet-specific 0x16 fee-delegation
// transaction. This is a clean-room implementation (ADR-0001): it reproduces
// only the sighash/envelope formulas from the spec and is verified against
// golden vectors sourced from a go-stablenet node.
package tx

import (
	"math/big"

	"github.com/0xmhha/accounts/internal/rlp"
	"github.com/0xmhha/accounts/types"
)

// EIP-2718 transaction type identifiers.
const (
	LegacyTxType                = 0x00
	AccessListTxType            = 0x01
	DynamicFeeTxType            = 0x02
	BlobTxType                  = 0x03
	SetCodeTxType               = 0x04
	FeeDelegateDynamicFeeTxType = 0x16 // StableNet-specific (decimal 22)
)

// AccessTuple is one entry of an EIP-2930 access list.
type AccessTuple struct {
	Address     types.Address
	StorageKeys []types.Hash
}

// AccessList is an EIP-2930 access list.
type AccessList []AccessTuple

// rlpAddressPtr encodes an optional "to" address: nil encodes as the empty
// string (contract creation), otherwise the 20 address bytes.
func rlpAddressPtr(a *types.Address) []byte {
	if a == nil {
		return rlp.EncodeBytes(nil)
	}
	return rlp.EncodeBytes(a.Bytes())
}

// rlpAccessList encodes an access list as rlp([ [addr, [keys...]], ... ]).
func rlpAccessList(al AccessList) []byte {
	items := make([][]byte, 0, len(al))
	for _, t := range al {
		keys := make([][]byte, 0, len(t.StorageKeys))
		for _, k := range t.StorageKeys {
			keys = append(keys, rlp.EncodeBytes(k.Bytes()))
		}
		tuple := rlp.EncodeList(
			rlp.EncodeBytes(t.Address.Bytes()),
			rlp.EncodeList(keys...),
		)
		items = append(items, tuple)
	}
	return rlp.EncodeList(items...)
}

// nz returns a non-nil big.Int (zero if x is nil) to keep RLP encoding total.
func nz(x *big.Int) *big.Int {
	if x == nil {
		return big.NewInt(0)
	}
	return x
}
