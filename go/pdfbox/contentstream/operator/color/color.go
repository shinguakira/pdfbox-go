// Package color holds the operator processors that set the colour and the
// colour space paint is made in.
//
// Port of org.apache.pdfbox.contentstream.operator.color. Java gives each
// processor a file of its own; they are a few lines each and eight of the
// thirteen are one overridden method, so the port keeps them together.
package color

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/pattern"
)

// AddAll registers every processor in this package with the engine.
func AddAll(context *contentstream.PDFStreamEngine) {
	context.AddOperator(NewSetStrokingColorSpace(context))
	context.AddOperator(NewSetNonStrokingColorSpace(context))
	context.AddOperator(NewSetStrokingDeviceCMYKColor(context))
	context.AddOperator(NewSetNonStrokingDeviceCMYKColor(context))
	context.AddOperator(NewSetNonStrokingDeviceRGBColor(context))
	context.AddOperator(NewSetStrokingDeviceRGBColor(context))
	context.AddOperator(NewSetNonStrokingDeviceGrayColor(context))
	context.AddOperator(NewSetStrokingDeviceGrayColor(context))
	context.AddOperator(NewSetStrokingColor(context))
	context.AddOperator(NewSetNonStrokingColor(context))
	context.AddOperator(NewSetStrokingColorN(context))
	context.AddOperator(NewSetNonStrokingColorN(context))
}

// colorTarget is the three abstract methods of SetColor: which half of the
// graphics state -- stroking or non-stroking -- the processor works on.
//
// Go has no virtual dispatch through an embedded struct, so SetColor is handed
// the half it is for rather than asking itself.
type colorTarget interface {
	// Color returns the colour currently set on this half.
	Color() *color.PDColor

	// SetColor sets the colour of this half.
	SetColor(c *color.PDColor)

	// ColorSpace returns the colour space of this half.
	ColorSpace() color.PDColorSpace

	// SetColorSpace sets the colour space of this half.
	//
	// Java has no such abstract method: the six device colour operators name
	// their half themselves, in the one line each of them overrides. The port
	// asks the target instead, so that the line stays written once.
	SetColorSpace(cs color.PDColorSpace)
}

// SetColor is the shared body of sc, scn, SC and SCN: read the operands as
// components of the current colour space and set the colour.
//
// Port of the abstract SetColor.
type SetColor struct {
	contentstream.BaseOperatorProcessor

	target colorTarget
}

// newSetColor returns the shared body working on the given half of the state.
func newSetColor(context *contentstream.PDFStreamEngine, target colorTarget) SetColor {
	return SetColor{
		BaseOperatorProcessor: contentstream.NewBaseOperatorProcessor(context),
		target:                target,
	}
}

// Process sets the colour from the operands.
func (p SetColor) Process(op *operator.Operator, arguments []cos.Base) error {
	colorSpace := p.target.ColorSpace()
	if _, isPattern := colorSpace.(*pattern.PDPattern); !isPattern {
		if len(arguments) < colorSpace.NumberOfComponents() {
			return operator.MissingOperand(op, arguments)
		}
		if !contentstream.AllOperandsAre(arguments, isNumber) {
			// PDFBOX-5851: set an invalid color because Pattern colorspace is
			// missing; this will produce transparency in PageDrawer
			p.target.SetColor(color.NewPDColorOfComponents(nil, nil))
			return nil
		}
	}
	array := cos.NewArrayOf(arguments)
	p.target.SetColor(color.NewPDColorOfCOSArray(array, colorSpace))
	return nil
}

// isNumber is the `instanceof COSNumber` these processors test with.
func isNumber(base cos.Base) bool {
	_, ok := base.(cos.Number)
	return ok
}

// strokingColor is the stroking half of the graphics state.
//
// Port of the three overridden methods of SetStrokingColor, which every
// stroking processor here inherits unchanged.
type strokingColor struct {
	context *contentstream.PDFStreamEngine
}

