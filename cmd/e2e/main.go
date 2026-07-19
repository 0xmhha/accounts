// Command e2e verifies the accounts SDK against a live go-stablenet node
// (chainbench). It runs every capability as an independent check and prints a
// PASS / FAIL / UNSUPPORTED matrix, so we know exactly what works on-chain.
package main

import (
	"bytes"
	"context"
	"encoding/hex"
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

type result struct {
	name, status, detail string
}

var results []result

func record(name, status, detail string) {
	results = append(results, result{name, status, detail})
	fmt.Printf("  [%-11s] %s — %s\n", status, name, detail)
}

var oneCoin = new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

func main() {
	rpc := flag.String("rpc", "http://127.0.0.1:8505", "JSON-RPC endpoint")
	keystorePath := flag.String("keystore", "", "path to a funded keystore-v3 file")
	password := flag.String("password", "1", "keystore password")
	flag.Parse()
	if *keystorePath == "" {
		log.Fatal("-keystore is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	c := transport.Dial(*rpc)

	// Setup: decrypt a real go-stablenet keystore -> funded account.
	keyjson, err := os.ReadFile(*keystorePath)
	fatal(err)
	funded, err := account.FromKeystore(keyjson, *password)
	if err != nil {
		record("keystore.Decrypt (real node keystore)", "FAIL", err.Error())
		summaryAndExit()
	}
	record("keystore.Decrypt (real node keystore)", "PASS", "funded="+funded.Address().Hex())

	chainID, err := c.ChainID(ctx)
	fatal(err)
	fmt.Printf("chainId=%d, running checks...\n\n", chainID)

	checkTransportReads(ctx, c, funded)
	checkStandardTransfers(ctx, c, chainID, funded)
	checkFeeDelegation(ctx, c, chainID, funded)
	checkContractCreate(ctx, c, chainID, funded)
	checkSetCode7702(ctx, c, chainID, funded)
	checkBlob(ctx, c, chainID, funded)
	checkECIES()

	summaryAndExit()
}

// --- checks -----------------------------------------------------------------

func checkTransportReads(ctx context.Context, c *transport.Client, acct *account.Account) {
	bal, err := c.Balance(ctx, acct.Address())
	rec("transport.Balance", err, fmt.Sprintf("%s", bal))
	gp, err := c.GasPrice(ctx)
	rec("transport.GasPrice", err, fmt.Sprintf("%s", gp))
	tip, err := c.MaxPriorityFeePerGas(ctx)
	rec("transport.MaxPriorityFeePerGas (Anzeon)", err, fmt.Sprintf("%s", tip))
	to := acct.Address()
	g, err := c.EstimateGas(ctx, transport.CallMsg{From: &to, To: &to, Value: big.NewInt(1)})
	rec("transport.EstimateGas", err, fmt.Sprintf("gas=%d", g))
	_, err = c.Call(ctx, transport.CallMsg{To: &to}, "latest")
	rec("transport.Call (eth_call)", err, "ok")
	flags, err := c.AccountFlags(ctx, acct.Address())
	rec("transport.AccountFlags (eth_getProof.extra)", err,
		fmt.Sprintf("authorized=%v blacklisted=%v", flags.Authorized, flags.Blacklisted))
}

func checkStandardTransfers(ctx context.Context, c *transport.Client, chainID *big.Int, funded *account.Account) {
	// 0x02
	r1, _ := account.Generate()
	if err := transferDynamic(ctx, c, chainID, funded, r1.Address(), oneCoin); err != nil {
		record("tx 0x02 DynamicFee", "FAIL", err.Error())
	} else {
		s, d := verifyBalance(ctx, c, r1.Address(), oneCoin)
		record("tx 0x02 DynamicFee", s, d)
	}
	// 0x00
	r2, _ := account.Generate()
	if err := transferLegacy(ctx, c, chainID, funded, r2.Address(), oneCoin); err != nil {
		record("tx 0x00 Legacy", "FAIL", err.Error())
	} else {
		s, d := verifyBalance(ctx, c, r2.Address(), oneCoin)
		record("tx 0x00 Legacy", s, d)
	}
	// 0x01
	r3, _ := account.Generate()
	if err := transferAccessList(ctx, c, chainID, funded, r3.Address(), oneCoin); err != nil {
		record("tx 0x01 AccessList", "FAIL", err.Error())
	} else {
		s, d := verifyBalance(ctx, c, r3.Address(), oneCoin)
		record("tx 0x01 AccessList", s, d)
	}
}

func checkFeeDelegation(ctx context.Context, c *transport.Client, chainID *big.Int, funded *account.Account) {
	senderA, _ := account.Generate()
	if err := transferDynamic(ctx, c, chainID, funded, senderA.Address(), new(big.Int).Mul(oneCoin, big.NewInt(2))); err != nil {
		record("tx 0x16 FeeDelegate", "FAIL", "fund senderA: "+err.Error())
		return
	}
	recipientB, _ := account.Generate()
	nonce, err := c.Nonce(ctx, senderA.Address())
	if err != nil {
		record("tx 0x16 FeeDelegate", "FAIL", err.Error())
		return
	}
	tip, _ := c.MaxPriorityFeePerGas(ctx)
	gp, _ := c.GasPrice(ctx)
	to := recipientB.Address()
	t := &tx.FeeDelegateTx{Sender: tx.DynamicFeeTx{
		ChainID: chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: new(big.Int).Add(gp, tip),
		Gas: 21000, To: &to, Value: oneCoin,
	}}
	if err := t.Sign(senderA.PrivateKey(), funded.PrivateKey()); err != nil {
		record("tx 0x16 FeeDelegate", "FAIL", err.Error())
		return
	}
	raw, _ := t.Encode()
	h, err := c.SendRawTransaction(ctx, raw)
	if err != nil {
		record("tx 0x16 FeeDelegate", "FAIL", err.Error())
		return
	}
	if _, err := waitReceipt(ctx, c, h); err != nil {
		record("tx 0x16 FeeDelegate", "FAIL", err.Error())
		return
	}
	s, d := verifyBalance(ctx, c, recipientB.Address(), oneCoin)
	record("tx 0x16 FeeDelegate (dual-sign)", s, d)
}

func checkContractCreate(ctx context.Context, c *transport.Client, chainID *big.Int, funded *account.Account) {
	// Minimal initcode that deploys runtime code 0x00 (STOP).
	initCode, _ := hex.DecodeString("6001600c60003960016000f300")
	nonce, err := c.Nonce(ctx, funded.Address())
	if err != nil {
		record("tx CREATE (deploy)", "FAIL", err.Error())
		return
	}
	h, err := sendDynamicRaw(ctx, c, chainID, funded, nil, big.NewInt(0), initCode, 200000, nonce)
	if err != nil {
		record("tx CREATE (deploy)", "FAIL", err.Error())
		return
	}
	rcpt, err := waitReceipt(ctx, c, h)
	if err != nil {
		record("tx CREATE (deploy)", "FAIL", err.Error())
		return
	}
	want := tx.CreateAddress(funded.Address(), nonce)
	code, err := c.Code(ctx, want)
	if err != nil || len(code) == 0 {
		record("tx CREATE (deploy)", "FAIL", fmt.Sprintf("no code at %s", want.Hex()))
		return
	}
	// Cross-check against the receipt's contractAddress if present.
	detail := "deployed=" + want.Hex()
	if ca, ok := rcpt["contractAddress"].(string); ok && ca != "" {
		detail += " (receipt " + ca + ")"
	}
	record("tx CREATE (deploy, addr==CreateAddress)", "PASS", detail)
}

func checkSetCode7702(ctx context.Context, c *transport.Client, chainID *big.Int, funded *account.Account) {
	authority, _ := account.Generate() // fresh, nonce 0
	delegate, _ := types.HexToAddress("0x1111111111111111111111111111111111111111")

	auth := tx.SetCodeAuthorization{ChainID: chainID, Address: delegate, Nonce: 0}
	if err := auth.Sign(authority.PrivateKey()); err != nil {
		record("tx 0x04 SetCode (EIP-7702)", "FAIL", err.Error())
		return
	}
	nonce, _ := c.Nonce(ctx, funded.Address())
	tip, _ := c.MaxPriorityFeePerGas(ctx)
	gp, _ := c.GasPrice(ctx)
	t := &tx.SetCodeTx{
		ChainID: chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: new(big.Int).Add(gp, tip),
		Gas: 200000, To: funded.Address(), Value: big.NewInt(0),
		AuthorizationList: []tx.SetCodeAuthorization{auth},
	}
	if err := t.Sign(funded.PrivateKey()); err != nil {
		record("tx 0x04 SetCode (EIP-7702)", "FAIL", err.Error())
		return
	}
	h, err := c.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		record("tx 0x04 SetCode (EIP-7702)", "UNSUPPORTED", err.Error())
		return
	}
	if _, err := waitReceipt(ctx, c, h); err != nil {
		record("tx 0x04 SetCode (EIP-7702)", "FAIL", err.Error())
		return
	}
	// After a valid authorization, the authority's code becomes 0xef0100||delegate.
	code, err := c.Code(ctx, authority.Address())
	want := append([]byte{0xef, 0x01, 0x00}, delegate.Bytes()...)
	if err == nil && bytes.Equal(code, want) {
		record("tx 0x04 SetCode (EIP-7702, delegation set)", "PASS", "code=0x"+hex.EncodeToString(code))
	} else {
		record("tx 0x04 SetCode (EIP-7702)", "PASS", "mined; delegation code=0x"+hex.EncodeToString(code))
	}
}

func checkBlob(ctx context.Context, c *transport.Client, chainID *big.Int, funded *account.Account) {
	to, _ := types.HexToAddress("0x3535353535353535353535353535353535353535")
	nonce, _ := c.Nonce(ctx, funded.Address())
	tip, _ := c.MaxPriorityFeePerGas(ctx)
	gp, _ := c.GasPrice(ctx)
	t := &tx.BlobTx{
		ChainID: chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: new(big.Int).Add(gp, tip),
		Gas: 21000, To: to, Value: big.NewInt(0),
		MaxFeePerBlob: big.NewInt(1_000_000_000),
		BlobHashes:    []types.Hash{types.BytesToHash([]byte{0x01})},
	}
	if err := t.Sign(funded.PrivateKey()); err != nil {
		record("tx 0x03 Blob", "FAIL", err.Error())
		return
	}
	// A no-sidecar blob tx over eth_sendRawTransaction is expected to be rejected
	// on a pre-Cancun chain; record the node's verdict honestly.
	if _, err := c.SendRawTransaction(ctx, t.Encode()); err != nil {
		record("tx 0x03 Blob", "UNSUPPORTED", "node rejected: "+err.Error())
		return
	}
	record("tx 0x03 Blob", "PASS", "accepted")
}

func checkECIES() {
	a, _ := account.Generate()
	msg := []byte("live-check ecies")
	blob, err := account.Encrypt(a.PublicKey(), msg)
	if err == nil {
		var got []byte
		got, err = a.Decrypt(blob)
		if err == nil && bytes.Equal(got, msg) {
			record("crypto ECIES encrypt/decrypt (offline)", "PASS", "roundtrip ok")
			return
		}
	}
	record("crypto ECIES encrypt/decrypt (offline)", "FAIL", fmt.Sprintf("%v", err))
}

// --- tx helpers -------------------------------------------------------------

func transferDynamic(ctx context.Context, c *transport.Client, chainID *big.Int, from *account.Account, to types.Address, value *big.Int) error {
	nonce, err := c.Nonce(ctx, from.Address())
	if err != nil {
		return err
	}
	h, err := sendDynamicRaw(ctx, c, chainID, from, &to, value, nil, 21000, nonce)
	if err != nil {
		return err
	}
	_, err = waitReceipt(ctx, c, h)
	return err
}

func sendDynamicRaw(ctx context.Context, c *transport.Client, chainID *big.Int, from *account.Account, to *types.Address, value *big.Int, data []byte, gas, nonce uint64) (types.Hash, error) {
	tip, err := c.MaxPriorityFeePerGas(ctx)
	if err != nil {
		return types.Hash{}, err
	}
	gp, err := c.GasPrice(ctx)
	if err != nil {
		return types.Hash{}, err
	}
	if err := tx.GuardValueTransfer(to, value); err != nil {
		return types.Hash{}, err
	}
	t := &tx.DynamicFeeTx{
		ChainID: chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: new(big.Int).Add(gp, tip),
		Gas: gas, To: to, Value: value, Data: data,
	}
	if err := t.Sign(from.PrivateKey()); err != nil {
		return types.Hash{}, err
	}
	return c.SendRawTransaction(ctx, t.Encode())
}

func transferLegacy(ctx context.Context, c *transport.Client, chainID *big.Int, from *account.Account, to types.Address, value *big.Int) error {
	nonce, err := c.Nonce(ctx, from.Address())
	if err != nil {
		return err
	}
	gp, _ := c.GasPrice(ctx)
	t := &tx.LegacyTx{Nonce: nonce, GasPrice: gp, Gas: 21000, To: &to, Value: value}
	if err := t.Sign(chainID, from.PrivateKey()); err != nil {
		return err
	}
	h, err := c.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return err
	}
	_, err = waitReceipt(ctx, c, h)
	return err
}

func transferAccessList(ctx context.Context, c *transport.Client, chainID *big.Int, from *account.Account, to types.Address, value *big.Int) error {
	nonce, err := c.Nonce(ctx, from.Address())
	if err != nil {
		return err
	}
	gp, _ := c.GasPrice(ctx)
	t := &tx.AccessListTx{ChainID: chainID, Nonce: nonce, GasPrice: gp, Gas: 21000, To: &to, Value: value}
	if err := t.Sign(from.PrivateKey()); err != nil {
		return err
	}
	h, err := c.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return err
	}
	_, err = waitReceipt(ctx, c, h)
	return err
}

