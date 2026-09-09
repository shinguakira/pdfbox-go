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
// These are written before the backend, on purpose: a backend written first
// and compared afterwards is a backend written to whatever it happens to
// produce. Until phase B they fail with raster.ErrNotDrawn.

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
func TestLigaturesAndKerningRenderIdent(t *testing.T) {
	checkRenderIdent(t, "GlyphLayoutLigaturesAndKerning.pdf", ligaturesAndKerningPage)
}

// TestBidiRenderIdent is GlyphLayoutBidiTest.testBidi's.
func TestBidiRenderIdent(t *testing.T) {
	checkRenderIdent(t, "GlyphLayoutBidi.pdf", bidiPage)
}

// TestSupplementaryPlaneRenderIdent is GlyphLayoutSMPTest's.
func TestSupplementaryPlaneRenderIdent(t *testing.T) {
	checkRenderIdent(t, "GlyphLayoutSMP.pdf", supplementaryPlanePage)
}

// checkRenderIdent lays the page out, saves it, renders it, renders the AWT
// reference of the same name, and compares the two images.
//
// Port of TestBase.checkRenderIdent, whose own comparison is copied from
// ValidateXImage.checkIdent.
func checkRenderIdent(t *testing.T, reference string,
	page func(*testing.T, *pdmodel.PDDocument, *pdmodel.PDPageContentStream)) {
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
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			er, eg, eb, ea := expected.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			ar, ag, ab, aa := actual.At(actual.Bounds().Min.X+x, actual.Bounds().Min.Y+y).RGBA()
			if er != ar || eg != ag || eb != ab || ea != aa {
				t.Fatalf("(%d,%d) expected: <%04X%04X%04X%04X> but was: <%04X%04X%04X%04X>",
					x, y, ea, er, eg, eb, aa, ar, ag, ab)
			}
		}
	}
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
