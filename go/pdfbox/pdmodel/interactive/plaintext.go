// Package interactive holds the text layout the interactive form appearances
// are written with.
//
// Port of org.apache.pdfbox.pdmodel.interactive, which holds only these four
// classes; the annotations, actions, forms and the rest are its subpackages.
package interactive

import (
	"unicode"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// fontScale is the glyph space a font's widths are given in.
//
// Port of the private PlainText.FONTSCALE.
const fontScale = 1000

// PlainText is the text of a form field, split into paragraphs.
//
// Port of PlainText.
type PlainText struct {
	paragraphs []*Paragraph
}

// NewPlainText splits the given value into paragraphs on its line breaks.
func NewPlainText(textValue string) *PlainText {
	if textValue == "" {
		return &PlainText{paragraphs: []*Paragraph{NewParagraph("")}}
	}
	parts := splitOnLineBreaks(replaceTabsWithSpaces(textValue))
	paragraphs := make([]*Paragraph, 0, len(parts))
	for _, part := range parts {
		// Acrobat prints a space for an empty paragraph
		if part == "" {
			part = " "
		}
		paragraphs = append(paragraphs, NewParagraph(part))
	}
	return &PlainText{paragraphs: paragraphs}
}

// NewPlainTextOfList takes each value as one paragraph.
func NewPlainTextOfList(listValue []string) *PlainText {
	paragraphs := make([]*Paragraph, 0, len(listValue))
	for _, part := range listValue {
		paragraphs = append(paragraphs, NewParagraph(part))
	}
	return &PlainText{paragraphs: paragraphs}
}

// Paragraphs returns the paragraphs of the text.
func (t *PlainText) Paragraphs() []*Paragraph { return t.paragraphs }

// replaceTabsWithSpaces is String.replace('\t', ' ').
func replaceTabsWithSpaces(text string) string {
	runes := []rune(text)
	for i, r := range runes {
		if r == '\t' {
			runes[i] = ' '
		}
	}
	return string(runes)
}

// splitOnLineBreaks splits on the line breaks Java's \R matches: a carriage
// return and line feed pair, or any one of the seven single line break
// characters.
func splitOnLineBreaks(text string) []string {
	parts := []string{}
	current := []rune{}
	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\r' && i+1 < len(runes) && runes[i+1] == '\n' {
			parts = append(parts, string(current))
			current = current[:0]
			i++
			continue
		}
		if isLineBreak(r) {
			parts = append(parts, string(current))
			current = current[:0]
			continue
		}
		current = append(current, r)
	}
	if len(parts) == 0 {
		// Pattern.split returns the input whole where the regex never matched,
		// before it strips anything, so a value with no line break in it is one
		// paragraph even when it is empty or only whitespace.
		return []string{string(current)}
	}
	parts = append(parts, string(current))
	// String.split with a limit of zero drops every trailing empty string, not
	// all but one: for "\n" Pattern.split builds ["", ""] and strips both, so
	// the result is empty and the value has no paragraphs at all.
	last := len(parts)
	for last > 0 && parts[last-1] == "" {
		last--
	}
	return parts[:last]
}

// isLineBreak reports whether r is one of the characters Java's \R matches on
// its own.
func isLineBreak(r rune) bool {
	switch r {
	case '\n', '\v', '\f', '\r', 0x85, 0x2028, 0x2029:
		return true
	}
	return false
}

// Paragraph is one paragraph of the text.
//
// Port of the nested class PlainText.Paragraph.
type Paragraph struct {
	textContent string
}

// NewParagraph returns a paragraph of the given text. Java declares the
// constructor package-private.
func NewParagraph(text string) *Paragraph {
	return &Paragraph{textContent: text}
}

// Text returns the text of the paragraph. Java declares it package-private.
func (p *Paragraph) Text() string { return p.textContent }

