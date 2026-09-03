package geom

import (
	"errors"
	"math"
)

// ErrNoninvertibleTransform reports a transform that cannot be inverted because
// it collapses the plane onto a line or a point.
//
// Port of java.awt.geom.NoninvertibleTransformException.
var ErrNoninvertibleTransform = errors.New("geom: transform is noninvertible")

// AffineTransform is a 2D affine transform, which maps straight lines to
// straight lines and keeps parallel lines parallel.
//
// Port of java.awt.geom.AffineTransform, holding
//
//	[ m00 m01 m02 ]   [ scaleX shearX translateX ]
//	[ m10 m11 m12 ] = [ shearY scaleY translateY ]
//	[  0   0   1  ]
//
// The JDK caches a "state" and a "type" alongside the six values so that it can
// skip terms it knows are zero. That is a speed optimisation over an immutable
// classification of the matrix, and reproducing it would only add ways for the
// cache and the values to disagree, so it is left out. Everything observable is
// the same, except that a few methods here do arithmetic the JDK would have
// skipped.
type AffineTransform struct {
	m00, m10, m01, m11, m02, m12 float64
}

// NewAffineTransform returns the transform with the given six values.
//
// The argument order is the JDK's and is column-major: m00, m10, m01, m11, m02,
// m12 — that is, scaleX, shearY, shearX, scaleY, translateX, translateY.
func NewAffineTransform(m00, m10, m01, m11, m02, m12 float64) *AffineTransform {
	return &AffineTransform{m00: m00, m10: m10, m01: m01, m11: m11, m02: m02, m12: m12}
}

// NewIdentityTransform returns a transform that leaves every point where it is.
func NewIdentityTransform() *AffineTransform {
	return &AffineTransform{m00: 1, m11: 1}
}

// TranslateInstance returns a transform that moves points by (tx, ty).
func TranslateInstance(tx, ty float64) *AffineTransform {
	return NewAffineTransform(1, 0, 0, 1, tx, ty)
}

// ScaleInstance returns a transform that scales by sx and sy.
func ScaleInstance(sx, sy float64) *AffineTransform {
	return NewAffineTransform(sx, 0, 0, sy, 0, 0)
}

// ShearInstance returns a transform that shears by shx and shy.
func ShearInstance(shx, shy float64) *AffineTransform {
	return NewAffineTransform(1, shy, shx, 1, 0, 0)
}

// RotateInstance returns a transform that rotates by theta radians about the
// origin, turning the positive x axis toward the positive y axis.
func RotateInstance(theta float64) *AffineTransform {
	at := &AffineTransform{}
	at.SetToRotation(theta)
	return at
}

// SetToIdentity resets the transform to the identity.
func (at *AffineTransform) SetToIdentity() {
	at.m00, at.m11 = 1, 1
	at.m10, at.m01, at.m02, at.m12 = 0, 0, 0, 0
}

// SetToRotation replaces the transform with a rotation of theta radians.
//
// Rotating by a quadrant is handled apart from the general case: Math.cos of a
// right angle is 6.1e-17 rather than zero, and letting that into the matrix
// would smear a tiny shear through everything the transform is later
// concatenated with. The exact value is substituted instead.
func (at *AffineTransform) SetToRotation(theta float64) {
	sin := math.Sin(theta)
	var cos float64
	if sin == 1.0 || sin == -1.0 {
		cos = 0.0
	} else {
		cos = math.Cos(theta)
		if cos == -1.0 {
			sin = 0.0
		} else if cos == 1.0 {
			sin = 0.0
		}
	}
	at.m00 = cos
	at.m10 = sin
	at.m01 = -sin
	at.m11 = cos
	at.m02 = 0.0
	at.m12 = 0.0
}

// ScaleX returns the m00 element.
func (at *AffineTransform) ScaleX() float64 { return at.m00 }

// ScaleY returns the m11 element.
func (at *AffineTransform) ScaleY() float64 { return at.m11 }

// ShearX returns the m01 element.
func (at *AffineTransform) ShearX() float64 { return at.m01 }

// ShearY returns the m10 element.
func (at *AffineTransform) ShearY() float64 { return at.m10 }

// TranslateX returns the m02 element.
func (at *AffineTransform) TranslateX() float64 { return at.m02 }

// TranslateY returns the m12 element.
func (at *AffineTransform) TranslateY() float64 { return at.m12 }

