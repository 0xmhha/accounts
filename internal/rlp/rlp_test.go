package rlp

import (
	"bytes"
	"encoding/hex"
	"math/big"
	"testing"
)

func hexb(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func TestEncodeBytes(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
		want []byte
	}{
		{"empty string", nil, []byte{0x80}},
		{"single zero byte", []byte{0x00}, []byte{0x00}},
		{"single byte 0x0f", []byte{0x0f}, []byte{0x0f}},
		{"single byte 0x7f", []byte{0x7f}, []byte{0x7f}},
		{"single byte 0x80", []byte{0x80}, []byte{0x81, 0x80}},
		{"dog", []byte("dog"), []byte{0x83, 'd', 'o', 'g'}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EncodeBytes(tt.in); !bytes.Equal(got, tt.want) {
				t.Fatalf("EncodeBytes(%x) = %x, want %x", tt.in, got, tt.want)
			}
		})
	}
}

func TestEncodeBytesLong(t *testing.T) {
	// 56-byte string: 0xb8, 0x38, then 56 bytes.
	in := bytes.Repeat([]byte{'a'}, 56)
	got := EncodeBytes(in)
	if got[0] != 0xb8 || got[1] != 0x38 {
		t.Fatalf("long string header = %x %x, want b8 38", got[0], got[1])
	}
	if len(got) != 58 {
		t.Fatalf("len = %d, want 58", len(got))
	}
}

func TestEncodeUint(t *testing.T) {
	tests := []struct {
		in   uint64
		want []byte
	}{
		{0, []byte{0x80}},
		{15, []byte{0x0f}},
		{127, []byte{0x7f}},
		{128, []byte{0x81, 0x80}},
		{1024, []byte{0x82, 0x04, 0x00}},
	}
	for _, tt := range tests {
		if got := EncodeUint(tt.in); !bytes.Equal(got, tt.want) {
			t.Fatalf("EncodeUint(%d) = %x, want %x", tt.in, got, tt.want)
		}
	}
}

func TestEncodeBig(t *testing.T) {
	if got := EncodeBig(nil); !bytes.Equal(got, []byte{0x80}) {
		t.Fatalf("EncodeBig(nil) = %x, want 80", got)
	}
	if got := EncodeBig(big.NewInt(0)); !bytes.Equal(got, []byte{0x80}) {
		t.Fatalf("EncodeBig(0) = %x, want 80", got)
	}
	if got := EncodeBig(big.NewInt(1024)); !bytes.Equal(got, []byte{0x82, 0x04, 0x00}) {
		t.Fatalf("EncodeBig(1024) = %x, want 820400", got)
	}
}

func TestEncodeList(t *testing.T) {
	// empty list
	if got := EncodeList(); !bytes.Equal(got, []byte{0xc0}) {
		t.Fatalf("EncodeList() = %x, want c0", got)
	}
	// ["cat","dog"] => c8 83 636174 83 646f67
	got := EncodeList(EncodeBytes([]byte("cat")), EncodeBytes([]byte("dog")))
	want := hexb("c88363617483646f67")
	if !bytes.Equal(got, want) {
		t.Fatalf("EncodeList(cat,dog) = %x, want %x", got, want)
	}
}

func TestEncodeListLong(t *testing.T) {
	// List whose payload exceeds 55 bytes uses 0xf7+llen header.
	item := EncodeBytes(bytes.Repeat([]byte{'a'}, 56)) // 58 bytes
	got := EncodeList(item)
	if got[0] != 0xf8 || got[1] != 0x3a { // 58 = 0x3a
		t.Fatalf("long list header = %x %x, want f8 3a", got[0], got[1])
	}
}
