// Package graphics holds the operator processors that build and paint a path.
//
// Port of org.apache.pdfbox.contentstream.operator.graphics. Java gives each
// processor a file of its own; they are a few lines each and every one of them
// forwards to the graphics engine, so the port keeps them together.
//
// The two that draw an image -- DrawObject and BeginInlineImage -- are here
// with the rest. What an engine does with the image is behind the raster
// backend interface; deciding which image to hand it is not, and that is all
// these two do.
package graphics

import (
	"fmt"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
)

// AddAll registers every processor in this package with the engine.
func AddAll(context *contentstream.PDFGraphicsStreamEngine) {
	context.AddOperator(NewAppendRectangleToPath(context))
	context.AddOperator(NewBeginInlineImage(context))
	context.AddOperator(NewClipEvenOddRule(context))
	context.AddOperator(NewClipNonZeroRule(context))
	context.AddOperator(NewCloseAndStrokePath(context))
	context.AddOperator(NewCloseFillEvenOddAndStrokePath(context))
	context.AddOperator(NewCloseFillNonZeroAndStrokePath(context))
	context.AddOperator(NewClosePath(context))
	context.AddOperator(NewCurveTo(context))
	context.AddOperator(NewCurveToReplicateFinalPoint(context))
	context.AddOperator(NewCurveToReplicateInitialPoint(context))
	context.AddOperator(NewDrawObject(context))
	context.AddOperator(NewEndPath(context))
	context.AddOperator(NewFillEvenOddAndStrokePath(context))
	context.AddOperator(NewFillEvenOddRule(context))
	context.AddOperator(NewFillNonZeroAndStrokePath(context))
	context.AddOperator(NewFillNonZeroRule(context))
	context.AddOperator(NewLegacyFillNonZeroRule(context))
	context.AddOperator(NewLineTo(context))
	context.AddOperator(NewMoveTo(context))
	context.AddOperator(NewShadingFill(context))
	context.AddOperator(NewStrokePath(context))
}

// GraphicsOperatorProcessor is the shared part of every processor here.
//
// Port of the abstract GraphicsOperatorProcessor, whose whole body is the cast
// of the context to a PDFGraphicsStreamEngine. The port holds the graphics
// engine rather than casting, because a Go embedded field cannot be narrowed.
type GraphicsOperatorProcessor struct {
	contentstream.BaseOperatorProcessor

	graphics *contentstream.PDFGraphicsStreamEngine
}

// newGraphicsOperatorProcessor returns the shared part running against the
// given engine.
func newGraphicsOperatorProcessor(
	context *contentstream.PDFGraphicsStreamEngine) GraphicsOperatorProcessor {
	return GraphicsOperatorProcessor{
		BaseOperatorProcessor: contentstream.NewBaseOperatorProcessor(context.PDFStreamEngine),
		graphics:              context,
	}
}

// GraphicsContext returns the engine this processor draws through.
func (p GraphicsOperatorProcessor) GraphicsContext() *contentstream.PDFGraphicsStreamEngine {
	return p.graphics
}

// drawing is the half of the engine every processor here calls, which the
// engine's owner supplies.
//
// Java cannot reach this: PDFGraphicsStreamEngine declares the thirteen
// abstract, so a graphics engine has them by construction. The port hands them
// over through SetOverrides instead, and an engine that never did so is a
// programming error rather than a bad PDF -- so this panics, the way the
// unchecked exception Java would raise does.
func (p GraphicsOperatorProcessor) drawing() contentstream.GraphicsStreamEngineOverrides {
	overrides, isGraphics := p.graphics.Overrides().(contentstream.GraphicsStreamEngineOverrides)
	if !isGraphics {
		panic("graphics: the engine has no drawing half, " +
			"call SetOverrides with a GraphicsStreamEngineOverrides")
	}
	return overrides
}

// isNumber is the `instanceof COSNumber` these processors test with.
func isNumber(base cos.Base) bool {
	_, ok := base.(cos.Number)
	return ok
}

