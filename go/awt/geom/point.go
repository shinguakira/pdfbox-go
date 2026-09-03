// Package geom holds the parts of java.awt.geom that PDFBox depends on.
//
// This is not PDFBox code: it is the JDK geometry the Java source uses for
// points, transforms and paths, ported far enough to carry the PDFBox code
// above it. Go has no equivalent in its standard library, and PLAN.md's slice 9
// settles on porting the geometry while leaving rasterisation behind an
// interface — the same split PdfBox-Android made when it vendored Harmony's
// AffineTransform and left the drawing to the platform.
//
// Only what PDFBox calls is here. The JDK classes are much larger.
package geom

import (
	"math"
	"strconv"
	"strings"
)

// Point2D is a point in 2D space.
//
// Port of the abstract java.awt.geom.Point2D. Its two concrete forms differ
// only in the precision they store, so code that just reads coordinates takes
// this interface and does not care which it has.
type Point2D interface {
	// X returns the x coordinate.
	X() float64

	// Y returns the y coordinate.
	Y() float64

	// SetLocation moves the point.
	SetLocation(x, y float64)
}

// PointFloat stores its coordinates as float32.
//
// Port of java.awt.geom.Point2D.Float.
type PointFloat struct {
	x, y float32
}

var _ Point2D = (*PointFloat)(nil)

// NewPointFloat returns the point (x, y).
func NewPointFloat(x, y float32) *PointFloat { return &PointFloat{x: x, y: y} }

// X returns the x coordinate widened to a float64.
func (p *PointFloat) X() float64 { return float64(p.x) }

// Y returns the y coordinate widened to a float64.
func (p *PointFloat) Y() float64 { return float64(p.y) }

// XFloat returns the x coordinate at its stored precision.
func (p *PointFloat) XFloat() float32 { return p.x }

// YFloat returns the y coordinate at its stored precision.
func (p *PointFloat) YFloat() float32 { return p.y }

// SetLocation moves the point, narrowing both coordinates to float32.
func (p *PointFloat) SetLocation(x, y float64) {
	p.x = float32(x)
	p.y = float32(y)
}

// SetLocationFloat moves the point without a narrowing conversion.
func (p *PointFloat) SetLocationFloat(x, y float32) {
	p.x = x
	p.y = y
}

// Distance returns the distance from this point to other.
func (p *PointFloat) Distance(other Point2D) float64 {
	return Distance(p.X(), p.Y(), other.X(), other.Y())
}

// String returns the Java toString form.
func (p *PointFloat) String() string {
	return "Point2D.Float[" + javaFloat32String(p.x) + ", " + javaFloat32String(p.y) + "]"
}

// PointDouble stores its coordinates as float64.
//
// Port of java.awt.geom.Point2D.Double.
type PointDouble struct {
	x, y float64
}

var _ Point2D = (*PointDouble)(nil)

// NewPointDouble returns the point (x, y).
func NewPointDouble(x, y float64) *PointDouble { return &PointDouble{x: x, y: y} }

// X returns the x coordinate.
func (p *PointDouble) X() float64 { return p.x }

// Y returns the y coordinate.
func (p *PointDouble) Y() float64 { return p.y }

// SetLocation moves the point.
func (p *PointDouble) SetLocation(x, y float64) {
	p.x = x
	p.y = y
}

// Distance returns the distance from this point to other.
func (p *PointDouble) Distance(other Point2D) float64 {
	return Distance(p.X(), p.Y(), other.X(), other.Y())
}

// String returns the Java toString form.
func (p *PointDouble) String() string {
	return "Point2D.Double[" + javaFloat64String(p.x) + ", " + javaFloat64String(p.y) + "]"
}

// DistanceSq returns the square of the distance between two points, which
// avoids the square root when only an ordering is needed.
func DistanceSq(x1, y1, x2, y2 float64) float64 {
	x1 -= x2
	y1 -= y2
	return x1*x1 + y1*y1
}

// Distance returns the distance between two points.
func Distance(x1, y1, x2, y2 float64) float64 {
	x1 -= x2
	y1 -= y2
	return math.Sqrt(x1*x1 + y1*y1)
}

// javaFloat32String renders a float the way Java's String.valueOf(float) does,
// which always shows a fraction part.
func javaFloat32String(value float32) string {
	return withFractionPart(strconv.FormatFloat(float64(value), 'g', -1, 32))
}

// javaFloat64String renders a double the way Java's String.valueOf(double)
// does.
func javaFloat64String(value float64) string {
	return withFractionPart(strconv.FormatFloat(value, 'g', -1, 64))
}

func withFractionPart(s string) string {
	if strings.ContainsAny(s, ".eEnI") {
		return s
	}
	return s + ".0"
}
