// Package account models go-stablenet account state, in particular the
// StableNet-specific StateAccount.Extra bitfield.
//
// Spec: docs/spec/protocol/v0/account.md.
//
// Extra is a 64-bit flag word. Currently two bits are defined; the rest are
// reserved. Per the spec's forward-compatibility rule, decoding is LENIENT
// (unknown bits are ignored, but the raw word is preserved) while any encoding
// the SDK performs is STRICT (only defined bits).
package account

// Extra is the 64-bit account flag word (StateAccount.Extra).
type Extra uint64

const (
	// MaskBlacklisted is bit 63 (MSB): the account is blacklisted.
	MaskBlacklisted Extra = 1 << 63
	// MaskAuthorized is bit 62: the account has the Anzeon gas-tip privilege.
	MaskAuthorized Extra = 1 << 62

	// ValidMask is the union of all currently defined bits.
	ValidMask = MaskBlacklisted | MaskAuthorized
)

// Flags is the decoded, human-readable view of an Extra word. Raw preserves the
// original value so callers can access bits not yet interpreted by this spec
// version (forward compatibility).
type Flags struct {
	Blacklisted bool
	Authorized  bool
	Raw         uint64
}

// IsBlacklisted reports whether bit 63 is set.
func (e Extra) IsBlacklisted() bool { return e&MaskBlacklisted != 0 }

// IsAuthorized reports whether bit 62 is set.
func (e Extra) IsAuthorized() bool { return e&MaskAuthorized != 0 }

// SetBlacklisted returns a copy with bit 63 set.
func (e Extra) SetBlacklisted() Extra { return e | MaskBlacklisted }

// ClearBlacklisted returns a copy with bit 63 cleared.
func (e Extra) ClearBlacklisted() Extra { return e &^ MaskBlacklisted }

// SetAuthorized returns a copy with bit 62 set.
func (e Extra) SetAuthorized() Extra { return e | MaskAuthorized }

// ClearAuthorized returns a copy with bit 62 cleared.
func (e Extra) ClearAuthorized() Extra { return e &^ MaskAuthorized }

// Decode performs LENIENT decoding: it interprets only known bits and preserves
// the raw value. Unknown/reserved bits never cause an error (spec §6).
func Decode(raw uint64) Flags {
	e := Extra(raw)
	return Flags{
		Blacklisted: e.IsBlacklisted(),
		Authorized:  e.IsAuthorized(),
		Raw:         raw,
	}
}

// Validate performs STRICT validation: it returns false if any bit outside
// ValidMask is set. Use this only when the SDK itself produces an Extra word
// (e.g. state-override simulation); never for lenient reads from a node.
func (e Extra) Validate() bool {
	return e&^ValidMask == 0
}
