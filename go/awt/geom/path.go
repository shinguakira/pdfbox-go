package geom

import "math"

// Path2D is a path built from straight lines and quadratic and cubic curves.
//
// Port of java.awt.geom.Path2D. Java splits it into Path2D.Float — which is
// what GeneralPath is — and Path2D.Double, differing only in the precision the
// coordinates are stored at. The port keeps one type holding float64 and
// rounds through float32 on the way in when the path is a float one, so that
// reading a coordinate back gives what Java would give.
//
// The hit-testing methods are left out; see the note on Shape.
type Path2D struct {
	types  []byte
	coords []float64

	windingRule int

	// singlePrecision rounds each coordinate as it is stored, which is what
	// makes this a Path2D.Float rather than a Path2D.Double.
	singlePrecision bool
}

var _ Shape = (*Path2D)(nil)

// NewPathFloat returns an empty path storing single-precision coordinates,
// under the non-zero winding rule.
//
// Port of new Path2D.Float(), which is what new GeneralPath() builds.
func NewPathFloat() *Path2D {
	return &Path2D{windingRule: WindNonZero, singlePrecision: true}
}

// NewPathFloatRule returns an empty single-precision path under the given
// winding rule.
func NewPathFloatRule(rule int) *Path2D {
	return &Path2D{windingRule: rule, singlePrecision: true}
}

// NewPathFloatShape returns a single-precision copy of the given shape.
func NewPathFloatShape(s Shape) *Path2D {
	return copyShape(NewPathFloat(), s)
}

// copyShape fills an empty path from a shape, taking the shape's winding rule
// with it.
func copyShape(path *Path2D, s Shape) *Path2D {
	pi := s.PathIterator(nil)
	path.windingRule = pi.WindingRule()
	path.AppendIterator(pi, false)
	return path
}

// NewPathDouble returns an empty path storing double-precision coordinates,
// under the non-zero winding rule.
func NewPathDouble() *Path2D {
	return &Path2D{windingRule: WindNonZero}
}

// NewPathDoubleRule returns an empty double-precision path under the given
// winding rule.
func NewPathDoubleRule(rule int) *Path2D {
	return &Path2D{windingRule: rule}
}

// NewPathDoubleShape returns a double-precision copy of the given shape.
func NewPathDoubleShape(s Shape) *Path2D {
	return copyShape(NewPathDouble(), s)
}

// store rounds a coordinate to the precision this path keeps.
func (p *Path2D) store(v float64) float64 {
	if p.singlePrecision {
		return float64(float32(v))
	}
	return v
}

func (p *Path2D) appendSegment(segment byte, values ...float64) {
	p.types = append(p.types, segment)
	for _, v := range values {
		p.coords = append(p.coords, p.store(v))
	}
}

// MoveTo begins a new subpath at the given point.
func (p *Path2D) MoveTo(x, y float64) {
	if len(p.types) > 0 && p.types[len(p.types)-1] == SegMoveTo {
		// Java replaces a move that nothing was drawn from rather than keeping
		// two in a row.
		n := len(p.coords)
		p.coords[n-2] = p.store(x)
		p.coords[n-1] = p.store(y)
		return
	}
	p.appendSegment(SegMoveTo, x, y)
}

// LineTo draws a line from the current point to the given one.
func (p *Path2D) LineTo(x, y float64) {
	p.needMove()
	p.appendSegment(SegLineTo, x, y)
}

// QuadTo draws a quadratic curve from the current point through the control
// point (x1, y1) to (x2, y2).
func (p *Path2D) QuadTo(x1, y1, x2, y2 float64) {
	p.needMove()
	p.appendSegment(SegQuadTo, x1, y1, x2, y2)
}

// CurveTo draws a cubic curve from the current point through the control points
// (x1, y1) and (x2, y2) to (x3, y3).
func (p *Path2D) CurveTo(x1, y1, x2, y2, x3, y3 float64) {
	p.needMove()
	p.appendSegment(SegCubicTo, x1, y1, x2, y2, x3, y3)
}

// ClosePath draws a line back to the point the current subpath began at.
func (p *Path2D) ClosePath() {
	if len(p.types) == 0 || p.types[len(p.types)-1] != SegClose {
		p.needMove()
		p.types = append(p.types, SegClose)
	}
}

// needMove rejects a segment added before any subpath has been begun.
//
// Java throws the unchecked IllegalPathStateException from needRoom here; it is
// a caller's mistake rather than bad input, so the port panics rather than
// putting an error return on every drawing method.
func (p *Path2D) needMove() {
	if len(p.types) == 0 {
		panic("geom: missing initial moveto in path definition")
	}
}

// Reset empties the path, keeping the winding rule.
func (p *Path2D) Reset() {
	p.types = p.types[:0]
	p.coords = p.coords[:0]
}

// WindingRule returns the rule that decides the interior of the path.
func (p *Path2D) WindingRule() int { return p.windingRule }

// SetWindingRule sets the rule that decides the interior of the path.
func (p *Path2D) SetWindingRule(rule int) { p.windingRule = rule }

