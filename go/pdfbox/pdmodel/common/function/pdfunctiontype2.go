package function

import (
	"fmt"
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// PDFunctionType2 is an exponential interpolation function.
//
// Port of org.apache.pdfbox.pdmodel.common.function.PDFunctionType2.
type PDFunctionType2 struct {
	pdFunctionBase

	c0       *cos.Array
	c1       *cos.Array
	exponent float32
}

var _ PDFunction = (*PDFunctionType2)(nil)

// NewPDFunctionType2 creates an exponential function from its dictionary.
func NewPDFunctionType2(function cos.Base) *PDFunctionType2 {
	f := &PDFunctionType2{pdFunctionBase: newPDFunctionBase(function)}

	cosObject := f.COSDictionary()
	f.c0 = cosObject.GetCOSArray(cos.C0)
	if f.c0 == nil {
		f.c0 = cos.NewArray()
	}
	if f.c0.IsEmpty() {
		f.c0.Add(cos.FloatZero)
	}

	f.c1 = cosObject.GetCOSArray(cos.C1)
	if f.c1 == nil {
		f.c1 = cos.NewArray()
	}
	if f.c1.IsEmpty() {
		f.c1.Add(cos.FloatOne)
	}

	f.exponent = f.COSDictionary().GetFloat(cos.N, 0)
	return f
}

// FunctionType returns 2.
func (f *PDFunctionType2) FunctionType() int { return 2 }

// Eval interpolates exponentially between C0 and C1.
func (f *PDFunctionType2) Eval(input []float32) ([]float32, error) {
	// exponential interpolation
	xToN := float32(math.Pow(float64(input[0]), float64(f.exponent))) // x^exponent

	size := f.c0.Size()
	if f.c1.Size() < size {
		size = f.c1.Size()
	}
	result := make([]float32, size)
	for j := 0; j < size; j++ {
		c0j := f.c0.Get(j).(cos.Number).FloatValue()
		c1j := f.c1.Get(j).(cos.Number).FloatValue()
		result[j] = c0j + xToN*(c1j-c0j)
	}

	return f.clipToRangeAll(result), nil
}

// C0 returns the /C0 array, the value at x = 0.
func (f *PDFunctionType2) C0() *cos.Array { return f.c0 }

// C1 returns the /C1 array, the value at x = 1.
func (f *PDFunctionType2) C1() *cos.Array { return f.c1 }

// N returns the exponent.
func (f *PDFunctionType2) N() float32 { return f.exponent }

// String is Java's toString.
func (f *PDFunctionType2) String() string {
	return fmt.Sprintf("FunctionType2{C0: %v C1: %v N: %v}", f.C0(), f.C1(), f.N())
}
