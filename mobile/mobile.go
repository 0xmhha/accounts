// Package mobile is a gomobile-friendly facade over the SDK, exposing a simple
// API (only string / []byte / int64 / error / bound-pointer signatures) so it
// can be bound to Android (AAR) and iOS (XCFramework) via `gomobile bind`.
//
// Values that don't map to mobile types are passed as strings: addresses,
// private keys, hashes, and signatures are 0x-hex; wei amounts are decimal
// strings. Build instructions: see mobile/README.md.
package mobile

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/hdwallet"
	"github.com/0xmhha/accounts/keystore"
	"github.com/0xmhha/accounts/tx"
	"github.com/0xmhha/accounts/types"
)

// Account is a bound handle to a StableNet account.
type Account struct {
	acct *account.Account
}

// GenerateAccount creates a new random account.
func GenerateAccount() (*Account, error) {
	a, err := account.Generate()
	if err != nil {
		return nil, err
	}
	return &Account{acct: a}, nil
}

// AccountFromPrivateKeyHex loads an account from a 0x-hex private key.
func AccountFromPrivateKeyHex(privHex string) (*Account, error) {
	b, err := hexBytes(privHex)
	if err != nil {
		return nil, err
	}
	a, err := account.FromPrivateKeyBytes(b)
	if err != nil {
		return nil, err
	}
	return &Account{acct: a}, nil
}

// AccountFromKeystore decrypts a keystore-v3 JSON document.
func AccountFromKeystore(keyjson []byte, password string) (*Account, error) {
	a, err := account.FromKeystore(keyjson, password)
	if err != nil {
		return nil, err
	}
	return &Account{acct: a}, nil
}

// DeriveAccount derives an account from a BIP-39 mnemonic at the standard
// Ethereum path m/44'/60'/0'/0/index.
func DeriveAccount(mnemonic, passphrase string, index int) (*Account, error) {
	w, err := hdwallet.FromMnemonic(mnemonic, passphrase)
	if err != nil {
		return nil, err
	}
	a, err := w.DeriveEthereum(uint32(index))
	if err != nil {
		return nil, err
	}
	return &Account{acct: a}, nil
}

// NewMnemonic returns a fresh BIP-39 mnemonic (bits: 128 or 256).
func NewMnemonic(bits int) (string, error) { return hdwallet.NewMnemonic(bits) }

// AddressHex returns the account's 0x-hex address.
func (a *Account) AddressHex() string { return a.acct.Address().Hex() }

// PrivateKeyHex returns the 0x-hex private key. Handle with care.
func (a *Account) PrivateKeyHex() string {
	return "0x" + hex.EncodeToString(a.acct.PrivateKeyBytes())
}

// SignHashHex signs a 32-byte hash (0x-hex) and returns the 0x-hex signature.
func (a *Account) SignHashHex(hashHex string) (string, error) {
	h, err := hexBytes(hashHex)
	if err != nil {
		return "", err
	}
	sig, err := a.acct.Sign(h)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(sig), nil
}

// SignPersonal signs a message with EIP-191 personal_sign; returns 0x-hex sig.
func (a *Account) SignPersonal(msg []byte) (string, error) {
	sig, err := a.acct.SignPersonal(msg)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(sig), nil
}

// ToKeystore encrypts the private key to keystore-v3 JSON (standard scrypt).
func (a *Account) ToKeystore(password string) ([]byte, error) {
	return a.acct.ToKeystore(password, keystore.StandardScryptN, keystore.StandardScryptP)
}

// SignDynamicFeeTransfer builds and signs an EIP-1559 (0x02) transfer and
// returns the 0x-hex raw transaction for submission via any RPC. Amounts are
// decimal wei strings.
func SignDynamicFeeTransfer(privHex string, chainID, nonce, gas int64, tipWei, feeCapWei, toHex, valueWei string) (string, error) {
	a, err := AccountFromPrivateKeyHex(privHex)
	if err != nil {
		return "", err
	}
	to, err := types.HexToAddress(toHex)
	if err != nil {
		return "", err
	}
	tip, err := decBig(tipWei)
	if err != nil {
		return "", err
	}
	feeCap, err := decBig(feeCapWei)
	if err != nil {
		return "", err
	}
	value, err := decBig(valueWei)
	if err != nil {
		return "", err
	}
	t := &tx.DynamicFeeTx{
		ChainID: big.NewInt(chainID), Nonce: uint64(nonce),
		GasTipCap: tip, GasFeeCap: feeCap, Gas: uint64(gas),
		To: &to, Value: value,
	}
	if err := tx.GuardValueTransfer(t.To, t.Value); err != nil {
		return "", err
	}
	if err := t.Sign(a.acct.PrivateKey()); err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(t.Encode()), nil
}

// Keccak256Hex returns the 0x-hex Keccak-256 hash of the input bytes.
func Keccak256Hex(data []byte) string {
	return "0x" + hex.EncodeToString(crypto.Keccak256(data))
}

func hexBytes(s string) ([]byte, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}
	return b, nil
}

func decBig(s string) (*big.Int, error) {
	if s == "" {
		return big.NewInt(0), nil
	}
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimal amount %q", s)
	}
	return n, nil
}
