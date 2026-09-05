package rendering

// The entry point: render a page.
//
// Port of org.apache.pdfbox.rendering.PDFRenderer.
//
// Java renders into a java.awt.image.BufferedImage it makes itself. The port
// has none, so the two renderImage families work out everything that does not
// need pixels -- which page, how big, which image type, what transform, which
// page drawer -- and then hand the drawing to a Backend. With no Backend
// installed they answer ErrNoBackend. renderPageToGraphics, which in Java takes
// the drawing surface as an argument, is RenderPageToBackend here and is the
// one that always works.

import (
	"fmt"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/optionalcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// maxInt32 is java.lang.Integer.MAX_VALUE, which caps the pixel count of an
// image Java can make.
const maxInt32 = 1<<31 - 1

// PDFRenderer renders a PDF document.
//
// Port of PDFRenderer. It may be embedded in order to perform custom rendering,
// which is what Java's "may be overridden" means; CreatePageDrawer is the hook.
type PDFRenderer struct {
	document *pdmodel.PDDocument

	// TODO keep rendering state such as caches here

	// annotationFilter is the default annotations filter, which returns all
	// annotations.
	annotationFilter                      annotation.AnnotationFilter
	subsamplingAllowed                    bool
	defaultDestination                    RenderDestination
	hasDefaultDestination                 bool
	renderingHints                        *RenderingHints
	imageDownscalingOptimizationThreshold float32

	pageTree *pdmodel.PDPageTree

	// backend is what the two renderImage families draw through. Java makes a
	// BufferedImage instead; see the file comment.
	backend Backend

	// isBitonal says the backend draws to a one-bit device, which Java asks the
	// Graphics2D's display for.
	isBitonal bool
}

// NewPDFRenderer returns a renderer over the given document.
//
// Port of the PDFRenderer(PDDocument) constructor. Java's defaultDestination
// starts null, which the two renderImage families read as EXPORT and
// renderPageToGraphics as VIEW; the port keeps that "not set" state in a flag,
// since a Go enum has no null.
func NewPDFRenderer(document *pdmodel.PDDocument) *PDFRenderer {
	return &PDFRenderer{
		document:                              document,
		pageTree:                              document.Pages(),
		annotationFilter:                      func(annotation.PDAnnotation) bool { return true },
		imageDownscalingOptimizationThreshold: 0.5,
	}
}

// Document returns the document being rendered.
//
// Java's field is protected, which a subclass in the same tree reads; the port
// exports the reader.
func (r *PDFRenderer) Document() *pdmodel.PDDocument { return r.document }

// AnnotationsFilter returns the annotation filter.
func (r *PDFRenderer) AnnotationsFilter() annotation.AnnotationFilter { return r.annotationFilter }

// SetAnnotationsFilter sets the annotation filter, so that only the annotations
// it accepts are rendered.
func (r *PDFRenderer) SetAnnotationsFilter(filter annotation.AnnotationFilter) {
	r.annotationFilter = filter
}

// IsSubsamplingAllowed reports whether the renderer may subsample images before
// drawing them, according to the image dimensions and the requested scale.
//
// Subsampling may be faster and less memory-intensive in some cases, but it may
// also lead to loss of quality, especially in images with high spatial
// frequency.
func (r *PDFRenderer) IsSubsamplingAllowed() bool { return r.subsamplingAllowed }

// SetSubsamplingAllowed says whether the renderer may subsample images before
// drawing them.
func (r *PDFRenderer) SetSubsamplingAllowed(subsamplingAllowed bool) {
	r.subsamplingAllowed = subsamplingAllowed
}

// DefaultDestination returns the destination pages are rendered for, the second
// result being false where none was set.
func (r *PDFRenderer) DefaultDestination() (RenderDestination, bool) {
	return r.defaultDestination, r.hasDefaultDestination
}

// SetDefaultDestination sets the destination pages are rendered for.
func (r *PDFRenderer) SetDefaultDestination(defaultDestination RenderDestination) {
	r.defaultDestination = defaultDestination
	r.hasDefaultDestination = true
}

// RenderingHints returns the rendering hints, the second result being false
// where none were set and PDFBox decides at runtime.
func (r *PDFRenderer) RenderingHints() (RenderingHints, bool) {
	if r.renderingHints == nil {
		return RenderingHints{}, false
	}
	return *r.renderingHints, true
}

// SetRenderingHints sets the rendering hints. Use this to influence rendering
// quality and speed. If you don't set them yourself, PDFBox will decide at
// runtime depending on the destination.
func (r *PDFRenderer) SetRenderingHints(renderingHints RenderingHints) {
	r.renderingHints = &renderingHints
}

// ImageDownscalingOptimizationThreshold returns the scale below which a slower,
// higher-quality downscale is used.
func (r *PDFRenderer) ImageDownscalingOptimizationThreshold() float32 {
	return r.imageDownscalingOptimizationThreshold
}

// SetImageDownscalingOptimizationThreshold sets the image downscaling
// optimization threshold. This must be a value between 0 and 1. When rendering
// downscaled images and rendering hints are set to bicubic+quality and the
// scaling is smaller than the threshold, a more quality-optimized but slower
// method will be used. The default is 0.5 which is a good compromise.
func (r *PDFRenderer) SetImageDownscalingOptimizationThreshold(threshold float32) {
	r.imageDownscalingOptimizationThreshold = threshold
}

// Backend returns the raster backend pages are drawn through, or nil.
//
// Java has no such thing: it draws onto a Graphics2D it makes from a
// BufferedImage. See the file comment.
func (r *PDFRenderer) Backend() Backend { return r.backend }

// SetBackend installs the raster backend the two renderImage families draw
// through, isBitonal saying whether it draws to a one-bit device.
func (r *PDFRenderer) SetBackend(backend Backend, isBitonal bool) {
	r.backend = backend
	r.isBitonal = isBitonal
}

// RenderImage renders the given page at 72 DPI.
//
// Port of renderImage(int).
func (r *PDFRenderer) RenderImage(pageIndex int) error {
	return r.RenderImageScaled(pageIndex, 1)
}

// RenderImageScaled renders the given page at the given scale. A scale of 1
// renders at 72 DPI.
//
// Port of renderImage(int, float).
func (r *PDFRenderer) RenderImageScaled(pageIndex int, scale float32) error {
	return r.RenderImageOfType(pageIndex, scale, RGB)
}

// RenderImageWithDPI renders the given page at the given DPI.
//
// Port of renderImageWithDPI(int, float).
func (r *PDFRenderer) RenderImageWithDPI(pageIndex int, dpi float32) error {
	return r.RenderImageOfType(pageIndex, dpi/72, RGB)
}

// RenderImageWithDPIOfType renders the given page at the given DPI and type.
//
// Port of renderImageWithDPI(int, float, ImageType).
func (r *PDFRenderer) RenderImageWithDPIOfType(pageIndex int, dpi float32,
	imageType ImageType) error {
	return r.RenderImageOfType(pageIndex, dpi/72, imageType)
}

// RenderImageOfType renders the given page at the given scale and type, for the
// default destination.
//
// Port of renderImage(int, float, ImageType).
func (r *PDFRenderer) RenderImageOfType(pageIndex int, scale float32, imageType ImageType) error {
	destination := Export
	if r.hasDefaultDestination {
		destination = r.defaultDestination
	}
	return r.RenderImageTo(pageIndex, scale, imageType, destination)
}

// RenderImageTo renders the given page at the given scale and type, for the
// given destination.
//
// Port of renderImage(int, float, ImageType, RenderDestination), minus the
// BufferedImage it makes and returns: everything up to the drawing is here, and
// the drawing goes to the installed Backend.
func (r *PDFRenderer) RenderImageTo(pageIndex int, scale float32, imageType ImageType,
	destination RenderDestination) error {
	page := r.pageTree.Get(pageIndex)
	cropBox := page.CropBox()
	widthPt := cropBox.Width()
	heightPt := cropBox.Height()

	// PDFBOX-4306 avoid single blank pixel line on the right or on the bottom
	widthPx := int(math.Max(math.Floor(float64(widthPt*scale)), 1))
	heightPx := int(math.Max(math.Floor(float64(heightPt*scale)), 1))

	// PDFBOX-4518 the maximum size (w*h) of a buffered image is limited to Integer.MAX_VALUE
	if int64(widthPx)*int64(heightPx) > maxInt32 {
		return fmt.Errorf("Maximum size of image exceeded (w * h * scale ^ 2) = %v * %v * %v ^ 2 > %d",
			widthPt, heightPt, scale, maxInt32)
	}

	rotationAngle := page.Rotation()
	surfaceType := imageType
	if imageType != ARGB && r.hasBlendModeOnPage(page) {
		// PDFBOX-4095: if the PDF has blending on the top level, draw on transparent background
		// Inspired from PDF.js: if a PDF page uses any blend modes other than Normal,
		// PDF.js renders everything on a fully transparent RGBA canvas.
		// Finally when the page has been rendered, PDF.js draws the RGBA canvas on a white canvas.
		surfaceType = ARGB
	}

	// swap width and height
	surfaceWidth, surfaceHeight := widthPx, heightPx
	if rotationAngle == 90 || rotationAngle == 270 {
		surfaceWidth, surfaceHeight = heightPx, widthPx
	}

	if r.backend == nil {
		return fmt.Errorf("%w: page %d would be %dx%d %v", ErrNoBackend,
			pageIndex, surfaceWidth, surfaceHeight, surfaceType)
	}

	// use a transparent background if the image type supports alpha
	r.backend.SetTransform(transformOfPage(rotationAngle, cropBox, scale, scale))
	return r.drawPage(page, destination, cropBox)
}

// RenderPageToBackend renders the given page onto the given backend.
//
// Port of renderPageToGraphics(int, Graphics2D, float, float, RenderDestination)
// and the three that default its arguments, which Java overloads; Go has no
// overloading, and a caller that wants the defaults passes them.
//
// Java's known problems with this method are Java2D's, and do not carry over:
// the transparency bug of PDFBOX-4581 is JDK-6689349, and the clipping one of
// PDFBOX-4583 is Graphics2D's.
func (r *PDFRenderer) RenderPageToBackend(pageIndex int, backend Backend,
	scaleX, scaleY float32, destination RenderDestination) error {
	page := r.pageTree.Get(pageIndex)
	// TODO need width/height calculations? should these be in PageDrawer?
	cropBox := page.CropBox()
	backend.SetTransform(transformOfPage(page.Rotation(), cropBox, scaleX, scaleY))

	saved := r.backend
	r.backend = backend
	defer func() { r.backend = saved }()
	return r.drawPage(page, destination, cropBox)
}

// drawPage is the tail both entry points share: build the parameters, make the
// drawer, draw.
func (r *PDFRenderer) drawPage(page *pdmodel.PDPage, destination RenderDestination,
	cropBox *common.PDRectangle) error {
	// the end-user may provide a custom PageDrawer
	actualRenderingHints := DefaultRenderingHints(r.isBitonal)
	if r.renderingHints != nil {
		actualRenderingHints = *r.renderingHints
	}
	parameters := newPageDrawerParameters(r, page, r.subsamplingAllowed, destination,
		actualRenderingHints, r.imageDownscalingOptimizationThreshold)
	drawer, err := r.CreatePageDrawer(parameters)
	if err != nil {
		return err
	}
	return drawer.DrawPage(r.backend, cropBox)
}

// CreatePageDrawer returns a new page drawer for the given parameters. An
// embedder replaces it, which is what Java's "may be overridden" means.
//
// Port of the protected createPageDrawer.
func (r *PDFRenderer) CreatePageDrawer(parameters PageDrawerParameters) (*PageDrawer, error) {
	pageDrawer, err := NewPageDrawer(parameters)
	if err != nil {
		return nil, err
	}
	pageDrawer.SetAnnotationFilter(r.annotationFilter)
	return pageDrawer, nil
}

// IsGroupEnabled reports whether the given optional content group is enabled.
func (r *PDFRenderer) IsGroupEnabled(group *optionalcontent.PDOptionalContentGroup) bool {
	ocProperties := r.document.DocumentCatalog().OCProperties()
	return ocProperties == nil || ocProperties.IsGroupEnabled(group)
}

// transformOfPage returns the scale, rotate and translate a page is drawn
// through.
//
// Port of the private transform(Graphics2D, int, PDRectangle, float, float),
// which mutates the Graphics2D; the port builds the transform and lets the
// caller install it.
func transformOfPage(rotationAngle int, cropBox *common.PDRectangle,
	scaleX, scaleY float32) *geom.AffineTransform {
	at := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)
	at.Scale(float64(scaleX), float64(scaleY))
	// TODO should we be passing the scale to PageDrawer rather than messing with Graphics?
	if rotationAngle != 0 {
		var translateX, translateY float32
		switch rotationAngle {
		case 90:
			translateX = cropBox.Height()
		case 270:
			translateY = cropBox.Width()
		case 180:
			translateX = cropBox.Width()
			translateY = cropBox.Height()
		}
		at.Translate(float64(translateX), float64(translateY))
		at.Rotate(float64(rotationAngle) * math.Pi / 180)
	}
	return at
}

// hasBlendModeOnPage reports whether the page's own resources set any blend
// mode other than Normal.
//
// Port of the private hasBlendMode(PDPage).
func (r *PDFRenderer) hasBlendModeOnPage(page *pdmodel.PDPage) bool {
	// check the current resources for blend modes
	resources := page.Resources()
	if resources == nil {
		return false
	}
	for _, name := range resources.ExtGStateNames() {
		extGState := resources.GetExtGState(name)
		if extGState != nil {
			// extGState null can happen if key exists but no value
			// see PDFBOX-3950-23EGDHXSBBYQLKYOKGZUOVYVNE675PRD.pdf
			if extGState.BlendMode() != blend.Normal {
				return true
			}
		}
	}
	return false
}
