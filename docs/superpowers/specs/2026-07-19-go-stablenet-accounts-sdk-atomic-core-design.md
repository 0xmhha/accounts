# go-stablenet Accounts SDK — Atomic Core 설계 (사이클 1)

- 작성일: 2026-07-19
- 대상 체인: go-stablenet (geth 포크, WBFT PoA, stablecoin base coin)
- 이 문서의 범위: **사이클 1 = atomic 서명 코어 (Go + TypeScript)**
- 후속 사이클(별도 문서): 모바일(Android/iOS) 바인딩, 응용확장(fee-token/거버넌스 연동), 지식화 + MCP 서버

> 근거 규율: 이 설계는 go-stablenet `dev` 브랜치(`build-source-files.md` 기준: 160 패키지 / 778 파일)에서 **실제 바이너리에 빌드되는 코드만**을 근거로 한다. 미빌드/미사용 경로(예: Cancun signer)에는 의존하지 않는다.

---

## 1. 목적과 설계 철학

블록체인의 상태변화 요청(Transaction)은 각 주체가 private key로 서명해 처리해야 하며, 이 서명·계정 처리는 모든 DApp이 반복 구현해야 하는 허들이다. 이 프로젝트는 **go-stablenet용 서명·계정 처리를 쉽고 안전하게 제공하는 재사용 코드**를 만든다.

Tempo `accounts` SDK가 참조 개념이지만, go-stablenet의 온체인 실물은 다르다(§2 참조). 따라서 Tempo 기능을 이식하는 것이 아니라, **원자적(atomic) 서명·계정 프리미티브를 먼저 구현하고, 그 위에 응용확장을 조립**한다.

### 최우선 품질 속성

1. **보안성** — 코드에서 보안 취약점이 발견되지 않도록. bespoke 암호 코드를 최소화하고 감사 표면을 한 곳에 모은다.
2. **유지보수성** — go-stablenet 노드 변경이 다언어 구현으로 무분별하게 전파되지 않도록, **protocol을 명시적으로 추상화**한다.

---

## 2. go-stablenet 온체인 실물 (설계 전제)

Tempo SDK의 간판 기능 대부분(access key/세션키, WebAuthn tx 서명, embed dialog, relay 스폰서십, DEX/deposits, MPP)은 **go-stablenet 온체인에 존재하지 않는다.** 실측 결과:

| 요소 | go-stablenet 실물 | 근거 (file) |
|------|------------------|------------|
| tx 서명 스킴 | 스톡 secp256k1 v/r/s | `core/types/transaction_signing.go`, `crypto/signature_cgo.go` |
| 세션키/access key | **없음** (근접: 온체인 `authorized` 플래그, 서명권한 아님) | `core/vm/native_manager.go` |
| WebAuthn/p256 tx 서명 | **없음** (p256은 검증 precompile `0x100`만, Boho) | `crypto/secp256r1/verifier.go` |
| fee-token 필드 | **없음** — stablecoin이 곧 네이티브 base coin | — |
| 커스텀 RPC 네임스페이스 | 없음. `eth_*`에 2개 메서드만 추가 | `internal/ethapi/api.go` |
| **StableNet 고유 tx** | tx type `0x16` FeeDelegateDynamicFeeTx (이중서명) | `core/types/tx_fee_delegation.go` |
| **StableNet 고유 계정 상태** | `StateAccount.Extra uint64` 비트필드 | `core/types/state_account_extra.go` |

### 하드포크 상태 (mainnet 기준)

| 포크 | 활성 블록 | 계정/tx 관련 변경 |
|------|----------|------------------|
| Applepie | 0 | fee delegation(0x16) 게이팅 |
| Anzeon | 0 | WBFT, 시스템계약 v1, `Extra` 플래그, authorized-tip, blacklist 강제 |
| Boho | Mainnet 0 / Testnet 100 | GovMinter v2, P256VERIFY(`0x100`) precompile |

Mainnet은 세 포크 모두 block 0부터 활성. **SDK는 pre-fork 상태를 다루지 않는다.**

### 체인 파라미터

| 항목 | 값 |
|------|-----|
| Mainnet chainId | `8282` |
| Testnet chainId | `8283` |

