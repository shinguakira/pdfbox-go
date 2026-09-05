package geom

import (
	"math"
	"sort"
)

// Area is a set of points in the plane, closed under the four set operations.
//
// Port of java.awt.geom.Area, which is the JDK rather than PDFBox --
// PDGraphicsState and PageDrawer are the two classes that use it, for the
// clipping path and for the shading and glyph clips. The JDK's own
// implementation is not in this repository, so this is written from the
// documented contract rather than translated; area_test.go says which parts of
// that contract it pins.
//
// # How the set is held
//
// An Area keeps its boundary as closed polygons, wound so that the interior is
// always on the left, and the set is the points those polygons enclose under
// the non-zero winding rule. That is why getPathIterator always reports
// WindNonZero whatever the source path used: the source rule is spent when the
// Area is built, which is the contract.
//
// # The one deviation, and what it costs
//
// The JDK holds curves and intersects them exactly. This flattens every curve
// to a polyline when the Area is built, so a curved boundary is an
// approximation from that point on, and getPathIterator answers lines where the
// JDK would answer the curves back. The error is bounded by flatness below,
// which is relative to the shape, and the endpoints of every curve survive
// exactly -- so a circle's extremes, which are what a bounding box is made of,
// are exact. Intersecting curve against curve exactly needs the resultant
// machinery the JDK keeps in sun.awt.geom, which is a port of its own. See
// migration/STATUS.md.
type Area struct {
	// rings are the closed boundary polygons. Each holds its first point once,
	// not twice: the closing edge runs from the last point back to the first.
	rings [][]point
}

var _ Shape = (*Area)(nil)

// point is one vertex. Point2D is an interface in this package, so the rings
// hold a concrete pair rather than boxing every vertex.
type point struct{ x, y float64 }

// flatness is how far a flattened curve may stray from the true one, as a
// fraction of the shape's diagonal. It bounds the error the deviation above
// describes.
const flatness = 1e-4

// maxSubdivisions caps the recursion that flattens one curve, so that a
// degenerate control polygon cannot spin.
const maxSubdivisions = 16

// NewArea returns an empty area.
//
// Port of the Area() constructor.
func NewArea() *Area { return &Area{} }

// NewAreaOfShape returns the area the given shape encloses, under that shape's
// own winding rule.
//
// Port of the Area(Shape) constructor.
func NewAreaOfShape(s Shape) *Area {
	if s == nil {
		return NewArea()
	}
	if other, isArea := s.(*Area); isArea {
		return &Area{rings: cloneRings(other.rings)}
	}
	source, rule := flattenShape(s)
	if len(source) == 0 {
		return NewArea()
	}
	return &Area{rings: resolve(source, func(p point) bool {
		return insideRings(source, p, rule)
	})}
}

// Add makes this area the union of itself and the given one.
func (a *Area) Add(other *Area) { a.combine(other, func(inA, inB bool) bool { return inA || inB }) }

// Intersect makes this area the intersection of itself and the given one.
func (a *Area) Intersect(other *Area) {
	a.combine(other, func(inA, inB bool) bool { return inA && inB })
}

// Subtract removes the points of the given area from this one.
func (a *Area) Subtract(other *Area) {
	a.combine(other, func(inA, inB bool) bool { return inA && !inB })
}

// ExclusiveOr keeps the points in one area or the other but not in both.
func (a *Area) ExclusiveOr(other *Area) {
	a.combine(other, func(inA, inB bool) bool { return inA != inB })
}

// combine replaces this area with the result of a set operation against the
// given one, keep saying which combinations of membership are in the result.
func (a *Area) combine(other *Area, keep func(inA, inB bool) bool) {
	var otherRings [][]point
	if other != nil {
		otherRings = other.rings
	}
	// The two shortcuts below are not an optimisation: with one side empty
	// there are no edges from it to split against, and the general path would
	// still be correct, but these save building the whole edge set for the
	// commonest cases.
	if len(otherRings) == 0 {
		if !keep(true, false) {
			a.rings = nil
		}
		return
	}
	if len(a.rings) == 0 {
		if keep(false, true) {
			a.rings = cloneRings(otherRings)
		} else {
			a.rings = nil
		}
		return
	}

	mine := a.rings
	all := append(cloneRings(mine), cloneRings(otherRings)...)
	a.rings = resolve(all, func(p point) bool {
		return keep(insideRings(mine, p, WindNonZero), insideRings(otherRings, p, WindNonZero))
	})
}

