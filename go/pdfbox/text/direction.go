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

	// if there is pure LTR text no need to process further
	if order.NumRuns() <= 1 && p.Direction() == bidi.LeftToRight {
		return word
	}

	var result strings.Builder
	for i := 0; i < order.NumRuns(); i++ {
		run := order.Run(i)
		runText := run.String()
		if run.Direction() == bidi.RightToLeft {
			runes := []rune(runText)
			for j := len(runes) - 1; j >= 0; j-- {
				character := runes[j]
				if mirrored, ok := mirroringCharMap[character]; ok {
					result.WriteRune(mirrored)
				} else {
					result.WriteRune(character)
				}
			}
		} else {
			result.WriteString(runText)
		}
	}
	return result.String()
}
