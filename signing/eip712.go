package signing

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"

	"github.com/0xmhha/accounts/crypto"
	"github.com/0xmhha/accounts/types"
)

// EIP-712 typed structured data hashing (standard; matches
// go-stablenet signer/core/signed_data.go).

// TypedDataField is one field of a struct type: {Name, Type}.
type TypedDataField struct {
	Name string
	Type string
}

// TypedDataTypes maps a struct type name to its ordered fields. It MUST include
// the "EIP712Domain" type describing the domain fields in use.
type TypedDataTypes map[string][]TypedDataField

// TypedDataDomain holds the EIP-712 domain values. Only the fields listed in
// Types["EIP712Domain"] are used.
type TypedDataDomain struct {
	Name              string
	Version           string
	ChainID           *big.Int
	VerifyingContract *types.Address
	Salt              *[32]byte
}

func (d TypedDataDomain) toMap() map[string]interface{} {
	m := map[string]interface{}{}
	if d.Name != "" {
		m["name"] = d.Name
	}
	if d.Version != "" {
		m["version"] = d.Version
	}
	if d.ChainID != nil {
		m["chainId"] = d.ChainID
	}
	if d.VerifyingContract != nil {
		m["verifyingContract"] = *d.VerifyingContract
	}
	if d.Salt != nil {
		m["salt"] = *d.Salt
	}
	return m
}

// TypedData is an EIP-712 typed data payload.
type TypedData struct {
	Types       TypedDataTypes
	PrimaryType string
	Domain      TypedDataDomain
	Message     map[string]interface{}
}

// Digest returns the EIP-712 signing digest:
//
//	keccak256(0x19 || 0x01 || domainSeparator || hashStruct(primaryType, message))
func (td *TypedData) Digest() ([]byte, error) {
	domainSep, err := td.hashStruct("EIP712Domain", td.Domain.toMap())
	if err != nil {
		return nil, fmt.Errorf("domain: %w", err)
	}
	msgHash, err := td.hashStruct(td.PrimaryType, td.Message)
	if err != nil {
		return nil, fmt.Errorf("message: %w", err)
	}
	return crypto.Keccak256([]byte{0x19, 0x01}, domainSep, msgHash), nil
}

// hashStruct = keccak256(typeHash(type) || encodeData(fields)).
func (td *TypedData) hashStruct(typeName string, data map[string]interface{}) ([]byte, error) {
	enc, err := td.encodeData(typeName, data)
	if err != nil {
		return nil, err
	}
	return crypto.Keccak256(enc), nil
}

func (td *TypedData) encodeData(typeName string, data map[string]interface{}) ([]byte, error) {
	fields, ok := td.Types[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown type %q", typeName)
	}
	out := td.typeHash(typeName)
	for _, f := range fields {
		enc, err := td.encodeField(f.Type, data[f.Name])
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", f.Name, err)
		}
		out = append(out, enc...)
	}
	return out, nil
}

// typeHash = keccak256(encodeType(type)).
func (td *TypedData) typeHash(typeName string) []byte {
	return crypto.Keccak256([]byte(td.encodeType(typeName)))
}

// encodeType builds "Primary(...)Ref1(...)Ref2(...)" with referenced structs
// sorted alphabetically after the primary type.
func (td *TypedData) encodeType(primary string) string {
	deps := map[string]bool{}
	td.collectDeps(primary, deps)
	delete(deps, primary)
	sorted := make([]string, 0, len(deps))
	for d := range deps {
		sorted = append(sorted, d)
	}
	sort.Strings(sorted)

	var b strings.Builder
	writeOne := func(name string) {
		b.WriteString(name)
		b.WriteByte('(')
		for i, f := range td.Types[name] {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(f.Type)
			b.WriteByte(' ')
			b.WriteString(f.Name)
		}
		b.WriteByte(')')
	}
	writeOne(primary)
	for _, d := range sorted {
		writeOne(d)
	}
	return b.String()
}

func (td *TypedData) collectDeps(typeName string, found map[string]bool) {
	if found[typeName] {
		return
	}
	if _, ok := td.Types[typeName]; !ok {
		return
	}
	found[typeName] = true
	for _, f := range td.Types[typeName] {
		base := strings.TrimSuffix(f.Type, "[]")
		if _, ok := td.Types[base]; ok {
			td.collectDeps(base, found)
		}
	}
}

