package cos

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/TestCOSString.java.
//
// The Java tests assert the PDF-serialised form through COSWriter.writeString,
// which belongs to pdfwriter and is not ported. Those assertions are replaced
// here by checks on the pieces COSWriter reads — Bytes, ToHexString and
// ForceHexForm — so the same behaviour is pinned. The serialisation assertions
// return with pdfwriter; see migration/STATUS.md.

const (
	escCharString = "( test#some) escaped< \\chars>!~1239857 "
)

// createHex mirrors the helper of the same name in the Java test: each UTF-16
// code unit rendered as upper-case hex, with no zero padding.
func createHex(s string) string {
	var sb strings.Builder
	for _, r := range utf16Units(s) {
		sb.WriteString(strings.ToUpper(fmt.Sprintf("%x", r)))
	}
	return sb.String()
}

// utf16Units returns the UTF-16 code units of s, which is what Java's
// String.toCharArray yields.
func utf16Units(s string) []uint16 {
	var out []uint16
	for _, r := range s {
		if r < 0x10000 {
			out = append(out, uint16(r))
			continue
		}
		r -= 0x10000
		out = append(out, uint16(0xD800+(r>>10)), uint16(0xDC00+(r&0x3FF)))
	}
	return out
}

func TestStringObjBaseContract(t *testing.T) {
	assertBaseContract(t, NewStringObj("test cos string"))
}

func TestStringObjAccept(t *testing.T) {
	assertVisits(t, NewStringObj(escCharString), "string")
}

func TestStringObjForceHexForm(t *testing.T) {
	plain := NewStringObj(escCharString)
	if plain.ForceHexForm() {
		t.Error("a plain string reports ForceHexForm() = true")
	}
	hexed := NewStringObjHex(escCharString, true)
	if !hexed.ForceHexForm() {
		t.Error("a hex-forced string reports ForceHexForm() = false")
	}
	// the flag must not change the bytes
	if !bytes.Equal(plain.Bytes(), hexed.Bytes()) {
		t.Error("ForceHexForm changed the stored bytes")
	}
}

func TestStringObjFromHex(t *testing.T) {
	expected := "Quick and simple test"
	hexForm := createHex(expected)

	test1, err := ParseHexString(hexForm)
	if err != nil {
		t.Fatalf("ParseHexString: %v", err)
	}
	if got := test1.Value(); got != expected {
		t.Errorf("Value() = %q, want %q", got, expected)
	}

	test2, err := ParseHexString(createHex(escCharString))
	if err != nil {
		t.Fatalf("ParseHexString: %v", err)
	}
	if got := test2.Value(); got != escCharString {
		t.Errorf("Value() = %q, want %q", got, escCharString)
	}

	if _, err := ParseHexString(hexForm + "xx"); err == nil {
		t.Error("ParseHexString of invalid hex succeeded, want an error")
	}
}

func TestStringObjToHexString(t *testing.T) {
	expected := "Test subject for testing getHex"
	if got := NewStringObj(expected).ToHexString(); got != createHex(expected) {
		t.Errorf("ToHexString() = %q, want %q", got, createHex(expected))
	}
	// escape characters are not escaped in the hex form, only encoded
	esc := NewStringObj(escCharString)
	if got := esc.ToHexString(); got != createHex(escCharString) {
		t.Errorf("ToHexString() = %q, want %q", got, createHex(escCharString))
	}
}

func TestStringObjValue(t *testing.T) {
	testStr := "Test subject for getString()"
	if got := NewStringObj(testStr).Value(); got != testStr {
		t.Errorf("Value() = %q, want %q", got, testStr)
	}

	hexStr, err := ParseHexString(createHex(testStr))
	if err != nil {
		t.Fatalf("ParseHexString: %v", err)
	}
	if got := hexStr.Value(); got != testStr {
		t.Errorf("Value() = %q, want %q", got, testStr)
	}

	if got := NewStringObj(escCharString).Value(); got != escCharString {
		t.Errorf("Value() = %q, want %q", got, escCharString)
	}

	lineFeeds := "Line1\nLine2\nLine3\n"
	if got := NewStringObj(lineFeeds).Value(); got != lineFeeds {
		t.Errorf("Value() = %q, want %q", got, lineFeeds)
	}
}

func TestStringObjBytes(t *testing.T) {
	s := NewStringObj(escCharString)
	assertBytesEqual(t, []byte(escCharString), s.Bytes())
}

