// Package text holds the operator processors for text state and positioning.
//
// Port of org.apache.pdfbox.contentstream.operator.text. Java gives each
// processor a file of its own; they are a few lines each, so the port keeps
// them together.
//
// The processors that put glyphs on the page are not here — Tj, TJ, ' and ",
// along with Tf, which resolves the font they use. All of them need PDFont,
// which this port has not reached. See migration/STATUS.md.
package text

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// AddAll registers every processor in this package with the engine.
func AddAll(context *contentstream.PDFStreamEngine) {
	context.AddOperator(NewBeginText(context))
	context.AddOperator(NewEndText(context))
	context.AddOperator(NewMoveText(context))
	context.AddOperator(NewMoveTextSetLeading(context))
	context.AddOperator(NewNextLine(context))
	context.AddOperator(NewSetCharSpacing(context))
	context.AddOperator(NewSetTextHorizontalScaling(context))
	context.AddOperator(NewSetTextLeading(context))
	context.AddOperator(NewSetTextRenderingMode(context))
	context.AddOperator(NewSetTextRise(context))
	context.AddOperator(NewSetWordSpacing(context))
	context.AddOperator(NewSetFontAndSize(context))
	context.AddOperator(NewShowText(context))
	context.AddOperator(NewShowTextAdjusted(context))
	context.AddOperator(NewShowTextLine(context))
	context.AddOperator(NewShowTextLineAndSpace(context))
}

// BeginText is BT: begin a text object.
type BeginText struct {
	contentstream.BaseOperatorProcessor
}

// NewBeginText returns the BT processor.
func NewBeginText(context *contentstream.PDFStreamEngine) *BeginText {
	return &BeginText{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *BeginText) Name() string { return operator.BeginText }

// Process resets both text matrices to the identity.
func (p *BeginText) Process(op *operator.Operator, arguments []cos.Base) error {
	context := p.Context()
	context.SetTextMatrix(util.NewMatrix())
	context.SetTextLineMatrix(util.NewMatrix())
	return context.Overrides().BeginText()
}

// EndText is ET: end a text object.
type EndText struct {
	contentstream.BaseOperatorProcessor
}

// NewEndText returns the ET processor.
func NewEndText(context *contentstream.PDFStreamEngine) *EndText {
	return &EndText{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *EndText) Name() string { return operator.EndText }

// Process clears both text matrices.
func (p *EndText) Process(op *operator.Operator, arguments []cos.Base) error {
	context := p.Context()
	context.SetTextMatrix(nil)
	context.SetTextLineMatrix(nil)
	return context.Overrides().EndText()
}

// MoveText is Td: move to the start of the next line, offset from the start of
// the current one.
type MoveText struct {
	contentstream.BaseOperatorProcessor
}

// NewMoveText returns the Td processor.
func NewMoveText(context *contentstream.PDFStreamEngine) *MoveText {
	return &MoveText{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *MoveText) Name() string { return operator.MoveText }

// Process moves the text line matrix, and the text matrix with it.
func (p *MoveText) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) < 2 {
		return operator.MissingOperand(op, arguments)
	}
	context := p.Context()
	textLineMatrix := context.TextLineMatrix()
	if textLineMatrix == nil {
		slog.Warn("TextLineMatrix is null, operator will be ignored", "operator", p.Name())
		return nil
	}
	x, ok := arguments[0].(cos.Number)
	if !ok {
		return nil
	}
	y, ok := arguments[1].(cos.Number)
	if !ok {
		return nil
	}
	textLineMatrix.Concatenate(util.NewMatrixOf(1, 0, 0, 1, x.FloatValue(), y.FloatValue()))
	context.SetTextMatrix(textLineMatrix.Clone())
	return nil
}

// MoveTextSetLeading is TD: move to the start of the next line and set the
// leading to the negated vertical offset.
type MoveTextSetLeading struct {
	contentstream.BaseOperatorProcessor
}

// NewMoveTextSetLeading returns the TD processor.
func NewMoveTextSetLeading(context *contentstream.PDFStreamEngine) *MoveTextSetLeading {
	return &MoveTextSetLeading{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *MoveTextSetLeading) Name() string { return operator.MoveTextSetLeading }

// Process sets the leading and then moves, by running the two operators that
// do those things rather than repeating them.
func (p *MoveTextSetLeading) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) < 2 {
		return operator.MissingOperand(op, arguments)
	}
	// move text position and set leading
	y, ok := arguments[1].(cos.Number)
	if !ok {
		return nil
	}
	context := p.Context()
	leading := []cos.Base{cos.NewFloat(-y.FloatValue())}
	if err := context.ProcessOperatorNamed(operator.SetTextLeading, leading); err != nil {
		return err
	}
	return context.ProcessOperatorNamed(operator.MoveText, arguments)
}

// NextLine is T*: move to the start of the next line.
type NextLine struct {
	contentstream.BaseOperatorProcessor
}

