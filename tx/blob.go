package tx

import (
	"math/big"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/internal/rlp"
	"github.com/0xmhha/accounts/types"
)

// BlobTx is an EIP-4844 blob transaction (type 0x03). This implements the
// canonical (no-sidecar) form used for signing and pool identity. Blob data,
// KZG commitments, and proofs (the network "sidecar") are out of scope for the
// signing path — only the blob versioned hashes are committed to by the
// signature.
type BlobTx struct {
	ChainID       *big.Int
	Nonce         uint64
	GasTipCap     *big.Int
	GasFeeCap     *big.Int
	Gas           uint64
	To            types.Address // blob txs cannot create contracts (no nil)
	Value         *big.Int
	Data          []byte
	AccessList    AccessList
	MaxFeePerBlob *big.Int
	BlobHashes    []types.Hash

	V *big.Int // yParity 0/1
	R *big.Int
	S *big.Int
}

func (t *BlobTx) rlpBlobHashes() []byte {
	items := make([][]byte, 0, len(t.BlobHashes))
	for _, h := range t.BlobHashes {
		items = append(items, rlp.EncodeBytes(h.Bytes()))
	}
	return rlp.EncodeList(items...)
}

func (t *BlobTx) coreFields() [][]byte {
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
		rlp.EncodeBig(nz(t.MaxFeePerBlob)),
		t.rlpBlobHashes(),
	}
}

// SigHash returns keccak(0x03 || rlp([chainId, nonce, tipCap, feeCap, gas, to,
// value, data, accessList, maxFeePerBlobGas, blobVersionedHashes])).
func (t *BlobTx) SigHash() []byte {
	return crypto.Keccak256([]byte{BlobTxType}, rlp.EncodeList(t.coreFields()...))
}

// Sign signs and populates yParity/R/S.
func (t *BlobTx) Sign(priv *crypto.PrivateKey) error {
	sig, err := crypto.Sign(t.SigHash(), priv)
	if err != nil {
		return err
	}
	t.R = new(big.Int).SetBytes(sig[0:32])
	t.S = new(big.Int).SetBytes(sig[32:64])
	t.V = new(big.Int).SetInt64(int64(sig[64]))
	return nil
}

// Encode returns the no-sidecar envelope 0x03 || rlp([core..., yParity, r, s]).
func (t *BlobTx) Encode() []byte {
	fields := append(t.coreFields(),
		rlp.EncodeBig(nz(t.V)),
		rlp.EncodeBig(nz(t.R)),
		rlp.EncodeBig(nz(t.S)),
	)
	return append([]byte{BlobTxType}, rlp.EncodeList(fields...)...)
}
