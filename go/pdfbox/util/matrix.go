package util

import (
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/internal/javafmt"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// Size is the number of values a matrix holds.
const Size = 9

// Matrix is a 3x3 transformation matrix.
//
// Port of org.apache.pdfbox.util.Matrix. The values are held in row-major order
//
//	a  b  0      single[0] single[1] single[2]
//	c  d  0  =   single[3] single[4] single[5]
//	tx ty 1      single[6] single[7] single[8]
//
// note: hx and hy are reversed vs. the PDF spec as we use AffineTransform's
// definition x and y shear.
type Matrix struct {
	single [Size]float32
}

// NewMatrix returns the identity matrix.
func NewMatrix() *Matrix {
	return &Matrix{single: [Size]float32{1, 0, 0, 0, 1, 0, 0, 0, 1}}
}

// NewMatrixOf returns a transformation matrix with the given 6 elements.
// Transformation matrices are discussed in 8.3.3, "Common Transformations" and
// 8.3.4, "Transformation Matrices" of the PDF specification. For simple
// purposes (rotate, scale, translate) it is recommended to use RotateInstance,
// ScaleInstance and TranslateInstance.
//
// Produces the following matrix:
//
//	a b 0
//	c d 0
//	e f 1
//
// where a is the X coordinate scaling element (m00) of the 3x3 matrix, b the Y
// coordinate shearing element (m10), c the X coordinate shearing element (m01),
// d the Y coordinate scaling element (m11), e the X coordinate translation
// element (m02) and f the Y coordinate translation element (m12).
func NewMatrixOf(a, b, c, d, e, f float32) *Matrix {
	return &Matrix{single: [Size]float32{a, b, 0, c, d, 0, e, f, 1}}
}

// newMatrixFromCOSArray creates a matrix from a 6-element (a b c d e f) COS
// array whose elements must all be numbers.
func newMatrixFromCOSArray(array *cos.Array) *Matrix {
	m := &Matrix{}
	m.single[0] = array.GetObject(0).(cos.Number).FloatValue()
	m.single[1] = array.GetObject(1).(cos.Number).FloatValue()
	m.single[3] = array.GetObject(2).(cos.Number).FloatValue()
	m.single[4] = array.GetObject(3).(cos.Number).FloatValue()
	m.single[6] = array.GetObject(4).(cos.Number).FloatValue()
	m.single[7] = array.GetObject(5).(cos.Number).FloatValue()
	m.single[8] = 1
	return m
}

// NewMatrixFromAffineTransform creates a matrix with the same elements as the
// given AffineTransform, as follows:
//
//	scaleX shearY 0
//	shearX scaleY 0
//	transX transY 1
func NewMatrixFromAffineTransform(at *geom.AffineTransform) *Matrix {
	m := &Matrix{}
	m.single[0] = float32(at.ScaleX())
	m.single[1] = float32(at.ShearY())
	m.single[3] = float32(at.ShearX())
	m.single[4] = float32(at.ScaleY())
	m.single[6] = float32(at.TranslateX())
	m.single[7] = float32(at.TranslateY())
	m.single[8] = 1
	return m
}

// CreateMatrix is a convenience function to be used when creating a matrix from
// unverified data. If the parameter is a COSArray with at least six numbers, a
// Matrix is created from the first six numbers and returned. If not, then the
// identity matrix is returned.
func CreateMatrix(base cos.Base) *Matrix {
	array, ok := base.(*cos.Array)
	if !ok {
		return NewMatrix()
	}
	if array.Size() < 6 {
		return NewMatrix()
	}
	for i := 0; i < 6; i++ {
		if _, ok := array.GetObject(i).(cos.Number); !ok {
			return NewMatrix()
		}
	}
	return newMatrixFromCOSArray(array)
}

// CreateAffineTransform returns an affine transform with this matrix's values.
func (m *Matrix) CreateAffineTransform() *geom.AffineTransform {
	return geom.NewAffineTransform(
		float64(m.single[0]), float64(m.single[1]), // m00 m10 = scaleX shearY
		float64(m.single[3]), float64(m.single[4]), // m01 m11 = shearX scaleY
		float64(m.single[6]), float64(m.single[7])) // m02 m12 = tx ty
}

// Value returns the value at the given row and column.
func (m *Matrix) Value(row, column int) float32 {
	return m.single[row*3+column]
}

// SetValue sets the value at the given row and column.
func (m *Matrix) SetValue(row, column int, value float32) {
	m.single[row*3+column] = value
}

// Values returns all the values of this matrix.
func (m *Matrix) Values() [3][3]float32 {
	return [3][3]float32{
		{m.single[0], m.single[1], m.single[2]},
		{m.single[3], m.single[4], m.single[5]},
		{m.single[6], m.single[7], m.single[8]},
	}
}

// Concatenate premultiplies the given matrix into this one.
func (m *Matrix) Concatenate(matrix *Matrix) {
	m.single = checkFloatValues(multiplyArrays(matrix.single, m.single))
}

// TranslateVector translates this matrix by the given vector.
func (m *Matrix) TranslateVector(vector Vector) {
	m.Translate(vector.X(), vector.Y())
}

// Translate translates this matrix by the given amount.
func (m *Matrix) Translate(tx, ty float32) {
	m.single[6] += tx*m.single[0] + ty*m.single[3]
	m.single[7] += tx*m.single[1] + ty*m.single[4]
	m.single[8] += tx*m.single[2] + ty*m.single[5]
	checkFloatValues(m.single)
}

// Scale scales this matrix by the given factors.
func (m *Matrix) Scale(sx, sy float32) {
	m.single[0] *= sx
	m.single[1] *= sx
	m.single[2] *= sx
	m.single[3] *= sy
	m.single[4] *= sy
	m.single[5] *= sy
	checkFloatValues(m.single)
}

// Rotate rotates this matrix by theta radians.
func (m *Matrix) Rotate(theta float64) {
	m.Concatenate(RotateInstance(theta, 0, 0))
}

// Multiply returns the product of this matrix and other, in a new matrix. It is
// allowed to have other == m.
func (m *Matrix) Multiply(other *Matrix) *Matrix {
	return &Matrix{single: checkFloatValues(multiplyArrays(m.single, other.single))}
}

// checkFloatValues rejects a matrix that has run off the end of the float
// range.
//
// Java throws the unchecked IllegalArgumentException here and nothing in
// PDFBox catches it, so the port panics rather than putting an error return on
// every arithmetic method.
func checkFloatValues(values [Size]float32) [Size]float32 {
	for _, v := range values {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			panic("util: multiplying two matrices produces illegal values")
		}
	}
	return values
}

