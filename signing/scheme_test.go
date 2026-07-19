package signing

import (
	"encoding/hex"
	"testing"

	"github.com/0xmhha/accounts/crypto"
)

func TestSecp256k1SchemeID(t *testing.T) {
	if (Secp256k1{}).ID() != "secp256k1@1" {
		t.Fatalf("ID = %q", (Secp256k1{}).ID())
	}
}

func TestSchemeSignRecover(t *testing.T) {
	privBytes, _ := hex.DecodeString("4646464646464646464646464646464646464646464646464646464646464646")
	priv, err := crypto.PrivKeyFromBytes(privBytes)
	if err != nil {
		t.Fatal(err)
	}
	var s Scheme = Secp256k1{}
	hash := crypto.Keccak256([]byte("scheme"))
	sig, err := s.Sign(hash, priv)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.Recover(hash, sig)
	if err != nil {
		t.Fatal(err)
	}
	if got != crypto.PrivKeyToAddress(priv) {
		t.Fatalf("recovered %s, want %s", got.Hex(), crypto.PrivKeyToAddress(priv).Hex())
	}
}
