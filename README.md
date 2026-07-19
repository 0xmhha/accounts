# stablenet-accounts

**StableNet용 계정·서명 SDK.** 블록체인의 상태변화 요청(Transaction)을 private key로 서명해 처리하는, 모든 DApp의 공통 작업을 [go-stablenet](https://github.com/stable-net/go-stablenet) 위에서 쉽고 안전하게 만드는 재사용 라이브러리.

- 언어(사이클 1): **Go** (`go 1.23.12`, go-stablenet과 호환)
- 라이선스: **Apache-2.0 OR MIT** (dual, permissive)
- 방식: **Spec Driven Development + TDD**. 정답 오라클은 go-stablenet 노드(골든 벡터).

> 이 SDK는 **독립 clean-room 구현**이다. go-stablenet(LGPL/GPL) 코드를 포함·링크하지 않는다. 표준 암호는 permissive 라이브러리를 사용하고, StableNet 고유 로직(0x16 수수료위임 이중서명, 계정 `Extra` 비트맵)만 스펙 기반으로 재구현한다. → [ADR-0001](docs/adr/ADR-0001-go-stablenet-dependency-and-license.md).

## 목적

블록체인에서 트랜잭션 서명·계정 처리는 모든 DApp이 반복 구현해야 하는 허들이다. 이 SDK는 그 원자적(atomic) 프리미티브를 제공해, go-stablenet 위에서 서비스를 만드는 팀이 계정 관련 구현 단계를 대폭 줄이도록 한다. AI-에이전트 기반 개발을 돕는 지식화·MCP 지원은 후속 사이클 목표다.

## 무엇을 제공하는가 (사이클 1: atomic 코어)

| 기능 | 내용 |
|------|------|
| 키/계정 | secp256k1 키 생성·파생·임포트, 주소 파생 |
| 계정 상태 | nonce/balance/code + **StableNet `Extra` 플래그(blacklisted/authorized)** 쿼리 |
| 트랜잭션 | 전 tx type(0x00~0x04) 조립·서명 + **0x16 수수료위임 이중서명** |
| 배포 | CREATE / CREATE2 결정적 주소 |
| 서명 | `SigningScheme` 추상화(알고리즘 교체 격리) |
| 안전 가드 | zero-addr/precompile 전송·blacklist 사전 차단 |

go-stablenet 온체인 실물상, 세션키·WebAuthn tx 서명·fee-token 필드 등은 존재하지 않으며 스코프에서 제외된다(설계 §2 참조).

## 아키텍처

노드 변경이 SDK로 전파되지 않도록 **protocol 스펙**을 노드와 SDK 사이 계약으로 둔다.

```
[앱 / AI 에이전트] → 관용 API → [Go SDK] → protocol/v0 스펙 → [go-stablenet 노드]
                                     │
                          골든 벡터(노드=오라클)로 정확성 보증
```

- 스펙: [`docs/spec/protocol/v0/`](docs/spec/protocol/v0/README.md) — 계정 구조체([account.md](docs/spec/protocol/v0/account.md))가 핵심.
- 결정: [`docs/adr/`](docs/adr/) (ADR-0001~0004, 모두 Accepted).
- 위협 모델: [`docs/threat-model.md`](docs/threat-model.md).
- 계획: [`docs/plans/`](docs/plans/) (P1~P7).

## 구조

go-ethereum(=go-stablenet가 포크한 원본)·표준 라이브러리와 동일한 **평면 root 라이브러리 레이아웃**을 따른다. root의 각 폴더는 소비자가 직접 import하는 **공개 API**이고, 공개 계약이 아닌 구현 디테일만 `internal/`에 둔다.

```
accounts/
├─ account/     계정 생성·서명(EIP-191/712 포함)·keystore/ECIES (공개)
├─ wallet/      고수준 facade: auto nonce/gas/tip + blacklist 가드, 송금/배포 (공개)
├─ tx/          전 tx type(0x00~0x04, 0x16) + CREATE2 + 안전가드 (공개)
├─ signing/     SigningScheme + EIP-191 personal_sign + EIP-712 typed data (공개)
├─ crypto/      Keccak-256·secp256k1·ECIES (공개)
├─ keystore/    keystore v3 암복호화 (공개)
├─ transport/   JSON-RPC 클라이언트 + 계정 상태 쿼리 (공개)
├─ types/       Address·Hash (공개)
├─ internal/rlp minimal RLP 인코더 (구현 디테일, 비공개)
├─ cmd/e2e/     라이브 e2e 실행기
├─ doc.go       module root 패키지(개요·package map·Version)
└─ docs/        설계·스펙·ADR·위협모델
```

> `/pkg` 컨벤션은 쓰지 않는다 — 라이브러리에는 불필요한 중첩이며 go-ethereum 생태계 관례가 아니다(ADR/README 참조).

**TypeScript SDK는 별도 저장소** [`0xmhha/accounts-ts`](https://github.com/0xmhha/accounts-ts)에 있으며, 이 저장소의 conformance 골든 벡터를 동일하게 통과한다(ADR-0002 정정).

## 예제

| 위치 | 내용 |
|------|------|
| `examples/basic/` | 복사-시작용 스타터(오프라인): 계정 생성·서명·0x02/0x16 tx 빌드·keystore·ECIES·CREATE2. `go run ./examples/basic` |
| `Example*` 함수 (각 패키지 `example_test.go`) | pkg.go.dev에 노출되고 `go test`로 검증되는 사용 예 (account/tx/crypto/keystore) |
| `cmd/e2e/` | 라이브 노드 대상 전체 흐름(계정→서명→전송→상태쿼리). `make live-e2e` |

```bash
go run ./examples/basic     # 오프라인 스타터 실행
go test -run Example ./...  # 예제 코드 검증
go doc github.com/0xmhha/accounts/tx   # 패키지 예제/문서 보기
```

## 개발

```bash
make test        # 유닛 테스트 (오프라인)
make cover       # 커버리지
make vet         # go vet
make fmtcheck    # gofmt 검사
make build       # 전체 빌드
make live-e2e    # chainbench 네트워크 부팅 → e2e → 정리 (원샷)
```

동등한 직접 커맨드: `go test ./...`, `gofmt -l .`, `go vet ./...`, `./scripts/live-e2e.sh`.

- **Spec Driven**: 모든 동작은 `docs/spec/protocol/v0`를 근거로 구현한다. 스펙에 없는 노드 동작(미빌드/미사용 경로)에 의존하지 않는다.
- **TDD**: 각 스펙 문서 말미의 "검증 대상"을 테스트로 먼저 작성하고 구현한다. 크로스체크 오라클은 골든 벡터.
- Go 버전은 go-stablenet과 동일한 `1.23.12`를 선언한다(상위 toolchain으로 빌드 가능).

## 로드맵

| 사이클 | 내용 |
|--------|------|
| 1 (현재) | atomic 서명 코어 (Go), protocol 스펙 v0, conformance |
| 2 | 모바일(Android/iOS) + 응용확장(토큰/permit, fee-delegation 헬퍼, 거버넌스) + TS SDK |
| 3 | 코드 지식화 + MCP 서버 |

## 라이선스

Apache-2.0 또는 MIT 중 선택([LICENSE](LICENSE), [LICENSE-APACHE](LICENSE-APACHE), [LICENSE-MIT](LICENSE-MIT)). 서드파티 고지는 [NOTICE](NOTICE).
