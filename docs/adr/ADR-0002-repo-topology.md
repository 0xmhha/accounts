# ADR-0002: 저장소 토폴로지 (모노레포 vs 폴리레포)

## Status

Accepted (2026-07-19) — 모노레포 채택.

## Context

산출물이 여러 언어 SDK(Go, TypeScript, 후속 Android/iOS) + 단일 프로토콜 스펙 + 공유 골든 벡터로 구성된다. 핵심 유지보수 요구는 **스펙과 벡터를 단일 소스로 두고 모든 언어 구현이 동일 벡터로 수렴**하는 것이다.

## Decision

**권장: 모노레포.**

```
/spec              protocol 스펙(단일 소스) — protocol/v0 이관
/conformance       언어 공통 골든 벡터 + 러너
/core-go           (Option C 시) StableNet 고유 로직 clean-room Go 구현
/sdk-go            Go SDK
/sdk-ts            TypeScript SDK
/bindings          (후속) wasm, gomobile
/docs              설계·ADR·위협모델
```

## Consequences

### Positive
- 스펙·벡터·구현이 한 커밋에서 원자적으로 변경·검증됨 → drift 차단(핵심 요구 충족).
- 크로스언어 conformance CI를 단일 파이프라인으로 구성.
- 스펙 버전과 각 SDK 지원 버전의 정합을 PR 단위로 강제.

### Negative
- 언어별 릴리스/버저닝을 모노레포 내에서 분리 관리해야 함(태그 규칙 필요).
- CI가 다언어 툴체인(Go/Node/후속 모바일)을 모두 포함 → 빌드 매트릭스 복잡.

### Risks
- 모노레포 비대화 → 경로 필터·부분 CI로 완화.

## Alternatives Considered

### 폴리레포 (스펙 저장소 + 언어별 저장소)
- 장점: 언어별 독립 릴리스·권한 분리.
- 단점: 스펙/벡터 동기화가 저장소 경계를 넘어 발생 → **drift 위험 증가**, submodule/패키지 핀 관리 부담. 핵심 요구(단일 소스 수렴)에 불리.
- 채택 조건: 팀/소유권이 언어별로 완전히 분리되고 독립 릴리스가 최우선인 경우.
