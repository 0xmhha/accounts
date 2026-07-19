package tx

import (
	"math/big"
	"testing"
)

func TestGuardValueTransfer(t *testing.T) {
	zero := mustAddr("0x0000000000000000000000000000000000000000")
	precompile := mustAddr("0x0000000000000000000000000000000000000001")
	manager := mustAddr("0x0000000000000000000000000000000000b00003")
	normal := mustAddr("0x3535353535353535353535353535353535353535")

	// value transfers that must be blocked
	if err := GuardValueTransfer(&zero, big.NewInt(1)); err != ErrZeroAddressTransfer {
		t.Fatalf("zero: got %v", err)
	}
	if err := GuardValueTransfer(&precompile, big.NewInt(1)); err != ErrValueTransferToPrecompile {
		t.Fatalf("precompile: got %v", err)
	}
	if err := GuardValueTransfer(&manager, big.NewInt(1)); err != ErrValueTransferToPrecompile {
		t.Fatalf("manager: got %v", err)
	}

	// allowed cases
	if err := GuardValueTransfer(&normal, big.NewInt(1)); err != nil {
		t.Fatalf("normal transfer blocked: %v", err)
	}
	if err := GuardValueTransfer(&zero, big.NewInt(0)); err != nil {
		t.Fatalf("zero-value call blocked: %v", err)
	}
	if err := GuardValueTransfer(nil, big.NewInt(1)); err != nil {
		t.Fatalf("contract creation blocked: %v", err)
	}
}
