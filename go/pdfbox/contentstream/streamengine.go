package contentstream

import (
	"compress/flate"
	"compress/zlib"
	"errors"
	"log/slog"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/contentstream/operator"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// OperatorProcessor processes one PDF operator.
//
// Port of org.apache.pdfbox.contentstream.operator.OperatorProcessor. Java
// keeps it beside Operator, but a processor holds the engine and the engine
// holds processors, which in Go is a package cycle; the interface lives here,
// with the engine, and the concrete processors stay in the subpackages Java
// puts them in.
type OperatorProcessor interface {
	// Process runs the operator against the operands it was given.
	Process(op *operator.Operator, operands []cos.Base) error

	// Name returns the name of the operator this processes, such as "BI".
	Name() string
}

// BaseOperatorProcessor holds the engine a processor runs against. A processor
// embeds it and adds Process and Name.
//
// Port of the state and constructor of OperatorProcessor.
type BaseOperatorProcessor struct {
	context *PDFStreamEngine
}

// NewBaseOperatorProcessor returns the shared part of a processor running
// against the given engine.
func NewBaseOperatorProcessor(context *PDFStreamEngine) BaseOperatorProcessor {
	return BaseOperatorProcessor{context: context}
}

// Context returns the processing context.
func (p BaseOperatorProcessor) Context() *PDFStreamEngine { return p.context }

// AllOperandsAre reports whether every operand satisfies the given test, which
// is how a processor checks that an array holds what it expects.
//
// Port of checkArrayTypesClass. Java takes a Class and asks isInstance; Go has
// no such value, so the caller passes the test as a function — typically a type
// assertion.
func AllOperandsAre(operands []cos.Base, test func(cos.Base) bool) bool {
	for _, operand := range operands {
		if !test(operand) {
			return false
		}
	}
	return true
}

// StreamEngineOverrides is the set of PDFStreamEngine methods a subclass
// overrides. Embedding gives no virtual dispatch in Go, so an engine calls
// these through an interface it is handed rather than through itself; see
// SetOverrides.
type StreamEngineOverrides interface {
	// BeginText is called when the BT operator is encountered.
	BeginText() error

	// EndText is called when the ET operator is encountered.
	EndText() error

	// UnsupportedOperator is called when an operator with no processor is
	// encountered.
	UnsupportedOperator(op *operator.Operator, operands []cos.Base) error

	// BeginMarkedContentSequence handles the BMC and BDC operators.
	BeginMarkedContentSequence(tag *cos.Name, properties *cos.Dictionary)

	// EndMarkedContentSequence handles the EMC operator.
	EndMarkedContentSequence()

	// MarkedContentPoint handles the MP and DP operators.
	MarkedContentPoint(tag *cos.Name, properties *cos.Dictionary)

	// ShowGlyph is called for each glyph the engine decodes.
	ShowGlyph(textRenderingMatrix *util.Matrix, f font.PDFont, code int, displacement util.Vector) error

	// ShowFontGlyph is called for each glyph of a font that is not Type 3.
	ShowFontGlyph(textRenderingMatrix *util.Matrix, f font.PDFont, code int, displacement util.Vector) error

	// ShowType3Glyph is called for each glyph of a Type 3 font.
	ShowType3Glyph(textRenderingMatrix *util.Matrix, f *font.PDType3Font, code int, displacement util.Vector) error

	// ApplyTextAdjustment moves the pen by the amounts a TJ array asks for.
	ApplyTextAdjustment(tx, ty float32)
}

// PDFStreamEngine walks a content stream and hands each operator to the
// processor registered for it, keeping the graphics state as it goes.
//
// Port of org.apache.pdfbox.contentstream.PDFStreamEngine.
//
// What is missing is everything that draws or measures: showText and the glyph
// hooks, the default font, forms and transparency groups, Type 3 char procs,
// tiling patterns and annotations. Each needs a font, an XObject or a pattern,
// none of which this port has reached. Two consequences are worth naming:
// shouldProcessColorOperators is always true here, because the two cases that
// clear it are an uncoloured tiling pattern and a Type 3 char proc beginning
// with d1; and processChildStream, which forms and annotations reach the engine
// through, is absent with them. See migration/STATUS.md.
//
// An engine is not safe for concurrent use.
type PDFStreamEngine struct {
	operators     map[string]OperatorProcessor
	graphicsStack []*state.PDGraphicsState

	resources        *pdmodel.PDResources
	currentPage      *pdmodel.PDPage
	isProcessingPage bool
	initialMatrix    *util.Matrix

	// level counts how deep a potentially recursive operation has gone.
	level int

	shouldProcessColorOperators bool

	// overrides is what the engine calls instead of its own hooks. It is the
	// engine itself until SetOverrides is called.
	overrides StreamEngineOverrides

	// defaultFont is what the engine draws with where the content stream set no
	// font, read the first time one is needed.
	defaultFont font.PDFont
}

var _ StreamEngineOverrides = (*PDFStreamEngine)(nil)

// NewPDFStreamEngine returns an engine with no operators registered and no
// overrides, so that every hook does nothing.
func NewPDFStreamEngine() *PDFStreamEngine {
	e := &PDFStreamEngine{operators: map[string]OperatorProcessor{}}
	e.overrides = e
	return e
}

// SetOverrides installs the implementation whose hooks the engine should call.
// An embedder passes itself here, which is what stands in for Java's dynamic
// dispatch from the superclass into the subclass.
func (e *PDFStreamEngine) SetOverrides(overrides StreamEngineOverrides) {
	e.overrides = overrides
}

// AddOperator adds an operator processor to the engine.
func (e *PDFStreamEngine) AddOperator(op OperatorProcessor) {
	e.operators[op.Name()] = op
}

// initPage initializes the stream engine for the given page.
func (e *PDFStreamEngine) initPage(page *pdmodel.PDPage) {
	if page == nil {
		panic("contentstream: page cannot be null")
	}
	e.currentPage = page
	e.graphicsStack = []*state.PDGraphicsState{state.NewPDGraphicsState(page.CropBox())}
	e.resources = nil
	e.initialMatrix = page.Matrix()
}

// ProcessPage initializes the engine and processes the contents of the page.
func (e *PDFStreamEngine) ProcessPage(page *pdmodel.PDPage) error {
	e.initPage(page)
	if page.HasContents() {
		e.isProcessingPage = true
		err := e.processStream(page)
		e.isProcessingPage = false
		return err
	}
	return nil
}

// processStream processes a content stream.
func (e *PDFStreamEngine) processStream(contentStream PDContentStream) error {
	parent := e.pushResources(contentStream)
	savedStack := e.SaveGraphicsStack()
	parentMatrix := e.initialMatrix
	graphicsState := e.GraphicsState()

	// transform the CTM using the stream's matrix
	graphicsState.CurrentTransformationMatrix().Concatenate(contentStream.Matrix())

	// the stream's initial matrix includes the parent CTM, e.g. this allows a scaled form
	e.initialMatrix = graphicsState.CurrentTransformationMatrix().Clone()

	// clip to bounding box
	e.clipToRect(contentStream.BBox())

	err := e.processStreamOperators(contentStream)

	e.initialMatrix = parentMatrix
	e.RestoreGraphicsStack(savedStack)
	e.popResources(parent)
	return err
}

// processStreamOperators processes the operators of the given content stream.
func (e *PDFStreamEngine) processStreamOperators(contentStream PDContentStream) error {
	var arguments []cos.Base

	content, err := ContentsForStreamParsing(contentStream)
	if err != nil {
		return err
	}
	parser, err := pdfparser.NewStreamTokenParserSource(content)
	if err != nil {
		return err
	}

	oldShouldProcessColorOperators := e.shouldProcessColorOperators
	e.shouldProcessColorOperators = true
	defer func() { e.shouldProcessColorOperators = oldShouldProcessColorOperators }()

	for {
		token, err := parser.ParseNextToken()
		if err != nil {
			return err
		}
		if token == nil {
			return nil
		}
		if op, ok := token.(*operator.Operator); ok {
			if err := e.ProcessOperator(op, arguments); err != nil {
				return err
			}
			arguments = arguments[:0]
			continue
		}
		if base, ok := token.(cos.Base); ok {
			arguments = append(arguments, base)
		}
	}
}

// pushResources pushes the given stream's resources, returning the previous
// resources.
func (e *PDFStreamEngine) pushResources(contentStream PDContentStream) *pdmodel.PDResources {
	// resource lookup: first look for stream resources, then fallback to the current page
	parentResources := e.resources
	streamResources := contentStream.Resources()
	switch {
	case streamResources != nil:
		e.resources = streamResources
	case e.resources != nil:
		// inherit directly from parent stream, this is not in the PDF spec, but
		// the file from PDFBOX-1359 does this and works in Acrobat
	default:
		e.resources = e.currentPage.Resources()

		// resources are required in PDF
		if e.resources == nil {
			e.resources = pdmodel.NewPDResources()
		}
	}
	return parentResources
}

// popResources pops the current resources, replacing them with the given ones.
func (e *PDFStreamEngine) popResources(parentResources *pdmodel.PDResources) {
	e.resources = parentResources
}

// clipToRect transforms the given rectangle using the CTM and then intersects
// it with the current clipping area.
func (e *PDFStreamEngine) clipToRect(rectangle *common.PDRectangle) {
	if rectangle != nil {
		graphicsState := e.GraphicsState()
		clip := rectangle.Transform(graphicsState.CurrentTransformationMatrix())
		graphicsState.IntersectClippingPath(clip)
	}
}

// BeginText is called when the BT operator is encountered. It does nothing;
// an engine that cares installs its own through SetOverrides.
func (e *PDFStreamEngine) BeginText() error { return nil }

// EndText is called when the ET operator is encountered. It does nothing.
func (e *PDFStreamEngine) EndText() error { return nil }

// BeginMarkedContentSequence handles the BMC and BDC operators. It does
// nothing.
func (e *PDFStreamEngine) BeginMarkedContentSequence(tag *cos.Name, properties *cos.Dictionary) {}

// EndMarkedContentSequence handles the EMC operator. It does nothing.
func (e *PDFStreamEngine) EndMarkedContentSequence() {}

// MarkedContentPoint handles the MP and DP operators. It does nothing.
func (e *PDFStreamEngine) MarkedContentPoint(tag *cos.Name, properties *cos.Dictionary) {}

// UnsupportedOperator is called when an operator with no processor is
// encountered. It does nothing.
func (e *PDFStreamEngine) UnsupportedOperator(op *operator.Operator, operands []cos.Base) error {
	return nil
}

// Overrides returns what the engine calls instead of its own hooks, so that a
// processor can reach them.
func (e *PDFStreamEngine) Overrides() StreamEngineOverrides { return e.overrides }

// ProcessOperatorNamed handles the operator with the given name.
func (e *PDFStreamEngine) ProcessOperatorNamed(operation string, arguments []cos.Base) error {
	op, err := operator.GetChecked(operation)
	if err != nil {
		return err
	}
	return e.ProcessOperator(op, arguments)
}

// ProcessOperator handles one operator.
func (e *PDFStreamEngine) ProcessOperator(op *operator.Operator, operands []cos.Base) error {
	processor, ok := e.operators[op.Name()]
	if !ok {
		return e.overrides.UnsupportedOperator(op, operands)
	}
	if err := processor.Process(op, operands); err != nil {
		return e.OperatorException(op, operands, err)
	}
	return nil
}

// OperatorException is called when a processor reports an error. Errors that
// mean one operator could not be carried out are logged and swallowed, so that
// the rest of the stream is still walked; anything else ends the walk.
func (e *PDFStreamEngine) OperatorException(op *operator.Operator, operands []cos.Base, err error) error {
	switch {
	case errors.Is(err, operator.ErrMissingOperand),
		errors.Is(err, pdmodel.ErrMissingResource):
		slog.Error(err.Error())
	case errors.Is(err, operator.ErrEmptyGraphicsStack):
		slog.Warn(err.Error())
	case op.Name() == "Do":
		// todo: this too forgiving, but PDFBox has always worked this way for
		//       DrawObject; some careful refactoring is needed
		slog.Warn(err.Error())
	case isDataFormatError(err):
		slog.Warn(err.Error())
	default:
		return err
	}
	return nil
}

// isDataFormatError reports whether err is the Go equivalent of the
// java.util.zip.DataFormatException Java looks for behind an operator's
// exception: a compressed stream that will not inflate.
func isDataFormatError(err error) bool {
	var corrupt flate.CorruptInputError
	var internal flate.InternalError
	return errors.As(err, &corrupt) || errors.As(err, &internal) ||
		errors.Is(err, zlib.ErrHeader) || errors.Is(err, zlib.ErrChecksum) ||
		errors.Is(err, zlib.ErrDictionary)
}

// SaveGraphicsState pushes the current graphics state to the stack.
func (e *PDFStreamEngine) SaveGraphicsState() {
	e.graphicsStack = append(e.graphicsStack, e.GraphicsState().Clone())
}

// RestoreGraphicsState pops the current graphics state from the stack.
func (e *PDFStreamEngine) RestoreGraphicsState() {
	e.graphicsStack = e.graphicsStack[:len(e.graphicsStack)-1]
}

// SaveGraphicsStack saves the entire graphics stack, leaving the engine with a
// stack holding a copy of the current state.
func (e *PDFStreamEngine) SaveGraphicsStack() []*state.PDGraphicsState {
	savedStack := e.graphicsStack
	e.graphicsStack = []*state.PDGraphicsState{savedStack[len(savedStack)-1].Clone()}
	return savedStack
}

// RestoreGraphicsStack restores the entire graphics stack.
func (e *PDFStreamEngine) RestoreGraphicsStack(snapshot []*state.PDGraphicsState) {
	e.graphicsStack = snapshot
}

// GraphicsStackSize returns the size of the graphics stack.
func (e *PDFStreamEngine) GraphicsStackSize() int { return len(e.graphicsStack) }

// GraphicsState returns the current graphics state.
func (e *PDFStreamEngine) GraphicsState() *state.PDGraphicsState {
	if len(e.graphicsStack) == 0 {
		return nil
	}
	return e.graphicsStack[len(e.graphicsStack)-1]
}

// TextLineMatrix returns the text line matrix.
func (e *PDFStreamEngine) TextLineMatrix() *util.Matrix {
	return e.GraphicsState().TextLineMatrix()
}

// SetTextLineMatrix sets the text line matrix.
func (e *PDFStreamEngine) SetTextLineMatrix(value *util.Matrix) {
	e.GraphicsState().SetTextLineMatrix(value)
}

// TextMatrix returns the text matrix.
func (e *PDFStreamEngine) TextMatrix() *util.Matrix {
	return e.GraphicsState().TextMatrix()
}

// SetTextMatrix sets the text matrix.
func (e *PDFStreamEngine) SetTextMatrix(value *util.Matrix) {
	e.GraphicsState().SetTextMatrix(value)
}

// SetLineDashPattern sets the line dash pattern from a dash array and phase.
func (e *PDFStreamEngine) SetLineDashPattern(array *cos.Array, phase int) {
	e.GraphicsState().SetLineDashPattern(graphics.NewPDLineDashPatternOf(array, phase))
}

// Resources returns the current stream's resources. This is mainly to be used
// by the operator processors.
func (e *PDFStreamEngine) Resources() *pdmodel.PDResources { return e.resources }

// CurrentPage returns the current page.
func (e *PDFStreamEngine) CurrentPage() *pdmodel.PDPage { return e.currentPage }

// InitialMatrix returns the stream's initial matrix.
func (e *PDFStreamEngine) InitialMatrix() *util.Matrix { return e.initialMatrix }

// TransformedPoint transforms a point using the CTM.
func (e *PDFStreamEngine) TransformedPoint(x, y float32) *geom.PointFloat {
	position := []float32{x, y}
	e.GraphicsState().CurrentTransformationMatrix().CreateAffineTransform().
		TransformFloats(position, 0, position, 0, 1)
	return geom.NewPointFloat(position[0], position[1])
}

// TransformWidth transforms a width using the CTM.
func (e *PDFStreamEngine) TransformWidth(width float32) float32 {
	ctm := e.GraphicsState().CurrentTransformationMatrix()
	x := ctm.ScaleX() + ctm.ShearX()
	y := ctm.ScaleY() + ctm.ShearY()
	return width * float32(math.Sqrt(float64(x*x+y*y)*0.5))
}

// Level returns the current level, which says how deep a potentially recursive
// operation has gone, so that a caller can skip one that would go too far.
func (e *PDFStreamEngine) Level() int { return e.level }

// IncreaseLevel increases the level. Call it before running a potentially
// recursive operation.
func (e *PDFStreamEngine) IncreaseLevel() { e.level++ }

// DecreaseLevel decreases the level. A level below zero is logged: it means an
// operation finished without its level being given back.
func (e *PDFStreamEngine) DecreaseLevel() {
	e.level--
	if e.level < 0 {
		slog.Error("level is negative", "level", e.level)
	}
}

// ShouldProcessColorOperators tells whether colour operators should be
// processed. To be used in some operator processors.
func (e *PDFStreamEngine) ShouldProcessColorOperators() bool {
	return e.shouldProcessColorOperators
}
