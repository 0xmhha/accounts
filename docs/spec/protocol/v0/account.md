# Account 구조체 Protocol — v0 (핵심)

> go-stablenet의 계정 상태는 스톡 이더리움과 **정확히 한 필드**(`Extra`)가 다르다. 이 문서는 그 divergence와, SDK가 계정 상태를 안전·후방호환적으로 읽는 방법을 규정한다.

- 근거 소스: `core/types/state_account.go`, `core/types/state_account_extra.go`, `core/state/statedb.go`, `internal/ethapi/api.go`, `core/vm/native_manager.go`

---

## 1. StateAccount 구조체 (divergence)

go-stablenet은 스톡 이더리움 `StateAccount {Nonce, Balance, Root, CodeHash}` 에 **`Extra uint64` 한 필드를 추가**한다.

```go
// core/types/state_account.go:31-38
type StateAccount struct {
    Nonce    uint64
    Balance  *uint256.Int
    Root     common.Hash   // storage trie root
    CodeHash []byte
    Extra    uint64 `rlp:"optional"`   // ← StableNet 추가
}
```

### 인코딩 규칙 (normative)

- `Extra`는 RLP `optional` 필드다. `Extra == 0`인 계정은 스톡 geth와 **바이트 단위로 동일한 RLP**로 인코딩된다(후방호환).
- SDK가 계정 상태를 스스로 RLP 인코딩할 일은 일반적으로 없다(상태는 노드가 소유). SDK가 계정 상태 프루프를 검증하는 경우, `Extra == 0`이면 필드를 생략하고, `Extra != 0`이면 마지막 필드로 포함한다.
- 빈 계정(empty) 판정은 nonce/balance/codehash 조건에 더해 **`Extra == 0`** 을 요구한다 (`core/state/state_object.go:95`).

> `Extra`는 `SlimAccount`·`Copy()`·`FullAccount`·`SlimAccountRLP` 에도 일관되게 전파된다.

---

## 2. Extra 비트필드 (protocol 상수) — normative

`Extra`는 64비트 플래그 워드다. **현재 2비트만 정의**되며, 나머지는 예약(reserved)이다.

| 비트 | 마스크(hex) | 이름 | 상수 (`core/types/state_account_extra.go`) |
|------|------------|------|-------------------------------------------|
| 63 (MSB) | `0x8000000000000000` | Blacklisted | `AccountExtraMaskBlacklisted = 1 << 63` (:33) |
| 62 | `0x4000000000000000` | Authorized | `AccountExtraMaskAuthorized = 1 << 62` (:36) |
| 61 | `0x2000000000000000` | Reserved (미정의) | 주석 처리 (:38-40) |
| 60 .. 0 | — | Reserved (미정의) | — |

- **유효 마스크**: `AccountExtraValidMask = Blacklisted | Authorized` (:45).
- **검증**: `ValidateExtra(extra)`는 유효 마스크 밖의 비트가 켜져 있으면 거부한다 (:103-108).
- **헬퍼(불변 패턴 — 새 값을 반환, 원본 불변)**: `Is/Set/Clear{Blacklisted,Authorized}(extra)` (:72-99).

### 참조 디코딩 (언어 중립 의사코드)

```
func decodeExtra(extra uint64) ExtraFlags:
    return {
        blacklisted: (extra & 0x8000000000000000) != 0,   // bit 63
        authorized:  (extra & 0x4000000000000000) != 0,   // bit 62
        raw:         extra,
    }
```

---

## 3. 의미론 (semantics)

### Blacklisted (bit 63)

- Anzeon 활성 시, blacklisted 계정은 트랜잭션 발신/수신, EVM `CALL` 계열, 컨트랙트 생성에서 **차단**된다. → 세부 강제 지점은 [`transactions.md`](./transactions.md) §안전 제약 참조.
- SDK는 트랜잭션 구성 전 관련 주소의 blacklist 여부를 확인해 실패를 사전 예방 SHOULD.

### Authorized (bit 62)

- **가스팁 정책 권한**을 의미한다. Anzeon에서 비인증(authorized=false) 계정은 `gasTipCap`이 거버넌스 GasTip으로 강제되고, 인증 계정은 자유 설정이 허용된다. → [`params.md`](./params.md) / `eth/gasprice/anzeon.go`.
- **서명 권한과 무관**하다. authorized가 계정의 서명 능력이나 트랜잭션 유효성을 바꾸지 않는다.

### 상위 role은 `Extra`에 없다

minter / master-minter / validator / council 같은 상위 권한은 `Extra` 비트가 **아니라** 거버넌스 시스템계약 저장소에 있다. 이를 읽으려면 해당 거버넌스 계약을 `eth_call` 해야 한다([`system-contracts.md`](./system-contracts.md)).

---

## 4. 쓰기 경로 (참고 — SDK는 변경하지 않음)

`Extra` 비트는 **AccountManager precompile `0x…B00003`** 를 통해서만 변경된다.

| 작업 | 허용 호출자 | 허용 Op | 근거 |
|------|------------|---------|------|
| `blacklist(address)` / `unBlacklist(address)` | GovCouncil (`0x1004`) | CALL | `core/vm/native_manager.go` (canRunAccountManager) |
| `authorize(address)` / `unAuthorize(address)` | GovCouncil (`0x1004`) | CALL | 〃 |

