package tx

import (
	"math/big"
	"testing"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/types"
)

func recoverFromVRS(hash []byte, r, s, v *big.Int) (types.Address, error) {
	sig, err := sigBytes(r, s, v)
	if err != nil {
		return types.Address{}, err
	}
	return crypto.Recover(hash, sig)
}

func TestAccessListTxSignRecover(t *testing.T) {
	to := mustAddr("0x3535353535353535353535353535353535353535")
	priv := key("4646464646464646464646464646464646464646464646464646464646464646")
	tx := &AccessListTx{
		ChainID:  big.NewInt(8282),
		Nonce:    3,
		GasPrice: big.NewInt(20_000_000_000),
		Gas:      21000,
		To:       &to,
		Value:    big.NewInt(1),
		AccessList: AccessList{{
			Address:     to,
			StorageKeys: []types.Hash{types.BytesToHash([]byte{0x01})},
		}},
	}
	if err := tx.Sign(priv); err != nil {
		t.Fatal(err)
	}
	if tx.Encode()[0] != AccessListTxType {
		t.Fatal("type byte 0x01")
	}
	got, err := recoverFromVRS(tx.SigHash(), tx.R, tx.S, tx.V)
	if err != nil {
		t.Fatal(err)
	}
	if got != crypto.PrivKeyToAddress(priv) {
		t.Fatalf("recovered %s", got.Hex())
	}
}

func TestBlobTxSignRecover(t *testing.T) {
	to := mustAddr("0x3535353535353535353535353535353535353535")
	priv := key("4646464646464646464646464646464646464646464646464646464646464646")
	tx := &BlobTx{
		ChainID:       big.NewInt(8282),
		Nonce:         5,
		GasTipCap:     big.NewInt(1_000_000_000),
		GasFeeCap:     big.NewInt(20_000_000_000),
		Gas:           21000,
		To:            to,
		Value:         big.NewInt(0),
		MaxFeePerBlob: big.NewInt(1_000_000_000),
		BlobHashes:    []types.Hash{types.BytesToHash([]byte{0x01, 0x02, 0x03})},
	}
	if err := tx.Sign(priv); err != nil {
		t.Fatal(err)
	}
	if tx.Encode()[0] != BlobTxType {
		t.Fatal("type byte 0x03")
	}
	got, err := recoverFromVRS(tx.SigHash(), tx.R, tx.S, tx.V)
	if err != nil {
		t.Fatal(err)
	}
	if got != crypto.PrivKeyToAddress(priv) {
		t.Fatalf("recovered %s", got.Hex())
	}
}

func TestSetCodeTxAndAuthorization(t *testing.T) {
	priv := key("4646464646464646464646464646464646464646464646464646464646464646")
	authKey := key("0000000000000000000000000000000000000000000000000000000000000001")
	delegate := mustAddr("0x1111111111111111111111111111111111111111")
	to := mustAddr("0x3535353535353535353535353535353535353535")

	auth := SetCodeAuthorization{ChainID: big.NewInt(8282), Address: delegate, Nonce: 0}
	if err := auth.Sign(authKey); err != nil {
		t.Fatal(err)
	}
	gotAuth, err := auth.Authority()
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != crypto.PrivKeyToAddress(authKey) {
		t.Fatalf("authority = %s, want %s", gotAuth.Hex(), crypto.PrivKeyToAddress(authKey).Hex())
	}

	tx := &SetCodeTx{
		ChainID:           big.NewInt(8282),
		Nonce:             2,
		GasTipCap:         big.NewInt(1_000_000_000),
		GasFeeCap:         big.NewInt(20_000_000_000),
		Gas:               50000,
		To:                to,
		Value:             big.NewInt(0),
		AuthorizationList: []SetCodeAuthorization{auth},
	}
	if err := tx.Sign(priv); err != nil {
		t.Fatal(err)
	}
	if tx.Encode()[0] != SetCodeTxType {
		t.Fatal("type byte 0x04")
	}
	got, err := recoverFromVRS(tx.SigHash(), tx.R, tx.S, tx.V)
	if err != nil {
		t.Fatal(err)
	}
	if got != crypto.PrivKeyToAddress(priv) {
		t.Fatalf("recovered %s", got.Hex())
	}
}
