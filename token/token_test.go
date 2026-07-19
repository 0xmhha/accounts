package token

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/accounts/signing"
	"github.com/0xmhha/accounts/types"
)

func TestTransferData(t *testing.T) {
	to, _ := types.HexToAddress("0x3535353535353535353535353535353535353535")
	data, err := TransferData(to, big.NewInt(1000))
	if err != nil {
		t.Fatal(err)
	}
	got := hex.EncodeToString(data)
	if !strings.HasPrefix(got, "a9059cbb") {
		t.Fatalf("transfer selector missing: %s", got)
	}
	if len(data) != 4+32+32 {
		t.Fatalf("transfer calldata len = %d", len(data))
	}
}

func TestApproveData(t *testing.T) {
	sp, _ := types.HexToAddress("0x3535353535353535353535353535353535353535")
	data, _ := ApproveData(sp, big.NewInt(1))
	if !strings.HasPrefix(hex.EncodeToString(data), "095ea7b3") {
		t.Fatal("approve selector")
	}
}

func TestDefaultAdapterAddress(t *testing.T) {
	if DefaultNativeCoinAdapter.Hex() != "0x0000000000000000000000000000000000001000" {
		t.Fatalf("adapter addr = %s", DefaultNativeCoinAdapter.Hex())
	}
}

func TestPermitTypedDataAndCalldata(t *testing.T) {
	owner, _ := types.HexToAddress("0x1111111111111111111111111111111111111111")
	spender, _ := types.HexToAddress("0x2222222222222222222222222222222222222222")
	vc := DefaultNativeCoinAdapter
	td := PermitTypedData(
		signing.TypedDataDomain{Name: "USD", Version: "1", ChainID: big.NewInt(8283), VerifyingContract: &vc},
		owner, spender, big.NewInt(100), big.NewInt(0), big.NewInt(9999999999),
	)
	if _, err := td.Digest(); err != nil {
		t.Fatalf("permit digest: %v", err)
	}
	sig := make([]byte, 65) // dummy signature bytes
	sig[64] = 0
	data, err := PermitData(owner, spender, big.NewInt(100), big.NewInt(9999999999), sig)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hex.EncodeToString(data), "d505accf") {
		t.Fatal("permit selector")
	}
}
