package account

import (
	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/keystore"
	"github.com/0xmhha/accounts/signing"
	"github.com/0xmhha/accounts/types"
)

// Account is a StableNet account backed by a secp256k1 private key. It is the
// primary entry point for creating accounts, signing, and encrypting.
type Account struct {
	priv    *crypto.PrivateKey
	address types.Address
}

// Generate creates a new account with a fresh random private key.
func Generate() (*Account, error) {
	priv, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return FromPrivateKey(priv), nil
}

// FromPrivateKey wraps an existing private key.
func FromPrivateKey(priv *crypto.PrivateKey) *Account {
	return &Account{priv: priv, address: crypto.PrivKeyToAddress(priv)}
}

// FromPrivateKeyBytes builds an account from a 32-byte private key.
func FromPrivateKeyBytes(b []byte) (*Account, error) {
	priv, err := crypto.PrivKeyFromBytes(b)
	if err != nil {
		return nil, err
	}
	return FromPrivateKey(priv), nil
}

// FromKeystore decrypts a keystore-v3 JSON document with the password and
// returns the account.
func FromKeystore(keyjson []byte, password string) (*Account, error) {
	keyBytes, err := keystore.Decrypt(keyjson, password)
	if err != nil {
		return nil, err
	}
	return FromPrivateKeyBytes(keyBytes)
}

// Address returns the account's 20-byte address.
func (a *Account) Address() types.Address { return a.address }

// PrivateKey returns the underlying private key (for use with tx signing).
func (a *Account) PrivateKey() *crypto.PrivateKey { return a.priv }

// PublicKey returns the account's public key.
func (a *Account) PublicKey() *crypto.PublicKey { return a.priv.PubKey() }

// PrivateKeyBytes returns the 32-byte private key. Handle with care.
func (a *Account) PrivateKeyBytes() []byte { return a.priv.Serialize() }

// Sign signs a 32-byte hash using the default signing scheme (secp256k1),
// returning a recoverable signature [R || S || V].
func (a *Account) Sign(hash []byte) ([]byte, error) {
	return signing.Default.Sign(hash, a.priv)
}

// ToKeystore encrypts the account's private key into a keystore-v3 JSON document
// protected by password. Use keystore.StandardScryptN/P for production strength.
func (a *Account) ToKeystore(password string, scryptN, scryptP int) ([]byte, error) {
	return keystore.Encrypt(a.PrivateKeyBytes(), password, scryptN, scryptP)
}

// Decrypt decrypts an ECIES ciphertext addressed to this account's public key.
func (a *Account) Decrypt(ciphertext []byte) ([]byte, error) {
	return crypto.Decrypt(a.priv, ciphertext)
}

// Encrypt encrypts msg to a recipient public key (ECIES). This is a convenience
// wrapper; the recipient decrypts with their own account's Decrypt.
func Encrypt(recipient *crypto.PublicKey, msg []byte) ([]byte, error) {
	return crypto.Encrypt(recipient, msg)
}
