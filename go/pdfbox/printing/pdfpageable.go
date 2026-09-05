package printing

// Printing a document at its original paper size.
//
// Port of org.apache.pdfbox.printing.PDFPageable, which Java declares final and
// which extends java.awt.print.Book.

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// PDFPageable prints a PDF document using its original paper size.
//
// Port of PDFPageable. Java extends java.awt.print.Book, whose own state --
// the list of (Printable, PageFormat) pairs -- PDFPageable never uses: it
// overrides all three of getNumberOfPages, getPageFormat and getPrintable. So
// the port has no base to embed.
type PDFPageable struct {
	document       *pdmodel.PDDocument
	numberOfPages  int
	showPageBorder bool
	dpi            float32
	center         bool
	orientation    Orientation

	subsamplingAllowed bool
	renderingHints     *rendering.RenderingHints
}

// NewPDFPageable returns a pageable that prints every page as it stands.
//
// Port of PDFPageable(PDDocument).
func NewPDFPageable(document *pdmodel.PDDocument) *PDFPageable {
	return NewPDFPageableCentered(document, Auto, false, 0, true)
}

// NewPDFPageableOriented returns a pageable with the given page orientation.
//
// Port of PDFPageable(PDDocument, Orientation).
func NewPDFPageableOriented(document *pdmodel.PDDocument, orientation Orientation) *PDFPageable {
	return NewPDFPageableCentered(document, orientation, false, 0, true)
}

// NewPDFPageableBordered returns a pageable with the given page orientation and
// with optional page borders shown.
//
// Port of PDFPageable(PDDocument, Orientation, boolean).
func NewPDFPageableBordered(document *pdmodel.PDDocument, orientation Orientation,
	showPageBorder bool) *PDFPageable {
	return NewPDFPageableCentered(document, orientation, showPageBorder, 0, true)
}

// NewPDFPageableRasterized returns a pageable that rasterizes each page at the
// given DPI before printing it, where the dpi is non-zero.
//
// Port of PDFPageable(PDDocument, Orientation, boolean, float).
func NewPDFPageableRasterized(document *pdmodel.PDDocument, orientation Orientation,
	showPageBorder bool, dpi float32) *PDFPageable {
	return NewPDFPageableCentered(document, orientation, showPageBorder, dpi, true)
}

// NewPDFPageableCentered returns a pageable, center saying whether each page is
// centred in the imageable area.
//
// Port of PDFPageable(PDDocument, Orientation, boolean, float, boolean).
func NewPDFPageableCentered(document *pdmodel.PDDocument, orientation Orientation,
	showPageBorder bool, dpi float32, center bool) *PDFPageable {
	return &PDFPageable{
		document:       document,
		orientation:    orientation,
		showPageBorder: showPageBorder,
		dpi:            dpi,
		center:         center,
		numberOfPages:  document.NumberOfPages(),
	}
}

// RenderingHints returns the rendering hints, the second result being false
// where none were set and PDFBox decides at runtime.
func (p *PDFPageable) RenderingHints() (rendering.RenderingHints, bool) {
	if p.renderingHints == nil {
		return rendering.RenderingHints{}, false
	}
	return *p.renderingHints, true
}

// SetRenderingHints sets the rendering hints.
func (p *PDFPageable) SetRenderingHints(renderingHints rendering.RenderingHints) {
	p.renderingHints = &renderingHints
}

// IsSubsamplingAllowed reports whether the renderer may subsample images before
// drawing them.
func (p *PDFPageable) IsSubsamplingAllowed() bool { return p.subsamplingAllowed }

// SetSubsamplingAllowed says whether the renderer may subsample images before
// drawing them.
func (p *PDFPageable) SetSubsamplingAllowed(subsamplingAllowed bool) {
	p.subsamplingAllowed = subsamplingAllowed
}

// NumberOfPages returns how many pages there are to print.
func (p *PDFPageable) NumberOfPages() int { return p.numberOfPages }

// PageFormat returns the actual physical size of the given page. It may not fit
// the local printer.
func (p *PDFPageable) PageFormat(pageIndex int) PageFormat {
	page := p.document.Page(pageIndex)
	mediaBox := RotatedMediaBox(page)
	cropBox := RotatedCropBox(page)

	// Java does not seem to understand landscape paper sizes, i.e. where width > height, it
	// always crops the imageable area as if the page were in portrait. I suspect that this is
	// a JDK bug but it might be by design, see PDFBOX-2922.
	//
	// As a workaround, we normalise all Page(s) to be portrait, then flag them as landscape in
	// the PageFormat.
	var paper Paper
	var isLandscape bool
	if mediaBox.Width() > mediaBox.Height() {
		// rotate
		paper.SetSize(float64(mediaBox.Height()), float64(mediaBox.Width()))
		paper.SetImageableArea(float64(cropBox.LowerLeftY()), float64(cropBox.LowerLeftX()),
			float64(cropBox.Height()), float64(cropBox.Width()))
		isLandscape = true
	} else {
		paper.SetSize(float64(mediaBox.Width()), float64(mediaBox.Height()))
		paper.SetImageableArea(float64(cropBox.LowerLeftX()), float64(cropBox.LowerLeftY()),
			float64(cropBox.Width()), float64(cropBox.Height()))
		isLandscape = false
	}

	format := NewPageFormat()
	format.Paper = paper

	// auto portrait/landscape
	switch p.orientation {
	case Auto:
		if isLandscape {
			format.Orientation = LandscapePage
		} else {
			format.Orientation = PortraitPage
		}
	case Landscape:
		format.Orientation = LandscapePage
	case ReverseLandscape:
		format.Orientation = ReverseLandscapePage
	case Portrait:
		format.Orientation = PortraitPage
	}
	return format
}

// Printable returns the printable for the given page.
//
// Java throws IndexOutOfBoundsException for a page past the end, which is
// unchecked; the port panics for the same reason.
func (p *PDFPageable) Printable(i int) *PDFPrintable {
	if i >= p.numberOfPages {
		panic(fmt.Sprintf("printing: %d >= %d", i, p.numberOfPages))
	}
	printable := NewPDFPrintableCentered(p.document, ActualSize, p.showPageBorder, p.dpi, p.center)
	printable.SetSubsamplingAllowed(p.subsamplingAllowed)
	if p.renderingHints != nil {
		printable.SetRenderingHints(*p.renderingHints)
	}
	return printable
}