---

## 3. 아키텍처 — 하이브리드

> **[라이선스 정합 · ADR-0001 Accepted 2026-07-19]** 본 절과 §11·§15-3의 초기 표현 "Go 코어가 go-stablenet 정본 코드를 직접 재사용(bespoke 0)"은 **채택되지 않았다**. go-stablenet(LGPL/GPL) import는 SDK의 permissive 배포를 막으므로, **Option C(clean-room)** 를 채택한다: go-stablenet를 import하지 않고, StableNet 고유의 작은 divergent 로직(**0x16 이중서명 · Extra 비트맵**)만 스펙(`protocol/v0`) 기반으로 clean-room 재구현하며, 표준 서명/인코딩은 permissive 라이브러리(Go: dcrd secp256k1 등, TS: ox/noble)를 쓴다. 정확성은 **골든 벡터(노드=오라클)** 로 바이트 일치 보증한다. → "bespoke 0"이 아니라 "bespoke = 벡터로 검증되는 2개 로직"이며, 감사 표면 최소화(보안 목표)는 유지된다. 아래 본문의 "정본 코드 재사용" 서술은 이 노트로 대체해 읽는다.

보안(취약점 zero)과 유지보수(노드 변경 격리)를 동시에 만족시키기 위해 **하이브리드**를 채택한다.

```
[앱 코드 / AI 에이전트]
      │  관용 API (언어별)
      ▼
┌───────────────────────────────────────────────────────────┐
│  언어별 얇은 SDK  (사이클1: Go / TypeScript)                  │  관용 표현, 플랫폼 보안저장, RPC 배선
├───────────────────────────────────────────────────────────┤
│  Protocol 스펙 (버전드)  ── 노드 변경 완충재(계약) ──          │
│   · Account 구조체/Extra 비트맵  (§4, 필수)                    │
│   · tx-type/봉투 스키마 + 0x16 이중 sighash  (§5)             │
│   · SigningScheme 스킴@버전  (§6)                             │
│   · 시스템계약 주소 레지스트리 + ABI  (§9)                      │
│   · 골든 테스트 벡터  (§14)                                    │
├───────────────────────────────────────────────────────────┤
│  Go 공유 보안 코어  (키·서명·봉투 인코딩)                        │  go-stablenet 정본 코드 재사용, 1회 감사
│   → (사이클1) Go native · WASM(TS)                           │
│   → (후속) gomobile → Android/iOS                            │
└───────────────────────────────────────────────────────────┘
                     │  go-stablenet JSON-RPC
                     ▼
                [go-stablenet 노드]
```

### 계층 책임

| 계층 | 책임 | 보안 등급 |
|------|------|----------|
| Go 공유 보안 코어 | 키 생성/파생, 서명, sighash, EIP-2718 봉투 인코딩 | **크리티컬** (bespoke 암호 여기로 수렴) |
| Protocol 스펙 | 노드↔SDK 계약(구조/스키마/주소/벡터), 버전관리 | 계약 |
| 언어별 얇은 SDK | 관용 API, 플랫폼 보안저장, RPC 전송, 상태 쿼리 | 비암호 |

### 언어별 코어 전략

- **Go SDK**: 공유 코어가 go-stablenet `core/types`·`crypto`를 그대로 import → **bespoke 암호 0**, 노드 변경 컴파일타임 감지.
- **TypeScript SDK**: 스톡 부분은 `viem`/`ox` 사용, **0x16 이중서명만** 추가 구현(유일한 bespoke). 필요 시 Go 코어를 WASM으로 활용 가능하나, 1차는 `ox` 기반 + 골든 벡터 일치 검증을 기본으로 한다.

---

## 4. Account 구조체 Protocol (필수 · 핵심)

> 이 절은 본 설계의 필수 포함 항목이다. go-stablenet 계정 상태는 이더리움과 **정확히 한 필드**가 다르며, SDK는 이 divergence를 protocol로 고정해 다룬다.

### 4.1 StateAccount 구조체 (divergence)

go-stablenet은 스톡 이더리움 `StateAccount {Nonce, Balance, Root, CodeHash}`에 **`Extra uint64` 한 필드를 추가**한다.

