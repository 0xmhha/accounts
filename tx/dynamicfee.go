package tx

import (
	"math/big"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/internal/rlp"
	"github.com/0xmhha/accounts/types"
)

// DynamicFeeTx is an EIP-1559 transaction (type 0x02).
type DynamicFeeTx struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int // maxPriorityFeePerGas
	GasFeeCap  *big.Int // maxFeePerGas
	Gas        uint64
	To         *types.Address // nil = contract creation
	Value      *big.Int
	Data       []byte
	AccessList AccessList

	// Signature values (set after signing).
	V *big.Int
	R *big.Int
	S *big.Int
}

// coreFields returns the RLP-encoded field list common to the sighash and the
// signed encoding, excluding the signature: [chainId, nonce, tipCap, feeCap,
// gas, to, value, data, accessList].
func (t *DynamicFeeTx) coreFields() [][]byte {
	return [][]byte{
		rlp.EncodeBig(nz(t.ChainID)),
		rlp.EncodeUint(t.Nonce),
		rlp.EncodeBig(nz(t.GasTipCap)),
		rlp.EncodeBig(nz(t.GasFeeCap)),
		rlp.EncodeUint(t.Gas),
		rlpAddressPtr(t.To),
		rlp.EncodeBig(nz(t.Value)),
		rlp.EncodeBytes(t.Data),
		rlpAccessList(t.AccessList),
	}
}

// SigHash returns the EIP-1559 signing hash (type 0x02):
// keccak(0x02 || rlp([chainId, nonce, tipCap, feeCap, gas, to, value, data, accessList])).
func (t *DynamicFeeTx) SigHash() []byte {
	payload := rlp.EncodeList(t.coreFields()...)
	return crypto.Keccak256([]byte{DynamicFeeTxType}, payload)
}

// Sign signs the transaction and populates V (yParity 0/1), R, S.
func (t *DynamicFeeTx) Sign(priv *crypto.PrivateKey) error {
	sig, err := crypto.Sign(t.SigHash(), priv)
	if err != nil {
		return err
	}
	t.R = new(big.Int).SetBytes(sig[0:32])
	t.S = new(big.Int).SetBytes(sig[32:64])
	t.V = new(big.Int).SetInt64(int64(sig[64]))
	return nil
}

// signedListPayload returns the RLP list payload including signature:
// [core..., v, r, s] (without the list header). Used by both Encode and the
// fee-delegation envelope (where this list is nested).
func (t *DynamicFeeTx) signedListPayload() []byte {
	fields := t.coreFields()
	fields = append(fields,
		rlp.EncodeBig(nz(t.V)),
		rlp.EncodeBig(nz(t.R)),
		rlp.EncodeBig(nz(t.S)),
	)
	return rlp.EncodeList(fields...)
}

// Encode returns the typed-envelope encoding: 0x02 || rlp([core..., v, r, s]).
func (t *DynamicFeeTx) Encode() []byte {
	return append([]byte{DynamicFeeTxType}, t.signedListPayload()...)
}
