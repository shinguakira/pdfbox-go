package common

import (
	"testing"

	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// The immutability tests are a port of
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/common/PDImmutableRectangleTest.java.
// The rest are written from PDRectangle, which the Java suite does not cover
// directly.

// TestImmutableClass is the port of testClass, which asserts that the
// predefined sizes are of the immutable class. Go has no subclassing, so the
// port carries the distinction as a flag on the rectangle and this asks about
// that instead.
func TestImmutableClass(t *testing.T) {
	rect := A4
	if !rect.IsImmutable() {
		t.Error("A4 is not immutable")
	}
	for name, r := range map[string]*PDRectangle{
		"A0": A0, "A1": A1, "A2": A2, "A3": A3, "A4": A4, "A5": A5, "A6": A6,
		"LEGAL": Legal, "LETTER": Letter, "TABLOID": Tabloid,
	} {
		if !r.IsImmutable() {
			t.Errorf("%s is not immutable", name)
		}
	}
}

func TestImmutableSetters(t *testing.T) {
	cases := map[string]func(*PDRectangle){
		"SetUpperRightY": func(r *PDRectangle) { r.SetUpperRightY(0) },
		"SetUpperRightX": func(r *PDRectangle) { r.SetUpperRightX(0) },
		"SetLowerLeftY":  func(r *PDRectangle) { r.SetLowerLeftY(0) },
		"SetLowerLeftX":  func(r *PDRectangle) { r.SetLowerLeftX(0) },
	}
	for name, set := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("no panic on an immutable rectangle, want one")
				}
			}()
			set(A4)
		})
	}
}

func TestPDRectangleConstruction(t *testing.T) {
	if got := NewPDRectangle(); got.String() != "[0.0,0.0,0.0,0.0]" {
		t.Errorf("empty rectangle = %v", got)
	}

	sized := NewPDRectangleOfSize(100, 200)
	if sized.LowerLeftX() != 0 || sized.LowerLeftY() != 0 ||
		sized.UpperRightX() != 100 || sized.UpperRightY() != 200 {
		t.Errorf("sized rectangle = %v, want [0.0,0.0,100.0,200.0]", sized)
	}

	placed := NewPDRectangleOf(10, 20, 100, 200)
	if placed.UpperRightX() != 110 || placed.UpperRightY() != 220 {
		t.Errorf("placed rectangle = %v, want [10.0,20.0,110.0,220.0]", placed)
	}
	if placed.Width() != 100 || placed.Height() != 200 {
		t.Errorf("size = %vx%v, want 100x200", placed.Width(), placed.Height())
	}
}

func TestPDRectangleFromBoundingBox(t *testing.T) {
	r := NewPDRectangleOfBoundingBox(fontutil.NewBoundingBoxOf(1, 2, 3, 4))
	if got, want := r.String(), "[1.0,2.0,3.0,4.0]"; got != want {
		t.Errorf("rectangle = %v, want %v", got, want)
	}
}

// TestPDRectangleFromCOSArraySortsCorners pins that a rectangle given with its
// corners the wrong way round is turned the right way round, because the array
// in a file is not required to start at the lower left.
func TestPDRectangleFromCOSArraySortsCorners(t *testing.T) {
	array := cos.ArrayOfFloats([]float32{100, 200, 10, 20})
	r := NewPDRectangleOfCOSArray(array)

	if got, want := r.String(), "[10.0,20.0,100.0,200.0]"; got != want {
		t.Errorf("rectangle = %v, want %v", got, want)
	}
}

// TestPDRectangleFromShortCOSArray pins that a short array is padded with
// zeroes rather than refused.
func TestPDRectangleFromShortCOSArray(t *testing.T) {
	r := NewPDRectangleOfCOSArray(cos.ArrayOfFloats([]float32{1, 2}))
	if got, want := r.String(), "[0.0,0.0,1.0,2.0]"; got != want {
		t.Errorf("rectangle = %v, want %v", got, want)
	}
}

