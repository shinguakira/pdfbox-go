package raster

// Scan conversion with antialiasing off.
//
// A pixel is in or out, and Java2D decides which by asking whether the shape
// covers one point of it -- its centre, or its corner, which fillrule.go has
// already turned into a move of the path. It does not ask how much of the
// pixel is covered. The difference is not only at the edges of a shape that
// crosses a pixel: it is exactly on the ties. A rectangle whose edges fall on
// pixel centres covers half of the pixels either side of it, and Java2D fills
// the one whose centre the shape reaches -- `ceil(x - 0.5)` for the first and
// `ceil(x - 0.5) - 1` for the last, in both of its fillers -- where a
// half-covered pixel quantised by its coverage would take both. The port used
// to quantise coverage, which was right except on those ties, and the ties are
// where every rectangle lands once the path is moved.
//
// So a fill with antialiasing off is scan-converted here rather than by
// freetype: the crossings of each scanline through the pixels' centres, in the
// winding rule the path asked for, and the pixels whose centres fall between
// them.
//
// A curve is flattened, which Java2D also does -- ShapeSpanIterator subdivides
// until the control points are within a pixel of the chord, ProcessPath steps
// it by forward differencing -- but not the same way, so a curve filled with
// antialiasing off can differ from Java2D's by a pixel along its edge. PDFBox
// turns antialiasing off only for a rectangular path, so that is reachable
// only on a page rendered with antialiasing off throughout.

