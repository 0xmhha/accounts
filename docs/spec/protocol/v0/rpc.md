# JSON-RPC Surface — v0

> SDK가 사용하는 JSON-RPC 메서드. go-stablenet은 커스텀 네임스페이스가 없으며, `eth_*`에 수수료위임 서명 메서드 2개만 추가된다.

- 근거 소스: `internal/ethapi/api.go`, `internal/ethapi/transaction_args.go`, `eth/backend.go`

---

## 1. 완전 클라이언트측 SDK가 쓰는 스톡 메서드

| 메서드 | 용도 |
|--------|------|
| `eth_chainId` | 체인 식별 → 서명 chainId |
| `eth_getTransactionCount` | nonce 조회 |
| `eth_estimateGas` | 가스 양 예측 (스톡) |
| `eth_gasPrice` / `eth_maxPriorityFeePerGas` | 가스 가격/팁 (Anzeon 정책 반영, [`params.md`](./params.md) §4) |
| `eth_getProof` | **계정 `extra` 플래그 조회** ([`account.md`](./account.md) §5-A) |
| `eth_call` | `0xB00003` getter 등 읽기 ([`account.md`](./account.md) §5-B) |
| `eth_sendRawTransaction` | 서명된 raw tx(0x16 포함) 전송 |
| `eth_getTransactionReceipt` | 영수증 |

- `eth_getProof` 응답 `AccountResult`에 StableNet이 `extra`(hex uint64, omitempty)를 추가했다(`internal/ethapi/api.go:734,830`).
- `eth_call`/`eth_estimateGas`의 `OverrideAccount`에 `extra`가 추가되어 authorized/blacklisted 상태 시뮬레이션이 가능하다(`api.go:1035,1052-1053`).

## 2. StableNet 고유 추가 메서드 (선택 — 노드측 대납 서명)

완전 클라이언트측 SDK는 필요 없다. 노드가 FeePayer로 서명하게 하려는 경우에만 사용한다.

| 메서드 | 설명 | 근거 |
|--------|------|------|
| `eth_signRawFeeDelegateTransaction` | sender-서명된 raw DynamicFee tx + `feePayer`를 받아 `0x16`으로 감싸 노드 키로 대납 서명 | `internal/ethapi/api.go:2189-2235` |
| `personal_signRawFeeDelegateTransaction` | 위의 personal 네임스페이스 버전 | `internal/ethapi/api.go:629+` |

- `TransactionArgs`에 `FeePayer` 필드가 추가되어 있고, sender V/R/S + FeePayer가 모두 있으면 `toTransaction`이 `0x16` tx를 만든다(`transaction_args.go:67`, `:538-558`).
- `0x16` tx의 JSON 출력에는 `feePayer`와 FeePayer v/r/s가 포함된다(`api.go:1411,1474-1489`).

## 3. 규정

- SDK 기본 경로는 **완전 클라이언트측 구성·서명 + `eth_sendRawTransaction`** 이다(MUST 지원). 노드측 대납 서명은 선택 기능으로 노출 MAY.
- 커스텀 네임스페이스(`stablenet_*` 등)는 **존재하지 않는다.** 그런 메서드에 의존하지 말 것(MUST NOT).

## 4. OpenRPC

정식 OpenRPC 문서(`rpc.openrpc.json`)는 사이클 1 P1에서 위 목록을 기계판독 형태로 확정한다. 본 문서는 그 산출 전의 규범 목록이다.
