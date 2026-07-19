package tx_test

import (
	"fmt"
	"math/big"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/tx"
	"github.com/0xmhha/accounts/types"
)

func fixedKey() *crypto.PrivateKey {
	p, _ := crypto.PrivKeyFromBytes([]byte{
		0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46,
		0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46, 0x46,
	})
	return p
}

func recoverSender(t *tx.DynamicFeeTx) types.Address {
	sig := make([]byte, 65)
	t.R.FillBytes(sig[0:32])
	t.S.FillBytes(sig[32:64])
	sig[64] = byte(t.V.Int64())
	addr, _ := crypto.Recover(t.SigHash(), sig)
	return addr
}

// Build and sign a standard EIP-1559 (0x02) transfer, then read back the
// encoded envelope type and signer.
func ExampleDynamicFeeTx() {
	to, _ := types.HexToAddress("0x3535353535353535353535353535353535353535")
	t := &tx.DynamicFeeTx{
		ChainID:   big.NewInt(8283), // StableNet testnet
		Nonce:     0,
		GasTipCap: big.NewInt(1_000_000_000),
		GasFeeCap: big.NewInt(20_000_000_000),
		Gas:       21000,
		To:        &to,
		Value:     big.NewInt(1),
	}
	// Reject unsafe targets (zero address / precompiles) before signing.
	if err := tx.GuardValueTransfer(t.To, t.Value); err != nil {
		panic(err)
	}
	_ = t.Sign(fixedKey())

	fmt.Printf("type=0x%02x signer=%s\n", t.Encode()[0], recoverSender(t))
	// Output: type=0x02 signer=0x9d8a62f656a8d1615c1294fd71e9cfb3e4855a4f
}

// Build a fee-delegation (0x16) transaction: the sender pays value, a separate
// fee payer pays gas. Both signatures recover to their respective accounts.
func ExampleFeeDelegateTx() {
	senderKey := fixedKey()
	feePayerKey, _ := crypto.PrivKeyFromBytes(make32(0x01))
	to, _ := types.HexToAddress("0x3535353535353535353535353535353535353535")

	t := &tx.FeeDelegateTx{
		Sender: tx.DynamicFeeTx{
			ChainID:   big.NewInt(8283),
			Nonce:     0,
			GasTipCap: big.NewInt(1_000_000_000),
			GasFeeCap: big.NewInt(20_000_000_000),
			Gas:       21000,
			To:        &to,
			Value:     big.NewInt(1),
		},
	}
	// Sender signs first, then the fee payer.
	_ = t.Sign(senderKey, feePayerKey)

	sender, _ := t.SenderAddress()
	feePayer, _ := t.RecoverFeePayer()
	fmt.Printf("sender=%s\nfeePayer=%s\n", sender, feePayer)
	// Output:
	// sender=0x9d8a62f656a8d1615c1294fd71e9cfb3e4855a4f
	// feePayer=0x7e5f4552091a69125d5dfcb7b8c2659029395bdf
}

// Compute a CREATE2 contract address deterministically (EIP-1014).
func ExampleCreateAddress2() {
	sender, _ := types.HexToAddress("0x0000000000000000000000000000000000000000")
	var salt [32]byte // all zero
	addr := tx.CreateAddress2(sender, salt, []byte{0x00})
	fmt.Println(addr)
	// Output: 0x4d1a2e2bb4f88f0250f26ffff098b0b30b26bf38
}

func make32(b byte) []byte {
	out := make([]byte, 32)
	out[31] = b
	return out
}
