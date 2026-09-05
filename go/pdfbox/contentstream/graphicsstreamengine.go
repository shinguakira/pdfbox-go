package contentstream

// The engine a renderer drives, and the boundary the raster backend sits
// behind.
//
// Port of org.apache.pdfbox.contentstream.PDFGraphicsStreamEngine.

import (
	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
)

// GraphicsStreamEngineOverrides is what a graphics engine supplies.
//
// PDFGraphicsStreamEngine is an abstract class in Java with thirteen abstract
// methods; the port splits it into this interface for those and the struct
// below for the state and the operator registrations.
//
// This is the boundary slice 9's raster backend decision draws. Everything
// above it -- the operators, the graphics state, the colour and shading
// arithmetic -- is ported; what an implementation does with these calls is the
// drawing, and no implementation of them ships in this slice. See
// migration/STATUS.md.
type GraphicsStreamEngineOverrides interface {
	StreamEngineOverrides

	// AppendRectangle appends a rectangle to the current path, given its four
	// corners already in device space.
	AppendRectangle(p0, p1, p2, p3 geom.Point2D) error

	// DrawImage draws the given image.
	DrawImage(pdImage image.PDImage) error

	// Clip modifies the current clipping path by intersecting it with the
	// current path, under the given winding rule.
	Clip(windingRule int) error

	// MoveTo starts a new subpath at the given point.
	MoveTo(x, y float32) error

	// LineTo draws a line from the current point to the given one.
	LineTo(x, y float32) error

	// CurveTo draws a cubic Bezier curve from the current point.
	CurveTo(x1, y1, x2, y2, x3, y3 float32) error

	// CurrentPoint returns the current point of the current path, or nil where
	// the path has not started.
	CurrentPoint() (geom.Point2D, error)

	// ClosePath closes the current subpath.
	ClosePath() error

	// EndPath ends the current path without filling or stroking it, applying
	// any pending clip.
	EndPath() error

	// StrokePath strokes the current path.
	StrokePath() error

	// FillPath fills the current path under the given winding rule.
	FillPath(windingRule int) error

	// FillAndStrokePath fills and then strokes the current path.
	FillAndStrokePath(windingRule int) error

	// ShadingFill paints the shading of the given name over the clip.
	ShadingFill(shadingName *cos.Name) error
}

// PDFGraphicsStreamEngine is a stream engine that draws.
//
// Port of the non-abstract half of PDFGraphicsStreamEngine: the page it runs
// over, and the operators it registers on top of the text ones.
type PDFGraphicsStreamEngine struct {
	*PDFStreamEngine

	// page may be nil, for example if the stream is a tiling pattern.
	page *pdmodel.PDPage
}

// NewPDFGraphicsStreamEngine returns an engine over the given page, with no
// operators registered.
//
// Port of the protected PDFGraphicsStreamEngine(PDPage) constructor, minus the
// sixty addOperator calls that make up the rest of it. This package cannot make
// them: every processor holds the engine, so the operator packages import this
// one and it cannot import them back. The concrete engine registers them
// instead, which is what text.NewLegacyPDFStreamEngine already does, and the
// set Java's constructor names is exactly:
//
//	state.AddAll(e.PDFStreamEngine)
//	text.AddAll(e.PDFStreamEngine)
//	markedcontent.AddSequenceOperators(e.PDFStreamEngine)
//	color.AddAll(e.PDFStreamEngine)
//	graphics.AddAll(e)
func NewPDFGraphicsStreamEngine(page *pdmodel.PDPage) *PDFGraphicsStreamEngine {
	e := &PDFGraphicsStreamEngine{PDFStreamEngine: NewPDFStreamEngine(), page: page}
	return e
}

// Page returns the page being drawn, or nil where the stream is not a page --
// a tiling pattern, for instance.
func (e *PDFGraphicsStreamEngine) Page() *pdmodel.PDPage { return e.page }
