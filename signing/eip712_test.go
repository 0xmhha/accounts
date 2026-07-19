package signing

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/0xmhha/accounts/types"
)

// Canonical EIP-712 "Mail" example. The known digest verifies our typed-data
// encoding end to end (typeHash, encodeData, domain separator, 0x1901 digest).
func TestEIP712MailDigest(t *testing.T) {
	vc, _ := types.HexToAddress("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC")
	cow, _ := types.HexToAddress("0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826")
	bob, _ := types.HexToAddress("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB")

	td := &TypedData{
		Types: TypedDataTypes{
			"EIP712Domain": {
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Person": {
				{Name: "name", Type: "string"},
				{Name: "wallet", Type: "address"},
			},
			"Mail": {
				{Name: "from", Type: "Person"},
				{Name: "to", Type: "Person"},
				{Name: "contents", Type: "string"},
			},
		},
		PrimaryType: "Mail",
		Domain: TypedDataDomain{
			Name:              "Ether Mail",
			Version:           "1",
			ChainID:           big.NewInt(1),
			VerifyingContract: &vc,
		},
		Message: map[string]interface{}{
			"from":     map[string]interface{}{"name": "Cow", "wallet": cow},
			"to":       map[string]interface{}{"name": "Bob", "wallet": bob},
			"contents": "Hello, Bob!",
		},
	}

	digest, err := td.Digest()
	if err != nil {
		t.Fatal(err)
	}
	want := "be609aee343fb3c4b28e1df9e632fca64fcfaede20f02e86244efddf30957bd2"
	if got := hex.EncodeToString(digest); got != want {
		t.Fatalf("digest = %s, want %s", got, want)
	}
}

func TestEIP712EncodeType(t *testing.T) {
	td := &TypedData{Types: TypedDataTypes{
		"Person": {{Name: "name", Type: "string"}, {Name: "wallet", Type: "address"}},
		"Mail":   {{Name: "from", Type: "Person"}, {Name: "to", Type: "Person"}, {Name: "contents", Type: "string"}},
	}}
	got := td.encodeType("Mail")
	want := "Mail(Person from,Person to,string contents)Person(string name,address wallet)"
	if got != want {
		t.Fatalf("encodeType = %q, want %q", got, want)
	}
}

func TestEIP191Hash(t *testing.T) {
	// keccak256("\x19Ethereum Signed Message:\n5hello")
	got := hex.EncodeToString(EIP191Hash([]byte("hello")))
	// Well-known personal_sign digest for "hello".
	want := "50b2c43fd39106bafbba0da34fc430e1f91e3c96ea2acee2bc34119f92b37750"
	if got != want {
		t.Fatalf("EIP191Hash(hello) = %s, want %s", got, want)
	}
}