// floatAt reads one operand as a number.
func floatAt(operands []cos.Base, index int) float32 {
	return operands[index].(cos.Number).FloatValue()
}

// AppendRectangleToPath is re: append a rectangle to the current path.
type AppendRectangleToPath struct{ GraphicsOperatorProcessor }

// NewAppendRectangleToPath returns the re processor.
func NewAppendRectangleToPath(c *contentstream.PDFGraphicsStreamEngine) *AppendRectangleToPath {
	return &AppendRectangleToPath{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *AppendRectangleToPath) Name() string { return operator.AppendRect }

// Process appends the rectangle, transforming its four corners into device
// space first.
func (p *AppendRectangleToPath) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) < 4 {
		return operator.MissingOperand(op, operands)
	}
	if !contentstream.AllOperandsAre(operands, isNumber) {
		return nil
	}
	x1 := floatAt(operands, 0)
	y1 := floatAt(operands, 1)
	// create a pair of coordinates for the transformation
	x2 := floatAt(operands, 2) + x1
	y2 := floatAt(operands, 3) + y1

	context := p.GraphicsContext()
	p0 := context.TransformedPoint(x1, y1)
	p1 := context.TransformedPoint(x2, y1)
	p2 := context.TransformedPoint(x2, y2)
	p3 := context.TransformedPoint(x1, y2)
	return p.drawing().AppendRectangle(p0, p1, p2, p3)
}

// MoveTo is m: begin a new subpath.
type MoveTo struct{ GraphicsOperatorProcessor }

// NewMoveTo returns the m processor.
func NewMoveTo(c *contentstream.PDFGraphicsStreamEngine) *MoveTo {
	return &MoveTo{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *MoveTo) Name() string { return operator.MoveTo }

// Process begins the subpath.
func (p *MoveTo) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) < 2 {
		return operator.MissingOperand(op, operands)
	}
	if !isNumber(operands[0]) || !isNumber(operands[1]) {
		return nil
	}
	pos := p.GraphicsContext().TransformedPoint(floatAt(operands, 0), floatAt(operands, 1))
	return p.drawing().MoveTo(pos.XFloat(), pos.YFloat())
}

// LineTo is l: append a straight line segment.
type LineTo struct{ GraphicsOperatorProcessor }

// NewLineTo returns the l processor.
func NewLineTo(c *contentstream.PDFGraphicsStreamEngine) *LineTo {
	return &LineTo{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *LineTo) Name() string { return operator.LineTo }

// Process appends the segment, beginning the subpath first where there is no
// current point.
func (p *LineTo) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) < 2 {
		return operator.MissingOperand(op, operands)
	}
	if !isNumber(operands[0]) || !isNumber(operands[1]) {
		return nil
	}
	// append straight line segment from the current point to the point
	pos := p.GraphicsContext().TransformedPoint(floatAt(operands, 0), floatAt(operands, 1))
	drawing := p.drawing()
	current, err := drawing.CurrentPoint()
	if err != nil {
		return err
	}
	if current == nil {
		slog.Warn("graphics: LineTo without initial MoveTo", "x", pos.XFloat(), "y", pos.YFloat())
		return drawing.MoveTo(pos.XFloat(), pos.YFloat())
	}
	return drawing.LineTo(pos.XFloat(), pos.YFloat())
}

// CurveTo is c: append a cubic Bezier curve with two control points.
type CurveTo struct{ GraphicsOperatorProcessor }

