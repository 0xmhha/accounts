package governance

import "testing"

func TestDefaultAddresses(t *testing.T) {
	if DefaultGovValidator.Hex() != "0x0000000000000000000000000000000000001001" {
		t.Fatalf("validator addr %s", DefaultGovValidator.Hex())
	}
	if DefaultGovMasterMinter.Hex() != "0x0000000000000000000000000000000000001002" {
		t.Fatalf("master-minter addr %s", DefaultGovMasterMinter.Hex())
	}
	if DefaultGovCouncil.Hex() != "0x0000000000000000000000000000000000001004" {
		t.Fatalf("council addr %s", DefaultGovCouncil.Hex())
	}
}