// Reset empties the area.
func (a *Area) Reset() { a.rings = nil }

// IsEmpty reports whether the area holds no points.
func (a *Area) IsEmpty() bool { return len(a.rings) == 0 }

// Contains reports whether the given point is in the set.
//
// A point exactly on the boundary is not promised either way, which is the
// contract java.awt.geom.Area states.
func (a *Area) Contains(x, y float64) bool {
	return insideRings(a.rings, point{x, y}, WindNonZero)
}

// Bounds2D returns a rectangle that encloses the area, and the empty rectangle
// where the area is empty.
func (a *Area) Bounds2D() *Rectangle2D {
	if len(a.rings) == 0 {
		return &Rectangle2D{}
	}
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, ring := range a.rings {
		for _, p := range ring {
			minX = math.Min(minX, p.x)
			minY = math.Min(minY, p.y)
			maxX = math.Max(maxX, p.x)
			maxY = math.Max(maxY, p.y)
		}
	}
	return &Rectangle2D{X: minX, Y: minY, Width: maxX - minX, Height: maxY - minY}
}

// Bounds returns the smallest integer rectangle that encloses the area.
func (a *Area) Bounds() *Rectangle { return a.Bounds2D().Bounds() }

// Transform maps every point of the area through the given transform.
func (a *Area) Transform(at *AffineTransform) {
	if at == nil {
		return
	}
	for _, ring := range a.rings {
		for i, p := range ring {
			src := NewPointDouble(p.x, p.y)
			dst := at.Transform(src, nil)
			ring[i] = point{dst.X(), dst.Y()}
		}
	}
}

// CreateTransformedArea returns a copy of this area mapped through the given
// transform, leaving this one alone.
func (a *Area) CreateTransformedArea(at *AffineTransform) *Area {
	copied := &Area{rings: cloneRings(a.rings)}
	copied.Transform(at)
	return copied
}

// Clone returns a copy of the area.
func (a *Area) Clone() *Area { return &Area{rings: cloneRings(a.rings)} }

// Equals reports whether the two areas hold the same rings in the same order.
//
// Java's Area.equals compares the sets, by subtracting each from the other.
// This is the cheaper test and answers false for two areas that hold the same
// points through different rings, so a caller that needs the set comparison
// subtracts.
func (a *Area) Equals(other *Area) bool {
	if other == nil || len(a.rings) != len(other.rings) {
		return false
	}
	for i, ring := range a.rings {
		if len(ring) != len(other.rings[i]) {
			return false
		}
		for j, p := range ring {
			if p != other.rings[i][j] {
				return false
			}
		}
	}
	return true
}

// PathIterator walks the boundary of the area, always reporting WindNonZero.
func (a *Area) PathIterator(at *AffineTransform) PathIterator {
	path := NewPathDoubleRule(WindNonZero)
	for _, ring := range a.rings {
		if len(ring) < 3 {
			continue
		}
		path.MoveTo(ring[0].x, ring[0].y)
		for _, p := range ring[1:] {
			path.LineTo(p.x, p.y)
		}
		path.ClosePath()
	}
	return path.PathIterator(at)
}

// cloneRings copies the rings so that two areas never share one.
func cloneRings(rings [][]point) [][]point {
	if len(rings) == 0 {
		return nil
	}
	out := make([][]point, len(rings))
	for i, ring := range rings {
		out[i] = append([]point(nil), ring...)
	}
	return out
}

