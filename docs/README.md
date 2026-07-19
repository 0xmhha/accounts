# StableNet Accounts SDK — 착수 전 문서 패키지 (핸드오프)

> go-stablenet용 다언어 accounts SDK의 **구현 착수 전** 문서 모음. 실제 구현(착수)은 별도 신규 프로젝트에서 진행하며, 이 문서들을 그대로 이관해 시작한다.

- 작성: 2026-07-19
- 대상 체인: go-stablenet (geth 포크, WBFT PoA, stablecoin base coin; chainId 8282/8283)
- 첫 사이클 범위: **atomic 서명 코어 (Go + TypeScript)**. 모바일·응용확장·지식화/MCP는 후속 사이클.

---

## 1. 이 프로젝트가 무엇인가

블록체인의 상태변화 요청(Transaction)을 각 주체가 private key로 서명해 처리하는 것은 모든 DApp의 공통 허들이다. 이 프로젝트는 **go-stablenet에서 그 서명·계정 처리를 쉽고 안전하게 제공하는 재사용 SDK**를 만든다. Tempo `accounts` SDK가 개념 참조지만, go-stablenet 온체인 실물에 맞춰 **atomic 프리미티브 우선 + 응용확장** 구조로 새로 설계했다.

최우선 품질: **보안성**(취약점 zero) · **유지보수성**(노드 변경이 다언어 구현으로 전파되지 않도록 protocol 추상화).

## 2. 문서 지도 (읽는 순서)

| 순서 | 문서 | 목적 |
|------|------|------|
| 1 | [`superpowers/specs/2026-07-19-…-atomic-core-design.md`](./superpowers/specs/2026-07-19-go-stablenet-accounts-sdk-atomic-core-design.md) | 전체 설계(아키텍처/범위/근거). **먼저 읽기** |
| 2 | [`spec/protocol/v0/README.md`](./spec/protocol/v0/README.md) | 프로토콜 스펙 진입점 |
| 2a | [`spec/protocol/v0/account.md`](./spec/protocol/v0/account.md) | **계정 구조체·Extra 비트맵·상태쿼리 (핵심)** |
| 2b | [`spec/protocol/v0/transactions.md`](./spec/protocol/v0/transactions.md) | 전 tx type + 0x16 이중서명 |
| 2c | [`spec/protocol/v0/signing.md`](./spec/protocol/v0/signing.md) | SigningScheme 추상화 |
| 2d | [`spec/protocol/v0/system-contracts.md`](./spec/protocol/v0/system-contracts.md) | 주소 레지스트리 |
| 2e | [`spec/protocol/v0/params.md`](./spec/protocol/v0/params.md) | chainId/포크/가스팁 |
| 2f | [`spec/protocol/v0/rpc.md`](./spec/protocol/v0/rpc.md) + [`rpc.openrpc.json`](./spec/protocol/v0/rpc.openrpc.json) | JSON-RPC 표면 |
| 3 | [`spec/conformance/vectors-schema.md`](./spec/conformance/vectors-schema.md) | 골든 벡터 스키마·커버리지 |
| 4 | [`adr/`](./adr/) | 착수 전 결정(ADR-0001~0004) |
| 5 | [`threat-model.md`](./threat-model.md) | 위협 모델(보안) |
| 6 | [`plans/2026-07-19-…-cycle1-plan.md`](./plans/2026-07-19-go-stablenet-accounts-sdk-cycle1-plan.md) | 구현 계획 P1~P7 |

## 3. 착수 전 결정 (ADR) — 사용자 확인 필요

| ADR | 주제 | 결정 | 상태 |
|-----|------|------|------|
| [0001](./adr/ADR-0001-go-stablenet-dependency-and-license.md) | go-stablenet 의존 & **라이선스** | Permissive + clean-room(0x16·Extra만), import 안 함 | **Accepted** |
| [0002](./adr/ADR-0002-repo-topology.md) | 저장소 토폴로지 | 모노레포 | **Accepted** |
| [0003](./adr/ADR-0003-key-storage-abstraction.md) | 키 저장 추상화 | 최소 KeyStore + 2 백엔드 | **Accepted** |
| [0004](./adr/ADR-0004-typescript-core-strategy.md) | TS 코어 전략 | ox/noble 네이티브 + 0x16만 | **Accepted** |

> **ADR-0001(라이선스) 확정:** SDK는 **permissive(MIT/Apache)** 배포를 목표로 하며, go-stablenet(LGPL/GPL)을 import하지 않는다. StableNet 고유의 작은 로직(0x16·Extra)만 clean-room 재구현하고 골든 벡터로 노드 일치를 보증한다. 이로써 permissive 배포와 보안(감사 표면 최소)을 동시에 달성한다. 잔여: 최종 라이선스 텍스트(MIT vs Apache-2.0)는 착수 시 확정.

## 4. 착수 방법 (신규 프로젝트에서)

1. 이 `docs/`를 신규 저장소로 이관. `spec/` → 리포 루트 `/spec`, `conformance` 스키마 → `/conformance`. 모노레포(ADR-0002).
2. ADR-0001~0004 확정 완료(Accepted). SDK 라이선스는 permissive(MIT/Apache), go-stablenet import 안 함. 착수 시 MIT/Apache 최종 텍스트만 선택.
3. 계획 P1→P2(스펙 확정 → 노드로 골든 벡터 생성) → P3~P5(clean-room Go/TS 구현) → P6~P7(CI·보안).
4. 모든 구현은 스펙 각 문서의 "검증 대상"을 골든 벡터로 통과해야 한다(TDD).

## 5. 사이클 로드맵

| 사이클 | 내용 |
|--------|------|
| 1 (본 문서) | atomic 서명 코어 (Go+TS), 프로토콜 스펙 v0, conformance |
| 2 | 모바일(Android/iOS) 바인딩 + 응용확장(NativeCoinAdapter 토큰/permit, fee-delegation 헬퍼, 거버넌스 admin) |
| 3 | 코드 지식화 + MCP 서버 (기존 cks/domain-pack 재사용 여부 검토) |

## 6. 근거 규율

모든 스펙·설계는 go-stablenet `dev`의 **빌드 참여 코드**(`go-stablenet/.claude/docs/build-source-files.md`)만을 근거로 하며, 미사용/미빌드 경로(예: Cancun signer)에 의존하지 않는다. 정답 오라클은 노드다.
