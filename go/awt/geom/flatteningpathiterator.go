package geom

import "math"

// flatteningLimit is the deepest a curve is subdivided before it is accepted as
// flat, which is the limit Shape.getPathIterator(at, flatness) uses.
//
// Port of the default recursion limit of java.awt.geom.FlatteningPathIterator.
const flatteningLimit = 10

// FlatteningPathIterator walks another iterator's segments with every curve
// broken into lines that stay within flatness of it.
//
// Port of java.awt.geom.FlatteningPathIterator. The subdivision has to match
// Java's exactly, because the points it answers are written into a content
// stream.
type FlatteningPathIterator struct {
	src        PathIterator
	squareflat float64
	limit      int

	// hold is the working array of coordinates, filled from the end.
	hold       []float64
	curx, cury float64
	movx, movy float64
	// levels records how deep each pending curve has been subdivided.
	levels     []int
	levelIndex int
	holdType   int
	holdEnd    int
	holdIndex  int
	done       bool
}

var _ PathIterator = (*FlatteningPathIterator)(nil)

// NewFlatteningPathIterator returns an iterator over src whose curves are
// broken into lines, with the default recursion limit.
func NewFlatteningPathIterator(src PathIterator, flatness float64) *FlatteningPathIterator {
	return NewFlatteningPathIteratorLimit(src, flatness, flatteningLimit)
}

// NewFlatteningPathIteratorLimit returns one that subdivides at most limit
// times.
//
// Java throws IllegalArgumentException for a negative flatness or limit, which
// is unchecked, so the port panics.
func NewFlatteningPathIteratorLimit(src PathIterator, flatness float64,
	limit int) *FlatteningPathIterator {
	if flatness < 0 {
		panic("negative flatness")
	}
	if limit < 0 {
		panic("negative limit")
	}
	it := &FlatteningPathIterator{
		src:        src,
		squareflat: flatness * flatness,
		limit:      limit,
		hold:       make([]float64, 14),
		levels:     make([]int, limit+1),
	}
	it.next(false)
	return it
}

// Flatness returns the flatness of this iterator.
func (it *FlatteningPathIterator) Flatness() float64 {
	return math.Sqrt(it.squareflat)
}

// RecursionLimit returns the recursion limit of this iterator.
func (it *FlatteningPathIterator) RecursionLimit() int { return it.limit }

// WindingRule returns the winding rule of the underlying path.
func (it *FlatteningPathIterator) WindingRule() int { return it.src.WindingRule() }

// IsDone reports whether the walk is finished.
func (it *FlatteningPathIterator) IsDone() bool { return it.done }

// ensureHoldCapacity grows the working array so that want more values fit
// before holdIndex.
func (it *FlatteningPathIterator) ensureHoldCapacity(want int) {
	if it.holdIndex-want < 0 {
		have := len(it.hold) - it.holdIndex
		newsize := len(it.hold) * 2
		newhold := make([]float64, newsize)
		copy(newhold[newsize-have:], it.hold[it.holdIndex:it.holdIndex+have])
		it.hold = newhold
		it.holdIndex = newsize - have
		it.holdEnd = newsize - 2
	}
}

// Next advances to the next segment.
func (it *FlatteningPathIterator) Next() { it.next(true) }

