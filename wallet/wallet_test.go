package wallet_test

import (
	"context"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/transport"
	"github.com/0xmhha/accounts/wallet"
)

func oneCoin() *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) }

// fakeNode is a minimal JSON-RPC node for wallet tests. blacklisted addresses
// (lowercased hex) report the blacklisted Extra bit; sent raw txs are captured.
type fakeNode struct {
	blacklisted map[string]bool
	lastRaw     string
}

func (f *fakeNode) start(t *testing.T) (*transport.Client, func()) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
			ID     uint64        `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		var result interface{}
		switch req.Method {
		case "eth_chainId":
			result = "0x205b"
		case "eth_getTransactionCount":
			result = "0x0"
		case "eth_gasPrice", "eth_maxPriorityFeePerGas":
			result = "0x3b9aca00"
		case "eth_getProof":
			addr, _ := req.Params[0].(string)
			extra := "0x0"
			if f.blacklisted[strings.ToLower(addr)] {
				extra = "0x8000000000000000"
			}
			result = map[string]interface{}{"extra": extra}
		case "eth_sendRawTransaction":
			f.lastRaw, _ = req.Params[0].(string)
			result = "0xab00000000000000000000000000000000000000000000000000000000000000"
		default:
			result = "0x0"
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	return transport.Dial(srv.URL), srv.Close
}

func TestWalletSendCoin(t *testing.T) {
	f := &fakeNode{blacklisted: map[string]bool{}}
	c, closeFn := f.start(t)
	defer closeFn()

	acct, _ := account.Generate()
	w, err := wallet.New(context.Background(), acct, c)
	if err != nil {
		t.Fatal(err)
	}
	recipient, _ := account.Generate()
	h, err := w.SendCoin(context.Background(), recipient.Address(), oneCoin())
	if err != nil {
		t.Fatal(err)
	}
	if h.Bytes()[0] != 0xab {
		t.Fatalf("unexpected tx hash %s", h.Hex())
	}
	// A signed EIP-1559 (0x02) raw tx must have been submitted.
	if !strings.HasPrefix(f.lastRaw, "0x02") {
		t.Fatalf("submitted raw is not a 0x02 tx: %s", f.lastRaw)
	}
}

func TestWalletBlacklistGuard(t *testing.T) {
	acct, _ := account.Generate()
	recipient, _ := account.Generate()

	f := &fakeNode{blacklisted: map[string]bool{
		strings.ToLower(recipient.Address().Hex()): true,
	}}
	c, closeFn := f.start(t)
	defer closeFn()

	w, err := wallet.New(context.Background(), acct, c)
	if err != nil {
		t.Fatal(err)
	}
	_, err = w.SendCoin(context.Background(), recipient.Address(), oneCoin())
	if err == nil || !strings.Contains(err.Error(), "blacklisted") {
		t.Fatalf("expected blacklist rejection, got %v", err)
	}
	if f.lastRaw != "" {
		t.Fatal("no transaction should have been submitted to a blacklisted recipient")
	}
}
