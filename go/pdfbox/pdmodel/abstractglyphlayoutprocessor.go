package pdmodel

// Abstract super class for classes implementing GlyphLayoutProcessorInterface.
//
// Port of org.apache.pdfbox.pdmodel.AbstractGlyphLayoutProcessor.
//
// Java's abstract methods are function fields here, filled in by whatever
// backend embeds this: Go has no abstract method, and the two Java backends --
// java.awt.font.TextLayout and Apache FOP -- are the whole of the
// `pdfbox-layout-*` modules, which have no Go equivalent to port. See
// migration/STATUS.md; this class is the part of that branch that does not need
// one.

import (
	"unicode/utf16"

	"github.com/shinguakira/pdfbox-go/go/javatext/bidi"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// TextAndBidiLevel is a run of text and the embedding level it resolved to.
//
// Port of the protected static nested class of the same name.
type TextAndBidiLevel struct {
	text      string
	bidiLevel int
}

// Text returns the text of the run.
func (t TextAndBidiLevel) Text() string { return t.text }

// BidiLevel returns the embedding level. Even is left to right.
func (t TextAndBidiLevel) BidiLevel() int { return t.bidiLevel }

// AbstractGlyphLayoutProcessor holds what the two Java backends share: the
// bidirectional splitting, and the loops over its runs.
type AbstractGlyphLayoutProcessor struct {
	// StringWidthUni is the protected abstract getStringWidthUni: the width of
	// a run that goes one way.
	StringWidthUni func(f *font.PDType0Font, fontSize float32, text string,
		bidiLevel int) (float32, error)

	// ShowTextUniFunc is the protected abstract showTextUni: write a run that
	// goes one way.
	ShowTextUniFunc func(contentStream ContentStreamForGlyphLayout, f *font.PDType0Font,
		fontSize float32, text string, bidiLevel int) error
}

// StringWidth computes the width of a text.
//
// Port of getStringWidth.
func (p *AbstractGlyphLayoutProcessor) StringWidth(f *font.PDType0Font, fontSize float32,
	text string) (float32, error) {
	width := float32(0)
	for _, run := range p.DoBidiSplittingAndReordering(text) {
		runWidth, err := p.StringWidthUni(f, fontSize, run.Text(), run.BidiLevel())
		if err != nil {
			return 0, err
		}
		width += runWidth
	}
	return width, nil
}

// ShowText writes a text using glyph positioning.
//
// Port of showText.
func (p *AbstractGlyphLayoutProcessor) ShowText(contentStream ContentStreamForGlyphLayout,
	f *font.PDType0Font, fontSize float32, text string) error {
	for _, run := range p.DoBidiSplittingAndReordering(text) {
		if err := p.ShowTextUniFunc(contentStream, f, fontSize,
			run.Text(), run.BidiLevel()); err != nil {
			return err
		}
	}
	return nil
}

// DoBidiSplittingAndReordering splits the text into runs that each go one way,
// in the order they are drawn.
//
// Port of the protected doBidiSplittingAndReordering. Java's
// `Objects.requireNonNull(text, "Text must be set")` has no counterpart: a Go
// string cannot be nil, and the empty string answers one empty run, which is
// what the running Java does for "".
func (p *AbstractGlyphLayoutProcessor) DoBidiSplittingAndReordering(
	text string) []TextAndBidiLevel {
	var textAndBidiLevels []TextAndBidiLevel

	units := utf16.Encode([]rune(text))
	if !bidi.RequiresBidi(units) {
		return []TextAndBidiLevel{{text: text, bidiLevel: bidi.DirectionLeftToRight}}
	}

	paragraph := bidi.NewParagraph(text, bidi.DirectionDefaultLeftToRight)
	if !paragraph.IsMixed() {
		return []TextAndBidiLevel{{text: text, bidiLevel: paragraph.BaseLevel()}}
	}

	// Split and Reorder
	// See PDFTextStripper.handleDirection
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

	for i := 0; i < runCount; i++ {
		index := runs[i]
		part := paragraph.Text(paragraph.RunStart(index), paragraph.RunLimit(index))
		textAndBidiLevels = append(textAndBidiLevels,
			TextAndBidiLevel{text: part, bidiLevel: levels[index]})
	}
	return textAndBidiLevels
}
