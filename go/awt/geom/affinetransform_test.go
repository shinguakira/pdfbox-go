package geom

import (
	"errors"
	"math"
	"testing"
)

// Written from the java.awt.geom.AffineTransform specification. There is no
// PDFBox test for this class — it belongs to the JDK — so per
// migration/conventions/tdd.md the expected values come from the documented
// behaviour, not from this port.

func assertMatrix(t *testing.T, at *AffineTransform, want [6]float64) {
	t.Helper()
	var got [6]float64
	at.GetMatrix(got[:])
	for i := range want {
		if math.Abs(got[i]-want[i]) > 1e-12 {
			t.Errorf("matrix = %v, want %v", got, want)
			return
		}
	}
}

// TestNewAffineTransformArgumentOrder pins the constructor's column-major
// argument order, which is the easiest thing in the class to get wrong:
// AffineTransform(m00, m10, m01, m11, m02, m12).
func TestNewAffineTransformArgumentOrder(t *testing.T) {
	at := NewAffineTransform(1, 2, 3, 4, 5, 6)

	if got := at.ScaleX(); got != 1 {
		t.Errorf("ScaleX = %v, want 1 (m00)", got)
	}
	if got := at.ShearY(); got != 2 {
		t.Errorf("ShearY = %v, want 2 (m10)", got)
	}
	if got := at.ShearX(); got != 3 {
		t.Errorf("ShearX = %v, want 3 (m01)", got)
	}
	if got := at.ScaleY(); got != 4 {
		t.Errorf("ScaleY = %v, want 4 (m11)", got)
	}
	if got := at.TranslateX(); got != 5 {
		t.Errorf("TranslateX = %v, want 5 (m02)", got)
	}
	if got := at.TranslateY(); got != 6 {
		t.Errorf("TranslateY = %v, want 6 (m12)", got)
	}
	// GetMatrix hands back the same order the constructor takes.
	assertMatrix(t, at, [6]float64{1, 2, 3, 4, 5, 6})
}

func TestIdentity(t *testing.T) {
	at := NewIdentityTransform()
	assertMatrix(t, at, [6]float64{1, 0, 0, 1, 0, 0})
	if !at.IsIdentity() {
		t.Error("IsIdentity = false for a new transform")
	}
	if got := at.Determinant(); got != 1 {
		t.Errorf("Determinant = %v, want 1", got)
	}

	at.Scale(2, 2)
	if at.IsIdentity() {
		t.Error("IsIdentity = true after scaling")
	}
	at.SetToIdentity()
	if !at.IsIdentity() {
		t.Error("IsIdentity = false after SetToIdentity")
	}
}

func TestTransformPoint(t *testing.T) {
	// x' = m00 x + m01 y + m02, y' = m10 x + m11 y + m12
	at := NewAffineTransform(2, 3, 4, 5, 6, 7)
	dst := NewPointDouble(0, 0)
	at.Transform(NewPointDouble(1, 1), dst)

	if got, want := dst.X(), 2.0*1+4*1+6; got != want {
		t.Errorf("x = %v, want %v", got, want)
	}
	if got, want := dst.Y(), 3.0*1+5*1+7; got != want {
		t.Errorf("y = %v, want %v", got, want)
	}
}

// TestDeltaTransform pins that the translation is left out of a delta
// transform, which is what makes it right for a vector rather than a point.
func TestDeltaTransform(t *testing.T) {
	at := NewAffineTransform(2, 3, 4, 5, 6, 7)
	dst := NewPointDouble(0, 0)
	at.DeltaTransform(NewPointDouble(1, 1), dst)

	if got, want := dst.X(), 6.0; got != want {
		t.Errorf("x = %v, want %v", got, want)
	}
	if got, want := dst.Y(), 8.0; got != want {
		t.Errorf("y = %v, want %v", got, want)
	}
}

func TestTransformFloatArray(t *testing.T) {
	at := TranslateInstance(10, 20)
	src := []float32{0, 0, 1, 2}
	dst := make([]float32, 4)
	at.TransformFloats(src, 0, dst, 0, 2)

	want := []float32{10, 20, 11, 22}
	for i := range want {
		if dst[i] != want[i] {
			t.Fatalf("TransformFloats = %v, want %v", dst, want)
		}
	}
}

// TestConcatenateOrder pins that concatenate applies the argument first and the
// receiver second, while preConcatenate does the reverse. Getting these the
// wrong way round silently produces a plausible but wrong transform.
func TestConcatenateOrder(t *testing.T) {
	scale := ScaleInstance(2, 2)
	translate := TranslateInstance(10, 0)

	// concatenate: the point is translated, then scaled.
	at := scale.Clone()
	at.Concatenate(translate)
	dst := NewPointDouble(0, 0)
	at.Transform(NewPointDouble(1, 0), dst)
	if got, want := dst.X(), 22.0; got != want {
		t.Errorf("concatenate: x = %v, want %v", got, want)
	}

	// preConcatenate: the point is scaled, then translated.
	at = scale.Clone()
	at.PreConcatenate(translate)
	at.Transform(NewPointDouble(1, 0), dst)
	if got, want := dst.X(), 12.0; got != want {
		t.Errorf("preConcatenate: x = %v, want %v", got, want)
	}
}

func TestTranslate(t *testing.T) {
	at := ScaleInstance(2, 3)
	at.Translate(4, 5)
	// The translation is expressed in the transform's own coordinates, so it is
	// scaled by the existing matrix.
	assertMatrix(t, at, [6]float64{2, 0, 0, 3, 8, 15})
}

