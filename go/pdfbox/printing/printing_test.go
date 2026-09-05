package printing

// Port of org.apache.pdfbox.printing.TestPDFPrintable.
//
// Of its five cases, two read back pixels of a rendered page to see whether the
// border came out grey; with no rasteriser there is no page to read and, per
// slice 9's A5 decision, pixels are not what this port compares against. What
// they were asking -- that the border is drawn, in grey, on the printer's own
// surface rather than on the rasterised copy -- is asked here of the recorded
// calls instead. The other three port as they stand.

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

const (
	imageWidth  = 100
	imageHeight = 100
)

// pageFormatOf is Java's createPageFormat: a sheet exactly the given size whose
// whole area is printable.
func pageFormatOf(width, height float64) PageFormat {
	var paper Paper
	paper.SetSize(width, height)
	paper.SetImageableArea(0, 0, width, height)
	format := NewPageFormat()
	format.Paper = paper
	return format
}

// documentOfSize returns a one-page document of the given page size.
func documentOfSize(width, height float32) *pdmodel.PDDocument {
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(width, height)))
	return document
}

// TestPrintReturnsNoSuchPageForInvalidIndex is Java's test of the same name.
func TestPrintReturnsNoSuchPageForInvalidIndex(t *testing.T) {
	document := documentOfSize(imageWidth, imageHeight)
	printable := NewPDFPrintable(document)
	backend := newRecordingBackend()
	pf := pageFormatOf(imageWidth, imageHeight)

	for _, pageIndex := range []int{-1, 1} {
		result, err := printable.Print(backend, pf, pageIndex)
		if err != nil {
			t.Fatal(err)
		}
		if result != NoSuchPage {
			t.Errorf("Print(%d) = %v, want NoSuchPage", pageIndex, result)
		}
	}
	result, err := printable.Print(backend, pf, 0)
	if err != nil {
		t.Fatal(err)
	}
	if result != PageExists {
		t.Errorf("Print(0) = %v, want PageExists", result)
	}
}

// TestPrinterStateIsUnchangedAfterPrint is Java's
// testPrinterGraphicsStateIsUnchangedAfterPrint: the translate and scale print
// does inside must not leak onto the surface it was handed.
func TestPrinterStateIsUnchangedAfterPrint(t *testing.T) {
	document := documentOfSize(imageWidth, imageHeight)
	printable := NewPDFPrintableRasterized(document, ActualSize, true, RasterizeOff)

	backend := newRecordingBackend()
	// set a distinctive transform so we can detect leaks from internal
	// translate/scale calls
	at := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)
	at.Translate(7.0, 11.0)
	at.Scale(1.3, 1.3)
	backend.SetTransform(at)
	originalTransform := backend.Transform().Clone()

	result, err := printable.Print(backend, pageFormatOf(imageWidth, imageHeight), 0)
	if err != nil {
		t.Fatal(err)
	}
	if result != PageExists {
		t.Fatalf("Print = %v, want PageExists", result)
	}
	if !backend.Transform().Equals(originalTransform) {
		t.Errorf("transform = %v, want %v unchanged after Print",
			backend.Transform(), originalTransform)
	}
}

// TestPageBorderIsDrawnInGrey is what the two pixel tests were asking: the
// border is stroked, thinly, in Color.GRAY, and on the printer's own surface.
func TestPageBorderIsDrawnInGrey(t *testing.T) {
	document := documentOfSize(imageWidth, imageHeight)
	printable := NewPDFPrintableRasterized(document, ActualSize, true, RasterizeOff)

	backend := newRecordingBackend()
	if _, err := printable.Print(backend, pageFormatOf(imageWidth, imageHeight), 0); err != nil {
		t.Fatal(err)
	}
	border := backend.LastDraw()
	if border == "" {
		t.Fatal("nothing was stroked, want the page border")
	}
	// Color.GRAY is (128, 128, 128), which is 128/255 in each channel
	if !strings.Contains(border, "color(0.502 0.502 0.502 1.000)") {
		t.Errorf("border = %q, want it stroked in grey", border)
	}
	if !strings.Contains(border, "w=0.500") {
		t.Errorf("border = %q, want a line width of 0.5", border)
	}
	if !strings.Contains(border, "[0.00 0.00 100.00 100.00]") {
		t.Errorf("border = %q, want it around the crop box", border)
	}
}

