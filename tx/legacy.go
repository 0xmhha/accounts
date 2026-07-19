package tx

import (
	"math/big"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/internal/rlp"
	"github.com/0xmhha/accounts/types"
)

// LegacyTx is an EIP-155 legacy transaction (type 0x00).
type LegacyTx struct {
	Nonce    uint64
	GasPrice *big.Int
	Gas      uint64
	To       *types.Address // nil = contract creation
	Value    *big.Int
	Data     []byte

	// Signature values (set after signing).
	V *big.Int
	R *big.Int
	S *big.Int
}

// SigningHash returns the EIP-155 signing hash for the given chain id:
// keccak(rlp([nonce, gasPrice, gas, to, value, data, chainId, 0, 0])).
func (t *LegacyTx) SigningHash(chainID *big.Int) []byte {
	payload := rlp.EncodeList(
		rlp.EncodeUint(t.Nonce),
		rlp.EncodeBig(nz(t.GasPrice)),
		rlp.EncodeUint(t.Gas),
		rlpAddressPtr(t.To),
		rlp.EncodeBig(nz(t.Value)),
		rlp.EncodeBytes(t.Data),
		rlp.EncodeBig(nz(chainID)),
		rlp.EncodeUint(0),
		rlp.EncodeUint(0),
	)
	return crypto.Keccak256(payload)
}

// Sign signs the transaction with priv over the EIP-155 hash and populates
// V, R, S. V follows EIP-155: recid + 35 + 2*chainId.
func (t *LegacyTx) Sign(chainID *big.Int, priv *crypto.PrivateKey) error {
	sig, err := crypto.Sign(t.SigningHash(chainID), priv)
	if err != nil {
		return err
	}
	t.R = new(big.Int).SetBytes(sig[0:32])
	t.S = new(big.Int).SetBytes(sig[32:64])
	recid := int64(sig[64])
	v := new(big.Int).SetInt64(recid + 35)
	v.Add(v, new(big.Int).Mul(big.NewInt(2), nz(chainID)))
	t.V = v
	return nil
}

// Encode returns the RLP encoding of the signed legacy transaction:
// rlp([nonce, gasPrice, gas, to, value, data, v, r, s]).
func (t *LegacyTx) Encode() []byte {
	return rlp.EncodeList(
		rlp.EncodeUint(t.Nonce),
		rlp.EncodeBig(nz(t.GasPrice)),
		rlp.EncodeUint(t.Gas),
		rlpAddressPtr(t.To),
		rlp.EncodeBig(nz(t.Value)),
		rlp.EncodeBytes(t.Data),
		rlp.EncodeBig(nz(t.V)),
		rlp.EncodeBig(nz(t.R)),
		rlp.EncodeBig(nz(t.S)),
	)
}
