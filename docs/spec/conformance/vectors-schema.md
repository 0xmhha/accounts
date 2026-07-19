# Conformance Vectors — 스키마 & 커버리지 (v0)

> 골든 벡터의 형식·커버리지·생성 규칙을 규정한다. 벡터는 **노드=오라클**이며 모든 언어 SDK가 동일 벡터를 통과해야 한다(drift 차단). 실제 벡터 값은 P2에서 노드로 생성한다(본 문서는 그 전의 스키마).

- 관련: [`../protocol/v0/`](../protocol/v0/) 각 문서의 "검증 대상", 구현 계획 P2/P6

---

## 1. 파일 배치

```
/conformance/
  vectors/
    address.json          # privkey → address
    sighash.json          # tx type별 sighash
    sign.json             # tx type별 서명(고정 키)
    feedelegate.json      # 0x16 이중서명 전체
    create2.json          # CREATE/CREATE2 주소
    extra.json            # Extra 인코딩/디코딩(관용/부재 포함)
  runner-go/              # Go SDK 벡터 러너
  runner-ts/              # TS SDK 벡터 러너
```

## 2. 공통 벡터 포맷

각 벡터 파일은 배열. 각 항목:

```json
{
  "id": "feedelegate/basic-1",
  "specVersion": "protocol/v0",
  "description": "0x16 dual sign, mainnet",
  "input":  { "...": "케이스별" },
  "expected": { "...": "케이스별" }
}
```

- `specVersion`: 이 벡터가 준수하는 스펙 버전(러너가 SDK 지원 버전과 대조).
- 모든 바이트열은 `0x` hex 소문자. 정수는 문자열 십진 또는 hex(케이스별 명시).

## 3. 케이스별 스키마

### 3.1 address.json
```json
{ "input": { "privKey": "0x…32B" },
  "expected": { "address": "0x…20B" } }
```

### 3.2 sighash.json
```json
{ "input": { "txType": "0x02", "chainId": 8282, "tx": { "nonce": "0x1", "…": "…" } },
  "expected": { "sigHash": "0x…32B" } }
```
- 커버리지(MUST): `0x00, 0x01, 0x02, 0x03, 0x04, 0x16(sender), 0x16(feePayer)`.

### 3.3 sign.json
```json
{ "input": { "txType": "0x02", "chainId": 8282, "privKey": "0x…", "tx": { "…": "…" } },
  "expected": { "v": "0x…", "r": "0x…", "s": "0x…", "rawTx": "0x…" } }
```

### 3.4 feedelegate.json (핵심)
```json
{ "input": {
    "chainId": 8282,
    "senderKey": "0x…", "feePayerKey": "0x…",
    "tx": { "nonce": "0x1", "maxPriorityFeePerGas": "0x…", "maxFeePerGas": "0x…",
            "gas": "0x…", "to": "0x…", "value": "0x…", "data": "0x", "accessList": [] },
    "feePayer": "0x…20B" },
  "expected": {
    "senderSigHash": "0x…32B",
    "senderSig": { "v": "0x…", "r": "0x…", "s": "0x…" },
    "feePayerSigHash": "0x…32B",
    "feePayerSig": { "fv": "0x…", "fr": "0x…", "fs": "0x…" },
    "rawTx": "0x16…" } }
```
- 검증: sender-먼저 순서, feePayerSigHash가 senderSig 포함, 최종 봉투 바이트 일치.

### 3.5 create2.json
```json
{ "input": { "kind": "CREATE2", "sender": "0x…", "salt": "0x…32B", "initCode": "0x…" },
  "expected": { "address": "0x…20B" } }
// kind: "CREATE"면 { sender, nonce } 입력
```

### 3.6 extra.json
```json
[
  { "input": { "extra": "0x8000000000000000" }, "expected": { "blacklisted": true,  "authorized": false, "raw": "0x8000000000000000" } },
  { "input": { "extra": "0x4000000000000000" }, "expected": { "blacklisted": false, "authorized": true,  "raw": "0x4000000000000000" } },
  { "input": { "extra": "0xC000000000000000" }, "expected": { "blacklisted": true,  "authorized": true,  "raw": "0xC000000000000000" } },
  { "input": { "extra": "0x2000000000000000" }, "expected": { "blacklisted": false, "authorized": false, "raw": "0x2000000000000000", "note": "reserved bit61: 관용 디코딩, raw 보존" } },
  { "input": { "absent": true }, "expected": { "blacklisted": false, "authorized": false, "raw": "0x0", "note": "getProof extra 부재 → 0" } }
]
```

## 4. 생성 규칙 (P2, normative)

- 벡터는 go-stablenet 노드/패키지로 생성한다(오라클). 고정 키·고정 입력을 사용해 결정적.
- 생성 스크립트는 저장소에 포함하되, 벡터 JSON은 리뷰 대상 산출물로 커밋한다.
- 노드 테스트(`transaction_signing_test.go` 등)와 교차 확인.

## 5. 러너 계약 (P6, normative)

- 각 언어 러너는 벡터 JSON을 읽어 SDK로 재현하고 `expected`와 대조. 불일치=fail.
- `specVersion`이 러너 지원 범위를 벗어나면 skip이 아니라 명시적 fail(capability, README §버전관리).
- CI: PR마다 Go·TS 러너 실행. 의도적 변조 테스트로 회귀 검출 능력 증명.

## 6. 최소 커버리지 (DoD)

- tx type 6종 sighash + 서명.
- 0x16 이중서명 최소 2케이스(기본, data 포함).
- CREATE + CREATE2.
- Extra 5케이스(정상 3 + 예약비트 관용 + 부재).
- 주소 파생.