// TestNoBorderWhenNotAskedFor pins the other half: showPageBorder off draws
// nothing extra.
func TestNoBorderWhenNotAskedFor(t *testing.T) {
	document := documentOfSize(imageWidth, imageHeight)
	printable := NewPDFPrintableRasterized(document, ActualSize, false, RasterizeOff)

	backend := newRecordingBackend()
	if _, err := printable.Print(backend, pageFormatOf(imageWidth, imageHeight), 0); err != nil {
		t.Fatal(err)
	}
	if border := backend.LastDraw(); border != "" {
		t.Errorf("something was stroked (%q), want nothing without a page border", border)
	}
}

// TestRasterizingSaysItCannot is what the B0 decision costs printing: Java
// renders the page into a BufferedImage at the requested DPI and blits it, and
// there is nothing to make that image with.
func TestRasterizingSaysItCannot(t *testing.T) {
	document := documentOfSize(imageWidth, imageHeight)
	printable := NewPDFPrintableRasterized(document, ActualSize, true, 150)

	backend := newRecordingBackend()
	_, err := printable.Print(backend, pageFormatOf(imageWidth, imageHeight), 0)
	if !errors.Is(err, ErrRasterizeUnsupported) {
		t.Errorf("Print = %v, want ErrRasterizeUnsupported", err)
	}
}

// TestScalingChoosesTheScale pins the four scaling modes against a page half
// the size of the paper: actual size never scales, shrink only shrinks, stretch
// only stretches, and scale does both.
func TestScalingChoosesTheScale(t *testing.T) {
	for _, test := range []struct {
		scaling   Scaling
		pageSize  float32
		wantScale float64
	}{
		{ActualSize, 50, 1},
		{ActualSize, 200, 1},
		{ShrinkToFit, 50, 1},    // would be 2, but shrink only shrinks
		{ShrinkToFit, 200, 0.5}, /* shrinks */
		{StretchToFit, 50, 2},   // stretches
		{StretchToFit, 200, 1},  // would be 0.5, but stretch only stretches
		{ScaleToFit, 50, 2},
		{ScaleToFit, 200, 0.5},
	} {
		document := documentOfSize(test.pageSize, test.pageSize)
		printable := NewPDFPrintableCentered(document, test.scaling, false, RasterizeOff, false)

		backend := newRecordingBackend()
		if _, err := printable.Print(backend, pageFormatOf(100, 100), 0); err != nil {
			t.Fatal(err)
		}
		// the renderer scales the device transform by the printable's scale
		if got := backend.Rendered().ScaleX(); got != test.wantScale {
			t.Errorf("%v of a %v page: scale = %v, want %v",
				test.scaling, test.pageSize, got, test.wantScale)
		}
	}
}

// TestCenteringTranslatesByHalfTheSlack pins the centering arithmetic: a 40
// point page on 100 point paper is pushed 30 points in from each edge.
func TestCenteringTranslatesByHalfTheSlack(t *testing.T) {
	document := documentOfSize(40, 40)
	printable := NewPDFPrintableCentered(document, ActualSize, false, RasterizeOff, true)

	backend := newRecordingBackend()
	if _, err := printable.Print(backend, pageFormatOf(100, 100), 0); err != nil {
		t.Fatal(err)
	}
	at := backend.Rendered()
	if at.TranslateX() != 30 {
		t.Errorf("translate x = %v, want 30", at.TranslateX())
	}
	// y is flipped by the drawer, so the page's top edge is 30 + its height
	if at.TranslateY() != 30+40 {
		t.Errorf("translate y = %v, want 70", at.TranslateY())
	}
}

