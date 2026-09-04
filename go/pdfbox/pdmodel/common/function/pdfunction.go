// Package function holds the PDF functions, which map numbers to numbers:
// sampled, exponential, stitching and PostScript calculator.
//
// Port of org.apache.pdfbox.pdmodel.common.function.
package function

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDFunction is a function in a PDF, which maps m input numbers to n output
// numbers.
//
// Port of the abstract org.apache.pdfbox.pdmodel.common.function.PDFunction:
// an interface plus the embedded pdFunctionBase below, which is how the port
// takes every abstract class.
type PDFunction interface {
	common.COSObjectable

	// FunctionType returns the type of this function: 0 sampled, 2
	// exponential, 3 stitching, 4 PostScript calculator.
	//
	// PDFunctionTypeIdentity has no type and panics, as Java throws
	// UnsupportedOperationException there.
	FunctionType() int

	// Eval evaluates the function for the given input.
	Eval(input []float32) ([]float32, error)

	// COSDictionary returns the dictionary below this function, which is what
	// Java's getCOSObject returns; the interface above returns a cos.Base,
	// because COSObjectable does.
	COSDictionary() *cos.Dictionary

	// NumberOfOutputParameters returns how many numbers Eval returns.
	NumberOfOutputParameters() int

	// NumberOfInputParameters returns how many numbers Eval takes.
	NumberOfInputParameters() int

	// DomainForInput returns the domain of input n.
	DomainForInput(n int) *common.PDRange

	// RangeForOutput returns the range of output n.
	RangeForOutput(n int) *common.PDRange
}

// pdFunctionBase carries what every function type shares.
type pdFunctionBase struct {
	functionStream       *common.PDStream
	functionDictionary   *cos.Dictionary
	domain               *cos.Array
	rangeArray           *cos.Array
	numberOfInputValues  int
	numberOfOutputValues int
}

// newPDFunctionBase is Java's protected PDFunction(COSBase) constructor.
func newPDFunctionBase(function cos.Base) pdFunctionBase {
	b := pdFunctionBase{numberOfInputValues: -1, numberOfOutputValues: -1}
	switch f := function.(type) {
	case *cos.Stream:
		b.functionStream = common.NewPDStream(f)
		b.functionStream.Stream().SetItem(cos.Type, cos.Function)
	case *cos.Dictionary:
		b.functionDictionary = f
	}
	return b
}

// COSObject returns the dictionary below this function.
func (b *pdFunctionBase) COSObject() cos.Base { return b.COSDictionary() }

// COSDictionary returns the dictionary below this function.
//
// Java's getCOSObject is final and returns the stream's dictionary where the
// function is a stream. PDFunctionTypeIdentity passes null to the constructor
// and so has neither, which its own TODO in the Java calls out; the port
// returns nil there rather than dereferencing.
func (b *pdFunctionBase) COSDictionary() *cos.Dictionary {
	if b.functionStream != nil {
		return &b.functionStream.Stream().Dictionary
	}
	return b.functionDictionary
}

// PDStream returns the stream below this function, or nil where it is a plain
// dictionary.
func (b *pdFunctionBase) PDStream() *common.PDStream { return b.functionStream }

// NewPDFunction creates a function from its COS object.
//
// Port of the static PDFunction.create.
func NewPDFunction(function cos.Base) (PDFunction, error) {
	if function == cos.Base(cos.Identity) {
		return NewPDFunctionTypeIdentity(nil), nil
	}

	base := function
	if object, ok := function.(*cos.Object); ok {
		base = object.Object()
	}
	functionDictionary, ok := base.(*cos.Dictionary)
	if !ok {
		return nil, fmt.Errorf("Error: Function must be a Dictionary, but is %s",
			simpleName(base))
	}
	functionType := functionDictionary.GetInt(cos.FunctionType)
	switch functionType {
	case 0:
		return NewPDFunctionType0(functionDictionary), nil
	case 2:
		return NewPDFunctionType2(functionDictionary), nil
	case 3:
		return NewPDFunctionType3(functionDictionary), nil
	case 4:
		return NewPDFunctionType4(functionDictionary)
	default:
		return nil, fmt.Errorf("Error: Unknown function type %d", functionType)
	}
}

// simpleName is Java's Class.getSimpleName for the message above, which names
// the COS type the caller passed.
func simpleName(base cos.Base) string {
	switch base.(type) {
	case nil:
		return "(null)"
	case *cos.Array:
		return "COSArray"
	case *cos.Name:
		return "COSName"
	case *cos.Stream:
		return "COSStream"
	case *cos.Integer:
		return "COSInteger"
	case *cos.Float:
		return "COSFloat"
	case *cos.StringObj:
		return "COSString"
	case *cos.Boolean:
		return "COSBoolean"
	case *cos.Null:
		return "COSNull"
	default:
		return fmt.Sprintf("%T", base)
	}
}

// NumberOfOutputParameters returns how many numbers the function returns.
func (b *pdFunctionBase) NumberOfOutputParameters() int {
	if b.numberOfOutputValues == -1 {
		rangeValues := b.rangeValues()
		if rangeValues == nil {
			b.numberOfOutputValues = 0
		} else {
			b.numberOfOutputValues = rangeValues.Size() / 2
		}
	}
	return b.numberOfOutputValues
}

