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

## 라이브 검증 (chainbench + go-stablenet) — 능력 매트릭스

- 네트워크: chainbench `default`, go-stablenet `gstable` v1.1.0, 5노드 WBFT, chainId **8283**(Testnet).
- 실행: `make live-e2e` (또는 `go run ./cmd/e2e -keystore <preset> -password 1`).
- 결과: **PASS 19 / UNSUPPORTED 1 / FAIL 0**.

| 항목 | 결과 | 근거 |
|------|------|------|
| keystore.Decrypt (실 노드 keystore) | PASS | 자금 계정 복호화 |
| transport: Balance/GasPrice/**MaxPriorityFeePerGas(Anzeon)**/EstimateGas/Call | PASS | 라이브 조회 |
| transport.AccountFlags(`eth_getProof.extra`) | PASS | authorized/blacklisted 조회 |
| tx 0x00 Legacy | PASS | 채굴·잔고 확인 |
| tx 0x01 AccessList | PASS | 채굴·잔고 확인 |
| tx 0x02 DynamicFee | PASS | 채굴·잔고 확인 |
| **tx 0x16 FeeDelegate(이중서명)** | PASS | 채굴·잔고 확인, sender/feePayer 분리 |
| **tx CREATE(배포)** | PASS | 배포 주소 == `CreateAddress(sender,nonce)` == 영수증 contractAddress, code 존재 |
| **tx 0x04 SetCode(EIP-7702)** | PASS | authority 코드 == `0xef0100‖delegate` (위임 성공) |
| **tx CREATE2(팩토리 배포·호출)** | PASS | child 주소 == `CreateAddress2(factory,salt,initCode)`, 온체인 일치 |
| tx 0x03 Blob | **UNSUPPORTED** | 노드 거부: `type 3 rejected, pool not yet in Cancun` — **체인이 Cancun 미채택**(SDK 결함 아님) |
| crypto ECIES 암복호(offline) | PASS | 라운드트립 |
| **signing EIP-191 personal_sign** | PASS | known-answer + 서명자 복구 |
| **signing EIP-712 typed data** | PASS | 공식 Mail digest + 서명자 복구 |
| **wallet.SendCoin(auto nonce/gas/tip + blacklist guard)** | PASS | 채굴·잔고 확인 |
| **wallet.Deploy** | PASS | 배포·code 확인 |

> 노드가 SDK 서명을 수락·채굴했다는 것은 sighash·RLP·봉투·이중서명·EIP-7702 위임이 노드와 **정확히 일치**함을 authoritative하게 증명한다. 0x03 Blob은 SDK가 올바른 tx를 만들지만 이 체인이 4844(Cancun)를 채택하지 않아 거부된다 — 스펙 `params.md`(Cancun 미채택)와 일치.

## 남은 항목 (후속)

| 항목 | 상태 | 비고 |
|------|------|------|
| **EIP-712/EIP-191 서명 헬퍼** | ✅ 완료 | `account.SignPersonal/SignTypedData`, 공식 벡터 검증 |
| **고수준 facade** | ✅ 완료 | `wallet` 패키지(SendCoin/Deploy/SendFeeDelegated/Call, auto nonce/gas/tip) |
| **blacklist 사전조회 통합** | ✅ 완료 | `wallet.guardTransfer`가 sender/recipient blacklist 확인 |
| **CREATE2 라이브 배포** | ✅ 완료 | 팩토리 컨트랙트로 온체인 CREATE2 실증(`cmd/e2e`) |
| **conformance 골든 벡터 + CI(P6)** | ✅ 완료 | `conformance/vectors/core.json` + 러너 + `.github/workflows/ci.yml` |
| tx 0x03 Blob 라이브 | 체인 미지원 | go-stablenet가 Cancun/4844 미채택. Cancun 도입 시 재검증(불가) |
| KeyStore OS 키체인/HSM/모바일 백엔드 | 파일 keystore만(ADR-0003 사이클1 범위) | 사이클 2 |
| ABI 인코딩/바인딩(시스템계약 호출) | raw call만 | 사이클 2 응용확장 |
| HD 지갑/니모닉(BIP-32/39/44) | 미구현 | 사이클 2 |

> no silent caps: 체인이 지원하는 모든 기능(전 tx type 중 0x03 제외, 계정·서명(EIP-191/712 포함)·암복호·배포·7702·상태쿼리·고수준 facade)은 라이브로 검증 완료. 0x03은 체인 한계이며 SDK는 올바른 tx를 생성한다.
