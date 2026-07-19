# StableNet Accounts Protocol — v0

> 노드↔SDK 계약(contract)의 단일 소스(single source of truth). go-stablenet accounts SDK(Go/TypeScript/…)의 모든 언어 구현이 이 스펙을 참조·검증한다.

- 스펙 버전: `protocol/v0`
- 대상 노드: go-stablenet `dev` (geth 포크, WBFT PoA, stablecoin base coin)
- 상태: Draft (사이클 1 착수 전 문서)
- 문서 유형: Reference (정확·완전·일관 형식)

---

## 왜 이 스펙이 존재하는가 (설계 의도)

이 프로젝트의 최우선 품질은 **보안성**과 **유지보수성**이다. 다언어 SDK가 노드 내부 구현에 직접 결합하면 노드 변경이 모든 구현으로 전파된다. 이를 막기 위해 노드와 SDK 사이에 **명시적 계약**을 둔다:

- 노드가 바뀌어도 이 스펙의 버전이 오르지 않으면 SDK는 무변경으로 동작한다.
- SDK는 노드의 임의 내부 코드가 아니라 **이 스펙에 적힌 것만** 재현한다.
- 스펙에 없는 노드 동작(미사용/미빌드 경로 포함)에 의존하지 않는다.

## 근거 규율 (normative)

1. 이 스펙은 go-stablenet에서 **실제 바이너리에 빌드되는 코드**(`.claude/docs/build-source-files.md` 기준)만을 근거로 한다.
2. 미채택 경로(예: Cancun signer, `CancunTime=nil`)에 의존하지 않는다.
3. 모든 상수·공식·주소에는 노드 소스의 file:line 앵커를 병기한다.
4. 정답 오라클은 노드다. 스펙의 모든 계산은 골든 벡터(§conformance)로 노드와 대조된다.

## 파일 색인

| 파일 | 내용 | 상태 |
|------|------|------|
| [`account.md`](./account.md) | **StateAccount 구조체 + `Extra` 비트맵 + 계정 상태 쿼리** (핵심) | normative |
| [`transactions.md`](./transactions.md) | 지원 tx type 전체 + `0x16` 수수료위임 이중서명 | normative |
| [`signing.md`](./signing.md) | `SigningScheme` 추상화 + `secp256k1@1` | normative |
| [`system-contracts.md`](./system-contracts.md) | 시스템계약 주소 레지스트리 + `0xB00003` getter | normative |
| [`params.md`](./params.md) | chainId, 하드포크 게이트 | normative |
| [`rpc.md`](./rpc.md) | SDK가 사용하는 JSON-RPC 메서드 | normative |

## 버전관리와 capability negotiation

- **버전 체계**: `protocol/vN`. Breaking change(봉투/서명/비트 의미 변경)는 major 증가.
- **후방호환 규칙(§account 비트맵 참조)**: 알 수 없는 항목은 **관용 디코딩**(무시), **엄격 인코딩**(정의된 것만 생성). 노드가 새 예약 비트/필드를 채워도 구 SDK는 깨지지 않는다.
- **capability negotiation**: SDK는 `eth_chainId`로 체인을 식별하고, 필요한 스펙 버전을 자기 지원 범위와 대조한다. 미지원 버전 항목을 만나면 안전하게 거절(fail-closed)한다.

## 용어

| 용어 | 의미 |
|------|------|
| Sender | 트랜잭션을 발신하고 value/nonce의 주체가 되는 계정 |
| FeePayer | `0x16` 수수료위임 tx에서 가스비를 대납하는 계정 |
| Extra | `StateAccount`에 추가된 64비트 상태 플래그 워드 |
| Authorized | Anzeon 가스팁 정책상 자유 팁 설정 권한(서명권한 아님) |
| System contract | 거버넌스/코인 시스템계약(`0x1000`~`0x1004`) |
| Native manager | EVM-내장 precompile(`0xB00002`/`0xB00003`) |

## RFC-2119 키워드

이 스펙에서 MUST / MUST NOT / SHOULD / MAY 는 RFC 2119 의미로 쓴다.
