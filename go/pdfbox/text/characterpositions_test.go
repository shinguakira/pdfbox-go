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
// every position is found again however the positions arrived, and a repeat is
// not recorded twice.
func TestCharacterPositionsStaySorted(t *testing.T) {
	added := [][2]float32{
		{3, 30}, {1, 10}, {2, 20}, {1, 5}, {1, 10}, {2, 20},
	}

	var positions characterPositions
	for _, xy := range added {
		positions.add(xy[0], xy[1])
	}

	// a tolerance small enough that only an exact position answers
	const exact = 0.001
	for _, xy := range added {
		if !positions.anyWithin(xy[0]+exact/2, xy[1]+exact/2, exact) {
			t.Errorf("(%v, %v) was added and is not found", xy[0], xy[1])
		}
	}
	if positions.anyWithin(1+exact/2, 20+exact/2, exact) {
		t.Error("(1, 20) was never added and is found")
	}

	if got := positions.count(); got != 4 {
		t.Errorf("holding %d positions, want 4 -- the two repeats must not be recorded twice", got)
	}
}

// TestCharacterPositionsAgreeWithABruteForceScan checks the blocked structure
// against the obvious implementation over enough positions to fill several
// blocks and force the splits.
//
// The structure earns its complexity on speed alone -- it answers exactly what
// a linear scan of the same set answers -- so a linear scan is what it is held
// to. The positions are generated rather than listed because the interesting
// cases are the ones at a block boundary, and which positions those are depends
// on the split.
func TestCharacterPositionsAgreeWithABruteForceScan(t *testing.T) {
	// deterministic, and deliberately not in order: insertion order is what
	// decides where the blocks break
	const count = 4000
	xs := make([]float32, 0, count)
	ys := make([]float32, 0, count)
	seed := uint32(12345)
	next := func() float32 {
		seed = seed*1664525 + 1013904223
		return float32(seed%2000) / 4
	}

	var positions characterPositions
	var flat []position
	for i := 0; i < count; i++ {
		x, y := next(), next()
		xs, ys = append(xs, x), append(ys, y)
		positions.add(x, y)

		known := false
		for _, p := range flat {
			if p.x == x && p.y == y {
				known = true
				break
			}
		}
		if !known {
			flat = append(flat, position{x: x, y: y})
		}
	}

	if got := positions.count(); got != len(flat) {
		t.Fatalf("holding %d positions, want %d -- duplicates are not being dropped", got, len(flat))
	}

	bruteForce := func(x, y, tolerance float32) bool {
		for _, p := range flat {
			if p.x >= x-tolerance && p.x < x+tolerance &&
				p.y >= y-tolerance && p.y < y+tolerance {
				return true
			}
		}
		return false
	}

	// every tolerance from "only this exact spot" to "most of the page"
	for _, tolerance := range []float32{0.001, 0.25, 1, 10} {
		for i := 0; i < count; i++ {
			// the position itself, and a point beside it
			for _, probe := range [][2]float32{
				{xs[i], ys[i]},
				{xs[i] + 0.3, ys[i] - 0.3},
			} {
				want := bruteForce(probe[0], probe[1], tolerance)
				if got := positions.anyWithin(probe[0], probe[1], tolerance); got != want {
					t.Fatalf("anyWithin(%v, %v, %v) = %v, want %v",
						probe[0], probe[1], tolerance, got, want)
				}
			}
		}
	}
}
