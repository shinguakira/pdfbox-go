package text

import (
	"strings"

	"github.com/shinguakira/pdfbox-go/go/javatext/bidi"
)

// handleDirection puts a word into visual order, reversing each right to left
// run and swapping its mirrored characters.
//
// Port of org.apache.pdfbox.text.PDFTextStripper.handleDirection, on
// go/javatext/bidi, the port of the java.text.Bidi the Java asks. It used to
// ask golang.org/x/text/unicode/bidi, which answers the runs of a word in
// logical order and not their levels, so a word with runs at more than two
// levels -- a number inside a right-to-left line -- came out with its runs in
// the wrong order; iText's kernel/parser/BidiTextExtractionTest/in02.pdf read
// from the middle of each line. See TestHandleDirectionPutsRunsInVisualOrder.
func handleDirection(word string) string {
	paragraph := bidi.NewParagraph(word, bidi.DirectionDefaultLeftToRight)

	// if there is pure LTR text no need to process further
	if !paragraph.IsMixed() && paragraph.BaseLevel() == bidi.DirectionLeftToRight {
		return word
	}

	// collect individual bidi information
	runCount := paragraph.RunCount()
	levels := make([]int, runCount)
	runs := make([]int, runCount)
	for i := 0; i < runCount; i++ {
		levels[i] = paragraph.RunLevel(i)
		runs[i] = i
	}

	// reorder individual parts based on their levels
	bidi.ReorderVisually(levels, runs)

	// collect the parts based on the direction within the run
	var result strings.Builder
	for i := 0; i < runCount; i++ {
		index := runs[i]
		start := paragraph.RunStart(index)
		end := paragraph.RunLimit(index)

		level := levels[index]

		if level&1 != 0 {
			// Java walks the run backwards with charAt, a UTF-16 code unit at a
			// time, so the two halves of a character outside the basic plane
			// come out in the wrong order, no longer pair, and the character is
			// destroyed. This walks it backwards by code point, which is what
			// visual ordering means and what StringBuilder.reverse -- the same
			// class's other reversal, in TextPosition.getVisuallyOrderedUnicode
			// -- does. See migration/JAVA-BUGS.md 15.
			//
			// Java mirrors a character that Character.isMirrored says is
			// mirrored and MIRRORING_CHAR_MAP has; every character the map
			// holds, from BidiMirroring 8.0.0, is mirrored in the Unicode the
			// running JDK carries, so the map alone answers the same.
			characters := []rune(paragraph.Text(start, end))
			for j := len(characters) - 1; j >= 0; j-- {
				character := characters[j]
				if mirrored, ok := mirroringCharMap[character]; ok {
					character = mirrored
				}
				result.WriteRune(character)
			}
		} else {
			result.WriteString(paragraph.Text(start, end))
		}
	}
	return result.String()
}