func (t strokingColor) Color() *color.PDColor { return t.context.GraphicsState().StrokingColor() }

func (t strokingColor) SetColor(c *color.PDColor) { t.context.GraphicsState().SetStrokingColor(c) }

func (t strokingColor) ColorSpace() color.PDColorSpace {
	return t.context.GraphicsState().StrokingColorSpace()
}

func (t strokingColor) SetColorSpace(cs color.PDColorSpace) {
	t.context.GraphicsState().SetStrokingColorSpace(cs)
}

// nonStrokingColor is the non-stroking half of the graphics state.
//
// Port of the three overridden methods of SetNonStrokingColor.
type nonStrokingColor struct {
	context *contentstream.PDFStreamEngine
}

func (t nonStrokingColor) Color() *color.PDColor { return t.context.GraphicsState().NonStrokingColor() }

func (t nonStrokingColor) SetColor(c *color.PDColor) {
	t.context.GraphicsState().SetNonStrokingColor(c)
}

func (t nonStrokingColor) ColorSpace() color.PDColorSpace {
	return t.context.GraphicsState().NonStrokingColorSpace()
}

func (t nonStrokingColor) SetColorSpace(cs color.PDColorSpace) {
	t.context.GraphicsState().SetNonStrokingColorSpace(cs)
}

// SetStrokingColor is SC: set the stroking colour.
type SetStrokingColor struct{ SetColor }

// NewSetStrokingColor returns the SC processor.
func NewSetStrokingColor(c *contentstream.PDFStreamEngine) *SetStrokingColor {
	return &SetStrokingColor{newSetColor(c, strokingColor{c})}
}

// Name returns the operator this processes.
func (p *SetStrokingColor) Name() string { return operator.StrokingColor }

// SetStrokingColorN is SCN: SC with pattern operands allowed, and the same body.
type SetStrokingColorN struct{ SetStrokingColor }

// NewSetStrokingColorN returns the SCN processor.
func NewSetStrokingColorN(c *contentstream.PDFStreamEngine) *SetStrokingColorN {
	return &SetStrokingColorN{*NewSetStrokingColor(c)}
}

// Name returns the operator this processes.
func (p *SetStrokingColorN) Name() string { return operator.StrokingColorN }

// SetNonStrokingColor is sc: set the non-stroking colour.
type SetNonStrokingColor struct{ SetColor }

// NewSetNonStrokingColor returns the sc processor.
func NewSetNonStrokingColor(c *contentstream.PDFStreamEngine) *SetNonStrokingColor {
	return &SetNonStrokingColor{newSetColor(c, nonStrokingColor{c})}
}

// Name returns the operator this processes.
func (p *SetNonStrokingColor) Name() string { return operator.NonStrokingColor }

// SetNonStrokingColorN is scn: sc with pattern operands allowed, and the same
// body.
type SetNonStrokingColorN struct{ SetNonStrokingColor }

// NewSetNonStrokingColorN returns the scn processor.
func NewSetNonStrokingColorN(c *contentstream.PDFStreamEngine) *SetNonStrokingColorN {
	return &SetNonStrokingColorN{*NewSetNonStrokingColor(c)}
}

// Name returns the operator this processes.
func (p *SetNonStrokingColorN) Name() string { return operator.NonStrokingColorN }

// setDeviceColorSpace is the body the six device colour operators share: name
// the device colour space, install it, then run the plain colour operator.
//
// Java writes it out in each of the six, differing only in the COSName and in
// which half of the state it sets.
func setDeviceColorSpace(
	context *contentstream.PDFStreamEngine, target colorTarget, deviceName *cos.Name,
	op *operator.Operator, arguments []cos.Base, plain SetColor) error {
	if !context.ShouldProcessColorOperators() {
		return nil
	}
	cs, err := context.Resources().ColorSpace(deviceName)
	if err != nil {
		return err
	}
	target.SetColorSpace(cs)
	return plain.Process(op, arguments)
}

