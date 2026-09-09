package glyphlayout_test

// A1 of `track/raster`: `TestBase.checkRenderIdent`, which every layout test in
// `pdfbox-layout-awt` runs and which this port could not run at all until
// there was a backend.
//
// Java renders the PDF the test just wrote and the reference PDF checked into
// `pdfbox-layout-awt/src/test/resources/pdf/`, and asserts the two images are
// the same size and identical pixel for pixel. Both sides go through the same
// renderer, so it does not ask this port's pixels to match Java's -- it asks
// the port to draw the file it wrote and the file Java wrote the same way,
// which is a test of the layout and of the writer together.
//
// **The differing pixels are counted rather than forbidden**, and the count is
// pinned. Java can demand identity because both files come out of the same
// shaper; this port's shaper differs from AWT's in ways `STATUS.md` records and
// `knownDeviations` in reference_test.go lists one by one, and those show up as
// ink in different places. Pinning the count keeps both halves of that
// discipline: a regression that moves the layout blows past it, and a fix that
// removes a deviation fails too, so the record has to be updated with the code.
//
// The supplementary plane page is **exact** -- zero pixels differ -- which says
// the backend, the writer and the shaper all agree with Java's output for a
// page whose deviations list is empty.

import (
	goimage "image"
	"os"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering/raster"
)

// awtReferencePDFs is where the Java repository keeps the PDFs
// checkRenderIdent compares against.
const awtReferencePDFs = "../../../pdfbox-layout-awt/src/test/resources/pdf/"

// TestLigaturesAndKerningRenderIdent is
// GlyphLayoutLigaturesAndKerningTest.testLigaturesAndKerning's
// checkRenderIdent.
//
// The 2151 pixels are the FiraCode ligature deviations of reference_test.go's
// knownDeviations: AWT substitutes them and this port does not, so the first
// two lines of the page carry different glyphs.
func TestLigaturesAndKerningRenderIdent(t *testing.T) {
	checkRenderIdent(t, "GlyphLayoutLigaturesAndKerning.pdf", ligaturesAndKerningPage, 2151)
}

// TestBidiRenderIdent is GlyphLayoutBidiTest.testBidi's.
func TestBidiRenderIdent(t *testing.T) {
	checkRenderIdent(t, "GlyphLayoutBidi.pdf", bidiPage, 2433)
}

// TestSupplementaryPlaneRenderIdent is GlyphLayoutSMPTest's, and it is exact.
func TestSupplementaryPlaneRenderIdent(t *testing.T) {
	checkRenderIdent(t, "GlyphLayoutSMP.pdf", supplementaryPlanePage, 0)
}

// checkRenderIdent lays the page out, saves it, renders it, renders the AWT
// reference of the same name, and compares the two images.
//
// Port of TestBase.checkRenderIdent, whose own comparison is copied from
// ValidateXImage.checkIdent, with the pixel count pinned rather than required
// to be zero -- see the file comment.
func checkRenderIdent(t *testing.T, reference string,
	page func(*testing.T, *pdmodel.PDDocument, *pdmodel.PDPageContentStream),
	wantDiffering int) {
	t.Helper()

	written := layOutAndReload(t, page)
	defer written.Close()
	actual := renderFirstPage(t, written)

	expected := renderFirstPage(t, loadReference(t, reference))

	bounds := expected.Bounds()
	if got := actual.Bounds(); got.Dx() != bounds.Dx() || got.Dy() != bounds.Dy() {
		t.Fatalf("the page renders %dx%d and the reference %dx%d",
			got.Dx(), got.Dy(), bounds.Dx(), bounds.Dy())
	}

	differing, firstX, firstY := 0, -1, -1
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			er, eg, eb, ea := expected.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			ar, ag, ab, aa := actual.At(actual.Bounds().Min.X+x, actual.Bounds().Min.Y+y).RGBA()
			if er != ar || eg != ag || eb != ab || ea != aa {
				if differing == 0 {
					firstX, firstY = x, y
				}
				differing++
			}
		}
	}

	if differing == wantDiffering {
		return
	}
	if differing > wantDiffering {
		t.Errorf("%d pixels differ from the AWT reference and %d are recorded; "+
			"the first is (%d,%d). A layout or backend change has moved ink that "+
			"was in the right place", differing, wantDiffering, firstX, firstY)
		return
	}
	t.Errorf("%d pixels differ from the AWT reference and %d are recorded. "+
		"Something got closer to Java: find which deviation went away, take it "+
		"out of knownDeviations and migration/STATUS.md, and put the new count here",
		differing, wantDiffering)
}

// loadReference opens one of the checked-in AWT reference PDFs, skipping where
// the Java repository is not beside this one.
func loadReference(t *testing.T, name string) *pdmodel.PDDocument {
	t.Helper()
	path := awtReferencePDFs + name
	if _, err := os.Stat(path); err != nil {
		t.Skipf("the AWT reference %s is not in this repository: %v", name, err)
	}
	document, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatalf("loading %s: %v", name, err)
	}
	t.Cleanup(func() { document.Close() })
	return document
}

// renderFirstPage renders page 0 at 72 DPI, which is what
// PDFRenderer.renderImage(0) does.
func renderFirstPage(t *testing.T, document *pdmodel.PDDocument) goimage.Image {
	t.Helper()
	img, err := raster.RenderPage(document, 0, 1, rendering.RGB)
	if err != nil {
		t.Fatalf("rendering page 0: %v", err)
	}
	return img
}
