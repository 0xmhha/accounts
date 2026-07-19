package crypto

import (
	"bytes"
	"testing"
)

func TestECIESRoundtrip(t *testing.T) {
	priv, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("stablenet accounts: encrypt/decrypt with account keys")
	blob, err := Encrypt(priv.PubKey(), msg)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(blob, msg) {
		t.Fatal("plaintext leaked into ciphertext")
	}
	got, err := Decrypt(priv, blob)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("decrypt = %q, want %q", got, msg)
	}
}

func TestECIESWrongKeyFails(t *testing.T) {
	a, _ := GenerateKey()
	b, _ := GenerateKey()
	blob, err := Encrypt(a.PubKey(), []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Decrypt(b, blob); err == nil {
		t.Fatal("expected decryption failure with wrong key")
	}
}

func TestECIESTamperFails(t *testing.T) {
	priv, _ := GenerateKey()
	blob, _ := Encrypt(priv.PubKey(), []byte("secret"))
	blob[len(blob)-1] ^= 0xff // flip a ciphertext/tag byte
	if _, err := Decrypt(priv, blob); err == nil {
		t.Fatal("expected authentication failure on tampered ciphertext")
	}
}
