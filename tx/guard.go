package tx

import (
	"errors"
	"math/big"

	"github.com/0xmhha/accounts/types"
)

// Safety-guard errors mirror the node's rejection reasons so the SDK can fail
// early instead of building a transaction the node will reject (spec
// transactions.md §5).
var (
	ErrZeroAddressTransfer       = errors.New("value transfer to the zero address is rejected")
	ErrValueTransferToPrecompile = errors.New("value transfer to a precompile/native manager is rejected")
)

// Native manager / notable precompile addresses (spec system-contracts.md).
var (
	nativeCoinManager = mustAddr20("0x0000000000000000000000000000000000b00002")
	accountManager    = mustAddr20("0x0000000000000000000000000000000000b00003")
	blsPoP            = mustAddr20("0x0000000000000000000000000000000000b00001")
	p256Verify        = mustAddr20("0x0000000000000000000000000000000000000100")
)

// IsRestrictedTransferTarget reports whether addr is an address that must not
// receive a value transfer under Anzeon: the zero address, a standard precompile
// (0x01..0x0a), the P256VERIFY precompile (0x100), or a native manager.
func IsRestrictedTransferTarget(addr types.Address) bool {
	if addr == (types.Address{}) {
		return true
	}
	// Standard precompiles 0x01..0x0a: first 19 bytes zero, last byte in [1,10].
	allZero := true
	for i := 0; i < 19; i++ {
		if addr[i] != 0 {
			allZero = false
			break
		}
	}
	if allZero && addr[19] >= 1 && addr[19] <= 0x0a {
		return true
	}
	return addr == p256Verify || addr == blsPoP || addr == nativeCoinManager || addr == accountManager
}

// GuardValueTransfer returns an error if sending value to `to` would be rejected
// by the node. A zero value (or nil) is always allowed (e.g. plain calls). A nil
// `to` (contract creation) is allowed.
func GuardValueTransfer(to *types.Address, value *big.Int) error {
	if value == nil || value.Sign() == 0 {
		return nil
	}
	if to == nil {
		return nil // contract creation carries value to the new contract
	}
	if *to == (types.Address{}) {
		return ErrZeroAddressTransfer
	}
	if IsRestrictedTransferTarget(*to) {
		return ErrValueTransferToPrecompile
	}
	return nil
}

func mustAddr20(s string) types.Address {
	a, err := types.HexToAddress(s)
	if err != nil {
		panic(err)
	}
	return a
}
