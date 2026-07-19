// Cross-language conformance: the TypeScript SDK must reproduce the SAME golden
// vectors as the Go SDK (accounts/conformance/vectors/core.json).

import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";

import { fromHex, toHex } from "../src/bytes.js";
import { privKeyToAddress } from "../src/crypto.js";
import { createAddress2, signLegacy, type LegacyTx } from "../src/tx.js";
import { decodeExtra } from "../src/account.js";
import { eip191Hash } from "../src/eip191.js";
import * as eip712 from "../src/eip712.js";

const vectorsPath = fileURLToPath(new URL("../../conformance/vectors/core.json", import.meta.url));
const V = JSON.parse(readFileSync(vectorsPath, "utf8"));

describe("conformance golden vectors", () => {
  it("address derivation", () => {
    for (const c of V.address) {
      expect(toHex(privKeyToAddress(fromHex(c.privKey)))).toBe(c.address);
    }
  });

  it("CREATE2 (EIP-1014)", () => {
    for (const c of V.create2) {
      const addr = createAddress2(fromHex(c.sender), fromHex(c.salt), fromHex(c.initCode));
      expect(toHex(addr)).toBe(c.address);
    }
  });

  it("Extra bitmap decode", () => {
    for (const c of V.extra) {
      const f = decodeExtra(BigInt(c.raw));
      expect(f.blacklisted).toBe(c.blacklisted);
      expect(f.authorized).toBe(c.authorized);
    }
  });

  it("EIP-191 personal_sign digest", () => {
    for (const c of V.eip191) {
      expect(toHex(eip191Hash(new TextEncoder().encode(c.message)))).toBe(c.digest);
    }
  });

  it("EIP-712 Mail digest", () => {
    const td: eip712.TypedData = {
      types: {
        EIP712Domain: [
          { name: "name", type: "string" },
          { name: "version", type: "string" },
          { name: "chainId", type: "uint256" },
          { name: "verifyingContract", type: "address" },
        ],
        Person: [
          { name: "name", type: "string" },
          { name: "wallet", type: "address" },
        ],
        Mail: [
          { name: "from", type: "Person" },
          { name: "to", type: "Person" },
          { name: "contents", type: "string" },
        ],
      },
      primaryType: "Mail",
      domain: {
        name: "Ether Mail",
        version: "1",
        chainId: 1n,
        verifyingContract: fromHex("0xCcCCccccCCCCcCCCCCCcCcCccCcCCCcCcccccccC"),
      },
      message: {
        from: { name: "Cow", wallet: fromHex("0xCD2a3d9F938E13CD947Ec05AbC7FE734Df8DD826") },
        to: { name: "Bob", wallet: fromHex("0xbBbBBBBbbBBBbbbBbbBbbbbBBbBbbbbBbBbbBBbB") },
        contents: "Hello, Bob!",
      },
    };
    expect(toHex(eip712.digest(td))).toBe(V.eip712MailDigest);
  });

  it("legacy EIP-155 signature", () => {
    const v = V.legacyEip155;
    const tx: LegacyTx = {
      nonce: BigInt(v.nonce),
      gasPrice: BigInt(v.gasPrice),
      gas: BigInt(v.gas),
      to: fromHex(v.to),
      value: BigInt(v.value),
      data: new Uint8Array(0),
    };
    const sig = signLegacy(tx, BigInt(v.chainId), fromHex(v.privKey));
    expect(toHex(sig.r)).toBe(v.r);
    expect(toHex(sig.s)).toBe(v.s);
    expect(sig.v).toBe(BigInt(v.v));
  });
});
