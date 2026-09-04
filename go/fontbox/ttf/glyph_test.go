package ttf

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.fontbox.ttf.GlyfCompositeDescriptTest.
//
// Java reads the font through OTFParser, which for a TrueType file such as this
// one reads the same tables the plain parser does; the port uses the plain
// parser, which is what this slice carries.

// TestGetComponentsView checks that Components returns every component of the
// glyph, and returns it as a copy rather than as the list itself. Java asserts
// the copy by calling remove on the unmodifiable list it gets back.
func TestGetComponentsView(t *testing.T) {
	fontFile, err := pdfio.OpenBufferedFile(ttfFixture + "LiberationSans-Regular.ttf")
	if err != nil {
		t.Fatalf("open font: %v", err)
	}
	font, err := NewParser().Parse(fontFile)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer font.Close()

	glyphTable, err := font.Glyph()
	if err != nil {
		t.Fatalf("glyf: %v", err)
	}
	// A acute
	aacuteGlyph, err := glyphTable.GetGlyph(131)
	if err != nil {
		t.Fatalf("glyph 131: %v", err)
	}
	if aacuteGlyph == nil {
		t.Fatal("glyph 131 is nil")
	}

	glyphDescription := aacuteGlyph.Description()
	// consists of glyphs 36 & 2335
	if !glyphDescription.IsComposite() {
		t.Fatal("glyph 131 is not composite")
	}

	compositeGlyphDescription, ok := glyphDescription.(*GlyfCompositeDescript)
	if !ok {
		t.Fatalf("description is a %T", glyphDescription)
	}

	componentsView := compositeGlyphDescription.Components()
	if len(componentsView) != 2 {
		t.Fatalf("len(components) = %d, want 2", len(componentsView))
	}

	// the returned slice is a copy: writing through it leaves the glyph alone
	componentsView[0] = nil
	if again := compositeGlyphDescription.Components(); again[0] == nil {
		t.Error("Components returned the components themselves, not a copy")
	}
}