// CurrentPoint returns the point the last segment ended at, or nil for an empty
// path. After a close it is the point the closed subpath began at.
func (p *Path2D) CurrentPoint() Point2D {
	index := len(p.coords)
	if len(p.types) < 1 || index < 1 {
		return nil
	}
	if p.types[len(p.types)-1] == SegClose {
		// Walk back over the segments of the subpath just closed, to the move
		// that began it. As in Java the walk stops above index 0 rather than at
		// it, which is why a path that is nothing but a move and a close comes
		// back with the move's own point.
	loop:
		for i := len(p.types) - 2; i > 0; i-- {
			switch p.types[i] {
			case SegMoveTo:
				break loop
			case SegClose:
				// carries no points
			default:
				index -= segmentPointCount[p.types[i]] * 2
			}
		}
	}
	return NewPointDouble(p.coords[index-2], p.coords[index-1])
}

// Append adds the segments of the given shape to this path. When connect is
// true the first segment of the shape is turned into a line from the current
// point rather than a move.
func (p *Path2D) Append(s Shape, connect bool) {
	p.AppendIterator(s.PathIterator(nil), connect)
}

// AppendIterator adds the segments the iterator walks to this path.
func (p *Path2D) AppendIterator(pi PathIterator, connect bool) {
	coords := make([]float64, 6)
	for ; !pi.IsDone(); pi.Next() {
		switch pi.CurrentSegment(coords) {
		case SegMoveTo:
			if connect && len(p.types) > 0 {
				p.LineTo(coords[0], coords[1])
			} else {
				p.MoveTo(coords[0], coords[1])
			}
			// Only the first segment is connected.
			connect = false
		case SegLineTo:
			p.LineTo(coords[0], coords[1])
			connect = false
		case SegQuadTo:
			p.QuadTo(coords[0], coords[1], coords[2], coords[3])
			connect = false
		case SegCubicTo:
			p.CurveTo(coords[0], coords[1], coords[2], coords[3], coords[4], coords[5])
			connect = false
		case SegClose:
			p.ClosePath()
		}
	}
}

// Transform maps every point of the path through at, in place.
func (p *Path2D) Transform(at *AffineTransform) {
	for i := 0; i+1 < len(p.coords); i += 2 {
		x, y := p.coords[i], p.coords[i+1]
		p.coords[i] = p.store(x*at.m00 + y*at.m01 + at.m02)
		p.coords[i+1] = p.store(x*at.m10 + y*at.m11 + at.m12)
	}
}

// CreateTransformedShape returns a copy of this path mapped through at.
func (p *Path2D) CreateTransformedShape(at *AffineTransform) *Path2D {
	clone := p.Clone()
	if at != nil {
		clone.Transform(at)
	}
	return clone
}

// Clone returns an independent copy of the path.
func (p *Path2D) Clone() *Path2D {
	return &Path2D{
		types:           append([]byte(nil), p.types...),
		coords:          append([]float64(nil), p.coords...),
		windingRule:     p.windingRule,
		singlePrecision: p.singlePrecision,
	}
}

// Bounds2D returns the smallest rectangle enclosing every point the path names.
//
// As in Java this bounds the control points rather than the curves themselves,
// so a path with curves may come back with a rectangle larger than it needs.
func (p *Path2D) Bounds2D() *Rectangle2D {
	if len(p.coords) == 0 {
		return &Rectangle2D{}
	}
	x1, y1 := math.Inf(1), math.Inf(1)
	x2, y2 := math.Inf(-1), math.Inf(-1)
	for i := 0; i+1 < len(p.coords); i += 2 {
		x1 = math.Min(x1, p.coords[i])
		x2 = math.Max(x2, p.coords[i])
		y1 = math.Min(y1, p.coords[i+1])
		y2 = math.Max(y2, p.coords[i+1])
	}
	return NewRectangle2D(x1, y1, x2-x1, y2-y1)
}

// Bounds returns the smallest integer rectangle enclosing the path.
func (p *Path2D) Bounds() *Rectangle { return p.Bounds2D().Bounds() }

// PathIterator walks the segments of the path, mapping each point through at
// when at is not nil.
func (p *Path2D) PathIterator(at *AffineTransform) PathIterator {
	return &pathIterator{path: p, at: at}
}

// pathIterator is the cursor Path2D.PathIterator hands out. It reads the path
// as it stands; Java's iterators do the same and are documented as undefined if
// the path changes underneath them.
type pathIterator struct {
	path     *Path2D
	at       *AffineTransform
	typeIdx  int
	coordIdx int
}

func (it *pathIterator) WindingRule() int { return it.path.windingRule }

func (it *pathIterator) IsDone() bool { return it.typeIdx >= len(it.path.types) }

func (it *pathIterator) Next() {
	if it.IsDone() {
		return
	}
	it.coordIdx += segmentPointCount[it.path.types[it.typeIdx]] * 2
	it.typeIdx++
}

func (it *pathIterator) CurrentSegment(coords []float64) int {
	segment := it.path.types[it.typeIdx]
	n := segmentPointCount[segment] * 2
	copy(coords, it.path.coords[it.coordIdx:it.coordIdx+n])
	if it.at != nil && n > 0 {
		it.at.TransformDoubles(coords, 0, coords, 0, n/2)
	}
	return int(segment)
}