// flattenShape walks a shape and returns its closed subpaths as polygons,
// together with the winding rule they are to be read under.
//
// An open subpath is closed, which is what filling one does.
func flattenShape(s Shape) ([][]point, int) {
	it := s.PathIterator(nil)
	rule := it.WindingRule()
	tolerance := flatness * shapeDiagonal(s)
	if tolerance <= 0 {
		tolerance = flatness
	}

	var rings [][]point
	var current []point
	coords := make([]float64, 6)
	var last point
	flush := func() {
		if len(current) >= 3 {
			rings = append(rings, current)
		}
		current = nil
	}
	for ; !it.IsDone(); it.Next() {
		switch it.CurrentSegment(coords) {
		case SegMoveTo:
			flush()
			last = point{coords[0], coords[1]}
			current = []point{last}
		case SegLineTo:
			last = point{coords[0], coords[1]}
			current = append(current, last)
		case SegQuadTo:
			control := point{coords[0], coords[1]}
			end := point{coords[2], coords[3]}
			// A quadratic is the cubic with its control points two thirds of
			// the way to the quadratic's one, which is the standard lift.
			c1 := point{last.x + 2.0/3.0*(control.x-last.x), last.y + 2.0/3.0*(control.y-last.y)}
			c2 := point{end.x + 2.0/3.0*(control.x-end.x), end.y + 2.0/3.0*(control.y-end.y)}
			current = flattenCubic(current, last, c1, c2, end, tolerance, 0)
			last = end
		case SegCubicTo:
			c1 := point{coords[0], coords[1]}
			c2 := point{coords[2], coords[3]}
			end := point{coords[4], coords[5]}
			current = flattenCubic(current, last, c1, c2, end, tolerance, 0)
			last = end
		case SegClose:
			flush()
		}
	}
	flush()
	return rings, rule
}

// shapeDiagonal is the diagonal of the shape's bounding box, which the
// flattening tolerance is taken as a fraction of.
func shapeDiagonal(s Shape) float64 {
	b := s.Bounds2D()
	return math.Hypot(b.Width, b.Height)
}

// flattenCubic appends the points of a cubic curve to a polyline, subdividing
// until the control polygon is within tolerance of the chord.
func flattenCubic(into []point, p0, p1, p2, p3 point, tolerance float64, depth int) []point {
	if depth >= maxSubdivisions || cubicIsFlat(p0, p1, p2, p3, tolerance) {
		return append(into, p3)
	}
	// de Casteljau at the midpoint.
	p01 := midpoint(p0, p1)
	p12 := midpoint(p1, p2)
	p23 := midpoint(p2, p3)
	p012 := midpoint(p01, p12)
	p123 := midpoint(p12, p23)
	mid := midpoint(p012, p123)
	into = flattenCubic(into, p0, p01, p012, mid, tolerance, depth+1)
	return flattenCubic(into, mid, p123, p23, p3, tolerance, depth+1)
}

// cubicIsFlat reports whether both control points lie within tolerance of the
// chord, which is the usual flatness test.
func cubicIsFlat(p0, p1, p2, p3 point, tolerance float64) bool {
	return distanceToLine(p1, p0, p3) <= tolerance && distanceToLine(p2, p0, p3) <= tolerance
}

// distanceToLine is the distance from p to the line through a and b, or to a
// where the two are the same point.
func distanceToLine(p, a, b point) float64 {
	dx, dy := b.x-a.x, b.y-a.y
	length := math.Hypot(dx, dy)
	if length == 0 {
		return math.Hypot(p.x-a.x, p.y-a.y)
	}
	return math.Abs((p.x-a.x)*dy-(p.y-a.y)*dx) / length
}

func midpoint(a, b point) point { return point{(a.x + b.x) / 2, (a.y + b.y) / 2} }

// insideRings reports whether p is inside the given polygons under the given
// winding rule, by counting the crossings of a ray going right from p.
func insideRings(rings [][]point, p point, rule int) bool {
	winding := 0
	crossings := 0
	for _, ring := range rings {
		for i := range ring {
			a := ring[i]
			b := ring[(i+1)%len(ring)]
			if a.y == b.y {
				continue
			}
			// Count the edge when the ray at p.y crosses it, taking each edge
			// as half-open in y so that a vertex is counted once.
			if (a.y <= p.y) == (b.y <= p.y) {
				continue
			}
			x := a.x + (p.y-a.y)/(b.y-a.y)*(b.x-a.x)
			if x <= p.x {
				continue
			}
			crossings++
			if b.y > a.y {
				winding++
			} else {
				winding--
			}
		}
	}
	if rule == WindEvenOdd {
		return crossings%2 == 1
	}
	return winding != 0
}