- 즉 일반 SDK/사용자는 이 상태를 **바꿀 수 없고 읽기만** 한다. 변경은 GovCouncil 거버넌스 흐름(제안/실행)을 통해서만 일어난다.

---

## 5. 클라이언트 쿼리 경로 (SDK 구현 규정) — normative

SDK가 계정 상태를 읽는 **작동 확인된** JSON-RPC 경로는 두 가지다. 주력은 (A).

### (A) 주력 — `eth_getProof` 의 `extra` 필드

한 번의 호출로 raw 플래그 워드를 얻는다. **미래 비트까지 그대로 노출**되므로 후방호환에 유리하다.

```
요청:  eth_getProof(address, [], "latest")
응답:  result.extra   // hex uint64, 값이 0이면 omitempty 로 생략됨
```

- 근거: `internal/ethapi/api.go:734`(struct 필드), `:830`(`Extra: hexutil.Uint64(statedb.GetExtra(address))`).
- **부재 처리(MUST)**: `extra` 필드가 없으면 `0`으로 간주한다(모든 플래그 false).
- 디코딩: §2 참조 디코딩 사용.

### (B) 대안 — `eth_call` to AccountManager `0x…B00003`

per-flag boolean이 필요할 때 사용한다. getter는 호출자 제약이 없어 익명 `eth_call`로 동작한다.

```
eth_call({ to: "0x0000000000000000000000000000000000B00003",
           data: selector("isBlacklisted(address)") + pad32(address) }, "latest")
  → 32바이트 워드, LSB == 1 이면 true
// isAuthorized(address) 도 동일
```

- 근거: `core/vm/native_manager.go:349-368`(isBlacklisted, `CanRun == nil`), `:438-457`(isAuthorized).
- selector는 `keccak256("isBlacklisted(address)")[:4]` / `keccak256("isAuthorized(address)")[:4]`.

### 사용 금지 경로 (MUST NOT)

- `eth_getAccount` — **이 포크에 존재하지 않는다.** 사용 금지.
- `NativeCoinAdapter(0x1000)` 의 상태 getter — public getter가 없다(내부 `_isBlacklisted`가 `0xB00003`을 staticcall할 뿐). 상태 읽기 대상으로 삼지 말 것.

---

## 6. 후방호환 유지보수 메커니즘 (normative) — 핵심 가치

`Extra`에는 예약 비트(61..0)가 있어 향후 새 플래그가 추가될 수 있다. SDK는 다음을 **반드시** 지킨다:

1. **raw `uint64`를 읽어** 스펙의 비트맵으로 디코딩한다(하드코딩 금지, 스펙 참조).
2. **관용 디코딩**: 정의되지 않은 비트가 켜져 있어도 오류를 내지 않는다. 알려진 비트만 해석하고 나머지는 무시한다. (단, `raw`는 항상 보존해 상위 계층이 필요 시 접근 가능하게 한다.)
3. **엄격 인코딩**: SDK가 `Extra`를 생성/검증하는 경우(예: 상태 오버라이드, 프루프 검증) 정의된 비트만 사용한다.
4. 새 비트 해석은 **스펙 버전을 올린 뒤에만** 추가한다.

이 규칙으로, 노드가 "당장 SDK 업데이트를 요하지 않는" 새 비트를 도입해도 구 SDK는 계속 안전하게 동작한다.

---

## 7. SDK가 노출할 Account API (관용 계층, 권고)

언어별 SDK는 다음에 상응하는 관용 API를 제공 SHOULD. (이름은 언어 관례를 따른다.)

| API | 반환 | 구현 경로 |
|-----|------|----------|
| `getAccount(address)` | nonce, balance, codeHash + `extraFlags` | `eth_getProof` (또는 개별 eth_* + getProof) |
| `getExtraFlags(address)` | `{ blacklisted, authorized, raw }` | §5-A getProof.extra → §2 디코딩 |
| `isBlacklisted(address)` | bool | §5-A 또는 §5-B |
| `isAuthorized(address)` | bool | §5-A 또는 §5-B |

- `getExtraFlags`는 `raw`를 반드시 포함해 미래 비트를 상위에서 다룰 수 있게 한다(§6-2).
- 상태 오버라이드 시뮬레이션이 필요하면 `eth_call`/`eth_estimateGas`의 `OverrideAccount.extra`를 사용할 수 있다(`internal/ethapi/api.go:1035,1052-1053`).

---

## 8. 검증 대상 (conformance vectors)

이 문서를 구현한 SDK는 아래 골든 벡터를 통과해야 한다(오라클=노드):

- `Extra` 인코딩/디코딩: `raw uint64 → {blacklisted, authorized}` 정상 케이스.
- 예약 비트 세팅 케이스(관용 디코딩): 알 수 없는 비트가 켜진 raw → known 플래그만 해석, 오류 없음, `raw` 보존.
- `Extra == 0` 부재 처리: getProof 응답에 `extra` 없음 → 모든 플래그 false.
- 주소 파생: privkey → address (스톡, secp256k1).
