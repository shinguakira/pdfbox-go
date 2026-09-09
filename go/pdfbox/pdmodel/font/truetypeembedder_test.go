package font

// The rounding the embedders write every width and vertical metric through.
//
// Java rounds half **up** -- towards positive infinity -- so a negative value
// on a half boundary goes the other way from Go's math.Round, which rounds half
// away from zero. Every /W and /W2 entry of a vertical font is a negated
// metric, so the difference is not a curiosity there: -62.5 is Java's -62 and
// Go's -63.

import (
	"testing"
)

// TestJavaRoundHalfUp checks the two helpers against Math.round.
//
// The wanted values are what `jshell` prints for Math.round of each input, not
// what the Go answers.
func TestJavaRoundHalfUp(t *testing.T) {
	floats := []struct {
		in   float32
		want int
	}{
		{-62.6, -63},
		{-62.5, -62},
		{-62.4, -62},
		{-2.5, -2},
		{-1.5, -1},
		{-0.5, 0},
		{0.5, 1},
		{1.5, 2},
		{2.5, 3},
		{62.5, 63},
	}
	for _, c := range floats {
		if got := javaRound(c.in); got != c.want {
			t.Errorf("javaRound(%v) = %d, want %d", c.in, got, c.want)
		}
	}

	doubles := []struct {
		in   float64
		want int64
	}{
		{-62.5, -62},
		{-2.5, -2},
		{-0.5, 0},
		{0.5, 1},
		{2.5, 3},
	}
	for _, c := range doubles {
		if got := javaRoundLong(c.in); got != c.want {
			t.Errorf("javaRoundLong(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}
