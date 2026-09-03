package util

import (
	"math"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Port of pdfbox/src/test/java/org/apache/pdfbox/util/MatrixTest.java.
//
// testMultiplicationPerformance is not ported: it is commented out in the Java
// and only times a hundred million multiplications.

// assertMatrixIsPristine asserts that the matrix values for the given Matrix
// object are equal to the pristine, or original, values.
func assertMatrixIsPristine(t *testing.T, m *Matrix) {
	t.Helper()
	assertMatrixValuesEqualTo(t, []float32{1, 0, 0, 0, 1, 0, 0, 0, 1}, m)
}

// assertMatrixValuesEqualTo asserts that the matrix values for the given Matrix
// object have the specified values.
func assertMatrixValuesEqualTo(t *testing.T, values []float32, m *Matrix) {
	t.Helper()
	const delta = 0.00001
	for i := range values {
		// Need to convert a (row, column) coordinate into a straight index.
		row := i / 3
		column := i % 3
		if math.Abs(float64(values[i]-m.Value(row, column))) > delta {
			t.Errorf("Incorrect value for matrix[%d,%d]: got %v, want %v",
				row, column, m.Value(row, column), values[i])
		}
	}
}

func TestConstructionAndCopy(t *testing.T) {
	m1 := NewMatrix()
	assertMatrixIsPristine(t, m1)

	m2 := m1.Clone()
	if m1 == m2 {
		t.Error("Clone returned the same matrix")
	}
	assertMatrixIsPristine(t, m2)
}

func TestGetScalingFactor(t *testing.T) {
	// check scaling factor of an initial matrix
	m1 := NewMatrix()
	if got := m1.ScalingFactorX(); got != 1 {
		t.Errorf("ScalingFactorX = %v, want 1", got)
	}
	if got := m1.ScalingFactorY(); got != 1 {
		t.Errorf("ScalingFactorY = %v, want 1", got)
	}

	// check scaling factor of an initial matrix
	m2 := NewMatrixOf(2, 4, 4, 2, 0, 0)
	want := float32(math.Sqrt(20))
	if got := m2.ScalingFactorX(); got != want {
		t.Errorf("ScalingFactorX = %v, want %v", got, want)
	}
	if got := m2.ScalingFactorY(); got != want {
		t.Errorf("ScalingFactorY = %v, want %v", got, want)
	}
}

func TestCreateMatrixUsingInvalidInput(t *testing.T) {
	// anything but a COSArray is invalid and leads to an initial matrix
	createMatrix := CreateMatrix(cos.A)
	assertMatrixIsPristine(t, createMatrix)

	// a COSArray with fewer than 6 entries leads to an initial matrix
	cosArray := cos.NewArray()
	cosArray.Add(cos.A)
	createMatrix = CreateMatrix(cosArray)
	assertMatrixIsPristine(t, createMatrix)

	// a COSArray containing other kind of objects than COSNumber leads to an initial matrix
	cosArray = cos.NewArray()
	for i := 0; i < 6; i++ {
		cosArray.Add(cos.A)
	}
	createMatrix = CreateMatrix(cosArray)
	assertMatrixIsPristine(t, createMatrix)
}

func TestMultiplication(t *testing.T) {
	// These matrices will not change - we use it to drive the various multiplications.
	const1 := NewMatrix()
	const2 := NewMatrix()

	// Create matrix with values
	// [ 0, 1, 2
	// 1, 2, 3
	// 2, 3, 4]
	for x := 0; x < 3; x++ {
		for y := 0; y < 3; y++ {
			const1.SetValue(x, y, float32(x+y))
			const2.SetValue(x, y, float32(8+x+y))
		}
	}

	m1MultipliedByM1 := []float32{5, 8, 11, 8, 14, 20, 11, 20, 29}
	m1MultipliedByM2 := []float32{29, 32, 35, 56, 62, 68, 83, 92, 101}
	m2MultipliedByM1 := []float32{29, 56, 83, 32, 62, 92, 35, 68, 101}

	var1 := const1.Clone()
	var2 := const2.Clone()

	// Multiply two matrices together producing a new result matrix.
	result := var1.Multiply(var2)
	assertMatrixEquals(t, const1, var1)
	assertMatrixEquals(t, const2, var2)
	assertMatrixValuesEqualTo(t, m1MultipliedByM2, result)

	// Multiply two matrices together with the result being written to a third matrix
	// (Any existing values there will be overwritten).
	result = var1.Multiply(var2)
	assertMatrixEquals(t, const1, var1)
	assertMatrixEquals(t, const2, var2)
	assertMatrixValuesEqualTo(t, m1MultipliedByM2, result)

	// Multiply two matrices together with the result being written into 'this' matrix
	var1 = const1.Clone()
	var2 = const2.Clone()
	var1.Concatenate(var2)
	assertMatrixEquals(t, const2, var2)
	assertMatrixValuesEqualTo(t, m2MultipliedByM1, var1)

	var1 = const1.Clone()
	var2 = const2.Clone()
	result = Concatenate(var1, var2)
	assertMatrixEquals(t, const1, var1)
	assertMatrixEquals(t, const2, var2)
	assertMatrixValuesEqualTo(t, m2MultipliedByM1, result)

	// Multiply the same matrix with itself with the result being written into 'this' matrix
	var1 = const1.Clone()
	result = var1.Multiply(var1)
	assertMatrixEquals(t, const1, var1)
	assertMatrixValuesEqualTo(t, m1MultipliedByM1, result)
}

func assertMatrixEquals(t *testing.T, want, got *Matrix) {
	t.Helper()
	if !want.Equals(got) {
		t.Errorf("matrix = %v, want %v", got, want)
	}
}

func TestOldMultiplication(t *testing.T) {
	// This matrix will not change - we use it to drive the various multiplications.
	testMatrix := NewMatrix()

	// Create matrix with values
	// [ 0, 1, 2
	// 1, 2, 3
	// 2, 3, 4]
	for x := 0; x < 3; x++ {
		for y := 0; y < 3; y++ {
			testMatrix.SetValue(x, y, float32(x+y))
		}
	}

	m1 := testMatrix.Clone()
	m2 := testMatrix.Clone()

	// Multiply two matrices together producing a new result matrix.
	product := m1.Multiply(m2)

	if m1 == product {
		t.Error("the product is the first operand")
	}
	if m2 == product {
		t.Error("the product is the second operand")
	}

	// Operand 1 should not have changed
	assertMatrixValuesEqualTo(t, []float32{0, 1, 2, 1, 2, 3, 2, 3, 4}, m1)
	// Operand 2 should not have changed
	assertMatrixValuesEqualTo(t, []float32{0, 1, 2, 1, 2, 3, 2, 3, 4}, m2)
	assertMatrixValuesEqualTo(t, []float32{5, 8, 11, 8, 14, 20, 11, 20, 29}, product)

	retVal := m1.Multiply(m2)
	// Operand 1 should not have changed
	assertMatrixValuesEqualTo(t, []float32{0, 1, 2, 1, 2, 3, 2, 3, 4}, m1)
	// Operand 2 should not have changed
	assertMatrixValuesEqualTo(t, []float32{0, 1, 2, 1, 2, 3, 2, 3, 4}, m2)
	assertMatrixValuesEqualTo(t, []float32{5, 8, 11, 8, 14, 20, 11, 20, 29}, retVal)

	// Multiply the same matrix with itself with the result being written into 'this' matrix
	m1 = testMatrix.Clone()

	retVal = m1.Multiply(m1)
	// Operand 1 should not have changed
	assertMatrixValuesEqualTo(t, []float32{0, 1, 2, 1, 2, 3, 2, 3, 4}, m1)
	assertMatrixValuesEqualTo(t, []float32{5, 8, 11, 8, 14, 20, 11, 20, 29}, retVal)
}

// assertPanics stands in for JUnit's assertThrows: Java raises the unchecked
// IllegalArgumentException here and nothing in PDFBox catches it, so the port
// panics rather than growing an error return on every arithmetic method.
func assertPanics(t *testing.T, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("no panic, want one")
		}
	}()
	f()
}

