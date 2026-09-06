package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// GlyphsAndPositions is a run of glyph ids with the positioning adjustments
// between them, as the TJ operator takes them.
//
// Port of GlyphsAndPositions. Java's list holds a GlyphSubList or a Float in
// each slot; the port holds the same two in an any.
type GlyphsAndPositions struct {
	list []any
}

// GlyphSubList is one run of glyph ids.
//
// Port of the nested class GlyphsAndPositions.GlyphSubList.
type GlyphSubList struct {
	glyphs []int
}

// IntArray returns the glyph ids of the run.
func (l *GlyphSubList) IntArray() []int { return l.glyphs }

// AddGlyph appends a glyph id to the run at the end of the list, starting a run
// where the last entry is a position.
func (g *GlyphsAndPositions) AddGlyph(glyph int) {
	var last any
	if len(g.list) != 0 {
		last = g.list[len(g.list)-1]
	}
	glyphSubList, isSubList := last.(*GlyphSubList)
	if !isSubList {
		glyphSubList = &GlyphSubList{}
		g.list = append(g.list, glyphSubList)
	}
	glyphSubList.glyphs = append(glyphSubList.glyphs, glyph)
}

// AddPosition appends a positioning adjustment.
func (g *GlyphsAndPositions) AddPosition(position float32) {
	g.list = append(g.list, position)
}

// IsEmpty reports whether nothing has been added.
func (g *GlyphsAndPositions) IsEmpty() bool { return len(g.list) == 0 }

// Clear empties the list.
func (g *GlyphsAndPositions) Clear() { g.list = nil }

// Array returns the runs and adjustments in order.
//
// Java answers a copy through Collections.unmodifiableList, so the caller
// cannot change the list; the port copies the slice for the same reason.
func (g *GlyphsAndPositions) Array() []any {
	array := make([]any, len(g.list))
	copy(array, g.list)
	return array
}

// ContentStreamForGlyphLayout is what a glyph layout processor writes through.
//
// Port of ContentStreamForGlyphLayoutInterface.
type ContentStreamForGlyphLayout interface {
	// ShowGlyphsWithPositioning writes the given runs and adjustments.
	ShowGlyphsWithPositioning(glyphsAndPositions *GlyphsAndPositions) error

	// ShowGlyphCodes writes the given glyph ids.
	ShowGlyphCodes(glyphCodes []int) error

	// SetTextRise sets the text rise.
	SetTextRise(rise float32) error
}

// GlyphLayoutProcessor lays text out into glyphs, for a script that needs more
// than the font's own encoding.
//
// Port of GlyphLayoutProcessorInterface.
type GlyphLayoutProcessor interface {
	// SupportsFont reports whether this processor can lay text out in the
	// given font.
	SupportsFont(f font.PDFont) bool

	// StringWidth returns how wide the given text is once laid out.
	StringWidth(f *font.PDType0Font, fontSize float32, text string) (float32, error)

	// ShowText writes the given text, laid out, to the given content stream.
	ShowText(contentStream ContentStreamForGlyphLayout, f *font.PDType0Font,
		fontSize float32, text string) error
}
