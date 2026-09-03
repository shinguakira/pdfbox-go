package contentstream_test

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// Written from org.apache.pdfbox.contentstream.PDFStreamEngine.showText and the
// five text-showing operator processors. Java exercises them through
// PDFTextStripper against the test corpus, which this port has not reached.
//
// Every expected width below comes from Helvetica.afm: A and B are 667 units
// wide, the space is 278, and the font matrix scales by a thousandth.

// closeTo reports whether two float32 values agree to within a rounding error,
// which is what comparing an expected advance written as a decimal against one
// the engine accumulated in float32 needs.
func closeTo(got, want float32) bool {
	diff := got - want
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-4
}

// glyph is one call the engine made into the glyph hooks.
type glyph struct {
	code         int
	displacement util.Vector
	trm          *util.Matrix
	isType3      bool
}

// ShowFontGlyph records a glyph of a font that is not Type 3.
func (r *recorder) ShowFontGlyph(trm *util.Matrix, f font.PDFont, code int, displacement util.Vector) error {
	r.glyphs = append(r.glyphs, glyph{code: code, displacement: displacement, trm: trm})
	return nil
}

// ShowType3Glyph records a glyph of a Type 3 font, without running its content
// stream.
func (r *recorder) ShowType3Glyph(trm *util.Matrix, f *font.PDType3Font, code int, displacement util.Vector) error {
	r.glyphs = append(r.glyphs, glyph{code: code, displacement: displacement, trm: trm, isType3: true})
	return nil
}

// helveticaPage returns a page whose resources name Helvetica as /F1, holding
// the given content stream.
func helveticaPage(t *testing.T, content string) *pdmodel.PDPage {
	t.Helper()
	page := pageWith(t, content)

	fontDict := cos.NewDictionary()
	fontDict.SetItem(cos.Type, cos.Font)
	fontDict.SetItem(cos.Subtype, cos.Type1)
	fontDict.SetName(cos.BaseFont, "Helvetica")

	fonts := cos.NewDictionary()
	fonts.SetItem(cos.GetPDFName("F1"), fontDict)
	page.Resources().Dictionary().SetItem(cos.Font, fonts)
	return page
}

// runFont processes a page whose resources name Helvetica as /F1.
func (r *recorder) runFont(t *testing.T, content string) {
	t.Helper()
	if err := r.ProcessPage(helveticaPage(t, content+" sh")); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	if r.probed == nil {
		t.Fatal("the probe operator never ran")
	}
}

// TestEngineSetFontAndSize checks that Tf finds the font in the resources and
// sets it along with the size.
func TestEngineSetFontAndSize(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /F1 12 Tf ET")

	if got := r.probed.TextState().FontSize(); got != 12 {
		t.Errorf("font size = %v, want 12", got)
	}
	f := r.probed.TextState().Font()
	if f == nil {
		t.Fatal("Tf set no font")
	}
	if got := f.Name(); got != "Helvetica" {
		t.Errorf("font name = %q, want %q", got, "Helvetica")
	}
}

// TestEngineSetFontAndSizeMissingFont checks that a font the resources do not
// carry leaves the font unset, which is what Java does after logging.
func TestEngineSetFontAndSizeMissingFont(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /NoSuchFont 12 Tf ET")

	if got := r.probed.TextState().FontSize(); got != 12 {
		t.Errorf("font size = %v, want 12", got)
	}
	if f := r.probed.TextState().Font(); f != nil {
		t.Errorf("a font that is not in the resources gave %v", f)
	}
}