// segment is one directed piece of a boundary, after splitting.
type segment struct{ a, b point }

// resolve rebuilds the boundary of the set the given predicate describes, out
// of the edges of the given rings.
//
// Every edge is cut at each crossing with every other, so that each resulting
// piece lies wholly inside or wholly outside the result. A piece is kept when
// the set is on one side of it and not the other, oriented with the inside on
// its left, and the kept pieces are then chained into closed rings.
func resolve(rings [][]point, inside func(point) bool) [][]point {
	segments := splitEdges(rings)
	if len(segments) == 0 {
		return nil
	}
	scale := ringsDiagonal(rings)

	kept := make([]segment, 0, len(segments))
	for _, s := range segments {
		length := math.Hypot(s.b.x-s.a.x, s.b.y-s.a.y)
		if length == 0 {
			continue
		}
		// Step off the midpoint perpendicular to the piece, far enough to be
		// clear of rounding and never so far as to cross a neighbouring
		// feature: a quarter of the piece is always short of the next vertex.
		step := math.Min(1e-6*scale, length/4)
		if step == 0 {
			continue
		}
		mid := midpoint(s.a, s.b)
		nx, ny := -(s.b.y-s.a.y)/length, (s.b.x-s.a.x)/length
		leftInside := inside(point{mid.x + nx*step, mid.y + ny*step})
		rightInside := inside(point{mid.x - nx*step, mid.y - ny*step})
		switch {
		case leftInside && !rightInside:
			kept = append(kept, s)
		case rightInside && !leftInside:
			kept = append(kept, segment{a: s.b, b: s.a})
		}
	}
	return chain(kept, scale)
}

// ringsDiagonal is the diagonal of the bounding box of every ring, which the
// tolerances above and below are taken as a fraction of.
func ringsDiagonal(rings [][]point) float64 {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, ring := range rings {
		for _, p := range ring {
			minX = math.Min(minX, p.x)
			minY = math.Min(minY, p.y)
			maxX = math.Max(maxX, p.x)
			maxY = math.Max(maxY, p.y)
		}
	}
	if math.IsInf(minX, 1) {
		return 1
	}
	diagonal := math.Hypot(maxX-minX, maxY-minY)
	if diagonal == 0 {
		return 1
	}
	return diagonal
}

// splitEdges cuts every edge of every ring at each point where another edge
// crosses it, so that no two resulting pieces cross.
func splitEdges(rings [][]point) []segment {
	var edges []segment
	for _, ring := range rings {
		for i := range ring {
			a, b := ring[i], ring[(i+1)%len(ring)]
			if a != b {
				edges = append(edges, segment{a, b})
			}
		}
	}

	// cuts[i] holds the parameters along edge i at which it is to be cut.
	cuts := make([][]float64, len(edges))
	for i := 0; i < len(edges); i++ {
		for j := i + 1; j < len(edges); j++ {
			ti, tj, ok := intersectSegments(edges[i], edges[j])
			if !ok {
				continue
			}
			cuts[i] = append(cuts[i], ti)
			cuts[j] = append(cuts[j], tj)
		}
	}

	// An edge must also be cut where another edge merely touches it, at a
	// vertex rather than a crossing. Tangency is not a corner case here: the
	// clipping path of PDGraphicsState intersects each path against the
	// bounding box of them all, so the very first intersection has the two
	// touching wherever the path reaches its own extreme. Without this cut the
	// touched edge stays whole, and one point of it being on the result makes
	// the classification below keep the lot.
	touchTolerance := 1e-9 * ringsDiagonal(rings)
	for i, e := range edges {
		for j, other := range edges {
			if i == j {
				continue
			}
			for _, end := range []point{other.a, other.b} {
				if t, ok := pointAlongSegment(e, end, touchTolerance); ok {
					cuts[i] = append(cuts[i], t)
				}
			}
		}
	}

	var out []segment
	for i, e := range edges {
		ts := append([]float64{0, 1}, cuts[i]...)
		sort.Float64s(ts)
		previous := ts[0]
		for _, t := range ts[1:] {
			if t-previous < 1e-12 {
				continue
			}
			out = append(out, segment{a: along(e, previous), b: along(e, t)})
			previous = t
		}
	}
	return out
}

