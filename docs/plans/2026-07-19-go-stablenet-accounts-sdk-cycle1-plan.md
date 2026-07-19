# go-stablenet Accounts SDK — 사이클 1 구현 계획 (P1–P7)

- 작성일: 2026-07-19
- 근거 설계: `docs/superpowers/specs/2026-07-19-go-stablenet-accounts-sdk-atomic-core-design.md`
- 범위: atomic 서명 코어 (Go + TypeScript), 모노레포
- 참조 소스(빌드 참여 코드만): go-stablenet `dev` 브랜치

> 원칙: (1) 골든 벡터의 오라클은 **go-stablenet 노드** — 벡터 먼저, 구현 나중(TDD). (2) bespoke 암호 최소화 — Go는 노드 정본 재사용, TS는 0x16만 추가. (3) 미사용 코드(Cancun signer 등) 의존 금지.

---

## 페이즈 개요

| 페이즈 | 작업 | 선행 | 병렬 |
|--------|------|------|------|
| A. 기반 | P1 스펙 v0 · P2 골든 벡터 | — | P1→P2 |
| B. 코어 | P3 Go 코어 스캐폴드 | P1 | — |
| C. SDK | P4 Go SDK · P5 TS SDK | P2, P3 | P4∥P5 |
| D. 검증 | P6 Conformance CI · P7 보안 리뷰 | P4, P5 | P6∥P7 |

---

## P1 — Protocol 스펙 v0 골격

**목표:** 노드↔SDK 계약을 단일 소스로 고정. 이후 모든 구현이 이 스펙을 참조.

**단계**
1. `/spec` 디렉터리 생성. 버전 `protocol/v0`.
2. `account.md` — StateAccount 필드 + `Extra` 비트맵(비트63 Blacklisted, 62 Authorized, 61+ 예약) + 디코딩 규칙(관용 디코딩/엄격 인코딩) + 쿼리 경로(`eth_getProof.extra` 주력, `0xB00003` 대안). (설계 §4)
3. `transactions.md` — 지원 tx type(0x00~0x04, 0x16) + 각 sighash + **0x16 이중서명 규정**(sender inner-0x02 → feePayer 0x16, 순서 불변). (설계 §5)
4. `signing.md` — `SigningScheme` 인터페이스 + `secp256k1@1` 식별자. (설계 §6)
5. `system-contracts.md` — 주소 레지스트리(기본값 + config override) + `0xB00003` getter selector. (설계 §9)
6. `rpc.openrpc.json` — 사용하는 eth_* 메서드 + `eth_signRawFeeDelegateTransaction`.
7. `params.md` — chainId 8282/8283, 포크(Applepie/Anzeon/Boho, mainnet 전부 block 0).

**산출물:** `/spec/protocol/v0/*`
**수용 기준:** 설계 §4·§5·§9의 모든 상수/공식이 스펙에 1:1 존재. 미사용 경로 미포함. 리뷰어가 스펙만 보고 0x16 tx를 손으로 인코딩할 수 있음.

---

## P2 — go-stablenet 골든 벡터 추출

**목표:** Go/TS가 동일하게 만족해야 할 정답 벡터를 노드에서 뽑는다.

**단계**
1. go-stablenet 내에 일회성 벡터 생성기(`//go:build none` 또는 별도 tool) 작성 — 노드 `core/types`·`crypto` 사용.
2. 생성 대상:
   - 각 tx type의 sighash 및 서명 결과(고정 키·고정 입력).
   - **0x16 이중서명**: (senderKey, feePayerKey, tx 입력) → SenderTx.V/R/S, FV/FR/FS, 최종 raw 봉투.
   - CREATE/CREATE2 주소(고정 sender/nonce/salt/initcode).
   - `Extra` 인코딩/디코딩: `{raw uint64 → {blacklisted, authorized}}` 및 예약 비트 케이스.
   - 주소 파생(privkey → address).
3. 벡터를 언어중립 JSON으로 `/conformance/vectors/*.json` 에 고정. 각 벡터에 스펙 버전 태그.

> 라이선스 주의(ADR-0001): 벡터 생성기는 go-stablenet(GPL) 안의 일회성 도구로, **SDK에 배포되지 않는다.** 배포물은 생성된 JSON 데이터뿐이므로 SDK의 permissive 라이선스에 영향 없음.

**산출물:** `/conformance/vectors/*.json` + 생성기 스크립트(go-stablenet 측, 미배포)
**수용 기준:** 노드 단위테스트(`transaction_signing_test.go` 등)와 교차확인해 벡터가 노드 동작과 일치. 최소 커버리지: 6개 tx type + 0x16 + CREATE2 + Extra 4케이스 + 주소파생.

---

## P3 — Go 코어 스캐폴드 (clean-room, ADR-0001)

**목표:** go-stablenet import 없이 permissive 라이브러리 기반 공유 보안 코어 뼈대.

