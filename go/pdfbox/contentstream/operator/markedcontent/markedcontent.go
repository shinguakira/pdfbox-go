// Package markedcontent holds the operator processors for marked content.
//
// Port of org.apache.pdfbox.contentstream.operator.markedcontent. Java gives
// each processor a file of its own; they are a few lines each, so the port
// keeps them together.
//
// DrawObject lives in this package in Java although it draws an XObject,
// because it is the one the marked content extractor registers: it records the
// XObject in the sequence being collected on the way past.
package markedcontent

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// AddAll registers every processor in this package with the engine.
func AddAll(context *contentstream.PDFStreamEngine) {
	AddSequenceOperators(context)
	context.AddOperator(NewMarkedContentPoint(context))
	context.AddOperator(NewMarkedContentPointWithProperties(context))
}

// AddSequenceOperators registers the three that begin and end a marked content
// sequence, and not the two that mark a point.
//
// That is the set PDFGraphicsStreamEngine and PDFTextStripper each name in
// their constructors; nothing in PDFBox registers the marked content points.
func AddSequenceOperators(context *contentstream.PDFStreamEngine) {
	context.AddOperator(NewBeginMarkedContentSequence(context))
	context.AddOperator(NewBeginMarkedContentSequenceWithProperties(context))
	context.AddOperator(NewEndMarkedContentSequence(context))
}

// propertiesOf reads the second operand of a BDC or DP as a property
// dictionary, returning nil where it is neither a dictionary nor a name the
// resources hold a property list under.
//
// Java writes the two branches out in each of the two processors; they are the
// same seven lines, so the port has them once.
func propertiesOf(resources *pdmodel.PDResources, op1 cos.Base) *cos.Dictionary {
	if name, isName := op1.(*cos.Name); isName {
		// PDFBOX-5980 and SO79549651
		if prop := resources.GetProperties(name); prop != nil {
			return prop.PropertyDictionary()
		}
		return nil
	}
	if dict, isDictionary := op1.(*cos.Dictionary); isDictionary {
		return dict
	}
	return nil
}

// BeginMarkedContentSequence is BMC: begin a marked content sequence.
type BeginMarkedContentSequence struct {
	contentstream.BaseOperatorProcessor
}

// NewBeginMarkedContentSequence returns the BMC processor.
func NewBeginMarkedContentSequence(context *contentstream.PDFStreamEngine) *BeginMarkedContentSequence {
	return &BeginMarkedContentSequence{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *BeginMarkedContentSequence) Name() string { return operator.BeginMarkedContent }

// Process begins the sequence. The tag is the last name among the operands,
// rather than the first, which is what Java's loop leaves behind.
func (p *BeginMarkedContentSequence) Process(op *operator.Operator, arguments []cos.Base) error {
	var tag *cos.Name
	for _, argument := range arguments {
		if name, ok := argument.(*cos.Name); ok {
			tag = name
		}
	}
	p.Context().Overrides().BeginMarkedContentSequence(tag, nil)
	return nil
}

// BeginMarkedContentSequenceWithProperties is BDC: begin a marked content
// sequence with a property list.
type BeginMarkedContentSequenceWithProperties struct {
	contentstream.BaseOperatorProcessor
}

// NewBeginMarkedContentSequenceWithProperties returns the BDC processor.
func NewBeginMarkedContentSequenceWithProperties(context *contentstream.PDFStreamEngine) *BeginMarkedContentSequenceWithProperties {
	return &BeginMarkedContentSequenceWithProperties{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *BeginMarkedContentSequenceWithProperties) Name() string {
	return operator.BeginMarkedContentSeq
}

// Process begins the sequence, doing nothing when the properties are of the
// wrong type or cannot be found.
func (p *BeginMarkedContentSequenceWithProperties) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) < 2 {
		return operator.MissingOperand(op, operands)
	}
	tag, ok := operands[0].(*cos.Name)
	if !ok {
		return nil
	}
	propDict := propertiesOf(p.Context().Resources(), operands[1])
	if propDict == nil {
		// wrong type or property not found
		return nil
	}
	p.Context().Overrides().BeginMarkedContentSequence(tag, propDict)
	return nil
}