// next advances, reading another segment from the source when the current one
// is spent.
func (it *FlatteningPathIterator) next(doNext bool) {
	if it.holdIndex >= it.holdEnd {
		if doNext {
			it.src.Next()
		}
		if it.src.IsDone() {
			it.done = true
			return
		}
		it.holdType = it.src.CurrentSegment(it.hold)
		it.levelIndex = 0
		it.levels[0] = 0
	}

	switch it.holdType {
	case SegMoveTo, SegLineTo:
		it.curx = it.hold[0]
		it.cury = it.hold[1]
		if it.holdType == SegMoveTo {
			it.movx = it.curx
			it.movy = it.cury
		}
		it.holdIndex = 0
		it.holdEnd = 0

	case SegClose:
		it.curx = it.movx
		it.cury = it.movy
		it.holdIndex = 0
		it.holdEnd = 0

	case SegQuadTo:
		if it.holdIndex >= it.holdEnd {
			// Move the coordinates to the end of the array.
			it.holdIndex = len(it.hold) - 6
			it.holdEnd = len(it.hold) - 2
			it.hold[it.holdIndex+0] = it.curx
			it.hold[it.holdIndex+1] = it.cury
			it.hold[it.holdIndex+2] = it.hold[0]
			it.hold[it.holdIndex+3] = it.hold[1]
			it.curx = it.hold[2]
			it.hold[it.holdIndex+4] = it.curx
			it.cury = it.hold[3]
			it.hold[it.holdIndex+5] = it.cury
		}
		level := it.levels[it.levelIndex]
		for level < it.limit {
			if quadFlatnessSq(it.hold, it.holdIndex) < it.squareflat {
				break
			}
			it.ensureHoldCapacity(4)
			subdivideQuad(it.hold, it.holdIndex, it.hold, it.holdIndex-4, it.hold, it.holdIndex)
			it.holdIndex -= 4
			level++
			it.levels[it.levelIndex] = level
			it.levelIndex++
			it.levels[it.levelIndex] = level
		}
		// trim off last point from the curve
		it.holdIndex += 4
		it.levelIndex--

	case SegCubicTo:
		if it.holdIndex >= it.holdEnd {
			// Move the coordinates to the end of the array.
			it.holdIndex = len(it.hold) - 8
			it.holdEnd = len(it.hold) - 2
			it.hold[it.holdIndex+0] = it.curx
			it.hold[it.holdIndex+1] = it.cury
			it.hold[it.holdIndex+2] = it.hold[0]
			it.hold[it.holdIndex+3] = it.hold[1]
			it.hold[it.holdIndex+4] = it.hold[2]
			it.hold[it.holdIndex+5] = it.hold[3]
			it.curx = it.hold[4]
			it.hold[it.holdIndex+6] = it.curx
			it.cury = it.hold[5]
			it.hold[it.holdIndex+7] = it.cury
		}
		level := it.levels[it.levelIndex]
		for level < it.limit {
			if cubicFlatnessSq(it.hold, it.holdIndex) < it.squareflat {
				break
			}
			it.ensureHoldCapacity(6)
			subdivideCubic(it.hold, it.holdIndex, it.hold, it.holdIndex-6, it.hold, it.holdIndex)
			it.holdIndex -= 6
			level++
			it.levels[it.levelIndex] = level
			it.levelIndex++
			it.levels[it.levelIndex] = level
		}
		// trim off last point from the curve
		it.holdIndex += 6
		it.levelIndex--
	}
}

// CurrentSegment writes the points of the current segment into coords and
// returns the segment type, which is never a curve.
func (it *FlatteningPathIterator) CurrentSegment(coords []float64) int {
	if it.IsDone() {
		panic("geom: flattening iterator out of bounds")
	}
	segmentType := it.holdType
	if segmentType != SegClose {
		coords[0] = it.hold[it.holdIndex+0]
		coords[1] = it.hold[it.holdIndex+1]
		if segmentType != SegMoveTo {
			segmentType = SegLineTo
		}
	}
	return segmentType
}

// cubicFlatnessSq returns the square of the flatness of the cubic curve at
// offset, which is the greater of the two control points' distances from the
// line through its ends.
//
// Port of CubicCurve2D.getFlatnessSq(double[], int).
func cubicFlatnessSq(coords []float64, offset int) float64 {
	return maxFloat64(
		ptSegDistSq(coords[offset+0], coords[offset+1],
			coords[offset+6], coords[offset+7],
			coords[offset+2], coords[offset+3]),
		ptSegDistSq(coords[offset+0], coords[offset+1],
			coords[offset+6], coords[offset+7],
			coords[offset+4], coords[offset+5]))
}

// quadFlatnessSq returns the square of the flatness of the quadratic curve at
// offset.
//
// Port of QuadCurve2D.getFlatnessSq(double[], int).
func quadFlatnessSq(coords []float64, offset int) float64 {
	return ptSegDistSq(coords[offset+0], coords[offset+1],
		coords[offset+4], coords[offset+5],
		coords[offset+2], coords[offset+3])
}

