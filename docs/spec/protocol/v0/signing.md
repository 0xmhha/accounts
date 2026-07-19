# Signing Scheme Protocol — v0

> 서명 알고리즘/로직 변경을 한 지점에 격리하기 위한 추상화. 서명은 보안 크리티컬 표면이므로 이 인터페이스가 SDK 전 구현의 유일한 서명 진입점이다.

- 근거 소스: `core/types/transaction_signing.go`, `crypto/crypto.go`, `crypto/signature_cgo.go`, `crypto/secp256r1/verifier.go`

---

## 1. 목적

- 서명 알고리즘이나 sighash 로직이 바뀌어도 SDK 상위 코드가 영향받지 않도록 **격리**한다.
- 서명/키 취급 코드를 한 곳으로 모아 **감사 표면을 최소화**한다(보안 최우선).

## 2. SigningScheme 인터페이스 (normative)

언어별 SDK는 아래에 상응하는 인터페이스를 구현한다.

```
interface SigningScheme {
    id() -> string                          // 예: "secp256k1@1"

    // role ∈ { "sender", "feePayer" }
    sigHash(tx, role) -> hash               // tx type과 role에 맞는 preimage 해시

    sign(hash, key) -> signature            // 원자적 서명
    recover(hash, signature) -> address     // 복구/검증
}
```

- `sigHash`는 tx type과 role을 받아 [`transactions.md`](./transactions.md)의 공식을 적용한다. `0x16`의 `sender`/`feePayer`는 서로 다른 preimage를 갖는다(§transactions 3.3).
- `sign`/`recover`만이 원시 키·서명을 다룬다. 상위 계층은 이 인터페이스 밖에서 키를 직접 만지지 않는다(MUST NOT).

## 3. 유일 구현 — `secp256k1@1`

| 항목 | 값 |
|------|-----|
| id | `secp256k1@1` |
| 곡선 | secp256k1 (`crypto.S256()`) |
| 서명 | `crypto.Sign` → `secp256k1.Sign`, 형식 `[R ‖ S ‖ V]` (`crypto/signature_cgo.go:53-59`) |
| 복구 | `crypto.Ecrecover` / `SigToPub` (스톡) |
| role 처리 | `sender`: tx type별 표준 sighash. `feePayer`: `0x16` 전용 sighash |

- go-stablenet의 tx 서명은 **secp256k1 전용**이다. 비-secp256k1 키(p256/webauthn)는 tx 서명에 쓰이지 않는다.

## 4. 확장 지점 (extension points)

향후 서명 스킴 변경/추가는 **새 `SigningScheme` 구현 + 새 id(`scheme@version`)** 로 국소화한다. 상위 코드·봉투 인코딩은 스킴 id를 통해 협상한다.

### P256 / passkey 는 코어 밖 (informative)

- 체인에는 **P256VERIFY precompile `0x100`**(Boho, `crypto/secp256r1/verifier.go`)이 있으나 이는 **검증 전용**이며 네이티브 tx 서명 경로가 아니다.
- 따라서 P256/passkey 지원은 (Tempo식) **컨트랙트 레벨 검증**으로 다루는 응용확장이다. 코어의 tx 서명 스킴은 `secp256k1@1` 로 고정한다.

## 5. 키 취급 규정 (normative, 보안)

- 개인키는 `sign` 호출 경계 밖으로 노출되지 않는다. 로깅/에러 메시지에 키·서명 원본을 포함하지 않는다(MUST NOT).
- 키 영속화는 코어가 아니라 언어별 SDK의 플랫폼 보안저장(OS 키체인, 브라우저 WebCrypto/비추출 키 등)에 위임한다. 코어는 서명 시점에만 키 소재를 다룬다.
- nonce/서명 malleability 등은 스톡 secp256k1 구현에 위임하되, 재현 구현(TypeScript 등)은 골든 벡터로 노드와 바이트 일치를 보증한다.

## 6. 검증 대상 (conformance vectors)

- `secp256k1@1`의 `sign`/`recover` 라운드트립(고정 키·해시).
- 각 tx type·role의 `sigHash` 결과가 노드와 일치.
- `0x16` sender/feePayer 두 role의 서로 다른 sighash.