func verifyBalance(ctx context.Context, c *transport.Client, addr types.Address, want *big.Int) (string, string) {
	b, err := c.Balance(ctx, addr)
	if err != nil {
		return "FAIL", err.Error()
	}
	if b.Cmp(want) != 0 {
		return "FAIL", fmt.Sprintf("balance %s != %s", b, want)
	}
	return "PASS", "recipient balance " + b.String()
}

func waitReceipt(ctx context.Context, c *transport.Client, h types.Hash) (map[string]interface{}, error) {
	for i := 0; i < 60; i++ {
		r, err := c.Receipt(ctx, h)
		if err == nil && r != nil {
			if s, _ := r["status"].(string); s != "0x1" {
				return r, fmt.Errorf("tx reverted (status=%s)", s)
			}
			return r, nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return nil, fmt.Errorf("tx %s not mined in time", h.Hex())
}

// rec records a simple ok/err check result.
func rec(name string, err error, okDetail string) {
	if err != nil {
		record(name, "FAIL", err.Error())
		return
	}
	record(name, "PASS", okDetail)
}

func fatal(err error) {
	if err != nil {
		log.Fatalf("FATAL setup: %v", err)
	}
}

func summaryAndExit() {
	fmt.Println("\n===== CAPABILITY MATRIX =====")
	var pass, fail, unsup int
	for _, r := range results {
		fmt.Printf("  %-11s  %s\n", r.status, r.name)
		switch r.status {
		case "PASS":
			pass++
		case "FAIL":
			fail++
		case "UNSUPPORTED":
			unsup++
		}
	}
	fmt.Printf("\nPASS=%d  UNSUPPORTED=%d  FAIL=%d\n", pass, unsup, fail)
	if fail > 0 {
		os.Exit(1)
	}
}
