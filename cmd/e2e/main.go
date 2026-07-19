// Command e2e exercises the accounts SDK end-to-end against a live go-stablenet
// node (e.g. a chainbench network): decrypt a funded keystore, create accounts,
// sign standard (0x02) and fee-delegation (0x16) transactions, submit them, and
// verify on-chain state. This is the authoritative check that the SDK's signing
// matches the node.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/big"
	"os"
	"time"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/transport"
	"github.com/0xmhha/accounts/tx"
	"github.com/0xmhha/accounts/types"
)

func main() {
	rpc := flag.String("rpc", "http://127.0.0.1:8505", "JSON-RPC endpoint")
	keystorePath := flag.String("keystore", "", "path to a funded keystore-v3 file")
	password := flag.String("password", "1", "keystore password")
	flag.Parse()

	if *keystorePath == "" {
		log.Fatal("-keystore is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	keyjson, err := os.ReadFile(*keystorePath)
	must(err)

	// 1. Decrypt a real go-stablenet keystore -> funded account.
	funded, err := account.FromKeystore(keyjson, *password)
	must(err)
	fmt.Printf("[1] keystore decrypted -> funded account %s\n", funded.Address().Hex())

	c := transport.Dial(*rpc)

	chainID, err := c.ChainID(ctx)
	must(err)
	fmt.Printf("[2] chainId = %d\n", chainID)

	bal, err := c.Balance(ctx, funded.Address())
	must(err)
	fmt.Printf("    funded balance = %s\n", bal)

	flags, err := c.AccountFlags(ctx, funded.Address())
	must(err)
	fmt.Printf("    funded Extra flags: authorized=%v blacklisted=%v (raw=%#x)\n",
		flags.Authorized, flags.Blacklisted, flags.Raw)

	// 3. Standard 0x02 transfer: funded -> fresh recipient.
	recipient, _ := account.Generate()
	fmt.Printf("[3] generated recipient %s\n", recipient.Address().Hex())
	oneCoin := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)
	send02(ctx, c, chainID, funded, recipient.Address(), oneCoin)

	rbal, err := c.Balance(ctx, recipient.Address())
	must(err)
	fmt.Printf("    recipient balance after 0x02 = %s\n", rbal)
	if rbal.Cmp(oneCoin) != 0 {
		log.Fatalf("FAIL: recipient balance %s != %s", rbal, oneCoin)
	}
	fmt.Println("    OK: 0x02 transfer confirmed on-chain")

	// 3b. Legacy 0x00 and access-list 0x01 transfers (same pipeline, more types).
	r00 := mustGen()
	send00(ctx, c, chainID, funded, r00, oneCoin)
	assertBalance(ctx, c, r00, oneCoin, "0x00 legacy")
	r01 := mustGen()
	send01(ctx, c, chainID, funded, r01, oneCoin)
	assertBalance(ctx, c, r01, oneCoin, "0x01 access-list")

	// 4. Fee-delegation 0x16: senderA pays value, funded pays gas.
	senderA, _ := account.Generate()
	// fund senderA so it has value to send
	send02(ctx, c, chainID, funded, senderA.Address(), new(big.Int).Mul(oneCoin, big.NewInt(2)))
	fmt.Printf("[4] funded senderA %s for fee-delegation test\n", senderA.Address().Hex())

	recipientB, _ := account.Generate()
	send16(ctx, c, chainID, senderA, funded, recipientB.Address(), oneCoin)
	bbal, err := c.Balance(ctx, recipientB.Address())
	must(err)
	fmt.Printf("    recipientB balance after 0x16 = %s\n", bbal)
	if bbal.Cmp(oneCoin) != 0 {
		log.Fatalf("FAIL: 0x16 recipient balance %s != %s", bbal, oneCoin)
	}
	fmt.Println("    OK: 0x16 fee-delegation transfer confirmed on-chain")

	fmt.Println("\nALL E2E CHECKS PASSED")
}

// send02 builds, signs, submits a 0x02 tx and waits for the receipt.
func send02(ctx context.Context, c *transport.Client, chainID *big.Int, from *account.Account, to types.Address, value *big.Int) {
	nonce, err := c.Nonce(ctx, from.Address())
	must(err)
	tip, err := c.MaxPriorityFeePerGas(ctx)
	must(err)
	gp, err := c.GasPrice(ctx)
	must(err)
	feeCap := new(big.Int).Add(gp, tip)

	t := &tx.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     nonce,
		GasTipCap: tip,
		GasFeeCap: feeCap,
		Gas:       21000,
		To:        &to,
		Value:     value,
	}
	must(tx.GuardValueTransfer(t.To, t.Value))
	must(t.Sign(from.PrivateKey()))
	h, err := c.SendRawTransaction(ctx, t.Encode())
	must(err)
	fmt.Printf("    0x02 sent: %s (nonce %d)\n", h.Hex(), nonce)
	waitReceipt(ctx, c, h)
}

