package mobile

import (
	"strings"
	"testing"
)

func TestGenerateSignRoundtrip(t *testing.T) {
	a, err := GenerateAccount()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(a.AddressHex(), "0x") || len(a.AddressHex()) != 42 {
		t.Fatalf("bad address %s", a.AddressHex())
	}
	sig, err := a.SignPersonal([]byte("hello"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(sig, "0x") || len(sig) != 132 { // 65 bytes
		t.Fatalf("bad signature %s", sig)
	}
}

func TestDeriveKnownAnswer(t *testing.T) {
	const m = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"
	a, err := DeriveAccount(m, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.AddressHex() != "0x9858effd232b4033e47d90003d41ec34ecaeda94" {
		t.Fatalf("derived %s", a.AddressHex())
	}
}

func TestSignDynamicFeeTransfer(t *testing.T) {
	a, _ := AccountFromPrivateKeyHex("0x4646464646464646464646464646464646464646464646464646464646464646")
	raw, err := SignDynamicFeeTransfer(
		a.PrivateKeyHex(), 8283, 0, 21000,
		"1000000000", "20000000000",
		"0x3535353535353535353535353535353535353535", "1000000000000000000",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(raw, "0x02") {
		t.Fatalf("expected 0x02 tx, got prefix %s", raw[:6])
	}
}

func TestKeystoreRoundtrip(t *testing.T) {
	a, _ := GenerateAccount()
	doc, err := a.ToKeystore("pw")
	if err != nil {
		t.Fatal(err)
	}
	b, err := AccountFromKeystore(doc, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if b.AddressHex() != a.AddressHex() {
		t.Fatal("keystore roundtrip changed address")
	}
}
