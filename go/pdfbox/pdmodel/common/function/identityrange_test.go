package function_test

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function"
)

// TestIdentityRangeForOutputHasNoArray pins the call PDFunction.getRangeForOutput
// makes: getRangeValues, which PDFunctionTypeIdentity overrides to answer null.
// So on the identity function it returns a PDRange over no array -- one whose
// getMin then throws -- where the port's base method read the range out of the
// identity function's dictionary, which is nil, and panicked there.
//
// The running PDFBox, over PDFunction.create(COSName.IDENTITY), printed:
//
//	getRangeForOutput(0) returned a PDRange; getCOSArray=null
//	getMin threw NullPointerException
//	getDomainForInput(0) threw NullPointerException
func TestIdentityRangeForOutputHasNoArray(t *testing.T) {
	identity, err := function.NewPDFunction(cos.Identity)
	if err != nil {
		t.Fatalf("NewPDFunction(Identity): %v", err)
	}

	var rangeValue interface{ Min() float32 }
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("RangeForOutput(0) panicked: %v", recovered)
			}
		}()
		r := identity.RangeForOutput(0)
		if r == nil {
			t.Fatal("RangeForOutput(0) = nil, want a range over no array")
		}
		if array, _ := r.COSObject().(*cos.Array); array != nil {
			t.Errorf("RangeForOutput(0) is over %v, want no array", array)
		}
		rangeValue = r
	}()

	panics := func(call func()) (didPanic bool) {
		defer func() { didPanic = recover() != nil }()
		call()
		return false
	}
	if !panics(func() { rangeValue.Min() }) {
		t.Error("Min() on the identity function's range did not panic; Java throws NullPointerException")
	}
	if !panics(func() { identity.DomainForInput(0) }) {
		t.Error("DomainForInput(0) on the identity function did not panic; Java throws NullPointerException")
	}
}