// send16 builds, dual-signs, submits a 0x16 fee-delegation tx and waits.
func send16(ctx context.Context, c *transport.Client, chainID *big.Int, sender, feePayer *account.Account, to types.Address, value *big.Int) {
	nonce, err := c.Nonce(ctx, sender.Address())
	must(err)
	tip, err := c.MaxPriorityFeePerGas(ctx)
	must(err)
	gp, err := c.GasPrice(ctx)
	must(err)
	feeCap := new(big.Int).Add(gp, tip)

	t := &tx.FeeDelegateTx{
		Sender: tx.DynamicFeeTx{
			ChainID:   chainID,
			Nonce:     nonce,
			GasTipCap: tip,
			GasFeeCap: feeCap,
			Gas:       21000,
			To:        &to,
			Value:     value,
		},
	}
	must(t.Sign(sender.PrivateKey(), feePayer.PrivateKey()))
	raw, err := t.Encode()
	must(err)
	h, err := c.SendRawTransaction(ctx, raw)
	must(err)
	fmt.Printf("    0x16 sent: %s (sender %s, feePayer %s)\n", h.Hex(), sender.Address().Hex(), feePayer.Address().Hex())
	waitReceipt(ctx, c, h)
}

// send00 builds, signs, submits an EIP-155 legacy (0x00) tx.
func send00(ctx context.Context, c *transport.Client, chainID *big.Int, from *account.Account, to types.Address, value *big.Int) {
	nonce, err := c.Nonce(ctx, from.Address())
	must(err)
	gp, err := c.GasPrice(ctx)
	must(err)
	t := &tx.LegacyTx{Nonce: nonce, GasPrice: gp, Gas: 21000, To: &to, Value: value}
	must(tx.GuardValueTransfer(t.To, t.Value))
	must(t.Sign(chainID, from.PrivateKey()))
	h, err := c.SendRawTransaction(ctx, t.Encode())
	must(err)
	fmt.Printf("    0x00 sent: %s (nonce %d)\n", h.Hex(), nonce)
	waitReceipt(ctx, c, h)
}

// send01 builds, signs, submits an access-list (0x01) tx.
func send01(ctx context.Context, c *transport.Client, chainID *big.Int, from *account.Account, to types.Address, value *big.Int) {
	nonce, err := c.Nonce(ctx, from.Address())
	must(err)
	gp, err := c.GasPrice(ctx)
	must(err)
	t := &tx.AccessListTx{ChainID: chainID, Nonce: nonce, GasPrice: gp, Gas: 21000, To: &to, Value: value}
	must(tx.GuardValueTransfer(t.To, t.Value))
	must(t.Sign(from.PrivateKey()))
	h, err := c.SendRawTransaction(ctx, t.Encode())
	must(err)
	fmt.Printf("    0x01 sent: %s (nonce %d)\n", h.Hex(), nonce)
	waitReceipt(ctx, c, h)
}

func mustGen() types.Address {
	a, err := account.Generate()
	must(err)
	return a.Address()
}

func assertBalance(ctx context.Context, c *transport.Client, addr types.Address, want *big.Int, label string) {
	b, err := c.Balance(ctx, addr)
	must(err)
	if b.Cmp(want) != 0 {
		log.Fatalf("FAIL: %s recipient balance %s != %s", label, b, want)
	}
	fmt.Printf("    OK: %s transfer confirmed (balance %s)\n", label, b)
}

func waitReceipt(ctx context.Context, c *transport.Client, h types.Hash) {
	for i := 0; i < 60; i++ {
		r, err := c.Receipt(ctx, h)
		if err == nil && r != nil {
			status, _ := r["status"].(string)
			if status != "0x1" {
				log.Fatalf("FAIL: tx %s reverted (status=%s)", h.Hex(), status)
			}
			fmt.Printf("    mined in block %v, status=%s\n", r["blockNumber"], status)
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	log.Fatalf("FAIL: tx %s not mined in time", h.Hex())
}

func must(err error) {
	if err != nil {
		log.Fatalf("FATAL: %v", err)
	}
}