func TestScale(t *testing.T) {
	at := NewAffineTransform(1, 2, 3, 4, 5, 6)
	at.Scale(2, 3)
	assertMatrix(t, at, [6]float64{2, 4, 9, 12, 5, 6})
}

func TestShear(t *testing.T) {
	at := NewIdentityTransform()
	at.Shear(2, 3)
	assertMatrix(t, at, [6]float64{1, 3, 2, 1, 0, 0})
}

// TestRotateQuadrant pins the documented special case: at a quadrant angle the
// cosine that Math.cos returns is not quite zero, so the class substitutes an
// exact value rather than letting 6.1e-17 leak into every later product.
func TestRotateQuadrant(t *testing.T) {
	at := NewIdentityTransform()
	at.Rotate(math.Pi / 2)
	assertMatrix(t, at, [6]float64{0, 1, -1, 0, 0, 0})

	dst := NewPointDouble(0, 0)
	at.Transform(NewPointDouble(1, 0), dst)
	if dst.X() != 0 || dst.Y() != 1 {
		t.Errorf("rotated (1,0) to (%v, %v), want exactly (0, 1)", dst.X(), dst.Y())
	}

	// cos(pi) is exactly -1, so the half turn is exact as well.
	at = NewIdentityTransform()
	at.Rotate(math.Pi)
	assertMatrix(t, at, [6]float64{-1, 0, 0, -1, 0, 0})

	// A quarter turn the other way.
	at = NewIdentityTransform()
	at.Rotate(-math.Pi / 2)
	assertMatrix(t, at, [6]float64{0, -1, 1, 0, 0, 0})

	// A rotation of zero leaves the transform alone.
	at = NewAffineTransform(1, 2, 3, 4, 5, 6)
	at.Rotate(0)
	assertMatrix(t, at, [6]float64{1, 2, 3, 4, 5, 6})
}

func TestRotateGeneralAngle(t *testing.T) {
	at := NewIdentityTransform()
	at.Rotate(math.Pi / 4)
	cos := math.Cos(math.Pi / 4)
	sin := math.Sin(math.Pi / 4)
	assertMatrix(t, at, [6]float64{cos, sin, -sin, cos, 0, 0})
}

func TestRotateInstance(t *testing.T) {
	at := RotateInstance(math.Pi)
	dst := NewPointDouble(0, 0)
	at.Transform(NewPointDouble(1, 0), dst)
	if dst.X() != -1 || dst.Y() != 0 {
		t.Errorf("rotated (1,0) to (%v, %v), want exactly (-1, 0)", dst.X(), dst.Y())
	}
}

func TestCreateInverse(t *testing.T) {
	at := NewAffineTransform(2, 0, 0, 4, 10, 20)
	inverse, err := at.CreateInverse()
	if err != nil {
		t.Fatalf("CreateInverse: %v", err)
	}
	assertMatrix(t, inverse, [6]float64{0.5, 0, 0, 0.25, -5, -5})

	// Round-tripping a point through both must give it back.
	dst := NewPointDouble(0, 0)
	at.Transform(NewPointDouble(3, 7), dst)
	inverse.Transform(dst, dst)
	if math.Abs(dst.X()-3) > 1e-12 || math.Abs(dst.Y()-7) > 1e-12 {
		t.Errorf("round trip gave (%v, %v), want (3, 7)", dst.X(), dst.Y())
	}
}

func TestCreateInverseNoninvertible(t *testing.T) {
	// A determinant of zero collapses the plane onto a line, which cannot be
	// undone. Java throws NoninvertibleTransformException here.
	at := NewAffineTransform(0, 0, 0, 0, 0, 0)
	if _, err := at.CreateInverse(); !errors.Is(err, ErrNoninvertibleTransform) {
		t.Errorf("CreateInverse err = %v, want ErrNoninvertibleTransform", err)
	}
}

func TestInverseTransform(t *testing.T) {
	at := NewAffineTransform(2, 0, 0, 4, 10, 20)
	dst := NewPointDouble(0, 0)
	if err := at.InverseTransform(NewPointDouble(12, 24), dst); err != nil {
		t.Fatalf("InverseTransform: %v", err)
	}
	if dst.X() != 1 || dst.Y() != 1 {
		t.Errorf("InverseTransform gave (%v, %v), want (1, 1)", dst.X(), dst.Y())
	}
}

func TestDeterminant(t *testing.T) {
	// m00 * m11 - m01 * m10
	if got := NewAffineTransform(1, 2, 3, 4, 5, 6).Determinant(); got != -2 {
		t.Errorf("Determinant = %v, want -2", got)
	}
}

func TestCloneIsIndependent(t *testing.T) {
	at := NewAffineTransform(1, 2, 3, 4, 5, 6)
	clone := at.Clone()
	clone.Scale(10, 10)

	assertMatrix(t, at, [6]float64{1, 2, 3, 4, 5, 6})
	if at.Equals(clone) {
		t.Error("the clone shares state with the original")
	}
}

func TestEquals(t *testing.T) {
	at := NewAffineTransform(1, 2, 3, 4, 5, 6)
	if !at.Equals(NewAffineTransform(1, 2, 3, 4, 5, 6)) {
		t.Error("Equals = false for equal transforms")
	}
	if at.Equals(NewAffineTransform(1, 2, 3, 4, 5, 7)) {
		t.Error("Equals = true for differing transforms")
	}
	if at.Equals(nil) {
		t.Error("Equals = true against nil")
	}
}
