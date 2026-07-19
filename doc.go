// Package accounts is the module root for the StableNet accounts SDK — a
// clean-room Go library for creating accounts, signing every StableNet
// transaction type, and encrypting/decrypting keys and data on go-stablenet.
//
// The SDK follows the flat-root library layout used by go-ethereum (which
// go-stablenet forks) and the Go standard library: each subpackage below is
// public API, imported directly by consumers. Implementation details that are
// not part of the public contract live under internal/.
//
// # Package map (public API)
//
//	account    account creation, signing (incl. EIP-191/EIP-712), keystore + ECIES
//	hdwallet   BIP-39 mnemonic + BIP-32/44 HD account derivation
//	wallet     high-level facade: auto nonce/gas/tip + blacklist guard, send/deploy/execute
//	tx         all transaction types (0x00,0x01,0x02,0x03,0x04,0x16) + CREATE2 + safety guards
//	signing    SigningScheme (secp256k1@1), EIP-191 personal_sign, EIP-712 typed data
//	crypto     Keccak-256, secp256k1 sign/recover, ECIES encrypt/decrypt
//	keystore   Web3 Secret Storage (keystore v3) encrypt/decrypt
//	abi        minimal ABI encoder/decoder for contract calls
//	token      ERC-20 / NativeCoinAdapter bindings + EIP-2612 permit
//	transport  JSON-RPC client + account state (Extra flag) queries
//	mobile     gomobile-safe facade for Android/iOS bindings
//	types      primitive value types (Address, Hash)
//
// A TypeScript SDK mirroring this one (conformance-tested against the same
// golden vectors) lives under sdk-ts/.
//
// # Internal
//
//	internal/rlp   minimal RLP encoder (implementation detail, not public API)
//
// Design, protocol spec, ADRs, and the threat model live under docs/. See
// docs/spec/protocol/v0 for the normative contract with the node.
package accounts

// Version is the SDK version (cycle 1: atomic signing core).
const Version = "0.1.0"
