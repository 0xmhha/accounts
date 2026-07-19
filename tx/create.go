package tx

import (
	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/internal/rlp"
	"github.com/0xmhha/accounts/types"
)

// CreateAddress computes the address of a contract created via CREATE:
// keccak(rlp([sender, nonce]))[12:]. (Stock Ethereum; go-stablenet does not
// diverge.)
func CreateAddress(sender types.Address, nonce uint64) types.Address {
	payload := rlp.EncodeList(
		rlp.EncodeBytes(sender.Bytes()),
		rlp.EncodeUint(nonce),
	)
	h := crypto.Keccak256(payload)
	return types.BytesToAddress(h[12:])
}

// CreateAddress2 computes the address of a contract created via CREATE2:
// keccak(0xff || sender || salt || keccak(initCode))[12:] (EIP-1014). salt must
// be 32 bytes.
func CreateAddress2(sender types.Address, salt [32]byte, initCode []byte) types.Address {
	initHash := crypto.Keccak256(initCode)
	h := crypto.Keccak256([]byte{0xff}, sender.Bytes(), salt[:], initHash)
	return types.BytesToAddress(h[12:])
}