// NewCurveTo returns the c processor.
func NewCurveTo(c *contentstream.PDFGraphicsStreamEngine) *CurveTo {
	return &CurveTo{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *CurveTo) Name() string { return operator.CurveTo }

// Process appends the curve.
func (p *CurveTo) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) < 6 {
		return operator.MissingOperand(op, operands)
	}
	if !contentstream.AllOperandsAre(operands, isNumber) {
		return nil
	}
	context := p.GraphicsContext()
	point1 := context.TransformedPoint(floatAt(operands, 0), floatAt(operands, 1))
	point2 := context.TransformedPoint(floatAt(operands, 2), floatAt(operands, 3))
	point3 := context.TransformedPoint(floatAt(operands, 4), floatAt(operands, 5))

	drawing := p.drawing()
	current, err := drawing.CurrentPoint()
	if err != nil {
		return err
	}
	if current == nil {
		slog.Warn("graphics: curveTo without initial MoveTo",
			"x", point3.XFloat(), "y", point3.YFloat())
		return drawing.MoveTo(point3.XFloat(), point3.YFloat())
	}
	return drawing.CurveTo(point1.XFloat(), point1.YFloat(),
		point2.XFloat(), point2.YFloat(), point3.XFloat(), point3.YFloat())
}

// CurveToReplicateInitialPoint is v: a cubic curve whose first control point is
// the current point.
type CurveToReplicateInitialPoint struct{ GraphicsOperatorProcessor }

// NewCurveToReplicateInitialPoint returns the v processor.
func NewCurveToReplicateInitialPoint(
	c *contentstream.PDFGraphicsStreamEngine) *CurveToReplicateInitialPoint {
	return &CurveToReplicateInitialPoint{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *CurveToReplicateInitialPoint) Name() string {
	return operator.CurveToReplicateInitialPoint
}

// Process appends the curve.
func (p *CurveToReplicateInitialPoint) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) < 4 {
		return operator.MissingOperand(op, operands)
	}
	if !contentstream.AllOperandsAre(operands, isNumber) {
		return nil
	}
	context := p.GraphicsContext()
	drawing := p.drawing()
	currentPoint, err := drawing.CurrentPoint()
	if err != nil {
		return err
	}
	point2 := context.TransformedPoint(floatAt(operands, 0), floatAt(operands, 1))
	point3 := context.TransformedPoint(floatAt(operands, 2), floatAt(operands, 3))
	if currentPoint == nil {
		slog.Warn("graphics: curveTo without initial MoveTo",
			"x", point3.XFloat(), "y", point3.YFloat())
		return drawing.MoveTo(point3.XFloat(), point3.YFloat())
	}
	return drawing.CurveTo(float32(currentPoint.X()), float32(currentPoint.Y()),
		point2.XFloat(), point2.YFloat(), point3.XFloat(), point3.YFloat())
}

// CurveToReplicateFinalPoint is y: a cubic curve whose second control point is
// its end point.
type CurveToReplicateFinalPoint struct{ GraphicsOperatorProcessor }

// NewCurveToReplicateFinalPoint returns the y processor.
func NewCurveToReplicateFinalPoint(
	c *contentstream.PDFGraphicsStreamEngine) *CurveToReplicateFinalPoint {
	return &CurveToReplicateFinalPoint{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *CurveToReplicateFinalPoint) Name() string {
	return operator.CurveToReplicateFinalPoint
}

// Process appends the curve.
func (p *CurveToReplicateFinalPoint) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) < 4 {
		return operator.MissingOperand(op, operands)
	}
	if !contentstream.AllOperandsAre(operands, isNumber) {
		return nil
	}
	context := p.GraphicsContext()
	drawing := p.drawing()
	currentPoint, err := drawing.CurrentPoint()
	if err != nil {
		return err
	}
	point1 := context.TransformedPoint(floatAt(operands, 0), floatAt(operands, 1))
	point3 := context.TransformedPoint(floatAt(operands, 2), floatAt(operands, 3))
	if currentPoint == nil {
		slog.Warn("graphics: curveTo without initial MoveTo",
			"x", point3.XFloat(), "y", point3.YFloat())
		return drawing.MoveTo(point3.XFloat(), point3.YFloat())
	}
	return drawing.CurveTo(point1.XFloat(), point1.YFloat(),
		point3.XFloat(), point3.YFloat(), point3.XFloat(), point3.YFloat())
}

// ClosePath is h: close the current subpath.
type ClosePath struct{ GraphicsOperatorProcessor }

