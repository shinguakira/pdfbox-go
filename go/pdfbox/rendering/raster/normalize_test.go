package raster

// The two coordinate roundings, and the shape of what they are applied to.
//
// What normalize.go does end to end is measured by TestAgainstJava2DNormalized,
// which draws all seventeen shapes under VALUE_STROKE_NORMALIZE and holds them
// to a JDK that was told to do the same. This is the arithmetic underneath,
// asserted on its own so that a wrong rounding says so in one line rather than
// as a pixel count.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// TestNearestPixelCenterIsFloorPlusAHalf is
// `FloatMath.floor_f(coord) + 0.5f`, which anti-aliased strokes go through.
func TestNearestPixelCenterIsFloorPlusAHalf(t *testing.T) {
	for _, c := range []struct{ coord, want float32 }{
		{0, 0.5},
		{20, 20.5},
		{20.4, 20.5},
		{20.5, 20.5},
		{20.6, 20.5},
		{-1, -0.5},
		{-0.5, -0.5},
	} {
		if got := nearestPixelCenter(c.coord); got != c.want {
			t.Errorf("nearestPixelCenter(%v) = %v, want %v", c.coord, got, c.want)
		}
	}
}

// TestNearestPixelQuarterIsJavas is
// `FloatMath.floor_f(coord + 0.25f) + 0.25f`, which strokes go through with
// anti-aliasing off.
func TestNearestPixelQuarterIsJavas(t *testing.T) {
	for _, c := range []struct{ coord, want float32 }{
		{0, 0.25},
		{10, 10.25},
		{9.8, 10.25},
		{9.7, 9.25},
		{-1, -0.75},
	} {
		if got := nearestPixelQuarter(c.coord); got != c.want {
			t.Errorf("nearestPixelQuarter(%v) = %v, want %v", c.coord, got, c.want)
		}
	}
}

// TestNormalizationOffLeavesThePathAlone is VALUE_STROKE_PURE.
func TestNormalizationOffLeavesThePathAlone(t *testing.T) {
	n := newNormalizer(false, true)
	coords := []float64{20, 20, 0, 0, 0, 0}
	n.segment(geom.SegLineTo, coords)
	if coords[0] != 20 || coords[1] != 20 {
		t.Errorf("the point moved to (%v,%v) with normalization off", coords[0], coords[1])
	}
}

// TestACurvesControlPointsMoveWithItsEnds is the second switch: a cubic's first
// control point takes the adjustment made to the point before it and its second
// takes the one made to its own endpoint, so the curve keeps its shape.
//
//	case PathIterator.SEG_CUBICTO:
//	    coords[0] += curx_adjust;
//	    coords[1] += cury_adjust;
//	    coords[2] += x_adjust;
//	    coords[3] += y_adjust;
func TestACurvesControlPointsMoveWithItsEnds(t *testing.T) {
	n := newNormalizer(true, true)

	// A moveTo to (4,26), which becomes (4.5,26.5): both adjustments are +0.5.
	start := []float64{4, 26, 0, 0, 0, 0}
	n.segment(geom.SegMoveTo, start)
	if start[0] != 4.5 || start[1] != 26.5 {
		t.Fatalf("the moveTo went to (%v,%v), want (4.5,26.5)", start[0], start[1])
	}

	// Then a cubic to (28,26), whose endpoint moves by +0.5 as well.
	curve := []float64{4, 4, 28, 4, 28, 26}
	n.segment(geom.SegCubicTo, curve)

	want := []float64{4.5, 4.5, 28.5, 4.5, 28.5, 26.5}
	for i, w := range want {
		if curve[i] != w {
			t.Errorf("the cubic came out %v, want %v", curve, want)
			break
		}
	}
}

// TestClosePathRestoresTheMoveTosAdjustment is the SEG_CLOSE arm, which puts
// the adjustment back to the one the subpath began with so that whatever
// follows is measured from where the subpath started.
func TestClosePathRestoresTheMoveTosAdjustment(t *testing.T) {
	n := newNormalizer(true, true)

	n.segment(geom.SegMoveTo, []float64{4, 4, 0, 0, 0, 0})
	movX, movY := n.curXAdjust, n.curYAdjust

	// A line whose endpoint is already on a pixel centre, so its adjustment is
	// zero and differs from the moveTo's.
	n.segment(geom.SegLineTo, []float64{8.5, 8.5, 0, 0, 0, 0})
	if n.curXAdjust == movX {
		t.Fatal("the line left the same adjustment as the moveTo, so the test proves nothing")
	}

	n.segment(geom.SegClose, []float64{0, 0, 0, 0, 0, 0})
	if n.curXAdjust != movX || n.curYAdjust != movY {
		t.Errorf("closePath left (%v,%v), want the moveTo's (%v,%v)",
			n.curXAdjust, n.curYAdjust, movX, movY)
	}
}
