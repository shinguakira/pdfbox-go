package function_test

import (
	"math"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function"
)

// Port of
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/common/function/TestPDFunctionType4.java.

func createFunction(t *testing.T, program string, domain, rangeValues []float32) *function.PDFunctionType4 {
	t.Helper()
	stream := cos.NewStream(filter.Provider{})
	stream.SetInt(cos.FunctionType, 4)
	stream.SetItem(cos.Domain, cos.ArrayOfFloats(domain))
	stream.SetItem(cos.Range, cos.ArrayOfFloats(rangeValues))
	out, err := stream.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := out.Write([]byte(program)); err != nil {
		t.Fatalf("writing the program: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("closing the stream: %v", err)
	}
	f, err := function.NewPDFunctionType4(stream)
	if err != nil {
		t.Fatalf("NewPDFunctionType4: %v", err)
	}
	return f
}

func TestFunctionSimple(t *testing.T) {
	functionText := "{ add }"
	// Simply adds the two arguments and returns the result
	f := createFunction(t, functionText,
		[]float32{-1.0, 1.0, -1.0, 1.0},
		[]float32{-1.0, 1.0})

	output := mustEval(t, f, []float32{0.8, 0.1})
	if len(output) != 1 {
		t.Fatalf("got %d outputs, want 1", len(output))
	}
	if math.Abs(float64(output[0])-0.9) > 0.0001 {
		t.Errorf("output = %v, want 0.9", output[0])
	}

	output = mustEval(t, f, []float32{0.8, 0.3}) // results in 1.1f being outside Range
	if len(output) != 1 {
		t.Fatalf("got %d outputs, want 1", len(output))
	}
	if output[0] != 1 {
		t.Errorf("output = %v, want 1", output[0])
	}

	output = mustEval(t, f, []float32{0.8, 1.2}) // input argument outside Dimension
	if len(output) != 1 {
		t.Fatalf("got %d outputs, want 1", len(output))
	}
	if output[0] != 1 {
		t.Errorf("output = %v, want 1", output[0])
	}
}

func TestFunctionArgumentOrder(t *testing.T) {
	functionText := "{ pop }"
	// pops an argument (2nd) and returns the next argument (1st)
	f := createFunction(t, functionText,
		[]float32{-1.0, 1.0, -1.0, 1.0},
		[]float32{-1.0, 1.0})

	output := mustEval(t, f, []float32{-0.7, 0.0})
	if len(output) != 1 {
		t.Fatalf("got %d outputs, want 1", len(output))
	}
	if math.Abs(float64(output[0])+0.7) > 0.0001 {
		t.Errorf("output = %v, want -0.7", output[0])
	}
}

func mustEval(t *testing.T, f function.PDFunction, input []float32) []float32 {
	t.Helper()
	output, err := f.Eval(input)
	if err != nil {
		t.Fatalf("Eval(%v): %v", input, err)
	}
	return output
}
