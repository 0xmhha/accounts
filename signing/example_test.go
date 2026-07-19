package signing_test

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/0xmhha/accounts/signing"
	"github.com/0xmhha/accounts/types"
)

// Compute the EIP-712 signing digest for typed structured data (the canonical
// "Mail" example).
func ExampleTypedData_Digest() {
	vc, _ := types.HexToAddress("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC")
	cow, _ := types.HexToAddress("0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826")
	bob, _ := types.HexToAddress("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB")

	td := &signing.TypedData{
		Types: signing.TypedDataTypes{
			"EIP712Domain": {
				{Name: "name", Type: "string"}, {Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"}, {Name: "verifyingContract", Type: "address"},
			},
			"Person": {{Name: "name", Type: "string"}, {Name: "wallet", Type: "address"}},
			"Mail":   {{Name: "from", Type: "Person"}, {Name: "to", Type: "Person"}, {Name: "contents", Type: "string"}},
		},
		PrimaryType: "Mail",
		Domain: signing.TypedDataDomain{
			Name: "Ether Mail", Version: "1", ChainID: big.NewInt(1), VerifyingContract: &vc,
		},
		Message: map[string]interface{}{
			"from":     map[string]interface{}{"name": "Cow", "wallet": cow},
			"to":       map[string]interface{}{"name": "Bob", "wallet": bob},
			"contents": "Hello, Bob!",
		},
	}
	digest, _ := td.Digest()
	fmt.Println(hex.EncodeToString(digest))
	// Output: be609aee343fb3c4b28e1df9e632fca64fcfaede20f02e86244efddf30957bd2
}

// Compute the EIP-191 personal_sign digest.
func ExampleEIP191Hash() {
	fmt.Println(hex.EncodeToString(signing.EIP191Hash([]byte("hello"))))
	// Output: 50b2c43fd39106bafbba0da34fc430e1f91e3c96ea2acee2bc34119f92b37750
}
