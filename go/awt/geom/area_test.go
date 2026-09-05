package geom

// These tests are written from the documented contract of java.awt.geom.Area,
// not from any implementation of it. The JDK's Area is not in this repository
// -- only PDFBox's Java is -- so there is nothing to port line for line, and
// the slice 9 plan says so: "write Area tests from the JDK contract, not from
// the implementation".
//
// What the contract says, in the terms these tests use:
//
//   - An Area is a set of points. add, subtract, intersect and exclusiveOr are
//     the four set operations, and each replaces the receiver.
//   - The interior is decided before the operation: an Area built from an
//     even-odd path holds the same points that path encloses under even-odd,
//     and from then on the winding rule of the source is spent. Area's own
//     path iterator always reports WIND_NON_ZERO.
//   - contains asks whether a point is in the set. A point on the boundary is
//     not promised either way, so no test here puts one there.
//   - isEmpty is true exactly when the set has no points.
//   - getBounds2D encloses the set, and for these inputs is tight.

import (
	"math"
	"testing"
)

// rect is a rectangle as a Shape, which is the commonest thing PDFBox makes an
// Area from.
func rect(x, y, w, h float64) *Rectangle2D {
	return &Rectangle2D{X: x, Y: y, Width: w, Height: h}
}

// assertBounds checks a bounding box to within a tolerance that allows for a
// curve having been flattened, but not for a wrong answer.
func assertBounds(t *testing.T, what string, got *Rectangle2D, x, y, w, h float64) {
	t.Helper()
	const tolerance = 1e-6
	if math.Abs(got.X-x) > tolerance || math.Abs(got.Y-y) > tolerance ||
		math.Abs(got.Width-w) > tolerance || math.Abs(got.Height-h) > tolerance {
		t.Errorf("%s bounds = (%g,%g %gx%g), want (%g,%g %gx%g)",
			what, got.X, got.Y, got.Width, got.Height, x, y, w, h)
	}
}

// assertContains checks membership of a list of points, each given with what it
// should answer.
func assertContains(t *testing.T, what string, a *Area, points []struct {
	x, y float64
	want bool
}) {
	t.Helper()
	for _, p := range points {
		if got := a.Contains(p.x, p.y); got != p.want {
			t.Errorf("%s: Contains(%g, %g) = %v, want %v", what, p.x, p.y, got, p.want)
		}
	}
}

// TestEmptyArea is the no-argument constructor: an area with no points at all.
func TestEmptyArea(t *testing.T) {
	a := NewArea()
	if !a.IsEmpty() {
		t.Error("NewArea().IsEmpty() = false, want true")
	}
	if got := a.Bounds2D(); got.Width != 0 || got.Height != 0 {
		t.Errorf("NewArea().Bounds2D() = %v, want an empty rectangle", got)
	}
	if a.Contains(0, 0) {
		t.Error("NewArea().Contains(0, 0) = true, want false")
	}
}

// TestAreaOfRectangle is the Shape constructor over the simplest shape.
func TestAreaOfRectangle(t *testing.T) {
	a := NewAreaOfShape(rect(10, 20, 30, 40))
	if a.IsEmpty() {
		t.Error("IsEmpty() = true, want false")
	}
	assertBounds(t, "Area(rect)", a.Bounds2D(), 10, 20, 30, 40)
	assertContains(t, "Area(rect)", a, []struct {
		x, y float64
		want bool
	}{
		{25, 40, true},  // the middle
		{11, 21, true},  // just inside a corner
		{9, 40, false},  // left of it
		{41, 40, false}, // right of it
		{25, 19, false}, // below it
		{25, 61, false}, // above it
	})
}

// TestAddDisjoint unions two rectangles that do not touch.
func TestAddDisjoint(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 10, 10))
	a.Add(NewAreaOfShape(rect(20, 0, 10, 10)))
	assertBounds(t, "disjoint union", a.Bounds2D(), 0, 0, 30, 10)
	assertContains(t, "disjoint union", a, []struct {
		x, y float64
		want bool
	}{
		{5, 5, true},
		{25, 5, true},
		{15, 5, false}, // the gap between them
	})
}

