// EIP-712 typed structured data hashing (port of Go signing/eip712.go).

import { keccak256 } from "./crypto.js";
import { concat, utf8, bigToBytes32 } from "./bytes.js";

export interface Field {
  name: string;
  type: string;
}
export type Types = Record<string, Field[]>;

export interface Domain {
  name?: string;
  version?: string;
  chainId?: bigint;
  verifyingContract?: Uint8Array;
  salt?: Uint8Array;
}

export interface TypedData {
  types: Types;
  primaryType: string;
  domain: Domain;
  message: Record<string, unknown>;
}

function domainToMap(d: Domain): Record<string, unknown> {
  const m: Record<string, unknown> = {};
  if (d.name !== undefined) m.name = d.name;
  if (d.version !== undefined) m.version = d.version;
  if (d.chainId !== undefined) m.chainId = d.chainId;
  if (d.verifyingContract !== undefined) m.verifyingContract = d.verifyingContract;
  if (d.salt !== undefined) m.salt = d.salt;
  return m;
}

export function digest(td: TypedData): Uint8Array {
  const domainSep = hashStruct(td, "EIP712Domain", domainToMap(td.domain));
  const msgHash = hashStruct(td, td.primaryType, td.message);
  return keccak256(concat([new Uint8Array([0x19, 0x01]), domainSep, msgHash]));
}

function hashStruct(td: TypedData, typeName: string, data: Record<string, unknown>): Uint8Array {
  return keccak256(encodeData(td, typeName, data));
}

function encodeData(td: TypedData, typeName: string, data: Record<string, unknown>): Uint8Array {
  const fields = td.types[typeName];
  if (!fields) throw new Error(`unknown type ${typeName}`);
  const parts: Uint8Array[] = [typeHash(td, typeName)];
  for (const f of fields) parts.push(encodeField(td, f.type, data[f.name]));
  return concat(parts);
}

function typeHash(td: TypedData, typeName: string): Uint8Array {
  return keccak256(utf8(encodeType(td, typeName)));
}

function encodeType(td: TypedData, primary: string): string {
  const deps = new Set<string>();
  collectDeps(td, primary, deps);
  deps.delete(primary);
  const sorted = [...deps].sort();
  const one = (name: string) =>
    `${name}(${td.types[name].map((f) => `${f.type} ${f.name}`).join(",")})`;
  return one(primary) + sorted.map(one).join("");
}

function collectDeps(td: TypedData, typeName: string, found: Set<string>) {
  if (found.has(typeName) || !td.types[typeName]) return;
  found.add(typeName);
  for (const f of td.types[typeName]) {
    const base = f.type.replace(/\[\]$/, "");
    if (td.types[base]) collectDeps(td, base, found);
  }
}

function encodeField(td: TypedData, fieldType: string, value: unknown): Uint8Array {
  if (fieldType.endsWith("[]")) {
    const elem = fieldType.slice(0, -2);
    const arr = value as unknown[];
    return keccak256(concat(arr.map((e) => encodeField(td, elem, e))));
  }
  if (td.types[fieldType]) {
    return hashStruct(td, fieldType, value as Record<string, unknown>);
  }
  if (fieldType === "string") return keccak256(utf8(value as string));
  if (fieldType === "bytes") return keccak256(value as Uint8Array);
  if (fieldType === "address") return leftPad32(value as Uint8Array);
  if (fieldType === "bool") {
    const out = new Uint8Array(32);
    if (value) out[31] = 1;
    return out;
  }
  if (fieldType.startsWith("bytes")) return rightPad32(value as Uint8Array);
  if (fieldType.startsWith("uint") || fieldType.startsWith("int")) {
    return bigToBytes32(value as bigint);
  }
  throw new Error(`unsupported field type ${fieldType}`);
}

function leftPad32(b: Uint8Array): Uint8Array {
  const out = new Uint8Array(32);
  out.set(b.subarray(Math.max(0, b.length - 32)), 32 - Math.min(32, b.length));
  return out;
}

function rightPad32(b: Uint8Array): Uint8Array {
  const out = new Uint8Array(32);
  out.set(b.subarray(0, 32), 0);
  return out;
}