// TestRotatedCropBoxSwapsTheSides pins getRotatedCropBox and
// getRotatedMediaBox: a quarter turn swaps width and height, and the lower left
// corner with it, while a half turn leaves both alone.
func TestRotatedCropBoxSwapsTheSides(t *testing.T) {
	for _, test := range []struct {
		rotation                  int
		wantWidth, wantHeight     float32
		wantLowerLeftX, lowerLeft float32
	}{
		{0, 100, 200, 0, 0},
		{90, 200, 100, 0, 0},
		{180, 100, 200, 0, 0},
		{270, 200, 100, 0, 0},
	} {
		page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 200))
		page.SetRotation(test.rotation)

		cropBox := RotatedCropBox(page)
		if cropBox.Width() != test.wantWidth || cropBox.Height() != test.wantHeight {
			t.Errorf("rotation %d: crop box = %vx%v, want %vx%v", test.rotation,
				cropBox.Width(), cropBox.Height(), test.wantWidth, test.wantHeight)
		}
		mediaBox := RotatedMediaBox(page)
		if mediaBox.Width() != test.wantWidth || mediaBox.Height() != test.wantHeight {
			t.Errorf("rotation %d: media box = %vx%v, want %vx%v", test.rotation,
				mediaBox.Width(), mediaBox.Height(), test.wantWidth, test.wantHeight)
		}
	}
}

// TestPageableNormalisesLandscapeToPortrait pins the PDFBOX-2922 workaround:
// Java's print system crops a landscape sheet as if it were portrait, so a
// landscape page is described as portrait paper flagged landscape.
func TestPageableNormalisesLandscapeToPortrait(t *testing.T) {
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(200, 100)))
	pageable := NewPDFPageable(document)

	format := pageable.PageFormat(0)
	if format.Paper.Width != 100 || format.Paper.Height != 200 {
		t.Errorf("paper = %vx%v, want 100x200 -- the sides swapped",
			format.Paper.Width, format.Paper.Height)
	}
	if format.Orientation != LandscapePage {
		t.Errorf("orientation = %v, want landscape", format.Orientation)
	}
	// and the imageable area swapped with it
	if format.Paper.ImageableWidth != 100 || format.Paper.ImageableHeight != 200 {
		t.Errorf("imageable area = %vx%v, want 100x200",
			format.Paper.ImageableWidth, format.Paper.ImageableHeight)
	}
}

// TestPageableOrientationOverridesAuto pins that a chosen orientation is used
// as it stands, rather than being worked out from the page's aspect ratio.
func TestPageableOrientationOverridesAuto(t *testing.T) {
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 200)))

	for _, test := range []struct {
		orientation Orientation
		want        PageOrientation
	}{
		{Auto, PortraitPage}, // a portrait page
		{Landscape, LandscapePage},
		{ReverseLandscape, ReverseLandscapePage},
		{Portrait, PortraitPage},
	} {
		got := NewPDFPageableOriented(document, test.orientation).PageFormat(0).Orientation
		if got != test.want {
			t.Errorf("%v: orientation = %v, want %v", test.orientation, got, test.want)
		}
	}
}

// TestPageableCountsThePages pins getNumberOfPages, which the print system asks
// before anything else.
func TestPageableCountsThePages(t *testing.T) {
	document := pdmodel.NewPDDocument()
	for i := 0; i < 3; i++ {
		document.AddPage(pdmodel.NewPDPage())
	}
	if got := NewPDFPageable(document).NumberOfPages(); got != 3 {
		t.Errorf("NumberOfPages() = %d, want 3", got)
	}
}

