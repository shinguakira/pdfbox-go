package contentstream

// The half of PDFStreamEngine that runs a stream other than the page's own: a
// form XObject, a transparency group, a soft mask and a tiling pattern.
//
// Port of PDFStreamEngine.showForm, showTransparencyGroup, processSoftMask,
// processTransparencyGroup, processTilingPattern and processChildStream. Slice
// 2 left these out because none of the three types existed; slice 9 brings
// them.
//
// showAnnotation, getAppearance and processAnnotation are the fourth kind of
// child stream, and are here too.

import (
	"errors"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// ErrNoCurrentPage reports a child stream processed without a page.
//
// Port of the IllegalStateException the three methods below throw. Java's is
// unchecked, so the port would panic; it is an error here because every caller
// of these is an operator, and an operator's errors already travel back through
// processOperator, which is where a PDF that asks for a form outside a page has
// to be dealt with.
var ErrNoCurrentPage = errors.New("contentstream: no current page, " +
	"call ProcessChildStream(PDContentStream, *pdmodel.PDPage) instead")

// ErrPageAlreadySet reports ProcessChildStream called while a page is being
// processed.
//
// Port of the other IllegalStateException, for the reason above.
var ErrPageAlreadySet = errors.New("contentstream: current page has already " +
	"been set via ProcessPage(*pdmodel.PDPage), call ProcessChildStream instead")

// formContentStream adapts a form XObject to PDContentStream.
//
// PDFormXObject implements PDContentStream in Java. It cannot here: its
// getResources answers a PDResources, which lives in pdmodel, and
// graphics/form cannot import pdmodel. This package can name both.
type formContentStream struct{ *form.PDFormXObject }

var _ PDContentStream = formContentStream{}

// Resources returns the form's resources, or nil where it has none.
func (f formContentStream) Resources() *pdmodel.PDResources {
	resources, isResources := f.PDFormXObject.Resources().(*pdmodel.PDResources)
	if !isResources {
		return nil
	}
	return resources
}

// ShowTransparencyGroup shows a transparency group from the content stream.
func (e *PDFStreamEngine) ShowTransparencyGroup(group *form.PDTransparencyGroup) error {
	return e.ProcessTransparencyGroup(group)
}

// ShowForm shows a form from the content stream.
func (e *PDFStreamEngine) ShowForm(f *form.PDFormXObject) error {
	if e.currentPage == nil {
		return ErrNoCurrentPage
	}
	length, err := f.Stream().Length()
	if err != nil {
		return err
	}
	if length > 0 {
		return e.processStream(formContentStream{f})
	}
	return nil
}

// ProcessSoftMask processes a soft mask transparency group stream.
func (e *PDFStreamEngine) ProcessSoftMask(group *form.PDTransparencyGroup) error {
	e.SaveGraphicsState()
	graphicsState := e.GraphicsState()
	softMaskCTM := graphicsState.SoftMask().InitialTransformationMatrix()
	graphicsState.SetCurrentTransformationMatrix(softMaskCTM)
	graphicsState.SetTextMatrix(util.NewMatrix())
	graphicsState.SetTextLineMatrix(util.NewMatrix())
	graphicsState.SetNonStrokingColorSpace(color.DeviceGray)
	graphicsState.SetNonStrokingColor(color.DeviceGray.InitialColor())
	graphicsState.SetStrokingColorSpace(color.DeviceGray)
	graphicsState.SetStrokingColor(color.DeviceGray.InitialColor())

	err := e.ProcessTransparencyGroup(group)
	e.RestoreGraphicsState()
	return err
}

// ProcessTransparencyGroup processes a transparency group stream.
func (e *PDFStreamEngine) ProcessTransparencyGroup(group *form.PDTransparencyGroup) error {
	if e.currentPage == nil {
		return ErrNoCurrentPage
	}

	contentStream := formContentStream{&group.PDFormXObject}
	parent := e.pushResources(contentStream)
	savedStack := e.SaveGraphicsStack()

	parentMatrix := e.initialMatrix
	graphicsState := e.GraphicsState()

	// the stream's initial matrix includes the parent CTM, e.g. this allows a scaled form
	e.initialMatrix = graphicsState.CurrentTransformationMatrix().Clone()

	// transform the CTM using the stream's matrix
	graphicsState.CurrentTransformationMatrix().Concatenate(group.Matrix())

	// Before execution of the transparency group XObject's content stream,
	// the current blend mode in the graphics state shall be initialized to Normal,
	// the current stroking and nonstroking alpha constants to 1.0, and the current soft mask to None.
	graphicsState.SetBlendMode(blend.Normal)
	graphicsState.SetAlphaConstant(1)
	graphicsState.SetNonStrokeAlphaConstant(1)
	graphicsState.SetSoftMask(nil)

	// clip to bounding box
	e.clipToRect(group.BBox())

	err := e.processStreamOperators(contentStream)

	e.initialMatrix = parentMatrix
	e.RestoreGraphicsStack(savedStack)
	e.popResources(parent)
	return err
}

// ProcessTilingPattern processes the given tiling pattern, through the pattern's
// own matrix.
//
// c and colorSpace are the colour to use where this is an uncoloured pattern,
// and nil otherwise.
func (e *PDFStreamEngine) ProcessTilingPattern(tilingPattern *pattern.PDTilingPattern,
	c *color.PDColor, colorSpace color.PDColorSpace) error {
	return e.ProcessTilingPatternMatrix(tilingPattern, c, colorSpace, tilingPattern.Matrix())
}

// ProcessTilingPatternMatrix processes the given tiling pattern, through the
// given matrix rather than the pattern's own, which lets custom rendering
// override it.
//
// Port of the four-argument processTilingPattern. Java overloads the name; Go
// has no overloading, and this is the one the renderer calls.
func (e *PDFStreamEngine) ProcessTilingPatternMatrix(tilingPattern *pattern.PDTilingPattern,
	c *color.PDColor, colorSpace color.PDColorSpace, patternMatrix *util.Matrix) error {
	parent := e.pushResources(tilingPattern)

	parentMatrix := e.initialMatrix
	e.initialMatrix = util.Concatenate(e.initialMatrix, patternMatrix)

	// save the original graphics state
	savedStack := e.SaveGraphicsStack()

	// save a clean state (new clipping path, line path, etc.)
	tilingBBox := tilingPattern.BBox()
	bbox := tilingBBox.Transform(patternMatrix).Bounds2D()
	rect := common.NewPDRectangleOf(float32(bbox.X), float32(bbox.Y),
		float32(bbox.Width), float32(bbox.Height))
	e.graphicsStack = append(e.graphicsStack, state.NewPDGraphicsState(rect))
	graphicsState := e.GraphicsState()

	// non-colored patterns have to be given a color
	if colorSpace != nil {
		c = color.NewPDColorOfComponents(c.Components(), colorSpace)
		graphicsState.SetNonStrokingColorSpace(colorSpace)
		graphicsState.SetNonStrokingColor(c)
		graphicsState.SetStrokingColorSpace(colorSpace)
		graphicsState.SetStrokingColor(c)
	}

	// transform the CTM using the stream's matrix
	graphicsState.CurrentTransformationMatrix().Concatenate(patternMatrix)

	// clip to bounding box
	e.clipToRect(tilingBBox)

	err := e.processStreamOperators(tilingPattern)

	e.initialMatrix = parentMatrix
	e.RestoreGraphicsStack(savedStack)
	e.popResources(parent)
	return err
}

// ProcessChildStream processes a child stream of the given page. It cannot be
// used with ProcessPage.
func (e *PDFStreamEngine) ProcessChildStream(contentStream PDContentStream,
	page *pdmodel.PDPage) error {
	if e.isProcessingPage {
		return ErrPageAlreadySet
	}
	e.initPage(page)
	err := e.processStream(contentStream)
	e.currentPage = nil
	return err
}

// appearanceContentStream adapts an appearance stream to PDContentStream, for
// the reason formContentStream gives.
type appearanceContentStream struct{ *annotation.PDAppearanceStream }

var _ PDContentStream = appearanceContentStream{}

// Resources returns the appearance's resources, or nil where it has none.
func (a appearanceContentStream) Resources() *pdmodel.PDResources {
	resources, isResources := a.PDAppearanceStream.Resources().(*pdmodel.PDResources)
	if !isResources {
		return nil
	}
	return resources
}

// ShowAnnotation shows the given annotation, which must be one on the current
// page.
func (e *PDFStreamEngine) ShowAnnotation(a annotation.PDAnnotation) error {
	appearanceStream := e.Overrides().Appearance(a)
	if appearanceStream != nil {
		return e.ProcessAnnotation(a, appearanceStream)
	}
	return nil
}

// Appearance returns the appearance stream to process for the given
// annotation. An engine overrides it to render a particular appearance, such
// as "hover".
//
// Port of getAppearance, which Java names for the getter it is; the port drops
// the prefix as it does elsewhere.
func (e *PDFStreamEngine) Appearance(a annotation.PDAnnotation) *annotation.PDAppearanceStream {
	return a.NormalAppearanceStream()
}

// ProcessAnnotation runs the given appearance stream of the given annotation.
func (e *PDFStreamEngine) ProcessAnnotation(a annotation.PDAnnotation,
	appearance *annotation.PDAppearanceStream) error {
	bbox := appearance.BBox()
	rect := a.Rectangle()

	// PDFBOX-4783: zero-sized rectangles are not valid
	if rect == nil || rect.Width() <= 0 || rect.Height() <= 0 ||
		bbox == nil || bbox.Width() <= 0 || bbox.Height() <= 0 {
		return nil
	}
	matrix := appearance.Matrix()

	// transformed appearance box  fixme: may be an arbitrary shape
	transformedBox := bbox.Transform(matrix).Bounds2D()
	if transformedBox.IsEmpty() {
		// PDFBOX-6095: zero-sized rectangles are not valid
		return nil
	}

	// compute a matrix which scales and translates the transformed appearance box to align
	// with the edges of the annotation's rectangle
	transform := util.TranslateInstance(rect.LowerLeftX(), rect.LowerLeftY())
	transform.Scale(float32(float64(rect.Width())/transformedBox.Width),
		float32(float64(rect.Height())/transformedBox.Height))
	transform.Translate(float32(-transformedBox.X), float32(-transformedBox.Y))

	// Matrix shall be concatenated with A to form a matrix AA that maps from the appearance's
	// coordinate system to the annotation's rectangle in default user space
	//
	// HOWEVER only the opposite order works for rotated pages with
	// filled fields / annotations that have a matrix in the appearance stream, see PDFBOX-3083
	aa := util.Concatenate(transform, matrix)

	contentStream := appearanceContentStream{appearance}
	parent := e.pushResources(contentStream)
	savedStack := e.SaveGraphicsStack()

	// make matrix AA the CTM
	e.GraphicsState().SetCurrentTransformationMatrix(aa)

	// clip to bounding box
	e.clipToRect(bbox)

	// needed for patterns in appearance streams, e.g. PDFBOX-2182
	e.initialMatrix = aa.Clone()

	err := e.processStreamOperators(contentStream)

	e.RestoreGraphicsStack(savedStack)
	e.popResources(parent)
	return err
}
