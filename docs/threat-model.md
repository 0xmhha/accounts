# Threat Model — StableNet Accounts SDK (atomic 코어)

- 작성일: 2026-07-19
- 범위: 사이클 1 atomic 코어(키·서명·봉투·상태쿼리·전송), Go + TypeScript
- 방법: 자산 식별 → 신뢰경계 → STRIDE 위협 → 완화 → 잔여 리스크
- 상태: 설계 단계 위협모델 (구현 착수 전). P7 보안 리뷰에서 코드 대비 재검증.

> 최우선 요구: "코드에서 보안 취약점이 발견되지 않도록." 이 문서는 서명·키 경로를 중심으로 위협을 선제 식별한다.

---

## 1. 자산 (Assets)

| 자산 | 민감도 | 비고 |
|------|--------|------|
| 개인키 소재 | 최상 | 유출 시 자금 탈취 |
| 서명(V/R/S, FV/FR/FS) | 상 | 재사용/malleability 위험 |
| 트랜잭션 내용(to/value/data) | 중 | 무결성·변조 |
| 계정 상태 조회 결과(nonce/extra) | 중 | 신선도·정확성 |
| 골든 벡터 | 상(무결성) | 정답 오라클, 변조 시 전 구현 오염 |

## 2. 신뢰 경계 (Trust Boundaries)

```
[앱/에이전트] ── API ──> [언어별 SDK] ── SigningScheme ──> [보안 코어(키·서명)]
                              │
                              └── JSON-RPC ──> [go-stablenet 노드(신뢰: 상태 오라클)]
                              └── KeyStore ──> [플랫폼 보안저장]
```

- 코어(보안 크리티컬)와 그 외 계층 사이가 1차 경계. 키 소재는 이 경계 안에만.
- 노드는 상태/가스/전송의 오라클로 신뢰하되, 응답은 형식·범위 검증.

## 3. STRIDE 위협 및 완화

| # | 위협(STRIDE) | 시나리오 | 완화 (normative) |
|---|--------------|----------|------------------|
| T1 | Information Disclosure | 개인키가 로그/에러/직렬화로 유출 | 키는 `sign` 경계 밖 노출 금지. 로깅·에러에 키/서명 원본 금지(스펙 signing §5). `KeyStore` 기본 export 비활성(ADR-0003) |
| T2 | Tampering | 서명 malleability / 잘못된 봉투로 무효·재해석 | 표준 secp256k1 malleability 규칙 준수. 봉투/ sighash를 골든 벡터로 노드와 바이트 일치 검증 |
| T3 | Spoofing | `0x16` FeePayer 서명 위조/오더 뒤집기 | 이중서명 순서 불변(sender→feePayer). 로컬 검증에서 `RecoverFeePayer` 미러링, 복구 주소≠선언 주소면 거부(스펙 transactions §3.4) |
| T4 | Elevation/Bypass | blacklisted 주소로 전송 구성해 실패/우회 시도 | 빌더 안전 가드: zero-addr/precompile 전송·blacklist from/to/feePayer 사전 차단(스펙 transactions §5). 사전 `eth_getProof.extra` 확인 |
| T5 | Tampering (state) | 위조 노드/MITM가 잘못된 nonce/extra 반환 | TLS 사용. 가능 시 `eth_getProof` 머클 검증. raw `extra` 보존·범위 검증(ValidateExtra 미러) |
| T6 | Repudiation/replay | 동일 서명 재사용 | nonce는 노드 조회값 사용, 재사용 금지. chainId 포함 sighash로 크로스체인 재사용 차단 |
| T7 | DoS/오용 | 잘못된 팁으로 tx 지연/거부 | 팁은 노드 오라클(`eth_maxPriorityFeePerGas`) 사용(스펙 params §4) |
| T8 | Supply chain | 암호 의존성(ox/noble/secp256k1) 변조 | 버전 핀·해시 고정, 의존성 감사(P7). permissive·성숙 lib만 |
| T9 | Tampering (vectors) | 골든 벡터 변조로 전 구현 오염 | 벡터는 노드에서 생성·서명/해시 기록, PR 리뷰 필수, CI에서 노드 재생성 대조 가능 |

## 4. 미사용/제외 경로 (오검토 방지)

- Cancun signer 경로: 노드 미채택(`CancunTime=nil`). 의존·검토 대상 아님.
- 세션키/access key, WebAuthn tx 서명: 온체인 부재. 코어 범위 밖.
- P256 tx 서명: 없음. P256VERIFY(`0x100`)는 검증 precompile로 응용확장 맥락(스펙 signing §4).

## 5. 잔여 리스크

| 리스크 | 상태 | 대응 |
|--------|------|------|
| clean-room 재현과 노드의 미묘한 불일치 | 개방 | 골든 벡터 커버리지 강화, P6 CI |
| 비추출 키의 백업/이전 흐름 부재 | 개방 | 사이클 2 KeyStore 확장에서 설계 |
| 모바일 보안저장(Keystore/Keychain) 미포함 | 범위 밖 | 사이클 2 |

## 6. P7에서 검증할 항목

- 서명/키/봉투/가드 경로 코드 리뷰(위 T1~T4 중심).
- 의존성 감사(T8), 버전 핀 확인.
- 골든 벡터 커버리지가 스펙 각 문서 §검증 대상을 모두 포함하는지.
- 크리티컬/중요 이슈 0(또는 조치 완료)이 DoD.
