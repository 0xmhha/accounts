// Package token provides ERC-20 bindings for StableNet's NativeCoinAdapter (the
// ERC-20 face of the native base coin, default 0x1000) plus EIP-2612 permit.
// Read methods use eth_call; state-changing calls return calldata to submit via
// wallet.Execute.
//
// Spec: docs/spec/protocol/v0/system-contracts.md.
package token

import (
	"context"
	"math/big"

	"github.com/0xmhha/accounts/abi"
	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/signing"
	"github.com/0xmhha/accounts/transport"
	"github.com/0xmhha/accounts/types"
)

// DefaultNativeCoinAdapter is the default NativeCoinAdapter address (0x1000).
// Read the actual address from chain config in production.
var DefaultNativeCoinAdapter = mustAddr("0x0000000000000000000000000000000000001000")

// Token is an ERC-20 (FiatToken) binding.
type Token struct {
	Address types.Address
	Client  *transport.Client
}

// NativeCoinAdapter returns a binding to the default NativeCoinAdapter.
func NativeCoinAdapter(c *transport.Client) *Token {
	return &Token{Address: DefaultNativeCoinAdapter, Client: c}
}

// New returns a binding to a token at addr.
func New(addr types.Address, c *transport.Client) *Token {
	return &Token{Address: addr, Client: c}
}

func (t *Token) callUint(ctx context.Context, data []byte) (*big.Int, error) {
	ret, err := t.Client.Call(ctx, transport.CallMsg{To: &t.Address, Data: data}, "latest")
	if err != nil {
		return nil, err
	}
	return abi.DecodeUint256(ret), nil
}

// BalanceOf returns the token balance of account (for NativeCoinAdapter this is
// the native coin balance).
func (t *Token) BalanceOf(ctx context.Context, account types.Address) (*big.Int, error) {
	data, err := abi.Pack("balanceOf(address)", account)
	if err != nil {
		return nil, err
	}
	return t.callUint(ctx, data)
}

// TotalSupply returns the total token supply.
func (t *Token) TotalSupply(ctx context.Context) (*big.Int, error) {
	data, _ := abi.Pack("totalSupply()")
	return t.callUint(ctx, data)
}

// Allowance returns the spender's allowance over owner's tokens.
func (t *Token) Allowance(ctx context.Context, owner, spender types.Address) (*big.Int, error) {
	data, err := abi.Pack("allowance(address,address)", owner, spender)
	if err != nil {
		return nil, err
	}
	return t.callUint(ctx, data)
}

// DomainSeparator reads the token's EIP-712 DOMAIN_SEPARATOR() (bytes32).
func (t *Token) DomainSeparator(ctx context.Context) ([32]byte, error) {
	data, _ := abi.Pack("DOMAIN_SEPARATOR()")
	ret, err := t.Client.Call(ctx, transport.CallMsg{To: &t.Address, Data: data}, "latest")
	if err != nil {
		return [32]byte{}, err
	}
	return abi.DecodeBytes32(ret), nil
}

// Nonces reads the EIP-2612 permit nonce for owner.
func (t *Token) Nonces(ctx context.Context, owner types.Address) (*big.Int, error) {
	data, err := abi.Pack("nonces(address)", owner)
	if err != nil {
		return nil, err
	}
	return t.callUint(ctx, data)
}

// permitTypeHash = keccak256("Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)").
var permitTypeHash = crypto.Keccak256([]byte("Permit(address owner,address spender,uint256 value,uint256 nonce,uint256 deadline)"))

// PermitDigest computes the EIP-2612 permit signing digest using the token's
// on-chain DOMAIN_SEPARATOR (so no name/version fetch is needed):
// keccak(0x1901 || domainSeparator || keccak(PERMIT_TYPEHASH || owner || spender || value || nonce || deadline)).
func PermitDigest(domainSep [32]byte, owner, spender types.Address, value, nonce, deadline *big.Int) []byte {
	structHash := crypto.Keccak256(
		permitTypeHash,
		word(owner.Bytes()), word(spender.Bytes()),
		bigWord(value), bigWord(nonce), bigWord(deadline),
	)
	return crypto.Keccak256([]byte{0x19, 0x01}, domainSep[:], structHash)
}

func word(b []byte) []byte {
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func bigWord(x *big.Int) []byte {
	out := make([]byte, 32)
	x.FillBytes(out)
	return out
}

// TransferData returns calldata for transfer(to, amount).
func TransferData(to types.Address, amount *big.Int) ([]byte, error) {
	return abi.Pack("transfer(address,uint256)", to, amount)
}

// ApproveData returns calldata for approve(spender, amount).
func ApproveData(spender types.Address, amount *big.Int) ([]byte, error) {
	return abi.Pack("approve(address,uint256)", spender, amount)
}

// PermitData returns calldata for EIP-2612 permit(owner,spender,value,deadline,v,r,s),
// given a recoverable signature [R||S||V] over the permit typed data.
func PermitData(owner, spender types.Address, value, deadline *big.Int, sig []byte) ([]byte, error) {
	var r, s [32]byte
	copy(r[:], sig[0:32])
	copy(s[:], sig[32:64])
	v := uint64(sig[64]) + 27 // EIP-2612 expects 27/28
	return abi.Pack("permit(address,address,uint256,uint256,uint8,bytes32,bytes32)",
		owner, spender, value, deadline, v, r, s)
}

// PermitTypedData builds the EIP-712 typed data an owner signs to authorize a
// permit. The domain (name/version/chainId/verifyingContract) must match the
// token's on-chain EIP-712 domain.
func PermitTypedData(domain signing.TypedDataDomain, owner, spender types.Address, value, nonce, deadline *big.Int) *signing.TypedData {
	return &signing.TypedData{
		Types: signing.TypedDataTypes{
			"EIP712Domain": {
				{Name: "name", Type: "string"}, {Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"}, {Name: "verifyingContract", Type: "address"},
			},
			"Permit": {
				{Name: "owner", Type: "address"}, {Name: "spender", Type: "address"},
				{Name: "value", Type: "uint256"}, {Name: "nonce", Type: "uint256"},
				{Name: "deadline", Type: "uint256"},
			},
		},
		PrimaryType: "Permit",
		Domain:      domain,
		Message: map[string]interface{}{
			"owner": owner, "spender": spender, "value": value,
			"nonce": nonce, "deadline": deadline,
		},
	}
}

func mustAddr(s string) types.Address {
	a, err := types.HexToAddress(s)
	if err != nil {
		panic(err)
	}
	return a
}
