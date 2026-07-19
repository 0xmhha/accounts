// Transactions: EIP-155 legacy signing and CREATE/CREATE2 addressing.
// (0x02/0x16 and other types follow the same primitives; this is the
// conformance-covered subset.)

import { keccak256, sign } from "./crypto.js";
import * as rlp from "./rlp.js";
import { concat, bigToBytes32 } from "./bytes.js";

export interface LegacyTx {
  nonce: bigint;
  gasPrice: bigint;
  gas: bigint;
  to: Uint8Array; // empty for contract creation
  value: bigint;
  data: Uint8Array;
}

export interface LegacySignature {
  v: bigint;
  r: Uint8Array;
  s: Uint8Array;
}

// legacySigningHash returns the EIP-155 signing hash.
export function legacySigningHash(tx: LegacyTx, chainId: bigint): Uint8Array {
  const payload = rlp.encodeList(
    rlp.encodeUint(tx.nonce),
    rlp.encodeUint(tx.gasPrice),
    rlp.encodeUint(tx.gas),
    rlp.encodeBytes(tx.to),
    rlp.encodeUint(tx.value),
    rlp.encodeBytes(tx.data),
    rlp.encodeUint(chainId),
    rlp.encodeUint(0n),
    rlp.encodeUint(0n),
  );
  return keccak256(payload);
}

// signLegacy signs an EIP-155 legacy transaction; V = recid + 35 + 2*chainId.
export function signLegacy(tx: LegacyTx, chainId: bigint, priv: Uint8Array): LegacySignature {
  const sig = sign(legacySigningHash(tx, chainId), priv);
  const recid = BigInt(sig[64]);
  return {
    r: sig.subarray(0, 32),
    s: sig.subarray(32, 64),
    v: recid + 35n + 2n * chainId,
  };
}

// createAddress computes a CREATE contract address: keccak(rlp([sender, nonce]))[12:].
export function createAddress(sender: Uint8Array, nonce: bigint): Uint8Array {
  const payload = rlp.encodeList(rlp.encodeBytes(sender), rlp.encodeUint(nonce));
  return keccak256(payload).subarray(12);
}

// createAddress2 computes a CREATE2 address (EIP-1014):
// keccak(0xff || sender || salt || keccak(initCode))[12:].
export function createAddress2(sender: Uint8Array, salt: Uint8Array, initCode: Uint8Array): Uint8Array {
  const initHash = keccak256(initCode);
  return keccak256(concat([new Uint8Array([0xff]), sender, salt, initHash])).subarray(12);
}

export { bigToBytes32 };