// TestAddOverlapping unions two rectangles that overlap, which must leave one
// region rather than two.
func TestAddOverlapping(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 20, 20))
	a.Add(NewAreaOfShape(rect(10, 10, 20, 20)))
	assertBounds(t, "overlapping union", a.Bounds2D(), 0, 0, 30, 30)
	assertContains(t, "overlapping union", a, []struct {
		x, y float64
		want bool
	}{
		{5, 5, true},   // only in the first
		{25, 25, true}, // only in the second
		{15, 15, true}, // in both
		{25, 5, false}, // in neither
		{5, 25, false}, // in neither
	})
}

// TestIntersectOverlapping is the operation PDFBox leans on: the clipping path
// is a run of intersections.
func TestIntersectOverlapping(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 20, 20))
	a.Intersect(NewAreaOfShape(rect(10, 10, 20, 20)))
	assertBounds(t, "intersection", a.Bounds2D(), 10, 10, 10, 10)
	assertContains(t, "intersection", a, []struct {
		x, y float64
		want bool
	}{
		{15, 15, true},
		{5, 5, false},
		{25, 25, false},
	})
}

// TestIntersectDisjoint leaves nothing.
func TestIntersectDisjoint(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 10, 10))
	a.Intersect(NewAreaOfShape(rect(20, 0, 10, 10)))
	if !a.IsEmpty() {
		t.Errorf("IsEmpty() = false, want true; bounds %v", a.Bounds2D())
	}
}

// TestIntersectWithEmpty leaves nothing, and adding nothing changes nothing.
func TestIntersectWithEmpty(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 10, 10))
	a.Intersect(NewArea())
	if !a.IsEmpty() {
		t.Error("intersecting with an empty area left something")
	}

	b := NewAreaOfShape(rect(0, 0, 10, 10))
	b.Add(NewArea())
	assertBounds(t, "adding an empty area", b.Bounds2D(), 0, 0, 10, 10)
	if !b.Contains(5, 5) {
		t.Error("adding an empty area lost the original")
	}
}

// TestIntersectNested keeps the inner one where one contains the other.
func TestIntersectNested(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 100, 100))
	a.Intersect(NewAreaOfShape(rect(25, 25, 10, 10)))
	assertBounds(t, "nested intersection", a.Bounds2D(), 25, 25, 10, 10)
	assertContains(t, "nested intersection", a, []struct {
		x, y float64
		want bool
	}{
		{30, 30, true},
		{50, 50, false},
	})
}

// TestSubtract removes one set from another, leaving a shape with a bite out of
// it rather than a smaller rectangle.
func TestSubtract(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 20, 20))
	a.Subtract(NewAreaOfShape(rect(10, 10, 20, 20)))
	assertBounds(t, "subtraction", a.Bounds2D(), 0, 0, 20, 20)
	assertContains(t, "subtraction", a, []struct {
		x, y float64
		want bool
	}{
		{5, 5, true},    // untouched
		{15, 5, true},   // below the bite
		{5, 15, true},   // left of the bite
		{15, 15, false}, // the bite
	})
}

// TestExclusiveOr keeps what is in one set or the other but not both.
func TestExclusiveOr(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 20, 20))
	a.ExclusiveOr(NewAreaOfShape(rect(10, 10, 20, 20)))
	assertBounds(t, "exclusive or", a.Bounds2D(), 0, 0, 30, 30)
	assertContains(t, "exclusive or", a, []struct {
		x, y float64
		want bool
	}{
		{5, 5, true},
		{25, 25, true},
		{15, 15, false}, // in both, so in neither
	})
}

// TestReset empties an area in place, which is what PDGraphicsState does to
// each intermediate clipping area.
func TestReset(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 10, 10))
	a.Reset()
	if !a.IsEmpty() {
		t.Error("Reset() left something behind")
	}
	if got := a.Bounds2D(); got.Width != 0 || got.Height != 0 {
		t.Errorf("Bounds2D() after Reset = %v, want an empty rectangle", got)
	}
}