func multiplyArrays(a, b [Size]float32) [Size]float32 {
	var c [Size]float32
	c[0] = a[0]*b[0] + a[1]*b[3] + a[2]*b[6]
	c[1] = a[0]*b[1] + a[1]*b[4] + a[2]*b[7]
	c[2] = a[0]*b[2] + a[1]*b[5] + a[2]*b[8]
	c[3] = a[3]*b[0] + a[4]*b[3] + a[5]*b[6]
	c[4] = a[3]*b[1] + a[4]*b[4] + a[5]*b[7]
	c[5] = a[3]*b[2] + a[4]*b[5] + a[5]*b[8]
	c[6] = a[6]*b[0] + a[7]*b[3] + a[8]*b[6]
	c[7] = a[6]*b[1] + a[7]*b[4] + a[8]*b[7]
	c[8] = a[6]*b[2] + a[7]*b[5] + a[8]*b[8]
	return c
}

// TransformPoint2D transforms the given point by this matrix, in place.
func (m *Matrix) TransformPoint2D(point geom.Point2D) {
	x := float32(point.X())
	y := float32(point.Y())
	a := m.single[0]
	b := m.single[1]
	c := m.single[3]
	d := m.single[4]
	e := m.single[6]
	f := m.single[7]
	point.SetLocation(float64(x*a+y*c+e), float64(x*b+y*d+f))
}

// TransformPoint returns the given point transformed by this matrix.
func (m *Matrix) TransformPoint(x, y float32) *geom.PointFloat {
	a := m.single[0]
	b := m.single[1]
	c := m.single[3]
	d := m.single[4]
	e := m.single[6]
	f := m.single[7]
	return geom.NewPointFloat(x*a+y*c+e, x*b+y*d+f)
}

// TransformVector returns the given vector transformed by this matrix.
func (m *Matrix) TransformVector(vector Vector) Vector {
	a := m.single[0]
	b := m.single[1]
	c := m.single[3]
	d := m.single[4]
	e := m.single[6]
	f := m.single[7]
	x := vector.X()
	y := vector.Y()
	return NewVector(x*a+y*c+e, x*b+y*d+f)
}

// ScaleInstance returns a matrix with just the x/y scaling:
//
//	x 0 0
//	0 y 0
//	0 0 1
func ScaleInstance(x, y float32) *Matrix {
	return NewMatrixOf(x, 0, 0, y, 0, 0)
}

// TranslateInstance returns a matrix with just the x/y translating:
//
//	1 0 0
//	0 1 0
//	x y 1
func TranslateInstance(x, y float32) *Matrix {
	return NewMatrixOf(1, 0, 0, 1, x, y)
}

