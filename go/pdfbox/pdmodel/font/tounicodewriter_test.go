package font

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/font/TestToUnicodeWriter.java.
//
// Slice 3 deferred this one to slice 7 because ToUnicodeWriter writes a PDF
// object rather than reading one.

import (
	"bytes"
	"strings"
	"testing"
)

// writeToString runs the writer and returns what it wrote. Java reads the bytes
// back as ISO-8859-1; everything the writer emits is ASCII, so the Go string is
// the same characters.
func writeToString(t *testing.T, w *toUnicodeWriter) string {
	t.Helper()
	var baos bytes.Buffer
	if err := w.writeTo(&baos); err != nil {
		t.Fatalf("writeTo: %v", err)
	}
	return baos.String()
}

func assertContains(t *testing.T, output, want string) {
	t.Helper()
	if !strings.Contains(output, want) {
		t.Errorf("the output does not contain %q", want)
	}
}

func TestCMapLigatures(t *testing.T) {
	w := newToUnicodeWriter()

	w.add(0x400, "a")
	w.add(0x401, "b")
	w.add(0x402, "ff")
	w.add(0x403, "fi")
	w.add(0x404, "ffl")

	output := writeToString(t, w)
	assertContains(t, output, "4 beginbfrange")
	assertContains(t, output, "<0402> <0402> <00660066>")
	assertContains(t, output, "<0403> <0403> <00660069>")
	assertContains(t, output, "<0404> <0404> <00660066006C>")
}

func TestCMapCIDOverflow(t *testing.T) {
	w := newToUnicodeWriter()

	w.add(0x3ff, "6")
	w.add(0x400, "7")

	output := writeToString(t, w)
	assertContains(t, output, "2 beginbfrange")
	assertContains(t, output, "<03FF> <03FF> <0036>")
	assertContains(t, output, "<0400> <0400> <0037>")
}

func TestCMapStringOverflow(t *testing.T) {
	w := newToUnicodeWriter()

	string1 := string(rune(0x04FF))
	string2 := string(rune(0x0500))
	w.add(0x3ff, string1)
	w.add(0x400, string2)

	output := writeToString(t, w)
	assertContains(t, output, "2 beginbfrange")
	assertContains(t, output, "<03FF> <03FF> <04FF>")
	assertContains(t, output, "<0400> <0400> <0500>")
}

func TestCMapSurrogates(t *testing.T) {
	w := newToUnicodeWriter()

	w.add(0x300, string(rune(0x2F874)))
	w.add(0x301, string(rune(0x2F876)))
	w.add(0x304, string(rune(0x2F884)))
	w.add(0x305, string(rune(0x2F885)))
	w.add(0x306, string(rune(0x2F886)))

	output := writeToString(t, w)
	assertContains(t, output, "3 beginbfrange")
	assertContains(t, output, "<0300> <0300> <D87EDC74>")
	assertContains(t, output, "<0301> <0301> <D87EDC76>")
	assertContains(t, output, "<0304> <0306> <D87EDC84>")
}

func TestAllowCIDToUnicodeRange(t *testing.T) {
	six := &toUnicodeEntry{cid: 0x03FF, text: "6"}
	seven := &toUnicodeEntry{cid: 0x0400, text: "7"}
	eight := &toUnicodeEntry{cid: 0x0401, text: "8"}

	assertRange(t, false, allowCIDToUnicodeRange(nil, seven), "nil, seven")
	assertRange(t, false, allowCIDToUnicodeRange(six, nil), "six, nil")
	assertRange(t, false, allowCIDToUnicodeRange(six, seven), "six, seven")
	assertRange(t, true, allowCIDToUnicodeRange(seven, eight), "seven, eight")
}

func assertRange(t *testing.T, want, got bool, what string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func TestAllowCodeRange(t *testing.T) {
	denied := [][2]int{
		// Denied progressions (negative)
		{0x000F, 0x0007}, {0x00FF, 0x0000}, {0x03FF, 0x0300},
		{0x0401, 0x0400}, {0xFFFF, 0x0000},
		// Denied progressions (non sequential)
		{0x0000, 0x0000}, {0x0000, 0x000F}, {0x0000, 0x007F}, {0x0000, 0x00FF},
		{0x0007, 0x000F}, {0x007F, 0x00FF}, {0x00FF, 0x00FF},
		// Denied progressions (overflow)
		{0x00FF, 0x0100}, {0x01FF, 0x0200}, {0x03FF, 0x0400}, {0x07FF, 0x0800},
		{0x0FFF, 0x1000}, {0x1FFF, 0x2000}, {0x3FFF, 0x4000}, {0x7FFF, 0x8000},
	}
	for _, pair := range denied {
		if allowCodeRange(pair[0], pair[1]) {
			t.Errorf("allowCodeRange(%#04x, %#04x) = true, want false", pair[0], pair[1])
		}
	}

	// Allowed progressions (positive, sequential, and w/o overflow)
	allowed := [][2]int{
		{0x00, 0x01}, {0x01, 0x02}, {0x03, 0x04}, {0x07, 0x08}, {0x0E, 0x0F},
		{0x1F, 0x20}, {0x3F, 0x40}, {0x7F, 0x80}, {0xFE, 0xFF},
		{0x03FE, 0x03FF}, {0x0400, 0x0401}, {0xFFFE, 0xFFFF},
	}
	for _, pair := range allowed {
		if !allowCodeRange(pair[0], pair[1]) {
			t.Errorf("allowCodeRange(%#04x, %#04x) = false, want true", pair[0], pair[1])
		}
	}
}

func TestAllowDestinationRange(t *testing.T) {
	denied := [][2]string{
		// Denied (bogus)
		{"", ""}, {"0", ""}, {"", "0"},
		// Denied (non sequential)
		{"0", "A"}, {"A", "a"},
		// Denied (overflow)
		{"ÿ", "Ā"},
		// Denied (ligatures)
		{"ff", "fi"},
	}
	for _, pair := range denied {
		if allowDestinationRange(pair[0], pair[1]) {
			t.Errorf("allowDestinationRange(%q, %q) = true, want false", pair[0], pair[1])
		}
	}

	// Allowed (sequential w/o surrogate)
	allowed := [][2]string{
		{" ", "!"}, {"(", ")"}, {"0", "1"}, {"a", "b"}, {"A", "B"},
		{"À", "Á"}, {"þ", "ÿ"},
	}
	for _, pair := range allowed {
		if !allowDestinationRange(pair[0], pair[1]) {
			t.Errorf("allowDestinationRange(%q, %q) = false, want true", pair[0], pair[1])
		}
	}
}

func TestAllowDestinationRangeSurrogates(t *testing.T) {
	// Check surrogates
	endOfBMP := string(rune(0xFFFF))
	beyondBMP := string(rune(0x10000))
	cjk1 := string(rune(0x2F884))
	cjk2 := string(rune(0x2F885))
	cjk3 := string(rune(0x2F886))

	// Denied (overflow)
	assertRange(t, false, allowDestinationRange(endOfBMP, beyondBMP), "endOfBMP, beyondBMP")
	// Allowed (sequential surrogates)
	assertRange(t, true, allowDestinationRange(cjk1, cjk2), "cjk1, cjk2")
	assertRange(t, true, allowDestinationRange(cjk2, cjk3), "cjk2, cjk3")
	// Denied (non sequential surrogates)
	assertRange(t, false, allowDestinationRange(cjk1, cjk3), "cjk1, cjk3")
}