// RangeForOutput returns the range of output n.
func (b *pdFunctionBase) RangeForOutput(n int) *common.PDRange {
	return common.NewPDRangeOfIndex(b.rangeValues(), n)
}

// SetRangeValues sets the /Range array.
func (b *pdFunctionBase) SetRangeValues(rangeValues *cos.Array) {
	b.rangeArray = rangeValues
	b.COSDictionary().SetItem(cos.Range, rangeValues)
}

// NumberOfInputParameters returns how many numbers the function takes.
func (b *pdFunctionBase) NumberOfInputParameters() int {
	if b.numberOfInputValues == -1 {
		array := b.domainValues()
		b.numberOfInputValues = array.Size() / 2
	}
	return b.numberOfInputValues
}

// DomainForInput returns the domain of input n.
func (b *pdFunctionBase) DomainForInput(n int) *common.PDRange {
	return common.NewPDRangeOfIndex(b.domainValues(), n)
}

// SetDomainValues sets the /Domain array.
func (b *pdFunctionBase) SetDomainValues(domainValues *cos.Array) {
	b.domain = domainValues
	b.COSDictionary().SetItem(cos.Domain, domainValues)
}

// rangeValues returns the /Range array.
//
// PDFunctionTypeIdentity overrides it to return null; Go has no dynamic
// dispatch on an embedded method, so the only caller that depends on that
// override -- NumberOfOutputParameters -- is written out on the override too.
func (b *pdFunctionBase) rangeValues() *cos.Array {
	if b.rangeArray == nil {
		b.rangeArray = b.COSDictionary().GetCOSArray(cos.Range)
	}
	return b.rangeArray
}

func (b *pdFunctionBase) domainValues() *cos.Array {
	if b.domain == nil {
		b.domain = b.COSDictionary().GetCOSArray(cos.Domain)
	}
	return b.domain
}

// clipToRangeAll clips every output to its range.
func (b *pdFunctionBase) clipToRangeAll(inputValues []float32) []float32 {
	rangesArray := b.rangeValues()
	if rangesArray == nil || rangesArray.IsEmpty() {
		return inputValues
	}
	rangeValues := rangesArray.ToFloatArray()
	numberOfRanges := len(rangeValues) / 2
	result := make([]float32, numberOfRanges)
	for i := 0; i < numberOfRanges; i++ {
		index := i << 1
		result[i] = clipToRange(inputValues[i], rangeValues[index], rangeValues[index+1])
	}
	return result
}

// clipToRange clips one value.
func clipToRange(x, rangeMin, rangeMax float32) float32 {
	if x < rangeMin {
		return rangeMin
	} else if x > rangeMax {
		return rangeMax
	}
	return x
}

// interpolate maps x from one interval to another.
func interpolate(x, xRangeMin, xRangeMax, yRangeMin, yRangeMax float32) float32 {
	if xRangeMax == xRangeMin {
		// PDFBOX-5593 / PR #162
		return yRangeMin
	}
	return yRangeMin + ((x - xRangeMin) * (yRangeMax - yRangeMin) / (xRangeMax - xRangeMin))
}

// PDFunctionTypeIdentity is the function that returns its input.
//
// Port of org.apache.pdfbox.pdmodel.common.function.PDFunctionTypeIdentity.
type PDFunctionTypeIdentity struct {
	pdFunctionBase
}

var _ PDFunction = (*PDFunctionTypeIdentity)(nil)

// NewPDFunctionTypeIdentity creates the identity function.
func NewPDFunctionTypeIdentity(function cos.Base) *PDFunctionTypeIdentity {
	// Java passes null to the super constructor, with a TODO saying that
	// getCOSObject can then throw; the port keeps the null and returns nil
	// from COSDictionary rather than reproducing a NullPointerException.
	f := &PDFunctionTypeIdentity{pdFunctionBase: newPDFunctionBase(nil)}

	return f
}

// FunctionType panics, where Java throws UnsupportedOperationException.
//
// Java's comment says it "shouldn't be called", and that this is a violation of
// the interface segregation principle.
func (f *PDFunctionTypeIdentity) FunctionType() int {
	panic("PDFunctionTypeIdentity has no function type")
}

// Eval returns the input.
func (f *PDFunctionTypeIdentity) Eval(input []float32) ([]float32, error) {
	return input, nil
}

// rangeValues returns nil, which is Java's override.
func (f *PDFunctionTypeIdentity) rangeValues() *cos.Array { return nil }

// NumberOfOutputParameters is 0, because rangeValues is null.
//
// Java reaches the override through dynamic dispatch from the base class; Go
// has no such dispatch on an embedded method, so the one caller that depends on
// it is written out here.
func (f *PDFunctionTypeIdentity) NumberOfOutputParameters() int { return 0 }

// String is Java's toString.
func (f *PDFunctionTypeIdentity) String() string { return "FunctionTypeIdentity" }