```go
// core/types/state_account.go:31-38
type StateAccount struct {
    Nonce    uint64
    Balance  *uint256.Int
    Root     common.Hash   // storage trie root
    CodeHash []byte
    Extra    uint64 `rlp:"optional"`   // ← StableNet 추가
}
```

- `rlp:"optional"`: `Extra == 0`인 계정은 스톡 geth와 **동일 RLP 인코딩** → 하위호환. (`SlimAccount`·`Copy()`·`FullAccount`에도 반영)
- 빈 계정 판정도 `Extra == 0`을 요구한다 (`core/state/state_object.go:95`).

### 4.2 Extra 비트필드 정의 (protocol 상수)

`Extra`는 64비트 플래그 워드다. 현재 **2비트만 정의**, 나머지는 예약.

| 비트 | 마스크 | 의미 | 상수 (`core/types/state_account_extra.go`) |
|------|--------|------|-------------------------------------------|
| 63 (MSB) | `0x8000000000000000` | Blacklisted | `AccountExtraMaskBlacklisted = 1<<63` (:33) |
| 62 | `0x4000000000000000` | Authorized | `AccountExtraMaskAuthorized = 1<<62` (:36) |
| 61 | `0x2000000000000000` | 예약(미정의) | 주석 처리 (:38-40) |
| 60..0 | — | 예약(미정의) | — |

- `AccountExtraValidMask` = 정의된 비트의 합집합 (:45). `ValidateExtra`는 미정의 비트를 거부 (:103-108).
- 헬퍼(불변 패턴, 새 값 반환): `Is/Set/Clear{Blacklisted,Authorized}(extra) ` (:72-99).

### 4.3 의미론

- **Blacklisted (비트 63)**: 트랜잭션·EVM 호출·컨트랙트 생성에서 차단됨(§8.2). Anzeon 활성 시에만 강제.
- **Authorized (비트 62)**: Anzeon 가스팁 정책상 **자유 팁 설정** 권한(검증자/발행자 등). 비인증 계정은 거버넌스 GasTip이 강제됨(§8.1). 서명 권한과 무관.
- 상위 role(minter/master-minter/validator/council)은 `Extra`가 아니라 거버넌스 시스템계약 저장소에 있다(§9).

### 4.4 쓰기 경로 (참고 — SDK는 호출만)

- `Extra` 비트는 **AccountManager precompile `0x…B00003`** 를 통해서만 변경된다(`core/vm/native_manager.go`).
- `blacklist/unBlacklist/authorize/unAuthorize` 는 **호출자 == GovCouncil(`0x1004`) 且 op == CALL** 일 때만 허용(`canRunAccountManager`). 즉 일반 SDK는 상태를 바꾸지 않고 **읽기만** 한다.

### 4.5 클라이언트 쿼리 경로 (SDK 구현 규정)

SDK가 계정 상태를 읽는 **작동 확인된** JSON-RPC 경로는 둘이다.

1. **주력 — `eth_getProof`** (raw 플래그 워드, 미래 비트까지 노출):
   ```
   eth_getProof(address, [], "latest")  →  result.extra  (hex uint64, omitempty)
   ```
   - `internal/ethapi/api.go:734`(struct 필드), `:830`(`GetExtra` 채움).
   - 부재(`extra` 없음) ⇒ `0` 으로 취급.
   - 디코딩: `blacklisted = (extra >> 63) & 1`, `authorized = (extra >> 62) & 1`.

2. **대안 — `eth_call` to `0x…B00003`** (per-flag boolean):
   - `isBlacklisted(address)` / `isAuthorized(address)` 의 `CanRun == nil`(호출자 제약 없음)이라 익명 `eth_call` 가능(`native_manager.go:349-351,438-440`).
   - 반환 32바이트 워드의 LSB=결과.

> `eth_getAccount`은 **이 포크에 없다**. `NativeCoinAdapter(0x1000)`에는 public status getter가 없다(내부 `_isBlacklisted`가 `0xB00003`을 staticcall). 상태 읽기는 위 두 경로만 사용.

