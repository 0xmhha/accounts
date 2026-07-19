package tx

import (
	"math/big"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/internal/rlp"
	"github.com/0xmhha/accounts/types"
)

// AccessListTx is an EIP-2930 transaction (type 0x01).
type AccessListTx struct {
	ChainID    *big.Int
	Nonce      uint64
	GasPrice   *big.Int
	Gas        uint64
	To         *types.Address
	Value      *big.Int
	Data       []byte
	AccessList AccessList

	V *big.Int // yParity 0/1
	R *big.Int
	S *big.Int
}

func (t *AccessListTx) coreFields() [][]byte {
	return [][]byte{
		rlp.EncodeBig(nz(t.ChainID)),
		rlp.EncodeUint(t.Nonce),
		rlp.EncodeBig(nz(t.GasPrice)),
		rlp.EncodeUint(t.Gas),
		rlpAddressPtr(t.To),
		rlp.EncodeBig(nz(t.Value)),
		rlp.EncodeBytes(t.Data),
		rlpAccessList(t.AccessList),
	}
}

// SigHash returns keccak(0x01 || rlp([chainId, nonce, gasPrice, gas, to, value, data, accessList])).
func (t *AccessListTx) SigHash() []byte {
	return crypto.Keccak256([]byte{AccessListTxType}, rlp.EncodeList(t.coreFields()...))
}

// Sign signs and populates yParity/R/S.
func (t *AccessListTx) Sign(priv *crypto.PrivateKey) error {
	sig, err := crypto.Sign(t.SigHash(), priv)
	if err != nil {
		return err
	}
	t.R = new(big.Int).SetBytes(sig[0:32])
	t.S = new(big.Int).SetBytes(sig[32:64])
	t.V = new(big.Int).SetInt64(int64(sig[64]))
	return nil
}

// Encode returns 0x01 || rlp([core..., yParity, r, s]).
func (t *AccessListTx) Encode() []byte {
	fields := append(t.coreFields(),
		rlp.EncodeBig(nz(t.V)),
		rlp.EncodeBig(nz(t.R)),
		rlp.EncodeBig(nz(t.S)),
	)
	return append([]byte{AccessListTxType}, rlp.EncodeList(fields...)...)
}
