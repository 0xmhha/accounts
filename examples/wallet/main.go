// Command wallet shows the high-level facade: connect to a node, send coin with
// auto nonce/gas/tip + blacklist guard, deploy, and sign a login message. It
// needs a running go-stablenet node (e.g. chainbench).
//
// Run: go run ./examples/wallet -rpc http://127.0.0.1:8505 -key <hex-private-key>
package main

import (
	"context"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/transport"
	"github.com/0xmhha/accounts/wallet"
)

func main() {
	rpc := flag.String("rpc", "http://127.0.0.1:8505", "JSON-RPC endpoint")
	keyHex := flag.String("key", "", "funded private key (hex)")
	flag.Parse()
	if *keyHex == "" {
		log.Fatal("-key (funded private key hex) is required")
	}
	keyBytes, err := hex.DecodeString(*keyHex)
	must(err)
	acct, err := account.FromPrivateKeyBytes(keyBytes)
	must(err)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	w, err := wallet.New(ctx, acct, transport.Dial(*rpc))
	must(err)
	fmt.Printf("wallet: %s\n", w.Address())

	// Sign a login message (EIP-191).
	sig, err := acct.SignPersonal([]byte("Login to StableNet"))
	must(err)
	fmt.Printf("login signature: 0x%s\n", hex.EncodeToString(sig))

	// Send 1 coin (auto nonce/gas/tip + blacklist guard).
	recipient, _ := account.Generate()
	oneCoin := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	h, err := w.SendCoin(ctx, recipient.Address(), oneCoin)
	must(err)
	fmt.Printf("sent 1 coin to %s: %s\n", recipient.Address(), h)
}

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
