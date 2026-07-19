// Byte and hex utilities.

export function fromHex(s: string): Uint8Array {
  let h = s.startsWith("0x") ? s.slice(2) : s;
  if (h.length % 2 === 1) h = "0" + h;
  const out = new Uint8Array(h.length / 2);
  for (let i = 0; i < out.length; i++) {
    out[i] = parseInt(h.substring(i * 2, i * 2 + 2), 16);
  }
  return out;
}

export function toHex(b: Uint8Array): string {
  let s = "0x";
  for (const x of b) s += x.toString(16).padStart(2, "0");
  return s;
}

export function concat(parts: Uint8Array[]): Uint8Array {
  let n = 0;
  for (const p of parts) n += p.length;
  const out = new Uint8Array(n);
  let o = 0;
  for (const p of parts) {
    out.set(p, o);
    o += p.length;
  }
  return out;
}

export function utf8(s: string): Uint8Array {
  return new TextEncoder().encode(s);
}

export function equal(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return false;
  return true;
}

// bigToBytes32 returns the 32-byte big-endian encoding of a non-negative bigint.
export function bigToBytes32(x: bigint): Uint8Array {
  const out = new Uint8Array(32);
  let v = x;
  for (let i = 31; i >= 0 && v > 0n; i--) {
    out[i] = Number(v & 0xffn);
    v >>= 8n;
  }
  return out;
}

// minimalBE returns the minimal big-endian bytes of a non-negative bigint (no
// leading zeros; 0 -> empty).
export function minimalBE(x: bigint): Uint8Array {
  if (x === 0n) return new Uint8Array(0);
  const tmp: number[] = [];
  let v = x;
  while (v > 0n) {
    tmp.unshift(Number(v & 0xffn));
    v >>= 8n;
  }
  return new Uint8Array(tmp);
}