// TestPDRectangleClampsHugeValues pins the guard against a malformed file: a
// coordinate beyond the range of an int is pulled back to its edge.
func TestPDRectangleClampsHugeValues(t *testing.T) {
	const huge = 1e30
	r := NewPDRectangleOfCOSArray(cos.ArrayOfFloats([]float32{-huge, -huge, huge, huge}))

	const maxInt = float32(2147483647)
	if got := r.UpperRightX(); got != maxInt {
		t.Errorf("UpperRightX = %v, want %v", got, maxInt)
	}
	if got := r.LowerLeftX(); got != -maxInt {
		t.Errorf("LowerLeftX = %v, want %v", got, -maxInt)
	}
}

func TestPDRectangleContains(t *testing.T) {
	r := NewPDRectangleOf(0, 0, 10, 10)
	cases := []struct {
		x, y float32
		want bool
	}{
		{5, 5, true},
		// The edges count as inside, unlike geom.Rectangle2D.
		{0, 0, true},
		{10, 10, true},
		{10.1, 5, false},
	}
	for _, c := range cases {
		if got := r.Contains(c.x, c.y); got != c.want {
			t.Errorf("Contains(%v, %v) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
}

func TestPDRectangleCreateRetranslatedRectangle(t *testing.T) {
	r := NewPDRectangleOf(100, 100, 300, 300)
	got := r.CreateRetranslatedRectangle()

	if want := "[0.0,0.0,300.0,300.0]"; got.String() != want {
		t.Errorf("retranslated = %v, want %v", got, want)
	}
}

func TestPDRectangleSetters(t *testing.T) {
	r := NewPDRectangle()
	r.SetLowerLeftX(1)
	r.SetLowerLeftY(2)
	r.SetUpperRightX(3)
	r.SetUpperRightY(4)

	if got, want := r.String(), "[1.0,2.0,3.0,4.0]"; got != want {
		t.Errorf("rectangle = %v, want %v", got, want)
	}
	// The COS array behind it is what was changed.
	if got := r.COSArray().ToFloatArray(); got[0] != 1 || got[3] != 4 {
		t.Errorf("COS array = %v, want [1 2 3 4]", got)
	}
}

func TestPDRectangleToGeneralPath(t *testing.T) {
	path := NewPDRectangleOf(1, 2, 3, 4).ToGeneralPath()

	if got, want := path.Bounds2D().Width, 3.0; got != want {
		t.Errorf("path width = %v, want %v", got, want)
	}
	if got, want := path.Bounds2D().X, 1.0; got != want {
		t.Errorf("path x = %v, want %v", got, want)
	}
}

func TestPDRectangleTransform(t *testing.T) {
	path := NewPDRectangleOf(0, 0, 10, 10).Transform(util.TranslateInstance(5, 5))

	bounds := path.Bounds2D()
	if bounds.X != 5 || bounds.Y != 5 || bounds.Width != 10 || bounds.Height != 10 {
		t.Errorf("transformed bounds = %v, want [5 5 10 10]", bounds)
	}
}

// TestPDRectangleCOSObject pins that the rectangle hands back the very array it
// holds, not a copy: callers write a rectangle into a dictionary and then keep
// changing it through the wrapper.
func TestPDRectangleCOSObject(t *testing.T) {
	r := NewPDRectangleOf(1, 2, 3, 4)
	if r.COSObject() != cos.Base(r.COSArray()) {
		t.Error("COSObject and COSArray disagree")
	}
	r.SetLowerLeftX(9)
	if got := r.COSArray().ToFloatArray()[0]; got != 9 {
		t.Errorf("the array behind the rectangle holds %v, want 9", got)
	}
}

func TestPredefinedSizes(t *testing.T) {
	// 8.5" x 11" at 72 user space units per inch.
	if got, want := Letter.Width(), float32(612); got != want {
		t.Errorf("Letter width = %v, want %v", got, want)
	}
	if got, want := Letter.Height(), float32(792); got != want {
		t.Errorf("Letter height = %v, want %v", got, want)
	}
	// A4 is 210mm x 297mm, at 1/25.4 inch per mm.
	if got := A4.Width(); got < 595 || got > 596 {
		t.Errorf("A4 width = %v, want about 595.3", got)
	}
	if got := A4.Height(); got < 841 || got > 842 {
		t.Errorf("A4 height = %v, want about 841.9", got)
	}
}
