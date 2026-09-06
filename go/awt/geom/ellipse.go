package geom

// Ellipse2D is an ellipse defined by the rectangle that frames it.
//
// Port of java.awt.geom.Ellipse2D.Double, of which only the path iterator is
// used: PDFBox flattens an ellipse into a polygon to draw a cloudy border.
type Ellipse2D struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

var _ Shape = (*Ellipse2D)(nil)

// NewEllipse2D returns the ellipse framed by the given rectangle.
func NewEllipse2D(x, y, w, h float64) *Ellipse2D {
	return &Ellipse2D{X: x, Y: y, Width: w, Height: h}
}

// IsEmpty reports whether the ellipse encloses nothing.
func (e *Ellipse2D) IsEmpty() bool { return e.Width <= 0 || e.Height <= 0 }

// Bounds2D returns the rectangle that frames the ellipse.
func (e *Ellipse2D) Bounds2D() *Rectangle2D {
	return NewRectangle2D(e.X, e.Y, e.Width, e.Height)
}

// ctrlVal is the distance, as a fraction of the radius, from an axis endpoint
// to the control point that turns a cubic into a quarter of a circle.
//
// Port of the private EllipseIterator.CtrlVal.
const ctrlVal = 0.5522847498307933

// pcv and ncv are the control point coordinates as fractions of the frame.
const (
	pcv = 0.5 + ctrlVal*0.5
	ncv = 0.5 - ctrlVal*0.5
)

// ellipseCtrlPts holds the two control points and the end point of each of the
// four quarters, as fractions of the frame.
//
// Port of the private EllipseIterator.ctrlpts.
var ellipseCtrlPts = [4][6]float64{
	{1.0, pcv, pcv, 1.0, 0.5, 1.0},
	{ncv, 1.0, 0.0, pcv, 0.0, 0.5},
	{0.0, ncv, ncv, 0.0, 0.5, 0.0},
	{pcv, 0.0, 1.0, ncv, 1.0, 0.5},
}

// PathIterator returns a cursor over the four cubic curves of the ellipse.
func (e *Ellipse2D) PathIterator(at *AffineTransform) PathIterator {
	return &ellipseIterator{ellipse: e, affine: at}
}

// FlatteningPathIterator returns a cursor over the ellipse, with the curves
// broken into lines no further than flatness from them.
//
// Port of Shape.getPathIterator(AffineTransform, double), which wraps the path
// iterator in a FlatteningPathIterator with the default recursion limit.
func (e *Ellipse2D) FlatteningPathIterator(at *AffineTransform, flatness float64) PathIterator {
	return NewFlatteningPathIterator(e.PathIterator(at), flatness)
}

// ellipseIterator walks the four cubic curves of an ellipse.
//
// Port of the private class EllipseIterator.
type ellipseIterator struct {
	ellipse *Ellipse2D
	affine  *AffineTransform
	index   int
}

// WindingRule returns the non-zero rule, which is what an ellipse uses.
func (it *ellipseIterator) WindingRule() int { return WindNonZero }

// IsDone reports whether the walk is finished.
func (it *ellipseIterator) IsDone() bool { return it.index > 5 }

// Next advances to the next segment.
func (it *ellipseIterator) Next() { it.index++ }

// CurrentSegment writes the points of the current segment into coords and
// returns the segment type.
func (it *ellipseIterator) CurrentSegment(coords []float64) int {
	if it.IsDone() {
		panic("geom: ellipse iterator out of bounds")
	}
	if it.index == 5 {
		return SegClose
	}
	if it.index == 0 {
		ctrls := ellipseCtrlPts[3]
		coords[0] = it.ellipse.X + ctrls[4]*it.ellipse.Width
		coords[1] = it.ellipse.Y + ctrls[5]*it.ellipse.Height
		if it.affine != nil {
			it.affine.TransformDoubles(coords, 0, coords, 0, 1)
		}
		return SegMoveTo
	}
	ctrls := ellipseCtrlPts[it.index-1]
	coords[0] = it.ellipse.X + ctrls[0]*it.ellipse.Width
	coords[1] = it.ellipse.Y + ctrls[1]*it.ellipse.Height
	coords[2] = it.ellipse.X + ctrls[2]*it.ellipse.Width
	coords[3] = it.ellipse.Y + ctrls[3]*it.ellipse.Height
	coords[4] = it.ellipse.X + ctrls[4]*it.ellipse.Width
	coords[5] = it.ellipse.Y + ctrls[5]*it.ellipse.Height
	if it.affine != nil {
		it.affine.TransformDoubles(coords, 0, coords, 0, 3)
	}
	return SegCubicTo
}