// subdivideCubic splits the cubic curve at src[srcoff] in half, writing the two
// halves at leftoff and rightoff. Either destination may be nil.
//
// Port of CubicCurve2D.subdivide(double[], int, double[], int, double[], int).
func subdivideCubic(src []float64, srcoff int, left []float64, leftoff int,
	right []float64, rightoff int) {
	x1 := src[srcoff+0]
	y1 := src[srcoff+1]
	ctrlx1 := src[srcoff+2]
	ctrly1 := src[srcoff+3]
	ctrlx2 := src[srcoff+4]
	ctrly2 := src[srcoff+5]
	x2 := src[srcoff+6]
	y2 := src[srcoff+7]
	if left != nil {
		left[leftoff+0] = x1
		left[leftoff+1] = y1
	}
	if right != nil {
		right[rightoff+6] = x2
		right[rightoff+7] = y2
	}
	x1 = (x1 + ctrlx1) / 2
	y1 = (y1 + ctrly1) / 2
	x2 = (x2 + ctrlx2) / 2
	y2 = (y2 + ctrly2) / 2
	centerx := (ctrlx1 + ctrlx2) / 2
	centery := (ctrly1 + ctrly2) / 2
	ctrlx1 = (x1 + centerx) / 2
	ctrly1 = (y1 + centery) / 2
	ctrlx2 = (x2 + centerx) / 2
	ctrly2 = (y2 + centery) / 2
	centerx = (ctrlx1 + ctrlx2) / 2
	centery = (ctrly1 + ctrly2) / 2
	if left != nil {
		left[leftoff+2] = x1
		left[leftoff+3] = y1
		left[leftoff+4] = ctrlx1
		left[leftoff+5] = ctrly1
		left[leftoff+6] = centerx
		left[leftoff+7] = centery
	}
	if right != nil {
		right[rightoff+0] = centerx
		right[rightoff+1] = centery
		right[rightoff+2] = ctrlx2
		right[rightoff+3] = ctrly2
		right[rightoff+4] = x2
		right[rightoff+5] = y2
	}
}

// subdivideQuad splits the quadratic curve at src[srcoff] in half.
//
// Port of QuadCurve2D.subdivide(double[], int, double[], int, double[], int).
func subdivideQuad(src []float64, srcoff int, left []float64, leftoff int,
	right []float64, rightoff int) {
	x1 := src[srcoff+0]
	y1 := src[srcoff+1]
	ctrlx := src[srcoff+2]
	ctrly := src[srcoff+3]
	x2 := src[srcoff+4]
	y2 := src[srcoff+5]
	if left != nil {
		left[leftoff+0] = x1
		left[leftoff+1] = y1
	}
	if right != nil {
		right[rightoff+4] = x2
		right[rightoff+5] = y2
	}
	x1 = (x1 + ctrlx) / 2
	y1 = (y1 + ctrly) / 2
	x2 = (x2 + ctrlx) / 2
	y2 = (y2 + ctrly) / 2
	ctrlx = (x1 + x2) / 2
	ctrly = (y1 + y2) / 2
	if left != nil {
		left[leftoff+2] = x1
		left[leftoff+3] = y1
		left[leftoff+4] = ctrlx
		left[leftoff+5] = ctrly
	}
	if right != nil {
		right[rightoff+0] = ctrlx
		right[rightoff+1] = ctrly
		right[rightoff+2] = x2
		right[rightoff+3] = y2
	}
}

// ptSegDistSq returns the square of the distance from the point (px, py) to the
// line segment from (x1, y1) to (x2, y2).
//
// Port of Line2D.ptSegDistSq.
func ptSegDistSq(x1, y1, x2, y2, px, py float64) float64 {
	// Adjust vectors relative to x1,y1
	// x2,y2 becomes relative vector from x1,y1 to end of segment
	x2 -= x1
	y2 -= y1
	// px,py becomes relative vector from x1,y1 to test point
	px -= x1
	py -= y1
	dotprod := px*x2 + py*y2
	var projlenSq float64
	if dotprod <= 0.0 {
		// px,py is on the side of x1,y1 away from x2,y2
		// distance to segment is length of px,py vector
		// "length of its (clipped) projection" is now 0.0
		projlenSq = 0.0
	} else {
		// switch to backwards vectors relative to x2,y2
		// x2,y2 are already the negative of x1,y1=>x2,y2
		// to get px,py to be the negative of px,py=>x2,y2
		// the dot product of two negated vectors is the same
		// as the dot product of the two normal vectors
		px = x2 - px
		py = y2 - py
		dotprod = px*x2 + py*y2
		if dotprod <= 0.0 {
			// px,py is on the side of x2,y2 away from x1,y1
			// distance to segment is length of (backwards) px,py vector
			// "length of its (clipped) projection" is now 0.0
			projlenSq = 0.0
		} else {
			// px,py is between x1,y1 and x2,y2
			// dotprod is the length of the px,py vector
			// projected on the x2,y2=>x1,y1 vector times the
			// length of the x2,y2=>x1,y1 vector
			projlenSq = dotprod * dotprod / (x2*x2 + y2*y2)
		}
	}
	// Distance to line is now the length of the relative point
	// vector minus the length of its projection onto the line
	// (which is zero if the projection falls outside the range
	//  of the line segment).
	lenSq := px*px + py*py - projlenSq
	if lenSq < 0 {
		lenSq = 0
	}
	return lenSq
}

// maxFloat64 is Math.max for a double.
func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
