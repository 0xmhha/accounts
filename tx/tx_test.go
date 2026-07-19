package tx

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/types"
)

func mustHex(s string) []byte {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		panic(err)
	}
	return b
}

func mustAddr(s string) types.Address {
	a, err := types.HexToAddress(s)
	if err != nil {
		panic(err)
	}
	return a
}

func key(hexKey string) *crypto.PrivateKey {
	p, err := crypto.PrivKeyFromBytes(mustHex(hexKey))
	if err != nil {
		panic(err)
	}
	return p
}

// EIP-155 published example: known-answer for the full rlp+keccak+sign pipeline.
func TestLegacyEIP155KnownAnswer(t *testing.T) {
	to := mustAddr("0x3535353535353535353535353535353535353535")
	tx := &LegacyTx{
		Nonce:    9,
		GasPrice: big.NewInt(20000000000),
		Gas:      21000,
		To:       &to,
		Value:    new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil), // 1 ether
		Data:     nil,
	}
	priv := key("4646464646464646464646464646464646464646464646464646464646464646")
	if err := tx.Sign(big.NewInt(1), priv); err != nil {
		t.Fatal(err)
	}
	wantR := "28ef61340bd939bc2195fe537567866003e1a15d3c71ff63e1590620aa636276"
	wantS := "67cbe9d8997f761aecb703304b3800ccf555c9f3dc64214b297fb1966a3b6d83"
	if got := hex.EncodeToString(tx.R.Bytes()); got != wantR {
		t.Fatalf("R = %s, want %s", got, wantR)
	}
	if got := hex.EncodeToString(tx.S.Bytes()); got != wantS {
		t.Fatalf("S = %s, want %s", got, wantS)
	}
	if tx.V.Int64() != 37 {
		t.Fatalf("V = %d, want 37", tx.V.Int64())
	}
}

func TestDynamicFeeSignRecover(t *testing.T) {
	to := mustAddr("0x3535353535353535353535353535353535353535")
	priv := key("4646464646464646464646464646464646464646464646464646464646464646")
	tx := &DynamicFeeTx{
		ChainID:   big.NewInt(8282),
		Nonce:     1,
		GasTipCap: big.NewInt(1_000_000_000),
		GasFeeCap: big.NewInt(20_000_000_000),
		Gas:       21000,
		To:        &to,
		Value:     big.NewInt(1000),
	}
	if err := tx.Sign(priv); err != nil {
		t.Fatal(err)
	}
	if tx.V.Int64() > 1 {
		t.Fatalf("yParity = %d, want 0/1", tx.V.Int64())
	}
	// Envelope must start with the 0x02 type byte.
	if enc := tx.Encode(); enc[0] != DynamicFeeTxType {
		t.Fatalf("envelope type = %#x, want 0x02", enc[0])
	}
	// Recover sender from the sighash.
	sig := make([]byte, 65)
	copy(sig[32-len(tx.R.Bytes()):32], tx.R.Bytes())
	copy(sig[64-len(tx.S.Bytes()):64], tx.S.Bytes())
	sig[64] = byte(tx.V.Int64())
	got, err := crypto.Recover(tx.SigHash(), sig)
	if err != nil {
		t.Fatal(err)
	}
	if got != crypto.PrivKeyToAddress(priv) {
		t.Fatalf("recovered %s, want %s", got.Hex(), crypto.PrivKeyToAddress(priv).Hex())
	}
}

func TestFeeDelegateDualSign(t *testing.T) {
	to := mustAddr("0x3535353535353535353535353535353535353535")
	senderKey := key("4646464646464646464646464646464646464646464646464646464646464646")
	feePayerKey := key("0000000000000000000000000000000000000000000000000000000000000001")

	tx := &FeeDelegateTx{
		Sender: DynamicFeeTx{
			ChainID:   big.NewInt(8282),
			Nonce:     7,
			GasTipCap: big.NewInt(1_000_000_000),
			GasFeeCap: big.NewInt(20_000_000_000),
			Gas:       21000,
			To:        &to,
			Value:     big.NewInt(500),
		},
	}
	if err := tx.Sign(senderKey, feePayerKey); err != nil {
		t.Fatal(err)
	}

	// Sender recovers correctly.
	sender, err := tx.SenderAddress()
	if err != nil {
		t.Fatal(err)
	}
	if sender != crypto.PrivKeyToAddress(senderKey) {
		t.Fatalf("sender = %s, want %s", sender.Hex(), crypto.PrivKeyToAddress(senderKey).Hex())
	}

	// Fee payer recovers and matches the declared FeePayer.
	fp, err := tx.RecoverFeePayer()
	if err != nil {
		t.Fatal(err)
	}
	if fp != crypto.PrivKeyToAddress(feePayerKey) {
		t.Fatalf("feePayer = %s, want %s", fp.Hex(), crypto.PrivKeyToAddress(feePayerKey).Hex())
	}

	// Sender and fee-payer sighashes must differ (different preimages).
	fpHash, _ := tx.FeePayerSigHash()
	if hex.EncodeToString(tx.SenderSigHash()) == hex.EncodeToString(fpHash) {
		t.Fatal("sender and fee-payer sighashes must differ")
	}

	// Envelope starts with 0x16.
	enc, err := tx.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if enc[0] != FeeDelegateDynamicFeeTxType {
		t.Fatalf("envelope type = %#x, want 0x16", enc[0])
	}
}

func TestFeePayerRequiresSenderFirst(t *testing.T) {
	tx := &FeeDelegateTx{Sender: DynamicFeeTx{ChainID: big.NewInt(1)}}
	fp := mustAddr("0x0000000000000000000000000000000000000001")
	tx.FeePayer = &fp
	if _, err := tx.FeePayerSigHash(); err == nil {
		t.Fatal("expected error when sender has not signed")
	}
}

func TestCreateAddress(t *testing.T) {
	// go-ethereum known vectors.
	sender := mustAddr("0x6ac7ea33f8831ea9dcc53393aaa88b25a785dbf0")
	tests := []struct {
		nonce uint64
		want  string
	}{
		{0, "0xcd234a471b72ba2f1ccf0a70fcaba648a5eecd8d"},
		{1, "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8"},
	}
	for _, tt := range tests {
		if got := CreateAddress(sender, tt.nonce).Hex(); got != tt.want {
			t.Fatalf("CreateAddress(nonce=%d) = %s, want %s", tt.nonce, got, tt.want)
		}
	}
}

func TestCreateAddress2(t *testing.T) {
	// EIP-1014 known vectors.
	tests := []struct {
		sender string
		salt   string
		init   string
		want   string
	}{
		{"0x0000000000000000000000000000000000000000",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0x00", "0x4d1a2e2bb4f88f0250f26ffff098b0b30b26bf38"},
		{"0xdeadbeef00000000000000000000000000000000",
			"0x0000000000000000000000000000000000000000000000000000000000000000",
			"0x00", "0xb928f69bb1d91cd65274e3c79d8986362984fda3"},
		{"0x00000000000000000000000000000000deadbeef",
			"0x00000000000000000000000000000000000000000000000000000000cafebabe",
			"0xdeadbeef", "0x60f3f640a8508fc6a86d45df051962668e1e8ac7"},
	}
	for _, tt := range tests {
		var salt [32]byte
		copy(salt[:], mustHex(tt.salt))
		got := CreateAddress2(mustAddr(tt.sender), salt, mustHex(tt.init)).Hex()
		if got != tt.want {
			t.Fatalf("CreateAddress2(%s) = %s, want %s", tt.sender, got, tt.want)
		}
	}
}