import (
	"image"
	"math"
	"slices"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// flatness is how far a flattened curve may sit from the curve, in pixels.
const flatness = 0.01

// sampledEdge is one line of a path, its ends ordered down the page, and the
// direction it was walked in for the winding count.
type sampledEdge struct {
	x0, y0, x1, y1 float64
	direction      int
}

// sampledCoverage answers the mask of a path filled with antialiasing off:
// 0xFF where the shape covers the pixel's centre and nothing anywhere else.
func (p *walkedPath) sampledCoverage(area image.Rectangle) *image.Alpha {
	mask := image.NewAlpha(area)
	if area.Empty() {
		return mask
	}
	edges := p.edges()
	if len(edges) == 0 {
		return mask
	}
	slices.SortFunc(edges, func(a, b sampledEdge) int {
		switch {
		case a.y0 < b.y0:
			return -1
		case a.y0 > b.y0:
			return 1
		}
		return 0
	})

	var active []sampledEdge
	var crossings []struct {
		x         float64
		direction int
	}
	next := 0
	for y := area.Min.Y; y < area.Max.Y; y++ {
		centre := float64(y) + 0.5
		for next < len(edges) && edges[next].y0 <= centre {
			active = append(active, edges[next])
			next++
		}
		active = slices.DeleteFunc(active, func(e sampledEdge) bool { return e.y1 <= centre })
		if len(active) == 0 {
			continue
		}

		crossings = crossings[:0]
		for _, e := range active {
			if e.y0 > centre {
				continue
			}
			x := e.x0 + (centre-e.y0)*(e.x1-e.x0)/(e.y1-e.y0)
			crossings = append(crossings, struct {
				x         float64
				direction int
			}{x, e.direction})
		}
		slices.SortFunc(crossings, func(a, b struct {
			x         float64
			direction int
		}) int {
			switch {
			case a.x < b.x:
				return -1
			case a.x > b.x:
				return 1
			}
			return 0
		})

		row := mask.Pix[mask.PixOffset(area.Min.X, y):][:area.Dx()]
		winding, from := 0, 0.0
		inside := false
		for _, crossing := range crossings {
			winding += crossing.direction
			nowInside := winding != 0
			if p.rule == geom.WindEvenOdd {
				nowInside = winding&1 != 0
			}
			switch {
			case nowInside && !inside:
				from = crossing.x
			case !nowInside && inside:
				fillCentres(row, area.Min.X, area.Max.X, from, crossing.x)
			}
			inside = nowInside
		}
	}
	return mask
}

// fillCentres sets the pixels of one row whose centres fall in [from, to):
// the first is `ceil(from - 0.5)` and the last the one before
// `ceil(to - 0.5)`, which is how both of Java2D's fillers bound a span.
func fillCentres(row []uint8, minX, maxX int, from, to float64) {
	first := max(javaIntOf(math.Ceil(from-0.5)), minX)
	last := min(javaIntOf(math.Ceil(to-0.5)), maxX)
	for x := first; x < last; x++ {
		row[x-minX] = 0xFF
	}
}

// edges answers the path's lines, curves flattened, each ordered down the page
// and carrying the direction it was walked in. A subpath that was left open is
// closed, which is what filling one does.
func (p *walkedPath) edges() []sampledEdge {
	var edges []sampledEdge
	add := func(x0, y0, x1, y1 float64) {
		switch {
		case y0 < y1:
			edges = append(edges, sampledEdge{x0, y0, x1, y1, 1})
		case y0 > y1:
			edges = append(edges, sampledEdge{x1, y1, x0, y0, -1})
		}
	}
	var startX, startY, curX, curY float64
	open := false
	closeSubpath := func() {
		if open {
			add(curX, curY, startX, startY)
		}
		open = false
	}
	at := 0
	for _, kind := range p.kinds {
		switch kind {
		case geom.SegMoveTo:
			closeSubpath()
			startX, startY = p.coords[at], p.coords[at+1]
			curX, curY = startX, startY
			open = true
		case geom.SegLineTo:
			x, y := p.coords[at], p.coords[at+1]
			add(curX, curY, x, y)
			curX, curY = x, y
		case geom.SegQuadTo, geom.SegCubicTo:
			points := append([]float64{curX, curY}, p.coords[at:at+2*pointsOf(kind)]...)
			curX, curY = flattenCurve(points, add)
		case geom.SegClose:
			closeSubpath()
			curX, curY = startX, startY
			// A close leaves the point where the subpath began, and what
			// follows without a move of its own carries on from there.
			open = true
		}
		at += 2 * pointsOf(kind)
	}
	closeSubpath()
	return edges
}

// flattenCurve splits a quadratic or a cubic until it is within flatness of
// its chord, handing each piece to add, and answers where it ended.
func flattenCurve(points []float64, add func(x0, y0, x1, y1 float64)) (float64, float64) {
	last := len(points) - 2
	if flatEnough(points) {
		add(points[0], points[1], points[last], points[last+1])
		return points[last], points[last+1]
	}
	left, right := splitCurve(points)
	flattenCurve(left, add)
	return flattenCurve(right, add)
}

// flatEnough reports whether every control point is within flatness of the
// line between the curve's ends.
func flatEnough(points []float64) bool {
	x0, y0 := points[0], points[1]
	last := len(points) - 2
	x1, y1 := points[last], points[last+1]
	for k := 2; k < last; k += 2 {
		if pointToLine(points[k], points[k+1], x0, y0, x1, y1) > flatness {
			return false
		}
	}
	return true
}

// pointToLine is how far a point is from a line segment's line.
func pointToLine(px, py, x0, y0, x1, y1 float64) float64 {
	dx, dy := x1-x0, y1-y0
	length := math.Hypot(dx, dy)
	if length == 0 {
		return math.Hypot(px-x0, py-y0)
	}
	return math.Abs((px-x0)*dy-(py-y0)*dx) / length
}

// splitCurve halves a quadratic or a cubic at its middle, de Casteljau.
func splitCurve(points []float64) ([]float64, []float64) {
	if len(points) == 6 {
		x0, y0, cx, cy, x1, y1 := points[0], points[1], points[2], points[3], points[4], points[5]
		ax, ay := (x0+cx)/2, (y0+cy)/2
		bx, by := (cx+x1)/2, (cy+y1)/2
		mx, my := (ax+bx)/2, (ay+by)/2
		return []float64{x0, y0, ax, ay, mx, my}, []float64{mx, my, bx, by, x1, y1}
	}
	x0, y0 := points[0], points[1]
	c1x, c1y := points[2], points[3]
	c2x, c2y := points[4], points[5]
	x1, y1 := points[6], points[7]
	ax, ay := (x0+c1x)/2, (y0+c1y)/2
	bx, by := (c1x+c2x)/2, (c1y+c2y)/2
	cx, cy := (c2x+x1)/2, (c2y+y1)/2
	dx, dy := (ax+bx)/2, (ay+by)/2
	ex, ey := (bx+cx)/2, (by+cy)/2
	mx, my := (dx+ex)/2, (dy+ey)/2
	return []float64{x0, y0, ax, ay, dx, dy, mx, my},
		[]float64{mx, my, ex, ey, cx, cy, x1, y1}
}
