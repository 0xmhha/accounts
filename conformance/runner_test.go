// Package conformance verifies that the SDK reproduces the language-neutral
// golden vectors in vectors/core.json. These vectors are anchored to external
// standards (EIP-155, EIP-1014, EIP-712, EIP-191), so this suite is a
// cross-implementation contract: a future SDK in another language must pass the
// same vectors. Runs offline as part of `go test ./...`.
package conformance

import (
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/hdwallet"
	"github.com/0xmhha/accounts/keystore"
	"github.com/0xmhha/accounts/signing"
	"github.com/0xmhha/accounts/tx"
	"github.com/0xmhha/accounts/types"
)

//go:embed vectors/core.json
var coreJSON []byte

type vectors struct {
	Address []struct {
		PrivKey string `json:"privKey"`
		Address string `json:"address"`
	} `json:"address"`
	Create2 []struct {
		Sender   string `json:"sender"`
		Salt     string `json:"salt"`
		InitCode string `json:"initCode"`
		Address  string `json:"address"`
	} `json:"create2"`
	Extra []struct {
		Raw         string `json:"raw"`
		Blacklisted bool   `json:"blacklisted"`
		Authorized  bool   `json:"authorized"`
	} `json:"extra"`
	EIP191 []struct {
		Message string `json:"message"`
		Digest  string `json:"digest"`
	} `json:"eip191"`
	EIP712MailDigest string `json:"eip712MailDigest"`
	LegacyEip155     struct {
		PrivKey  string `json:"privKey"`
		ChainID  int64  `json:"chainId"`
		Nonce    uint64 `json:"nonce"`
		GasPrice string `json:"gasPrice"`
		Gas      uint64 `json:"gas"`
		To       string `json:"to"`
		Value    string `json:"value"`
		R        string `json:"r"`
		S        string `json:"s"`
		V        int64  `json:"v"`
	} `json:"legacyEip155"`
	Keystore struct {
		JSON     string `json:"json"`
		Password string `json:"password"`
		PrivKey  string `json:"privKey"`
	} `json:"keystore"`
	HDWallet struct {
		Mnemonic   string `json:"mnemonic"`
		Passphrase string `json:"passphrase"`
		Accounts   []struct {
			Index   uint32 `json:"index"`
			Address string `json:"address"`
		} `json:"accounts"`
	} `json:"hdwallet"`
}

func load(t *testing.T) *vectors {
	var v vectors
	if err := json.Unmarshal(coreJSON, &v); err != nil {
		t.Fatal(err)
	}
	return &v
}

func unhex(s string) []byte {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		panic(err)
	}
	return b
}

func TestConformanceAddress(t *testing.T) {
	for _, v := range load(t).Address {
		priv, err := crypto.PrivKeyFromBytes(unhex(v.PrivKey))
		if err != nil {
			t.Fatal(err)
		}
		if got := crypto.PrivKeyToAddress(priv).Hex(); got != v.Address {
			t.Errorf("address(%s) = %s, want %s", v.PrivKey, got, v.Address)
		}
	}
}

func TestConformanceCreate2(t *testing.T) {
	for _, v := range load(t).Create2 {
		sender, _ := types.HexToAddress(v.Sender)
		var salt [32]byte
		copy(salt[:], unhex(v.Salt))
		if got := tx.CreateAddress2(sender, salt, unhex(v.InitCode)).Hex(); got != v.Address {
			t.Errorf("create2(%s) = %s, want %s", v.Sender, got, v.Address)
		}
	}
}

func TestConformanceExtra(t *testing.T) {
	for _, v := range load(t).Extra {
		raw := new(big.Int)
		raw.SetString(strings.TrimPrefix(v.Raw, "0x"), 16)
		f := account.Decode(raw.Uint64())
		if f.Blacklisted != v.Blacklisted || f.Authorized != v.Authorized {
			t.Errorf("extra(%s) = %+v, want bl=%v au=%v", v.Raw, f, v.Blacklisted, v.Authorized)
		}
	}
}

