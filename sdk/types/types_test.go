package types

import "testing"

func TestCloidFromIntMatchesPython(t *testing.T) {
	c := CloidFromInt(1)
	if c.ToRaw() != "0x00000000000000000000000000000001" {
		t.Fatalf("unexpected cloid: %s", c.ToRaw())
	}
}

func TestNewCloidValidationMatchesPython(t *testing.T) {
	if _, err := NewCloid("0x00000000000000000000000000000001"); err != nil {
		t.Fatalf("expected valid cloid: %v", err)
	}
	if _, err := NewCloid("00000000000000000000000000000001"); err == nil {
		t.Fatal("expected missing 0x prefix to fail")
	}
	if _, err := NewCloid("0x01"); err == nil {
		t.Fatal("expected wrong length to fail")
	}
}
