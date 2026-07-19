# 구현 현황 (사이클 1, Go)

- 갱신: 2026-07-19
- 방식: Spec Driven Development + TDD. 모든 항목은 `docs/spec/protocol/v0`를 근거로 하며, 검증 오라클은 공개 벡터(EIP-155/EIP-1014/Web3 Secret Storage) 및 **live go-stablenet 노드**(chainbench).
- 의존성: permissive만 — `decred/dcrd secp256k1`(ISC), `x/crypto`(BSD, sha3/scrypt/pbkdf2/hkdf). RLP·StableNet 고유 로직은 clean-room 자체 구현(ADR-0001). go-stablenet import 없음.
- 검증: `go test ./...` 전부 통과, `gofmt`/`go vet` clean. **라이브 e2e 통과**(아래).

## Spec ↔ Code ↔ Test 추적

| Spec / 요구 | 구현 | 테스트(대표) | 상태 |
|------|------|------------|------|
| account.md §2 Extra 비트맵 | `account/extra.go` | `TestDecode`, `TestValidateStrict` | 완료 |
| account.md §5 상태 쿼리(`eth_getProof.extra`) | `transport/client.go` (`AccountFlags`/`IsAuthorized`/`IsBlacklisted`) | live e2e | 완료 |
| **계정 생성**(요구 3) | `account/account.go` (`Generate`/`FromPrivateKey*`/`FromKeystore`) | `TestGenerateAndSign` | 완료 |
| signing.md SigningScheme | `signing/scheme.go`, `crypto/secp256k1.go` | `crypto`(EIP-155 known-answer) | 완료 |
| **전 tx type**(요구 4): 0x00/0x01/0x02/0x03/0x04/0x16 | `tx/{legacy,accesslist,dynamicfee,blob,setcode,feedelegation}.go` | 각 sign/recover + EIP-155/EIP-1014 known-answer | 완료 |
| **서명**(요구 5) | `account.Sign`, 각 tx `Sign` | 전 패키지 | 완료 |
| **암복호화**(요구 6): keystore + ECIES | `keystore/keystore.go`, `crypto/ecies.go` | `TestDecryptOfficialPBKDF2Vector`(geth 호환), ECIES roundtrip/tamper | 완료 |
| transactions.md §4 CREATE/CREATE2 | `tx/create.go` | `TestCreateAddress2`(EIP-1014) | 완료 |
| transactions.md §5 안전가드 | `tx/guard.go` | `TestGuardValueTransfer` | 완료 |
| params.md §4 가스팁 오라클 | `transport` (`MaxPriorityFeePerGas`) | live e2e | 완료 |
| rpc.md RPC 표면 | `transport/client.go` | live e2e | 완료 |

## 라이브 검증 (chainbench + go-stablenet)

- 네트워크: chainbench `default` 프로파일, go-stablenet `gstable` v1.1.0, 5노드 WBFT, chainId **8283**(Testnet).
- 실행: `go run ./cmd/e2e -keystore <preset> -password 1`.
- 결과(모두 on-chain 채굴·status 0x1):
  1. **실 go-stablenet keystore 복호화** → 자금 계정(keystore geth 호환 실증).
  2. **0x00 Legacy** 전송 확인.
  3. **0x01 AccessList** 전송 확인.
  4. **0x02 DynamicFee** 전송 확인.
  5. **0x16 FeeDelegate 이중서명** 전송 확인(sender/feePayer 분리) — StableNet 고유 로직이 노드와 바이트 일치 실증.
  6. `eth_getProof.extra`로 계정 Extra 플래그 조회.

> 노드가 SDK 서명을 수락·채굴했다는 것은 sighash·RLP·봉투·이중서명이 노드와 **정확히 일치**함을 authoritative하게 증명한다.

## 남은 항목 (후속)

| 항목 | 상태 | 비고 |
|------|------|------|
| 0x03 Blob / 0x04 SetCode **라이브** 전송 | 유닛 테스트만 | blob 사이드카(KZG)·7702 위임 셋업 필요 → 라이브는 후속 |
| KeyStore OS 키체인/HSM/모바일 백엔드 | 파일 keystore만(ADR-0003 사이클1 범위) | 사이클 2 |
| blacklist 사전조회 통합 가드 | transport에 조회 API 존재, 빌더 자동 연동은 미결 | 헬퍼로 통합 예정 |
| 상위 SDK facade(고수준 SendTransaction 등) | 저수준 완비 | 편의 API 후속 |
| conformance 골든 벡터 파일/러너(P2/P6) | 라이브 e2e로 대체 검증 | 회귀 자동화는 후속 |

> no silent caps: 위 미구현은 의도적 범위이며, 서명·계정·암복호화·전 tx type의 핵심은 완료·라이브 검증되었다.
