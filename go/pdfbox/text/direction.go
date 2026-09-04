package text

import (
	"strings"
	"unicode/utf16"

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

	// if there is pure LTR text no need to process further
	if order.NumRuns() <= 1 && p.Direction() == bidi.LeftToRight {
		return word
	}

	var result strings.Builder
	for i := 0; i < order.NumRuns(); i++ {
		run := order.Run(i)
		runText := run.String()
		if run.Direction() == bidi.RightToLeft {
			// JAVA-BUGS entry 15: Java walks the run backwards with charAt, a
			// UTF-16 code unit at a time, so the two halves of a character
			// outside the basic plane come out in the wrong order and no longer
			// pair. Ported as written: the units are reversed here too, and the
			// halves that no longer pair become the replacement character, which
			// is what Java's String becomes once it is written out as UTF-8.
			units := utf16.Encode([]rune(runText))
			for j := len(units) - 1; j >= 0; j-- {
				character := rune(units[j])
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
