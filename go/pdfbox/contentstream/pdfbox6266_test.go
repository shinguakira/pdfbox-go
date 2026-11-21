package contentstream_test

// PDFBOX-6266, Apache 96337d173, in the sync of 2026-11-21.
//
// A Type 3 glyph is a content stream, and nothing stops that stream from
// showing text in the same Type 3 font. Until this commit `showType3Glyph` ran
// it without asking how deep it already was, so such a font recursed until the
// stack ran out: StackOverflowError in Java, a fatal stack overflow in Go,
// which takes the process rather than the page. The engine has carried the
// counter for exactly this since before the port -- `increaseLevel`,
// `getLevel`, `decreaseLevel` -- and the three `DrawObject` processors use it
// for form XObjects. This commit wires it into the Type 3 path too, with the
// same limit of fifty.
//
// The port reproduced the asymmetry, which is why the test below crashed the
// test binary before the fix rather than failing: `ShowType3Glyph` had no
// bound, so there was nothing to report.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator/text"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// type3Depth is an engine that counts the Type 3 glyphs it is asked for and
// then hands each one to the engine, so a glyph procedure that shows itself is
// followed as far as the engine will follow it. The recorder in
// streamenginetext_test.go records the call instead of running the stream,
// which is the opposite of what this needs.
type type3Depth struct {
	*contentstream.PDFStreamEngine
	calls int
}

func newType3Depth() *type3Depth {
	d := &type3Depth{PDFStreamEngine: contentstream.NewPDFStreamEngine()}
	d.SetOverrides(d)
	state.AddAll(d.PDFStreamEngine)
	text.AddAll(d.PDFStreamEngine)
	return d
}

func (d *type3Depth) ShowType3Glyph(trm *util.Matrix, f *font.PDType3Font, code int,
	displacement util.Vector) error {
	d.calls++
	return d.PDFStreamEngine.ShowType3Glyph(trm, f, code, displacement)
}

// streamOf returns a COS stream holding the given content.
func streamOf(t *testing.T, content string) *cos.Stream {
	t.Helper()
	stream := cos.NewStream(filter.Provider{})
	w, err := stream.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return stream
}

// selfShowingType3Page returns a page that shows one glyph of a Type 3 font
// whose procedure for that glyph shows the same glyph again. The cycle is in
// the font's own /Resources, which names the font itself as /F1, and it is a
// file a few hundred bytes long.
func selfShowingType3Page(t *testing.T) *pdmodel.PDPage {
	t.Helper()

	fontDict := cos.NewDictionary()
	fontDict.SetItem(cos.Type, cos.Font)
	fontDict.SetItem(cos.Subtype, cos.Type3)

	matrix := cos.NewArray()
	for _, v := range []float32{0.001, 0, 0, 0.001, 0, 0} {
		matrix.Add(cos.NewFloat(v))
	}
	fontDict.SetItem(cos.FontMatrix, matrix)

	bbox := cos.NewArray()
	for _, v := range []float32{0, 0, 1000, 1000} {
		bbox.Add(cos.NewFloat(v))
	}
	fontDict.SetItem(cos.FontBBox, bbox)

	charProcs := cos.NewDictionary()
	charProcs.SetItem(cos.GetPDFName("a"), streamOf(t, "1000 0 d0 BT /F1 10 Tf (a) Tj ET"))
	fontDict.SetItem(cos.CharProcs, charProcs)

	differences := cos.NewArray()
	differences.Add(cos.GetInteger(97))
	differences.Add(cos.GetPDFName("a"))
	encoding := cos.NewDictionary()
	encoding.SetItem(cos.Type, cos.Encoding)
	encoding.SetItem(cos.Differences, differences)
	fontDict.SetItem(cos.Encoding, encoding)

	widths := cos.NewArray()
	widths.Add(cos.NewFloat(1000))
	fontDict.SetItem(cos.Widths, widths)
	fontDict.SetInt(cos.FirstChar, 97)
	fontDict.SetInt(cos.LastChar, 97)

	// the glyph procedure is drawn against the font's own resources, which name
	// the font again: this is the cycle
	fonts := cos.NewDictionary()
	fonts.SetItem(cos.GetPDFName("F1"), fontDict)
	fontResources := cos.NewDictionary()
	fontResources.SetItem(cos.Font, fonts)
	fontDict.SetItem(cos.Resources, fontResources)

	page := pageWith(t, "BT /F1 10 Tf (a) Tj ET")
	page.Resources().Dictionary().SetItem(cos.Font, fonts)
	return page
}

// TestType3GlyphRecursionIsBounded is the fix: the glyph that draws itself is
// entered fifty-one times and then skipped, so the page comes back rather than
// taking the process with it.
//
// Fifty-one is the Java's number, not a choice of the port's: `showType3Glyph`
// increases the level and then skips when the level is above fifty, so the
// fifty-first entry is the one that returns early.
func TestType3GlyphRecursionIsBounded(t *testing.T) {
	d := newType3Depth()
	if err := d.ProcessPage(selfShowingType3Page(t)); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	if d.calls != 51 {
		t.Errorf("the glyph was entered %d times, want 51", d.calls)
	}
}

// TestType3GlyphLevelIsGivenBack is the other half of the guard. Java releases
// the level in a `finally`, so a glyph that is skipped, or one whose stream
// fails, does not leave the counter raised for the rest of the page: a second
// paragraph of text would otherwise start fifty levels deep and be skipped
// outright.
func TestType3GlyphLevelIsGivenBack(t *testing.T) {
	d := newType3Depth()
	if err := d.ProcessPage(selfShowingType3Page(t)); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	if got := d.Level(); got != 0 {
		t.Errorf("level = %d after the page, want 0", got)
	}

	// and the engine still shows a Type 3 glyph after all that
	before := d.calls
	if err := d.ProcessPage(selfShowingType3Page(t)); err != nil {
		t.Fatalf("second ProcessPage: %v", err)
	}
	if got := d.calls - before; got != 51 {
		t.Errorf("the second page entered the glyph %d times, want 51", got)
	}
}
