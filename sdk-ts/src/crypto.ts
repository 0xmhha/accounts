// Cryptographic primitives: Keccak-256 and secp256k1, built on the permissively
// licensed @noble libraries (MIT). Mirrors the Go crypto package.

import { keccak_256 } from "@noble/hashes/sha3";
import { secp256k1 } from "@noble/curves/secp256k1";
import { concat } from "./bytes.js";

export function keccak256(...parts: Uint8Array[]): Uint8Array {
  return keccak_256(parts.length === 1 ? parts[0] : concat(parts));
}

// getPublicKey returns the uncompressed (65-byte) public key.
export function getPublicKey(priv: Uint8Array): Uint8Array {
  return secp256k1.getPublicKey(priv, false);
}

// privKeyToAddress derives the 20-byte address: keccak256(pub[1:])[12:].
export function privKeyToAddress(priv: Uint8Array): Uint8Array {
  const pub = getPublicKey(priv);
  return keccak256(pub.subarray(1)).subarray(12);
}

// sign returns a 65-byte recoverable signature [R(32) || S(32) || V(1)] where V
// is the recovery id (0/1). Canonical low-S (EIP-2) by default.
export function sign(hash: Uint8Array, priv: Uint8Array): Uint8Array {
  const sig = secp256k1.sign(hash, priv); // lowS enforced by default
  const out = new Uint8Array(65);
  out.set(sig.toCompactRawBytes(), 0); // R || S
  out[64] = sig.recovery!;
  return out;
}

// recoverAddress recovers the signer address from a signature [R||S||V].
export function recoverAddress(hash: Uint8Array, sig: Uint8Array): Uint8Array {
  const r = BigInt("0x" + toHexRaw(sig.subarray(0, 32)));
  const s = BigInt("0x" + toHexRaw(sig.subarray(32, 64)));
  const rec = sig[64];
  const signature = new secp256k1.Signature(r, s).addRecoveryBit(rec);
  const point = signature.recoverPublicKey(hash);
  const pub = point.toRawBytes(false); // 65 bytes
  return keccak256(pub.subarray(1)).subarray(12);
}

function toHexRaw(b: Uint8Array): string {
  let out = "";
  for (const x of b) out += x.toString(16).padStart(2, "0");
  return out;
}
