# ADR-0003: 키 저장 추상화 범위

## Status

Accepted (2026-07-19) — 최소 KeyStore + 사이클 1 2백엔드(Go 파일 keystore, TS WebCrypto 비추출) 채택.

## Context

보안 최우선 원칙상, 개인키 소재는 서명 경계(`SigningScheme.sign`) 밖으로 노출되지 않아야 하고(스펙 `signing.md` §5), 영속화는 플랫폼 보안저장에 위임한다. 사이클 1(Go+TS) 범위에서 어디까지 추상화·구현할지 정한다.

## Decision

**권장: 최소 `KeyStore` 인터페이스 + 사이클 1은 2개 백엔드만.**

```
interface KeyStore {
    create() -> KeyHandle           // 새 키 생성, 소재는 저장소 내부
    import(material) -> KeyHandle    // 외부 키 반입(명시적, 위험 경고)
    signer(handle) -> SigningScheme  // 서명 시점에만 소재 접근
    // 원시 개인키 export 는 기본 미제공(opt-in, 명시)
}
```

사이클 1 백엔드:
- **Go**: 파일 기반 keystore(암호화) — 서버/CLI 맥락. (OS 키체인/HSM은 후속.)
- **TypeScript(브라우저)**: WebCrypto **비추출(non-extractable) 키** 우선, IndexedDB 핸들. (Node 맥락은 파일 keystore.)

## Consequences

### Positive
- 개인키 소재가 서명 경계에 갇힘 → 유출 표면 최소.
- 플랫폼 네이티브 보안저장 사용(추측성 자체 암호 구현 회피).

### Negative
- `import`/원시 export는 위험 경로라 명시적 opt-in + 경고 필요(사용성↔보안 트레이드오프).
- 비추출 키는 이식성이 낮음(백업/이전 시 별도 흐름 필요).

### Risks
- 잘못된 export API 설계가 키 유출로 이어질 수 있음 → 기본 비활성, 문서·타입으로 위험 표시. 위협모델(`docs/threat-model.md`)에서 재점검.

## Alternatives Considered

### 광범위 백엔드 일괄 구현(OS 키체인·HSM·모바일 Keystore 등)
- 장점: 처음부터 전 플랫폼 커버.
- 단점: 사이클 1 범위 초과, 감사 표면 급증. → 모바일/HSM은 사이클 2로 이연.

### 키 저장을 코어에 내장
- 단점: 코어(보안 크리티컬)에 플랫폼 I/O가 섞여 감사 표면 확대. → 저장은 언어별 SDK 계층으로 분리(설계 §11 일치).