func TestConformanceEIP191(t *testing.T) {
	for _, v := range load(t).EIP191 {
		got := "0x" + hex.EncodeToString(signing.EIP191Hash([]byte(v.Message)))
		if got != v.Digest {
			t.Errorf("eip191(%q) = %s, want %s", v.Message, got, v.Digest)
		}
	}
}

func TestConformanceEIP712Mail(t *testing.T) {
	want := load(t).EIP712MailDigest
	vc, _ := types.HexToAddress("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC")
	cow, _ := types.HexToAddress("0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826")
	bob, _ := types.HexToAddress("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB")
	td := &signing.TypedData{
		Types: signing.TypedDataTypes{
			"EIP712Domain": {{Name: "name", Type: "string"}, {Name: "version", Type: "string"}, {Name: "chainId", Type: "uint256"}, {Name: "verifyingContract", Type: "address"}},
			"Person":       {{Name: "name", Type: "string"}, {Name: "wallet", Type: "address"}},
			"Mail":         {{Name: "from", Type: "Person"}, {Name: "to", Type: "Person"}, {Name: "contents", Type: "string"}},
		},
		PrimaryType: "Mail",
		Domain:      signing.TypedDataDomain{Name: "Ether Mail", Version: "1", ChainID: big.NewInt(1), VerifyingContract: &vc},
		Message: map[string]interface{}{
			"from":     map[string]interface{}{"name": "Cow", "wallet": cow},
			"to":       map[string]interface{}{"name": "Bob", "wallet": bob},
			"contents": "Hello, Bob!",
		},
	}
	d, err := td.Digest()
	if err != nil {
		t.Fatal(err)
	}
	if got := "0x" + hex.EncodeToString(d); got != want {
		t.Errorf("eip712 mail digest = %s, want %s", got, want)
	}
}

func TestConformanceKeystore(t *testing.T) {
	v := load(t).Keystore
	key, err := keystore.Decrypt([]byte(v.JSON), v.Password)
	if err != nil {
		t.Fatal(err)
	}
	if got := "0x" + hex.EncodeToString(key); got != v.PrivKey {
		t.Errorf("keystore decrypt = %s, want %s", got, v.PrivKey)
	}
}

func TestConformanceHDWallet(t *testing.T) {
	v := load(t).HDWallet
	w, err := hdwallet.FromMnemonic(v.Mnemonic, v.Passphrase)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range v.Accounts {
		acct, err := w.DeriveEthereum(a.Index)
		if err != nil {
			t.Fatal(err)
		}
		if acct.Address().Hex() != a.Address {
			t.Errorf("hd derive[%d] = %s, want %s", a.Index, acct.Address().Hex(), a.Address)
		}
	}
}

func TestConformanceLegacyEIP155(t *testing.T) {
	v := load(t).LegacyEip155
	priv, _ := crypto.PrivKeyFromBytes(unhex(v.PrivKey))
	to, _ := types.HexToAddress(v.To)
	gasPrice, _ := new(big.Int).SetString(v.GasPrice, 10)
	value, _ := new(big.Int).SetString(v.Value, 10)
	tr := &tx.LegacyTx{Nonce: v.Nonce, GasPrice: gasPrice, Gas: v.Gas, To: &to, Value: value}
	if err := tr.Sign(big.NewInt(v.ChainID), priv); err != nil {
		t.Fatal(err)
	}
	if got := "0x" + hex.EncodeToString(tr.R.Bytes()); got != v.R {
		t.Errorf("R = %s, want %s", got, v.R)
	}
	if got := "0x" + hex.EncodeToString(tr.S.Bytes()); got != v.S {
		t.Errorf("S = %s, want %s", got, v.S)
	}
	if tr.V.Int64() != v.V {
		t.Errorf("V = %d, want %d", tr.V.Int64(), v.V)
	}
}
