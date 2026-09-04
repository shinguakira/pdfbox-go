package function_test

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function"
)

// What the slice 6 feedback asked, written down as tests.
//
// A type 0 and a type 4 function are always streams -- one holds a sample
// table and the other a program -- and Java's PDFunction.create tests
// `base instanceof COSDictionary`, which a COSStream satisfies because
// COSStream extends COSDictionary. A Go *cos.Stream embeds cos.Dictionary but
// is not one, so the port's type assertion rejected every function of those two
// types. Neither of the ported Java tests caught it: TestPDFunctionType4 calls
// the type 4 constructor directly.

func newFunctionStream(t *testing.T, functionType int, body string,
	domain, rangeValues []float32) *cos.Stream {
	t.Helper()
	stream := cos.NewStream(filter.Provider{})
	stream.SetInt(cos.FunctionType, functionType)
	stream.SetItem(cos.Domain, cos.ArrayOfFloats(domain))
	stream.SetItem(cos.Range, cos.ArrayOfFloats(rangeValues))
	out, err := stream.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := out.Write([]byte(body)); err != nil {
		t.Fatalf("writing the body: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("closing: %v", err)
	}
	return stream
}

// TestCreateType4FromStream builds a calculator function through the factory,
// which is how a /TintTransform reaches it.
func TestCreateType4FromStream(t *testing.T) {
	stream := newFunctionStream(t, 4, "{ add }",
		[]float32{-1, 1, -1, 1}, []float32{-1, 1})

	f, err := function.NewPDFunction(stream)
	if err != nil {
		t.Fatalf("NewPDFunction on a type 4 stream: %v", err)
	}
	if got := f.FunctionType(); got != 4 {
		t.Fatalf("FunctionType = %d, want 4", got)
	}
	output, err := f.Eval([]float32{0.8, 0.1})
	if err != nil {
		t.Fatalf("Eval: %v", err)
	}
	if len(output) != 1 || output[0] < 0.89 || output[0] > 0.91 {
		t.Errorf("Eval gave %v, want about 0.9", output)
	}
}

// TestCreateType0FromStream builds a sampled function through the factory.
//
// The samples are four bytes making a ramp, and the function is asked for its
// two ends, which are the first and last sample; the values come from the
// specification's definition of a sampled function, not from the port.
func TestCreateType0FromStream(t *testing.T) {
	stream := newFunctionStream(t, 0, string([]byte{0, 85, 170, 255}),
		[]float32{0, 1}, []float32{0, 1})
	stream.SetItem(cos.Size, cos.ArrayOfIntegers([]int{4}))
	stream.SetInt(cos.BitsPerSample, 8)

	f, err := function.NewPDFunction(stream)
	if err != nil {
		t.Fatalf("NewPDFunction on a type 0 stream: %v", err)
	}
	if got := f.FunctionType(); got != 0 {
		t.Fatalf("FunctionType = %d, want 0", got)
	}
	low, err := f.Eval([]float32{0})
	if err != nil {
		t.Fatalf("Eval(0): %v", err)
	}
	if len(low) != 1 || low[0] != 0 {
		t.Errorf("Eval(0) = %v, want the first sample, 0", low)
	}
	high, err := f.Eval([]float32{1})
	if err != nil {
		t.Fatalf("Eval(1): %v", err)
	}
	if len(high) != 1 || high[0] != 1 {
		t.Errorf("Eval(1) = %v, want the last sample, 1", high)
	}
}

// TestCreateFromIndirectStream checks the same through an indirect reference,
// which is how a real document holds a function.
func TestCreateFromIndirectStream(t *testing.T) {
	stream := newFunctionStream(t, 4, "{ add }",
		[]float32{-1, 1, -1, 1}, []float32{-1, 1})
	indirect := cos.NewObject(stream)

	f, err := function.NewPDFunction(indirect)
	if err != nil {
		t.Fatalf("NewPDFunction on an indirect type 4 stream: %v", err)
	}
	if got := f.FunctionType(); got != 4 {
		t.Errorf("FunctionType = %d, want 4", got)
	}
}
