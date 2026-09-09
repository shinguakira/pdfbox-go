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

// TestTilingCeilingIsAFloor is JAVA-BUGS.md 85.
//
// The values are a JDK 17 run of TilingPaint.ceiling's own body:
//
//	BigDecimal decimal = BigDecimal.valueOf(num);
//	decimal = decimal.setScale(5, RoundingMode.CEILING);
//	return decimal.intValue();
//
// It rounds up at the fifth decimal place and then truncates to an int, so
// every value below the next whole number stays where it was. Its javadoc says
// "the closest integer which is larger than the given number", and that is not
// what it does.
func TestTilingCeilingIsAFloor(t *testing.T) {
	for _, c := range []struct {
		num  float64
		want int
	}{
		{3.0, 3},
		{3.2, 3},
		{3.5, 3},
		{3.9, 3},
		{2.999999999, 3},
		{3.000001, 3},
		{0.5, 0},
		{-3.2, -3},
	} {
		if got := tilingCeiling(c.num); got != c.want {
			t.Errorf("tilingCeiling(%v) = %d, want %d", c.num, got, c.want)
		}
	}
}

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