### 4.6 유지보수 메커니즘 — 버전드 비트맵 (핵심 가치)

`Extra`에는 예약 비트(61..0)가 있다. SDK는 **raw `uint64`를 읽고 protocol 스펙의 버전드 비트맵으로 디코딩**한다.

- 온체인에 새 비트(예: 비트 61)가 추가돼도 SDK는 **깨지지 않는다** — 모르는 비트는 무시(단, `ValidateExtra` 재현 시 unknown-bit 정책은 "관용 디코딩/엄격 인코딩"으로 분리).
- 스펙 버전을 올릴 때만 새 비트를 해석에 추가.
- 이것이 "노드의 당장 업데이트 불필요한 변경은 SDK 사용을 깨지 않는다"는 요구의 구체 구현이다.

### 4.7 SDK가 노출할 Account API (관용 계층)

| API | 구현 |
|-----|------|
| `getAccount(address)` | nonce/balance/code + `extra` 디코딩 결과 |
| `getExtraFlags(address)` | `{ blacklisted, authorized, raw }` (버전드 비트맵) |
| `isBlacklisted / isAuthorized(address)` | getProof 또는 0xB00003 |

---

## 5. Transaction Protocol (전 tx type + 0x16)

SDK는 go-stablenet이 지원하는 **모든 tx type**을 조립·서명·인코딩한다.

| type | 이름 | 표준/고유 | sighash |
|------|------|----------|---------|
| 0x00 | Legacy | 스톡 | EIP-155 |
| 0x01 | AccessList | 스톡 | EIP-2930 |
| 0x02 | DynamicFee | 스톡 | EIP-1559 |
| 0x03 | Blob | 스톡 | EIP-4844 |
| 0x04 | SetCode | 스톡(EIP-7702, Anzeon) | EIP-7702 |
| **0x16** | **FeeDelegateDynamicFee** | **StableNet 고유** | §5.2 |

> Go 코어는 `core/types`의 해당 타입을 재사용. TS는 `viem`가 0x00~0x04를 제공, 0x16만 추가.

### 5.1 SDK는 signer 선택이 아니라 sighash 공식을 재현한다

노드의 `MakeSigner`는 StableNet 설정에서 `anzeonSigner`(sender)·`feeDelegateSigner`(feePayer)를 쓴다(Cancun 미채택이므로 cancun 경로는 미사용 → 의존 금지). SDK 관점에서 필요한 것은 **tx type별 안정적인 sighash 공식**뿐이다.

### 5.2 0x16 FeeDelegateDynamicFeeTx — 이중서명 (유일한 bespoke)

구조 (`core/types/tx_fee_delegation.go:27-34`):

```go
type FeeDelegateDynamicFeeTx struct {
    SenderTx  DynamicFeeTx      // 내부 EIP-1559 tx (자체 V/R/S 포함)
    FeePayer  *common.Address
    FV, FR, FS *big.Int         // FeePayer secp256k1 서명
}
```

봉투: `0x16 || rlp([ SenderTx(DynamicFeeTx 필드…), FeePayer, FV, FR, FS ])`.

**이중서명 순서와 sighash (SDK가 정확히 재현):**

1. **Sender 서명** = 내부 0x02 EIP-1559 sighash와 **동일 preimage**:
   `keccak(0x02 || rlp([chainId, nonce, tipCap, feeCap, gas, to, value, data, accessList]))`
   → 결과를 `SenderTx.V/R/S`에 저장. (`transaction_signing.go:439-448`)
2. **FeePayer 서명** = sender 서명을 포함한 preimage:
   `keccak(0x16 || rlp([ [chainId, nonce, tipCap, feeCap, gas, to, value, data, accessList, senderV, senderR, senderS], feePayer ]))`
   → `FV/FR/FS`에 저장. (`tx_fee_delegation.go:158-178`, `transaction_signing.go:385-389`)

**불변 규칙:** sender가 먼저, feePayer가 나중. feePayer는 sender의 확정된 V/R/S 위에 서명한다.

**주의(문서 §11 대조):** `setSignatureValues()`는 FeePayer 서명에, `rawSignatureValues()`는 Sender 서명에, `rawFeePayerSignatureValues()`는 FeePayer 서명에 대응. 가스 가격은 Sender의 `GasFeeCap/GasTipCap` 사용.

