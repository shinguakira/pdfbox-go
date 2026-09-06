// Package state holds the operator processors that change the graphics state.
//
// Port of org.apache.pdfbox.contentstream.operator.state. Java gives each
// processor a file of its own; they are a few lines each, so the port keeps
// them together.
package state

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	graphicsstate "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// isNumber is the operand test the processors here use, standing in for Java's
// checkArrayTypesClass(arguments, COSNumber.class).
func isNumber(base cos.Base) bool {
	_, ok := base.(cos.Number)
	return ok
}

// AddAll registers every processor in this package with the engine.
func AddAll(context *contentstream.PDFStreamEngine) {
	context.AddOperator(NewSave(context))
	context.AddOperator(NewRestore(context))
	context.AddOperator(NewConcatenate(context))
	context.AddOperator(NewSetFlatness(context))
	context.AddOperator(NewSetLineCapStyle(context))
	context.AddOperator(NewSetLineDashPattern(context))
	context.AddOperator(NewSetLineJoinStyle(context))
	context.AddOperator(NewSetLineMiterLimit(context))
	context.AddOperator(NewSetLineWidth(context))
	context.AddOperator(NewSetMatrix(context))
	context.AddOperator(NewSetGraphicsStateParameters(context))
	context.AddOperator(NewSetRenderingIntent(context))
}

// matrixOf reads the first six operands as a matrix. They must all be numbers.
func matrixOf(arguments []cos.Base) *util.Matrix {
	return util.NewMatrixOf(
		arguments[0].(cos.Number).FloatValue(),
		arguments[1].(cos.Number).FloatValue(),
		arguments[2].(cos.Number).FloatValue(),
		arguments[3].(cos.Number).FloatValue(),
		arguments[4].(cos.Number).FloatValue(),
		arguments[5].(cos.Number).FloatValue(),
	)
}

// Save is q: push the graphics state.
type Save struct {
	contentstream.BaseOperatorProcessor
}

// NewSave returns the q processor.
func NewSave(context *contentstream.PDFStreamEngine) *Save {
	return &Save{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *Save) Name() string { return operator.Save }

// Process pushes the graphics state.
func (p *Save) Process(op *operator.Operator, arguments []cos.Base) error {
	p.Context().SaveGraphicsState()
	return nil
}

// Restore is Q: pop the graphics state.
type Restore struct {
	contentstream.BaseOperatorProcessor
}

// NewRestore returns the Q processor.
func NewRestore(context *contentstream.PDFStreamEngine) *Restore {
	return &Restore{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *Restore) Name() string { return operator.Restore }

// Process pops the graphics state, reporting a stream that pops more than it
// pushed.
func (p *Restore) Process(op *operator.Operator, arguments []cos.Base) error {
	context := p.Context()
	if context.GraphicsStackSize() > 1 {
		context.RestoreGraphicsState()
		return nil
	}
	// this shouldn't happen but it does, see PDFBOX-161
	return operator.ErrEmptyGraphicsStack
}

// Concatenate is cm: concatenate a matrix to the CTM.
type Concatenate struct {
	contentstream.BaseOperatorProcessor
}

// NewConcatenate returns the cm processor.
func NewConcatenate(context *contentstream.PDFStreamEngine) *Concatenate {
	return &Concatenate{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *Concatenate) Name() string { return operator.Concat }

// Process concatenates the matrix to the current transformation matrix.
func (p *Concatenate) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) < 6 {
		return operator.MissingOperand(op, arguments)
	}
	if !contentstream.AllOperandsAre(arguments, isNumber) {
		return nil
	}
	p.Context().GraphicsState().CurrentTransformationMatrix().Concatenate(matrixOf(arguments))
	return nil
}

// SetFlatness is i: set the flatness tolerance.
type SetFlatness struct {
	contentstream.BaseOperatorProcessor
}

// NewSetFlatness returns the i processor.
func NewSetFlatness(context *contentstream.PDFStreamEngine) *SetFlatness {
	return &SetFlatness{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetFlatness) Name() string { return operator.SetFlatness }

// Process sets the flatness tolerance.
func (p *SetFlatness) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) == 0 {
		return operator.MissingOperand(op, operands)
	}
	if !contentstream.AllOperandsAre(operands, isNumber) {
		return nil
	}
	p.Context().GraphicsState().SetFlatness(float64(operands[0].(cos.Number).FloatValue()))
	return nil
}

// SetLineCapStyle is J: set the line cap style.
type SetLineCapStyle struct {
	contentstream.BaseOperatorProcessor
}

// NewSetLineCapStyle returns the J processor.
func NewSetLineCapStyle(context *contentstream.PDFStreamEngine) *SetLineCapStyle {
	return &SetLineCapStyle{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetLineCapStyle) Name() string { return operator.SetLineCapstyle }

// Process sets the line cap style.
func (p *SetLineCapStyle) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return operator.MissingOperand(op, arguments)
	}
	if !contentstream.AllOperandsAre(arguments, isNumber) {
		return nil
	}
	p.Context().GraphicsState().SetLineCap(arguments[0].(cos.Number).IntValue())
	return nil
}

// SetLineJoinStyle is j: set the line join style.
type SetLineJoinStyle struct {
	contentstream.BaseOperatorProcessor
}

// NewSetLineJoinStyle returns the j processor.
func NewSetLineJoinStyle(context *contentstream.PDFStreamEngine) *SetLineJoinStyle {
	return &SetLineJoinStyle{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetLineJoinStyle) Name() string { return operator.SetLineJoinstyle }

// Process sets the line join style.
func (p *SetLineJoinStyle) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return operator.MissingOperand(op, arguments)
	}
	if !contentstream.AllOperandsAre(arguments, isNumber) {
		return nil
	}
	p.Context().GraphicsState().SetLineJoin(arguments[0].(cos.Number).IntValue())
	return nil
}

