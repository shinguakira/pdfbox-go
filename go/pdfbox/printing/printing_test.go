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
// cut down to what these tests read: the state a printable might leak, the last
// stroke, and the transform it was drawn through.
type recordingBackend struct {
	// log is shared with every copy Create makes, the way a Graphics2D copy
	// draws to the same destination as the one it came from.
	log *recordingLog

	transform     *geom.AffineTransform
	clip          *geom.Area
	paint         rendering.Paint
	stroke        *rendering.Stroke
	blendMode     *blend.BlendMode
	alphaConstant float64
	antiAliasing  bool
	interpolation rendering.Interpolation
}

// recordingLog is what every copy of a backend writes into.
type recordingLog struct {
	lastDraw          string
	lastDrawTransform *geom.AffineTransform
	disposals         int
	transforms        []*geom.AffineTransform
}

var _ rendering.Backend = (*recordingBackend)(nil)

func newRecordingBackend() *recordingBackend {
	return &recordingBackend{
		log:       &recordingLog{},
		transform: geom.NewAffineTransform(1, 0, 0, 1, 0, 0),
	}
}

// LastDraw returns the last stroked shape, or the empty string where nothing
// was stroked.
func (b *recordingBackend) LastDraw() string { return b.log.lastDraw }

// LastDrawTransform returns the transform the last stroke was drawn through.
func (b *recordingBackend) LastDrawTransform() *geom.AffineTransform {
	return b.log.lastDrawTransform
}

// Disposals returns how many copies of this backend were disposed of.
func (b *recordingBackend) Disposals() int { return b.log.disposals }

func (b *recordingBackend) Create() rendering.Backend {
	copied := *b
	copied.transform = b.transform.Clone()
	return &copied
}

func (b *recordingBackend) Dispose() { b.log.disposals++ }

func (b *recordingBackend) Transform() *geom.AffineTransform { return b.transform }

func (b *recordingBackend) SetTransform(at *geom.AffineTransform) {
	b.transform = at
	b.log.transforms = append(b.log.transforms, at.Clone())
}

// Rendered returns the last transform installed on this backend or any copy of
// it, which for a print without a page border is the one the page was drawn
// through.
func (b *recordingBackend) Rendered() *geom.AffineTransform {
	if len(b.log.transforms) == 0 {
		return b.transform
	}
	return b.log.transforms[len(b.log.transforms)-1]
}

func (b *recordingBackend) Clip() *geom.Area { return b.clip }

func (b *recordingBackend) SetClip(clip *geom.Area) { b.clip = clip }

func (b *recordingBackend) Paint() rendering.Paint { return b.paint }

func (b *recordingBackend) SetPaint(paint rendering.Paint) { b.paint = paint }

func (b *recordingBackend) Stroke() *rendering.Stroke { return b.stroke }

func (b *recordingBackend) SetStroke(stroke *rendering.Stroke) { b.stroke = stroke }

func (b *recordingBackend) BlendMode() *blend.BlendMode { return b.blendMode }

func (b *recordingBackend) AlphaConstant() float64 { return b.alphaConstant }

func (b *recordingBackend) SetComposite(blendMode *blend.BlendMode, alphaConstant float64) {
	b.blendMode = blendMode
	b.alphaConstant = alphaConstant
}

func (b *recordingBackend) AntiAliasing() bool { return b.antiAliasing }

func (b *recordingBackend) SetAntiAliasing(on bool) { b.antiAliasing = on }

func (b *recordingBackend) Interpolation() rendering.Interpolation { return b.interpolation }

func (b *recordingBackend) SetInterpolation(interpolation rendering.Interpolation) {
	b.interpolation = interpolation
}

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
	b.log.lastDraw = fmt.Sprintf("[%.2f %.2f %.2f %.2f] paint=%s w=%.3f",
		bounds.X, bounds.Y, bounds.Width, bounds.Height, paint, width)
	b.log.lastDrawTransform = b.transform.Clone()
	return nil
}

