package account

import (
	"bytes"
	"testing"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/keystore"
	"github.com/0xmhha/accounts/types"
)

func TestGenerateAndSign(t *testing.T) {
	a, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	if a.Address() == (types.Address{}) {
		t.Fatal("zero address from Generate")
	}
	hash := crypto.Keccak256([]byte("hello"))
	sig, err := a.Sign(hash)
	if err != nil {
		t.Fatal(err)
	}
	got, err := crypto.Recover(hash, sig)
	if err != nil {
		t.Fatal(err)
	}
	if got != a.Address() {
		t.Fatalf("recovered %s, want %s", got.Hex(), a.Address().Hex())
	}
}

func TestFromPrivateKeyBytesDeterministic(t *testing.T) {
	priv := bytes.Repeat([]byte{0x11}, 32)
	a, err := FromPrivateKeyBytes(priv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.PrivateKeyBytes(), priv) {
		t.Fatal("private key roundtrip mismatch")
	}
}

func TestKeystoreRoundtripViaAccount(t *testing.T) {
	a, _ := Generate()
	doc, err := a.ToKeystore("pw", keystore.LightScryptN, keystore.LightScryptP)
	if err != nil {
		t.Fatal(err)
	}
	b, err := FromKeystore(doc, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if b.Address() != a.Address() {
		t.Fatal("keystore roundtrip changed address")
	}
}

func TestECIESViaAccount(t *testing.T) {
	recipient, _ := Generate()
	msg := []byte("encrypt to account")
	blob, err := Encrypt(recipient.PublicKey(), msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := recipient.Decrypt(blob)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatal("ecies via account mismatch")
	}
}
