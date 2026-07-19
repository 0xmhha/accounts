package abi

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/0xmhha/accounts/types"
)

func TestSelectorKnownAnswers(t *testing.T) {
	cases := map[string]string{
		"transfer(address,uint256)":             "a9059cbb",
		"balanceOf(address)":                    "70a08231",
		"approve(address,uint256)":              "095ea7b3",
		"allowance(address,address)":            "dd62ed3e",
		"totalSupply()":                         "18160ddd",
		"transferFrom(address,address,uint256)": "23b872dd",
		"permit(address,address,uint256,uint256,uint8,bytes32,bytes32)": "d505accf",
	}
	for sig, want := range cases {
		if got := hex.EncodeToString(Selector(sig)); got != want {
			t.Errorf("Selector(%q) = %s, want %s", sig, got, want)
		}
	}
}

func TestPackTransfer(t *testing.T) {
	to, _ := types.HexToAddress("0x3535353535353535353535353535353535353535")
	data, err := Pack("transfer(address,uint256)", to, big.NewInt(256))
	if err != nil {
		t.Fatal(err)
	}
	want := "a9059cbb" +
		"0000000000000000000000003535353535353535353535353535353535353535" +
		"0000000000000000000000000000000000000000000000000000000000000100"
	if got := hex.EncodeToString(data); got != want {
		t.Fatalf("Pack = %s, want %s", got, want)
	}
}

func TestDecode(t *testing.T) {
	raw, _ := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000042")
	if DecodeUint256(raw).Int64() != 0x42 {
		t.Fatal("DecodeUint256")
	}
	if !DecodeBool(raw) {
		t.Fatal("DecodeBool")
	}
}