// SetLineMiterLimit is M: set the miter limit.
type SetLineMiterLimit struct {
	contentstream.BaseOperatorProcessor
}

// NewSetLineMiterLimit returns the M processor.
func NewSetLineMiterLimit(context *contentstream.PDFStreamEngine) *SetLineMiterLimit {
	return &SetLineMiterLimit{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetLineMiterLimit) Name() string { return operator.SetLineMiterlimit }

// Process sets the miter limit.
func (p *SetLineMiterLimit) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return operator.MissingOperand(op, arguments)
	}
	if !contentstream.AllOperandsAre(arguments, isNumber) {
		return nil
	}
	p.Context().GraphicsState().SetMiterLimit(arguments[0].(cos.Number).FloatValue())
	return nil
}

// SetLineWidth is w: set the line width.
type SetLineWidth struct {
	contentstream.BaseOperatorProcessor
}

// NewSetLineWidth returns the w processor.
func NewSetLineWidth(context *contentstream.PDFStreamEngine) *SetLineWidth {
	return &SetLineWidth{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetLineWidth) Name() string { return operator.SetLineWidth }

// Process sets the line width.
func (p *SetLineWidth) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return operator.MissingOperand(op, arguments)
	}
	if !contentstream.AllOperandsAre(arguments, isNumber) {
		return nil
	}
	p.Context().GraphicsState().SetLineWidth(arguments[0].(cos.Number).FloatValue())
	return nil
}

// SetMatrix is Tm: set the text matrix and the text line matrix.
type SetMatrix struct {
	contentstream.BaseOperatorProcessor
}

// NewSetMatrix returns the Tm processor.
func NewSetMatrix(context *contentstream.PDFStreamEngine) *SetMatrix {
	return &SetMatrix{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetMatrix) Name() string { return operator.SetMatrix }

// Process sets both text matrices.
func (p *SetMatrix) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) < 6 {
		return operator.MissingOperand(op, arguments)
	}
	if !contentstream.AllOperandsAre(arguments, isNumber) {
		return nil
	}
	matrix := matrixOf(arguments)
	context := p.Context()
	context.SetTextMatrix(matrix)
	context.SetTextLineMatrix(matrix.Clone())
	return nil
}

// SetRenderingIntent is ri: set the rendering intent.
type SetRenderingIntent struct {
	contentstream.BaseOperatorProcessor
}

// NewSetRenderingIntent returns the ri processor.
func NewSetRenderingIntent(context *contentstream.PDFStreamEngine) *SetRenderingIntent {
	return &SetRenderingIntent{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetRenderingIntent) Name() string { return operator.SetRenderingintent }

// Process sets the rendering intent.
func (p *SetRenderingIntent) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) == 0 {
		return operator.MissingOperand(op, operands)
	}
	name, ok := operands[0].(*cos.Name)
	if !ok {
		return nil
	}
	p.Context().GraphicsState().
		SetRenderingIntent(graphicsstate.RenderingIntentFromString(name.Name()))
	return nil
}

// SetLineDashPattern is d: set the line dash pattern.
type SetLineDashPattern struct {
	contentstream.BaseOperatorProcessor
}

// NewSetLineDashPattern returns the d processor.
func NewSetLineDashPattern(context *contentstream.PDFStreamEngine) *SetLineDashPattern {
	return &SetLineDashPattern{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetLineDashPattern) Name() string { return operator.SetLineDashpattern }

// Process sets the line dash pattern. A dash array holding anything that is not
// a number is discarded entirely, and one that is all zeroes is kept as it
// stands — the walk below stops at the first non-zero, so a pattern that would
// draw nothing still reaches the graphics state.
func (p *SetLineDashPattern) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) < 2 {
		return operator.MissingOperand(op, arguments)
	}
	dashArray, ok := arguments[0].(*cos.Array)
	if !ok {
		return nil
	}
	phase, ok := arguments[1].(cos.Number)
	if !ok {
		return nil
	}
	for _, base := range dashArray.ToList() {
		if number, isNumber := base.(cos.Number); isNumber {
			if number.FloatValue() != 0 {
				break
			}
			continue
		}
		slog.Warn("dash array has non number element, ignored", "element", base)
		dashArray = cos.NewArray()
		break
	}
	p.Context().SetLineDashPattern(dashArray, phase.IntValue())
	return nil
}

// SetGraphicsStateParameters is gs: set the graphics state parameters from an
// /ExtGState resource.
type SetGraphicsStateParameters struct {
	contentstream.BaseOperatorProcessor
}

// NewSetGraphicsStateParameters returns the gs processor.
func NewSetGraphicsStateParameters(context *contentstream.PDFStreamEngine) *SetGraphicsStateParameters {
	return &SetGraphicsStateParameters{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetGraphicsStateParameters) Name() string { return operator.SetGraphicsStateParams }

// Process applies the named extended graphics state to the graphics state.
func (p *SetGraphicsStateParameters) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) == 0 {
		return operator.MissingOperand(op, operands)
	}
	graphicsName, isName := operands[0].(*cos.Name)
	if !isName {
		return nil
	}
	// set parameters from graphics state parameter dictionary
	context := p.Context()
	gs := context.Resources().GetExtGState(graphicsName)
	if gs == nil {
		slog.Error("state: name for 'gs' operator not found in resources",
			slog.String("name", "/"+graphicsName.Name()))
		return nil
	}
	return gs.CopyIntoGraphicsState(context.GraphicsState())
}