func TestStringObjUnicode(t *testing.T) {
	// a single CJK character
	theString := "世"
	if got := NewStringObj(theString).Value(); got != theString {
		t.Errorf("Value() = %q, want %q", got, theString)
	}

	textAscii := "This is some regular text. It should all be expressible in ASCII"
	text8Bit := "En français où les choses sont accentués. En español, así"
	textHighBits := "をクリックしてく"

	for _, s := range []string{textAscii, text8Bit, textHighBits} {
		if got := NewStringObj(s).Value(); got != s {
			t.Errorf("Value() = %q, want %q", got, s)
		}
	}

	// The first two are representable in PDFDocEncoding, so they are stored
	// as single bytes with no byte order mark.
	for _, s := range []string{textAscii, text8Bit} {
		got := NewStringObj(s).Bytes()
		if len(got) != len([]rune(s)) {
			t.Errorf("%q stored in %d bytes, want %d — expected single-byte encoding",
				s, len(got), len([]rune(s)))
		}
		if len(got) >= 2 && got[0] == 0xFE && got[1] == 0xFF {
			t.Errorf("%q was stored with a byte order mark", s)
		}
	}

	// The Japanese text needs UTF-16BE, so it carries the BOM.
	got := NewStringObj(textHighBits).Bytes()
	if len(got) < 2 || got[0] != 0xFE || got[1] != 0xFF {
		t.Fatalf("high-bit text stored without a UTF-16BE byte order mark: % X", got)
	}
	wantHex := "FEFF"
	for _, u := range utf16Units(textHighBits) {
		wantHex += strings.ToUpper(fmt.Sprintf("%x", u))
	}
	if h := NewStringObj(textHighBits).ToHexString(); h != wantHex {
		t.Errorf("ToHexString() = %q, want %q", h, wantHex)
	}
}

func TestStringObjEquals(t *testing.T) {
	// Java repeats this ten times for consistency; once is enough in Go, where
	// nothing here is nondeterministic.
	x1 := NewStringObj("Test")
	if !x1.Equals(x1) {
		t.Error("not reflexive")
	}

	y1 := NewStringObj("Test")
	if !x1.Equals(y1) || !y1.Equals(x1) {
		t.Error("not symmetric")
	}

	// the hex flag is part of equality
	x2 := NewStringObjHex("Test", true)
	if x1.Equals(x2) || x2.Equals(x1) {
		t.Error("strings differing only in ForceHexForm compare equal")
	}

	z1 := NewStringObj("Test")
	if !x1.Equals(y1) || !y1.Equals(z1) || !x1.Equals(z1) {
		t.Error("not transitive")
	}
}

// TestStringObjCompareFromHexString covers PDFBOX-2401: strings parsed from
// different hex must not compare equal.
func TestStringObjCompareFromHexString(t *testing.T) {
	test1, err := ParseHexString("000000FF000000")
	if err != nil {
		t.Fatalf("ParseHexString: %v", err)
	}
	test2, err := ParseHexString("000000FF00FFFF")
	if err != nil {
		t.Fatalf("ParseHexString: %v", err)
	}

	if !test1.Equals(test1) || !test2.Equals(test2) {
		t.Error("not reflexive")
	}
	if test1.ToHexString() == test2.ToHexString() {
		t.Error("distinct strings share a hex form")
	}
	if bytes.Equal(test1.Bytes(), test2.Bytes()) {
		t.Error("distinct strings share their bytes")
	}
	if test1.Equals(test2) || test2.Equals(test1) {
		t.Error("distinct strings compare equal")
	}
	if test1.Value() == test2.Value() {
		t.Error("distinct strings share a decoded value")
	}
}

// TestStringObjEmptyStringWithBOM covers PDFBOX-3881: a string holding only a
// byte order mark decodes to the empty string.
func TestStringObjEmptyStringWithBOM(t *testing.T) {
	for _, hex := range []string{"FEFF", "FFFE"} {
		s, err := ParseHexString(hex)
		if err != nil {
			t.Fatalf("ParseHexString(%q): %v", hex, err)
		}
		if s.Value() != "" {
			t.Errorf("ParseHexString(%q).Value() = %q, want empty", hex, s.Value())
		}
	}
}

func TestStringObjParseHexOddLength(t *testing.T) {
	// An odd number of digits is padded with a trailing zero nibble, so "F"
	// is 0xF0. Java handles this in the isLengthUneven branch.
	s, err := ParseHexString("41F")
	if err != nil {
		t.Fatalf("ParseHexString: %v", err)
	}
	assertBytesEqual(t, []byte{0x41, 0xF0}, s.Bytes())
}

func TestStringObjParseHexWhitespace(t *testing.T) {
	// Leading and trailing whitespace is skipped.
	s, err := ParseHexString("  4142  ")
	if err != nil {
		t.Fatalf("ParseHexString: %v", err)
	}
	assertBytesEqual(t, []byte{0x41, 0x42}, s.Bytes())
}

func TestStringObjString(t *testing.T) {
	// Java: toString returns "COSString{" + getString() + "}"
	if got := NewStringObj("abc").String(); got != "COSString{abc}" {
		t.Errorf("String() = %q, want %q", got, "COSString{abc}")
	}
}

func TestStringObjBytesAreCopied(t *testing.T) {
	src := []byte("Type")
	s := NewStringObjBytes(src)
	src[0] = 'X'
	if got := s.Value(); got != "Type" {
		t.Errorf("Value() = %q after mutating the source slice, want %q", got, "Type")
	}
	// and the accessor hands back a copy too
	out := s.Bytes()
	out[0] = 'Y'
	if got := s.Value(); got != "Type" {
		t.Errorf("Value() = %q after mutating the returned slice, want %q", got, "Type")
	}
}
