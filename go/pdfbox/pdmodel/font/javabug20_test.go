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
// They are joined unsigned: the field is `supplementVersion` at offset 140 of
// the AAT "gcid" table and is a uint16, which the offsets the caller reads
// with bear out -- version, format and size take 8 bytes, then registry 2,
// registryName 64, order 2, orderName 64, landing the next field at 140.
// Java's `bytes[140] << 8` sign-extends a byte, so repairing only the `&`
// would answer -255 for FF 01.
func TestCIDSupplementIsTheTwoBytesJoined(t *testing.T) {
	for _, c := range []struct {
		high, low byte
		want      int
	}{
		{1, 2, 258},         // 0x0102
		{0, 5, 5},           // a supplement that fits in one byte
		{0, 0, 0},           // none
		{0x80, 0, 32768},    // the first high byte Java would sign-extend
		{0xFF, 0x01, 65281}, // and the largest
		{0xFF, 0xFF, 65535},
	} {
		if got := cidSupplementVersion(c.high, c.low); got != c.want {
			t.Errorf("a supplement of %#02x,%#02x is %d, want %d",
				c.high, c.low, got, c.want)
		}
	}
}