// EndMarkedContentSequence is EMC: end a marked content sequence.
type EndMarkedContentSequence struct {
	contentstream.BaseOperatorProcessor
}

// NewEndMarkedContentSequence returns the EMC processor.
func NewEndMarkedContentSequence(context *contentstream.PDFStreamEngine) *EndMarkedContentSequence {
	return &EndMarkedContentSequence{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *EndMarkedContentSequence) Name() string { return operator.EndMarkedContent }

// Process ends the sequence.
func (p *EndMarkedContentSequence) Process(op *operator.Operator, arguments []cos.Base) error {
	p.Context().Overrides().EndMarkedContentSequence()
	return nil
}

// MarkedContentPoint is MP: a marked content point.
type MarkedContentPoint struct {
	contentstream.BaseOperatorProcessor
}

// NewMarkedContentPoint returns the MP processor.
func NewMarkedContentPoint(context *contentstream.PDFStreamEngine) *MarkedContentPoint {
	return &MarkedContentPoint{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *MarkedContentPoint) Name() string { return operator.MarkedContentPoint }

// Process marks the point.
func (p *MarkedContentPoint) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) == 0 {
		return operator.MissingOperand(op, operands)
	}
	tag, ok := operands[0].(*cos.Name)
	if !ok {
		return nil
	}
	p.Context().Overrides().MarkedContentPoint(tag, nil)
	return nil
}

// MarkedContentPointWithProperties is DP: a marked content point with a
// property list.
type MarkedContentPointWithProperties struct {
	contentstream.BaseOperatorProcessor
}

// NewMarkedContentPointWithProperties returns the DP processor.
func NewMarkedContentPointWithProperties(context *contentstream.PDFStreamEngine) *MarkedContentPointWithProperties {
	return &MarkedContentPointWithProperties{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *MarkedContentPointWithProperties) Name() string {
	return operator.MarkedContentPointWithProps
}

// Process marks the point, doing nothing when the properties are of the wrong
// type or cannot be found.
func (p *MarkedContentPointWithProperties) Process(op *operator.Operator, operands []cos.Base) error {
	if len(operands) < 2 {
		return operator.MissingOperand(op, operands)
	}
	tag, ok := operands[0].(*cos.Name)
	if !ok {
		return nil
	}
	propDict := propertiesOf(p.Context().Resources(), operands[1])
	if propDict == nil {
		// wrong type or property not found
		return nil
	}
	p.Context().Overrides().MarkedContentPoint(tag, propDict)
	return nil
}

// xobjectSink is what the marked content DrawObject hands the XObject to.
//
// Java casts the context to PDFMarkedContentExtractor, which lives in
// pdfbox/text; that package imports this one, so the port names the one method
// it calls instead.
type xobjectSink interface {
	XObject(xobject any)
}

// DrawObject is Do: record the XObject in the sequence being collected, and
// walk into it where it is a form.
//
// Port of the markedcontent DrawObject, which is the one PDFMarkedContentExtractor
// registers. It differs from the other two in recording the XObject, and in not
// stepping over an image.
type DrawObject struct {
	contentstream.BaseOperatorProcessor
}

// NewDrawObject returns the Do processor for the marked content extractor.
func NewDrawObject(context *contentstream.PDFStreamEngine) *DrawObject {
	return &DrawObject{contentstream.NewBaseOperatorProcessor(context)}
}

// Name returns the operator this processes.
func (p *DrawObject) Name() string { return operator.DrawObject }

// Process records the XObject and walks into it where it is a form.
func (p *DrawObject) Process(op *operator.Operator, arguments []cos.Base) error {
	if len(arguments) == 0 {
		return operator.MissingOperand(op, arguments)
	}
	name, isName := arguments[0].(*cos.Name)
	if !isName {
		return nil
	}
	context := p.Context()
	xobject, err := context.Resources().GetXObject(name)
	if err != nil {
		return err
	}
	if sink, isSink := context.Overrides().(xobjectSink); isSink {
		sink.XObject(xobject)
	}
	return contentstream.ShowFormXObject(context, xobject)
}
