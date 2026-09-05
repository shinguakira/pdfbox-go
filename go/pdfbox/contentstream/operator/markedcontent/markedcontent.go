// Package markedcontent holds the operator processors for marked content.
//
// Port of org.apache.pdfbox.contentstream.operator.markedcontent. Java gives
// each processor a file of its own; they are a few lines each, so the port
// keeps them together.
//
// DrawObject, which lives in this package in Java although it draws an XObject,
// is not here: it needs PDXObject. Where a properties operand names a property
// list in the resources rather than giving one inline, that lookup is missing
// too, since PDResources cannot resolve one yet. See migration/STATUS.md.
package markedcontent

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
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
// dictionary, returning nil when it is neither a dictionary nor something this
// port can resolve.
func propertiesOf(op1 cos.Base) *cos.Dictionary {
	if dict, ok := op1.(*cos.Dictionary); ok {
		return dict
	}
	// PDFBOX-5980 and SO79549651: a name here refers to a property list in the
	// resources. Resolving it needs PDResources.getProperties and the
	// PDPropertyList it returns, neither of which is ported.
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
	propDict := propertiesOf(operands[1])
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
	propDict := propertiesOf(operands[1])
	if propDict == nil {
		// wrong type or property not found
		return nil
	}
	p.Context().Overrides().MarkedContentPoint(tag, propDict)
	return nil
}
