package filter

// JAVA-BUGS 30: `ASCIIHexFilter` logs a byte that is not a hexadecimal digit
// and then adds its table entry, -1, to the value anyway, so the bad digit
// moves the byte around it instead of the byte itself.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestASCIIHexTreatsABadDigitAsZero is the defect.
//
// The expected values are the ones the same method already gives a digit it
// does not have: "second value behaves like 0 in case of EOD", so "486>"
// decodes to 0x48, 0x60. A digit that is present but not hexadecimal is the
// same kind of missing digit, and reading it as zero is the only reading under
// which one bad digit changes one nibble. Java answers 63 for "4Z" -- 4*16 +
// (-1) -- and 0xF4 for "Z4", where -1*16 makes the value negative and
// `decoded.write` narrows it.
func TestASCIIHexTreatsABadDigitAsZero(t *testing.T) {
	cases := []struct {
		encoded string
		want    []byte
	}{
		// the bad digit is the low nibble: 0x40, not 63
		{"4Z65", []byte{0x40, 0x65}},
		// the bad digit is the high nibble: 0x04, not 0xF4
		{"Z465", []byte{0x04, 0x65}},
		// both nibbles bad is a zero byte, not 0xEF
		{"ZZ65", []byte{0x00, 0x65}},
	}
	for _, c := range cases {
		var out bytes.Buffer
		if _, err := (ASCIIHex{}).Decode(&out, strings.NewReader(c.encoded),
			cos.NewDictionary(), 0); err != nil {
			t.Errorf("decoding %q: %v", c.encoded, err)
			continue
		}
		if !bytes.Equal(out.Bytes(), c.want) {
			t.Errorf("decoding %q gave %v, want %v", c.encoded, out.Bytes(), c.want)
		}
	}
}
