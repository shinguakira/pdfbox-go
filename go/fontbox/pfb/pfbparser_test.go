package pfb

import (
	"testing"
)

// Port of the part of org.apache.fontbox.pfb.PfbParserTest that reaches
// PfbParser directly.
//
// The rest of that class goes through Type1Font -- testPfb, testPfbPDFBox5713,
// testPfbPDFBox3654, testEmpty, testMiscBadFonts and testPfbParser -- and lands
// with the Type 1 parser; three of those also need a font the Java build
// downloads into target/fonts. See migration/STATUS.md.

// TestNegativeRecordSize checks that a PFB with a negative size field (integer
// overflow) reports an error instead of trying to allocate. A crafted 18-byte
// PFB with size bytes 01 00 00 FF overflows the signed int to -16777215,
// bypassing the upper-bound check.
func TestNegativeRecordSize(t *testing.T) {
	// 18-byte crafted PFB: start marker 0x80, ASCII type 0x01,
	// size field 0x01 0x00 0x00 0xFF = -16777215 as signed int
	crashInput := []byte{
		0x80, 0x01, // header
		0x01, 0x00, 0x00, 0xFF, // size: overflows to negative
		0xFF, 0xFF, 0xFF, // garbage data
		0xFF, 0xFF, 0xFF,
		0x27, 0x05, 0xF8, 0xFF,
		0xD2, 0x40,
	}
	_, err := NewPfbParserBytes(crashInput)
	if err == nil {
		t.Fatal("a negative record size was accepted")
	}
	if got, want := err.Error(), "record size -16777215 is negative"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

// TestHighRecordSize checks that a PFB with a high size field reports an error.
func TestHighRecordSize(t *testing.T) {
	// 18-byte crafted PFB: start marker 0x80, ASCII type 0x01,
	// size field 0x7f 0x00 0x00 0x00 = 0x7f
	crashInput := []byte{
		0x80, 0x01, // header
		0x7f, 0x00, 0x00, 0x00, // size too high
		0xFF, 0xFF, 0xFF, // garbage data
		0xFF, 0xFF, 0xFF,
		0x27, 0x05, 0xF8, 0xFF,
		0xD2, 0x40,
	}
	_, err := NewPfbParserBytes(crashInput)
	if err == nil {
		t.Fatal("a size past the end of the input was accepted")
	}
	if got, want := err.Error(), "EOF while reading PFB font"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

// Test1SegmentOnly checks that a PFB with only 1 short segment reports an
// error.
func Test1SegmentOnly(t *testing.T) {
	// 18-byte crafted PFB: start marker 0x80, ASCII type 0x01,
	// size field 0x04 0x00 0x00 0x00 = 0x04
	crashInput := []byte{
		0x80, 0x01, // header
		0x03, 0x00, 0x00, 0x00, // size
		0xFF, 0xFF, 0xFF, // garbage data
	}
	_, err := NewPfbParserBytes(crashInput)
	if err == nil {
		t.Fatal("a one-segment PFB was accepted")
	}
	if got, want := err.Error(), "PFB header missing"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}
