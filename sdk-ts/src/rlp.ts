// Minimal RLP encoder (mirrors the Go internal/rlp package).

import { concat, minimalBE } from "./bytes.js";

export function encodeBytes(b: Uint8Array): Uint8Array {
  if (b.length === 1 && b[0] <= 0x7f) return b;
  return concat([encodeLength(b.length, 0x80), b]);
}

export function encodeUint(x: bigint): Uint8Array {
  return encodeBytes(minimalBE(x));
}

export function encodeList(...items: Uint8Array[]): Uint8Array {
  const payload = concat(items);
  return concat([encodeLength(payload.length, 0xc0), payload]);
}

function encodeLength(n: number, offset: number): Uint8Array {
  if (n < 56) return new Uint8Array([offset + n]);
  const lenBytes = minimalBE(BigInt(n));
  return concat([new Uint8Array([offset + 55 + lenBytes.length]), lenBytes]);
}
