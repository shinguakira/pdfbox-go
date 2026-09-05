package printing

// Printing pages at any page size or scaling mode.
//
// Port of org.apache.pdfbox.printing.PDFPrintable, which Java declares final
// and which implements java.awt.print.Printable.

import (
	"errors"
	"fmt"
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

const (
	// RasterizeOff prints the page as vectors, without rasterizing it first.
	RasterizeOff float32 = 0

	// RasterizeDPIAuto rasterizes at the printer's own resolution.
	RasterizeDPIAuto float32 = -1
)

// PrintResult is what Printable.print answers.
//
// Port of the java.awt.print.Printable constants, whose Java values are these.
type PrintResult int

const (
	// PageExists says the page was printed.
	PageExists PrintResult = 0
	// NoSuchPage says there is no such page.
	NoSuchPage PrintResult = 1
)

// ErrRasterizeUnsupported reports a print asked to rasterize the page to a
// bitmap first, which needs a raster backend the port does not have.
//
// Java makes a BufferedImage of the imageable area, renders into it and blits
// it onto the printer's Graphics2D. Everything up to making the image is
// computed here; making it is the drawing slice 9 defers. See
// migration/STATUS.md.
var ErrRasterizeUnsupported = errors.New("printing: rasterizing to a bitmap needs a raster backend")

// PDFPrintable prints pages from a PDF document using any page size or scaling
// mode.
//
// Port of PDFPrintable.
type PDFPrintable struct {
	pageTree       *pdmodel.PDPageTree
	renderer       *rendering.PDFRenderer
	showPageBorder bool
	scaling        Scaling
	dpi            float32
	center         bool

	subsamplingAllowed bool
	renderingHints     *rendering.RenderingHints
}

// NewPDFPrintable returns a printable that shrinks each page to fit.
//
// Port of PDFPrintable(PDDocument).
func NewPDFPrintable(document *pdmodel.PDDocument) *PDFPrintable {
	return NewPDFPrintableScaled(document, ShrinkToFit)
}

// NewPDFPrintableScaled returns a printable with the given page scaling.
//
// Port of PDFPrintable(PDDocument, Scaling).
func NewPDFPrintableScaled(document *pdmodel.PDDocument, scaling Scaling) *PDFPrintable {
	return NewPDFPrintableBordered(document, scaling, false)
}

// NewPDFPrintableBordered returns a printable with the given page scaling and
// with optional page borders shown.
//
// Port of PDFPrintable(PDDocument, Scaling, boolean).
func NewPDFPrintableBordered(document *pdmodel.PDDocument, scaling Scaling,
	showPageBorder bool) *PDFPrintable {
	return NewPDFPrintableRasterized(document, scaling, showPageBorder, RasterizeOff)
}

// NewPDFPrintableRasterized returns a printable that rasterizes each page at
// the given DPI before printing it, or prints it as vectors where the dpi is
// RasterizeOff. RasterizeDPIAuto uses the printer's own resolution.
//
// Port of PDFPrintable(PDDocument, Scaling, boolean, float).
func NewPDFPrintableRasterized(document *pdmodel.PDDocument, scaling Scaling,
	showPageBorder bool, dpi float32) *PDFPrintable {
	return NewPDFPrintableCentered(document, scaling, showPageBorder, dpi, true)
}

// NewPDFPrintableCentered returns a printable, center saying whether each page
// is centred in the imageable area.
//
// Port of PDFPrintable(PDDocument, Scaling, boolean, float, boolean).
func NewPDFPrintableCentered(document *pdmodel.PDDocument, scaling Scaling,
	showPageBorder bool, dpi float32, center bool) *PDFPrintable {
	return NewPDFPrintableWithRenderer(document, scaling, showPageBorder, dpi, center,
		rendering.NewPDFRenderer(document))
}

// NewPDFPrintableWithRenderer returns a printable that draws through the given
// renderer.
//
// Port of PDFPrintable(PDDocument, Scaling, boolean, float, boolean, PDFRenderer).
func NewPDFPrintableWithRenderer(document *pdmodel.PDDocument, scaling Scaling,
	showPageBorder bool, dpi float32, center bool,
	renderer *rendering.PDFRenderer) *PDFPrintable {
	return &PDFPrintable{
		pageTree:       document.Pages(),
		renderer:       renderer,
		scaling:        scaling,
		showPageBorder: showPageBorder,
		dpi:            dpi,
		center:         center,
	}
}

// IsSubsamplingAllowed reports whether the renderer may subsample images before
// drawing them.
func (p *PDFPrintable) IsSubsamplingAllowed() bool { return p.subsamplingAllowed }

// SetSubsamplingAllowed says whether the renderer may subsample images before
// drawing them.
func (p *PDFPrintable) SetSubsamplingAllowed(subsamplingAllowed bool) {
	p.subsamplingAllowed = subsamplingAllowed
}

// RenderingHints returns the rendering hints, the second result being false
// where none were set and PDFBox decides at runtime.
func (p *PDFPrintable) RenderingHints() (rendering.RenderingHints, bool) {
	if p.renderingHints == nil {
		return rendering.RenderingHints{}, false
	}
	return *p.renderingHints, true
}

// SetRenderingHints sets the rendering hints.
func (p *PDFPrintable) SetRenderingHints(renderingHints rendering.RenderingHints) {
	p.renderingHints = &renderingHints
}

// Print draws the given page onto the given backend, within the given page
// format's imageable area.
//
// Port of print(Graphics, PageFormat, int). Java is handed a Graphics it copies
// and disposes of; the port is handed the backend itself and restores the
// transform it found.
func (p *PDFPrintable) Print(backend rendering.Backend, pageFormat PageFormat,
	pageIndex int) (PrintResult, error) {
	if pageIndex < 0 || pageIndex >= p.pageTree.Count() {
		return NoSuchPage, nil
	}
	// work on a private copy so the caller's transform is never mutated
	savedTransform := backend.Transform()
	defer backend.SetTransform(savedTransform)

	// capture the DPI that will be used for rasterizing the image
	// if rasterizing is specified
	rasterDpi := p.dpi
	if rasterDpi == RasterizeDPIAuto {
		transform := backend.Transform()
		rasterDpi = util.NewMatrixFromAffineTransform(transform).ScalingFactorX() * 72.0
		slog.Debug("printing: auto raster dpi", "dpi", rasterDpi, "transform", transform)
	}

	page := p.pageTree.Get(pageIndex)
	cropBox := RotatedCropBox(page)

	// the imageable area is the area within the page margins
	imageableWidth := pageFormat.ImageableWidth()
	imageableHeight := pageFormat.ImageableHeight()

	scale := 1.0
	if p.scaling != ActualSize {
		// scale to fit
		scaleX := imageableWidth / float64(cropBox.Width())
		scaleY := imageableHeight / float64(cropBox.Height())
		scale = math.Min(scaleX, scaleY)

		// only shrink to fit when enabled
		if scale > 1 && p.scaling == ShrinkToFit {
			scale = 1
		}

		// only stretch to fit when enabled
		if scale < 1 && p.scaling == StretchToFit {
			scale = 1
		}
	}

	// set the graphics origin to the origin of the imageable area (i.e the margins)
	at := backend.Transform().Clone()
	at.Translate(pageFormat.ImageableX(), pageFormat.ImageableY())

	// center on page
	if p.center {
		dx := (imageableWidth - float64(cropBox.Width())*scale) / 2
		dy := (imageableHeight - float64(cropBox.Height())*scale) / 2
		if dx >= 0 && dy >= 0 {
			at.Translate(dx, dy)
		} else {
			// PDFBOX-3117 and
			// https://lists.apache.org/thread/12s9tc93ofgmjfq1dpqfps9p725l0wwr
			slog.Warn("printing: centering disabled because of negative translation value",
				"dx", dx, "dy", dy)
		}
	}
	backend.SetTransform(at)

	// rasterize to bitmap (optional)
	if rasterDpi > 0 {
		dpiScale := rasterDpi / 72
		return NoSuchPage, fmt.Errorf("%w: page %d would be rasterized at %v DPI, scale %v",
			ErrRasterizeUnsupported, pageIndex, rasterDpi, dpiScale)
	}

	// draw to graphics using PDFRender
	p.renderer.SetSubsamplingAllowed(p.subsamplingAllowed)
	if p.renderingHints != nil {
		p.renderer.SetRenderingHints(*p.renderingHints)
	}
	err := p.renderer.RenderPageToBackend(pageIndex, backend,
		float32(scale), float32(scale), rendering.Print)
	if err != nil {
		return NoSuchPage, err
	}

	// draw crop box on the printer graphics (always, whether rasterizing or not).
	// Drawing after the blit avoids losing the thin stroke during raster scale-down.
	if p.showPageBorder {
		if err := p.drawPageBorder(backend, savedTransform, cropBox,
			imageableWidth, imageableHeight, scale); err != nil {
			return NoSuchPage, err
		}
	}
	return PageExists, nil
}

// drawPageBorder outlines the crop box in grey, which is the tail of print.
func (p *PDFPrintable) drawPageBorder(backend rendering.Backend,
	printerBorderTransform *geom.AffineTransform, cropBox *common.PDRectangle,
	imageableWidth, imageableHeight, borderScale float64) error {
	backend.SetTransform(printerBorderTransform)
	backend.SetClip(geom.NewAreaOfShape(geom.NewRectangle2D(0, 0, math.Trunc(imageableWidth), math.Trunc(imageableHeight))))
	bordered := backend.Transform().Clone()
	bordered.Scale(borderScale, borderScale)
	backend.SetTransform(bordered)
	backend.SetPaint(rendering.ColorPaint{Red: 0.5019608, Green: 0.5019608, Blue: 0.5019608, Alpha: 1})
	backend.SetStroke(&rendering.Stroke{LineWidth: 0.5, MiterLimit: 10})
	return backend.Draw(geom.NewRectangle2D(0, 0, float64(int(cropBox.Width())), float64(int(cropBox.Height()))))
}

// RotatedCropBox returns the crop box of the given page with its rotation
// applied.
//
// Port of the package-private static getRotatedCropBox.
func RotatedCropBox(page *pdmodel.PDPage) *common.PDRectangle {
	cropBox := page.CropBox()
	rotationAngle := page.Rotation()
	if rotationAngle == 90 || rotationAngle == 270 {
		return common.NewPDRectangleOf(cropBox.LowerLeftY(), cropBox.LowerLeftX(),
			cropBox.Height(), cropBox.Width())
	}
	return cropBox
}

// RotatedMediaBox returns the media box of the given page with its rotation
// applied.
//
// Port of the package-private static getRotatedMediaBox.
func RotatedMediaBox(page *pdmodel.PDPage) *common.PDRectangle {
	mediaBox := page.MediaBox()
	rotationAngle := page.Rotation()
	if rotationAngle == 90 || rotationAngle == 270 {
		return common.NewPDRectangleOf(mediaBox.LowerLeftY(), mediaBox.LowerLeftX(),
			mediaBox.Height(), mediaBox.Width())
	}
	return mediaBox
}
