package transport

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/0xmhha/accounts/types"
)

// mockNode starts an httptest JSON-RPC server dispatching by method name.
func mockNode(t *testing.T, handlers map[string]func(params []interface{}) (interface{}, *rpcError)) (*Client, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string        `json:"method"`
			Params []interface{} `json:"params"`
			ID     uint64        `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		resp := map[string]interface{}{"jsonrpc": "2.0", "id": req.ID}
		if h, ok := handlers[req.Method]; ok {
			result, rerr := h(req.Params)
			if rerr != nil {
				resp["error"] = rerr
			} else {
				resp["result"] = result
			}
		} else {
			resp["error"] = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	return Dial(srv.URL), srv.Close
}

func ok(v interface{}) func([]interface{}) (interface{}, *rpcError) {
	return func([]interface{}) (interface{}, *rpcError) { return v, nil }
}

var ctx = context.Background()

func TestChainIDAndQuantities(t *testing.T) {
	c, closeFn := mockNode(t, map[string]func([]interface{}) (interface{}, *rpcError){
		"eth_chainId":              ok("0x205b"),
		"eth_getTransactionCount":  ok("0x5"),
		"eth_gasPrice":             ok("0x3b9aca00"),
		"eth_maxPriorityFeePerGas": ok("0x3b9aca00"),
		"eth_getBalance":           ok("0xde0b6b3a7640000"),
	})
	defer closeFn()

	if id, err := c.ChainID(ctx); err != nil || id.Int64() != 8283 {
		t.Fatalf("ChainID = %v, %v", id, err)
	}
	if n, err := c.Nonce(ctx, types.Address{}); err != nil || n != 5 {
		t.Fatalf("Nonce = %v, %v", n, err)
	}
	if gp, err := c.GasPrice(ctx); err != nil || gp.Int64() != 1000000000 {
		t.Fatalf("GasPrice = %v, %v", gp, err)
	}
	if b, err := c.Balance(ctx, types.Address{}); err != nil || b.String() != "1000000000000000000" {
		t.Fatalf("Balance = %v, %v", b, err)
	}
}

func TestAccountFlags(t *testing.T) {
	// blacklisted (bit 63) + authorized (bit 62)
	c, closeFn := mockNode(t, map[string]func([]interface{}) (interface{}, *rpcError){
		"eth_getProof": ok(map[string]interface{}{"extra": "0xc000000000000000"}),
	})
	defer closeFn()
	f, err := c.AccountFlags(ctx, types.Address{})
	if err != nil {
		t.Fatal(err)
	}
	if !f.Blacklisted || !f.Authorized {
		t.Fatalf("flags = %+v, want both true", f)
	}
}

func TestAccountFlagsMissingExtra(t *testing.T) {
	// extra absent => treated as 0.
	c, closeFn := mockNode(t, map[string]func([]interface{}) (interface{}, *rpcError){
		"eth_getProof": ok(map[string]interface{}{}),
	})
	defer closeFn()
	f, err := c.AccountFlags(ctx, types.Address{})
	if err != nil {
		t.Fatal(err)
	}
	if f.Blacklisted || f.Authorized || f.Raw != 0 {
		t.Fatalf("flags = %+v, want all false/0", f)
	}
}

func TestCodeAndCall(t *testing.T) {
	c, closeFn := mockNode(t, map[string]func([]interface{}) (interface{}, *rpcError){
		"eth_getCode": ok("0x604260005260206000f3"),
		"eth_call":    ok("0x0000000000000000000000000000000000000000000000000000000000000042"),
	})
	defer closeFn()
	code, err := c.Code(ctx, types.Address{})
	if err != nil || len(code) != 10 {
		t.Fatalf("Code len = %d, %v", len(code), err)
	}
	ret, err := c.Call(ctx, CallMsg{}, "latest")
	if err != nil || len(ret) != 32 || ret[31] != 0x42 {
		t.Fatalf("Call = 0x%x, %v", ret, err)
	}
}

func TestSendRawTransaction(t *testing.T) {
	c, closeFn := mockNode(t, map[string]func([]interface{}) (interface{}, *rpcError){
		"eth_sendRawTransaction": ok("0xab00000000000000000000000000000000000000000000000000000000000000"),
	})
	defer closeFn()
	h, err := c.SendRawTransaction(ctx, []byte{0x02, 0x01})
	if err != nil {
		t.Fatal(err)
	}
	if h.Bytes()[0] != 0xab {
		t.Fatalf("hash = %s", h.Hex())
	}
}

func TestRPCError(t *testing.T) {
	c, closeFn := mockNode(t, map[string]func([]interface{}) (interface{}, *rpcError){
		"eth_chainId": func([]interface{}) (interface{}, *rpcError) {
			return nil, &rpcError{Code: -32000, Message: "boom"}
		},
	})
	defer closeFn()
	if _, err := c.ChainID(ctx); err == nil {
		t.Fatal("expected rpc error")
	}
}
