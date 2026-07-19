package hdwallet

import "testing"

// Standard BIP-39/BIP-44 known-answer used across wallets (MetaMask default):
// mnemonic "abandon...about", empty passphrase, m/44'/60'/0'/0/0.
const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

func TestDeriveEthereumKnownAnswer(t *testing.T) {
	w, err := FromMnemonic(testMnemonic, "")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		index uint32
		addr  string
	}{
		{0, "0x9858effd232b4033e47d90003d41ec34ecaeda94"},
		{1, "0x6fac4d18c912343bf86fa7049364dd4e424ab9c0"},
	}
	for _, tt := range tests {
		acct, err := w.DeriveEthereum(tt.index)
		if err != nil {
			t.Fatal(err)
		}
		if got := acct.Address().Hex(); got != tt.addr {
			t.Fatalf("m/44'/60'/0'/0/%d = %s, want %s", tt.index, got, tt.addr)
		}
	}
}

func TestDerivePathAndSign(t *testing.T) {
	w, err := FromMnemonic(testMnemonic, "")
	if err != nil {
		t.Fatal(err)
	}
	a, err := w.Derive("m/44'/60'/0'/0/0")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := w.DeriveEthereum(0)
	if a.Address() != b.Address() {
		t.Fatal("Derive and DeriveEthereum disagree")
	}
	// Derived account is a usable signer.
	if _, err := a.Sign(make([]byte, 32)); err != nil {
		t.Fatal(err)
	}
}

func TestNewMnemonicRoundtrip(t *testing.T) {
	m, err := NewMnemonic(128)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := FromMnemonic(m, ""); err != nil {
		t.Fatalf("generated mnemonic invalid: %v", err)
	}
}

func TestInvalidMnemonic(t *testing.T) {
	if _, err := FromMnemonic("not a valid mnemonic phrase at all", ""); err == nil {
		t.Fatal("expected error for invalid mnemonic")
	}
}
