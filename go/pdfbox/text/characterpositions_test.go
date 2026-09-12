package text

import "testing"

// TestCharacterPositionsRange pins the bounds of the duplicate-suppression
// lookup, which is the part that must not move while the structure underneath
// it does.
//
// Java asks a TreeMap and a TreeSet for a range, and both of those are
// inclusive of the low end and exclusive of the high:
//
//	sameTextCharacters.subMap(textX - tolerance, textX + tolerance)
//	xMatch.subSet(textY - tolerance, textY + tolerance)
//
// This is in the internal test package because the structure is unexported.
func TestCharacterPositionsRange(t *testing.T) {
	var positions characterPositions
	positions.add(100, 200)

	cases := []struct {
		name    string
		x, y    float32
		tol     float32
		matches bool
	}{
		{"the same place", 100, 200, 1, true},
		{"inside on both axes", 100.5, 200.5, 1, true},
		{"x on the low bound is inside", 101, 200, 1, true},
		{"x on the high bound is outside", 99, 200, 1, false},
		{"y on the low bound is inside", 100, 201, 1, true},
		{"y on the high bound is outside", 100, 199, 1, false},
		{"too far in x", 102, 200, 1, false},
		{"too far in y", 100, 202, 1, false},
		{"zero tolerance matches nothing", 100, 200, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := positions.anyWithin(c.x, c.y, c.tol); got != c.matches {
				t.Errorf("anyWithin(%v, %v, %v) = %v, want %v", c.x, c.y, c.tol, got, c.matches)
			}
		})
	}
}

// TestCharacterPositionsStaySorted pins the invariant the search depends on:
// the columns are ordered and hold no duplicates however they are added.
func TestCharacterPositionsStaySorted(t *testing.T) {
	var positions characterPositions
	for _, xy := range [][2]float32{
		{3, 30}, {1, 10}, {2, 20}, {1, 5}, {1, 10}, {2, 20},
	} {
		positions.add(xy[0], xy[1])
	}

	wantXs := []float32{1, 2, 3}
	if len(positions.xs) != len(wantXs) {
		t.Fatalf("xs = %v, want %v", positions.xs, wantXs)
	}
	for i, want := range wantXs {
		if positions.xs[i] != want {
			t.Fatalf("xs = %v, want %v", positions.xs, wantXs)
		}
	}
	if got := positions.ys[0]; len(got) != 2 || got[0] != 5 || got[1] != 10 {
		t.Errorf("the column at x=1 is %v, want [5 10]", got)
	}
}
