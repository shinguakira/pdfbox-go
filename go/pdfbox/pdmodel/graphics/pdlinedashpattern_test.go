package graphics_test

// Port of org.apache.pdfbox.pdmodel.graphics.PDLineDashPatternTest.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics"
)

// TestGetCOSObject is testGetCOSObject.
//
// The dashes go in as integers and come back as floats: the pattern keeps them
// as a float array of its own rather than the array it was handed. The phase
// stays an integer.
func TestGetCOSObject(t *testing.T) {
	ar := cos.NewArray()
	ar.Add(cos.IntegerOne)
	ar.Add(cos.IntegerTwo)

	dash := graphics.NewPDLineDashPatternOf(ar, 3)

	dashBase, isArray := dash.COSObject().(*cos.Array)
	if !isArray {
		t.Fatalf("COSObject() is %T, want an array", dash.COSObject())
	}
	if dashBase.Size() != 2 {
		t.Fatalf("the pattern array holds %d entries, want 2", dashBase.Size())
	}

	dashArray, isArray := dashBase.GetObject(0).(*cos.Array)
	if !isArray {
		t.Fatalf("the first entry is %T, want an array", dashBase.GetObject(0))
	}
	if dashArray.Size() != 2 {
		t.Fatalf("the dash array holds %d entries, want 2", dashArray.Size())
	}
	wantFloat(t, dashArray.Get(0), cos.FloatOne, "the first dash")
	wantFloat(t, dashArray.Get(1), cos.NewFloat(2), "the second dash")

	phase, isInteger := dashBase.Get(1).(*cos.Integer)
	if !isInteger {
		t.Fatalf("the phase is %T, want an integer", dashBase.Get(1))
	}
	if !phase.Equals(cos.IntegerThree) {
		t.Errorf("the phase is %v, want 3", phase)
	}

	// Java ends with System.out.println(dash); the port checks String() answers
	// something rather than printing it, which is all that line proves.
	if dash.String() == "" {
		t.Error("String() = \"\", want a description of the pattern")
	}
}

// wantFloat is assertEquals(expected, dashArray.get(i)) for a COSFloat.
func wantFloat(t *testing.T, got cos.Base, want *cos.Float, what string) {
	t.Helper()
	value, isFloat := got.(*cos.Float)
	if !isFloat {
		t.Fatalf("%s is %T, want a float", what, got)
	}
	if !value.Equals(want) {
		t.Errorf("%s is %v, want %v", what, value, want)
	}
}
