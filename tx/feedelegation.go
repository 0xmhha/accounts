package tx

import (
	"errors"
	"math/big"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/internal/rlp"
	"github.com/0xmhha/accounts/types"
)

// FeeDelegateTx is the StableNet fee-delegation transaction (type 0x16). A
// sender signs the inner EIP-1559 transaction; a separate fee payer then signs
// over the sender-signed transaction plus the fee-payer address. The fee payer
// pays gas; the sender pays value.
//
// Spec: docs/spec/protocol/v0/transactions.md §3.
type FeeDelegateTx struct {
	Sender   DynamicFeeTx   // inner tx; its V/R/S hold the sender signature
	FeePayer *types.Address // fee payer address (set when signing as fee payer)

	FV *big.Int // fee payer signature V (recovery id 0/1)
	FR *big.Int
	FS *big.Int
}

// SenderSigHash is the hash the sender signs: identical to the inner 0x02
// EIP-1559 sighash.
func (t *FeeDelegateTx) SenderSigHash() []byte { return t.Sender.SigHash() }

// FeePayerSigHash is the hash the fee payer signs:
// keccak(0x16 || rlp([ [chainId,nonce,tipCap,feeCap,gas,to,value,data,accessList,
// senderV,senderR,senderS], feePayer ])). Requires the sender signature and
// FeePayer to be set.
func (t *FeeDelegateTx) FeePayerSigHash() ([]byte, error) {
	if t.Sender.V == nil || t.Sender.R == nil || t.Sender.S == nil {
		return nil, errors.New("sender must sign before fee payer")
	}
	if t.FeePayer == nil {
		return nil, errors.New("fee payer address not set")
	}
	innerList := t.Sender.signedListPayload() // rlp([core..., v, r, s])
	outer := rlp.EncodeList(innerList, rlp.EncodeBytes(t.FeePayer.Bytes()))
	return crypto.Keccak256([]byte{FeeDelegateDynamicFeeTxType}, outer), nil
}

// SignSender signs the inner transaction as the sender.
func (t *FeeDelegateTx) SignSender(priv *crypto.PrivateKey) error {
	return t.Sender.Sign(priv)
}

// SignFeePayer sets FeePayer to priv's address, computes the fee-payer sighash,
// and populates FV/FR/FS. SignSender MUST have run first.
func (t *FeeDelegateTx) SignFeePayer(priv *crypto.PrivateKey) error {
	addr := crypto.PrivKeyToAddress(priv)
	t.FeePayer = &addr
	hash, err := t.FeePayerSigHash()
	if err != nil {
		return err
	}
	sig, err := crypto.Sign(hash, priv)
	if err != nil {
		return err
	}
	t.FR = new(big.Int).SetBytes(sig[0:32])
	t.FS = new(big.Int).SetBytes(sig[32:64])
	t.FV = new(big.Int).SetInt64(int64(sig[64]))
	return nil
}

// Sign performs the full dual-signature in the required order: sender first,
// then fee payer.
func (t *FeeDelegateTx) Sign(senderKey, feePayerKey *crypto.PrivateKey) error {
	if err := t.SignSender(senderKey); err != nil {
		return err
	}
	return t.SignFeePayer(feePayerKey)
}

// Encode returns the typed envelope:
// 0x16 || rlp([ senderSignedList, feePayer, FV, FR, FS ]).
func (t *FeeDelegateTx) Encode() ([]byte, error) {
	if t.FeePayer == nil || t.FV == nil {
		return nil, errors.New("fee payer signature missing")
	}
	body := rlp.EncodeList(
		t.Sender.signedListPayload(),
		rlp.EncodeBytes(t.FeePayer.Bytes()),
		rlp.EncodeBig(nz(t.FV)),
		rlp.EncodeBig(nz(t.FR)),
		rlp.EncodeBig(nz(t.FS)),
	)
	return append([]byte{FeeDelegateDynamicFeeTxType}, body...), nil
}

// SenderAddress recovers the sender from the sender signature.
func (t *FeeDelegateTx) SenderAddress() (types.Address, error) {
	sig, err := sigBytes(t.Sender.R, t.Sender.S, t.Sender.V)
	if err != nil {
		return types.Address{}, err
	}
	return crypto.Recover(t.SenderSigHash(), sig)
}

// RecoverFeePayer recovers the fee payer and verifies it matches the declared
// FeePayer address (mirrors the node's RecoverFeePayer check).
func (t *FeeDelegateTx) RecoverFeePayer() (types.Address, error) {
	hash, err := t.FeePayerSigHash()
	if err != nil {
		return types.Address{}, err
	}
	sig, err := sigBytes(t.FR, t.FS, t.FV)
	if err != nil {
		return types.Address{}, err
	}
	got, err := crypto.Recover(hash, sig)
	if err != nil {
		return types.Address{}, err
	}
	if got != *t.FeePayer {
		return types.Address{}, errors.New("invalid fee payer: recovered address mismatch")
	}
	return got, nil
}

// sigBytes reconstructs a 65-byte [R || S || V] signature from big.Int values.
func sigBytes(r, s, v *big.Int) ([]byte, error) {
	if r == nil || s == nil || v == nil {
		return nil, errors.New("incomplete signature")
	}
	sig := make([]byte, crypto.SignatureLength)
	rb, sb := r.Bytes(), s.Bytes()
	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	sig[64] = byte(v.Int64())
	return sig, nil
}