// TestAreaOfEvenOddPathWithAHole checks that the source winding rule decides
// the set. The path is an outer square and an inner square, both wound the same
// way; under even-odd the inner one is a hole, and the Area must hold that hole
// even though its own iterator reports non-zero winding.
func TestAreaOfEvenOddPathWithAHole(t *testing.T) {
	path := NewPathDoubleRule(WindEvenOdd)
	appendRect(path, 0, 0, 30, 30)
	appendRect(path, 10, 10, 10, 10)
	a := NewAreaOfShape(path)

	assertBounds(t, "square with a hole", a.Bounds2D(), 0, 0, 30, 30)
	assertContains(t, "square with a hole", a, []struct {
		x, y float64
		want bool
	}{
		{5, 5, true},    // between the squares
		{15, 15, false}, // the hole
		{29, 29, true},  // between the squares, far corner
	})
}

// TestAreaOfNonZeroPathWithNoHole is the same two squares under the non-zero
// rule, where the inner one is not a hole because both are wound the same way.
func TestAreaOfNonZeroPathWithNoHole(t *testing.T) {
	path := NewPathDoubleRule(WindNonZero)
	appendRect(path, 0, 0, 30, 30)
	appendRect(path, 10, 10, 10, 10)
	a := NewAreaOfShape(path)

	assertContains(t, "square without a hole", a, []struct {
		x, y float64
		want bool
	}{
		{5, 5, true},
		{15, 15, true},
	})
}

// TestPathIteratorReportsNonZero is the contract on Area's own iterator: it
// always answers WIND_NON_ZERO, whatever the source path used.
func TestPathIteratorReportsNonZero(t *testing.T) {
	path := NewPathDoubleRule(WindEvenOdd)
	appendRect(path, 0, 0, 10, 10)
	a := NewAreaOfShape(path)
	if got := a.PathIterator(nil).WindingRule(); got != WindNonZero {
		t.Errorf("PathIterator().WindingRule() = %d, want WindNonZero", got)
	}
}

// TestRoundTripThroughPath is what PDGraphicsState does with the result: it
// builds a Path2D from the Area and keeps that. The points must survive.
func TestRoundTripThroughPath(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 20, 20))
	a.Intersect(NewAreaOfShape(rect(10, 10, 20, 20)))

	again := NewAreaOfShape(NewPathDoubleShape(a))
	assertBounds(t, "round trip", again.Bounds2D(), 10, 10, 10, 10)
	assertContains(t, "round trip", again, []struct {
		x, y float64
		want bool
	}{
		{15, 15, true},
		{5, 5, false},
	})
}

// TestTransform moves an area through an affine transform, which is how a
// bounding box in form space reaches device space.
func TestTransform(t *testing.T) {
	a := NewAreaOfShape(rect(0, 0, 10, 10))
	a.Transform(NewAffineTransform(2, 0, 0, 2, 5, 5))
	assertBounds(t, "scaled and translated", a.Bounds2D(), 5, 5, 20, 20)
	assertContains(t, "scaled and translated", a, []struct {
		x, y float64
		want bool
	}{
		{15, 15, true},
		{1, 1, false},
	})
}

// TestIntersectOfACurve checks that a shape with a curve in it survives the
// operations. The circle is inscribed in the square, so intersecting them is
// the circle, and its bounds are the square's.
func TestIntersectOfACurve(t *testing.T) {
	circle := NewPathDouble()
	appendCircle(circle, 50, 50, 25)
	a := NewAreaOfShape(circle)
	a.Intersect(NewAreaOfShape(rect(25, 25, 50, 50)))

	assertBounds(t, "circle in a square", a.Bounds2D(), 25, 25, 50, 50)
	assertContains(t, "circle in a square", a, []struct {
		x, y float64
		want bool
	}{
		{50, 50, true},  // the centre
		{27, 27, false}, // a corner of the square, outside the circle
		{73, 73, false},
	})
}

