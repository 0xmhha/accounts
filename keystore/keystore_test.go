package keystore

import (
	"encoding/hex"
	"testing"
)

// Canonical Web3 Secret Storage (keystore v3) pbkdf2 test vector. Proves our
// decrypt is wire-compatible with go-ethereum/go-stablenet keystores.
func TestDecryptOfficialPBKDF2Vector(t *testing.T) {
	const doc = `{
      "crypto": {
        "cipher": "aes-128-ctr",
        "cipherparams": { "iv": "6087dab2f9fdbbfaddc31a909735c1e6" },
        "ciphertext": "5318b4d5bcd28de64ee5559e671353e16f075ecae9f99c7a79a38af5f869aa46",
        "kdf": "pbkdf2",
        "kdfparams": { "c": 262144, "dklen": 32, "prf": "hmac-sha256", "salt": "ae3cd4e7013836a3df6bd7241b12db061dbe2c6785853cce422d148a624ce0bd" },
        "mac": "517ead924a9d0dc3124507e3393d175ce3ff7c1e96529c6c555ce9e51205e9b2"
      },
      "id": "3198bc9c-6672-5ab3-d995-4942343ae5b6",
      "version": 3
    }`
	key, err := Decrypt([]byte(doc), "testpassword")
	if err != nil {
		t.Fatal(err)
	}
	want := "7a28b5ba57c53603b0b07b56bba752f7784bf506fa95edc395f5cf6c7514fe9d"
	if got := hex.EncodeToString(key); got != want {
		t.Fatalf("decrypted key = %s, want %s", got, want)
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	priv, _ := hex.DecodeString("4646464646464646464646464646464646464646464646464646464646464646")
	doc, err := Encrypt(priv, "pw-correct", LightScryptN, LightScryptP)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decrypt(doc, "pw-correct")
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(got) != hex.EncodeToString(priv) {
		t.Fatalf("roundtrip mismatch")
	}
}

func TestDecryptWrongPassword(t *testing.T) {
	priv, _ := hex.DecodeString("4646464646464646464646464646464646464646464646464646464646464646")
	doc, _ := Encrypt(priv, "right", LightScryptN, LightScryptP)
	if _, err := Decrypt(doc, "wrong"); err == nil {
		t.Fatal("expected MAC mismatch with wrong password")
	}
}