// Lines breaks the paragraph into lines no wider than width.
func (p *Paragraph) Lines(f font.PDFont, fontSize, width float32) ([]*Line, error) {
	if width <= 0 {
		return []*Line{}, nil
	}

	segments := lineBreakSegments(p.textContent)
	scale := fontSize / fontScale
	lineWidth := float32(0)
	textLines := []*Line{}
	textLine := &Line{}

	for i := 0; i < len(segments); i++ {
		word := segments[i]
		wordWidth, err := stringWidth(f, word, scale)
		if err != nil {
			return nil, err
		}
		lineWidth += wordWidth

		// check if the last word would fit without the whitespace ending it
		wordRunes := []rune(word)
		if lineWidth >= width && isJavaWhitespace(wordRunes[len(wordRunes)-1]) {
			whitespaceWidth, err := stringWidth(f, string(wordRunes[len(wordRunes)-1:]), scale)
			if err != nil {
				return nil, err
			}
			lineWidth -= whitespaceWidth
		}

		if lineWidth >= width && len(textLine.words) != 0 {
			calculated, err := textLine.CalculateWidth(f, fontSize)
			if err != nil {
				return nil, err
			}
			textLine.SetWidth(calculated)
			textLines = append(textLines, textLine)
			textLine = &Line{}
			lineWidth, err = stringWidth(f, word, scale)
			if err != nil {
				return nil, err
			}
		}

		if len(wordRunes) > 1 && wordWidth > width && len(textLine.words) == 0 {
			// Single word does not fit into the available width.
			// PDFBOX-6082: at least 1 character must be placed per line.

			// PDFBOX-5049: The original approach was to decrement splitOffset
			// until the substring fits, but this can be very expensive for long words and
			// narrow widths (e.g. a long URL in a narrow column).
			//
			// Optimization: instead of decrementing splitOffset one step at a time and
			// calling getStringWidth on progressively shorter substrings:
			//   - compute the scaled width of every individual character once
			//   - build a prefix-sum array
			//   - binary-search for the largest prefix that fits
			//
			// TODO: The special case in PDFBOX-5049 should be handled by not generating an appearance
			// stream at all as the the height of the text box is only 1pt and the text is not visible.
			prefixWidth, err := buildPrefixWidths(wordRunes, f, scale)
			if err != nil {
				return nil, err
			}
			splitOffset := findMaxFittingChars(prefixWidth, width)
			word = string(wordRunes[:splitOffset])
			wordWidth = prefixWidth[splitOffset]
			lineWidth = wordWidth

			// the rest of the segment is measured again on the next line
			segments[i] = string(wordRunes[splitOffset:])
			i--
		}

		textLine.AddWord(&Word{textContent: word, width: wordWidth})
	}

	calculated, err := textLine.CalculateWidth(f, fontSize)
	if err != nil {
		return nil, err
	}
	textLine.SetWidth(calculated)
	textLines = append(textLines, textLine)
	return textLines, nil
}

// stringWidth returns how wide the given text is at the given scale.
func stringWidth(f font.PDFont, text string, scale float32) (float32, error) {
	width, err := f.StringWidth(text)
	if err != nil {
		return 0, err
	}
	return width * scale, nil
}

// buildPrefixWidths returns the cumulative width of every prefix of the word.
//
// Java indexes by char and carries the width of a surrogate pair on its first
// half; Go indexes by rune, where a code point is one element, so the second
// half has nothing to carry.
func buildPrefixWidths(word []rune, f font.PDFont, scale float32) ([]float32, error) {
	prefixWidth := make([]float32, len(word)+1)
	for i, r := range word {
		// Measure this code point as a single string.
		cpWidth, err := stringWidth(f, string(r), scale)
		if err != nil {
			return nil, err
		}
		prefixWidth[i+1] = prefixWidth[i] + cpWidth
	}
	return prefixWidth, nil
}

