package raster_test

// Port of org.apache.pdfbox.rendering.TestQuality.
//
// Four cases, each a single pixel or a single count, and each the whole of what
// a numbered bug was about. They are the cheapest kind of rendering test there
// is: no reference image, no tolerance, just "this one pixel must not be
// white" or "this page must still be bitonal". Every expected value below is
// the Java's, copied.
//
// They read from `target/pdfs`, which `migration/scripts/fetch-testdata.ps1`
// fills.

import (
	goimage "image"
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering/raster"
)

// qualityPDFDir is Java's TARGET_PDF_DIR.
const qualityPDFDir = "../../../../pdfbox/target/pdfs/"

// openQuality loads one of them, skipping where the fetch has not been run.
func openQuality(t *testing.T, name string) *pdmodel.PDDocument {
	t.Helper()
	path := qualityPDFDir + name
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s is not there; run migration/scripts/fetch-testdata.ps1", name)
	}
	document, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatalf("LoadPDF(%s): %v", name, err)
	}
	t.Cleanup(func() { document.Close() })
	return document
}

// rgbAt answers a pixel the way java.awt.image.BufferedImage.getRGB does: one
// int, alpha in the top byte.
func rgbAt(rendered goimage.Image, x, y int) uint32 {
	r, g, b, a := rendered.At(x, y).RGBA()
	return a>>8<<24 | r>>8<<16 | g>>8<<8 | b>>8
}

// colorCount is ValidateXImage.colorCount: how many distinct colours the image
// holds.
func colorCount(rendered goimage.Image) int {
	seen := map[uint32]bool{}
	bounds := rendered.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			seen[rgbAt(rendered, x, y)] = true
		}
	}
	return len(seen)
}

// TestPDFBox4831 is testPDFBox4831: a 300 dpi bitonal scan rendered at 300 dpi
// must still be bitonal, and must be the scan.
func TestPDFBox4831(t *testing.T) {
	document := openQuality(t, "PDFBOX-4831.pdf")

	rendered, err := raster.RenderPageWithDPI(document, 0, 300, rendering.RGB, false)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	if got := colorCount(rendered); got != 2 {
		t.Errorf("the rendered page holds %d colours, want 2: a bitonal scan "+
			"rendered at its own resolution must not be resampled", got)
	}

	xobject, err := document.Page(0).Resources().GetXObject(cos.GetPDFName("I0"))
	if err != nil {
		t.Fatalf("GetXObject(I0): %v", err)
	}
	imageXObject, isImage := xobject.(*image.PDImageXObject)
	if !isImage {
		t.Fatalf("/I0 is %T, want an image XObject", xobject)
	}
	extracted, err := imageXObject.Image()
	if err != nil {
		t.Fatalf("Image: %v", err)
	}

	// ValidateXImage.checkIdent: same size, same pixels.
	if extracted.Bounds().Dx() != rendered.Bounds().Dx() ||
		extracted.Bounds().Dy() != rendered.Bounds().Dy() {
		t.Fatalf("the scan is %v and the render is %v",
			extracted.Bounds(), rendered.Bounds())
	}
	for y := 0; y < extracted.Bounds().Dy(); y++ {
		for x := 0; x < extracted.Bounds().Dx(); x++ {
			want := rgbAt(extracted, extracted.Bounds().Min.X+x, extracted.Bounds().Min.Y+y)
			got := rgbAt(rendered, rendered.Bounds().Min.X+x, rendered.Bounds().Min.Y+y)
			if got&0xFFFFFF != want&0xFFFFFF {
				t.Fatalf("pixel (%d,%d) rendered %06x and the scan holds %06x",
					x, y, got&0xFFFFFF, want&0xFFFFFF)
			}
		}
	}
}

// TestPDFBox6077 is testPDFBox6077: a stencil mask filled with a pattern must
// not paint the gaps between the pattern's own tiles as opaque black.
//
// Before the fix the stencil mask's alpha overwrote the pattern paint's own
// alpha instead of being combined with it, so any pixel the pattern did not
// itself draw into turned solid black instead of staying transparent.
func TestPDFBox6077(t *testing.T) {
	document := openQuality(t, "PDFBOX-6077-example.pdf")

	rendered, err := raster.RenderPageWithDPI(document, 0, 100, rendering.RGB, false)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	// A gap between the tiling pattern's own painted tiles, which must stay
	// transparent -- that is, show the white page -- rather than turn black.
	if got := rgbAt(rendered, 280, 23); got != 0xFFFFFFFF {
		t.Errorf("the pixel at (280,23) is %08x, want ffffffff", got)
	}
}

// TestPDFBox5842 is testPDFBox5842: a soft mask on a pattern that is used as a
// stencil mask fill must still be visible.
//
// Such a pattern is rendered into a scratch image rather than onto the page,
// and the soft mask's alpha lookup is keyed to absolute page-device pixels, so
// a naive implementation renders the whole thing transparent.
func TestPDFBox5842(t *testing.T) {
	document := openQuality(t, "PDFBOX-5842-reduced.pdf")

	rendered, err := raster.RenderPageWithDPI(document, 0, 100, rendering.RGB, false)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	// Inside the soft-masked pattern's map marker. Blank white here means the
	// alpha lookup is broken and the whole region has gone.
	if got := rgbAt(rendered, 267, 1329); got == 0xFFFFFFFF {
		t.Error("the pixel at (267,1329) is white; the soft-masked pattern " +
			"rendered as nothing")
	}
}

// TestPDFBox5403 is testPDFBox5403: a stencil mask filled with a pattern
// repeated over a large area must not show a hairline seam between tiles.
//
// Combining the mask's alpha with the pattern's own -- which is what
// TestPDFBox6077 asks for -- can expose such a seam as a grey streak through
// otherwise solid text, if it is not smoothed over first.
func TestPDFBox5403(t *testing.T) {
	document := openQuality(t, "PDFBOX-5403-bad-rendering.pdf")

	rendered, err := raster.RenderPageWithDPI(document, 2, 100, rendering.RGB, false)
	if err != nil {
		t.Fatalf("rendering: %v", err)
	}
	// Inside a line of text drawn through a pattern-filled stencil mask.
	pixel := rgbAt(rendered, 159, 115)
	if red := (pixel >> 16) & 0xFF; red >= 100 {
		t.Errorf("the pixel at (159,115) is %06x, and its red is %d; expected "+
			"a dark text pixel but was too light -- a tile-boundary seam is "+
			"showing through", pixel&0xFFFFFF, red)
	}
}
