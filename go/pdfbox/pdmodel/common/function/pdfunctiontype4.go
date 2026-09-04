package function

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function/type4"
)

// type4Operators is the operator set every type 4 function runs against.
//
// Port of the static PDFunctionType4.OPERATORS.
var type4Operators = type4.NewOperators()

// PDFunctionType4 is a PostScript calculator function.
//
// Port of org.apache.pdfbox.pdmodel.common.function.PDFunctionType4.
type PDFunctionType4 struct {
	pdFunctionBase

	instructions *type4.InstructionSequence
}

var _ PDFunction = (*PDFunctionType4)(nil)

// NewPDFunctionType4 parses a calculator function out of its stream.
func NewPDFunctionType4(functionStream cos.Base) (*PDFunctionType4, error) {
	f := &PDFunctionType4{pdFunctionBase: newPDFunctionBase(functionStream)}
	strm := f.PDStream()
	var bytes []byte
	if strm != nil {
		var err error
		bytes, err = strm.ToByteArray()
		if err != nil {
			return nil, err
		}
	}
	// Java reads the program as ISO-8859-1, which maps each byte to the code
	// point of the same value; the tokenizer walks bytes, so the port hands it
	// the bytes as they are.
	f.instructions = type4.Parse(string(bytes))
	return f, nil
}

// FunctionType returns 4.
func (f *PDFunctionType4) FunctionType() int { return 4 }

// Eval runs the calculator program.
func (f *PDFunctionType4) Eval(input []float32) ([]float32, error) {
	// Setup the input values
	context := type4.NewExecutionContext(type4Operators)
	for i := 0; i < len(input); i++ {
		domain := f.DomainForInput(i)
		value := clipToRange(input[i], domain.Min(), domain.Max())
		context.Push(value)
	}

	// Execute the type 4 function.
	f.instructions.Execute(context)

	// Extract the output values
	numberOfOutputValues := f.NumberOfOutputParameters()
	numberOfActualOutputValues := context.Size()
	if numberOfActualOutputValues < numberOfOutputValues {
		// Java throws IllegalStateException, which is unchecked.
		panic(fmt.Sprintf("The type 4 function returned %d values but the Range entry "+
			"indicates that %d values be returned.",
			numberOfActualOutputValues, numberOfOutputValues))
	}
	outputValues := make([]float32, numberOfOutputValues)
	for i := numberOfOutputValues - 1; i >= 0; i-- {
		rangeForOutput := f.RangeForOutput(i)
		outputValues[i] = context.PopReal()
		outputValues[i] = clipToRange(outputValues[i], rangeForOutput.Min(), rangeForOutput.Max())
	}

	// Return the resulting array
	return outputValues, nil
}
