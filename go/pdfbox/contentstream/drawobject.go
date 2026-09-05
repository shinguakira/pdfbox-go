package contentstream

// Do, for an engine that reads rather than draws.
//
// Port of org.apache.pdfbox.contentstream.operator.DrawObject, the one of the
// three the text extractor registers: it walks into a form XObject so that the
// text inside is seen, and steps over an image, which it has no use for.
//
// Java has it in contentstream/operator. The port's operator package holds no
// processors -- a processor names the engine, and the engine's package imports
// operator -- so it lives here, one level up, with the engine it names.

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
)

// DrawObject is Do: walk into the named form XObject, and ignore an image.
type DrawObject struct {
	BaseOperatorProcessor
}

// NewDrawObject returns the Do processor for an engine that does not draw.
func NewDrawObject(context *PDFStreamEngine) *DrawObject {
	return &DrawObject{NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *DrawObject) Name() string { return operator.DrawObject }

// Process walks into the XObject where it is a form.
func (p *DrawObject) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return operator.MissingOperand(op, arguments)
	}
	name, isName := arguments[0].(*cos.Name)
	if !isName {
		return nil
	}
	context := p.Context()
	if context.Resources().IsImageXObject(name) {
		// we're done here, don't decode images when doing text extraction
		return nil
	}
	xobject, err := context.Resources().GetXObject(name)
	if err != nil {
		return err
	}
	return ShowFormXObject(context, xobject)
}

// ShowFormXObject runs the given XObject's content stream where it is a form,
// and does nothing where it is not.
//
// Java writes the same twelve lines in each of the three DrawObject
// processors; the port has them once, since all three reach an engine.
func ShowFormXObject(context *PDFStreamEngine, xobject any) error {
	switch object := xobject.(type) {
	case *form.PDTransparencyGroup:
		context.IncreaseLevel()
		defer context.DecreaseLevel()
		if context.Level() > 50 {
			slog.Error("contentstream: recursion is too deep, skipping form XObject")
			return nil
		}
		return context.Overrides().ShowTransparencyGroup(object)
	case *form.PDFormXObject:
		context.IncreaseLevel()
		defer context.DecreaseLevel()
		if context.Level() > 50 {
			slog.Error("contentstream: recursion is too deep, skipping form XObject")
			return nil
		}
		return context.Overrides().ShowForm(object)
	}
	return nil
}