// TestPrintLeavesEveryPieceOfStateAlone is what Java gets from
// `graphics.create()`: the printable draws on a copy and disposes of it, so
// nothing it did — not the transform, not the clip, not the paint, the stroke,
// the composite or the rendering hints — reaches the surface the print system
// handed it.
func TestPrintLeavesEveryPieceOfStateAlone(t *testing.T) {
	document := documentOfSize(imageWidth, imageHeight)
	// showPageBorder guarantees a paint, a stroke and a clip are set
	printable := NewPDFPrintableRasterized(document, ActualSize, true, RasterizeOff)

	backend := newRecordingBackend()
	at := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)
	at.Translate(7.0, 11.0)
	at.Scale(1.3, 1.3)
	backend.SetTransform(at)
	backend.SetClip(geom.NewAreaOfShape(geom.NewRectangle2D(1, 2, 3, 4)))
	backend.SetPaint(rendering.ColorPaint{Red: 1, Alpha: 1})
	backend.SetStroke(&rendering.Stroke{LineWidth: 3.7})
	backend.SetComposite(blend.Multiply, 0.25)
	backend.SetAntiAliasing(false)
	backend.SetInterpolation(rendering.NearestNeighbor)

	transform := backend.Transform().Clone()
	clip := backend.Clip()
	paint := backend.Paint()
	stroke := backend.Stroke()
	blendMode := backend.BlendMode()
	alpha := backend.AlphaConstant()

	if _, err := printable.Print(backend, pageFormatOf(imageWidth, imageHeight), 0); err != nil {
		t.Fatal(err)
	}

	if !backend.Transform().Equals(transform) {
		t.Errorf("transform = %v, want %v", backend.Transform(), transform)
	}
	if backend.Clip() != clip {
		t.Errorf("clip = %v, want the one that was set", backend.Clip())
	}
	if backend.Paint() != paint {
		t.Errorf("paint = %v, want %v", backend.Paint(), paint)
	}
	if backend.Stroke() != stroke {
		t.Errorf("stroke = %v, want %v", backend.Stroke(), stroke)
	}
	if backend.BlendMode() != blendMode || backend.AlphaConstant() != alpha {
		t.Errorf("composite = %v at %v, want %v at %v",
			backend.BlendMode(), backend.AlphaConstant(), blendMode, alpha)
	}
	if backend.AntiAliasing() {
		t.Error("anti-aliasing was turned on and left on")
	}
	if backend.Interpolation() != rendering.NearestNeighbor {
		t.Errorf("interpolation = %v, want NearestNeighbor", backend.Interpolation())
	}
	if backend.Disposals() == 0 {
		t.Error("the copy Print drew on was never disposed of")
	}
}

// TestPageBorderIsDrawnAroundTheRenderedPage pins where the border goes: Java
// captures the transform for it *after* translating to the imageable area and
// centring the page, so the border frames the page rather than sitting at the
// corner of the paper.
func TestPageBorderIsDrawnAroundTheRenderedPage(t *testing.T) {
	document := documentOfSize(40, 40)
	printable := NewPDFPrintableCentered(document, ActualSize, true, RasterizeOff, true)

	// paper with a margin of its own, so the imageable origin is not (0,0)
	var paper Paper
	paper.SetSize(200, 200)
	paper.SetImageableArea(20, 30, 100, 100)
	format := NewPageFormat()
	format.Paper = paper

	backend := newRecordingBackend()
	if _, err := printable.Print(backend, format, 0); err != nil {
		t.Fatal(err)
	}
	at := backend.LastDrawTransform()
	if at == nil {
		t.Fatal("nothing was stroked, want the page border")
	}
	// 20 in from the left and 30 from the top for the imageable area, then 30
	// more on each axis to centre a 40 point page in a 100 point one
	if at.TranslateX() != 20+30 || at.TranslateY() != 30+30 {
		t.Errorf("border drawn at (%v, %v), want (50, 60) -- the imageable origin "+
			"and the centring are both missing from it",
			at.TranslateX(), at.TranslateY())
	}
}
