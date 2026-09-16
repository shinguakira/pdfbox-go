package text_test

// Two more defects the corpus found, both in how the port stood in for the
// JDK's bidirectional text support, and both found by the digest of the text
// rather than by its length. See migration/TESTDATA.md, "iText against the
// Java". Every string is written with escapes: the characters are right to left
// or combining, and a source file that shows them reorders them.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// TestHandleDirectionPutsRunsInVisualOrder pins the defect iText's corpus found
// in kernel/parser/BidiTextExtractionTest/in02.pdf: two lines of Hebrew with
// Latin digits and parentheses in them, 179 characters on both sides and not
// the same ones.
//
// Java's handleDirection asks java.text.Bidi for the runs of the word and their
// levels, puts the runs into visual order with Bidi.reorderVisually, and then
// reverses each odd-level run. The port asked golang.org/x/text/unicode/bidi,
// which answers the runs in logical order and no levels, and wrote them in that
// order: a number inside a right-to-left line came out on the wrong side of
// the words around it, and the line read from its middle. go/javatext/bidi is
// the port of java.text.Bidi that answers levels and reorderVisually, measured
// against the running Java when AbstractGlyphLayoutProcessor was ported, and
// handleDirection now uses it.
//
// Every wanted value is PDFBox's, printed by calling its handleDirection on the
// same word. The first two words are the two lines of in02.pdf, as the stripper
// hands them over.
func TestHandleDirectionPutsRunsInVisualOrder(t *testing.T) {
	cases := []struct{ word, want string }{
		{" \u05E5\u05E0\u05D1 \u05DC\u05E8\u05E7) \u05DF\u05DB\u05D5 (1867 ,\u05D9\u05E0\u05DE\u05E8\u05D2\u05D4 \u05D5\u05D8\u05D5\u05D0 \u05E1\u05D5\u05D0\u05DC\u05D5\u05E7\u05D9\u05E0) - \u05EA\u05D9\u05E0\u05D5\u05DB\u05DE\u05D4 ,(1885 ,\u05DA\u05D0\u05D1\u05D9\u05D9\u05DE \u05DD\u05DC\u05D4\u05DC\u05D9\u05D5\u05D5 \u05E8\u05DC\u05DE\u05D9\u05D9\u05D3 \u05D1\u05D9\u05DC\u05D8\u05D5\u05D2) \u05E2\u05D5\u05E0\u05E4\u05D5\u05D0\u05D4",
			"\u05D4\u05D0\u05D5\u05E4\u05E0\u05D5\u05E2 (\u05D2\u05D5\u05D8\u05DC\u05D9\u05D1 \u05D3\u05D9\u05D9\u05DE\u05DC\u05E8 \u05D5\u05D5\u05D9\u05DC\u05D4\u05DC\u05DD \u05DE\u05D9\u05D9\u05D1\u05D0\u05DA, 1885), \u05D4\u05DE\u05DB\u05D5\u05E0\u05D9\u05EA - (\u05E0\u05D9\u05E7\u05D5\u05DC\u05D0\u05D5\u05E1 \u05D0\u05D5\u05D8\u05D5 \u05D4\u05D2\u05E8\u05DE\u05E0\u05D9, 1867) \u05D5\u05DB\u05DF (\u05E7\u05E8\u05DC \u05D1\u05E0\u05E5 "},
		{"(1897 ,\u05E5\u05E8\u05D5\u05D5\u05E9 \u05D3\u05D5\u05D3) \u05E8\u05D9\u05D5\u05D5\u05D0\u05D4 \u05EA\u05E0\u05D9\u05E4\u05E1\u05D5 ,\u05E8\u05D5\u05D8\u05D9\u05E7\u05D4 \u05EA\u05E0\u05D9\u05E4\u05E1 ,\u05EA\u05D1\u05DB\u05E8\u05D4\u05D5 \u05E8\u05D5\u05D8\u05D9\u05E7\u05D4 \u05E8\u05D8\u05E7 ,(1879 ,\u05D9\u05E0\u05DE\u05E8\u05D2\u05D4",
			"\u05D4\u05D2\u05E8\u05DE\u05E0\u05D9, 1879), \u05E7\u05D8\u05E8 \u05D4\u05E7\u05D9\u05D8\u05D5\u05E8 \u05D5\u05D4\u05E8\u05DB\u05D1\u05EA, \u05E1\u05E4\u05D9\u05E0\u05EA \u05D4\u05E7\u05D9\u05D8\u05D5\u05E8, \u05D5\u05E1\u05E4\u05D9\u05E0\u05EA \u05D4\u05D0\u05D5\u05D5\u05D9\u05E8 (\u05D3\u05D5\u05D3 \u05E9\u05D5\u05D5\u05E8\u05E5, 1897)"},
		{"\u05D0\u05D1\u05D2 abc", "abc \u05D2\u05D1\u05D0"},
		{"abc \u05D0\u05D1\u05D2", "abc \u05D2\u05D1\u05D0"},
		{"Hello \u0627\u0644\u0633\u0644\u0627\u0645 world", "Hello \u0645\u0627\u0644\u0633\u0644\u0627 world"},
		{"\u0627\u0644\u0633\u0644\u0627\u0645 Hello \u0634\u0643\u0631\u0627", "\u0627\u0631\u0643\u0634 Hello \u0645\u0627\u0644\u0633\u0644\u0627"},
		{"1 \u0627\u0644\u0633", "\u0633\u0644\u0627 1"},
		{"(1885) \u05D2\u05D1\u05D0", "\u05D0\u05D1\u05D2 (1885)"},
		{"\u05D2\u05D1\u05D0 (1885)", "(1885) \u05D0\u05D1\u05D2"},
		{"abc (\u05D0\u05D1\u05D2) def", "abc (\u05D2\u05D1\u05D0) def"},
		{"\u05D0\u05D1\u05D2 [abc] \u05D3\u05D4\u05D5", "\u05D5\u05D4\u05D3 [abc] \u05D2\u05D1\u05D0"},
		{"plain latin text", "plain latin text"},
		{"", ""},
		{"12 34", "12 34"},
	}
	for _, c := range cases {
		if got := text.HandleDirectionForTest(c.word); got != c.want {
			t.Errorf("handleDirection(%+q)\n  = %+q\nwant %+q", c.word, got, c.want)
		}
	}
}