func TestIllegalValueNaN1(t *testing.T) {
	m := NewMatrix()
	m.SetValue(0, 0, math.MaxFloat32)
	assertPanics(t, func() { m.Multiply(m) })
}

func TestIllegalValueNaN2(t *testing.T) {
	m := NewMatrix()
	m.SetValue(0, 0, float32(math.NaN()))
	assertPanics(t, func() { m.Multiply(m) })
}

func TestIllegalValuePositiveInfinity(t *testing.T) {
	m := NewMatrix()
	m.SetValue(0, 0, float32(math.Inf(1)))
	assertPanics(t, func() { m.Multiply(m) })
}

func TestIllegalValueNegativeInfinity(t *testing.T) {
	m := NewMatrix()
	m.SetValue(0, 0, float32(math.Inf(-1)))
	assertPanics(t, func() { m.Multiply(m) })
}

// TestPdfbox2872 is the test of the PDFBOX-2872 bug.
func TestPdfbox2872(t *testing.T) {
	m := NewMatrixOf(2, 4, 5, 8, 2, 0)
	toCOSArray := m.ToCOSArray()
	want := []float32{2, 4, 5, 8, 2, 0}
	for i, w := range want {
		got, ok := toCOSArray.Get(i).(*cos.Float)
		if !ok {
			t.Fatalf("element %d = %T, want *cos.Float", i, toCOSArray.Get(i))
		}
		if !got.Equals(cos.NewFloat(w)) {
			t.Errorf("element %d = %v, want %v", i, got, w)
		}
	}
}

