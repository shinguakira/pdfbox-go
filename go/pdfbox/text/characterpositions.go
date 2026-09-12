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
// Both are ordered, so both the lookup and the insertion are logarithmic and
// only the entries inside the range are visited. The port first used Go maps
// and walked every key, which answers the same question and is quadratic in the
// glyphs on the page: veraPDF's clause 6.1.12 files put 65,539 characters on one
// page on purpose, and that took 22 seconds where PDFBox takes a fraction of
// one.
//
// A single sorted slice fixes the lookup and not the insertion. A glyph that
// sorts before everything already recorded shifts the whole slice, so a page
// drawn right to left is quadratic again -- measured at 3.6 seconds against 6
// milliseconds for the same 65,539 positions in ascending order, which is the
// same cliff wearing different clothes.
//
// So the positions live in blocks: an ordered list of sorted runs, each capped.
// Finding the block is a binary search over their last entries, finding the
// position within it is another, and the shift an insertion costs is bounded by
// the block rather than by the page. Both orders then cost the same.
//
// The two levels Java has are flattened into one. Its question -- is there an x
// in range whose set holds a y in range -- is the same question as: is there a
// recorded (x, y) with both in range. The bounds are Java's and they are not
// symmetric: subMap and subSet are inclusive of the low end and exclusive of the
// high.
type characterPositions struct {
	blocks [][]position
}

// position is one place a character has been drawn.
type position struct {
	x, y float32
}

// before reports whether a sorts before b, the order the blocks are kept in.
func (a position) before(b position) bool {
	if a.x != b.x {
		return a.x < b.x
	}
	return a.y < b.y
}

// positionsPerBlock caps the shift an insertion costs. At 512 a block is four
// kilobytes, which is small enough to move without noticing and large enough
// that the list of blocks stays short.
const positionsPerBlock = 512

// anyWithin reports whether a position has already been recorded inside the
// tolerance of (x, y). This is Java's two nested subrange walks.
func (p *characterPositions) anyWithin(x, y, tolerance float32) bool {
	lowX, highX := x-tolerance, x+tolerance
	lowY, highY := y-tolerance, y+tolerance

	// The first block that can hold an x at or after the low bound, then every
	// block after it until the entries run past the high bound.
	for b := p.blockFor(position{x: lowX}); b < len(p.blocks); b++ {
		block := p.blocks[b]
		for i := lowerBound(block, position{x: lowX}); i < len(block); i++ {
			if block[i].x >= highX {
				return false
			}
			if block[i].y >= lowY && block[i].y < highY {
				return true
			}
		}
	}
	return false
}

// add records that the character was drawn at (x, y). A position already there
// is left alone, which is what putting into a TreeSet does.
func (p *characterPositions) add(x, y float32) {
	at := position{x: x, y: y}
	if len(p.blocks) == 0 {
		p.blocks = [][]position{{at}}
		return
	}

	b := p.blockFor(at)
	if b == len(p.blocks) {
		// past every block: it belongs at the end of the last one
		b = len(p.blocks) - 1
	}

	block := p.blocks[b]
	i := lowerBound(block, at)
	if i < len(block) && block[i] == at {
		return
	}

	block = append(block, position{})
	copy(block[i+1:], block[i:])
	block[i] = at
	p.blocks[b] = block

	if len(block) > positionsPerBlock {
		p.split(b)
	}
}

// blockFor returns the index of the first block that may hold a position at or
// after the given one, which is the first whose last entry is not before it.
func (p *characterPositions) blockFor(at position) int {
	return sort.Search(len(p.blocks), func(i int) bool {
		block := p.blocks[i]
		return !block[len(block)-1].before(at)
	})
}

// lowerBound returns the first index in a sorted block at or after the given
// position.
func lowerBound(block []position, at position) int {
	return sort.Search(len(block), func(i int) bool { return !block[i].before(at) })
}

// split halves a block that has grown past the cap, so the shift an insertion
// costs stays bounded.
func (p *characterPositions) split(b int) {
	block := p.blocks[b]
	half := len(block) / 2

	// The two halves are copied out rather than resliced, so neither keeps the
	// other's spare capacity and a later append cannot write into its sibling.
	left := make([]position, half, positionsPerBlock+1)
	left = append(left[:0], block[:half]...)
	right := make([]position, len(block)-half, positionsPerBlock+1)
	right = append(right[:0], block[half:]...)

	p.blocks = append(p.blocks, nil)
	copy(p.blocks[b+2:], p.blocks[b+1:])
	p.blocks[b] = left
	p.blocks[b+1] = right
}

// count returns how many positions are recorded. Only the tests ask.
func (p *characterPositions) count() int {
	total := 0
	for _, block := range p.blocks {
		total += len(block)
	}
	return total
}
