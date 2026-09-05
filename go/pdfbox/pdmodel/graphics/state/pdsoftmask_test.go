package state

// PDFBox has no test for PDSoftMask, so these are written from the Java source.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// TestSoftMaskOfNone is the first arm of the static create: /None is the way a
// graphics state says it has no soft mask, and answers nil rather than a mask
// over the name.
func TestSoftMaskOfNone(t *testing.T) {
	if got := NewPDSoftMaskOf(cos.None); got != nil {
		t.Errorf("NewPDSoftMaskOf(/None) = %v, want nil", got)
	}
}

// TestSoftMaskOfAnythingElse is the two arms that log and answer nil: any other
// name, and anything that is not a dictionary at all.
func TestSoftMaskOfAnythingElse(t *testing.T) {
	for _, base := range []cos.Base{
		cos.GetPDFName("Nonsense"),
		cos.NewArray(),
		cos.NewStringObj("not a mask"),
		nil,
	} {
		if got := NewPDSoftMaskOf(base); got != nil {
			t.Errorf("NewPDSoftMaskOf(%v) = %v, want nil", base, got)
		}
	}
}

// TestSoftMaskAccessors round trips the entries a soft mask carries, and checks
// that each is cached the way Java's fields are.
func TestSoftMaskAccessors(t *testing.T) {
	dict := cos.NewDictionary()
	dict.SetItem(cos.S, cos.Luminosity)
	backdrop := cos.NewArray()
	backdrop.Add(cos.NewFloat(0.25))
	dict.SetItem(cos.BC, backdrop)

	mask := NewPDSoftMaskOf(dict)
	if mask == nil {
		t.Fatal("NewPDSoftMaskOf(dictionary) = nil, want a mask")
	}
	if got := mask.SubType(); got != cos.Luminosity {
		t.Errorf("SubType() = %v, want /Luminosity", got)
	}
	if got := mask.BackdropColor(); got != backdrop {
		t.Errorf("BackdropColor() = %v, want the array that was set", got)
	}
	// A mask with no /G has no group, whether or not a constructor is
	// registered.
	if got := mask.Group(); got != nil {
		t.Errorf("Group() = %v, want nil for a mask with no /G", got)
	}
	// A mask with no /TR has no transfer function.
	transfer, err := mask.TransferFunction()
	if err != nil {
		t.Fatal(err)
	}
	if transfer != nil {
		t.Errorf("TransferFunction() = %v, want nil for a mask with no /TR", transfer)
	}
}

// TestSoftMaskTransferFunction reads the /TR entry, which maps the mask value
// before it is used.
func TestSoftMaskTransferFunction(t *testing.T) {
	exponential := cos.NewDictionary()
	exponential.SetInt(cos.FunctionType, 2)
	domain := cos.NewArray()
	domain.Add(cos.NewFloat(0))
	domain.Add(cos.NewFloat(1))
	exponential.SetItem(cos.Domain, domain)
	exponential.SetInt(cos.N, 1)

	dict := cos.NewDictionary()
	dict.SetItem(cos.TR, exponential)
	mask := NewPDSoftMaskOf(dict)

	transfer, err := mask.TransferFunction()
	if err != nil {
		t.Fatal(err)
	}
	if transfer == nil {
		t.Fatal("TransferFunction() = nil, want the type 2 function")
	}
	if got := transfer.FunctionType(); got != 2 {
		t.Errorf("FunctionType() = %d, want 2", got)
	}
}

// TestSoftMaskInitialTransformationMatrix pins what the matrix is for: a mask
// has to know the transformation in force when its graphics state was
// activated, not when it is painted through.
func TestSoftMaskInitialTransformationMatrix(t *testing.T) {
	mask := NewPDSoftMaskOf(cos.NewDictionary())
	if got := mask.InitialTransformationMatrix(); got != nil {
		t.Errorf("InitialTransformationMatrix() = %v, want nil before one is set", got)
	}
	ctm := util.NewMatrixOf(2, 0, 0, 2, 10, 20)
	mask.SetInitialTransformationMatrix(ctm)
	if got := mask.InitialTransformationMatrix(); got != ctm {
		t.Errorf("InitialTransformationMatrix() = %v, want the matrix that was set", got)
	}
}

// TestExtendedGraphicsStateAppliesTheSoftMask checks the /SMask arm of
// copyIntoGraphicsState: the mask reaches the graphics state, and it is handed
// a copy of the transformation matrix in force at that moment rather than the
// matrix itself.
func TestExtendedGraphicsStateAppliesTheSoftMask(t *testing.T) {
	smask := cos.NewDictionary()
	smask.SetItem(cos.S, cos.Luminosity)
	ext := cos.NewDictionary()
	ext.SetItem(cos.SMask, smask)

	gs := NewPDGraphicsState(common.NewPDRectangleOfSize(100, 100))
	ctm := util.NewMatrixOf(3, 0, 0, 3, 0, 0)
	gs.SetCurrentTransformationMatrix(ctm)

	if err := NewPDExtendedGraphicsStateOf(ext).CopyIntoGraphicsState(gs); err != nil {
		t.Fatal(err)
	}

	applied := gs.SoftMask()
	if applied == nil {
		t.Fatal("SoftMask() = nil, want the mask the ExtGState names")
	}
	if got := applied.SubType(); got != cos.Luminosity {
		t.Errorf("SubType() = %v, want /Luminosity", got)
	}
	initial := applied.InitialTransformationMatrix()
	if initial == nil {
		t.Fatal("InitialTransformationMatrix() = nil, want the CTM at activation")
	}
	if initial == ctm {
		t.Error("the mask kept the graphics state's own matrix, want a clone of it")
	}
	if initial.ScaleX() != 3 || initial.ScaleY() != 3 {
		t.Errorf("InitialTransformationMatrix() = %v, want the CTM that was in force", initial)
	}

	// An /SMask of /None clears the mask rather than setting one.
	cleared := cos.NewDictionary()
	cleared.SetItem(cos.SMask, cos.None)
	if err := NewPDExtendedGraphicsStateOf(cleared).CopyIntoGraphicsState(gs); err != nil {
		t.Fatal(err)
	}
	if got := gs.SoftMask(); got != nil {
		t.Errorf("SoftMask() = %v, want nil after /SMask /None", got)
	}
}