// along returns the point a fraction t of the way along a segment.
func along(s segment, t float64) point {
	if t <= 0 {
		return s.a
	}
	if t >= 1 {
		return s.b
	}
	return point{s.a.x + t*(s.b.x-s.a.x), s.a.y + t*(s.b.y-s.a.y)}
}

// intersectSegments returns where two segments cross, as the fraction along
// each, and reports false where they do not cross away from their ends.
//
// A crossing exactly at an end of either segment is not a cut: that end is
// already a vertex, so cutting there would only make a zero-length piece.
// Parallel segments are not cut either; an overlap between two of them leaves
// pieces whose sides are classified the same way, which the caller drops.
func intersectSegments(p, q segment) (float64, float64, bool) {
	rx, ry := p.b.x-p.a.x, p.b.y-p.a.y
	sx, sy := q.b.x-q.a.x, q.b.y-q.a.y
	denominator := rx*sy - ry*sx
	if denominator == 0 {
		return 0, 0, false
	}
	dx, dy := q.a.x-p.a.x, q.a.y-p.a.y
	t := (dx*sy - dy*sx) / denominator
	u := (dx*ry - dy*rx) / denominator
	const epsilon = 1e-12
	if t <= epsilon || t >= 1-epsilon || u <= epsilon || u >= 1-epsilon {
		return 0, 0, false
	}
	return t, u, true
}

// chain joins the kept pieces head to tail into closed rings.
//
// Every kept piece has exactly one successor in a well formed result, so this
// walks from each unused piece until it returns to where it started. A piece
// whose successor is missing -- which rounding can leave -- ends its ring
// early, and the ring is kept if it still has three points.
func chain(segments []segment, scale float64) [][]point {
	if len(segments) == 0 {
		return nil
	}
	tolerance := 1e-9 * scale
	starts := map[[2]int64][]int{}
	for i, s := range segments {
		key := quantise(s.a, tolerance)
		starts[key] = append(starts[key], i)
	}

	used := make([]bool, len(segments))
	var rings [][]point
	for i := range segments {
		if used[i] {
			continue
		}
		used[i] = true
		ring := []point{segments[i].a}
		at := segments[i].b
		from := segments[i].a
		for {
			if distance(at, from) <= tolerance && len(ring) >= 3 {
				break
			}
			next := -1
			for _, candidate := range starts[quantise(at, tolerance)] {
				if !used[candidate] {
					next = candidate
					break
				}
			}
			if next < 0 {
				break
			}
			used[next] = true
			ring = append(ring, segments[next].a)
			at = segments[next].b
		}
		if len(ring) >= 3 {
			rings = append(rings, ring)
		}
	}
	return rings
}

// quantise rounds a point onto a grid of the given size, so that two ends that
// meet are looked up under one key.
func quantise(p point, tolerance float64) [2]int64 {
	if tolerance <= 0 {
		tolerance = 1e-12
	}
	return [2]int64{int64(math.Round(p.x / tolerance)), int64(math.Round(p.y / tolerance))}
}

func distance(a, b point) float64 { return math.Hypot(a.x-b.x, a.y-b.y) }

// pointAlongSegment reports where p sits along s, and false unless p lies on
// the segment away from both of its ends.
//
// It is how a vertex of one edge is found to be touching the middle of
// another, which splitEdges needs and which a crossing test cannot see: the
// two edges do not cross there, they meet.
func pointAlongSegment(s segment, p point, tolerance float64) (float64, bool) {
	dx, dy := s.b.x-s.a.x, s.b.y-s.a.y
	lengthSquared := dx*dx + dy*dy
	if lengthSquared == 0 {
		return 0, false
	}
	t := ((p.x-s.a.x)*dx + (p.y-s.a.y)*dy) / lengthSquared
	if t <= 0 || t >= 1 {
		return 0, false
	}
	nearest := point{s.a.x + t*dx, s.a.y + t*dy}
	if distance(nearest, p) > tolerance {
		return 0, false
	}
	// Leave the ends alone: a cut there would only make a zero-length piece.
	length := math.Sqrt(lengthSquared)
	if t*length <= tolerance || (1-t)*length <= tolerance {
		return 0, false
	}
	return t, true
}
