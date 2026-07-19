// Package governance provides read-only bindings for StableNet's governance
// system contracts: GovValidator (0x1001), GovMasterMinter (0x1002), and
// GovCouncil (0x1004). Admin/write flows go through on-chain multisig
// governance and are out of the SDK's scope; this package exposes the view
// getters an application needs to reflect validator/minter/blacklist state.
//
// Spec: docs/spec/protocol/v0/system-contracts.md.
package governance

import (
	"context"
	"math/big"

	"github.com/0xmhha/accounts/abi"
	"github.com/0xmhha/accounts/transport"
	"github.com/0xmhha/accounts/types"
)

// Default governance contract addresses (config-overridable at genesis).
var (
	DefaultGovValidator    = mustAddr("0x0000000000000000000000000000000000001001")
	DefaultGovMasterMinter = mustAddr("0x0000000000000000000000000000000000001002")
	DefaultGovCouncil      = mustAddr("0x0000000000000000000000000000000000001004")
)

// Governance binds the governance contracts to a node connection.
type Governance struct {
	Validator    types.Address
	MasterMinter types.Address
	Council      types.Address
	Client       *transport.Client
}

// New returns a Governance bound to the default addresses.
func New(c *transport.Client) *Governance {
	return &Governance{
		Validator:    DefaultGovValidator,
		MasterMinter: DefaultGovMasterMinter,
		Council:      DefaultGovCouncil,
		Client:       c,
	}
}

func (g *Governance) callBool(ctx context.Context, to types.Address, data []byte) (bool, error) {
	ret, err := g.Client.Call(ctx, transport.CallMsg{To: &to, Data: data}, "latest")
	if err != nil {
		return false, err
	}
	return abi.DecodeBool(ret), nil
}

func (g *Governance) callUint(ctx context.Context, to types.Address, data []byte) (*big.Int, error) {
	ret, err := g.Client.Call(ctx, transport.CallMsg{To: &to, Data: data}, "latest")
	if err != nil {
		return nil, err
	}
	return abi.DecodeUint256(ret), nil
}

// IsValidator reports whether addr is in the active validator set.
func (g *Governance) IsValidator(ctx context.Context, addr types.Address) (bool, error) {
	data, err := abi.Pack("isValidator(address)", addr)
	if err != nil {
		return false, err
	}
	return g.callBool(ctx, g.Validator, data)
}

// ValidatorCount returns the number of active validators.
func (g *Governance) ValidatorCount(ctx context.Context) (*big.Int, error) {
	data, _ := abi.Pack("validatorCount()")
	return g.callUint(ctx, g.Validator, data)
}

// GasTipGwei returns the governance-set gas tip (gwei).
func (g *Governance) GasTipGwei(ctx context.Context) (*big.Int, error) {
	data, _ := abi.Pack("getGasTipGwei()")
	return g.callUint(ctx, g.Validator, data)
}

// IsMinter reports whether account is an authorized minter.
func (g *Governance) IsMinter(ctx context.Context, account types.Address) (bool, error) {
	data, err := abi.Pack("getIsMinter(address)", account)
	if err != nil {
		return false, err
	}
	return g.callBool(ctx, g.MasterMinter, data)
}

// MinterAllowance returns a minter's remaining mint allowance.
func (g *Governance) MinterAllowance(ctx context.Context, minter types.Address) (*big.Int, error) {
	data, err := abi.Pack("getMinterAllowance(address)", minter)
	if err != nil {
		return nil, err
	}
	return g.callUint(ctx, g.MasterMinter, data)
}

// MinterCount returns the number of registered minters.
func (g *Governance) MinterCount(ctx context.Context) (*big.Int, error) {
	data, _ := abi.Pack("getMinterCount()")
	return g.callUint(ctx, g.MasterMinter, data)
}

// IsBlacklisted reports whether account is blacklisted (GovCouncil view).
func (g *Governance) IsBlacklisted(ctx context.Context, account types.Address) (bool, error) {
	data, err := abi.Pack("isBlacklisted(address)", account)
	if err != nil {
		return false, err
	}
	return g.callBool(ctx, g.Council, data)
}

// BlacklistCount returns the number of blacklisted accounts.
func (g *Governance) BlacklistCount(ctx context.Context) (*big.Int, error) {
	data, _ := abi.Pack("getBlacklistCount()")
	return g.callUint(ctx, g.Council, data)
}

func mustAddr(s string) types.Address {
	a, err := types.HexToAddress(s)
	if err != nil {
		panic(err)
	}
	return a
}
