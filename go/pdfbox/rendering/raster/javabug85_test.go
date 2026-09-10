package raster

// JAVA-BUGS 85: `TilingPaint.ceiling` rounds up at the fifth decimal place and
// then truncates to an int, so every value below the next whole number stays
// where it was -- `ceiling(3.9)` is 3. `getImage` sizes the tile's raster with
// it, so a tile that measures 3.9 device pixels across is rasterized 3 pixels
// wide and then stretched over 3.9 by the paint that repeats it.
//
// **The expected values here are not the Java's.** They come from the method's
// own javadoc, which is the only statement of what it was for:
//
//	Returns the closest integer which is larger than the given number.
//	Uses BigDecimal to avoid floating point error which would cause gaps in
//	the tiling.
//
// Two sentences, and the implementation satisfies the second and not the
// first. A ceiling with a tolerance satisfies both: round up at the fifth
// decimal place, as Java does, and then take the ceiling rather than the
// truncation. 3.9 becomes 4, and 2.999999999 stays 3 rather than becoming 4 on
// a float error, which is what the second sentence is guarding.

import "testing"

// TestTilingCeilingRoundsUp is the correct behaviour, which the Java does not
// have.
func TestTilingCeilingRoundsUp(t *testing.T) {
	for _, c := range []struct {
		num  float64
		want int
		// java is what TilingPaint.ceiling answers, measured on JDK 17 and
		// kept beside the corrected value so that both are visible. See
		// JAVA-BUGS.md 85.
		java int
	}{
		{3.0, 3, 3},
		{3.2, 4, 3},
		{3.5, 4, 3},
		{3.9, 4, 3},
		{0.5, 1, 0},
		// The tolerance, which is the half the Java gets right: a width that
		// should have been whole and came out a hair under does not buy a
		// whole extra pixel.
		{2.999999999, 3, 3},
		{3.000001, 4, 3},
	} {
		if got := tilingCeiling(c.num); got != c.want {
			t.Errorf("tilingCeiling(%v) = %d, want %d (the Java answers %d)",
				c.num, got, c.want, c.java)
		}
	}
}