// TestEngineShowText checks that Tj reaches the glyph hook once per byte, and
// that the text matrix advances by the width of each glyph.
func TestEngineShowText(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /F1 10 Tf (AB) Tj ET")

	if len(r.glyphs) != 2 {
		t.Fatalf("%d glyphs, want 2", len(r.glyphs))
	}
	if r.glyphs[0].code != 'A' || r.glyphs[1].code != 'B' {
		t.Errorf("codes = %d %d, want %d %d",
			r.glyphs[0].code, r.glyphs[1].code, 'A', 'B')
	}
	if r.glyphs[0].isType3 {
		t.Error("a Type 1 glyph went to the Type 3 hook")
	}
	for i, g := range r.glyphs {
		if g.displacement.X() != 0.667 {
			t.Errorf("glyph %d displacement x = %v, want 0.667", i, g.displacement.X())
		}
		if g.displacement.Y() != 0 {
			t.Errorf("glyph %d displacement y = %v, want 0", i, g.displacement.Y())
		}
	}
	// the second glyph is drawn one advance to the right of the first: 0.667
	// times the font size of 10
	if got := r.glyphs[1].trm.TranslateX() - r.glyphs[0].trm.TranslateX(); !closeTo(got, 6.67) {
		t.Errorf("advance between the glyphs = %v, want 6.67", got)
	}
}

// TestEngineShowTextWordSpacing checks that word spacing is applied to the
// single-byte code 32 and to nothing else.
func TestEngineShowTextWordSpacing(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /F1 10 Tf 5 Tw (A B) Tj ET")

	if len(r.glyphs) != 3 {
		t.Fatalf("%d glyphs, want 3", len(r.glyphs))
	}
	// A is 667 wide and the space is 278; the space also picks up the 5 units
	// of word spacing, the A does not
	afterA := r.glyphs[1].trm.TranslateX() - r.glyphs[0].trm.TranslateX()
	afterSpace := r.glyphs[2].trm.TranslateX() - r.glyphs[1].trm.TranslateX()
	if !closeTo(afterA, 6.67) {
		t.Errorf("advance after A = %v, want 6.67", afterA)
	}
	if !closeTo(afterSpace, 2.78+5) {
		t.Errorf("advance after the space = %v, want %v", afterSpace, 2.78+5)
	}
}

// TestEngineShowTextCharSpacing checks that character spacing is applied after
// every glyph.
func TestEngineShowTextCharSpacing(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /F1 10 Tf 2 Tc (AB) Tj ET")

	if len(r.glyphs) != 2 {
		t.Fatalf("%d glyphs, want 2", len(r.glyphs))
	}
	if got := r.glyphs[1].trm.TranslateX() - r.glyphs[0].trm.TranslateX(); !closeTo(got, 6.67+2) {
		t.Errorf("advance between the glyphs = %v, want %v", got, 6.67+2)
	}
}

// TestEngineShowTextAdjusted checks that TJ draws the strings and moves the pen
// back by the numbers between them.
func TestEngineShowTextAdjusted(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /F1 10 Tf [(A) -1000 (B)] TJ ET")

	if len(r.glyphs) != 2 {
		t.Fatalf("%d glyphs, want 2", len(r.glyphs))
	}
	// A advances 6.67, then the -1000 moves the pen a further 10 units right:
	// tx is -tj / 1000 * fontSize, so a negative number widens the gap
	if got := r.glyphs[1].trm.TranslateX() - r.glyphs[0].trm.TranslateX(); !closeTo(got, 6.67+10) {
		t.Errorf("advance between the glyphs = %v, want %v", got, 6.67+10)
	}
}

// TestEngineShowTextAdjustedIgnoresJunk checks that TJ skips an entry that is
// neither a number nor a string, which Java logs and passes over.
func TestEngineShowTextAdjustedIgnoresJunk(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /F1 10 Tf [(A) /Name (B)] TJ ET")

	if len(r.glyphs) != 2 {
		t.Fatalf("%d glyphs, want 2", len(r.glyphs))
	}
	if got := r.glyphs[1].trm.TranslateX() - r.glyphs[0].trm.TranslateX(); !closeTo(got, 6.67) {
		t.Errorf("advance between the glyphs = %v, want 6.67", got)
	}
}