// RotateInstance returns a matrix with a rotation of theta radians and the x/y
// translating.
func RotateInstance(theta float64, tx, ty float32) *Matrix {
	cosTheta := float32(math.Cos(theta))
	sinTheta := float32(math.Sin(theta))

	return NewMatrixOf(cosTheta, sinTheta, -sinTheta, cosTheta, tx, ty)
}

// Concatenate produces a copy of the first matrix, with the second matrix
// concatenated.
func Concatenate(a, b *Matrix) *Matrix {
	return b.Multiply(a)
}

// Clone returns an independent copy of this matrix.
func (m *Matrix) Clone() *Matrix {
	clone := *m
	return &clone
}

// ScalingFactorX returns the x-scaling factor of this matrix, calculated from
// the scale and shear.
func (m *Matrix) ScalingFactorX() float32 {
	// BM: if the trm is rotated, the calculation is a little more complicated
	//
	// The rotation matrix multiplied with the scaling matrix is:
	// (   x   0   0)    ( cos  sin  0)    ( x*cos x*sin   0)
	// (   0   y   0) *  (-sin  cos  0)  = (-y*sin y*cos   0)
	// (   0   0   1)    (   0    0  1)    (     0     0   1)
	//
	// So, if you want to deduce x from the matrix you take
	// M(0,0) = x*cos and M(0,1) = x*sin and use the theorem of Pythagoras
	//
	// sqrt(M(0,0)^2+M(0,1)^2) =
	// sqrt(x2*cos2+x2*sin2) =
	// sqrt(x2*(cos2+sin2)) = <- here is the trick cos2+sin2 is one
	// sqrt(x2) =
	// abs(x)
	if m.single[1] != 0 {
		return float32(math.Sqrt(math.Pow(float64(m.single[0]), 2) +
			math.Pow(float64(m.single[1]), 2)))
	}
	return m.single[0]
}

// ScalingFactorY returns the y-scaling factor of this matrix, calculated from
// the scale and shear.
func (m *Matrix) ScalingFactorY() float32 {
	if m.single[3] != 0 {
		return float32(math.Sqrt(math.Pow(float64(m.single[3]), 2) +
			math.Pow(float64(m.single[4]), 2)))
	}
	return m.single[4]
}

// ScaleX returns the x-scaling element of this matrix. See ScalingFactorX.
func (m *Matrix) ScaleX() float32 { return m.single[0] }

// ShearY returns the y-shear element of this matrix.
func (m *Matrix) ShearY() float32 { return m.single[1] }

// ShearX returns the x-shear element of this matrix.
func (m *Matrix) ShearX() float32 { return m.single[3] }

// ScaleY returns the y-scaling element of this matrix. See ScalingFactorY.
func (m *Matrix) ScaleY() float32 { return m.single[4] }

// TranslateX returns the x-translation element of this matrix.
func (m *Matrix) TranslateX() float32 { return m.single[6] }

// TranslateY returns the y-translation element of this matrix.
func (m *Matrix) TranslateY() float32 { return m.single[7] }

// ToCOSArray returns a COS array holding the geometrically relevant components
// of the matrix. The last column is ignored, only the first two are returned.
// This is analogous to newMatrixFromCOSArray.
func (m *Matrix) ToCOSArray() *cos.Array {
	array := cos.NewArray()
	array.Add(cos.NewFloat(m.single[0]))
	array.Add(cos.NewFloat(m.single[1]))
	array.Add(cos.NewFloat(m.single[3]))
	array.Add(cos.NewFloat(m.single[4]))
	array.Add(cos.NewFloat(m.single[6]))
	array.Add(cos.NewFloat(m.single[7]))
	return array
}

// String returns the Java toString form.
func (m *Matrix) String() string {
	return "[" +
		javafmt.Float32(m.single[0]) + "," +
		javafmt.Float32(m.single[1]) + "," +
		javafmt.Float32(m.single[3]) + "," +
		javafmt.Float32(m.single[4]) + "," +
		javafmt.Float32(m.single[6]) + "," +
		javafmt.Float32(m.single[7]) + "]"
}

// Equals reports whether other holds the same nine values.
func (m *Matrix) Equals(other *Matrix) bool {
	if m == other {
		return true
	}
	if other == nil {
		return false
	}
	for i := range m.single {
		if floatToIntBits(m.single[i]) != floatToIntBits(other.single[i]) {
			return false
		}
	}
	return true
}

// floatToIntBits mirrors Java's Float.floatToIntBits, which collapses every NaN
// to one bit pattern. Java compares the arrays behind two matrices with
// Arrays.equals, which uses this rather than ==, so NaN equals itself and +0.0
// differs from -0.0 — the opposite of what Go's == gives on both counts.
func floatToIntBits(v float32) uint32 {
	if math.IsNaN(float64(v)) {
		return 0x7fc00000
	}
	return math.Float32bits(v)
}
