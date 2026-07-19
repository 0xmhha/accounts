package transport

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/0xmhha/accounts/types"
)

// CallMsg describes a message for eth_call / eth_estimateGas.
type CallMsg struct {
	From  *types.Address
	To    *types.Address
	Gas   uint64
	Value *big.Int
	Data  []byte
}

func (m CallMsg) toMap() map[string]interface{} {
	out := map[string]interface{}{}
	if m.From != nil {
		out["from"] = m.From.Hex()
	}
	if m.To != nil {
		out["to"] = m.To.Hex()
	}
	if m.Gas != 0 {
		out["gas"] = encodeQuantity(new(big.Int).SetUint64(m.Gas))
	}
	if m.Value != nil {
		out["value"] = encodeQuantity(m.Value)
	}
	if len(m.Data) > 0 {
		out["data"] = "0x" + hex.EncodeToString(m.Data)
	}
	return out
}

// parseQuantity parses a hex "0x"-prefixed quantity into a big.Int.
func parseQuantity(s string) (*big.Int, error) {
	s = strings.TrimPrefix(s, "0x")
	if s == "" {
		return big.NewInt(0), nil
	}
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		return nil, fmt.Errorf("invalid hex quantity %q", s)
	}
	return n, nil
}

// encodeQuantity encodes a big.Int as a minimal "0x" hex quantity.
func encodeQuantity(x *big.Int) string {
	if x == nil || x.Sign() == 0 {
		return "0x0"
	}
	return "0x" + x.Text(16)
}