// SetStrokingDeviceGrayColor is G: set the stroking colour in DeviceGray.
type SetStrokingDeviceGrayColor struct{ SetStrokingColor }

// NewSetStrokingDeviceGrayColor returns the G processor.
func NewSetStrokingDeviceGrayColor(c *contentstream.PDFStreamEngine) *SetStrokingDeviceGrayColor {
	return &SetStrokingDeviceGrayColor{*NewSetStrokingColor(c)}
}

// Name returns the operator this processes.
func (p *SetStrokingDeviceGrayColor) Name() string { return operator.StrokingColorGray }

// Process installs DeviceGray and sets the colour.
func (p *SetStrokingDeviceGrayColor) Process(op *operator.Operator, arguments []cos.Base) error {
	return setDeviceColorSpace(p.Context(), p.target, cos.DeviceGray, op, arguments, p.SetColor)
}

// SetStrokingDeviceRGBColor is RG: set the stroking colour in DeviceRGB.
type SetStrokingDeviceRGBColor struct{ SetStrokingColor }

// NewSetStrokingDeviceRGBColor returns the RG processor.
func NewSetStrokingDeviceRGBColor(c *contentstream.PDFStreamEngine) *SetStrokingDeviceRGBColor {
	return &SetStrokingDeviceRGBColor{*NewSetStrokingColor(c)}
}

// Name returns the operator this processes.
func (p *SetStrokingDeviceRGBColor) Name() string { return operator.StrokingColorRgb }

// Process installs DeviceRGB and sets the colour.
func (p *SetStrokingDeviceRGBColor) Process(op *operator.Operator, arguments []cos.Base) error {
	return setDeviceColorSpace(p.Context(), p.target, cos.DeviceRGB, op, arguments, p.SetColor)
}

// SetStrokingDeviceCMYKColor is K: set the stroking colour in DeviceCMYK.
type SetStrokingDeviceCMYKColor struct{ SetStrokingColor }

// NewSetStrokingDeviceCMYKColor returns the K processor.
func NewSetStrokingDeviceCMYKColor(c *contentstream.PDFStreamEngine) *SetStrokingDeviceCMYKColor {
	return &SetStrokingDeviceCMYKColor{*NewSetStrokingColor(c)}
}

// Name returns the operator this processes.
func (p *SetStrokingDeviceCMYKColor) Name() string { return operator.StrokingColorCmyk }

// Process installs DeviceCMYK and sets the colour.
func (p *SetStrokingDeviceCMYKColor) Process(op *operator.Operator, arguments []cos.Base) error {
	return setDeviceColorSpace(p.Context(), p.target, cos.DeviceCMYK, op, arguments, p.SetColor)
}

// SetNonStrokingDeviceGrayColor is g: set the non-stroking colour in DeviceGray.
type SetNonStrokingDeviceGrayColor struct{ SetNonStrokingColor }

// NewSetNonStrokingDeviceGrayColor returns the g processor.
func NewSetNonStrokingDeviceGrayColor(
	c *contentstream.PDFStreamEngine) *SetNonStrokingDeviceGrayColor {
	return &SetNonStrokingDeviceGrayColor{*NewSetNonStrokingColor(c)}
}

// Name returns the operator this processes.
func (p *SetNonStrokingDeviceGrayColor) Name() string { return operator.NonStrokingGray }

// Process installs DeviceGray and sets the colour.
func (p *SetNonStrokingDeviceGrayColor) Process(op *operator.Operator, arguments []cos.Base) error {
	return setDeviceColorSpace(p.Context(), p.target, cos.DeviceGray, op, arguments, p.SetColor)
}

// SetNonStrokingDeviceRGBColor is rg: set the non-stroking colour in DeviceRGB.
type SetNonStrokingDeviceRGBColor struct{ SetNonStrokingColor }

