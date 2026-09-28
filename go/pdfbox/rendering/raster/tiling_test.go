package raster

// The parts of TilingPaint the pattern page cannot reach.
//
// `page_test.go` renders four patterns against PDFBox and the coloured one is
// exact, which covers the anchor rectangle, the tile raster, the repeat and the
// pattern matrix. What it does not reach is the arithmetic at the edges: a
// method whose name lies, and the clamp PDFBOX-3653 added for a pattern that
// asks for a surface of billions of pixels.

import (
	"math"
	"testing"
)

// What TilingPaint.ceiling answers is no longer what tilingCeiling answers:
// JAVA-BUGS.md 85 is fixed, and javabug85_test.go holds both columns -- the
// corrected value and, beside each, what the Java gives. The test that pinned
// the Java's answers alone was here, and is there now.

// TestSignumIsJavas is Math.signum, which keeps the sign of a zero.
func TestSignumIsJavas(t *testing.T) {
	for _, c := range []struct {
		v, want float32
	}{
		{3, 1},
		{-3, -1},
		{0, 0},
		{float32(math.Copysign(0, -1)), float32(math.Copysign(0, -1))},
	} {
		got := signum(c.v)
		if got != c.want || math.Signbit(float64(got)) != math.Signbit(float64(c.want)) {
			t.Errorf("signum(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}

// TestIsPositiveZeroIsFloatCompare is `Float.compare(v, 0) == 0`, which orders
// -0.0 below +0.0 and so answers zero for +0.0 alone.
func TestIsPositiveZeroIsFloatCompare(t *testing.T) {
	for _, c := range []struct {
		v    float32
		want bool
	}{
		{0, true},
		{float32(math.Copysign(0, -1)), false},
		{1, false},
		{-1, false},
		{float32(math.NaN()), false},
	} {
		if got := isPositiveZero(c.v); got != c.want {
			t.Errorf("isPositiveZero(%v) = %v, want %v", c.v, got, c.want)
		}
	}
}