// encodeField encodes a single field value to its 32-byte (or hashed) form.
func (td *TypedData) encodeField(fieldType string, value interface{}) ([]byte, error) {
	// Arrays: keccak256 of the concatenated encoded elements.
	if strings.HasSuffix(fieldType, "[]") {
		elemType := strings.TrimSuffix(fieldType, "[]")
		arr, ok := value.([]interface{})
		if !ok {
			return nil, fmt.Errorf("expected array for %s", fieldType)
		}
		var buf []byte
		for _, e := range arr {
			enc, err := td.encodeField(elemType, e)
			if err != nil {
				return nil, err
			}
			buf = append(buf, enc...)
		}
		return crypto.Keccak256(buf), nil
	}
	// Struct reference.
	if _, ok := td.Types[fieldType]; ok {
		m, ok := value.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("expected object for struct %s", fieldType)
		}
		return td.hashStruct(fieldType, m)
	}

	switch {
	case fieldType == "string":
		s, _ := value.(string)
		return crypto.Keccak256([]byte(s)), nil
	case fieldType == "bytes":
		b, err := toBytes(value)
		if err != nil {
			return nil, err
		}
		return crypto.Keccak256(b), nil
	case fieldType == "address":
		a, err := toAddress(value)
		if err != nil {
			return nil, err
		}
		return leftPad32(a.Bytes()), nil
	case fieldType == "bool":
		out := make([]byte, 32)
		if b, _ := value.(bool); b {
			out[31] = 1
		}
		return out, nil
	case strings.HasPrefix(fieldType, "bytes"): // bytesN (1..32)
		b, err := toBytes(value)
		if err != nil {
			return nil, err
		}
		return rightPad32(b), nil
	case strings.HasPrefix(fieldType, "uint") || strings.HasPrefix(fieldType, "int"):
		n, err := toBig(value)
		if err != nil {
			return nil, err
		}
		return bigTo32(n), nil
	default:
		return nil, fmt.Errorf("unsupported field type %q", fieldType)
	}
}

// --- value coercion ---------------------------------------------------------

func toBig(v interface{}) (*big.Int, error) {
	switch x := v.(type) {
	case *big.Int:
		return x, nil
	case big.Int:
		return &x, nil
	case int:
		return big.NewInt(int64(x)), nil
	case int64:
		return big.NewInt(x), nil
	case uint64:
		return new(big.Int).SetUint64(x), nil
	case string:
		n, ok := new(big.Int).SetString(x, 10)
		if !ok {
			return nil, fmt.Errorf("invalid integer string %q", x)
		}
		return n, nil
	default:
		return nil, fmt.Errorf("cannot convert %T to integer", v)
	}
}

func toAddress(v interface{}) (types.Address, error) {
	switch x := v.(type) {
	case types.Address:
		return x, nil
	case string:
		return types.HexToAddress(x)
	default:
		return types.Address{}, fmt.Errorf("cannot convert %T to address", v)
	}
}

func toBytes(v interface{}) ([]byte, error) {
	switch x := v.(type) {
	case []byte:
		return x, nil
	case [32]byte:
		return x[:], nil
	case string:
		return hex.DecodeString(strings.TrimPrefix(x, "0x"))
	default:
		return nil, fmt.Errorf("cannot convert %T to bytes", v)
	}
}

func leftPad32(b []byte) []byte {
	out := make([]byte, 32)
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	copy(out[32-len(b):], b)
	return out
}

func rightPad32(b []byte) []byte {
	out := make([]byte, 32)
	if len(b) > 32 {
		b = b[:32]
	}
	copy(out, b)
	return out
}

// bigTo32 encodes a big.Int into a 32-byte big-endian word (two's complement
// for negative values).
func bigTo32(n *big.Int) []byte {
	out := make([]byte, 32)
	if n.Sign() >= 0 {
		n.FillBytes(out)
		return out
	}
	// two's complement: 2^256 + n
	mod := new(big.Int).Lsh(big.NewInt(1), 256)
	twos := new(big.Int).Add(mod, n)
	twos.FillBytes(out)
	return out
}