// TestEngineShowTextLine checks that the quote operator moves to the next line
// before drawing.
func TestEngineShowTextLine(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /F1 10 Tf 14 TL (A) Tj (B) ' ET")

	if len(r.glyphs) != 2 {
		t.Fatalf("%d glyphs, want 2", len(r.glyphs))
	}
	// the second glyph is one leading below the first, and back at the left
	if got := r.glyphs[1].trm.TranslateY() - r.glyphs[0].trm.TranslateY(); got != -14 {
		t.Errorf("vertical move = %v, want -14", got)
	}
	if got := r.glyphs[1].trm.TranslateX(); got != 0 {
		t.Errorf("the second glyph starts at x = %v, want 0", got)
	}
}

// TestEngineShowTextLineAndSpace checks that the double-quote operator sets the
// word and character spacing before drawing on the next line.
func TestEngineShowTextLineAndSpace(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /F1 10 Tf 14 TL 3 1 (A B) \" ET")

	if got := r.probed.TextState().WordSpacing(); got != 3 {
		t.Errorf("word spacing = %v, want 3", got)
	}
	if got := r.probed.TextState().CharacterSpacing(); got != 1 {
		t.Errorf("character spacing = %v, want 1", got)
	}
	if len(r.glyphs) != 3 {
		t.Fatalf("%d glyphs, want 3", len(r.glyphs))
	}
	// A is 667 wide and picks up the character spacing; the space picks up both
	afterA := r.glyphs[1].trm.TranslateX() - r.glyphs[0].trm.TranslateX()
	afterSpace := r.glyphs[2].trm.TranslateX() - r.glyphs[1].trm.TranslateX()
	if !closeTo(afterA, 6.67+1) {
		t.Errorf("advance after A = %v, want %v", afterA, 6.67+1)
	}
	if !closeTo(afterSpace, 2.78+1+3) {
		t.Errorf("advance after the space = %v, want %v", afterSpace, 2.78+1+3)
	}
}

// TestEngineShowTextOutsideBT checks that Tj and TJ are ignored outside a text
// object, where there is no text matrix to draw against.
func TestEngineShowTextOutsideBT(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "/F1 10 Tf (A) Tj [(B)] TJ")

	if len(r.glyphs) != 0 {
		t.Errorf("%d glyphs were drawn outside BT...ET, want 0", len(r.glyphs))
	}
}

// TestEngineShowTextWithNoFont checks that a stream drawing text without
// setting a font falls back to Helvetica rather than failing.
func TestEngineShowTextWithNoFont(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT (A) Tj ET")

	if len(r.glyphs) != 1 {
		t.Fatalf("%d glyphs, want 1", len(r.glyphs))
	}
	if r.glyphs[0].code != 'A' {
		t.Errorf("code = %d, want %d", r.glyphs[0].code, 'A')
	}
}

// TestEngineTextHorizontalScaling checks that horizontal scaling multiplies the
// advance.
func TestEngineTextHorizontalScaling(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /F1 10 Tf 50 Tz (AB) Tj ET")

	if len(r.glyphs) != 2 {
		t.Fatalf("%d glyphs, want 2", len(r.glyphs))
	}
	// half the scaling, half the advance
	if got := r.glyphs[1].trm.TranslateX() - r.glyphs[0].trm.TranslateX(); !closeTo(got, 6.67/2) {
		t.Errorf("advance between the glyphs = %v, want %v", got, 6.67/2)
	}
}

// TestEngineTextRise checks that the rise lifts the glyph off the baseline
// without moving the pen.
func TestEngineTextRise(t *testing.T) {
	r := newRecorder()
	r.runFont(t, "BT /F1 10 Tf 5 Ts (A) Tj ET")

	if len(r.glyphs) != 1 {
		t.Fatalf("%d glyphs, want 1", len(r.glyphs))
	}
	// the engine does not flip the page -- that is the renderer's doing -- so the
	// glyph sits at the rise above the baseline the text matrix starts at
	if got := r.glyphs[0].trm.TranslateY(); got != 5 {
		t.Errorf("glyph y = %v, want 5", got)
	}
}
