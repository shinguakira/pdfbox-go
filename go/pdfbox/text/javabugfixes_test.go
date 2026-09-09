package text_test

// JAVA-BUGS 15: `PDFTextStripper.handleDirection` reverses a right-to-left run
// with `word.charAt(end)` counting down, which walks UTF-16 code units. A
// character outside the basic multilingual plane is two of them, so its
// surrogates come out in the wrong order, no longer pair, and the character is
// destroyed.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// TestHandleDirectionReversesByCodePoint walks a right-to-left run holding a
// character outside the basic plane.
//
// The expected value is the run reversed by character, which is what visual
// ordering means: the last character of the logical run is drawn first. It is
// not read off the Java, whose `charAt` loop answers the surrogates swapped;
// `StringBuilder.reverse`, which the same class's `TextPosition
// .getVisuallyOrderedUnicode` uses for the same job, documents that "if there
// are any surrogate pairs included in the sequence, these are treated as
// single characters" and answers this.
func TestHandleDirectionReversesByCodePoint(t *testing.T) {
	// ALEF, ARABIC MATHEMATICAL ALEF (U+1EE00, bidi class AL and outside the
	// basic plane), BEH -- one right-to-left run.
	word := "\u0627\U0001EE00\u0628"

	// the three characters in the reverse order, each one whole
	want := "\u0628\U0001EE00\u0627"

	if got := text.HandleDirectionForTest(word); got != want {
		t.Errorf("handleDirection(%q) = %q, want %q -- reversing the run must not "+
			"split a character outside the basic plane", word, got, want)
	}
}

// TestHandleDirectionStillMirrors keeps the mirroring the reversal carries.
//
// A bracket in a right-to-left run is drawn as the bracket that mirrors it, so
// the run reads correctly right to left. The fix moves the loop from code
// units to code points; every mirrored character is inside the basic plane and
// one unit long, so this must be unchanged.
func TestHandleDirectionStillMirrors(t *testing.T) {
	// ALEF, LEFT PARENTHESIS, BEH -- the parenthesis takes the run's direction
	word := "\u0627(\u0628"

	// reversed, and the parenthesis mirrored on the way
	want := "\u0628)\u0627"

	if got := text.HandleDirectionForTest(word); got != want {
		t.Errorf("handleDirection(%q) = %q, want %q", word, got, want)
	}
}
