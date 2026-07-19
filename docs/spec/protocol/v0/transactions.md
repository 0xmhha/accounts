# Transaction Protocol — v0

> SDK가 조립·서명·인코딩해야 하는 트랜잭션 타입과, StableNet 고유 `0x16` 수수료위임 이중서명을 규정한다.

- 근거 소스: `core/types/transaction.go`, `core/types/tx_fee_delegation.go`, `core/types/transaction_signing.go`, `core/vm/evm.go`, `core/state_transition.go`

---

## 1. 지원 트랜잭션 타입 (전체)

SDK는 아래 **모든** tx type을 지원한다. `0x16`만이 StableNet 고유이며 나머지는 스톡 이더리움과 동일하다.

| type | 이름 | 표준/고유 | sighash preimage |
|------|------|----------|------------------|
| `0x00` | Legacy | 스톡 | EIP-155 |
| `0x01` | AccessList | 스톡 | EIP-2930: `keccak(0x01 ‖ rlp([...]))` |
| `0x02` | DynamicFee | 스톡 | EIP-1559: `keccak(0x02 ‖ rlp([chainId,nonce,tipCap,feeCap,gas,to,value,data,accessList]))` |
| `0x03` | Blob | 스톡 | EIP-4844 |
| `0x04` | SetCode | 스톡 (EIP-7702, Anzeon) | EIP-7702 |
| `0x16` | **FeeDelegateDynamicFee** | **StableNet 고유** | §3 |

- 타입 상수: `core/types/transaction.go:47-54` (`FeeDelegateDynamicFeeTxType = 0x16`, 십진 22).
- Go 구현은 노드 `core/types`의 해당 타입을 **재사용**한다. TypeScript는 `viem`가 `0x00`~`0x04`를 제공하고 `0x16`만 추가 구현한다.

---

## 2. 서명 원칙 (normative)

SDK는 노드의 signer *선택 로직*을 재현하지 않는다. **tx type별 안정적인 sighash 공식만** 재현한다.

- StableNet 체인 설정에서 노드는 `anzeonSigner`(sender 복구)와 `feeDelegateSigner`(feePayer 복구)를 사용한다. Cancun은 미채택(`CancunTime = nil`)이므로 **Cancun signer 경로에 의존하지 않는다(MUST NOT)**.
- 서명 값 형식: secp256k1 `[R ‖ S ‖ V]`, V는 0/1 정규화 후 복구용 표현을 따른다(`transaction_signing.go:655-663`).
- chainId는 EIP-155/1559 sighash에 스톡 방식으로 포함된다. chainId: [`params.md`](./params.md).

---

## 3. `0x16` FeeDelegateDynamicFeeTx — 이중서명 (유일한 bespoke)

### 3.1 구조

```go
// core/types/tx_fee_delegation.go:27-34
type FeeDelegateDynamicFeeTx struct {
    SenderTx   DynamicFeeTx      // 내부 EIP-1559 tx (자체 V/R/S 포함)
    FeePayer   *common.Address
    FV, FR, FS *big.Int          // FeePayer 의 secp256k1 서명
}
```

### 3.2 봉투 (EIP-2718)

```
envelope = 0x16 ‖ rlp([ <SenderTx: DynamicFeeTx 필드들…>, FeePayer, FV, FR, FS ])
```

- 인코딩/디코딩: `tx_fee_delegation.go:151-156`. 디코딩 디스패치: `transaction.go:234-236`.
- **fee-token 필드는 없다.** StableNet의 stablecoin은 네이티브 base coin이며 가스는 네이티브 잔고에서 지불된다. 어떤 tx에도 별도 가스토큰 필드가 없다.

### 3.3 이중서명 절차 (normative, 순서 불변)

**1단계 — Sender 서명.** 내부 `0x02` EIP-1559 sighash와 **동일한 preimage**에 서명한다.

```
senderSigHash = keccak( 0x02 ‖ rlp([chainId, nonce, gasTipCap, gasFeeCap, gas, to, value, data, accessList]) )
(SenderTx.V, SenderTx.R, SenderTx.S) = sign(senderSigHash, senderKey)
```
근거: `transaction_signing.go:439-448` (`londonSigner.Hash`가 `0x16`에 대해 내부 SenderTx의 `sigHash(chainId)` 반환).

**2단계 — FeePayer 서명.** sender의 확정된 V/R/S를 **포함한** preimage에 서명한다.

