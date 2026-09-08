package font

// JAVA-BUGS 20: the CID supplement version is built with `&` between the two
// halves of a 16-bit number, so it is always zero.

import "testing"

// TestCIDSupplementIsTheTwoBytesJoined is the arithmetic.
//
// `bytes[140] << 8 & (bytes[141] & 0xFF)` ANDs a value whose low eight bits
// are zero with one whose high bits are zero, so the answer is 0 for every
// input. The two bytes are the halves of one number and have to be ORed.
//
// For a high byte of 1 and a low byte of 2 the supplement is 0x0102 = 258.
func TestCIDSupplementIsTheTwoBytesJoined(t *testing.T) {
	if got := cidSupplementVersion(1, 2); got != 258 {
		t.Errorf("a supplement of 1,2 is %d, want 258", got)
	}
	if got := cidSupplementVersion(0, 5); got != 5 {
		t.Errorf("a supplement of 0,5 is %d, want 5", got)
	}
	if got := cidSupplementVersion(0, 0); got != 0 {
		t.Errorf("a supplement of 0,0 is %d, want 0", got)
	}
}
