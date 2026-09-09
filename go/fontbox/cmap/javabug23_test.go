package cmap

// JAVA-BUGS 23: `CMapStrings.getMapping` reads a zero-length code as the
// two-byte code 0.

import "testing"

// TestGetMappingOfAnEmptyCode is the defect.
//
// The ternary has two arms for three cases: one byte, two bytes, and no bytes
// at all. A zero-length code falls into the two-byte arm and comes back as
// U+0000, where the caller's other arm — the one for a code longer than two
// bytes — answers null.
//
// The expected value is what a mapping of no bytes is: nothing. Not the
// character whose code is zero.
func TestGetMappingOfAnEmptyCode(t *testing.T) {
	got, found := GetMapping(nil)
	if found && got != "" {
		t.Errorf("GetMapping(nil) is %q, want nothing: no bytes is not the code 0", got)
	}

	// The one and two byte codes still answer what they held.
	if got, found := GetMapping([]byte{0x41}); !found || got != "A" {
		t.Errorf("GetMapping([0x41]) is %q, %v, want \"A\", true", got, found)
	}
	if got, found := GetMapping([]byte{0x00, 0x41}); !found || got != "A" {
		t.Errorf("GetMapping([0x00 0x41]) is %q, %v, want \"A\", true", got, found)
	}
}
