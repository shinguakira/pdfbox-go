package encoding

import (
	"strings"
	"testing"
)

// Java has no test for GlyphList, so these are written from its source and from
// the two glyph list files themselves, which are the input the source reads.
// Every expected value below is either stated in glyphlist.txt /
// zapfdingbats.txt or follows from a rule written in GlyphList.java.

func TestGlyphListToUnicode(t *testing.T) {
	glyphList := AdobeGlyphList()
	cases := []struct {
		name string
		want string
		why  string
	}{
		{"A", "A", "glyphlist.txt: A;0041"},
		{"space", " ", "glyphlist.txt: space;0020"},
		{"ff", "ﬀ", "glyphlist.txt: ff;FB00"},
		{"dalethatafpatah", "דֲ", "glyphlist.txt: two code points"},
		{"A.alt", "A", "a suffix is stripped and the stem looked up again"},
		{"uni0041", "A", "uniXXXX names the code point outright"},
		{"u0041", "A", "uXXXX names the code point outright"},
		{"notaglyphname", "", "not in the list and not of a form that names a code point"},
		{"uniD800", "", "the surrogate area is disallowed"},
		{"uniZZZZ", "", "not a number"},
	}
	for _, c := range cases {
		if got := glyphList.ToUnicode(c.name); got != c.want {
			t.Errorf("ToUnicode(%q) = %q, want %q (%s)", c.name, got, c.want, c.why)
		}
	}
}

// TestGlyphListToUnicodeCache checks that a uniXXXX name gives the same answer
// the second time, which is the path through the cache.
func TestGlyphListToUnicodeCache(t *testing.T) {
	glyphList := AdobeGlyphList()
	first := glyphList.ToUnicode("uni0042")
	second := glyphList.ToUnicode("uni0042")
	if first != "B" || second != "B" {
		t.Errorf("ToUnicode(\"uni0042\") = %q then %q, want %q both times",
			first, second, "B")
	}
}

func TestGlyphListCodePointToName(t *testing.T) {
	glyphList := AdobeGlyphList()
	if got := glyphList.CodePointToName(0x0041); got != "A" {
		t.Errorf("CodePointToName(0x0041) = %q, want %q", got, "A")
	}
	if got := glyphList.CodePointToName(0x00C4); got != "Adieresis" {
		t.Errorf("CodePointToName(0x00C4) = %q, want %q", got, "Adieresis")
	}
	// 2701 is only in zapfdingbats.txt, not in glyphlist.txt
	if got := glyphList.CodePointToName(0x2701); got != ".notdef" {
		t.Errorf("CodePointToName(0x2701) = %q, want %q", got, ".notdef")
	}
}

// TestGlyphListPDFBox3884 checks the rule the reverse mapping is built with:
// where several names reach one code point, a name that one of the standard
// encodings carries wins, whichever order they were read in.
//
// glyphlist.txt has ilde;02DC at line 2304 and tilde;02DC at line 3826. Neither
// putIfAbsent nor read order would pick tilde; WinAnsiEncoding carrying "tilde"
// is what does.
func TestGlyphListPDFBox3884(t *testing.T) {
	glyphList := AdobeGlyphList()
	if got := glyphList.SequenceToName("˜"); got != "tilde" {
		t.Errorf("SequenceToName(U+02DC) = %q, want %q", got, "tilde")
	}
	if !WinAnsiEncodingInstance.ContainsName("tilde") {
		t.Error("WinAnsiEncoding does not carry tilde, which is what forces the override")
	}
	if WinAnsiEncodingInstance.ContainsName("ilde") {
		t.Error("WinAnsiEncoding carries ilde, so the test proves nothing")
	}
}

func TestGlyphListSequenceToName(t *testing.T) {
	glyphList := AdobeGlyphList()
	// neither name is in a standard encoding, so the first one read wins
	if got := glyphList.SequenceToName("דֲ"); got != "dalethatafpatah" {
		t.Errorf("SequenceToName(U+05D3 U+05B2) = %q, want %q", got, "dalethatafpatah")
	}
	if got := glyphList.SequenceToName("no such sequence"); got != ".notdef" {
		t.Errorf("SequenceToName of an unmapped sequence = %q, want %q", got, ".notdef")
	}
}

func TestZapfDingbatsGlyphList(t *testing.T) {
	glyphList := ZapfDingbats()
	if got := glyphList.ToUnicode("a1"); got != "✁" {
		t.Errorf("ToUnicode(\"a1\") = %q, want %q", got, "✁")
	}
	if got := glyphList.ToUnicode("a9"); got != "✠" {
		t.Errorf("ToUnicode(\"a9\") = %q, want %q", got, "✠")
	}
	if got := glyphList.CodePointToName(0x2701); got != "a1" {
		t.Errorf("CodePointToName(0x2701) = %q, want %q", got, "a1")
	}
}

// TestNewGlyphListFrom checks that a list built on another one carries the
// entries of both, which is how LegacyPDFStreamEngine adds additional.txt to
// the Adobe Glyph List.
func TestNewGlyphListFrom(t *testing.T) {
	extended, err := NewGlyphListFrom(AdobeGlyphList(), strings.NewReader("madeupglyph;0041\n"))
	if err != nil {
		t.Fatalf("new glyph list: %v", err)
	}
	if got := extended.ToUnicode("madeupglyph"); got != "A" {
		t.Errorf("ToUnicode(\"madeupglyph\") = %q, want %q", got, "A")
	}
	if got := extended.ToUnicode("space"); got != " " {
		t.Errorf("ToUnicode(\"space\") = %q, want %q", got, " ")
	}
	// the list it was built from is left alone
	if got := AdobeGlyphList().ToUnicode("madeupglyph"); got != "" {
		t.Errorf("the Adobe Glyph List gained madeupglyph -> %q", got)
	}
}

// TestGlyphListInvalidEntry checks that a line with no semicolon is rejected,
// which is the one error loadList reports.
func TestGlyphListInvalidEntry(t *testing.T) {
	_, err := NewGlyphList(strings.NewReader("nosemicolon\n"), 1)
	if err == nil {
		t.Fatal("a line with no semicolon was accepted")
	}
	if !strings.Contains(err.Error(), "Invalid glyph list entry") {
		t.Errorf("error = %v, want it to say Invalid glyph list entry", err)
	}
}

// TestGlyphListComments checks that a comment line is skipped, which every
// shipped list starts with.
func TestGlyphListComments(t *testing.T) {
	glyphList, err := NewGlyphList(strings.NewReader("# a comment\nX;0058\n"), 1)
	if err != nil {
		t.Fatalf("new glyph list: %v", err)
	}
	if got := glyphList.ToUnicode("X"); got != "X" {
		t.Errorf("ToUnicode(\"X\") = %q, want %q", got, "X")
	}
}