// TestVisuallyOrderedUnicodeAsksTheDirectionality pins the defect the digest
// found in pdf.js's TaroUTR50SortedList112.pdf: 64,255 characters on both
// sides, every text position the same on both, and one line of its table of
// Hebrew points with a space on the other side of U+05BF HEBREW POINT RAFE.
//
// TextPosition.getVisuallyOrderedUnicode reverses a position's text when any
// code point of it has Character.getDirectionality RIGHT_TO_LEFT or
// RIGHT_TO_LEFT_ARABIC. The port asked instead whether the code point belongs
// to the Hebrew, Arabic, Syriac, Thaana, NKo, Samaritan or Mandaic script,
// which is a different set: it holds the scripts' marks and digits, which are
// not right to left -- RAFE, merged with the space beside it, reversed the
// pair -- and misses the right-to-left characters of every other script and of
// no script, the right-to-left mark among them. Measured against the running
// JDK over every code point, the script test called 361 code points right to
// left that are not and missed 1,473 that are.
//
// Every wanted value is PDFBox's, printed by building a TextPosition over each
// string and asking it. Seven of them failed before the change.
func TestVisuallyOrderedUnicodeAsksTheDirectionality(t *testing.T) {
	cases := []struct{ unicode, want string }{
		{".\u05BF", ".\u05BF"},                           // FULL STOP, HEBREW POINT RAFE
		{"\u05BF.", "\u05BF."},                           // the other way round
		{"\u05BF ", "\u05BF "},                           // RAFE merged with a space, as in TaroUTR50SortedList112.pdf
		{"\u05D0\u05BF", "\u05BF\u05D0"},                 // ALEF, RAFE
		{"\u05BF\u05D0", "\u05D0\u05BF"},                 // RAFE, ALEF
		{"\u05BF", "\u05BF"},                             // one code point
		{"\u05D0", "\u05D0"},                             // one code point
		{"\u05D0\u05D1", "\u05D1\u05D0"},                 // ALEF, BET
		{"\u05FF.", "\u05FF."},                           // unassigned in the Hebrew block, FULL STOP
		{"\u0660\u0661", "\u0660\u0661"},                 // ARABIC-INDIC DIGIT ZERO and ONE
		{"\u0627\u064B", "\u064B\u0627"},                 // ALEF, FATHATAN
		{"\u064B.", "\u064B."},                           // FATHATAN, FULL STOP
		{"a\u0627", "\u0627a"},                           // LATIN SMALL LETTER A, ALEF
		{"\u200F.", ".\u200F"},                           // RIGHT-TO-LEFT MARK, FULL STOP
		{"\uFB1D\u05B4", "\u05B4\uFB1D"},                 // HEBREW LETTER YOD WITH HIRIQ, HIRIQ
		{"\uFE8D\uFE8E", "\uFE8E\uFE8D"},                 // ARABIC LETTER ALEF ISOLATED and FINAL FORM
		{"\u0710\u0711", "\u0711\u0710"},                 // SYRIAC LETTER ALAPH, SUPERSCRIPT ALAPH
		{"\u07C0\u07C1", "\u07C1\u07C0"},                 // NKO DIGIT ZERO and ONE
		{"\U00010900\U00010901", "\U00010901\U00010900"}, // PHOENICIAN LETTER ALF and BET
		{"\U0001E900.", ".\U0001E900"},                   // ADLAM CAPITAL LETTER ALIF, FULL STOP
		{"ab", "ab"},
	}
	for _, c := range cases {
		position := text.NewTextPosition(0, 612, 792, util.NewMatrix(), 0, 0, 10, 5, 2, c.unicode,
			[]int{0}, nil, 10, 10)
		if got := position.VisuallyOrderedUnicode(); got != c.want {
			t.Errorf("VisuallyOrderedUnicode of %+q = %+q, want %+q", c.unicode, got, c.want)
		}
	}
}
