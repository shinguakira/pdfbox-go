package geom

import "math"

// Rectangle2D is an axis-aligned rectangle given by a corner and a size.
//
// Port of java.awt.geom.Rectangle2D. Java splits it into an abstract class with
// Float and Double subclasses; PDFBox only ever uses the Double one, so the
// port keeps a single type.
type Rectangle2D struct {
	X, Y, Width, Height float64
}

var _ Shape = (*Rectangle2D)(nil)

// NewRectangle2D returns the rectangle with the given corner and size.
func NewRectangle2D(x, y, w, h float64) *Rectangle2D {
	return &Rectangle2D{X: x, Y: y, Width: w, Height: h}
}

// SetRect replaces the corner and size.
func (r *Rectangle2D) SetRect(x, y, w, h float64) {
	r.X, r.Y, r.Width, r.Height = x, y, w, h
}

// MinX returns the smallest x coordinate.
func (r *Rectangle2D) MinX() float64 { return r.X }

// MinY returns the smallest y coordinate.
func (r *Rectangle2D) MinY() float64 { return r.Y }

// MaxX returns the largest x coordinate.
func (r *Rectangle2D) MaxX() float64 { return r.X + r.Width }

// MaxY returns the largest y coordinate.
func (r *Rectangle2D) MaxY() float64 { return r.Y + r.Height }

// CenterX returns the x coordinate of the centre.
func (r *Rectangle2D) CenterX() float64 { return r.X + r.Width/2 }

// CenterY returns the y coordinate of the centre.
func (r *Rectangle2D) CenterY() float64 { return r.Y + r.Height/2 }

// IsEmpty reports whether the rectangle encloses nothing, which a negative or
// zero extent in either direction makes it do.
func (r *Rectangle2D) IsEmpty() bool { return r.Width <= 0 || r.Height <= 0 }

// Contains reports whether the point lies inside the rectangle. An empty
// rectangle contains nothing, which falls out of the comparisons rather than
// needing a check of its own.
func (r *Rectangle2D) Contains(x, y float64) bool {
	return x >= r.X && y >= r.Y && x < r.X+r.Width && y < r.Y+r.Height
}

// Intersects reports whether the two rectangles overlap.
func (r *Rectangle2D) Intersects(other *Rectangle2D) bool {
	if r.IsEmpty() || other.IsEmpty() {
		return false
	}
	return other.MaxX() > r.X && other.MaxY() > r.Y &&
		other.X < r.MaxX() && other.Y < r.MaxY()
}

// Intersect writes the overlap of src1 and src2 into dest, which may be either
// source. Rectangles that do not overlap give a rectangle with a negative
// extent, which IsEmpty reports on.
func Intersect(src1, src2, dest *Rectangle2D) {
	x1 := math.Max(src1.MinX(), src2.MinX())
	y1 := math.Max(src1.MinY(), src2.MinY())
	x2 := math.Min(src1.MaxX(), src2.MaxX())
	y2 := math.Min(src1.MaxY(), src2.MaxY())
	dest.SetRect(x1, y1, x2-x1, y2-y1)
}

// CreateIntersection returns the overlap of the two rectangles in a new one.
func (r *Rectangle2D) CreateIntersection(other *Rectangle2D) *Rectangle2D {
	dest := &Rectangle2D{}
	Intersect(r, other, dest)
	return dest
}

// Union writes the smallest rectangle enclosing both sources into dest, which
// may be either source.
func Union(src1, src2, dest *Rectangle2D) {
	x1 := math.Min(src1.MinX(), src2.MinX())
	y1 := math.Min(src1.MinY(), src2.MinY())
	x2 := math.Max(src1.MaxX(), src2.MaxX())
	y2 := math.Max(src1.MaxY(), src2.MaxY())
	dest.SetRect(x1, y1, x2-x1, y2-y1)
}

// CreateUnion returns the smallest rectangle enclosing both, in a new one.
func (r *Rectangle2D) CreateUnion(other *Rectangle2D) *Rectangle2D {
	dest := &Rectangle2D{}
	Union(r, other, dest)
	return dest
}

// Add grows the rectangle just enough to enclose the given point.
func (r *Rectangle2D) Add(newx, newy float64) {
	x1 := math.Min(r.MinX(), newx)
	x2 := math.Max(r.MaxX(), newx)
	y1 := math.Min(r.MinY(), newy)
	y2 := math.Max(r.MaxY(), newy)
	r.SetRect(x1, y1, x2-x1, y2-y1)
}

// Bounds2D returns a copy of the rectangle.
func (r *Rectangle2D) Bounds2D() *Rectangle2D {
	clone := *r
	return &clone
}

// Bounds returns the smallest integer rectangle that completely encloses this
// one. A rectangle with a negative extent gives the rectangle at the origin
// with no size, losing its position — which is what Java does, and is why a
// zero extent is treated as a real rectangle here rather than as empty.
func (r *Rectangle2D) Bounds() *Rectangle {
	if r.Width < 0 || r.Height < 0 {
		return &Rectangle{}
	}
	x1 := math.Floor(r.X)
	y1 := math.Floor(r.Y)
	x2 := math.Ceil(r.X + r.Width)
	y2 := math.Ceil(r.Y + r.Height)
	return &Rectangle{X: int(x1), Y: int(y1), Width: int(x2 - x1), Height: int(y2 - y1)}
}

// PathIterator walks the four corners of the rectangle.
func (r *Rectangle2D) PathIterator(at *AffineTransform) PathIterator {
	path := NewPathDouble()
	if !r.IsEmpty() {
		path.MoveTo(r.MinX(), r.MinY())
		path.LineTo(r.MaxX(), r.MinY())
		path.LineTo(r.MaxX(), r.MaxY())
		path.LineTo(r.MinX(), r.MaxY())
		path.ClosePath()
	}
	return path.PathIterator(at)
}

// Rectangle is an axis-aligned rectangle at integer coordinates.
//
// Port of the part of java.awt.Rectangle that PDFBox uses.
type Rectangle struct {
	X, Y, Width, Height int
}
