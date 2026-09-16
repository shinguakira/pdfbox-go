package ttf

import (
	"fmt"
	"math"
	"testing"
)

// TestOpenTypeWithoutCFFRefusesAGlyphTable pins the call OpenTypeFont.getPath
// makes into TrueTypeFont.getPath for a font it does not support: an OpenType
// font that says OTTO and carries CFF2 outlines but no CFF.
//
// That font is not isSupportedOTF, so getPath hands it to super.getPath, which
// asks getGlyph for the glyph table -- a virtual call that reaches
// OpenTypeFont.getGlyph, and that throws UnsupportedOperationException, "OTF
// fonts do not have a glyf table". The port's TrueTypeFont.GetPath asked its own
// Glyph instead, found no table and dereferenced nil.
//
// The running PDFBox, over an OpenTypeFont given the OTTO version and one table
// tagged CFF2, printed isPostScript=true, isSupportedOTF=false and
//
//	getPath threw UnsupportedOperationException: OTF fonts do not have a glyf table
//
// which is unchecked, so the port panics with the same message.
func TestOpenTypeWithoutCFFRefusesAGlyphTable(t *testing.T) {
	font := &OpenTypeFont{TrueTypeFont: NewTrueTypeFont(nil)}
	font.isOpenType = true
	font.SetVersion(math.Float32frombits(ottoVersion))
	font.tables[CFF2Tag] = &UnknownTable{}

	if !font.IsPostScript() || font.IsSupportedOTF() {
		t.Fatalf("IsPostScript = %v, IsSupportedOTF = %v; the fixture should be a PostScript font this does not support",
			font.IsPostScript(), font.IsSupportedOTF())
	}

	const want = "OTF fonts do not have a glyf table"
	got := func() (message string) {
		defer func() { message = fmt.Sprint(recover()) }()
		font.GetPath("A") //nolint:errcheck // it panics
		return "no panic"
	}()
	if got != want {
		t.Errorf("GetPath panicked with %q, want %q", got, want)
	}
}
