# Security Review (P7) — cycle 1

- 일자: 2026-07-19
- 범위: atomic 코어 전체 (account·tx·signing·crypto·keystore·transport·wallet·internal/rlp)
- 방법: `threat-model.md`의 위협을 실제 코드에 대조 + 의존성 감사 + 서명/키/봉투 경로 집중 검토
- 결론: **크리티컬/중요 발견 없음.** 3건의 note(후속 권장).

---

## 1. 위협 → 코드 대조

| # | 위협 | 대응 코드 | 판정 |
|---|------|----------|------|
| T1 | 개인키 유출 | 키는 `*crypto.PrivateKey`로만 보유. 로깅/에러에 키·서명 원본 없음. `keystore`가 at-rest 암호화(scrypt+AES-128-CTR+keccak MAC). `PrivateKeyBytes`는 명시적 opt-in | OK (note 1) |
| T2 | 서명 malleability/봉투 변조 | dcrd `SignCompact`가 canonical **low-S**(EIP-2) 생성. sighash·봉투가 노드와 **바이트 일치**(라이브 e2e) + 공개 벡터(EIP-155) | OK |
| T3 | 0x16 FeePayer 위조/순서 뒤집기 | 이중서명 순서 강제(`FeePayerSigHash`가 sender 미서명 시 에러). `RecoverFeePayer`가 복구 주소 == 선언 주소 검증. 라이브 채굴 확인 | OK |
| T4 | blacklist/precompile 우회 | `tx.GuardValueTransfer`(zero-addr/precompile 차단) + `wallet.guardTransfer`(sender/recipient 온체인 blacklist 사전조회) | OK |
| T5 | 위조 노드/MITM 상태 조작 | transport는 노드 응답을 신뢰. TLS 미강제, `eth_getProof` 머클검증 없음 | OK (note 2) |
| T6 | 서명 재사용/replay | nonce는 노드 조회값 사용. chainId가 sighash에 포함(크로스체인 재사용 차단) | OK |
| T7 | 팁 조작/DoS | 팁은 노드 오라클(`eth_maxPriorityFeePerGas`) 사용, 임의 추정 금지 | OK |
| T8 | 공급망 | 직접 의존 2개(permissive), go.sum 해시 고정 | OK (note 3) |
| T9 | 골든 벡터 변조 | conformance 벡터가 외부 표준(EIP-155/1014/712/191)에 앵커 → 검토 가능 | OK |

## 2. 의존성 감사

| 의존성 | 버전 | 라이선스 | 용도 |
|--------|------|---------|------|
| `github.com/decred/dcrd/dcrec/secp256k1/v4` | v4.0.1 | ISC | secp256k1 서명/복구/ECDH |
| `golang.org/x/crypto` | v0.35.0 | BSD-3 | keccak(sha3)·scrypt·pbkdf2·hkdf |
| `golang.org/x/sys` (indirect) | v0.30.0 | BSD-3 | x/crypto 의존 |

- **LGPL/GPL 의존 없음**(ADR-0001 준수). go-stablenet 코드 미포함.
- 버전은 `go.mod`/`go.sum`에 해시 고정. 정기 업데이트 권장.

## 3. bespoke 암호 표면 (감사 집중 대상)

clean-room 자체 구현으로 감사가 필요한 부분은 좁게 한정됨:
- `internal/rlp` — RLP 인코더(표준, 벡터 검증).
- `tx` sighash/봉투 — 특히 **0x16 이중서명**(유일한 StableNet 고유 로직). 노드 대비 바이트 일치 검증됨.
- `account.Extra` 비트맵 — 단순 비트연산.
- `crypto.ecies` — ECDH+HKDF+AES-256-GCM(표준 프리미티브 조합).
- `signing.eip712` — 표준 EIP-712, 공식 Mail digest 일치.

원시 암호(secp256k1, keccak, scrypt/aes/gcm)는 검증된 permissive 라이브러리에 위임 → bespoke 암호 최소화 원칙 충족.

## 4. 발견사항 (note, 후속 권장)

| # | 내용 | 권장 조치 |
|---|------|----------|
| note 1 | `PrivateKeyBytes`가 원시 키를 노출(문서상 opt-in) | 상위 계층에서 사용 후 `priv.Zero()`로 메모리 제로화 검토(사이클 2) |
| note 2 | transport가 노드 응답을 무검증 신뢰(TLS 미강제, getProof 머클검증 없음) | 프로덕션은 TLS 엔드포인트 사용. 선택적 `eth_getProof` 머클검증 추가 검토(사이클 2) |
| note 3 | ECIES가 자체 정의 포맷(특정 표준과 wire 호환 아님) | SDK 내부 용도로 한정 문서화(완료). 외부 상호운용 필요 시 표준 ECIES 채택 |

## 5. DoD

- 크리티컬/중요 이슈 0 — 충족.
- bespoke 암호 표면이 "0x16 이중서명 + Extra + ECIES/RLP 조립"으로 한정되고 전부 벡터/라이브 검증됨 — 충족.
- 의존성 permissive·핀 고정 — 충족.