// TestIntersectHalfOfACurve cuts the circle in half, which the bounds must
// follow.
func TestIntersectHalfOfACurve(t *testing.T) {
	circle := NewPathDouble()
	appendCircle(circle, 50, 50, 25)
	a := NewAreaOfShape(circle)
	a.Intersect(NewAreaOfShape(rect(0, 0, 50, 100)))

	// The left half of the circle: x from 25 to 50, y the full diameter.
	bounds := a.Bounds2D()
	assertBounds(t, "left half of a circle", bounds, 25, 25, 25, 50)
	assertContains(t, "left half of a circle", a, []struct {
		x, y float64
		want bool
	}{
		{40, 50, true},
		{60, 50, false},
	})
}

// appendRect adds a closed rectangle to a path, wound anticlockwise.
func appendRect(p *Path2D, x, y, w, h float64) {
	p.MoveTo(x, y)
	p.LineTo(x+w, y)
	p.LineTo(x+w, y+h)
	p.LineTo(x, y+h)
	p.ClosePath()
}

// appendCircle adds a closed circle to a path, as the four cubic curves the
// usual kappa approximation uses.
func appendCircle(p *Path2D, cx, cy, r float64) {
	const kappa = 0.5522847498307933
	k := r * kappa
	p.MoveTo(cx+r, cy)
	p.CurveTo(cx+r, cy+k, cx+k, cy+r, cx, cy+r)
	p.CurveTo(cx-k, cy+r, cx-r, cy+k, cx-r, cy)
	p.CurveTo(cx-r, cy-k, cx-k, cy-r, cx, cy-r)
	p.CurveTo(cx+k, cy-r, cx+r, cy-k, cx+r, cy)
	p.ClosePath()
}

// TestIntersectWithOwnBoundingBox is the shape PDGraphicsState.getCurrentClippingPath
// makes on its very first step: it starts from the bounding box of every
// clipping path and intersects each path into it, so the first intersection is
// always a shape against a box it touches at each of its own extremes.
//
// Tangency is therefore the common case here rather than a corner one, and
// intersecting a shape with a box that encloses it must give the shape back.
func TestIntersectWithOwnBoundingBox(t *testing.T) {
	for _, c := range []struct {
		name  string
		build func(*Path2D)
		in    [][2]float64
		out   [][2]float64
	}{
		{
			name:  "a triangle",
			build: func(p *Path2D) { appendTriangle(p, 0, 0, 100, 0, 50, 100) },
			in:    [][2]float64{{50, 50}, {50, 10}},
			out:   [][2]float64{{5, 90}, {95, 90}},
		},
		{
			name:  "a circle",
			build: func(p *Path2D) { appendCircle(p, 50, 50, 25) },
			in:    [][2]float64{{50, 50}, {50, 70}},
			out:   [][2]float64{{27, 27}, {73, 73}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			path := NewPathDouble()
			c.build(path)
			shape := NewAreaOfShape(path)
			box := shape.Bounds2D()

			shape.Intersect(NewAreaOfShape(box))

			assertBounds(t, c.name+" after intersecting its own box", shape.Bounds2D(),
				box.X, box.Y, box.Width, box.Height)
			for _, p := range c.in {
				if !shape.Contains(p[0], p[1]) {
					t.Errorf("Contains(%g, %g) = false, want true: the shape lost points",
						p[0], p[1])
				}
			}
			for _, p := range c.out {
				if shape.Contains(p[0], p[1]) {
					t.Errorf("Contains(%g, %g) = true, want false: the shape gained the box",
						p[0], p[1])
				}
			}
		})
	}
}

// appendTriangle adds a closed triangle to a path.
func appendTriangle(p *Path2D, x0, y0, x1, y1, x2, y2 float64) {
	p.MoveTo(x0, y0)
	p.LineTo(x1, y1)
	p.LineTo(x2, y2)
	p.ClosePath()
}