// NewSetNonStrokingDeviceRGBColor returns the rg processor.
func NewSetNonStrokingDeviceRGBColor(
	c *contentstream.PDFStreamEngine) *SetNonStrokingDeviceRGBColor {
	return &SetNonStrokingDeviceRGBColor{*NewSetNonStrokingColor(c)}
}

// Name returns the operator this processes.
func (p *SetNonStrokingDeviceRGBColor) Name() string { return operator.NonStrokingRgb }

// Process installs DeviceRGB and sets the colour.
func (p *SetNonStrokingDeviceRGBColor) Process(op *operator.Operator, arguments []cos.Base) error {
	return setDeviceColorSpace(p.Context(), p.target, cos.DeviceRGB, op, arguments, p.SetColor)
}

// SetNonStrokingDeviceCMYKColor is k: set the non-stroking colour in DeviceCMYK.
type SetNonStrokingDeviceCMYKColor struct{ SetNonStrokingColor }

// NewSetNonStrokingDeviceCMYKColor returns the k processor.
func NewSetNonStrokingDeviceCMYKColor(
	c *contentstream.PDFStreamEngine) *SetNonStrokingDeviceCMYKColor {
	return &SetNonStrokingDeviceCMYKColor{*NewSetNonStrokingColor(c)}
}

// Name returns the operator this processes.
func (p *SetNonStrokingDeviceCMYKColor) Name() string { return operator.NonStrokingCmyk }

// Process installs DeviceCMYK and sets the colour.
func (p *SetNonStrokingDeviceCMYKColor) Process(op *operator.Operator, arguments []cos.Base) error {
	return setDeviceColorSpace(p.Context(), p.target, cos.DeviceCMYK, op, arguments, p.SetColor)
}

// SetStrokingColorSpace is CS: set the stroking colour space, and with it the
// initial colour of that space.
type SetStrokingColorSpace struct {
	contentstream.BaseOperatorProcessor
}

// NewSetStrokingColorSpace returns the CS processor.
func NewSetStrokingColorSpace(c *contentstream.PDFStreamEngine) *SetStrokingColorSpace {
	return &SetStrokingColorSpace{contentstream.NewBaseOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *SetStrokingColorSpace) Name() string { return operator.StrokingColorspace }

// Process sets the colour space named by the operand.
func (p *SetStrokingColorSpace) Process(_ *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return nil
	}
	name, isName := arguments[0].(*cos.Name)
	if !isName {
		return nil
	}
	context := p.Context()
	if !context.ShouldProcessColorOperators() {
		return nil
	}
	cs, err := context.Resources().ColorSpace(name)
	if err != nil {
		return err
	}
	context.GraphicsState().SetStrokingColorSpace(cs)
	context.GraphicsState().SetStrokingColor(cs.InitialColor())
	return nil
}

// SetNonStrokingColorSpace is cs: set the non-stroking colour space, and with it
// the initial colour of that space.
type SetNonStrokingColorSpace struct {
	contentstream.BaseOperatorProcessor
}

// NewSetNonStrokingColorSpace returns the cs processor.
func NewSetNonStrokingColorSpace(c *contentstream.PDFStreamEngine) *SetNonStrokingColorSpace {
	return &SetNonStrokingColorSpace{contentstream.NewBaseOperatorProcessor(c)}
}

// Name returns the operator this processes.
func (p *SetNonStrokingColorSpace) Name() string { return operator.NonStrokingColorspace }

// Process sets the colour space named by the operand.
func (p *SetNonStrokingColorSpace) Process(_ *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return nil
	}
	name, isName := arguments[0].(*cos.Name)
	if !isName {
		return nil
	}
	context := p.Context()
	if !context.ShouldProcessColorOperators() {
		return nil
	}
	cs, err := context.Resources().ColorSpace(name)
	if err != nil {
		return err
	}
	context.GraphicsState().SetNonStrokingColorSpace(cs)
	context.GraphicsState().SetNonStrokingColor(cs.InitialColor())
	return nil
}
