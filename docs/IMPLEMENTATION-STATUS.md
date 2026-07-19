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
- 결과: **PASS 27 / UNSUPPORTED 1 / FAIL 0**.

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
| **token NativeCoinAdapter.balanceOf** | PASS | `0x1000` eth_call, 네이티브 잔고와 일치 |
| **token NativeCoinAdapter.transfer(ABI)** | PASS | ABI calldata + wallet.Execute, 수취인 balanceOf 확인 |
| **token EIP-2612 permit** | PASS | 오프체인 서명 → 온체인 allowance 설정(DOMAIN_SEPARATOR/nonces 사용) |
| **hdwallet BIP-39/44 파생** | PASS | known-answer(MetaMask 기본 `0x9858…`) + 파생계정 온체인 거래 |
| **token EIP-3009 transferWithAuthorization** | PASS | gasless, 제3자 relay |
| **governance read 바인딩** | PASS | GovValidator.validatorCount=4·isValidator, minter/blacklist 카운트 |

> 노드가 SDK 서명을 수락·채굴했다는 것은 sighash·RLP·봉투·이중서명·EIP-7702 위임이 노드와 **정확히 일치**함을 authoritative하게 증명한다. 0x03 Blob은 SDK가 올바른 tx를 만들지만 이 체인이 4844(Cancun)를 채택하지 않아 거부된다 — 스펙 `params.md`(Cancun 미채택)와 일치.

## 남은 항목 (후속)

| 항목 | 상태 | 비고 |
|------|------|------|
| **EIP-712/EIP-191 서명 헬퍼** | ✅ 완료 | `account.SignPersonal/SignTypedData`, 공식 벡터 검증 |
| **고수준 facade** | ✅ 완료 | `wallet` 패키지(SendCoin/Deploy/SendFeeDelegated/Call, auto nonce/gas/tip) |
| **blacklist 사전조회 통합** | ✅ 완료 | `wallet.guardTransfer`가 sender/recipient blacklist 확인 |
| **CREATE2 라이브 배포** | ✅ 완료 | 팩토리 컨트랙트로 온체인 CREATE2 실증(`cmd/e2e`) |
| **conformance 골든 벡터 + CI(P6)** | ✅ 완료 | `conformance/vectors/core.json` + 러너 + `.github/workflows/ci.yml` |
| tx 0x03 Blob | **범위 제외** | go-stablenet는 blob(EIP-4844)을 지원하지 않음. SDK는 tx를 만들 수 있으나 체인이 수락하지 않으므로 스코프에서 제외 |
| transport/wallet 유닛 테스트 | ✅ 완료 | httptest mock JSON-RPC로 오프라인 검증 |
| KeyStore OS 키체인/HSM/모바일 백엔드 | 파일 keystore만(ADR-0003 사이클1 범위) | 사이클 2 |
| **ABI 인코더 + NativeCoinAdapter 바인딩** | ✅ 완료 | `abi`·`token` 패키지, balanceOf/transfer 라이브 검증 |
| **EIP-2612 permit** | ✅ 완료 | 라이브 검증(off-chain 서명 → allowance) |
| **HD 지갑/니모닉(BIP-39/32/44)** | ✅ 완료 | `hdwallet` 패키지, known-answer + 라이브 |
| **transferWithAuthorization(EIP-3009)** | ✅ 완료 | 라이브 검증(gasless, 제3자 relay) |
| **TypeScript SDK (코어)** | ✅ 완료 | **별도 저장소 `0xmhha/accounts-ts`**(ADR-0002 정정). Go와 동일 골든 벡터 6종 전부 통과. 파리티는 후속 |
| 거버넌스 admin(minter/master-minter/validator/council) | 미구현 | 사이클 2 잔여(멀티시그 흐름) |
| **KeyStore 저장 백엔드** | ✅ 완료 | `vault` 패키지: pluggable Backend + Memory/File(테스트) + macOS Keychain(darwin, 컴파일·로직 검증). HSM/기타는 Backend 구현으로 추가 |
| TS 기능 파리티(전 tx type/keystore/transport/wallet/hdwallet/token) | 코어만 | 별도 repo `accounts-ts`에서 후속 |
| **모바일 wrapper(gomobile-safe)** | ✅ 완료(코드) | `mobile` 패키지 Go 테스트 통과. 네이티브 AAR/XCFramework 생성·실기기 검증은 툴체인 필요(mobile/README) |
| **거버넌스 read 바인딩** | ✅ 완료 | `governance` 패키지(validator/minter/blacklist 조회), 라이브 검증 |
| 거버넌스 admin write(mint 등) | 미구현 | 온체인 멀티시그 흐름(SDK 범위 밖) |

> no silent caps: 체인이 지원하는 모든 기능(전 tx type 중 0x03 제외, 계정·서명(EIP-191/712 포함)·암복호·배포·7702·상태쿼리·고수준 facade)은 라이브로 검증 완료. 0x03은 체인 한계이며 SDK는 올바른 tx를 생성한다.
