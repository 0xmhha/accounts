package tx

import (
	"math/big"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/internal/rlp"
	"github.com/0xmhha/accounts/types"
)

// setCodeAuthMagic is the EIP-7702 authorization magic prefix (0x05).
const setCodeAuthMagic = 0x05

// SetCodeAuthorization is an EIP-7702 authorization tuple: it delegates an
// account's code to Address, signed by the authority (the account itself).
type SetCodeAuthorization struct {
	ChainID *big.Int // 0 means "valid on any chain"
	Address types.Address
	Nonce   uint64

	V *big.Int // yParity 0/1
	R *big.Int
	S *big.Int
}

// SigningHash returns keccak(0x05 || rlp([chainId, address, nonce])).
func (a *SetCodeAuthorization) SigningHash() []byte {
	payload := rlp.EncodeList(
		rlp.EncodeBig(nz(a.ChainID)),
		rlp.EncodeBytes(a.Address.Bytes()),
		rlp.EncodeUint(a.Nonce),
	)
	return crypto.Keccak256([]byte{setCodeAuthMagic}, payload)
}

// Sign signs the authorization with the authority's key.
func (a *SetCodeAuthorization) Sign(priv *crypto.PrivateKey) error {
	sig, err := crypto.Sign(a.SigningHash(), priv)
	if err != nil {
		return err
	}
	a.R = new(big.Int).SetBytes(sig[0:32])
	a.S = new(big.Int).SetBytes(sig[32:64])
	a.V = new(big.Int).SetInt64(int64(sig[64]))
	return nil
}

// Authority recovers the account that authorized this delegation.
func (a *SetCodeAuthorization) Authority() (types.Address, error) {
	sig, err := sigBytes(a.R, a.S, a.V)
	if err != nil {
		return types.Address{}, err
	}
	return crypto.Recover(a.SigningHash(), sig)
}

func (a *SetCodeAuthorization) rlp() []byte {
	return rlp.EncodeList(
		rlp.EncodeBig(nz(a.ChainID)),
		rlp.EncodeBytes(a.Address.Bytes()),
		rlp.EncodeUint(a.Nonce),
		rlp.EncodeBig(nz(a.V)),
		rlp.EncodeBig(nz(a.R)),
		rlp.EncodeBig(nz(a.S)),
	)
}

// SetCodeTx is an EIP-7702 set-code transaction (type 0x04).
type SetCodeTx struct {
	ChainID           *big.Int
	Nonce             uint64
	GasTipCap         *big.Int
	GasFeeCap         *big.Int
	Gas               uint64
	To                types.Address // no contract creation
	Value             *big.Int
	Data              []byte
	AccessList        AccessList
	AuthorizationList []SetCodeAuthorization

	V *big.Int // yParity 0/1
	R *big.Int
	S *big.Int
}

func (t *SetCodeTx) rlpAuthList() []byte {
	items := make([][]byte, 0, len(t.AuthorizationList))
	for i := range t.AuthorizationList {
		items = append(items, t.AuthorizationList[i].rlp())
	}
	return rlp.EncodeList(items...)
}

func (t *SetCodeTx) coreFields() [][]byte {
	return [][]byte{
		rlp.EncodeBig(nz(t.ChainID)),
		rlp.EncodeUint(t.Nonce),
		rlp.EncodeBig(nz(t.GasTipCap)),
		rlp.EncodeBig(nz(t.GasFeeCap)),
		rlp.EncodeUint(t.Gas),
		rlp.EncodeBytes(t.To.Bytes()),
		rlp.EncodeBig(nz(t.Value)),
		rlp.EncodeBytes(t.Data),
		rlpAccessList(t.AccessList),
		t.rlpAuthList(),
	}
}

// SigHash returns keccak(0x04 || rlp([chainId, nonce, tipCap, feeCap, gas, to,
// value, data, accessList, authorizationList])).
func (t *SetCodeTx) SigHash() []byte {
	return crypto.Keccak256([]byte{SetCodeTxType}, rlp.EncodeList(t.coreFields()...))
}

// Sign signs and populates yParity/R/S.
func (t *SetCodeTx) Sign(priv *crypto.PrivateKey) error {
	sig, err := crypto.Sign(t.SigHash(), priv)
	if err != nil {
		return err
	}
	t.R = new(big.Int).SetBytes(sig[0:32])
	t.S = new(big.Int).SetBytes(sig[32:64])
	t.V = new(big.Int).SetInt64(int64(sig[64]))
	return nil
}

// Encode returns 0x04 || rlp([core..., yParity, r, s]).
func (t *SetCodeTx) Encode() []byte {
	fields := append(t.coreFields(),
		rlp.EncodeBig(nz(t.V)),
		rlp.EncodeBig(nz(t.R)),
		rlp.EncodeBig(nz(t.S)),
	)
	return append([]byte{SetCodeTxType}, rlp.EncodeList(fields...)...)
}
