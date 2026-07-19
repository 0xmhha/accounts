package account_test

import (
	"math/big"
	"testing"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/signing"
)

func TestSignPersonalRoundtrip(t *testing.T) {
	a, _ := account.Generate()
	msg := []byte("sign in to stablenet")
	sig, err := a.SignPersonal(msg)
	if err != nil {
		t.Fatal(err)
	}
	got, err := crypto.Recover(signing.EIP191Hash(msg), sig)
	if err != nil {
		t.Fatal(err)
	}
	if got != a.Address() {
		t.Fatalf("recovered %s, want %s", got.Hex(), a.Address().Hex())
	}
}

func TestSignTypedDataRoundtrip(t *testing.T) {
	a, _ := account.Generate()
	td := &signing.TypedData{
		Types: signing.TypedDataTypes{
			"EIP712Domain": {{Name: "name", Type: "string"}, {Name: "chainId", Type: "uint256"}},
			"Login":        {{Name: "user", Type: "address"}, {Name: "nonce", Type: "uint256"}},
		},
		PrimaryType: "Login",
		Domain:      signing.TypedDataDomain{Name: "StableNet", ChainID: big.NewInt(8283)},
		Message: map[string]interface{}{
			"user":  a.Address(),
			"nonce": big.NewInt(42),
		},
	}
	sig, err := a.SignTypedData(td)
	if err != nil {
		t.Fatal(err)
	}
	digest, _ := td.Digest()
	got, err := crypto.Recover(digest, sig)
	if err != nil {
		t.Fatal(err)
	}
	if got != a.Address() {
		t.Fatalf("recovered %s, want %s", got.Hex(), a.Address().Hex())
	}
}
