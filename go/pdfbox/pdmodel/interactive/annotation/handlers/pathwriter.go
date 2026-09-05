package handlers

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// pathWriter writes a run of path operators, keeping the first error and
// skipping the rest the way bufio.Writer does.
//
// The icon handlers are long literal paths that Java writes as one statement a
// line; checking every one of them in Go would bury the drawing, so the error
// is held here and read once at the end.
type pathWriter struct {
	cs  annotation.AppearanceContentStream
	err error
}

// newPathWriter returns a writer over the given content stream.
func newPathWriter(cs annotation.AppearanceContentStream) *pathWriter {
	return &pathWriter{cs: cs}
}

// moveTo begins a new subpath at the given point.
func (p *pathWriter) moveTo(x, y float32) {
	if p.err == nil {
		p.err = p.cs.MoveTo(x, y)
	}
}

// lineTo appends a straight line to the current path.
func (p *pathWriter) lineTo(x, y float32) {
	if p.err == nil {
		p.err = p.cs.LineTo(x, y)
	}
}

// curveTo appends a cubic Bezier curve to the current path.
func (p *pathWriter) curveTo(x1, y1, x2, y2, x3, y3 float32) {
	if p.err == nil {
		p.err = p.cs.CurveTo(x1, y1, x2, y2, x3, y3)
	}
}

// addRect adds a rectangle to the current path.
func (p *pathWriter) addRect(x, y, width, height float32) {
	if p.err == nil {
		p.err = p.cs.AddRect(x, y, width, height)
	}
}

// closePath closes the current subpath.
func (p *pathWriter) closePath() {
	if p.err == nil {
		p.err = p.cs.ClosePath()
	}
}

// clip intersects the clipping path with the current path.
func (p *pathWriter) clip() {
	if p.err == nil {
		p.err = p.cs.Clip()
	}
}

// fill fills the current path.
func (p *pathWriter) fill() {
	if p.err == nil {
		p.err = p.cs.Fill()
	}
}

// stroke strokes the current path.
func (p *pathWriter) stroke() {
	if p.err == nil {
		p.err = p.cs.Stroke()
	}
}

// fillAndStroke fills and strokes the current path.
func (p *pathWriter) fillAndStroke() {
	if p.err == nil {
		p.err = p.cs.FillAndStroke()
	}
}

// closeAndFillAndStroke closes, fills and strokes the current path.
func (p *pathWriter) closeAndFillAndStroke() {
	if p.err == nil {
		p.err = p.cs.CloseAndFillAndStroke()
	}
}

// drawShape closes the current path the way the given stroke and fill ask.
func (p *pathWriter) drawShape(lineWidth float32, hasStroke, hasFill bool) {
	if p.err == nil {
		p.err = p.cs.DrawShape(lineWidth, hasStroke, hasFill)
	}
}

// transform concatenates the given matrix onto the current transformation
// matrix.
func (p *pathWriter) transform(matrix *util.Matrix) {
	if p.err == nil {
		p.err = p.cs.Transform(matrix)
	}
}

// saveGraphicsState pushes the graphics state.
func (p *pathWriter) saveGraphicsState() {
	if p.err == nil {
		p.err = p.cs.SaveGraphicsState()
	}
}

// restoreGraphicsState pops the graphics state.
func (p *pathWriter) restoreGraphicsState() {
	if p.err == nil {
		p.err = p.cs.RestoreGraphicsState()
	}
}

// setLineWidth sets the line width.
func (p *pathWriter) setLineWidth(lineWidth float32) {
	if p.err == nil {
		p.err = p.cs.SetLineWidth(lineWidth)
	}
}

// setLineCapStyle sets the line cap style.
func (p *pathWriter) setLineCapStyle(lineCapStyle int) {
	if p.err == nil {
		p.err = p.cs.SetLineCapStyle(lineCapStyle)
	}
}

// setLineJoinStyle sets the line join style.
func (p *pathWriter) setLineJoinStyle(lineJoinStyle int) {
	if p.err == nil {
		p.err = p.cs.SetLineJoinStyle(lineJoinStyle)
	}
}

// setMiterLimit sets the miter limit.
func (p *pathWriter) setMiterLimit(miterLimit float32) {
	if p.err == nil {
		p.err = p.cs.SetMiterLimit(miterLimit)
	}
}

// setStrokingColor sets the colour to stroke with.
func (p *pathWriter) setStrokingColor(value *color.PDColor) {
	if p.err == nil {
		p.err = p.cs.SetStrokingColor(value)
	}
}

// setNonStrokingColor sets the colour to fill with.
func (p *pathWriter) setNonStrokingColor(value *color.PDColor) {
	if p.err == nil {
		p.err = p.cs.SetNonStrokingColor(value)
	}
}

// setNonStrokingColorGray sets the colour to fill with, in device gray.
func (p *pathWriter) setNonStrokingColorGray(g float32) {
	if p.err == nil {
		p.err = p.cs.SetNonStrokingColorGray(g)
	}
}

// setNonStrokingColorComponents sets the colour to fill with from its
// components.
func (p *pathWriter) setNonStrokingColorComponents(components []float32) {
	if p.err == nil {
		p.err = p.cs.SetNonStrokingColorComponents(components)
	}
}

// circle draws a circle through the handler's DrawCircle.
func (p *pathWriter) circle(h *PDAbstractAppearanceHandler, x, y, r float32) {
	if p.err == nil {
		p.err = h.DrawCircle(p.cs, x, y, r)
	}
}

// circle2 draws a circle the other way round, through DrawCircle2.
func (p *pathWriter) circle2(h *PDAbstractAppearanceHandler, x, y, r float32) {
	if p.err == nil {
		p.err = h.DrawCircle2(p.cs, x, y, r)
	}
}

// setGraphicsStateParameters sets the graphics state from the given extended
// graphics state.
func (p *pathWriter) setGraphicsStateParameters(gs *state.PDExtendedGraphicsState) {
	if p.err == nil {
		p.err = p.cs.SetGraphicsStateParameters(gs)
	}
}
