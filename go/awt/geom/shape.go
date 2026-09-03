package geom

// Winding rules, deciding which side of a path counts as inside.
//
// Port of the WIND_ constants of java.awt.geom.PathIterator.
const (
	// WindEvenOdd counts a point as inside when a ray from it crosses the path
	// an odd number of times.
	WindEvenOdd = 0

	// WindNonZero counts a point as inside when the crossings of a ray from it,
	// counted with direction, do not cancel out.
	WindNonZero = 1
)

// Segment types returned by PathIterator.CurrentSegment.
//
// Port of the SEG_ constants of java.awt.geom.PathIterator.
const (
	// SegMoveTo begins a new subpath at one point.
	SegMoveTo = 0

	// SegLineTo draws a line to one point.
	SegLineTo = 1

	// SegQuadTo draws a quadratic curve through one control point to one point.
	SegQuadTo = 2

	// SegCubicTo draws a cubic curve through two control points to one point.
	SegCubicTo = 3

	// SegClose closes the current subpath back to its starting point and takes
	// no points.
	SegClose = 4
)

// segmentPointCount is how many points each segment type carries, so how many
// pairs of coordinates it occupies.
var segmentPointCount = [...]int{
	SegMoveTo:  1,
	SegLineTo:  1,
	SegQuadTo:  2,
	SegCubicTo: 3,
	SegClose:   0,
}

// Shape is a geometric figure.
//
// Port of the part of java.awt.Shape that PDFBox uses. The hit-testing methods
// — contains and intersects — are not here: nothing in PDFBox calls them on a
// path, and answering them properly needs the curve-crossing machinery the JDK
// keeps in sun.awt.geom.
type Shape interface {
	// Bounds2D returns a rectangle that completely encloses the shape.
	Bounds2D() *Rectangle2D

	// PathIterator returns a cursor over the segments of the shape, with each
	// point mapped through at when at is not nil.
	PathIterator(at *AffineTransform) PathIterator
}

// PathIterator walks the segments of a shape.
//
// Port of java.awt.geom.PathIterator. It is a cursor rather than a Go iterator
// because the ported code drives it directly, one segment at a time.
type PathIterator interface {
	// WindingRule returns the winding rule for determining the interior of the
	// path, WindEvenOdd or WindNonZero.
	WindingRule() int

	// IsDone reports whether the walk is finished.
	IsDone() bool

	// Next advances to the next segment.
	Next()

	// CurrentSegment writes the points of the current segment into coords,
	// which must have room for six values, and returns the segment type.
	CurrentSegment(coords []float64) int
}
