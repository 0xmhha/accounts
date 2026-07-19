# System Contracts Registry — v0

> SDK가 참조/호출하는 시스템계약·네이티브 매니저 주소 레지스트리. atomic 코어는 `0xB00003` 읽기만 필수이며, 나머지는 응용확장에서 사용한다.

- 근거 소스: `params/config_wbft.go:31-46`, `params/protocol_params.go:219-220`, `core/vm/native_manager.go`, `systemcontracts/*`

---

## 1. 시스템계약 (config 기반 주소)

주소는 **genesis `AnzeonConfig.SystemContracts`** 에서 결정된다(합의에 하드코딩되지 않음). SDK는 체인 config에서 읽되, 없으면 아래 기본값을 fallback으로 사용한다(MUST fallback).

| 기본 주소 | 컨트랙트 | 역할 | SDK 용도 |
|-----------|----------|------|---------|
| `0x1000` | NativeCoinAdapter | 네이티브 코인 ERC-20 래퍼 (EIP-2612 permit, EIP-3009) | 토큰 전송/permit — 응용확장 |
| `0x1001` | GovValidator | 검증자 집합 관리 | admin — 응용확장 |
| `0x1002` | GovMasterMinter | 발행자 등록/삭제 | admin — 응용확장 |
| `0x1003` | GovMinter | 스테이블코인 발행/소각 (멀티시그; v1, Boho에서 v2) | admin — 응용확장 |
| `0x1004` | GovCouncil | 블랙리스트·권한 계정 관리 주체 | 참고(상태 변경 주체) |

## 2. 네이티브 매니저 (EVM-내장 precompile, 고정 주소)

| 주소 | 매니저 | 근거 |
|------|--------|------|
| `0xB00002` | NativeCoinManager (mint/burn/transfer — 네이티브 잔고) | `params/protocol_params.go:219` |
| `0xB00003` | AccountManager (blacklist/authorize) | `params/protocol_params.go:220` |

## 3. AccountManager getter (SDK 필수) — normative

atomic 코어가 유일하게 필수로 호출하는 시스템 표면. 상태 변경(blacklist/authorize)은 GovCouncil 전용이지만 **getter는 호출자 제약이 없어** 익명 `eth_call`로 동작한다.

| 메서드 | selector | 반환 | CanRun |
|--------|----------|------|--------|
| `isBlacklisted(address)` | `keccak256("isBlacklisted(address)")[:4]` | 32B 워드, LSB=bool | 제한 없음 (`native_manager.go:349-351`) |
| `isAuthorized(address)` | `keccak256("isAuthorized(address)")[:4]` | 32B 워드, LSB=bool | 제한 없음 (`native_manager.go:438-440`) |

호출 예시는 [`account.md`](./account.md) §5-B 참조. **주력 상태 쿼리는 `eth_getProof.extra`** 이며, 이 getter는 per-flag boolean이 필요할 때의 대안이다.

## 4. 전송 금지 대상 (normative)

Anzeon 활성 시 네이티브 매니저/precompile 주소로의 **value 전송은 거부**된다(`ErrValueTransferToPrecompile`). SDK 트랜잭션 빌더는 `0xB00002`/`0xB00003` 및 기타 precompile로의 value 전송을 사전 차단한다. → [`transactions.md`](./transactions.md) §5.

## 5. ABI 출처

- `NativeCoinAdapter` 등 시스템계약 ABI는 `systemcontracts/solidity/`(소스) 및 `systemcontracts/artifacts/`(아티팩트)에서 파생된다.
- 응용확장(사이클 2·3)에서 토큰/거버넌스 호출이 필요할 때 해당 ABI를 스펙에 편입한다. atomic 코어는 `0xB00003` getter selector만 필요.

## 6. 버전

- GovMinter는 v1(Anzeon)·v2(Boho, burn refund)가 있다. atomic 코어와 무관(admin 흐름). 응용확장에서 포크별 버전을 다룬다.
