package cos

import (
	"fmt"
	"testing"
)

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/PDFDocEncodingTest.java.

// pdfDocEncodingDeviations lists every character where PDFDocEncoding departs
// from ISO-8859-1, from the table in ISO 32000-1:2008. Same order as the Java.
var pdfDocEncodingDeviations = []rune{
	// block 1
	'˘', // BREVE
	'ˇ', // CARON
	'ˆ', // MODIFIER LETTER CIRCUMFLEX ACCENT
	'˙', // DOT ABOVE
	'˝', // DOUBLE ACUTE ACCENT
	'˛', // OGONEK
	'˚', // RING ABOVE
	'˜', // SMALL TILDE
	// block 2
	'•', // BULLET
	'†', // DAGGER
	'‡', // DOUBLE DAGGER
	'…', // HORIZONTAL ELLIPSIS
	'—', // EM DASH
	'–', // EN DASH
	'ƒ', // LATIN SMALL LETTER SCRIPT F
	'⁄', // FRACTION SLASH (solidus)
	'‹', // SINGLE LEFT-POINTING ANGLE QUOTATION MARK
	'›', // SINGLE RIGHT-POINTING ANGLE QUOTATION MARK
	'−', // MINUS SIGN
	'‰', // PER MILLE SIGN
	'„', // DOUBLE LOW-9 QUOTATION MARK (quotedblbase)
	'“', // LEFT DOUBLE QUOTATION MARK (quotedblleft)
	'”', // RIGHT DOUBLE QUOTATION MARK (quotedblright)
	'‘', // LEFT SINGLE QUOTATION MARK (quoteleft)
	'’', // RIGHT SINGLE QUOTATION MARK (quoteright)
	'‚', // SINGLE LOW-9 QUOTATION MARK (quotesinglbase)
	'™', // TRADE MARK SIGN
	'ﬁ', // LATIN SMALL LIGATURE FI
	'ﬂ', // LATIN SMALL LIGATURE FL
	'Ł', // LATIN CAPITAL LETTER L WITH STROKE
	'Œ', // LATIN CAPITAL LIGATURE OE
	'Š', // LATIN CAPITAL LETTER S WITH CARON
	'Ÿ', // LATIN CAPITAL LETTER Y WITH DIAERESIS
	'Ž', // LATIN CAPITAL LETTER Z WITH CARON
	'ı', // LATIN SMALL LETTER DOTLESS I
	'ł', // LATIN SMALL LETTER L WITH STROKE
	'œ', // LATIN SMALL LIGATURE OE
	'š', // LATIN SMALL LETTER S WITH CARON
	'ž', // LATIN SMALL LETTER Z WITH CARON
	'€', // EURO SIGN
}

func TestPDFDocEncodingDeviations(t *testing.T) {
	for _, r := range pdfDocEncodingDeviations {
		deviation := string(r)
		s := NewStringObj(deviation)
		if got := s.Value(); got != deviation {
			t.Errorf("round trip of %U = %q, want %q", r, got, deviation)
		}
	}
}

// TestPDFDocEncodingPDFBox3864 covers PDFBOX-3864: every byte value, written as
// a UTF-16BE string with a BOM, must round-trip through Value and back.
func TestPDFDocEncodingPDFBox3864(t *testing.T) {
	for i := 0; i < 256; i++ {
		hex := fmt.Sprintf("FEFF%04X", i)
		cs1, err := ParseHexString(hex)
		if err != nil {
			t.Fatalf("ParseHexString(%q): %v", hex, err)
		}
		cs2 := NewStringObj(cs1.Value())
		if !cs1.Equals(cs2) {
			t.Errorf("%q did not round trip: %q vs %q", hex, cs1.Value(), cs2.Value())
		}
	}
}

// TestPDFDocEncodingContainsChar spot-checks the membership test that
// NewStringObj uses to decide between PDFDocEncoding and UTF-16BE.
func TestPDFDocEncodingContainsChar(t *testing.T) {
	for _, r := range []rune{'A', 'z', '0', ' ', '€', '•'} {
		if !pdfDocEncodingContainsRune(r) {
			t.Errorf("%U reported as outside PDFDocEncoding", r)
		}
	}
	// characters with no PDFDocEncoding code
	for _, r := range []rune{'中', 'Ā', '☃'} {
		if pdfDocEncodingContainsRune(r) {
			t.Errorf("%U reported as inside PDFDocEncoding", r)
		}
	}
	// The gaps the table deliberately leaves unmapped. 0x18-0x1F and 0x7F-0xA0
	// are reassigned to other characters, so the original code points are not
	// themselves encodable; 0xAD (soft hyphen) is dropped outright.
	for _, r := range []rune{0x18, 0x1F, 0x7F, 0x80, 0xA0, 0xAD} {
		if pdfDocEncodingContainsRune(r) {
			t.Errorf("%U is an unmapped code point but reported as inside", r)
		}
	}
}
