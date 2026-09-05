package rendering

// PDFRenderer's arithmetic.
//
// Java's TestRendering only asserts that rendering a file threw nothing, which
// with no rasteriser says nothing here. What it was covering -- that the
// renderer works out the right size, type and transform for a page before it
// draws -- is what these check, against the values renderImage computes.

import (
	"errors"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// TestRenderImageSizeIsFlooredAtOnePixel pins PDFBOX-4306: the pixel size is
// the floor of the point size times the scale, and never zero, which is what
// avoids a blank line down the right or the bottom.
func TestRenderImageSizeIsFlooredAtOnePixel(t *testing.T) {
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100.9, 0.4)))

	err := NewPDFRenderer(document).RenderImageScaled(0, 1)
	if !errors.Is(err, ErrNoBackend) {
		t.Fatalf("RenderImageScaled = %v, want ErrNoBackend", err)
	}
	// the message carries the size that would have been made
	if !strings.Contains(err.Error(), "100x1") {
		t.Errorf("error = %q, want the page to be 100x1 pixels", err)
	}
}

// TestRenderImageSwapsTheSizeForARotatedPage pins that a page turned a quarter
// turn is rendered into an image whose width and height are the other way
// round.
func TestRenderImageSwapsTheSizeForARotatedPage(t *testing.T) {
	for _, rotation := range []int{90, 270} {
		document := pdmodel.NewPDDocument()
		page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(200, 100))
		page.SetRotation(rotation)
		document.AddPage(page)

		err := NewPDFRenderer(document).RenderImageScaled(0, 1)
		if !strings.Contains(err.Error(), "100x200") {
			t.Errorf("rotation %d: error = %q, want the image to be 100x200", rotation, err)
		}
	}
}

// TestRenderImageScalesTheSize pins that the scale multiplies both dimensions,
// which is what renderImageWithDPI turns a DPI into.
func TestRenderImageScalesTheSize(t *testing.T) {
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 200)))

	// 144 DPI is twice 72, so the page doubles
	err := NewPDFRenderer(document).RenderImageWithDPI(0, 144)
	if !strings.Contains(err.Error(), "200x400") {
		t.Errorf("error = %q, want the image to be 200x400", err)
	}
}

// TestBlendModeOnThePageForcesAlpha pins PDFBOX-4095: a page whose own
// resources set a blend mode other than Normal is drawn on a transparent
// surface even when RGB was asked for, and composited onto white afterwards.
func TestBlendModeOnThePageForcesAlpha(t *testing.T) {
	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 100))
	document.AddPage(page)

	extGState := cos.NewDictionary()
	extGState.SetItem(cos.BM, cos.Multiply)
	extGStates := cos.NewDictionary()
	extGStates.SetItem(cos.GetPDFName("GS0"), extGState)
	resources := pdmodel.NewPDResources()
	resources.Dictionary().SetItem(cos.ExtGState, extGStates)
	page.SetResources(resources)

	err := NewPDFRenderer(document).RenderImageOfType(0, 1, RGB)
	if !strings.Contains(err.Error(), "ARGB") {
		t.Errorf("error = %q, want the surface to be ARGB", err)
	}

	// and a page with only Normal keeps the type that was asked for
	extGState.SetItem(cos.BM, cos.Normal)
	err = NewPDFRenderer(document).RenderImageOfType(0, 1, RGB)
	if !strings.Contains(err.Error(), "RGB") || strings.Contains(err.Error(), "ARGB") {
		t.Errorf("error = %q, want the surface to stay RGB", err)
	}
}

// TestMaximumImageSizeIsRefused pins PDFBOX-4518: an image bigger than
// Integer.MAX_VALUE pixels cannot be made, and is refused before anything is
// allocated.
func TestMaximumImageSizeIsRefused(t *testing.T) {
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100000, 100000)))

	err := NewPDFRenderer(document).RenderImageScaled(0, 1)
	if err == nil || !strings.Contains(err.Error(), "Maximum size of image exceeded") {
		t.Errorf("error = %v, want the maximum size to be refused", err)
	}
	if errors.Is(err, ErrNoBackend) {
		t.Error("the size was refused only because there is no backend")
	}
}

