package text

import "sort"

// characterPositions records where one character has already been drawn, so
// that PDFTextStripper can suppress a second drawing of it at the same place.
//
// Java keeps `TreeMap<Float, TreeSet<Float>>` per character and asks it for a
// range:
//
//	SortedMap<Float, TreeSet<Float>> xMatches =
//	        sameTextCharacters.subMap(textX - tolerance, textX + tolerance);
//	for (TreeSet<Float> xMatch : xMatches.values())
//	{
//	    SortedSet<Float> yMatches = xMatch.subSet(textY - tolerance, textY + tolerance);
//
// Both are ordered, so both ranges are found by search and only the entries
// inside them are visited. The port first used Go maps and walked every key,
// which answers the same question and is the same code until the page is large:
// one linear scan per glyph is quadratic in the glyphs on the page, and
// veraPDF's clause 6.1.12 files -- the ones written to exceed implementation
// limits -- put 65,539 characters on a single page. That took 22 seconds where
// PDFBox takes well under one, and three documents of the corpus timed out on
// it. See TestCharacterPositionsRange.
//
// So this is the ordered structure, in the shape Go makes cheap: two parallel
// sorted slices rather than a tree. The bounds are Java's, and they are not
// symmetric -- subMap and subSet are inclusive of the low end and exclusive of
// the high.
type characterPositions struct {
	// xs is sorted and holds no duplicates; ys[i] belongs to xs[i] and is
	// sorted and holds no duplicates too.
	xs []float32
	ys [][]float32
}

// lowerBound returns the first index of values at or after v.
func lowerBound(values []float32, v float32) int {
	return sort.Search(len(values), func(i int) bool { return values[i] >= v })
}

// anyWithin reports whether a position has already been recorded inside the
// tolerance of (x, y). This is Java's two nested subrange walks.
func (p *characterPositions) anyWithin(x, y, tolerance float32) bool {
	lowX, highX := x-tolerance, x+tolerance
	for i := lowerBound(p.xs, lowX); i < len(p.xs) && p.xs[i] < highX; i++ {
		lowY, highY := y-tolerance, y+tolerance
		if j := lowerBound(p.ys[i], lowY); j < len(p.ys[i]) && p.ys[i][j] < highY {
			return true
		}
	}
	return false
}

// add records that the character was drawn at (x, y). A position already there
// is left alone, which is what putting into a TreeSet does.
func (p *characterPositions) add(x, y float32) {
	i := lowerBound(p.xs, x)
	if i == len(p.xs) || p.xs[i] != x {
		p.xs = append(p.xs, 0)
		copy(p.xs[i+1:], p.xs[i:])
		p.xs[i] = x

		p.ys = append(p.ys, nil)
		copy(p.ys[i+1:], p.ys[i:])
		p.ys[i] = nil
	}

	column := p.ys[i]
	j := lowerBound(column, y)
	if j < len(column) && column[j] == y {
		return
	}
	column = append(column, 0)
	copy(column[j+1:], column[j:])
	column[j] = y
	p.ys[i] = column
}