// NewNextLine returns the T* processor.
func NewNextLine(context *contentstream.PDFStreamEngine) *NextLine {
	return &NextLine{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *NextLine) Name() string { return operator.NextLine }

// Process moves down by the leading, by running Td rather than repeating it.
func (p *NextLine) Process(op *operator.Operator, arguments []cos.Base) error {
	// move to start of next text line
	context := p.Context()
	// this must be -leading instead of just leading as written in the
	// specification (p.369) the acrobat reader seems to implement it the same way
	args := []cos.Base{
		cos.FloatZero,
		cos.NewFloat(-context.GraphicsState().TextState().Leading()),
	}
	// use Td instead of repeating code
	return context.ProcessOperatorNamed(operator.MoveText, args)
}

// SetCharSpacing is Tc: set the character spacing.
type SetCharSpacing struct {
	contentstream.BaseOperatorProcessor
}

// NewSetCharSpacing returns the Tc processor.
func NewSetCharSpacing(context *contentstream.PDFStreamEngine) *SetCharSpacing {
	return &SetCharSpacing{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetCharSpacing) Name() string { return operator.SetCharSpacing }

// Process sets the character spacing from the last operand rather than the
// first: there are some documents which are incorrectly structured, and have a
// wrong number of arguments to this.
func (p *SetCharSpacing) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return operator.MissingOperand(op, arguments)
	}
	if characterSpacing, ok := arguments[len(arguments)-1].(cos.Number); ok {
		p.Context().GraphicsState().TextState().
			SetCharacterSpacing(characterSpacing.FloatValue())
	}
	return nil
}

// SetTextHorizontalScaling is Tz: set the horizontal scaling, as a percentage.
type SetTextHorizontalScaling struct {
	contentstream.BaseOperatorProcessor
}

// NewSetTextHorizontalScaling returns the Tz processor.
func NewSetTextHorizontalScaling(context *contentstream.PDFStreamEngine) *SetTextHorizontalScaling {
	return &SetTextHorizontalScaling{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetTextHorizontalScaling) Name() string { return operator.SetTextHorizontalScaling }

// Process sets the horizontal scaling.
func (p *SetTextHorizontalScaling) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return operator.MissingOperand(op, arguments)
	}
	scaling, ok := arguments[0].(cos.Number)
	if !ok {
		return nil
	}
	p.Context().GraphicsState().TextState().SetHorizontalScaling(scaling.FloatValue())
	return nil
}

// SetTextLeading is TL: set the leading.
type SetTextLeading struct {
	contentstream.BaseOperatorProcessor
}

// NewSetTextLeading returns the TL processor.
func NewSetTextLeading(context *contentstream.PDFStreamEngine) *SetTextLeading {
	return &SetTextLeading{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetTextLeading) Name() string { return operator.SetTextLeading }

// Process sets the leading.
func (p *SetTextLeading) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return operator.MissingOperand(op, arguments)
	}
	leading, ok := arguments[0].(cos.Number)
	if !ok {
		return nil
	}
	p.Context().GraphicsState().TextState().SetLeading(leading.FloatValue())
	return nil
}

// SetTextRise is Ts: set the text rise.
type SetTextRise struct {
	contentstream.BaseOperatorProcessor
}

// NewSetTextRise returns the Ts processor.
func NewSetTextRise(context *contentstream.PDFStreamEngine) *SetTextRise {
	return &SetTextRise{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetTextRise) Name() string { return operator.SetTextRise }

// Process sets the text rise. Unlike most of its neighbours it says nothing
// about an operator with no operands at all, and simply does nothing.
func (p *SetTextRise) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return nil
	}
	rise, ok := arguments[0].(cos.Number)
	if !ok {
		return nil
	}
	p.Context().GraphicsState().TextState().SetRise(rise.FloatValue())
	return nil
}

// SetWordSpacing is Tw: set the word spacing.
type SetWordSpacing struct {
	contentstream.BaseOperatorProcessor
}

// NewSetWordSpacing returns the Tw processor.
func NewSetWordSpacing(context *contentstream.PDFStreamEngine) *SetWordSpacing {
	return &SetWordSpacing{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetWordSpacing) Name() string { return operator.SetWordSpacing }

// Process sets the word spacing. Like Ts and unlike most of its neighbours, an
// operator with no operands is simply ignored.
func (p *SetWordSpacing) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return nil
	}
	wordSpacing, ok := arguments[0].(cos.Number)
	if !ok {
		return nil
	}
	p.Context().GraphicsState().TextState().SetWordSpacing(wordSpacing.FloatValue())
	return nil
}

// SetTextRenderingMode is Tr: set the text rendering mode.
type SetTextRenderingMode struct {
	contentstream.BaseOperatorProcessor
}

