// Package transport is a minimal JSON-RPC client for talking to a go-stablenet
// node. It covers exactly the methods the SDK needs (spec:
// docs/spec/protocol/v0/rpc.md), including the StableNet-specific account Extra
// flag exposed via eth_getProof.
package transport

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/types"
)

// Client is a JSON-RPC over HTTP client.
type Client struct {
	url  string
	http *http.Client
	id   atomic.Uint64
}

// Dial returns a client for the given JSON-RPC HTTP endpoint.
func Dial(url string) *Client {
	return &Client{url: url, http: http.DefaultClient}
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      uint64        `json:"id"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

func (c *Client) call(ctx context.Context, out interface{}, method string, params ...interface{}) error {
	if params == nil {
		params = []interface{}{}
	}
	body, err := json.Marshal(rpcRequest{JSONRPC: "2.0", ID: c.id.Add(1), Method: method, Params: params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var rr rpcResponse
	if err := json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return fmt.Errorf("%s: decode response: %w", method, err)
	}
	if rr.Error != nil {
		return fmt.Errorf("%s: rpc error %d: %s", method, rr.Error.Code, rr.Error.Message)
	}
	if out != nil {
		if err := json.Unmarshal(rr.Result, out); err != nil {
			return fmt.Errorf("%s: unmarshal result: %w", method, err)
		}
	}
	return nil
}

// ChainID returns eth_chainId.
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
	var s string
	if err := c.call(ctx, &s, "eth_chainId"); err != nil {
		return nil, err
	}
	return parseQuantity(s)
}

// Nonce returns the pending nonce for addr (eth_getTransactionCount).
func (c *Client) Nonce(ctx context.Context, addr types.Address) (uint64, error) {
	var s string
	if err := c.call(ctx, &s, "eth_getTransactionCount", addr.Hex(), "pending"); err != nil {
		return 0, err
	}
	n, err := parseQuantity(s)
	if err != nil {
		return 0, err
	}
	return n.Uint64(), nil
}

// GasPrice returns eth_gasPrice.
func (c *Client) GasPrice(ctx context.Context) (*big.Int, error) {
	var s string
	if err := c.call(ctx, &s, "eth_gasPrice"); err != nil {
		return nil, err
	}
	return parseQuantity(s)
}

// MaxPriorityFeePerGas returns the suggested tip (Anzeon policy aware). SDKs
// MUST use this rather than guessing a tip (spec params.md §4).
func (c *Client) MaxPriorityFeePerGas(ctx context.Context) (*big.Int, error) {
	var s string
	if err := c.call(ctx, &s, "eth_maxPriorityFeePerGas"); err != nil {
		return nil, err
	}
	return parseQuantity(s)
}

// EstimateGas estimates gas for a call (from/to/value/data as hex).
func (c *Client) EstimateGas(ctx context.Context, msg CallMsg) (uint64, error) {
	var s string
	if err := c.call(ctx, &s, "eth_estimateGas", msg.toMap()); err != nil {
		return 0, err
	}
	g, err := parseQuantity(s)
	if err != nil {
		return 0, err
	}
	return g.Uint64(), nil
}

// Call performs eth_call and returns the raw return data.
func (c *Client) Call(ctx context.Context, msg CallMsg, block string) ([]byte, error) {
	if block == "" {
		block = "latest"
	}
	var s string
	if err := c.call(ctx, &s, "eth_call", msg.toMap(), block); err != nil {
		return nil, err
	}
	return hex.DecodeString(strings.TrimPrefix(s, "0x"))
}

// SendRawTransaction submits a signed raw transaction and returns its hash.
func (c *Client) SendRawTransaction(ctx context.Context, raw []byte) (types.Hash, error) {
	var s string
	if err := c.call(ctx, &s, "eth_sendRawTransaction", "0x"+hex.EncodeToString(raw)); err != nil {
		return types.Hash{}, err
	}
	b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
	if err != nil {
		return types.Hash{}, err
	}
	return types.BytesToHash(b), nil
}

// Balance returns the account balance (eth_getBalance).
func (c *Client) Balance(ctx context.Context, addr types.Address) (*big.Int, error) {
	var s string
	if err := c.call(ctx, &s, "eth_getBalance", addr.Hex(), "latest"); err != nil {
		return nil, err
	}
	return parseQuantity(s)
}

// Receipt returns the transaction receipt as a decoded map (nil if not mined).
func (c *Client) Receipt(ctx context.Context, txHash types.Hash) (map[string]interface{}, error) {
	var out map[string]interface{}
	if err := c.call(ctx, &out, "eth_getTransactionReceipt", txHash.Hex()); err != nil {
		return nil, err
	}
	return out, nil
}

// proofResult is the subset of eth_getProof we need: the StableNet extra flags.
type proofResult struct {
	Extra string `json:"extra"`
}

// AccountFlags reads the StableNet account Extra flags via eth_getProof and
// decodes them (spec account.md §5-A). A missing extra field means 0.
func (c *Client) AccountFlags(ctx context.Context, addr types.Address) (account.Flags, error) {
	var pr proofResult
	if err := c.call(ctx, &pr, "eth_getProof", addr.Hex(), []string{}, "latest"); err != nil {
		return account.Flags{}, err
	}
	var raw uint64
	if pr.Extra != "" {
		q, err := parseQuantity(pr.Extra)
		if err != nil {
			return account.Flags{}, err
		}
		raw = q.Uint64()
	}
	return account.Decode(raw), nil
}

// IsBlacklisted reports whether addr is blacklisted.
func (c *Client) IsBlacklisted(ctx context.Context, addr types.Address) (bool, error) {
	f, err := c.AccountFlags(ctx, addr)
	return f.Blacklisted, err
}

// IsAuthorized reports whether addr is authorized.
func (c *Client) IsAuthorized(ctx context.Context, addr types.Address) (bool, error) {
	f, err := c.AccountFlags(ctx, addr)
	return f.Authorized, err
}
