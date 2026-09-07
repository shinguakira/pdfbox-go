package font

// The rounding the embedders write every width and vertical metric through.
//
// Java rounds half **up** -- towards positive infinity -- so a negative value
// on a half boundary goes the other way from Go's math.Round, which rounds half
// away from zero. Every /W and /W2 entry of a vertical font is a negated
// metric, so the difference is not a curiosity there: -62.5 is Java's -62 and
// Go's -63.

import (
	"math"
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

// TestSubsetTagMatchesJava is getTag, and JAVA-BUGS 74 with it.
//
// The tag becomes part of the font's name in the file, so it has to be Java's
// byte for byte. Both wanted values were printed by the running Java, calling
// getTag on a PDCIDFontType2Embedder built over LiberationSans.
func TestSubsetTagMatchesJava(t *testing.T) {
	// hashCode is (1^2) + (3^4) = 10, and Java answers AAAAAL+.
	ordinary := map[int]int{1: 2, 3: 4}
	if got := subsetTag(ordinary); got != "AAAAAL+" {
		t.Errorf("subsetTag(%v) = %q, want Java's %q", ordinary, got, "AAAAAL+")
	}

	// JAVA-BUGS 74. One entry whose key ^ value is 0x80000000, so the map's
	// hashCode is exactly Integer.MIN_VALUE -- the one input Math.abs(int)
	// cannot fix. Java then indexes BASE25 with -23 and raises
	// StringIndexOutOfBoundsException: String index out of range: -23.
	//
	// Go's % keeps the sign of the dividend as Java's does, so the port reaches
	// the same index and panics, which is what an unchecked exception is here.
	defer func() {
		if recover() == nil {
			t.Error("subsetTag of a map hashing to MinInt32 returned; Java raises " +
				"StringIndexOutOfBoundsException with index -23")
		}
	}()
	pathological := map[int]int{0: math.MinInt32}
	var hash int32
	for gid, cid := range pathological {
		hash += int32(gid) ^ int32(cid)
	}
	if hash != math.MinInt32 {
		t.Fatalf("the map hashes to %d, want MinInt32; the case proves nothing", hash)
	}
	t.Errorf("subsetTag returned %q; Java raises", subsetTag(pathological))
}
