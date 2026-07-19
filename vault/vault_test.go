package vault

import (
	"testing"

	"github.com/0xmhha/accounts/keystore"
)

func newTestVault(b Backend) *Vault {
	return NewWithScrypt(b, keystore.LightScryptN, keystore.LightScryptP)
}

func runVaultSuite(t *testing.T, backend Backend) {
	t.Helper()
	v := newTestVault(backend)

	addr, err := v.Generate("pw")
	if err != nil {
		t.Fatal(err)
	}
	if has, _ := v.Has(addr); !has {
		t.Fatal("Has should be true after Generate")
	}

	// Load with correct password.
	a, err := v.Load(addr, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if a.Address() != addr {
		t.Fatalf("loaded %s, want %s", a.Address().Hex(), addr.Hex())
	}

	// Wrong password fails.
	if _, err := v.Load(addr, "wrong"); err == nil {
		t.Fatal("expected wrong-password failure")
	}

	// List contains the address.
	list, err := v.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0] != addr {
		t.Fatalf("List = %v, want [%s]", list, addr.Hex())
	}

	// Delete removes it.
	if err := v.Delete(addr); err != nil {
		t.Fatal(err)
	}
	if has, _ := v.Has(addr); has {
		t.Fatal("Has should be false after Delete")
	}
}

func TestVaultMemoryBackend(t *testing.T) {
	runVaultSuite(t, NewMemoryBackend())
}

func TestVaultFileBackend(t *testing.T) {
	b, err := NewFileBackend(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	runVaultSuite(t, b)
}

func TestImportRoundtrip(t *testing.T) {
	priv := make([]byte, 32)
	priv[31] = 0x11
	v := newTestVault(NewMemoryBackend())
	addr, err := v.Import(priv, "pw")
	if err != nil {
		t.Fatal(err)
	}
	a, err := v.Load(addr, "pw")
	if err != nil {
		t.Fatal(err)
	}
	if len(a.PrivateKeyBytes()) != 32 || a.PrivateKeyBytes()[31] != 0x11 {
		t.Fatal("imported key mismatch")
	}
}

func TestBackendNotFound(t *testing.T) {
	b := NewMemoryBackend()
	if _, err := b.Read("missing"); err != ErrNotFound {
		t.Fatalf("Read missing = %v, want ErrNotFound", err)
	}
	if err := b.Delete("missing"); err != ErrNotFound {
		t.Fatalf("Delete missing = %v, want ErrNotFound", err)
	}
}