// NewClosePath returns the h processor.
func NewClosePath(c *contentstream.PDFGraphicsStreamEngine) *ClosePath {
	return &ClosePath{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *ClosePath) Name() string { return operator.ClosePath }

// Process closes the subpath, doing nothing where there is no current point.
func (p *ClosePath) Process(*operator.Operator, []cos.Base) error {
	drawing := p.drawing()
	current, err := drawing.CurrentPoint()
	if err != nil {
		return err
	}
	if current == nil {
		slog.Warn("graphics: ClosePath without initial MoveTo")
		return nil
	}
	return drawing.ClosePath()
}

// EndPath is n: end the path without painting it.
type EndPath struct{ GraphicsOperatorProcessor }

// NewEndPath returns the n processor.
func NewEndPath(c *contentstream.PDFGraphicsStreamEngine) *EndPath {
	return &EndPath{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *EndPath) Name() string { return operator.Endpath }

// Process ends the path.
func (p *EndPath) Process(*operator.Operator, []cos.Base) error {
	return p.drawing().EndPath()
}

// StrokePath is S: stroke the current path.
type StrokePath struct{ GraphicsOperatorProcessor }

// NewStrokePath returns the S processor.
func NewStrokePath(c *contentstream.PDFGraphicsStreamEngine) *StrokePath {
	return &StrokePath{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *StrokePath) Name() string { return operator.StrokePath }

// Process strokes the path.
func (p *StrokePath) Process(*operator.Operator, []cos.Base) error {
	return p.drawing().StrokePath()
}

// CloseAndStrokePath is s: close the subpath and then stroke the path.
type CloseAndStrokePath struct{ GraphicsOperatorProcessor }

// NewCloseAndStrokePath returns the s processor.
func NewCloseAndStrokePath(c *contentstream.PDFGraphicsStreamEngine) *CloseAndStrokePath {
	return &CloseAndStrokePath{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *CloseAndStrokePath) Name() string { return operator.CloseAndStroke }

// Process runs the two operators it stands for, which is what Java does rather
// than calling the engine twice.
func (p *CloseAndStrokePath) Process(_ *operator.Operator, arguments []cos.Base) error {
	context := p.Context()
	if err := context.ProcessOperatorNamed(operator.ClosePath, arguments); err != nil {
		return err
	}
	return context.ProcessOperatorNamed(operator.StrokePath, arguments)
}

// FillNonZeroRule is f: fill the path under the non-zero winding rule.
type FillNonZeroRule struct{ GraphicsOperatorProcessor }

// NewFillNonZeroRule returns the f processor.
func NewFillNonZeroRule(c *contentstream.PDFGraphicsStreamEngine) *FillNonZeroRule {
	return &FillNonZeroRule{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *FillNonZeroRule) Name() string { return operator.FillNonZero }

// Process fills the path.
func (p *FillNonZeroRule) Process(*operator.Operator, []cos.Base) error {
	return p.drawing().FillPath(geom.WindNonZero)
}

// LegacyFillNonZeroRule is F: the obsolete spelling of f, which fills exactly
// the same way.
//
// Port of LegacyFillNonZeroRule, which extends FillNonZeroRule and overrides
// nothing but the name.
type LegacyFillNonZeroRule struct{ FillNonZeroRule }

// NewLegacyFillNonZeroRule returns the F processor.
func NewLegacyFillNonZeroRule(c *contentstream.PDFGraphicsStreamEngine) *LegacyFillNonZeroRule {
	return &LegacyFillNonZeroRule{FillNonZeroRule{newGraphicsOperatorProcessor(c)}}
}

// Name returns the operator this processes.
func (p *LegacyFillNonZeroRule) Name() string { return operator.LegacyFillNonZero }

// FillEvenOddRule is f*: fill the path under the even-odd rule.
type FillEvenOddRule struct{ GraphicsOperatorProcessor }

// NewFillEvenOddRule returns the f* processor.
func NewFillEvenOddRule(c *contentstream.PDFGraphicsStreamEngine) *FillEvenOddRule {
	return &FillEvenOddRule{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *FillEvenOddRule) Name() string { return operator.FillEvenOdd }

// Process fills the path.
func (p *FillEvenOddRule) Process(*operator.Operator, []cos.Base) error {
	return p.drawing().FillPath(geom.WindEvenOdd)
}

// FillNonZeroAndStrokePath is B: fill and then stroke, under the non-zero rule.
type FillNonZeroAndStrokePath struct{ GraphicsOperatorProcessor }

// NewFillNonZeroAndStrokePath returns the B processor.
func NewFillNonZeroAndStrokePath(
	c *contentstream.PDFGraphicsStreamEngine) *FillNonZeroAndStrokePath {
	return &FillNonZeroAndStrokePath{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *FillNonZeroAndStrokePath) Name() string { return operator.FillNonZeroAndStroke }

// Process fills and strokes the path.
func (p *FillNonZeroAndStrokePath) Process(*operator.Operator, []cos.Base) error {
	return p.drawing().FillAndStrokePath(geom.WindNonZero)
}

// FillEvenOddAndStrokePath is B*: fill and then stroke, under the even-odd rule.
type FillEvenOddAndStrokePath struct{ GraphicsOperatorProcessor }

// NewFillEvenOddAndStrokePath returns the B* processor.
func NewFillEvenOddAndStrokePath(
	c *contentstream.PDFGraphicsStreamEngine) *FillEvenOddAndStrokePath {
	return &FillEvenOddAndStrokePath{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *FillEvenOddAndStrokePath) Name() string { return operator.FillEvenOddAndStroke }

// Process fills and strokes the path.
func (p *FillEvenOddAndStrokePath) Process(*operator.Operator, []cos.Base) error {
	return p.drawing().FillAndStrokePath(geom.WindEvenOdd)
}

// CloseFillNonZeroAndStrokePath is b: close, fill and stroke, non-zero.
type CloseFillNonZeroAndStrokePath struct{ GraphicsOperatorProcessor }

// NewCloseFillNonZeroAndStrokePath returns the b processor.
func NewCloseFillNonZeroAndStrokePath(
	c *contentstream.PDFGraphicsStreamEngine) *CloseFillNonZeroAndStrokePath {
	return &CloseFillNonZeroAndStrokePath{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *CloseFillNonZeroAndStrokePath) Name() string {
	return operator.CloseFillNonZeroAndStroke
}

// Process runs the two operators it stands for.
func (p *CloseFillNonZeroAndStrokePath) Process(_ *operator.Operator, operands []cos.Base) error {
	context := p.Context()
	if err := context.ProcessOperatorNamed(operator.ClosePath, operands); err != nil {
		return err
	}
	return context.ProcessOperatorNamed(operator.FillNonZeroAndStroke, operands)
}

// CloseFillEvenOddAndStrokePath is b*: close, fill and stroke, even-odd.
type CloseFillEvenOddAndStrokePath struct{ GraphicsOperatorProcessor }

// NewCloseFillEvenOddAndStrokePath returns the b* processor.
func NewCloseFillEvenOddAndStrokePath(
	c *contentstream.PDFGraphicsStreamEngine) *CloseFillEvenOddAndStrokePath {
	return &CloseFillEvenOddAndStrokePath{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *CloseFillEvenOddAndStrokePath) Name() string {
	return operator.CloseFillEvenOddAndStroke
}

// Process runs the two operators it stands for.
func (p *CloseFillEvenOddAndStrokePath) Process(_ *operator.Operator, operands []cos.Base) error {
	context := p.Context()
	if err := context.ProcessOperatorNamed(operator.ClosePath, operands); err != nil {
		return err
	}
	return context.ProcessOperatorNamed(operator.FillEvenOddAndStroke, operands)
}

// ClipNonZeroRule is W: intersect the clip with the path, non-zero.
type ClipNonZeroRule struct{ GraphicsOperatorProcessor }

// NewClipNonZeroRule returns the W processor.
func NewClipNonZeroRule(c *contentstream.PDFGraphicsStreamEngine) *ClipNonZeroRule {
	return &ClipNonZeroRule{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *ClipNonZeroRule) Name() string { return operator.ClipNonZero }

// Process intersects the clip.
func (p *ClipNonZeroRule) Process(*operator.Operator, []cos.Base) error {
	return p.drawing().Clip(geom.WindNonZero)
}

// ClipEvenOddRule is W*: intersect the clip with the path, even-odd.
type ClipEvenOddRule struct{ GraphicsOperatorProcessor }

// NewClipEvenOddRule returns the W* processor.
func NewClipEvenOddRule(c *contentstream.PDFGraphicsStreamEngine) *ClipEvenOddRule {
	return &ClipEvenOddRule{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *ClipEvenOddRule) Name() string { return operator.ClipEvenOdd }

// Process intersects the clip.
func (p *ClipEvenOddRule) Process(*operator.Operator, []cos.Base) error {
	return p.drawing().Clip(geom.WindEvenOdd)
}

// ShadingFill is sh: paint the named shading over the clip.
type ShadingFill struct{ GraphicsOperatorProcessor }

// NewShadingFill returns the sh processor.
func NewShadingFill(c *contentstream.PDFGraphicsStreamEngine) *ShadingFill {
	return &ShadingFill{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *ShadingFill) Name() string { return operator.ShadingFill }

// Process paints the shading.
func (p *ShadingFill) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) == 0 {
		return operator.MissingOperand(op, operands)
	}
	name, isName := operands[0].(*cos.Name)
	if !isName {
		return operator.MissingOperand(op, operands)
	}
	return p.drawing().ShadingFill(name)
}

// DrawObject is Do: draw the named XObject.
type DrawObject struct{ GraphicsOperatorProcessor }

// NewDrawObject returns the Do processor.
func NewDrawObject(c *contentstream.PDFGraphicsStreamEngine) *DrawObject {
	return &DrawObject{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *DrawObject) Name() string { return operator.DrawObject }

// Process draws the XObject: an image through the engine, a form or a
// transparency group by running its content stream.
func (p *DrawObject) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) == 0 {
		return operator.MissingOperand(op, operands)
	}
	objectName, isName := operands[0].(*cos.Name)
	if !isName {
		return nil
	}
	context := p.GraphicsContext()
	xobject, err := context.Resources().GetXObject(objectName)
	if err != nil {
		return err
	}
	switch object := xobject.(type) {
	case nil:
		return fmt.Errorf("%w: Missing XObject: %s", pdmodel.ErrMissingResource, objectName.Name())
	case *image.PDImageXObject:
		if !object.IsStencil() && !context.ShouldProcessColorOperators() {
			return nil
		}
		return p.drawing().DrawImage(object)
	}
	return contentstream.ShowFormXObject(context.PDFStreamEngine, xobject)
}

// BeginInlineImage is BI: draw the image the operator carries with it.
type BeginInlineImage struct{ GraphicsOperatorProcessor }

// NewBeginInlineImage returns the BI processor.
func NewBeginInlineImage(c *contentstream.PDFGraphicsStreamEngine) *BeginInlineImage {
	return &BeginInlineImage{newGraphicsOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *BeginInlineImage) Name() string { return operator.BeginInlineImage }

// Process draws the inline image.
func (p *BeginInlineImage) Process(op *operator.Operator, _ []cos.Base) error {
	if len(op.ImageData()) == 0 {
		return nil
	}
	context := p.GraphicsContext()
	inlineImage, err := image.NewPDInlineImage(op.ImageParameters(), op.ImageData(),
		context.Resources())
	if err != nil {
		return err
	}
	// maybe something went wrong when decoding the image data
	if inlineImage.IsEmpty() {
		return nil
	}
	if !inlineImage.IsStencil() && !context.ShouldProcessColorOperators() {
		return nil
	}
	return p.drawing().DrawImage(inlineImage)
}
