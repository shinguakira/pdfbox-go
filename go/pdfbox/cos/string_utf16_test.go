package cos

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf16"
)

// TestStringObjValueMalformedUTF16 checks what a text string with a byte order
// mark decodes to where its UTF-16 does not decode cleanly.
//
// Java decodes with new String(bytes, UTF_16BE or UTF_16LE), whose decoder
// replaces what it cannot read with U+FFFD: a high surrogate and the unit after
// it are one malformed sequence of four bytes when that unit is not a low
// surrogate, so the unit is lost with it; a low surrogate on its own is two
// malformed bytes; and whatever is left at the end is one. Found by the corpus
// comparison of document information: pdfjs/poppler-742-0-fuzzed.pdf has a
// /Title of odd length, which PDFBox reads with a U+FFFD at the end and the Go
// version read without.
//
// The expected values are PDFBox's, printed by COSString.getString as UTF-16
// code units.
func TestStringObjValueMalformedUTF16(t *testing.T) {
	for _, c := range []struct {
		what  string
		bytes []byte
		want  string
	}{
		{"an odd last byte", []byte{0xFE, 0xFF, 0x00, 0x41, 0x00}, "0041 FFFD"},
		{"only the mark and a byte", []byte{0xFE, 0xFF, 0x41}, "FFFD"},
		{"a surrogate pair", []byte{0xFE, 0xFF, 0xD8, 0x00, 0xDC, 0x00}, "D800 DC00"},
		{"a high surrogate before A", []byte{0xFE, 0xFF, 0xD8, 0x00, 0x00, 0x41}, "FFFD"},
		{"a high surrogate before A and B", []byte{0xFE, 0xFF, 0xD8, 0x00, 0x00, 0x41, 0x00, 0x42}, "FFFD 0042"},
		{"a high surrogate at the end", []byte{0xFE, 0xFF, 0x00, 0x41, 0xD8, 0x00}, "0041 FFFD"},
		{"a high surrogate and a byte at the end", []byte{0xFE, 0xFF, 0x00, 0x41, 0xD8, 0x00, 0x42}, "0041 FFFD"},
		{"a low surrogate before A", []byte{0xFE, 0xFF, 0xDC, 0x00, 0x00, 0x41}, "FFFD 0041"},
		{"two high surrogates and a low", []byte{0xFE, 0xFF, 0xD8, 0x00, 0xD8, 0x00, 0xDC, 0x00}, "FFFD FFFD"},
		{"U+FFFE", []byte{0xFE, 0xFF, 0xFF, 0xFE, 0x00, 0x41}, "FFFE 0041"},
		{"a second mark", []byte{0xFE, 0xFF, 0xFE, 0xFF, 0x00, 0x41}, "FEFF 0041"},
		{"little endian, an odd last byte", []byte{0xFF, 0xFE, 0x41, 0x00, 0x00}, "0041 FFFD"},
		{"little endian, a high surrogate before A", []byte{0xFF, 0xFE, 0x00, 0xD8, 0x41, 0x00}, "FFFD"},
		{"little endian, a low surrogate before A", []byte{0xFF, 0xFE, 0x00, 0xDC, 0x41, 0x00}, "FFFD 0041"},
		{"little endian, a surrogate pair", []byte{0xFF, 0xFE, 0x00, 0xD8, 0x00, 0xDC}, "D800 DC00"},
		{"the tail of poppler-742-0-fuzzed.pdf's title",
			[]byte{0xFE, 0xFF, 0x00, 0x54, 0x00, 0x20, 0x00, 0x20, 0x31, 0x00, 0x00}, "0054 0020 0020 3100 FFFD"},
	} {
		var units []string
		for _, unit := range utf16.Encode([]rune(NewStringObjBytes(c.bytes).Value())) {
			units = append(units, fmt.Sprintf("%04X", unit))
		}
		if got := strings.Join(units, " "); got != c.want {
			t.Errorf("%s decodes to [%s], want PDFBox's [%s]", c.what, got, c.want)
		}
	}
}
