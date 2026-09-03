package geom

import (
	"math"
	"testing"
)

// Written from the java.awt.geom.Path2D and Rectangle2D specifications. There
// is no PDFBox test for either — they belong to the JDK — so per
// migration/conventions/tdd.md the expected values come from the documented
// behaviour, not from this port.

// collect walks a path and returns one entry per segment, as the type followed
// by its points.
func collect(t *testing.T, s Shape, at *AffineTransform) [][]float64 {
	t.Helper()
	var out [][]float64
	coords := make([]float64, 6)
	for pi := s.PathIterator(at); !pi.IsDone(); pi.Next() {
		segment := pi.CurrentSegment(coords)
		entry := []float64{float64(segment)}
		entry = append(entry, coords[:segmentPointCount[segment]*2]...)
		out = append(out, entry)
	}
	return out
}

func assertSegments(t *testing.T, got, want [][]float64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d segments %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if len(got[i]) != len(want[i]) {
			t.Fatalf("segment %d = %v, want %v", i, got[i], want[i])
		}
		for j := range want[i] {
			if math.Abs(got[i][j]-want[i][j]) > 1e-9 {
				t.Fatalf("segment %d = %v, want %v", i, got[i], want[i])
			}
		}
	}
}

func TestPathSegments(t *testing.T) {
	p := NewPathDouble()
	p.MoveTo(1, 2)
	p.LineTo(3, 4)
	p.QuadTo(5, 6, 7, 8)
	p.CurveTo(9, 10, 11, 12, 13, 14)
	p.ClosePath()

	assertSegments(t, collect(t, p, nil), [][]float64{
		{SegMoveTo, 1, 2},
		{SegLineTo, 3, 4},
		{SegQuadTo, 5, 6, 7, 8},
		{SegCubicTo, 9, 10, 11, 12, 13, 14},
		{SegClose},
	})
}

// TestPathFloatRoundsCoordinates pins the difference between the two concrete
// path types: a float path stores what a float32 can hold and nothing finer.
func TestPathFloatRoundsCoordinates(t *testing.T) {
	f := NewPathFloat()
	f.MoveTo(0.1, 0.2)
	assertSegments(t, collect(t, f, nil), [][]float64{
		{SegMoveTo, float64(float32(0.1)), float64(float32(0.2))},
	})

	d := NewPathDouble()
	d.MoveTo(0.1, 0.2)
	assertSegments(t, collect(t, d, nil), [][]float64{{SegMoveTo, 0.1, 0.2}})
}

// TestPathRepeatedMoveTo pins that a move nothing was drawn from is replaced
// rather than kept, so the path never holds two moves in a row.
func TestPathRepeatedMoveTo(t *testing.T) {
	p := NewPathDouble()
	p.MoveTo(1, 1)
	p.MoveTo(2, 2)
	p.LineTo(3, 3)

	assertSegments(t, collect(t, p, nil), [][]float64{
		{SegMoveTo, 2, 2},
		{SegLineTo, 3, 3},
	})
}

// TestPathRepeatedClosePath pins that closing an already closed subpath does
// nothing.
func TestPathRepeatedClosePath(t *testing.T) {
	p := NewPathDouble()
	p.MoveTo(1, 1)
	p.LineTo(2, 2)
	p.ClosePath()
	p.ClosePath()

	assertSegments(t, collect(t, p, nil), [][]float64{
		{SegMoveTo, 1, 1},
		{SegLineTo, 2, 2},
		{SegClose},
	})
}

// TestPathNeedsInitialMoveTo pins that drawing before any subpath has been
// begun is a caller's mistake, which Java reports with the unchecked
// IllegalPathStateException.
func TestPathNeedsInitialMoveTo(t *testing.T) {
	cases := map[string]func(*Path2D){
		"LineTo":    func(p *Path2D) { p.LineTo(1, 1) },
		"QuadTo":    func(p *Path2D) { p.QuadTo(1, 1, 2, 2) },
		"CurveTo":   func(p *Path2D) { p.CurveTo(1, 1, 2, 2, 3, 3) },
		"ClosePath": func(p *Path2D) { p.ClosePath() },
	}
	for name, draw := range cases {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Error("no panic on an empty path, want one")
				}
			}()
			draw(NewPathDouble())
		})
	}
}

func TestPathCurrentPoint(t *testing.T) {
	p := NewPathDouble()
	if p.CurrentPoint() != nil {
		t.Error("an empty path has a current point")
	}

	p.MoveTo(1, 2)
	if got := p.CurrentPoint(); got.X() != 1 || got.Y() != 2 {
		t.Errorf("after MoveTo: (%v, %v), want (1, 2)", got.X(), got.Y())
	}

	p.LineTo(3, 4)
	if got := p.CurrentPoint(); got.X() != 3 || got.Y() != 4 {
		t.Errorf("after LineTo: (%v, %v), want (3, 4)", got.X(), got.Y())
	}

	// After a close the current point is where the closed subpath began.
	p.LineTo(5, 6)
	p.ClosePath()
	if got := p.CurrentPoint(); got.X() != 1 || got.Y() != 2 {
		t.Errorf("after ClosePath: (%v, %v), want (1, 2)", got.X(), got.Y())
	}
}

