# Chain Params & Forks — v0

> SDK가 알아야 할 chainId와 하드포크 게이트. 서명/트랜잭션 유효성에 영향을 주는 항목만 규정한다.

- 근거 소스: `params/config.go`, `params/config_wbft.go`, `.claude/docs/build-source-files.md` §5

---

## 1. chainId

| 네트워크 | chainId | 근거 |
|----------|---------|------|
| StableNet Mainnet | `8282` | `params/config.go` (`StableNetMainnetChainConfig`) |
| StableNet Testnet | `8283` | `params/config.go` (`StableNetTestnetChainConfig`) |

- SDK는 `eth_chainId`로 실측하고, 서명 sighash의 chainId에 사용한다.
- chainId 포함 방식은 EIP-155/1559 스톡과 동일.

## 2. 하드포크 (계정/tx 관련만)

| 포크 | 활성 블록 | 계정/tx 영향 |
|------|----------|-------------|
| Applepie | 0 | `0x16` 수수료위임 tx 활성화 게이트 |
| Anzeon | 0 | WBFT, 시스템계약 v1, `Extra` 플래그, authorized-tip 정책, blacklist 강제, EIP-7702(`0x04`) |
| Boho | Mainnet 0 / Testnet 100 | GovMinter v2, P256VERIFY(`0x100`) precompile |

- **Mainnet은 세 포크 모두 block 0부터 활성.** 따라서 SDK는 pre-fork 상태를 다루지 않는다.
- **Testnet 주의**: Boho는 block 100부터. block < 100 구간을 대상으로 하는 특수 도구가 아니라면 실무상 무시 가능하나, 스펙 준수 구현은 Boho 이전/이후를 혼동하지 않는다.

## 3. 서명 유효성에 영향을 주는 포크

- **Applepie**: 이전에는 `0x16` tx가 거부된다(`core/state_transition.go:269`). Mainnet block 0 활성이므로 실질 제약 없음.
- **Anzeon**: `0x04`(SetCode/EIP-7702) 활성화 + authorized-tip/blacklist 의미 부여.
- Cancun은 **미채택**(`CancunTime = nil`). Cancun signer 경로에 의존하지 않는다(MUST NOT). → [`signing.md`](./signing.md), [`transactions.md`](./transactions.md).

## 4. 가스팁 정책 (Anzeon) — SDK 동작 규정

- 블록 헤더 `WBFTExtra.GasTip`(`core/types/istanbul.go`)이 거버넌스 투표로 결정되는 기준 팁이다.
- Anzeon에서 **비인증(authorized=false) 계정의 `gasTipCap`은 이 거버넌스 GasTip으로 강제**된다(`eth/gasprice/anzeon.go`). 인증 계정은 자유 설정.
- **규정(MUST)**: SDK는 팁을 임의 지정하지 말고 `eth_maxPriorityFeePerGas` / `eth_gasPrice`(anzeon.go 반영)를 조회해 사용한다. 그래야 가격·멤풀 정렬이 노드 정책과 일치한다.
- authorized 여부는 [`account.md`](./account.md) §5로 조회.

## 5. 검증 대상 (conformance vectors)

- chainId 8282/8283가 각 tx type sighash에 올바로 반영되는지.
- (해당 시) Boho 경계 인식(Testnet block 100)이 스펙 준수 구현에서 혼동되지 않는지.
