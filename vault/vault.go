// Package vault manages a collection of password-encrypted accounts on top of a
// pluggable storage Backend. Each account is stored as a keystore-v3 document
// keyed by its address. Backends decide WHERE the (already password-encrypted)
// keystores live: a directory (FileBackend), memory (MemoryBackend), or the OS
// keychain (KeychainBackend, darwin). HSM/other secure stores implement the
// same Backend interface.
//
// Spec: docs/adr/ADR-0003.
package vault

import (
	"fmt"
	"strings"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/keystore"
	"github.com/0xmhha/accounts/types"
)

// Backend is a key/value store for keystore documents, keyed by a lowercase
// address id (40 hex chars, no 0x).
type Backend interface {
	Write(id string, data []byte) error
	Read(id string) ([]byte, error)
	List() ([]string, error)
	Delete(id string) error
}

// Vault stores and loads password-encrypted accounts via a Backend.
type Vault struct {
	backend Backend
	scryptN int
	scryptP int
}

// New returns a Vault using production-strength scrypt parameters.
func New(backend Backend) *Vault {
	return &Vault{backend: backend, scryptN: keystore.StandardScryptN, scryptP: keystore.StandardScryptP}
}

// NewWithScrypt returns a Vault with explicit scrypt cost parameters (use
// keystore.LightScryptN/P for tests).
func NewWithScrypt(backend Backend, n, p int) *Vault {
	return &Vault{backend: backend, scryptN: n, scryptP: p}
}

func id(addr types.Address) string {
	return strings.TrimPrefix(addr.Hex(), "0x")
}

// Generate creates a new random account, stores it encrypted, and returns its
// address.
func (v *Vault) Generate(password string) (types.Address, error) {
	a, err := account.Generate()
	if err != nil {
		return types.Address{}, err
	}
	return v.store(a, password)
}

// Import encrypts and stores an existing private key.
func (v *Vault) Import(privKey []byte, password string) (types.Address, error) {
	a, err := account.FromPrivateKeyBytes(privKey)
	if err != nil {
		return types.Address{}, err
	}
	return v.store(a, password)
}

func (v *Vault) store(a *account.Account, password string) (types.Address, error) {
	doc, err := a.ToKeystore(password, v.scryptN, v.scryptP)
	if err != nil {
		return types.Address{}, err
	}
	if err := v.backend.Write(id(a.Address()), doc); err != nil {
		return types.Address{}, err
	}
	return a.Address(), nil
}

// Load reads and decrypts the account at addr.
func (v *Vault) Load(addr types.Address, password string) (*account.Account, error) {
	doc, err := v.backend.Read(id(addr))
	if err != nil {
		return nil, err
	}
	return account.FromKeystore(doc, password)
}

// Has reports whether addr is stored.
func (v *Vault) Has(addr types.Address) (bool, error) {
	if _, err := v.backend.Read(id(addr)); err != nil {
		if err == ErrNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// List returns all stored addresses.
func (v *Vault) List() ([]types.Address, error) {
	ids, err := v.backend.List()
	if err != nil {
		return nil, err
	}
	out := make([]types.Address, 0, len(ids))
	for _, s := range ids {
		a, err := types.HexToAddress(s)
		if err != nil {
			continue // skip non-address entries
		}
		out = append(out, a)
	}
	return out, nil
}

// Delete removes the stored account at addr.
func (v *Vault) Delete(addr types.Address) error {
	return v.backend.Delete(id(addr))
}

// ErrNotFound is returned by a Backend when an id is absent.
var ErrNotFound = fmt.Errorf("vault: not found")

// Compile-time checks that all backends satisfy Backend.
var (
	_ Backend = (*MemoryBackend)(nil)
	_ Backend = (*FileBackend)(nil)
	_ Backend = (*KeychainBackend)(nil)
)