```
feePayerSigHash = keccak( 0x16 ‖ rlp([
    [ chainId, nonce, gasTipCap, gasFeeCap, gas, to, value, data, accessList,
      SenderTx.V, SenderTx.R, SenderTx.S ],   // ← sender 서명 포함
    FeePayer
]) )
(FV, FR, FS) = sign(feePayerSigHash, feePayerKey)
```
근거: `tx_fee_delegation.go:158-178` (`sigHash`), `transaction_signing.go:385-389` (`feeDelegateSigner.Hash`).

> **불변 규칙(MUST)**: Sender가 먼저 서명하고, FeePayer가 그 결과 위에 서명한다. 순서를 바꾸면 안 된다.

### 3.4 서명 접근자 의미 (구현 주의)

| 접근자 | 대응 서명 |
|--------|----------|
| `rawSignatureValues()` | **Sender** 서명 (V/R/S) |
| `rawFeePayerSignatureValues()` | **FeePayer** 서명 (FV/FR/FS) |
| `setSignatureValues()` | **FeePayer** 서명에 적용 (sender 아님) |

- 가스 가격은 Sender의 `GasFeeCap`/`GasTipCap`을 사용한다(FeePayer는 가격 필드 없음).
- 검증 시 Sender·FeePayer **두 서명 모두** 복구·확인해야 한다. FeePayer 복구는 `RecoverFeePayer(chainID, tx)`가 수행하며 복구된 주소가 `tx.FeePayer()`와 일치하지 않으면 `ErrInvalidFeePayer`.

### 3.5 게이팅

- `0x16`은 **Applepie 포크 이후**에만 유효하다(`core/txpool/validation.go`, `core/state_transition.go`). Mainnet은 block 0부터 활성이므로 실질 제약은 없다. → [`params.md`](./params.md).

### 3.6 노드측 대납 서명 (선택 경로)

완전 클라이언트측 SDK는 `0x16`을 스스로 구성·이중서명하고 `eth_sendRawTransaction`으로 전송한다(권장). 노드가 FeePayer로 서명하게 하려면 `eth_signRawFeeDelegateTransaction`을 사용한다([`rpc.md`](./rpc.md)).

---

## 4. 컨트랙트 배포 (CREATE / CREATE2)

배포는 스톡이다.

- **CREATE**: 주소 `= keccak(rlp([sender, nonce]))[12:]` (`crypto.CreateAddress`, `core/vm/evm.go:572`).
- **CREATE2**: 주소 `= keccak(0xff ‖ sender ‖ salt ‖ keccak(initcode))[12:]` (`core/vm/evm.go:578-582`).
- SDK는 배포 tx 빌더와 CREATE2 결정적 주소 계산기를 제공한다.
- **제약(유일)**: Anzeon 활성 시 **blacklisted 호출자의 배포는 거부**된다(`evm.go:480-482`). 배포에 authorization은 불필요 — 비-blacklist 계정이면 누구나 배포 가능.

---

## 5. 트랜잭션 빌더 안전 제약 (normative)

Anzeon 활성 시 노드가 거부하는 전송을 SDK 빌더가 **사전 차단**한다(취약점/실패 예방).

| 금지 조건 | 노드 에러 | 근거 |
|-----------|----------|------|
| zero address(`0x0`)로 value 전송 | `ErrZeroAddressTransfer` | `core/vm/evm.go` (~213) |
| precompile / native manager로 value 전송 | `ErrValueTransferToPrecompile` | `core/vm/evm.go` (~217) |
| blacklisted from / to / feePayer | `ErrBlacklistedAccount` | `core/state_transition.go:505-516, 579` |

- blacklist는 [`account.md`](./account.md) §5의 `eth_getProof.extra` 로 사전 확인 SHOULD.

---

## 6. 가스

- 가스 **양** 예측은 스톡 `eth_estimateGas`를 사용한다. `0x16`이나 authorized-tip은 예측되는 가스 양에 영향을 주지 않는다.
- 가스 **가격/팁**은 SDK가 임의로 정하지 말고 노드 오라클(`eth_maxPriorityFeePerGas` / `eth_gasPrice`)을 사용한다(MUST). Anzeon은 비인증 계정의 팁을 거버넌스 값으로 강제한다. → [`params.md`](./params.md), `eth/gasprice/anzeon.go`.

---

## 7. 검증 대상 (conformance vectors)

- 각 tx type(`0x00`~`0x04`, `0x16`)의 sighash 및 서명 결과(고정 키·입력).
- `0x16` 이중서명 전체: 입력 → `SenderTx.V/R/S`, `FV/FR/FS`, 최종 raw 봉투.
- CREATE / CREATE2 주소(고정 sender/nonce/salt/initcode).
- 안전 제약: 금지 전송 3종이 빌더 단계에서 거부되는지.
