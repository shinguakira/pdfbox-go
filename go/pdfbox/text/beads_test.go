package text_test

// Article beads, which the stripper sorts text by and which it never had.
//
// `fillBeadRectangles` set `beadRectangles` to nil and said why: "PDThreadBead
// is a slice this port has not reached". Slice 8 ported `PDThreadBead` and
// `PDPage.ThreadBeads`; the reason stopped being true and nothing went back to
// look, so every glyph on every page fell into one article.
//
// The machinery above it was ported and working the whole time --
// `processTextPosition` divides the glyphs by which bead rectangle contains
// them, `charactersByArticle` keeps a list per division, and `writePage` walks
// them in order. It was being handed an empty list.

import (
	"bytes"
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/pagenavigation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// TestTextIsSortedByArticleBeads is what the beads are for.
//
// The page carries two of them, one over each half, and the text is written
// bottom half first. Read down the page the words come out in the order they
// were written; read by article they come out in the order the beads are in.
func TestTextIsSortedByArticleBeads(t *testing.T) {
	const pageHeight = 200

	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(200, pageHeight))
	document.AddPage(page)

	// Two beads: the top half of the page, then the bottom half. Both in PDF
	// coordinates, where y grows upwards.
	top := pagenavigation.NewPDThreadBead()
	top.SetRectangle(common.NewPDRectangleOf(0, 100, 200, 100))
	bottom := pagenavigation.NewPDThreadBead()
	bottom.SetRectangle(common.NewPDRectangleOf(0, 0, 200, 100))
	page.SetThreadBeads([]*pagenavigation.PDThreadBead{top, bottom})

	// The text is written in the other order: the word that belongs to the
	// second bead is put on the page first.
	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	helvetica, err := font.NewPDType1FontStandard14(font.Helvetica)
	if err != nil {
		t.Fatalf("NewPDType1FontStandard14: %v", err)
	}
	for _, line := range []struct {
		text string
		y    float32
	}{{"BOTTOM", 40}, {"TOP", 150}} {
		if err := stream.BeginText(); err != nil {
			t.Fatal(err)
		}
		if err := stream.SetFont(helvetica, 12); err != nil {
			t.Fatal(err)
		}
		if err := stream.NewLineAtOffset(20, line.y); err != nil {
			t.Fatal(err)
		}
		if err := stream.ShowText(line.text); err != nil {
			t.Fatal(err)
		}
		if err := stream.EndText(); err != nil {
			t.Fatal(err)
		}
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}

	var saved bytes.Buffer
	if err := document.Save(&saved); err != nil {
		t.Fatalf("Save: %v", err)
	}
	reloaded, err := pdfbox.LoadPDFBytes(saved.Bytes())
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	defer reloaded.Close()

	// The page really does carry the beads, so a failure below is about the
	// sorting rather than about the document.
	if got := reloaded.Page(0).ThreadBeads().Size(); got != 2 {
		t.Fatalf("the reloaded page has %d thread beads, want 2", got)
	}

	withBeads := extractWithBeads(t, reloaded, true)
	again := extractWithBeads(t, reloaded, true)
	if again != withBeads {
		t.Errorf("extracting twice gave %q then %q", withBeads, again)
	}
	withoutBeads := extractWithBeads(t, reloaded, false)

	// With the beads off, the words come out in the order the page draws
	// them -- which is the order they were written, bottom first.
	if !before(withoutBeads, "BOTTOM", "TOP") {
		t.Errorf("with beads off the text is %q; the page draws BOTTOM first",
			withoutBeads)
	}
	// With them on, the words come out in the order the beads are in.
	if !before(withBeads, "TOP", "BOTTOM") {
		t.Errorf("with beads on the text is %q; the first bead covers TOP, so "+
			"it comes first", withBeads)
	}
}

// TestBeadRectanglesAreFlipped checks the coordinate flip, which is the part of
// fillBeadRectangles that can be wrong without anything looking wrong.
//
// A bead rectangle is in PDF coordinates and the glyphs are in image
// coordinates, so the y of the rectangle has to be measured from the top of the
// media box instead of the bottom. Getting it the wrong way round puts every
// glyph in the wrong article on a page whose beads are not symmetric.
func TestBeadRectanglesAreFlipped(t *testing.T) {
	const pageHeight = 200

	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(200, pageHeight))
	document.AddPage(page)

	// One bead over the bottom quarter of the page only, so a flip that does
	// not happen puts the glyph outside it.
	bead := pagenavigation.NewPDThreadBead()
	bead.SetRectangle(common.NewPDRectangleOf(0, 0, 200, 50))
	page.SetThreadBeads([]*pagenavigation.PDThreadBead{bead})

	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	helvetica, err := font.NewPDType1FontStandard14(font.Helvetica)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.BeginText(); err != nil {
		t.Fatal(err)
	}
	if err := stream.SetFont(helvetica, 12); err != nil {
		t.Fatal(err)
	}
	// y = 20 in PDF coordinates is inside the bead, and 180 from the top.
	if err := stream.NewLineAtOffset(20, 20); err != nil {
		t.Fatal(err)
	}
	if err := stream.ShowText("INSIDE"); err != nil {
		t.Fatal(err)
	}
	if err := stream.EndText(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}

	var saved bytes.Buffer
	if err := document.Save(&saved); err != nil {
		t.Fatal(err)
	}
	reloaded, err := pdfbox.LoadPDFBytes(saved.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	defer reloaded.Close()

	if got := strings.TrimSpace(extractWithBeads(t, reloaded, true)); got != "INSIDE" {
		t.Errorf("the text extracted with beads on is %q, want %q: the glyph is "+
			"inside the only bead there is", got, "INSIDE")
	}
}

// extractWithBeads extracts a document's text with the bead division on or off.
func extractWithBeads(t *testing.T, document *pdmodel.PDDocument, beads bool) string {
	t.Helper()
	stripper := text.NewPDFTextStripper()
	stripper.SetShouldSeparateByBeads(beads)
	extracted, err := stripper.GetText(document)
	if err != nil {
		t.Fatalf("GetText: %v", err)
	}
	return extracted
}

// before reports whether the first word comes before the second.
func before(text, first, second string) bool {
	i, j := strings.Index(text, first), strings.Index(text, second)
	return i >= 0 && j >= 0 && i < j
}