**게이팅:** Applepie 이후에만 유효(`core/txpool/validation.go`). Mainnet은 block 0부터 활성.

### 5.3 노드측 대납 서명(선택)

`eth_signRawFeeDelegateTransaction` / `personal_signRawFeeDelegateTransaction`(`internal/ethapi/api.go:2189-2235`)로 노드가 feePayer 서명을 대신할 수 있다. **완전 클라이언트측 SDK는 0x16을 스스로 구성·서명하고 `eth_sendRawTransaction`만 쓴다.** 노드측 서명은 선택 경로로 지원.

---

## 6. Signing 추상화 (확장성 요구)

서명 알고리즘/코드 변경에 대비해 서명을 **한 지점에 격리**한다.

```
interface SigningScheme {
    id        // 예: "secp256k1@1"
    sigHash(tx, role)      → hash      // role ∈ {sender, feePayer}
    sign(hash, key)        → signature
    recover(hash, sig)     → address
}
```

- 현재 유일 구현: `secp256k1` — sender/feePayer 두 role의 서로 다른 sighash(§5.2)를 캡슐화.
- 알고리즘 교체·sighash 로직 변경은 이 인터페이스 구현 교체로 국소화.
- protocol 스펙에 `scheme@version`으로 명시 → 버전 협상 가능.

**확장 시나리오(코어 밖):** 체인의 P256VERIFY(`0x100`) precompile을 이용한 컨트랙트-레벨 P256/passkey 검증은 **네이티브 tx 서명이 아니라 응용확장**으로 다룬다. 코어의 tx 서명은 secp256k1 고정.

---

## 7. 컨트랙트 배포 (CREATE / CREATE2)

배포는 스톡이다(제약 하나 제외).

- **CREATE**: `crypto.CreateAddress(sender, nonce)` (`core/vm/evm.go:572`).
- **CREATE2**: `keccak(0xff ‖ sender ‖ salt ‖ keccak(initcode))[12:]` (`evm.go:578-582`) — 표준. SDK는 배포 tx 빌더 + CREATE2 결정적 주소 계산기를 제공.
- **유일 제약**: blacklisted 호출자의 배포 거부(Anzeon, `evm.go:480-482`). 배포에 authorization은 **불필요** — 비-blacklist 계정이면 누구나 배포 가능.

---

## 8. Gas / Fee Protocol

### 8.1 가스 예측

- `eth_estimateGas`는 스톡(`internal/ethapi/api.go:1248-1292`, `eth/gasestimator`). authorized-tip 오버라이드·0x16은 가스 **양**에 영향 없음 → SDK 예측에 특수 처리 불필요.
- 단 0x16은 **구성** 시 feePayer + 이중서명이 필요(예측과 별개).

### 8.2 Anzeon 가스팁 정책 (SDK 필수 동작)

`eth/gasprice/anzeon.go`: **비인증(authorized=false) 계정의 `gasTipCap`은 거버넌스 GasTip(블록 헤더 `WBFTExtra.GasTip`, `core/types/istanbul.go`)으로 강제**된다. 인증 계정은 자유 설정.

> **규정:** SDK는 팁을 임의 지정하지 말고 `eth_maxPriorityFeePerGas`/`eth_gasPrice`(anzeon.go 반영)를 조회해 사용한다. 그래야 가격·멤풀 정렬이 노드 정책과 일치한다.

### 8.3 tx 빌더 안전 가드 (취약점/실패 예방)

Anzeon 활성 시 노드가 거부하는 전송을 SDK 빌더가 **사전 차단**한다:

| 조건 | 노드 에러 | 근거 |
|------|----------|------|
| zero address(`0x0`)로 value 전송 | `ErrZeroAddressTransfer` | `core/vm/evm.go:~213` |
| precompile/native manager로 value 전송 | `ErrValueTransferToPrecompile` | `evm.go:~217` |
| blacklisted from/to/feePayer | `ErrBlacklistedAccount` | `core/state_transition.go:505-516,579` |

