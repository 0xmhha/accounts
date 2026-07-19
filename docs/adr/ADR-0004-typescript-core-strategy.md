# ADR-0004: TypeScript 코어 전략 (네이티브 lib vs WASM)

## Status

Accepted (2026-07-19) — ox/viem/noble 네이티브 + 0x16·Extra만 자체 구현, WASM 미사용(1차). ADR-0001(clean-room)과 정합.

## Context

TypeScript SDK에서 서명·인코딩을 어떻게 구현할지 정한다. ADR-0001에서 clean-room(Option C)을 권장하면 "Go 코어 재사용"이라는 축이 사라지고, TS는 어차피 독립 구현이 된다. 그럼에도 Go 코어를 WASM으로 공유하는 선택지는 남는다.

## Decision

**권장: `ox`/`viem`/`@noble/*` 네이티브 라이브러리 기반 + `0x16`·`Extra`만 자체 구현. WASM 미사용(1차).**

- 표준 tx(0x00~0x04) 서명/인코딩은 `viem`/`ox`가 제공.
- StableNet 고유 `0x16` 이중서명과 `Extra` 비트맵만 TS로 clean-room 구현.
- 정확성은 Go SDK와 **동일한 골든 벡터**로 보증(크로스언어 일치).

## Consequences

### Positive
- 최고의 TS DX(번들 작음, async init 불필요, 트리셰이킹).
- 성숙한 permissive 암호 lib 사용(MIT) → ADR-0001과 정합.
- Go/TS가 동일 벡터로 수렴 → drift 차단.

### Negative
- 서명 조립 로직이 Go·TS 두 곳에 존재(중복). → 벡터가 유일 진실로서 이를 통제.

### Risks
- 두 구현의 미묘한 불일치. → conformance CI가 회귀를 즉시 검출(`docs/spec/conformance/vectors-schema.md`).

## Alternatives Considered

### Go 코어를 WASM으로 TS에서 공유
- 장점: 서명 로직 단일 소스(Go) → 중복 제거.
- 단점: WASM 번들 무게·async 초기화·디버깅 마찰, ADR-0001이 clean-room이면 Go 코어 자체가 permissive 재구현이라 "노드 정본 공유" 이점도 없음. DX 손해가 큼.
- 채택 조건: 서명 로직 중복을 절대 허용 못 하고 번들 무게를 감수하는 경우. 필요 시 후속 사이클에서 재평가.