// GetMatrix fills flatmatrix with m00, m10, m01 and m11, and with m02 and m12
// as well when there is room for them.
func (at *AffineTransform) GetMatrix(flatmatrix []float64) {
	flatmatrix[0] = at.m00
	flatmatrix[1] = at.m10
	flatmatrix[2] = at.m01
	flatmatrix[3] = at.m11
	if len(flatmatrix) >= 6 {
		flatmatrix[4] = at.m02
		flatmatrix[5] = at.m12
	}
}

// Determinant returns the determinant, which is zero exactly when the transform
// cannot be inverted.
func (at *AffineTransform) Determinant() float64 {
	return at.m00*at.m11 - at.m01*at.m10
}

// IsIdentity reports whether the transform leaves every point where it is.
func (at *AffineTransform) IsIdentity() bool {
	return at.m00 == 1 && at.m10 == 0 && at.m01 == 0 &&
		at.m11 == 1 && at.m02 == 0 && at.m12 == 0
}

// Clone returns an independent copy.
func (at *AffineTransform) Clone() *AffineTransform {
	clone := *at
	return &clone
}

// Equals reports whether other holds the same six values.
func (at *AffineTransform) Equals(other *AffineTransform) bool {
	if other == nil {
		return false
	}
	return at.m00 == other.m00 && at.m01 == other.m01 && at.m02 == other.m02 &&
		at.m10 == other.m10 && at.m11 == other.m11 && at.m12 == other.m12
}

// Translate concatenates a translation, so that a point is moved by (tx, ty)
// before this transform is applied to it. The offset is therefore expressed in
// the coordinates this transform starts from, not in the ones it produces.
func (at *AffineTransform) Translate(tx, ty float64) {
	at.m02 += tx*at.m00 + ty*at.m01
	at.m12 += tx*at.m10 + ty*at.m11
}

// Scale concatenates a scaling by sx and sy.
func (at *AffineTransform) Scale(sx, sy float64) {
	at.m00 *= sx
	at.m10 *= sx
	at.m01 *= sy
	at.m11 *= sy
}

// Shear concatenates a shear by shx and shy.
func (at *AffineTransform) Shear(shx, shy float64) {
	m00, m01 := at.m00, at.m01
	at.m00 = m00 + m01*shy
	at.m01 = m00*shx + m01

	m10, m11 := at.m10, at.m11
	at.m10 = m10 + m11*shy
	at.m11 = m10*shx + m11
}

// Rotate concatenates a rotation of theta radians about the origin. See
// SetToRotation for why the quadrants are handled apart.
func (at *AffineTransform) Rotate(theta float64) {
	sin := math.Sin(theta)
	switch {
	case sin == 1.0:
		at.rotate90()
	case sin == -1.0:
		at.rotate270()
	default:
		cos := math.Cos(theta)
		if cos == -1.0 {
			at.rotate180()
		} else if cos != 1.0 {
			// cos == 1.0 means a rotation of nothing, which is left alone.
			m00, m01 := at.m00, at.m01
			at.m00 = cos*m00 + sin*m01
			at.m01 = -sin*m00 + cos*m01

			m10, m11 := at.m10, at.m11
			at.m10 = cos*m10 + sin*m11
			at.m11 = -sin*m10 + cos*m11
		}
	}
}

func (at *AffineTransform) rotate90() {
	m00 := at.m00
	at.m00 = at.m01
	at.m01 = -m00
	m10 := at.m10
	at.m10 = at.m11
	at.m11 = -m10
}

func (at *AffineTransform) rotate180() {
	// The JDK negates m01 and m10 only when the matrix is known to have a shear
	// term. Where it does not they are already zero, so negating them always
	// gives the same matrix, give or take the sign of a zero.
	at.m00 = -at.m00
	at.m01 = -at.m01
	at.m10 = -at.m10
	at.m11 = -at.m11
}

func (at *AffineTransform) rotate270() {
	m00 := at.m00
	at.m00 = -at.m01
	at.m01 = m00
	m10 := at.m10
	at.m10 = -at.m11
	at.m11 = m10
}

// Concatenate appends tx to this transform, so that tx is applied to a point
// first and this transform second.
func (at *AffineTransform) Concatenate(tx *AffineTransform) {
	m00, m01 := at.m00, at.m01
	m10, m11 := at.m10, at.m11

	at.m00 = m00*tx.m00 + m01*tx.m10
	at.m01 = m00*tx.m01 + m01*tx.m11
	at.m02 += m00*tx.m02 + m01*tx.m12

	at.m10 = m10*tx.m00 + m11*tx.m10
	at.m11 = m10*tx.m01 + m11*tx.m11
	at.m12 += m10*tx.m02 + m11*tx.m12
}