func TestPathReset(t *testing.T) {
	p := NewPathDoubleRule(WindEvenOdd)
	p.MoveTo(1, 2)
	p.LineTo(3, 4)
	p.Reset()

	if got := collect(t, p, nil); len(got) != 0 {
		t.Errorf("Reset left %v", got)
	}
	if p.WindingRule() != WindEvenOdd {
		t.Error("Reset changed the winding rule")
	}
}

func TestPathWindingRule(t *testing.T) {
	if got := NewPathFloat().WindingRule(); got != WindNonZero {
		t.Errorf("default winding rule = %d, want WindNonZero", got)
	}
	p := NewPathFloatRule(WindEvenOdd)
	if got := p.WindingRule(); got != WindEvenOdd {
		t.Errorf("winding rule = %d, want WindEvenOdd", got)
	}
	p.SetWindingRule(WindNonZero)
	if got := p.PathIterator(nil).WindingRule(); got != WindNonZero {
		t.Errorf("the iterator reports %d, want WindNonZero", got)
	}
}

func TestPathTransform(t *testing.T) {
	p := NewPathDouble()
	p.MoveTo(1, 2)
	p.LineTo(3, 4)
	p.Transform(TranslateInstance(10, 20))

	assertSegments(t, collect(t, p, nil), [][]float64{
		{SegMoveTo, 11, 22},
		{SegLineTo, 13, 24},
	})
}

// TestPathIteratorTransform pins that the iterator applies a transform without
// touching the path it walks.
func TestPathIteratorTransform(t *testing.T) {
	p := NewPathDouble()
	p.MoveTo(1, 2)
	p.ClosePath()

	assertSegments(t, collect(t, p, TranslateInstance(10, 20)), [][]float64{
		{SegMoveTo, 11, 22},
		{SegClose},
	})
	assertSegments(t, collect(t, p, nil), [][]float64{
		{SegMoveTo, 1, 2},
		{SegClose},
	})
}

func TestPathAppend(t *testing.T) {
	first := NewPathDouble()
	first.MoveTo(0, 0)
	first.LineTo(1, 1)

	second := NewPathDouble()
	second.MoveTo(5, 5)
	second.LineTo(6, 6)

	joined := first.Clone()
	joined.Append(second, true)
	// connect turns the appended move into a line.
	assertSegments(t, collect(t, joined, nil), [][]float64{
		{SegMoveTo, 0, 0},
		{SegLineTo, 1, 1},
		{SegLineTo, 5, 5},
		{SegLineTo, 6, 6},
	})

	separate := first.Clone()
	separate.Append(second, false)
	assertSegments(t, collect(t, separate, nil), [][]float64{
		{SegMoveTo, 0, 0},
		{SegLineTo, 1, 1},
		{SegMoveTo, 5, 5},
		{SegLineTo, 6, 6},
	})
}

func TestPathCloneIsIndependent(t *testing.T) {
	p := NewPathDoubleRule(WindEvenOdd)
	p.MoveTo(1, 2)

	clone := p.Clone()
	clone.LineTo(3, 4)

	assertSegments(t, collect(t, p, nil), [][]float64{{SegMoveTo, 1, 2}})
	if clone.WindingRule() != WindEvenOdd {
		t.Error("the clone lost the winding rule")
	}
}

func TestPathFromShape(t *testing.T) {
	rect := NewRectangle2D(1, 2, 3, 4)
	p := NewPathDoubleShape(rect)

	assertSegments(t, collect(t, p, nil), [][]float64{
		{SegMoveTo, 1, 2},
		{SegLineTo, 4, 2},
		{SegLineTo, 4, 6},
		{SegLineTo, 1, 6},
		{SegClose},
	})
}

// TestPathBounds2DUsesControlPoints pins that the bounds enclose the points the
// path names rather than the curves they describe, so a curve can be given a
// rectangle larger than it needs. This is what Java does.
func TestPathBounds2DUsesControlPoints(t *testing.T) {
	p := NewPathDouble()
	p.MoveTo(0, 0)
	p.CurveTo(0, 100, 10, 100, 10, 0)

	got := p.Bounds2D()
	want := NewRectangle2D(0, 0, 10, 100)
	if *got != *want {
		t.Errorf("Bounds2D = %v, want %v", got, want)
	}
}

func TestPathBounds2DEmpty(t *testing.T) {
	if got := (NewPathDouble().Bounds2D()); *got != (Rectangle2D{}) {
		t.Errorf("Bounds2D of an empty path = %v, want the zero rectangle", got)
	}
}
