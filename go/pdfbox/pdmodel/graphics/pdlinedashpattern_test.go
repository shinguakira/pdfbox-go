package graphics

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Written from org.apache.pdfbox.pdmodel.graphics.PDLineDashPattern; the Java
// suite has no test for it.

func TestPDLineDashPatternDefault(t *testing.T) {
	p := NewPDLineDashPattern()
	if got := p.Phase(); got != 0 {
		t.Errorf("Phase = %d, want 0", got)
	}
	if got := p.DashArray(); len(got) != 0 {
		t.Errorf("DashArray = %v, want none", got)
	}
}

func TestPDLineDashPatternOf(t *testing.T) {
	p := NewPDLineDashPatternOf(cos.ArrayOfFloats([]float32{3, 2}), 1)
	if got := p.Phase(); got != 1 {
		t.Errorf("Phase = %d, want 1", got)
	}
	if got := p.DashArray(); len(got) != 2 || got[0] != 3 || got[1] != 2 {
		t.Errorf("DashArray = %v, want [3 2]", got)
	}
}

// TestPDLineDashPatternArrayIsCopied pins that the pattern is immutable: the
// array handed back cannot be used to change it.
func TestPDLineDashPatternArrayIsCopied(t *testing.T) {
	p := NewPDLineDashPatternOf(cos.ArrayOfFloats([]float32{3, 2}), 0)
	p.DashArray()[0] = 99
	if got := p.DashArray()[0]; got != 3 {
		t.Errorf("the pattern changed to %v", got)
	}
}

// TestPDLineDashPatternNegativePhase pins the rule from the PDF 2.0
// specification: a negative phase is raised by twice the sum of the dash
// lengths until it is positive.
func TestPDLineDashPatternNegativePhase(t *testing.T) {
	// Twice the sum is 10, so -3 goes to 7 in one step.
	p := NewPDLineDashPatternOf(cos.ArrayOfFloats([]float32{3, 2}), -3)
	if got := p.Phase(); got != 7 {
		t.Errorf("Phase = %d, want 7", got)
	}

	// -23 needs three steps of 10, landing on 7 again.
	p = NewPDLineDashPatternOf(cos.ArrayOfFloats([]float32{3, 2}), -23)
	if got := p.Phase(); got != 7 {
		t.Errorf("Phase = %d, want 7", got)
	}
}

// TestPDLineDashPatternNegativePhaseNoDashes pins that a negative phase with
// nothing to step by is simply zeroed rather than looping forever.
func TestPDLineDashPatternNegativePhaseNoDashes(t *testing.T) {
	p := NewPDLineDashPatternOf(cos.NewArray(), -5)
	if got := p.Phase(); got != 0 {
		t.Errorf("Phase = %d, want 0", got)
	}

	p = NewPDLineDashPatternOf(cos.ArrayOfFloats([]float32{0, 0}), -5)
	if got := p.Phase(); got != 0 {
		t.Errorf("Phase = %d, want 0", got)
	}
}

func TestPDLineDashPatternCOSObject(t *testing.T) {
	array, ok := NewPDLineDashPatternOf(cos.ArrayOfFloats([]float32{3, 2}), 1).COSObject().(*cos.Array)
	if !ok {
		t.Fatal("COSObject is not an array")
	}
	if array.Size() != 2 {
		t.Fatalf("array = %v, want the dashes and the phase", array)
	}
	dashes, ok := array.Get(0).(*cos.Array)
	if !ok || dashes.Size() != 2 {
		t.Errorf("the first entry is %v, want the dash array", array.Get(0))
	}
	if got := array.GetInt(1); got != 1 {
		t.Errorf("the phase is %d, want 1", got)
	}
}

func TestPDLineDashPatternString(t *testing.T) {
	got := NewPDLineDashPatternOf(cos.ArrayOfFloats([]float32{3, 2}), 1).String()
	if want := "PDLineDashPattern{array=[3.0, 2.0], phase=1}"; got != want {
		t.Errorf("String = %q, want %q", got, want)
	}
}