func TestGetValues(t *testing.T) {
	m := NewMatrixOf(2, 4, 4, 2, 15, 30)
	values := m.Values()
	want := [3][3]float32{{2, 4, 0}, {4, 2, 0}, {15, 30, 1}}
	if values != want {
		t.Errorf("Values() = %v, want %v", values, want)
	}
}

func TestScaling(t *testing.T) {
	m := NewMatrixOf(2, 4, 4, 2, 15, 30)
	m.Scale(2, 3)
	assertMatrixValuesEqualTo(t, []float32{
		// first row, multiplication with 2
		4, 8, 0,
		// second row, multiplication with 3
		12, 6, 0,
		// third row, no changes at all
		15, 30, 1,
	}, m)
}

func TestTranslation(t *testing.T) {
	m := NewMatrixOf(2, 4, 4, 2, 15, 30)
	m.Translate(2, 3)
	assertMatrixValuesEqualTo(t, []float32{
		// first row, no changes at all
		2, 4, 0,
		// second row, no changes at all
		4, 2, 0,
		// third row, translated values
		31, 44, 1,
	}, m)
}

// TestEqualsFollowsArraysEquals pins the comparison rule Java's Arrays.equals
// uses on the float array behind a matrix, which is not the one == gives:
// NaN equals itself, and +0.0 differs from -0.0. The Java suite does not cover
// this, so the expectations come from the Arrays.equals contract.
func TestEqualsFollowsArraysEquals(t *testing.T) {
	nan1 := NewMatrix()
	nan1.SetValue(0, 0, float32(math.NaN()))
	nan2 := NewMatrix()
	nan2.SetValue(0, 0, float32(math.NaN()))
	if !nan1.Equals(nan2) {
		t.Error("two matrices holding NaN compare unequal")
	}

	positiveZero := NewMatrix()
	positiveZero.SetValue(0, 0, 0)
	negativeZero := NewMatrix()
	negativeZero.SetValue(0, 0, float32(math.Copysign(0, -1)))
	if positiveZero.Equals(negativeZero) {
		t.Error("+0.0 and -0.0 compare equal")
	}

	if NewMatrix().Equals(nil) {
		t.Error("Equals = true against nil")
	}
}

func TestString(t *testing.T) {
	if got, want := NewMatrixOf(2, 4, 5, 8, 2, 0).String(), "[2.0,4.0,5.0,8.0,2.0,0.0]"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
