package crypto

import (
	"bytes"
	"encoding/hex"
	"testing"
)

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func TestKeccak256KnownVectors(t *testing.T) {
	// keccak256("") = c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470
	got := hex.EncodeToString(Keccak256(nil))
	want := "c5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470"
	if got != want {
		t.Fatalf("keccak256(empty) = %s, want %s", got, want)
	}
	// keccak256("abc") = 4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45
	got = hex.EncodeToString(Keccak256([]byte("abc")))
	want = "4e03657aea45a94fc7d47ba826c8d667c0d1e6e33a64a036ec44f58fa12d6c45"
	if got != want {
		t.Fatalf("keccak256(abc) = %s, want %s", got, want)
	}
}

// Known Ethereum private-key -> address vectors.
func TestPrivKeyToAddress(t *testing.T) {
	tests := []struct {
		priv string
		addr string
	}{
		// privkey = 1
		{"0000000000000000000000000000000000000000000000000000000000000001",
			"0x7e5f4552091a69125d5dfcb7b8c2659029395bdf"},
		// EIP-155 example key (sender per the EIP-155 spec example)
		{"4646464646464646464646464646464646464646464646464646464646464646",
			"0x9d8a62f656a8d1615c1294fd71e9cfb3e4855a4f"},
	}
	for _, tt := range tests {
		priv, err := PrivKeyFromBytes(mustHex(tt.priv))
		if err != nil {
			t.Fatal(err)
		}
		got := PrivKeyToAddress(priv).Hex()
		if got != tt.addr {
			t.Fatalf("addr(%s) = %s, want %s", tt.priv, got, tt.addr)
		}
	}
}

func TestSignRecoverRoundtrip(t *testing.T) {
	priv, err := PrivKeyFromBytes(mustHex("4646464646464646464646464646464646464646464646464646464646464646"))
	if err != nil {
		t.Fatal(err)
	}
	hash := Keccak256([]byte("stablenet"))
	sig, err := Sign(hash, priv)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != SignatureLength {
		t.Fatalf("sig len = %d, want %d", len(sig), SignatureLength)
	}
	if sig[64] > 1 {
		t.Fatalf("recovery id = %d, want 0 or 1", sig[64])
	}
	got, err := Recover(hash, sig)
	if err != nil {
		t.Fatal(err)
	}
	if got != PrivKeyToAddress(priv) {
		t.Fatalf("recovered %s, want %s", got.Hex(), PrivKeyToAddress(priv).Hex())
	}
}

func TestSignDeterministic(t *testing.T) {
	priv, _ := PrivKeyFromBytes(mustHex("4646464646464646464646464646464646464646464646464646464646464646"))
	hash := Keccak256([]byte("determinism"))
	a, _ := Sign(hash, priv)
	b, _ := Sign(hash, priv)
	if !bytes.Equal(a, b) {
		t.Fatal("signatures are not deterministic (RFC6979 expected)")
	}
}