// TestDefaultDestinationIsExportForAnImage pins the two different defaults Java
// reads a null defaultDestination as: EXPORT for renderImage and VIEW for
// renderPageToGraphics.
func TestDefaultDestinationIsExportForAnImage(t *testing.T) {
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 100)))

	renderer := NewPDFRenderer(document)
	if _, set := renderer.DefaultDestination(); set {
		t.Error("a new renderer has a default destination, want none")
	}
	renderer.SetDefaultDestination(Print)
	if destination, set := renderer.DefaultDestination(); !set || destination != Print {
		t.Errorf("DefaultDestination() = %v, %t, want Print, true", destination, set)
	}
}

// TestDefaultRenderingHints pins what PDFBox chooses when the caller sets no
// hints: quality and bicubic interpolation with anti-aliasing, unless the
// device is one bit deep, in which case neither would mean anything.
func TestDefaultRenderingHints(t *testing.T) {
	color := DefaultRenderingHints(false)
	if color.Interpolation != Bicubic || !color.AntiAliasing || !color.Quality {
		t.Errorf("DefaultRenderingHints(false) = %+v, want bicubic, anti-aliased, quality", color)
	}
	bitonal := DefaultRenderingHints(true)
	if bitonal.Interpolation != NearestNeighbor || bitonal.AntiAliasing || !bitonal.Quality {
		t.Errorf("DefaultRenderingHints(true) = %+v, want nearest neighbour, no anti-aliasing",
			bitonal)
	}
}

// TestRenderImageStartsFromAFreshSurface pins what Java gets for nothing:
// renderImage makes a BufferedImage and a Graphics2D of its own each time it is
// called, so the transform it works from is always the identity. The port draws
// through a Backend the caller installed and keeps, so it has to say so --
// otherwise the second page is drawn through the first page's flip.
func TestRenderImageStartsFromAFreshSurface(t *testing.T) {
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 200)))

	backend := newRecordingBackend()
	renderer := NewPDFRenderer(document)
	renderer.SetBackend(backend, false)

	if err := renderer.RenderImage(0); err != nil {
		t.Fatal(err)
	}
	first := backend.Rendered().Clone()

	if err := renderer.RenderImage(0); err != nil {
		t.Fatal(err)
	}
	if second := backend.Rendered(); !second.Equals(first) {
		t.Errorf("second render used %v, want %v -- the transforms accumulated", second, first)
	}
}

// TestRenderImageLeavesTheBackendAsItFoundIt is the other half: the surface the
// caller installed is theirs, and Java never touches it, because it never has
// it. The transform it carried before the render is the one it carries after.
func TestRenderImageLeavesTheBackendAsItFoundIt(t *testing.T) {
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 200)))

	backend := newRecordingBackend()
	installed := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)
	installed.Translate(3, 5)
	backend.SetTransform(installed)
	before := backend.Transform().Clone()

	renderer := NewPDFRenderer(document)
	renderer.SetBackend(backend, false)
	if err := renderer.RenderImage(0); err != nil {
		t.Fatal(err)
	}
	if !backend.Transform().Equals(before) {
		t.Errorf("transform = %v, want %v unchanged after RenderImage",
			backend.Transform(), before)
	}
}

// TestRenderImageIgnoresTheBackendsOwnTransform pins the same thing from the
// other side: Java's fresh Graphics2D starts at the identity whatever the
// caller's surface carries, so a translate on the backend must not move the
// page.
func TestRenderImageIgnoresTheBackendsOwnTransform(t *testing.T) {
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 200)))

	plain := newRecordingBackend()
	renderer := NewPDFRenderer(document)
	renderer.SetBackend(plain, false)
	if err := renderer.RenderImage(0); err != nil {
		t.Fatal(err)
	}
	want := plain.Rendered().Clone()

	translated := newRecordingBackend()
	installed := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)
	installed.Translate(3, 5)
	translated.SetTransform(installed)
	renderer.SetBackend(translated, false)
	if err := renderer.RenderImage(0); err != nil {
		t.Fatal(err)
	}
	if got := translated.Rendered(); !got.Equals(want) {
		t.Errorf("render used %v, want %v -- the backend's own transform leaked in", got, want)
	}
}
