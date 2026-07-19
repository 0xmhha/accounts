// Account state: the StableNet StateAccount.Extra bitfield (mirrors Go account).

export const MASK_BLACKLISTED = 1n << 63n;
export const MASK_AUTHORIZED = 1n << 62n;

export interface ExtraFlags {
  blacklisted: boolean;
  authorized: boolean;
  raw: bigint;
}

// decodeExtra performs lenient decoding: only known bits are interpreted, the
// raw word is preserved (forward compatibility; spec account.md §6).
export function decodeExtra(raw: bigint): ExtraFlags {
  return {
    blacklisted: (raw & MASK_BLACKLISTED) !== 0n,
    authorized: (raw & MASK_AUTHORIZED) !== 0n,
    raw,
  };
}
