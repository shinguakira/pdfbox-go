package cos

import (
	"bytes"
	"testing"
)

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/TestCOSName.java and,
// where that file has no direct coverage, written from
// pdfbox/src/main/java/org/apache/pdfbox/cos/COSName.java per
// migration/conventions/tdd.md.
//
// All three Java tests drive their assertions through PDDocument, Loader and
// AcroForm, none of which exist yet. Their actual expectations are about
// COSName itself, so they are asserted here directly against writePDF:
//
//	PDFBox4076        non-ASCII names survive without being replaced by '?'
//	PDFBox6178        a 0xE4 byte is written as the escape #E4
//	NameWithASCII_NUL a 0x00 byte is written as the escape #00
//
// The document round-trips those tests also perform return with the parser and
// writer; see migration/STATUS.md.

func TestNameBaseContract(t *testing.T) {
	assertBaseContract(t, GetPDFName("Type"))
}

func TestNameAccept(t *testing.T) {
	assertVisits(t, GetPDFName("Type"), "name")
}

func TestNameGetName(t *testing.T) {
	if got := GetPDFName("Type").Name(); got != "Type" {
		t.Errorf("Name() = %q, want %q", got, "Type")
	}
	if got := GetPDFName("").Name(); got != "" {
		t.Errorf("Name() = %q, want empty", got)
	}
}

// TestNamePDFBox4076 covers PDFBOX-4076: characters outside US-ASCII must not
// be replaced with '?'.
func TestNamePDFBox4076(t *testing.T) {
	const special = "中国你好!"
	n := GetPDFName(special)
	if got := n.Name(); got != special {
		t.Errorf("Name() = %q, want %q", got, special)
	}
}

// TestNameGetNameLatin1Fallback covers the lossy-decode branch: bytes that are
// not valid UTF-8 fall back to ISO-8859-1 rather than yielding U+FFFD.
func TestNameGetNameLatin1Fallback(t *testing.T) {
	// 0xE4 alone is not valid UTF-8; in ISO-8859-1 it is 'ä'.
	n := GetPDFNameBytes([]byte{'m', 0xE4, 'n'})
	if got := n.Name(); got != "män" {
		t.Errorf("Name() = %q, want %q", got, "män")
	}
}

func TestNameEquals(t *testing.T) {
	if !GetPDFName("Type").Equals(GetPDFName("Type")) {
		t.Error("Type != Type")
	}
	if GetPDFName("Type").Equals(GetPDFName("Kids")) {
		t.Error("Type == Kids, want them distinct")
	}
}

// TestNameIsInterned pins the identity guarantee COSName provides through its
// name map: equal names are the same instance. Dictionary lookups rely on it,
// since a Go map keyed by *Name compares pointers.
func TestNameIsInterned(t *testing.T) {
	if GetPDFName("Type") != GetPDFName("Type") {
		t.Error("GetPDFName returned distinct instances for the same name")
	}
	if GetPDFNameBytes([]byte("Type")) != GetPDFName("Type") {
		t.Error("the byte and string factories returned distinct instances")
	}
	if GetPDFName("Type") == GetPDFName("Kids") {
		t.Error("distinct names share an instance")
	}
}

func TestNameCompare(t *testing.T) {
	// Java compareTo is Arrays.compare on the raw bytes.
	if got := GetPDFName("A").Compare(GetPDFName("A")); got != 0 {
		t.Errorf("A vs A = %d, want 0", got)
	}
	if got := GetPDFName("A").Compare(GetPDFName("B")); got >= 0 {
		t.Errorf("A vs B = %d, want negative", got)
	}
	if got := GetPDFName("B").Compare(GetPDFName("A")); got <= 0 {
		t.Errorf("B vs A = %d, want positive", got)
	}
	// a prefix sorts before the longer name
	if got := GetPDFName("Ty").Compare(GetPDFName("Type")); got >= 0 {
		t.Errorf("Ty vs Type = %d, want negative", got)
	}
}

func TestNameIsEmpty(t *testing.T) {
	if !GetPDFName("").IsEmpty() {
		t.Error("the empty name reports IsEmpty() = false")
	}
	if GetPDFName("Type").IsEmpty() {
		t.Error("Type reports IsEmpty() = true")
	}
}

func TestNameString(t *testing.T) {
	// Java: toString returns "COSName{" + getName() + "}"
	if got := GetPDFName("Type").String(); got != "COSName{Type}" {
		t.Errorf("String() = %q, want %q", got, "COSName{Type}")
	}
}

func TestNameWritePDF(t *testing.T) {
	// The character set written literally is deliberately more restrictive
	// than the PDF spec allows; see PDFBOX-2073 in COSName.writePDF.
	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{"alphanumeric", []byte("Type"), "/Type"},
		{"digits", []byte("F1"), "/F1"},
		{"permitted punctuation", []byte("+-_@*$;."), "/+-_@*$;."},
		{"space is escaped", []byte("a b"), "/a#20b"},
		{"hash is escaped", []byte("a#b"), "/a#23b"},
		{"slash is escaped", []byte("a/b"), "/a#2Fb"},
		{"parenthesis is escaped", []byte("a(b"), "/a#28b"},
		{"empty name", []byte(""), "/"},
		// PDFBOX-6178: a 0xE4 byte is written as #E4, not as UTF-8
		{"PDFBOX-6178 latin1 byte", []byte{'m', 0xE4, 'n', 'n', 'l', 'i', 'c', 'h'}, "/m#E4nnlich"},
		// PDFBOX-6178: an ASCII NUL is written as #00
		{"PDFBOX-6178 NUL", []byte{'m', 0x00, 'n', 'n', 'l', 'i', 'c', 'h'}, "/m#00nnlich"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := GetPDFNameBytes(c.in).WritePDF(&buf); err != nil {
				t.Fatalf("WritePDF: %v", err)
			}
			if got := buf.String(); got != c.want {
				t.Errorf("WritePDF = %q, want %q", got, c.want)
			}
		})
	}
}

// TestNameBytesAreCopied checks that a name does not alias the slice it was
// built from. Java copies the bytes into the name map for exactly this reason.
func TestNameBytesAreCopied(t *testing.T) {
	src := []byte("Type")
	n := GetPDFNameBytes(src)
	src[0] = 'X'
	if got := n.Name(); got != "Type" {
		t.Errorf("Name() = %q after mutating the source slice, want %q", got, "Type")
	}
}

// TestNamePredefined spot-checks the predefined constants against the strings
// COSName.java defines them with.
func TestNamePredefined(t *testing.T) {
	cases := []struct {
		got  *Name
		want string
	}{
		{Type, "Type"},
		{Kids, "Kids"},
		{Page, "Page"},
		{Pages, "Pages"},
		{Root, "Root"},
		{Length, "Length"},
		{Filter, "Filter"},
		{FlateDecode, "FlateDecode"},
		{XObject, "XObject"},
		{WinAnsiEncoding, "WinAnsiEncoding"},
	}
	for _, c := range cases {
		if c.got == nil {
			t.Errorf("predefined name for %q is nil", c.want)
			continue
		}
		if got := c.got.Name(); got != c.want {
			t.Errorf("predefined name = %q, want %q", got, c.want)
		}
		// predefined constants must be the interned instance
		if c.got != GetPDFName(c.want) {
			t.Errorf("predefined %q is not the interned instance", c.want)
		}
	}
}
