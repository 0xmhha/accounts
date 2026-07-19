package account

import "testing"

// Vectors mirror docs/spec/conformance/vectors-schema.md §3.6.
func TestDecode(t *testing.T) {
	tests := []struct {
		name        string
		raw         uint64
		blacklisted bool
		authorized  bool
	}{
		{"blacklisted only", 0x8000000000000000, true, false},
		{"authorized only", 0x4000000000000000, false, true},
		{"both", 0xC000000000000000, true, true},
		{"reserved bit61 lenient", 0x2000000000000000, false, false},
		{"zero/absent", 0x0, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Decode(tt.raw)
			if f.Blacklisted != tt.blacklisted || f.Authorized != tt.authorized {
				t.Fatalf("Decode(%#x) = %+v, want bl=%v au=%v", tt.raw, f, tt.blacklisted, tt.authorized)
			}
			if f.Raw != tt.raw {
				t.Fatalf("raw not preserved: got %#x want %#x", f.Raw, tt.raw)
			}
		})
	}
}

func TestSetClearImmutable(t *testing.T) {
	var e Extra
	if e.SetBlacklisted() != MaskBlacklisted {
		t.Fatal("SetBlacklisted")
	}
	if e != 0 {
		t.Fatal("Set mutated receiver")
	}
	both := Extra(0).SetBlacklisted().SetAuthorized()
	if !both.IsBlacklisted() || !both.IsAuthorized() {
		t.Fatal("both flags")
	}
	if both.ClearBlacklisted().IsBlacklisted() {
		t.Fatal("ClearBlacklisted")
	}
}

func TestValidateStrict(t *testing.T) {
	if !ValidMask.Validate() {
		t.Fatal("ValidMask must validate")
	}
	if Extra(0x2000000000000000).Validate() {
		t.Fatal("reserved bit must fail strict validation")
	}
	if !Extra(0).Validate() {
		t.Fatal("zero must validate")
	}
}