// TestPageablePrintableIsActualSize pins that a pageable prints each page at
// its own size, since the paper it answered with is already that size.
func TestPageablePrintableIsActualSize(t *testing.T) {
	document := documentOfSize(40, 40)
	pageable := NewPDFPageable(document)
	printable := pageable.Printable(0)

	backend := newRecordingBackend()
	if _, err := printable.Print(backend, pageFormatOf(100, 100), 0); err != nil {
		t.Fatal(err)
	}
	if got := backend.Rendered().ScaleX(); got != 1 {
		t.Errorf("scale = %v, want 1 -- a pageable prints at actual size", got)
	}
}

// TestPageablePrintableRefusesAPagePastTheEnd pins the unchecked
// IndexOutOfBoundsException Java throws, which the port panics for.
func TestPageablePrintableRefusesAPagePastTheEnd(t *testing.T) {
	document := documentOfSize(40, 40)
	defer func() {
		if recover() == nil {
			t.Error("Printable(1) returned, want a panic for a page past the end")
		}
	}()
	NewPDFPageable(document).Printable(1)
}

// recordingBackend is rendering's, which this package cannot reach: a test
// file is not part of the package it is compiled into. This is the same thing
// cut down to what these tests read: the transform, and the last stroke.
type recordingBackend struct {
	transform  *geom.AffineTransform
	clip       *geom.Area
	paint      rendering.Paint
	stroke     *rendering.Stroke
	lastDraw   string
	transforms []*geom.AffineTransform
}

var _ rendering.Backend = (*recordingBackend)(nil)

func newRecordingBackend() *recordingBackend {
	return &recordingBackend{transform: geom.NewAffineTransform(1, 0, 0, 1, 0, 0)}
}

// LastDraw returns the last stroked shape, or the empty string where nothing
// was stroked.
func (b *recordingBackend) LastDraw() string { return b.lastDraw }

func (b *recordingBackend) Transform() *geom.AffineTransform { return b.transform }

func (b *recordingBackend) SetTransform(at *geom.AffineTransform) {
	b.transform = at
	b.transforms = append(b.transforms, at.Clone())
}

// Rendered returns the transform that was in force while the page was drawn,
// which is the one installed just before Print put back the caller's own.
func (b *recordingBackend) Rendered() *geom.AffineTransform {
	if len(b.transforms) < 2 {
		return b.transform
	}
	return b.transforms[len(b.transforms)-2]
}

func (b *recordingBackend) Clip() *geom.Area { return b.clip }

func (b *recordingBackend) SetClip(clip *geom.Area) { b.clip = clip }

func (b *recordingBackend) SetPaint(paint rendering.Paint) { b.paint = paint }

func (b *recordingBackend) SetStroke(stroke *rendering.Stroke) { b.stroke = stroke }

func (b *recordingBackend) SetComposite(*blend.BlendMode, float64) {}

func (b *recordingBackend) SetAntiAliasing(bool) {}

func (b *recordingBackend) SetInterpolation(rendering.Interpolation) {}

func (b *recordingBackend) Fill(geom.Shape) error { return nil }

func (b *recordingBackend) DrawImage(image.PDImage, *geom.AffineTransform, int) error {
	return nil
}

func (b *recordingBackend) DrawStencil(image.PDImage, *geom.AffineTransform,
	rendering.Paint) error {
	return nil
}

func (b *recordingBackend) PushGroup(*common.PDRectangle, bool, bool, *color.PDColor) error {
	return nil
}

func (b *recordingBackend) PopGroup() error { return nil }

func (b *recordingBackend) Draw(shape geom.Shape) error {
	bounds := shape.Bounds2D()
	paint := "none"
	if c, isColor := b.paint.(rendering.ColorPaint); isColor {
		paint = fmt.Sprintf("color(%.3f %.3f %.3f %.3f)", c.Red, c.Green, c.Blue, c.Alpha)
	}
	width := float32(0)
	if b.stroke != nil {
		width = b.stroke.LineWidth
	}
	b.lastDraw = fmt.Sprintf("[%.2f %.2f %.2f %.2f] paint=%s w=%.3f",
		bounds.X, bounds.Y, bounds.Width, bounds.Height, paint, width)
	return nil
}
