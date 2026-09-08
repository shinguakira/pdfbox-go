package filter

// JAVA-BUGS 28: `LZWFilter.findPatternCode` returns the byte itself for a
// pattern of one, and a Java byte is signed, so a byte of 0x80 or more comes
// back negative where the comment above it says the index matches the value.
// In-package, because the function is private in the Java too.

import "testing"

// TestFindPatternCodeOfOneByteIsItsIndex is the defect.
//
// The expected value is the index the function's own comment names -- "for the
// first 256 entries, index matches value" -- and the table `createCodeTable`
// builds bears out: entry N is the one-byte pattern N, for every N from 0 to
// 255. The masked read is also what the encoder's other single-byte code
// writes, `by & 0xff`.
func TestFindPatternCodeOfOneByteIsItsIndex(t *testing.T) {
	table := createCodeTable()
	for _, value := range []int{0, 0x7F, 0x80, 0xFE, 0xFF} {
		pattern := []byte{byte(value)}
		if got := findPatternCode(table, pattern); got != value {
			t.Errorf("findPatternCode(%#02x) = %d, want %d", value, got, value)
		}
		// and the index it answers is the table entry for that pattern
		if got := findPatternCode(table, pattern); got >= 0 && got < len(table) {
			if entry := table[got]; len(entry) != 1 || entry[0] != byte(value) {
				t.Errorf("entry %d of the table is %v, want the one-byte pattern %#02x",
					got, entry, value)
			}
		}
	}
}
