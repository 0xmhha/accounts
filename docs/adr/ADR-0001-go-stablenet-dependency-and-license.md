# ADR-0001: go-stablenet 의존 전략과 라이선스

## Status

Accepted (2026-07-19) — **Option C 채택**: Permissive(MIT/Apache) 배포 목표, go-stablenet import 안 함, 0x16·Extra만 clean-room 재구현 + permissive 암호 lib + 골든 벡터.

## Context

Atomic 코어의 보안 크리티컬 로직(키·서명·봉투)을 어떻게 확보할지 결정해야 한다. 설계 문서(§3/§11)는 **"Go 공유 코어가 go-stablenet 정본 코드(`core/types`, `crypto`)를 직접 재사용 → bespoke 암호 0"** 을 보안 관점에서 권장했다. 그러나 라이선스·패키징 관점에서 다음 제약이 확인되었다.

### 제약 1 — 라이선스 (치명적, [High])

- go-stablenet은 **GPL-3.0**(`COPYING`) / **LGPL-3.0**(`COPYING.LESSER`)이다. geth 계보상 라이브러리 패키지(`core/types`, `crypto` 등)는 LGPL-3.0, `cmd/` 바이너리는 GPL-3.0이다.
- LGPL-3.0 라이브러리를 **import**하면 SDK 배포물에 LGPL-3.0 의무(사용자의 재링크 권리 보장, 해당 부분 소스 제공 등)가 전파된다. **Go는 정적 링크가 기본**이라 LGPL 준수가 회색지대이며(오브젝트 제공/재링크 허용 필요), SDK를 permissive(MIT/Apache)로 배포하려는 목표와 충돌한다.
- 참고: Tempo `accounts`는 Apache/MIT(permissive)라 개념 참조는 자유롭지만, **go-stablenet 코드 재사용은 별개의 카피레프트 문제**다.

### 제약 2 — 모듈 경로 충돌 ([High])

- go-stablenet `go.mod` 모듈명은 `github.com/ethereum/go-ethereum`이다. 이를 의존성으로 `go get`하면 실제 go-ethereum과 경로가 충돌해 naive import가 불가하다. `replace` 지시자나 모듈 리네이밍 포크가 필요하다.

### 완화 요인 — divergent 표면이 작다 ([High])

- StableNet 고유로 재현이 필요한 것은 **(a) `0x16` 이중서명 sighash/봉투, (b) `Extra` 비트맵 디코딩** 뿐이다(스펙 `transactions.md`, `account.md`). 나머지 서명·인코딩은 표준 이더리움이라 **permissive 라이브러리**로 충족된다:
  - Go: `github.com/decred/dcrd/dcrec/secp256k1`(ISC/BSD, geth도 사용), `golang.org/x/crypto`, 표준 RLP(permissive 재구현 또는 permissive lib).
  - TypeScript: `ox`/`viem`/`@noble/*`(MIT).

## Decision

**권장: Option C — clean-room "작은 divergent 부분만 재구현" + permissive 암호 라이브러리.**

go-stablenet 패키지를 import하지 않는다. 표준 이더리움 서명/인코딩은 permissive 라이브러리로 처리하고, StableNet 고유의 작은 로직(0x16 이중서명, Extra 비트맵)만 **스펙(`protocol/v0`)을 보고 clean-room 재구현**한다. 정확성은 **골든 벡터(노드=오라클)** 로 바이트 일치 보증한다.

이로써 (1) SDK를 permissive로 배포 가능(사용자의 원래 라이선스 우려 해소), (2) 모듈 충돌 회피, (3) 보안 표면은 여전히 작고 벡터로 검증됨.

> 주의: 이 결정은 설계 §3/§11/§15-3의 "노드 정본 재사용(bespoke 0)" 전제를 **라이선스 관점에서 대체**한다. "bespoke 0" 대신 "bespoke = 작고 벡터로 검증되는 2개 로직"이 된다. 보안 목표(감사 표면 최소)는 유지된다.

## Consequences

### Positive
- SDK를 MIT/Apache 등 permissive로 배포 가능.
- 모듈 경로 충돌 없음.
- 보안 표면이 작고(2개 로직) 골든 벡터로 검증 가능.

### Negative
- 표준 서명/인코딩도 permissive 라이브러리에 의존(직접 노드 코드 재사용 아님) → 표준 부분도 벡터 검증 필요.
- clean-room 재현이므로 "노드 코드 그 자체"라는 강한 보증은 없음 → 골든 벡터 커버리지가 품질의 핵심.

### Risks
- 골든 벡터가 불완전하면 노드와 미묘한 불일치 가능. → 벡터 스키마(`docs/spec/conformance/vectors-schema.md`)로 커버리지 강제.
- LGPL 판단은 법률 자문 대상. clean-room을 택하면 이 리스크 자체가 제거된다.

## Alternatives Considered

### Option A — go-stablenet import (replace 지시자/포크)
- go.mod `replace`로 리네이밍 포크를 물려 `core/types`·`crypto`를 직접 재사용.
- 장점: bespoke 0(보안 최강), 노드 변경 컴파일타임 감지.
- 단점/리스크: **SDK가 LGPL-3.0 의무를 상속** → permissive 배포 불가/복잡. 정적 링크 준수 회색지대. 사용자 라이선스 목표와 충돌.
- 채택 조건: SDK를 LGPL-3.0(또는 GPL 호환)으로 배포해도 무방하다고 사용자가 결정하는 경우.

### Option B — 필요한 노드 파일만 vendoring
- 특정 파일만 복사(출처·라이선스 헤더 유지).
- 단점: 복사해도 **LGPL 의무는 그대로** 전파. 제약 1 미해결. 유지보수도 수동. → 라이선스 문제를 풀지 못하므로 비권장.

## 결정 기록

- 2026-07-19: SDK 배포 라이선스 목표 = **Permissive(MIT/Apache)**. 따라서 **Option C** 채택. go-stablenet 패키지 import 금지, 0x16·Extra clean-room 재구현, permissive 암호 lib(Go: dcrd secp256k1 등, TS: ox/noble) 사용, 정확성은 골든 벡터로 보증.
- 후속: 설계 §3/§11/§15-3 정합화 완료. SDK 최종 라이선스 텍스트(MIT vs Apache-2.0)는 착수 시 확정.
