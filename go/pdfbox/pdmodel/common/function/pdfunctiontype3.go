package function

import (
	"errors"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDFunctionType3 is a stitching function: it splits the domain into
// partitions and hands each to a function of its own.
//
// Port of org.apache.pdfbox.pdmodel.common.function.PDFunctionType3.
type PDFunctionType3 struct {
	pdFunctionBase

	functions      *cos.Array
	encode         *cos.Array
	bounds         *cos.Array
	functionsArray []PDFunction
	boundsValues   []float32
}

var _ PDFunction = (*PDFunctionType3)(nil)

// NewPDFunctionType3 creates a stitching function from its dictionary.
func NewPDFunctionType3(functionStream cos.Base) *PDFunctionType3 {
	return &PDFunctionType3{pdFunctionBase: newPDFunctionBase(functionStream)}
}

// FunctionType returns 3.
func (f *PDFunctionType3) FunctionType() int { return 3 }

// Eval picks the partition the input falls in and evaluates its function.
func (f *PDFunctionType3) Eval(input []float32) ([]float32, error) {
	// This function is known as a "stitching" function. Based on the input, it
	// decides which child function to call.
	// All functions in the array are 1-value-input functions
	// See PDF Reference section 3.9.3.
	var function PDFunction
	x := input[0]
	domain := f.DomainForInput(0)
	// clip input value to domain
	x = clipToRange(x, domain.Min(), domain.Max())

	if f.functionsArray == nil {
		ar := f.Functions()
		f.functionsArray = make([]PDFunction, ar.Size())
		for i := 0; i < ar.Size(); i++ {
			child, err := NewPDFunction(ar.GetObject(i))
			if err != nil {
				return nil, err
			}
			f.functionsArray[i] = child
		}
	}

	if len(f.functionsArray) == 1 {
		// This doesn't make sense but it may happen ...
		function = f.functionsArray[0]
		encRange := f.encodeForParameter(0)
		x = interpolate(x, domain.Min(), domain.Max(), encRange.Min(), encRange.Max())
	} else {
		if f.boundsValues == nil {
			f.boundsValues = f.Bounds().ToFloatArray()
		}
		boundsSize := len(f.boundsValues)
		// create a combined array containing the domain and the bounds values
		// domain.min, bounds[0], bounds[1], ...., bounds[boundsSize-1], domain.max
		partitionValues := make([]float32, boundsSize+2)
		partitionValuesSize := len(partitionValues)
		partitionValues[0] = domain.Min()
		partitionValues[partitionValuesSize-1] = domain.Max()
		copy(partitionValues[1:], f.boundsValues)
		// find the partition
		for i := 0; i < partitionValuesSize-1; i++ {
			if x >= partitionValues[i] &&
				(x < partitionValues[i+1] ||
					(i == partitionValuesSize-2 && floatCompare(x, partitionValues[i+1]) == 0)) {
				function = f.functionsArray[i]
				encRange := f.encodeForParameter(i)
				x = interpolate(x, partitionValues[i], partitionValues[i+1],
					encRange.Min(), encRange.Max())
				break
			}
		}
	}
	if function == nil {
		return nil, errors.New("partition not found in type 3 function")
	}
	functionValues := []float32{x}
	// calculate the output values using the chosen function
	functionResult, err := function.Eval(functionValues)
	if err != nil {
		return nil, err
	}
	// clip to range if available
	return f.clipToRangeAll(functionResult), nil
}

// floatCompare is java.lang.Float.compare, which orders NaN above everything
// and -0.0 below 0.0; the stitching test only asks whether two values are
// equal, and this keeps the answer the same for those two cases.
func floatCompare(a, b float32) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	aBits := float32Bits(a)
	bBits := float32Bits(b)
	switch {
	case aBits == bBits:
		return 0
	case aBits < bBits:
		return -1
	default:
		return 1
	}
}

// Functions returns the /Functions array.
func (f *PDFunctionType3) Functions() *cos.Array {
	if f.functions == nil {
		f.functions = f.COSDictionary().GetCOSArray(cos.Functions)
	}
	return f.functions
}

// Bounds returns the /Bounds array, which says where the partitions meet.
func (f *PDFunctionType3) Bounds() *cos.Array {
	if f.bounds == nil {
		f.bounds = f.COSDictionary().GetCOSArray(cos.Bounds)
	}
	return f.bounds
}

// Encode returns the /Encode array.
func (f *PDFunctionType3) Encode() *cos.Array {
	if f.encode == nil {
		f.encode = f.COSDictionary().GetCOSArray(cos.Encode)
	}
	return f.encode
}

func (f *PDFunctionType3) encodeForParameter(n int) *common.PDRange {
	return common.NewPDRangeOfIndex(f.Encode(), n)
}
