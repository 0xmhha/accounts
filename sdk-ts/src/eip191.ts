// EIP-191 personal_sign hashing (mirrors Go signing.EIP191Hash).

import { keccak256 } from "./crypto.js";
import { utf8, concat } from "./bytes.js";

export function eip191Hash(msg: Uint8Array): Uint8Array {
  const prefix = utf8(`\x19Ethereum Signed Message:\n${msg.length}`);
  return keccak256(concat([prefix, msg]));
}