// NewSetTextRenderingMode returns the Tr processor.
func NewSetTextRenderingMode(context *contentstream.PDFStreamEngine) *SetTextRenderingMode {
	return &SetTextRenderingMode{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetTextRenderingMode) Name() string { return operator.SetTextRenderingmode }

// Process sets the text rendering mode, ignoring a mode outside the eight the
// specification defines.
func (p *SetTextRenderingMode) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return operator.MissingOperand(op, arguments)
	}
	mode, ok := arguments[0].(cos.Number)
	if !ok {
		return nil
	}
	val := mode.IntValue()
	if val < 0 || val >= state.RenderingModeCount {
		return nil
	}
	p.Context().GraphicsState().TextState().SetRenderingMode(state.RenderingModeFromInt(val))
	return nil
}

// SetFontAndSize is Tf: set the font and the size it is set at.
type SetFontAndSize struct {
	contentstream.BaseOperatorProcessor
}

// NewSetFontAndSize returns the Tf processor.
func NewSetFontAndSize(context *contentstream.PDFStreamEngine) *SetFontAndSize {
	return &SetFontAndSize{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *SetFontAndSize) Name() string { return operator.SetFontAndSize }

// Process sets the font and its size.
func (p *SetFontAndSize) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) < 2 {
		return operator.MissingOperand(op, arguments)
	}
	base0 := arguments[0]
	base1 := arguments[1]
	fontName, ok := base0.(*cos.Name)
	if !ok {
		return nil
	}
	size, ok := base1.(cos.Number)
	if !ok {
		return nil
	}
	fontSize := size.FloatValue()

	context := p.Context()
	textState := context.GraphicsState().TextState()
	textState.SetFontSize(fontSize)

	// Get the font after the size has been set in case there is an exception
	// so that PDFBox will use a default font
	f, err := context.Resources().GetFont(fontName)
	if err != nil {
		return err
	}
	// a font that is not in the resources is left nil, which showText replaces
	// with the default font
	textState.SetFont(f)
	return nil
}

// ShowText is Tj: draw a string.
type ShowText struct {
	contentstream.BaseOperatorProcessor
}

// NewShowText returns the Tj processor.
func NewShowText(context *contentstream.PDFStreamEngine) *ShowText {
	return &ShowText{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *ShowText) Name() string { return operator.ShowText }

// Process draws the string.
func (p *ShowText) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		// ignore ( )Tj
		return nil
	}
	str, ok := arguments[0].(*cos.StringObj)
	if !ok {
		// ignore
		return nil
	}
	context := p.Context()
	if context.TextMatrix() == nil {
		// ignore: outside of BT...ET
		return nil
	}
	return context.ShowTextString(str.Bytes())
}

// ShowTextAdjusted is TJ: draw an array of strings, moving the pen by the
// numbers between them.
type ShowTextAdjusted struct {
	contentstream.BaseOperatorProcessor
}

// NewShowTextAdjusted returns the TJ processor.
func NewShowTextAdjusted(context *contentstream.PDFStreamEngine) *ShowTextAdjusted {
	return &ShowTextAdjusted{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *ShowTextAdjusted) Name() string { return operator.ShowTextAdjusted }

// Process draws the strings of the array.
func (p *ShowTextAdjusted) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return nil
	}
	array, ok := arguments[0].(*cos.Array)
	if !ok {
		return nil
	}
	context := p.Context()
	if context.TextMatrix() == nil {
		// ignore: outside of BT...ET
		return nil
	}
	return context.ShowTextStrings(array)
}

// ShowTextLine is ': move to the next line and draw a string.
type ShowTextLine struct {
	contentstream.BaseOperatorProcessor
}

// NewShowTextLine returns the ' processor.
func NewShowTextLine(context *contentstream.PDFStreamEngine) *ShowTextLine {
	return &ShowTextLine{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *ShowTextLine) Name() string { return operator.ShowTextLine }

// Process moves to the next line and draws the string.
func (p *ShowTextLine) Process(op *operator.Operator, arguments []cos.Base) error {
	context := p.Context()
	if err := context.ProcessOperatorNamed(operator.NextLine, nil); err != nil {
		return err
	}
	return context.ProcessOperatorNamed(operator.ShowText, arguments)
}

// ShowTextLineAndSpace is ": set the word and character spacing, move to the
// next line and draw a string.
type ShowTextLineAndSpace struct {
	contentstream.BaseOperatorProcessor
}

// NewShowTextLineAndSpace returns the " processor.
func NewShowTextLineAndSpace(context *contentstream.PDFStreamEngine) *ShowTextLineAndSpace {
	return &ShowTextLineAndSpace{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *ShowTextLineAndSpace) Name() string { return operator.ShowTextLineAndSpace }

// Process sets the two spacings, then defers to the ' operator.
func (p *ShowTextLineAndSpace) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) < 3 {
		return operator.MissingOperand(op, arguments)
	}
	context := p.Context()
	if err := context.ProcessOperatorNamed(operator.SetWordSpacing, arguments[0:1]); err != nil {
		return err
	}
	if err := context.ProcessOperatorNamed(operator.SetCharSpacing, arguments[1:2]); err != nil {
		return err
	}
	return context.ProcessOperatorNamed(operator.ShowTextLine, arguments[2:3])
}
