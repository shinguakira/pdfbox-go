package text

import "testing"

// The cost of recording a position is the cost of shifting whatever comes after
// it, so the order the positions arrive in is the whole question. Ascending is
// the easy case -- every one lands at the end. Descending is the hard one: every
// new position sorts before all of the others.
//
// Both orders occur in real documents. A page drawn right to left, or a
// generator that emits its glyphs bottom-up, produces the second.
//
// 65,539 is the number veraPDF's clause 6.1.12 files put on a single page, which
// is where the quadratic version of this was first noticed. With a single sorted
// slice the two orders measured 6ms and 3,620ms; they should now be within a
// small factor of each other, and a run where they are not means the blocks have
// stopped bounding the shift.
func benchmarkAdd(b *testing.B, n int, descending bool) {
	b.ReportAllocs()
	for range b.N {
		var positions characterPositions
		for i := 0; i < n; i++ {
			x := float32(i)
			if descending {
				x = float32(n - i)
			}
			positions.add(x, x)
		}
	}
}

func BenchmarkCharacterPositionsAscending65k(b *testing.B)  { benchmarkAdd(b, 65539, false) }
func BenchmarkCharacterPositionsDescending65k(b *testing.B) { benchmarkAdd(b, 65539, true) }
