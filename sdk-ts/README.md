# @0xmhha/accounts (TypeScript)

TypeScript SDK for StableNet accounts — a **clean-room** implementation mirroring
the Go SDK and verified against the **same** conformance golden vectors
(`../conformance/vectors/core.json`), so both languages produce byte-identical
outputs.

- Permissive deps only: `@noble/hashes`, `@noble/curves` (MIT).
- No go-stablenet (LGPL/GPL) code.

## Cycle-1 surface

| Module | Contents |
|--------|----------|
| `crypto` | Keccak-256, secp256k1 sign/recover, address derivation |
| `rlp` | minimal RLP encoder |
| `tx` | EIP-155 legacy signing, CREATE / CREATE2 addressing |
| `account` | `Extra` bitmap decode (blacklisted/authorized) |
| `eip191` | personal_sign digest |
| `eip712` | typed-data digest |
| `bytes` | hex/byte utilities |

Follow-up (parity with Go): full tx-type encode (0x01/0x02/0x03/0x04/0x16),
keystore, transport, wallet, hdwallet, token.

## Develop

```bash
npm install
npm test         # vitest: conformance vs shared golden vectors
npm run typecheck
npm run build
```

## Usage

```ts
import { privKeyToAddress, signLegacy, createAddress2 } from "@0xmhha/accounts";
```

## License

Apache-2.0 OR MIT.