blacklist는 사전 `eth_getProof.extra` 확인 권장(§4.5).

---

## 9. 시스템계약 레지스트리

주소는 **config 기반**(genesis `AnzeonConfig.SystemContracts`)이며, SDK는 체인 config에서 읽되 아래 기본값을 fallback으로 둔다.

| 주소(기본) | 컨트랙트 | SDK 용도 |
|-----------|----------|---------|
| `0x1000` | NativeCoinAdapter (ERC-20 base coin, EIP-2612/3009) | 토큰 전송/permit (응용확장) |
| `0x1001` | GovValidator | admin (확장) |
| `0x1002` | GovMasterMinter | admin (확장) |
| `0x1003` | GovMinter (v1/v2=Boho) | admin (확장) |
| `0x1004` | GovCouncil | blacklist/authorize 관리 주체 (참고) |
| `0xB00002` | NativeCoinManager | 참고 (전송 대상 금지) |
| `0xB00003` | AccountManager | `isAuthorized/isBlacklisted` 읽기 |

atomic 코어는 `0xB00003` 읽기만 필수. 나머지는 응용확장(사이클 2·3)에서 사용.

---

## 10. Protocol 스펙 형식과 버전관리

| 대상 | 형식 | 비고 |
|------|------|------|
| JSON-RPC 표면 | OpenRPC 문서 | 스톡 eth_* + 0x16 서명 메서드 |
| tx 봉투/sighash | 스키마 + 산문 규정(§5) | 0x16 이중서명 명시 |
| Account/Extra 비트맵 | 상수 테이블(§4.2) + 디코딩 규칙(§4.6) | 버전드 |
| SigningScheme | `scheme@version` 식별자 | §6 |
| 시스템계약 | 주소 레지스트리 + ABI JSON | config override 지원 |
| 일치 보증 | 골든 테스트 벡터 | §14 |

- 버전관리: protocol semver + capability negotiation. 노드 마이너 변경이 스펙 버전을 안 올리면 SDK 무변경 동작.
- 스펙은 **단일 소스**로 두고 언어별 SDK가 이를 참조/검증.

---

## 11. 보안 모델

| 원칙 | 적용 |
|------|------|
| bespoke 암호 최소화 (ADR-0001) | Go·TS 모두 표준은 permissive lib 사용, **0x16·Extra만 clean-room**(bespoke = 벡터 검증 2개 로직). go-stablenet import 안 함 |
| 감사 표면 집중 | 서명/키/봉투 로직은 각 언어에서 얇게, 골든 벡터로 노드 일치 강제. 언어별 레이어는 비암호 |
| 키 저장 | 플랫폼 보안저장 위임(사이클1: OS 키체인/파일, 브라우저 WebCrypto/IndexedDB). 코어는 키를 영속화하지 않음 |
| 안전 가드 | §8.3 전송 가드 + blacklist 사전확인 |
| 일치 검증 | 골든 벡터로 Go/TS·노드 대비 서명 일치 보증(§14) |
| 미사용 코드 배제 | Cancun signer 등 미빌드/미사용 경로에 의존하지 않음 |

---

## 12. 응용확장 모델 (후속 사이클, 코어 밖)

atomic 코어는 서명·계정 프리미티브만. 아래는 코어 API를 **조립**해 구성하며 코어를 수정하지 않는다.

- **StableNet 고유 확장**: NativeCoinAdapter 토큰 전송/permit(EIP-2612)/transferWithAuthorization(EIP-3009), fee delegation 헬퍼(feePayer 서비스), 계정 상태 대시보드.
- **거버넌스 확장(admin)**: minter/master-minter/validator/council 흐름.
- **P256/passkey 확장**: 컨트랙트-레벨 P256VERIFY 활용(§6).

---

## 13. 저장소 / 테스트 / Conformance

### 13.1 모노레포 구조(제안)

```
/spec              protocol 스펙(단일 소스): OpenRPC, 비트맵, sighash 규정, 주소 레지스트리
/core-go           Go 공유 보안 코어 (go-stablenet core/types·crypto 재사용)
/sdk-go            Go 얇은 SDK (관용 API, RPC, 상태 쿼리)
/sdk-ts            TypeScript SDK (ox 기반 + 0x16)
/conformance       언어 공통 골든 벡터(입력→기대 서명/주소/봉투)
/bindings          (후속) wasm, gomobile
```