// PreConcatenate prepends tx to this transform, so that this transform is
// applied to a point first and tx second.
func (at *AffineTransform) PreConcatenate(tx *AffineTransform) {
	m00, m01, m02 := at.m00, at.m01, at.m02
	m10, m11, m12 := at.m10, at.m11, at.m12

	at.m00 = tx.m00*m00 + tx.m01*m10
	at.m01 = tx.m00*m01 + tx.m01*m11
	at.m02 = tx.m00*m02 + tx.m01*m12 + tx.m02

	at.m10 = tx.m10*m00 + tx.m11*m10
	at.m11 = tx.m10*m01 + tx.m11*m11
	at.m12 = tx.m10*m02 + tx.m11*m12 + tx.m12
}

// CreateInverse returns the transform that undoes this one.
func (at *AffineTransform) CreateInverse() (*AffineTransform, error) {
	det := at.Determinant()
	if math.Abs(det) <= math.SmallestNonzeroFloat64 {
		return nil, ErrNoninvertibleTransform
	}
	return NewAffineTransform(
		at.m11/det,
		-at.m10/det,
		-at.m01/det,
		at.m00/det,
		(at.m01*at.m12-at.m11*at.m02)/det,
		(at.m10*at.m02-at.m00*at.m12)/det,
	), nil
}

// Transform maps ptSrc through this transform, writing the result into ptDst
// and returning it. ptSrc and ptDst may be the same point. A nil ptDst gets a
// new PointDouble.
func (at *AffineTransform) Transform(ptSrc, ptDst Point2D) Point2D {
	if ptDst == nil {
		ptDst = &PointDouble{}
	}
	x, y := ptSrc.X(), ptSrc.Y()
	ptDst.SetLocation(x*at.m00+y*at.m01+at.m02, x*at.m10+y*at.m11+at.m12)
	return ptDst
}

// DeltaTransform maps ptSrc through this transform's linear part, leaving the
// translation out. It is the right transform for a distance or a direction,
// which have a length and a heading but no position.
func (at *AffineTransform) DeltaTransform(ptSrc, ptDst Point2D) Point2D {
	if ptDst == nil {
		ptDst = &PointDouble{}
	}
	x, y := ptSrc.X(), ptSrc.Y()
	ptDst.SetLocation(x*at.m00+y*at.m01, x*at.m10+y*at.m11)
	return ptDst
}

// InverseTransform maps ptSrc back through this transform.
func (at *AffineTransform) InverseTransform(ptSrc, ptDst Point2D) error {
	det := at.Determinant()
	if math.Abs(det) <= math.SmallestNonzeroFloat64 {
		return ErrNoninvertibleTransform
	}
	x := ptSrc.X() - at.m02
	y := ptSrc.Y() - at.m12
	ptDst.SetLocation((x*at.m11-y*at.m01)/det, (y*at.m00-x*at.m10)/det)
	return nil
}

// TransformFloats maps numPts points, taken as x, y pairs from src starting at
// srcOff, into dst starting at dstOff. src and dst may be the same slice.
func (at *AffineTransform) TransformFloats(src []float32, srcOff int, dst []float32, dstOff int, numPts int) {
	if &src[0] == &dst[0] && dstOff > srcOff && dstOff < srcOff+numPts*2 {
		// The ranges overlap in a way that would let the writes run over
		// coordinates not yet read, so work from a copy.
		src = append([]float32(nil), src[srcOff:srcOff+numPts*2]...)
		srcOff = 0
	}
	for ; numPts > 0; numPts-- {
		x := float64(src[srcOff])
		srcOff++
		y := float64(src[srcOff])
		srcOff++
		dst[dstOff] = float32(x*at.m00 + y*at.m01 + at.m02)
		dstOff++
		dst[dstOff] = float32(x*at.m10 + y*at.m11 + at.m12)
		dstOff++
	}
}

// TransformDoubles maps numPts points, taken as x, y pairs from src starting at
// srcOff, into dst starting at dstOff. src and dst may be the same slice.
func (at *AffineTransform) TransformDoubles(src []float64, srcOff int, dst []float64, dstOff int, numPts int) {
	if &src[0] == &dst[0] && dstOff > srcOff && dstOff < srcOff+numPts*2 {
		src = append([]float64(nil), src[srcOff:srcOff+numPts*2]...)
		srcOff = 0
	}
	for ; numPts > 0; numPts-- {
		x := src[srcOff]
		srcOff++
		y := src[srcOff]
		srcOff++
		dst[dstOff] = x*at.m00 + y*at.m01 + at.m02
		dstOff++
		dst[dstOff] = x*at.m10 + y*at.m11 + at.m12
		dstOff++
	}
}
