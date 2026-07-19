// Command basic is a copy-paste starting point for building on the accounts
// SDK. It runs fully offline (no node required) and demonstrates the core
// flows: create an account, sign, build every-day transaction types, encrypt a
// key (keystore) and data (ECIES), and compute a CREATE2 address.
//
// Run: go run ./examples/basic
//
// To actually submit the signed transactions to a node, see the transport
// package and cmd/e2e (which sends against a live chainbench network).
package main

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/0xmhha/accounts/account"
	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/keystore"
	"github.com/0xmhha/accounts/tx"
)

const chainID = 8283 // StableNet testnet

func main() {
	// 1. Create accounts.
	alice, err := account.Generate()
	must(err)
	bob, err := account.Generate()
	must(err)
	fmt.Printf("alice = %s\nbob   = %s\n\n", alice.Address(), bob.Address())

	// 2. Sign an arbitrary message hash.
	hash := crypto.Keccak256([]byte("authorize login"))
	sig, err := alice.Sign(hash)
	must(err)
	recovered, _ := crypto.Recover(hash, sig)
	fmt.Printf("message signature recovers to alice: %v\n\n", recovered == alice.Address())

	// 3. Build and sign a standard EIP-1559 (0x02) transfer: alice -> bob.
	to := bob.Address()
	dyn := &tx.DynamicFeeTx{
		ChainID:   big.NewInt(chainID),
		Nonce:     0,
		GasTipCap: big.NewInt(1_000_000_000),
		GasFeeCap: big.NewInt(20_000_000_000),
		Gas:       21000,
		To:        &to,
		Value:     oneCoin(),
	}
	must(tx.GuardValueTransfer(dyn.To, dyn.Value)) // reject unsafe targets
	must(dyn.Sign(alice.PrivateKey()))
	fmt.Printf("0x02 raw tx: 0x%s\n\n", hex.EncodeToString(dyn.Encode()))

	// 4. Build and sign a fee-delegation (0x16) tx: alice pays value, bob pays gas.
	fd := &tx.FeeDelegateTx{Sender: tx.DynamicFeeTx{
		ChainID:   big.NewInt(chainID),
		Nonce:     0,
		GasTipCap: big.NewInt(1_000_000_000),
		GasFeeCap: big.NewInt(20_000_000_000),
		Gas:       21000,
		To:        &to,
		Value:     oneCoin(),
	}}
	must(fd.Sign(alice.PrivateKey(), bob.PrivateKey())) // sender first, then fee payer
	raw16, err := fd.Encode()
	must(err)
	fmt.Printf("0x16 raw tx: 0x%s\n\n", hex.EncodeToString(raw16))

	// 5. Encrypt the key at rest (keystore v3) and load it back.
	doc, err := alice.ToKeystore("password", keystore.LightScryptN, keystore.LightScryptP)
	must(err)
	loaded, err := account.FromKeystore(doc, "password")
	must(err)
	fmt.Printf("keystore roundtrip ok: %v\n\n", loaded.Address() == alice.Address())

	// 6. Encrypt data to bob's public key (ECIES); only bob can decrypt.
	ciphertext, err := account.Encrypt(bob.PublicKey(), []byte("private note for bob"))
	must(err)
	plaintext, err := bob.Decrypt(ciphertext)
	must(err)
	fmt.Printf("ecies decrypted: %q\n\n", plaintext)

	// 7. Compute a CREATE2 deployment address.
	var salt [32]byte
	addr := tx.CreateAddress2(alice.Address(), salt, []byte{0x60, 0x00})
	fmt.Printf("CREATE2 address: %s\n", addr)

	// To send dyn/fd to a node:
	//   c := transport.Dial("http://127.0.0.1:8505")
	//   nonce, _ := c.Nonce(ctx, alice.Address())
	//   tip, _   := c.MaxPriorityFeePerGas(ctx) // Anzeon-aware; do not guess
	//   ... set fields, Sign, then:
	//   h, _ := c.SendRawTransaction(ctx, dyn.Encode())
}

func oneCoin() *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) }

func must(err error) {
	if err != nil {
		panic(err)
	}
}
