// Package common holds the wrappers shared across the PDF document model.
//
// Port of org.apache.pdfbox.pdmodel.common. The PD prefix of the Java class
// names is kept: it marks the document model across the whole pdmodel tree
// rather than repeating the name of this package, and dropping it would leave
// names like common.Rectangle that read as something else.
package common

import (
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/internal/javafmt"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// COSObjectable is implemented by anything that can produce the COS object
// behind it.
//
// Port of org.apache.pdfbox.pdmodel.common.COSObjectable. Every cos.Base
// satisfies it, as every COSBase does in Java.
type COSObjectable interface {
	// COSObject returns the COS object that matches this object.
	COSObject() cos.Base
}

const (
	// pointsPerInch is the number of user space units per inch.
	pointsPerInch = 72

	// pointsPerMM is the number of user space units per millimeter.
	pointsPerMM = 1 / (10 * 2.54) * pointsPerInch
)

// The predefined page sizes. Each is immutable: its setters panic.
var (
	// Letter is a rectangle the size of U.S. Letter, 8.5" x 11".
	Letter = NewPDImmutableRectangle(8.5*pointsPerInch, 11*pointsPerInch)

	// Tabloid is a rectangle the size of U.S. Tabloid, 11" x 17".
	Tabloid = NewPDImmutableRectangle(11*pointsPerInch, 17*pointsPerInch)

	// Legal is a rectangle the size of U.S. Legal, 8.5" x 14".
	Legal = NewPDImmutableRectangle(8.5*pointsPerInch, 14*pointsPerInch)

	// A0 is a rectangle the size of A0 Paper.
	A0 = NewPDImmutableRectangle(841*pointsPerMM, 1189*pointsPerMM)

	// A1 is a rectangle the size of A1 Paper.
	A1 = NewPDImmutableRectangle(594*pointsPerMM, 841*pointsPerMM)

	// A2 is a rectangle the size of A2 Paper.
	A2 = NewPDImmutableRectangle(420*pointsPerMM, 594*pointsPerMM)

	// A3 is a rectangle the size of A3 Paper.
	A3 = NewPDImmutableRectangle(297*pointsPerMM, 420*pointsPerMM)

	// A4 is a rectangle the size of A4 Paper.
	A4 = NewPDImmutableRectangle(210*pointsPerMM, 297*pointsPerMM)

	// A5 is a rectangle the size of A5 Paper.
	A5 = NewPDImmutableRectangle(148*pointsPerMM, 210*pointsPerMM)

	// A6 is a rectangle the size of A6 Paper.
	A6 = NewPDImmutableRectangle(105*pointsPerMM, 148*pointsPerMM)
)

// PDRectangle is a rectangle in a PDF document.
//
// Port of org.apache.pdfbox.pdmodel.common.PDRectangle, and of the subclass
// PDImmutableRectangle it is given for the predefined page sizes. Java gets
// immutability by overriding the four setters to throw; Go has no subclassing,
// so the flag is carried here and the setters check it.
type PDRectangle struct {
	rectArray *cos.Array
	immutable bool
}

var _ COSObjectable = (*PDRectangle)(nil)

// NewPDRectangle returns the rectangle 0,0,0,0.
func NewPDRectangle() *PDRectangle {
	return NewPDRectangleOf(0, 0, 0, 0)
}

// NewPDRectangleOfSize returns the rectangle of the given width and height, at
// the origin.
func NewPDRectangleOfSize(width, height float32) *PDRectangle {
	return NewPDRectangleOf(0, 0, width, height)
}

// NewPDRectangleOf returns the rectangle of the given width and height with its
// lower left corner at (x, y).
func NewPDRectangleOf(x, y, width, height float32) *PDRectangle {
	return &PDRectangle{rectArray: cos.NewArrayOf([]cos.Base{
		cos.NewFloat(x),
		cos.NewFloat(y),
		cos.NewFloat(x + width),
		cos.NewFloat(y + height),
	})}
}

// NewPDImmutableRectangle returns a rectangle of the given width and height
// whose setters refuse to change it.
//
// Port of the PDImmutableRectangle constructor. Java throws
// UnsupportedOperationException from the setters; the port panics, since
// nothing catches it and it marks a caller's mistake rather than bad input.
func NewPDImmutableRectangle(width, height float32) *PDRectangle {
	r := NewPDRectangleOfSize(width, height)
	r.immutable = true
	return r
}

// NewPDRectangleOfBoundingBox returns the rectangle covering the given bounding
// box.
func NewPDRectangleOfBoundingBox(box *fontutil.BoundingBox) *PDRectangle {
	return &PDRectangle{rectArray: cos.NewArrayOf([]cos.Base{
		cos.NewFloat(box.LowerLeftX()),
		cos.NewFloat(box.LowerLeftY()),
		cos.NewFloat(box.UpperRightX()),
		cos.NewFloat(box.UpperRightY()),
	})}
}

// maxIntAsFloat is Integer.MAX_VALUE as Java sees it when it compares a float
// against it, which rounds the int up to the next float.
const maxIntAsFloat = float32(math.MaxInt32)

// NewPDRectangleOfCOSArray returns the rectangle given by an array of numbers,
// as specified in the PDF Reference for a rectangle type.
func NewPDRectangleOfCOSArray(array *cos.Array) *PDRectangle {
	values := make([]float32, 4)
	copy(values, array.ToFloatArray())

	// replace huge values, most likely those are invalid due to a malformed pdf
	for i := range values {
		if abs32(values[i]) > maxIntAsFloat {
			if values[i] > 0 {
				values[i] = maxIntAsFloat
			} else {
				values[i] = -maxIntAsFloat
			}
		}
	}
	return &PDRectangle{rectArray: cos.NewArrayOf([]cos.Base{
		// we have to start with the lower left corner
		cos.NewFloat(min(values[0], values[2])),
		cos.NewFloat(min(values[1], values[3])),
		cos.NewFloat(max(values[0], values[2])),
		cos.NewFloat(max(values[1], values[3])),
	})}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// IsImmutable reports whether this rectangle refuses to be changed, which the
// predefined page sizes do.
func (r *PDRectangle) IsImmutable() bool { return r.immutable }

// checkMutable rejects a change to one of the predefined sizes.
func (r *PDRectangle) checkMutable() {
	if r.immutable {
		panic("common: immutable class")
	}
}

// Contains reports whether the x/y point is inside this rectangle. Unlike
// geom.Rectangle2D, the edges count as inside.
func (r *PDRectangle) Contains(x, y float32) bool {
	llx := r.LowerLeftX()
	urx := r.UpperRightX()
	lly := r.LowerLeftY()
	ury := r.UpperRightY()
	return x >= llx && x <= urx &&
		y >= lly && y <= ury
}

// CreateRetranslatedRectangle returns a rectangle with the same dimensions as
// this one but with its lower left corner at the origin, so that
//
//	100, 100, 400, 400 (llx, lly, urx, ury)
//
// becomes 0, 0, 300, 300.
func (r *PDRectangle) CreateRetranslatedRectangle() *PDRectangle {
	retval := NewPDRectangle()
	retval.SetUpperRightX(r.Width())
	retval.SetUpperRightY(r.Height())
	return retval
}

// COSArray returns the underlying array for this rectangle.
func (r *PDRectangle) COSArray() *cos.Array { return r.rectArray }

// LowerLeftX returns the lower left x coordinate.
func (r *PDRectangle) LowerLeftX() float32 {
	return r.rectArray.Get(0).(cos.Number).FloatValue()
}

// SetLowerLeftX sets the lower left x coordinate.
func (r *PDRectangle) SetLowerLeftX(value float32) {
	r.checkMutable()
	r.rectArray.Set(0, cos.NewFloat(value))
}

// LowerLeftY returns the lower left y coordinate.
func (r *PDRectangle) LowerLeftY() float32 {
	return r.rectArray.Get(1).(cos.Number).FloatValue()
}

// SetLowerLeftY sets the lower left y coordinate.
func (r *PDRectangle) SetLowerLeftY(value float32) {
	r.checkMutable()
	r.rectArray.Set(1, cos.NewFloat(value))
}

// UpperRightX returns the upper right x coordinate.
func (r *PDRectangle) UpperRightX() float32 {
	return r.rectArray.Get(2).(cos.Number).FloatValue()
}

// SetUpperRightX sets the upper right x coordinate.
func (r *PDRectangle) SetUpperRightX(value float32) {
	r.checkMutable()
	r.rectArray.Set(2, cos.NewFloat(value))
}

// UpperRightY returns the upper right y coordinate.
func (r *PDRectangle) UpperRightY() float32 {
	return r.rectArray.Get(3).(cos.Number).FloatValue()
}

// SetUpperRightY sets the upper right y coordinate.
func (r *PDRectangle) SetUpperRightY(value float32) {
	r.checkMutable()
	r.rectArray.Set(3, cos.NewFloat(value))
}

// Width returns the width of this rectangle as calculated by
// upperRightX - lowerLeftX.
func (r *PDRectangle) Width() float32 { return r.UpperRightX() - r.LowerLeftX() }

// Height returns the height of this rectangle as calculated by
// upperRightY - lowerLeftY.
func (r *PDRectangle) Height() float32 { return r.UpperRightY() - r.LowerLeftY() }

// Transform returns a path which represents this rectangle having been
// transformed by the given matrix. Note that the resulting path need not be
// rectangular.
func (r *PDRectangle) Transform(matrix *util.Matrix) *geom.Path2D {
	x1 := r.LowerLeftX()
	y1 := r.LowerLeftY()
	x2 := r.UpperRightX()
	y2 := r.UpperRightY()

	p0 := matrix.TransformPoint(x1, y1)
	p1 := matrix.TransformPoint(x2, y1)
	p2 := matrix.TransformPoint(x2, y2)
	p3 := matrix.TransformPoint(x1, y2)

	path := geom.NewPathFloat()
	path.MoveTo(p0.X(), p0.Y())
	path.LineTo(p1.X(), p1.Y())
	path.LineTo(p2.X(), p2.Y())
	path.LineTo(p3.X(), p3.Y())
	path.ClosePath()
	return path
}

// COSObject returns the COS object that matches this rectangle, which is the
// very array behind it rather than a copy.
func (r *PDRectangle) COSObject() cos.Base { return r.rectArray }

// ToGeneralPath returns a path equivalent to this rectangle. It avoids the
// problems caused by geom.Rectangle2D not working well with negative
// rectangles.
func (r *PDRectangle) ToGeneralPath() *geom.Path2D {
	x1 := float64(r.LowerLeftX())
	y1 := float64(r.LowerLeftY())
	x2 := float64(r.UpperRightX())
	y2 := float64(r.UpperRightY())
	path := geom.NewPathFloat()
	path.MoveTo(x1, y1)
	path.LineTo(x2, y1)
	path.LineTo(x2, y2)
	path.LineTo(x1, y2)
	path.ClosePath()
	return path
}

// String returns the Java toString form.
func (r *PDRectangle) String() string {
	return "[" + javafmt.Float32(r.LowerLeftX()) + "," + javafmt.Float32(r.LowerLeftY()) + "," +
		javafmt.Float32(r.UpperRightX()) + "," + javafmt.Float32(r.UpperRightY()) + "]"
}