### 13.2 Conformance 벡터 (drift 차단)

- 0x16 이중서명, 각 tx type의 sighash·봉투, CREATE2 주소, Extra 디코딩을 **골든 벡터**로 고정.
- 벡터의 **오라클 = go-stablenet 노드**(`transaction_signing_test.go` 등에서 파생) → Go/TS 모두 동일 벡터로 CI 검증.
- 새 tx type/스킴 추가 시 벡터 우선 확장(TDD).

---

## 14. 준비작업 목록 (사이클 1 착수 전)

| # | 작업 | 산출물 | 선행 |
|---|------|--------|------|
| P1 | protocol 스펙 v0 골격 작성 | `/spec`: 비트맵(§4)·0x16 sighash(§5)·주소 레지스트리(§9)·OpenRPC | 본 문서 |
| P2 | go-stablenet에서 골든 벡터 추출 | `/conformance` 초기 벡터(0x16, sighash, CREATE2, Extra) | P1 |
| P3 | Go 코어 스캐폴드 (clean-room 0x16·Extra + permissive secp256k1 lib, ADR-0001) | `/core-go` 뼈대 | P1 |
| P4 | Go SDK: 키/계정/전 tx type/0x16/상태쿼리/RPC | `/sdk-go` MVP | P2,P3 |
| P5 | TS SDK: ox 기반 + 0x16 + 상태쿼리 | `/sdk-ts` MVP | P2 |
| P6 | Conformance CI (Go/TS ↔ 벡터) | CI 파이프라인 | P4,P5 |
| P7 | 보안 리뷰(서명/키/봉투/가드) + 위협 모델 | 리뷰 문서 | P4,P5 |

---

## 15. 착수 전 결정 (확정됨 2026-07-19, ADR 참조)

| # | 항목 | 결정 | ADR |
|---|------|------|-----|
| 1 | 라이선스/의존 | Permissive(MIT/Apache) + clean-room(0x16·Extra만), go-stablenet import 안 함 | [ADR-0001](../../adr/ADR-0001-go-stablenet-dependency-and-license.md) |
| 2 | 저장소 | 모노레포 | [ADR-0002](../../adr/ADR-0002-repo-topology.md) |
| 3 | 키 저장 | 최소 KeyStore + 2백엔드(Go 파일 keystore, TS WebCrypto 비추출) | [ADR-0003](../../adr/ADR-0003-key-storage-abstraction.md) |
| 4 | TS 코어 | ox/noble 네이티브 + 0x16·Extra만, WASM 미사용 | [ADR-0004](../../adr/ADR-0004-typescript-core-strategy.md) |

잔여(착수 시 확정): SDK 최종 라이선스 텍스트(MIT vs Apache-2.0).

---

## 16. 참조 앵커 (go-stablenet, 빌드 참여 코드)

- 계정 상태: `core/types/state_account.go:31-38`, `core/types/state_account_extra.go:33-108`, `core/state/statedb.go`(GetExtra/IsAuthorized/IsBlacklisted)
- 상태 쿼리 RPC: `internal/ethapi/api.go:734,830`(getProof.extra), `core/vm/native_manager.go:349-368,438-457`(0xB00003 getter)
- 0x16 tx: `core/types/tx_fee_delegation.go:27-34,151-178`, `core/types/transaction_signing.go:342-390,439-448`
- 배포: `core/vm/evm.go:480-482,572-582`
- 가스: `internal/ethapi/api.go:1248-1292`, `eth/gasprice/anzeon.go`, `core/types/istanbul.go`(WBFTExtra.GasTip)
- 안전 가드: `core/vm/evm.go:~213-217`, `core/state_transition.go:505-516,579`
- 시스템계약: `params/config_wbft.go:31-46`, `params/protocol_params.go:219-220`, `systemcontracts/*`
- 체인 파라미터: `params/config.go`(chainId 8282/8283, Applepie/Anzeon/Boho)
