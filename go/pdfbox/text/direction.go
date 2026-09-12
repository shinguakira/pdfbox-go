package text

import (
	"strings"

	"golang.org/x/text/unicode/bidi"
)

// handleDirection puts a word into visual order, reversing each right to left
// run and swapping its mirrored characters.
//
// Port of org.apache.pdfbox.text.PDFTextStripper.handleDirection, which uses
// java.text.Bidi; the port uses golang.org/x/text/unicode/bidi, which resolves
// the same algorithm.
func handleDirection(word string) string {
	var p bidi.Paragraph
	if _, err := p.SetString(word); err != nil {
		return word
	}
	order, err := p.Order()
	if err != nil {
		return word
	}

	// A word with no runs at all has to be answered before anything asks the
	// paragraph its direction, and this is the port's problem rather than
	// Java's. x/text resolves a word that is only paragraph separators -- the
	// empty string, or one made solely of Unicode bidi class B: U+000A, U+000D,
	// U+001C, U+001D, U+001E, U+0085 -- to zero runs, and its
	// Ordering.Direction indexes the first of them, so asking panics with an
	// index out of range. java.text.Bidi has no such edge: the paragraph is not
	// mixed and its base level is DIRECTION_LEFT_TO_RIGHT, so Java takes the
	// arm below and returns the word untouched. Nothing to reorder means the
	// word is already in visual order, which is the same answer.
	if order.NumRuns() == 0 {
		return word
	}

	// if there is pure LTR text no need to process further
	if order.NumRuns() <= 1 && p.Direction() == bidi.LeftToRight {
		return word
	}

	var result strings.Builder
	for i := 0; i < order.NumRuns(); i++ {
		run := order.Run(i)
		runText := run.String()
		if run.Direction() == bidi.RightToLeft {
			// Java walks the run backwards with charAt, a UTF-16 code unit at a
			// time, so the two halves of a character outside the basic plane
			// come out in the wrong order, no longer pair, and the character is
			// destroyed. This walks it backwards by code point, which is what
			// visual ordering means and what StringBuilder.reverse -- the same
			// class's other reversal, in TextPosition.getVisuallyOrderedUnicode
			// -- does. See migration/JAVA-BUGS.md 15.
			characters := []rune(runText)
			for j := len(characters) - 1; j >= 0; j-- {
				character := characters[j]
				if mirrored, ok := mirroringCharMap[character]; ok {
					character = mirrored
				}
				result.WriteRune(character)
			}
		} else {
			result.WriteString(runText)
		}
	}
	return result.String()
}