**단계**
1. `/core-go` 모듈 생성. **go-stablenet 패키지 import 금지(ADR-0001).** 표준 암호/RLP는 permissive lib(예: `github.com/decred/dcrd/dcrec/secp256k1`(ISC), permissive RLP)로.
2. 코어 인터페이스 정의(스펙 P1 기반): `Key`, `Account`, `TxBuilder`, `SigningScheme`, `Encoder`, `Transport`.
3. `secp256k1@1` SigningScheme 구현 — sender/feePayer sighash를 스펙(`transactions.md` §3.3)대로 clean-room 재현.
4. 0x16 봉투 인코딩/디코딩과 Extra 비트맵을 스펙대로 clean-room 구현.

**산출물:** `/core-go` 컴파일되는 뼈대 (permissive 의존만)
**수용 기준:** 코어가 P2 벡터의 0x16·sighash·주소파생·Extra를 통과(Go 단위테스트). 의존성에 LGPL/GPL 없음(라이선스 스캔).

---

## P4 — Go SDK (MVP)

**목표:** 관용 Go API로 atomic 코어 노출.

**단계**
1. `/sdk-go` — 키/계정(생성·파생·임포트), 상태 쿼리(`getAccount`, `getExtraFlags`, `isBlacklisted/isAuthorized` via getProof/0xB00003).
2. TxBuilder: 전 tx type + CREATE/CREATE2 배포 빌더 + **안전 가드**(zero-addr·precompile 전송 차단, blacklist 사전확인). (설계 §8.3)
3. 서명: 전 tx type + 0x16 이중서명(로컬 및 노드측 `eth_signRawFeeDelegateTransaction` 선택 경로).
4. Transport: `eth_sendRawTransaction`, `eth_estimateGas`, 팁은 `eth_maxPriorityFeePerGas` 조회(Anzeon 정책, 설계 §8.2).

**산출물:** `/sdk-go` MVP + 사용 예제 + 단위테스트
**수용 기준:** P2 벡터 통과. localnet(또는 chainbench) 대상 0x02·0x16 tx 전송·조회 e2e 성공.

---

## P5 — TypeScript SDK (MVP)

**목표:** `ox` 기반 + 0x16만 추가한 관용 TS API.

**단계**
1. `/sdk-ts` — 키/계정/상태쿼리(스톡은 `ox`/`viem`, Extra 디코딩은 스펙 비트맵).
2. TxBuilder: `viem`로 0x00~0x04, **0x16 조립·이중서명만 자체 구현**(유일 bespoke) + 안전 가드.
3. Transport: viem client로 send/estimate + 팁 오라클 조회.

**산출물:** `/sdk-ts` MVP + 예제 + 단위테스트
**수용 기준:** P2 **동일 벡터** 통과(Go와 일치). localnet e2e에서 0x16 tx 성공.

---

## P6 — Conformance CI

**목표:** Go/TS가 동일 골든 벡터를 만족함을 CI로 강제 → drift 차단.

**단계**
1. `/conformance` 러너: 각 언어 SDK가 벡터 JSON을 읽어 sighash/서명/봉투/주소/Extra를 재현·대조.
2. CI 파이프라인: PR마다 Go·TS 러너 실행. 벡터 불일치 시 fail.
3. 스펙 버전 태그 검증(벡터 버전 ↔ SDK 지원 버전).

**산출물:** CI 워크플로 + 러너
**수용 기준:** Go·TS 양쪽 그린. 의도적 서명 변조 시 red(회귀 검출 증명).

---

## P7 — 보안 리뷰 + 위협 모델

**목표:** 취약점 zero 목표 검증.

**단계**
1. 위협 모델 문서: 키 취급, nonce, 서명 malleability, 봉투 파싱, blacklist 우회, 팁 조작.
2. 서명/키/봉투/가드 경로 집중 리뷰(공유 코어 + TS 0x16).
3. 의존성 감사(`ox`/`viem` 버전 고정, go-stablenet 참조 버전 핀).

**산출물:** 위협 모델 + 리뷰 결과 + 조치 목록
**수용 기준:** 크리티컬/중요 이슈 0(또는 조치 완료). bespoke 암호 표면이 "0x16 조립"으로 한정됨을 확인.

---

## 착수 순서 (권장)

1. **P1 → P2** 를 먼저 완료(기반). 이 둘이 나머지를 unblock.
2. P3 착수 시 **라이선스/의존 전략(설계 §15-3, P3-1)** 을 반드시 확정.
3. P4 ∥ P5 병렬, 동일 벡터로 수렴.
4. P6 ∥ P7 로 마감.

## 사이클 1 완료 정의 (DoD)

- 전 tx type + 0x16 + CREATE/CREATE2 + Extra 상태쿼리를 Go·TS 양쪽이 지원.
- 동일 골든 벡터를 양쪽이 CI로 통과.
- 안전 가드·팁 오라클 규정 반영.
- 보안 리뷰 통과(크리티컬 0).
- 스펙 `protocol/v0` 고정 및 문서화.

## 후속 사이클(범위 밖, 예고)

- 사이클 2: 모바일(Android/iOS) 바인딩(gomobile/UniFFI) + 응용확장(NativeCoinAdapter 토큰/permit/fee-delegation 헬퍼).
- 사이클 3: 지식화 + MCP 서버(기존 cks/domain-pack 재사용 여부 검토).