// findMaxFittingChars returns the longest prefix that stays under width.
func findMaxFittingChars(prefixWidth []float32, width float32) int {
	lo := 1
	hi := len(prefixWidth) - 1
	for lo < hi {
		mid := (lo + hi + 1) / 2 // upper-mid to avoid infinite loop
		if prefixWidth[mid] < width {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// lineBreakSegments splits the text at the places a line may be broken, each
// segment carrying the break character that ends it.
//
// Java uses java.text.BreakIterator.getLineInstance(), which follows the line
// breaking of Unicode Annex 14. Go has no such iterator: the port breaks after
// a run of whitespace and after a hyphen, which is what those rules come to for
// the Latin text a form field holds. Text in a script that breaks by its own
// rules -- Thai, Khmer, Japanese -- will be broken differently here. See
// migration/STATUS.md.
func lineBreakSegments(text string) []string {
	runes := []rune(text)
	segments := []string{}
	start := 0
	for i := 0; i < len(runes); i++ {
		breakAfter := false
		switch {
		case isJavaWhitespace(runes[i]):
			breakAfter = i+1 < len(runes) && !isJavaWhitespace(runes[i+1])
		case runes[i] == '-':
			breakAfter = i+1 < len(runes) && !isJavaWhitespace(runes[i+1]) && runes[i+1] != '-'
		}
		if breakAfter {
			segments = append(segments, string(runes[start:i+1]))
			start = i + 1
		}
	}
	if start < len(runes) {
		segments = append(segments, string(runes[start:]))
	}
	return segments
}

// Line is one line of a paragraph.
//
// Port of the nested class PlainText.Line, which Java declares package-private.
type Line struct {
	words     []*Word
	lineWidth float32
}

// Width returns the width of the line.
func (l *Line) Width() float32 { return l.lineWidth }

// SetWidth sets the width of the line.
func (l *Line) SetWidth(width float32) { l.lineWidth = width }

// CalculateWidth adds up the words, less the whitespace ending the last one.
func (l *Line) CalculateWidth(f font.PDFont, fontSize float32) (float32, error) {
	scale := fontSize / fontScale
	calculatedWidth := float32(0)
	for indexOfWord, word := range l.words {
		calculatedWidth += word.width
		text := []rune(word.Text())
		if indexOfWord == len(l.words)-1 && isJavaWhitespace(text[len(text)-1]) {
			whitespaceWidth, err := stringWidth(f, string(text[len(text)-1:]), scale)
			if err != nil {
				return 0, err
			}
			calculatedWidth -= whitespaceWidth
		}
	}
	return calculatedWidth, nil
}

// Words returns the words of the line.
func (l *Line) Words() []*Word { return l.words }

// InterWordSpacing returns how much to add between the words to justify the
// line to the given width.
func (l *Line) InterWordSpacing(width float32) float32 {
	return (width - l.lineWidth) / float32(len(l.words)-1)
}

// AddWord appends a word to the line.
func (l *Line) AddWord(word *Word) { l.words = append(l.words, word) }

// Word is one word of a line, with the width it was measured at.
//
// Port of the nested class PlainText.Word. Java carries the width in an
// AttributedString under the TextAttribute.WIDTH attribute, which is a way of
// attaching a value to a string that Go has no need of.
type Word struct {
	textContent string
	width       float32
}

// Text returns the text of the word.
func (w *Word) Text() string { return w.textContent }

// Width returns the width the word was measured at, which is Java's
// TextAttribute.WIDTH attribute.
func (w *Word) Width() float32 { return w.width }

// isJavaWhitespace reports whether r is whitespace the way
// Character.isWhitespace does: a Unicode space that is not one of the three
// non-breaking ones, or one of the six control characters it names.
//
// Go's unicode.IsSpace counts the non-breaking space as space, which Java does
// not, so the port cannot use it here.
func isJavaWhitespace(r rune) bool {
	switch r {
	case 0x00A0, 0x2007, 0x202F:
		// non-breaking spaces
		return false
	case '\t', '\n', '\v', '\f', '\r', 0x1C, 0x1D, 0x1E, 0x1F:
		return true
	}
	return unicode.IsSpace(r)
}
