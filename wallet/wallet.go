// Package wallet is a high-level convenience layer over account + transport. It
// auto-fills nonce, gas price, and the Anzeon-aware tip, applies safety guards
// (including an on-chain blacklist pre-check), signs, and submits transactions.
//
// Lower layers (account, tx, transport) remain available for full control.
package wallet

import (
	"context"
	"fmt"
	"math/big"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/transport"
	"github.com/0xmhha/accounts/tx"
	"github.com/0xmhha/accounts/types"
)

// Wallet binds an account to a node connection.
type Wallet struct {
	Account *account.Account
	Client  *transport.Client
	chainID *big.Int
}

// New creates a Wallet and fetches the chain id from the node.
func New(ctx context.Context, acct *account.Account, client *transport.Client) (*Wallet, error) {
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch chainId: %w", err)
	}
	return &Wallet{Account: acct, Client: client, chainID: chainID}, nil
}

// Address returns the wallet's account address.
func (w *Wallet) Address() types.Address { return w.Account.Address() }

// fees returns (gasTipCap, gasFeeCap) using the node's oracles. The tip comes
// from eth_maxPriorityFeePerGas (Anzeon enforces a governance tip for
// unauthorized accounts, so the SDK must not guess).
func (w *Wallet) fees(ctx context.Context) (*big.Int, *big.Int, error) {
	tip, err := w.Client.MaxPriorityFeePerGas(ctx)
	if err != nil {
		return nil, nil, err
	}
	gp, err := w.Client.GasPrice(ctx)
	if err != nil {
		return nil, nil, err
	}
	return tip, new(big.Int).Add(gp, tip), nil
}

// guardTransfer applies the static value-transfer guard and an on-chain
// blacklist pre-check on sender and recipient.
func (w *Wallet) guardTransfer(ctx context.Context, to *types.Address, value *big.Int) error {
	if err := tx.GuardValueTransfer(to, value); err != nil {
		return err
	}
	if bl, err := w.Client.IsBlacklisted(ctx, w.Account.Address()); err != nil {
		return fmt.Errorf("blacklist check (sender): %w", err)
	} else if bl {
		return fmt.Errorf("sender %s is blacklisted", w.Account.Address().Hex())
	}
	if to != nil {
		if bl, err := w.Client.IsBlacklisted(ctx, *to); err != nil {
			return fmt.Errorf("blacklist check (recipient): %w", err)
		} else if bl {
			return fmt.Errorf("recipient %s is blacklisted", to.Hex())
		}
	}
	return nil
}

// SendCoin transfers value (in base-coin wei) to `to` using an EIP-1559 (0x02)
// transaction, auto-filling nonce, gas, and fees.
func (w *Wallet) SendCoin(ctx context.Context, to types.Address, value *big.Int) (types.Hash, error) {
	if err := w.guardTransfer(ctx, &to, value); err != nil {
		return types.Hash{}, err
	}
	nonce, err := w.Client.Nonce(ctx, w.Account.Address())
	if err != nil {
		return types.Hash{}, err
	}
	tip, feeCap, err := w.fees(ctx)
	if err != nil {
		return types.Hash{}, err
	}
	t := &tx.DynamicFeeTx{
		ChainID: w.chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: feeCap,
		Gas: 21000, To: &to, Value: value,
	}
	if err := t.Sign(w.Account.PrivateKey()); err != nil {
		return types.Hash{}, err
	}
	return w.Client.SendRawTransaction(ctx, t.Encode())
}

// Deploy deploys a contract from initCode and returns the tx hash and the
// deterministic contract address. Gas is estimated, with a safety margin.
func (w *Wallet) Deploy(ctx context.Context, initCode []byte, value *big.Int) (types.Hash, types.Address, error) {
	if value == nil {
		value = big.NewInt(0)
	}
	nonce, err := w.Client.Nonce(ctx, w.Account.Address())
	if err != nil {
		return types.Hash{}, types.Address{}, err
	}
	from := w.Account.Address()
	gas, err := w.Client.EstimateGas(ctx, transport.CallMsg{From: &from, Value: value, Data: initCode})
	if err != nil {
		gas = 1_000_000 // fallback if the node cannot estimate creation
	} else {
		gas = gas * 12 / 10 // +20% margin
	}
	tip, feeCap, err := w.fees(ctx)
	if err != nil {
		return types.Hash{}, types.Address{}, err
	}
	t := &tx.DynamicFeeTx{
		ChainID: w.chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: feeCap,
		Gas: gas, To: nil, Value: value, Data: initCode,
	}
	if err := t.Sign(w.Account.PrivateKey()); err != nil {
		return types.Hash{}, types.Address{}, err
	}
	h, err := w.Client.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return types.Hash{}, types.Address{}, err
	}
	return h, tx.CreateAddress(from, nonce), nil
}

// SendFeeDelegated sends a 0x16 fee-delegation transfer where this wallet is the
// sender (pays value) and feePayer covers gas.
func (w *Wallet) SendFeeDelegated(ctx context.Context, feePayer *account.Account, to types.Address, value *big.Int) (types.Hash, error) {
	if err := w.guardTransfer(ctx, &to, value); err != nil {
		return types.Hash{}, err
	}
	nonce, err := w.Client.Nonce(ctx, w.Account.Address())
	if err != nil {
		return types.Hash{}, err
	}
	tip, feeCap, err := w.fees(ctx)
	if err != nil {
		return types.Hash{}, err
	}
	t := &tx.FeeDelegateTx{Sender: tx.DynamicFeeTx{
		ChainID: w.chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: feeCap,
		Gas: 21000, To: &to, Value: value,
	}}
	if err := t.Sign(w.Account.PrivateKey(), feePayer.PrivateKey()); err != nil {
		return types.Hash{}, err
	}
	raw, err := t.Encode()
	if err != nil {
		return types.Hash{}, err
	}
	return w.Client.SendRawTransaction(ctx, raw)
}

// Call performs a read-only eth_call from this wallet to `to`.
func (w *Wallet) Call(ctx context.Context, to types.Address, data []byte) ([]byte, error) {
	from := w.Account.Address()
	return w.Client.Call(ctx, transport.CallMsg{From: &from, To: &to, Data: data}, "latest")
}

// Execute signs and submits a state-changing contract call (0x02) to `to` with
// the given calldata and value, auto-filling nonce, gas (estimated +20%), and
// fees. Use this to invoke contract methods such as ERC-20 transfer/approve.
func (w *Wallet) Execute(ctx context.Context, to types.Address, data []byte, value *big.Int) (types.Hash, error) {
	if value == nil {
		value = big.NewInt(0)
	}
	if err := w.guardTransfer(ctx, &to, value); err != nil {
		return types.Hash{}, err
	}
	nonce, err := w.Client.Nonce(ctx, w.Account.Address())
	if err != nil {
		return types.Hash{}, err
	}
	from := w.Account.Address()
	gas, err := w.Client.EstimateGas(ctx, transport.CallMsg{From: &from, To: &to, Value: value, Data: data})
	if err != nil {
		gas = 200000
	} else {
		gas = gas * 12 / 10
	}
	tip, feeCap, err := w.fees(ctx)
	if err != nil {
		return types.Hash{}, err
	}
	t := &tx.DynamicFeeTx{
		ChainID: w.chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: feeCap,
		Gas: gas, To: &to, Value: value, Data: data,
	}
	if err := t.Sign(w.Account.PrivateKey()); err != nil {
		return types.Hash{}, err
	}
	return w.Client.SendRawTransaction(ctx, t.Encode())
}
