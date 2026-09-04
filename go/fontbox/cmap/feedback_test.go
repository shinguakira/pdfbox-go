package cmap

import "testing"

// Tests for the slice 4 pull request feedback. Each fails without its fix.

// TestToIntNarrowsToJavaInt pins the width CMap.toInt accumulates at.
//
// Java's accumulator is an int, so a four-byte code whose first byte is 0x80 or
// more comes out negative. A Go int is 64 bits and would keep it positive,
// which changes the branch CMap.toUnicode(int) picks — it tests the code
// against 256, 0xFFFF and 0xFFFFFF to decide how many bytes the code had — and
// so changes which of the four maps is searched.
func TestToIntNarrowsToJavaInt(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want int
	}{
		{"one byte", []byte{0x7F}, 0x7F},
		{"one byte high", []byte{0xFF}, 0xFF},
		{"two bytes", []byte{0xFF, 0xFF}, 0xFFFF},
		{"three bytes", []byte{0xFF, 0xFF, 0xFF}, 0xFFFFFF},
		{"four bytes, top bit clear", []byte{0x7F, 0xFF, 0xFF, 0xFF}, 0x7FFFFFFF},
		{"four bytes, top bit set", []byte{0x80, 0x00, 0x00, 0x00}, -2147483648},
		{"four bytes, all set", []byte{0xFF, 0xFF, 0xFF, 0xFF}, -1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ToInt(c.data); got != c.want {
				t.Errorf("ToInt(% x) = %d, want %d", c.data, got, c.want)
			}
		})
	}
}

// TestToUnicodeOfHighFourByteCode is the difference the width above decides.
//
// A four-byte code with the top bit set is stored under a negative key, and
// Java's toUnicode(int) then tests that negative code against 256 and takes the
// one-byte map, then against 0xFFFF and returns from the two-byte map -- so the
// mapping it just stored is never found. A 64-bit accumulator would find it.
func TestToUnicodeOfHighFourByteCode(t *testing.T) {
	c := newCMap()
	code := []byte{0x80, 0x00, 0x00, 0x41}
	c.addCharMapping(code, "A")

	if got, ok := c.ToUnicode(ToInt(code)); ok {
		t.Errorf("ToUnicode(%d) = %q, want no mapping: Java looks in the two-byte map",
			ToInt(code), got)
	}
}
