package geom

import "testing"

func TestRectangle2DExtents(t *testing.T) {
	r := NewRectangle2D(1, 2, 3, 4)

	if got := r.MinX(); got != 1 {
		t.Errorf("MinX = %v, want 1", got)
	}
	if got := r.MinY(); got != 2 {
		t.Errorf("MinY = %v, want 2", got)
	}
	if got := r.MaxX(); got != 4 {
		t.Errorf("MaxX = %v, want 4", got)
	}
	if got := r.MaxY(); got != 6 {
		t.Errorf("MaxY = %v, want 6", got)
	}
	if got := r.CenterX(); got != 2.5 {
		t.Errorf("CenterX = %v, want 2.5", got)
	}
	if got := r.CenterY(); got != 4 {
		t.Errorf("CenterY = %v, want 4", got)
	}
}

// TestRectangle2DIsEmpty pins that a zero extent counts as empty, not just a
// negative one.
func TestRectangle2DIsEmpty(t *testing.T) {
	cases := []struct {
		r    *Rectangle2D
		want bool
	}{
		{NewRectangle2D(0, 0, 1, 1), false},
		{NewRectangle2D(0, 0, 0, 1), true},
		{NewRectangle2D(0, 0, 1, 0), true},
		{NewRectangle2D(0, 0, -1, 1), true},
	}
	for _, c := range cases {
		if got := c.r.IsEmpty(); got != c.want {
			t.Errorf("%v.IsEmpty() = %v, want %v", c.r, got, c.want)
		}
	}
}

// TestRectangle2DContains pins that the rectangle holds its lower edges and not
// its upper ones.
func TestRectangle2DContains(t *testing.T) {
	r := NewRectangle2D(0, 0, 10, 10)
	cases := []struct {
		x, y float64
		want bool
	}{
		{5, 5, true},
		{0, 0, true},
		{10, 10, false},
		{10, 5, false},
		{-1, 5, false},
	}
	for _, c := range cases {
		if got := r.Contains(c.x, c.y); got != c.want {
			t.Errorf("Contains(%v, %v) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
}

func TestRectangle2DIntersects(t *testing.T) {
	r := NewRectangle2D(0, 0, 10, 10)
	if !r.Intersects(NewRectangle2D(5, 5, 10, 10)) {
		t.Error("overlapping rectangles do not intersect")
	}
	// Touching edges do not count as an intersection.
	if r.Intersects(NewRectangle2D(10, 0, 10, 10)) {
		t.Error("touching rectangles intersect")
	}
	if r.Intersects(NewRectangle2D(0, 0, 0, 0)) {
		t.Error("an empty rectangle intersects")
	}
}

// TestRectangle2DIntersect pins that rectangles which do not meet give a
// rectangle with a negative extent rather than a zero one, which is how the
// caller is meant to tell them apart.
func TestRectangle2DIntersect(t *testing.T) {
	dest := &Rectangle2D{}
	Intersect(NewRectangle2D(0, 0, 10, 10), NewRectangle2D(5, 5, 10, 10), dest)
	if want := NewRectangle2D(5, 5, 5, 5); *dest != *want {
		t.Errorf("Intersect = %v, want %v", dest, want)
	}

	Intersect(NewRectangle2D(0, 0, 1, 1), NewRectangle2D(5, 5, 1, 1), dest)
	if !dest.IsEmpty() {
		t.Errorf("Intersect of disjoint rectangles = %v, want an empty one", dest)
	}

	// dest may be one of the sources.
	r := NewRectangle2D(0, 0, 10, 10)
	Intersect(r, NewRectangle2D(2, 2, 4, 4), r)
	if want := NewRectangle2D(2, 2, 4, 4); *r != *want {
		t.Errorf("in-place Intersect = %v, want %v", r, want)
	}
}

func TestRectangle2DUnionAndAdd(t *testing.T) {
	dest := &Rectangle2D{}
	Union(NewRectangle2D(0, 0, 1, 1), NewRectangle2D(4, 4, 1, 1), dest)
	if want := NewRectangle2D(0, 0, 5, 5); *dest != *want {
		t.Errorf("Union = %v, want %v", dest, want)
	}

	r := NewRectangle2D(0, 0, 1, 1)
	r.Add(-2, 3)
	if want := NewRectangle2D(-2, 0, 3, 3); *r != *want {
		t.Errorf("Add = %v, want %v", r, want)
	}
}

// TestRectangle2DBounds pins the rounding: outward on every side, and a
// rectangle with a negative extent collapsing to the origin.
func TestRectangle2DBounds(t *testing.T) {
	got := NewRectangle2D(1.2, 2.7, 3.1, 4.4).Bounds()
	if want := (Rectangle{X: 1, Y: 2, Width: 4, Height: 6}); *got != want {
		t.Errorf("Bounds = %v, want %v", got, want)
	}

	// A zero extent keeps its position; only a negative one loses it.
	got = NewRectangle2D(5, 6, 0, 0).Bounds()
	if want := (Rectangle{X: 5, Y: 6}); *got != want {
		t.Errorf("Bounds of a zero-size rectangle = %v, want %v", got, want)
	}

	got = NewRectangle2D(5, 6, -1, 1).Bounds()
	if want := (Rectangle{}); *got != want {
		t.Errorf("Bounds of a negative rectangle = %v, want %v", got, want)
	}
}

func TestRectangle2DPathIterator(t *testing.T) {
	assertSegments(t, collect(t, NewRectangle2D(1, 2, 3, 4), nil), [][]float64{
		{SegMoveTo, 1, 2},
		{SegLineTo, 4, 2},
		{SegLineTo, 4, 6},
		{SegLineTo, 1, 6},
		{SegClose},
	})

	// An empty rectangle walks no segments at all.
	if got := collect(t, NewRectangle2D(1, 2, 0, 4), nil); len(got) != 0 {
		t.Errorf("an empty rectangle walked %v", got)
	}
}
