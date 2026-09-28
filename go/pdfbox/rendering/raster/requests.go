package raster

// Where Java2D asks a paint for its pixels.
//
// Java2D does not ask a PaintContext for one pixel at a time. It asks for a
// rectangle, getRaster(x, y, w, h), and a texture's context walks from the
// rectangle's corner (see texturepaint.go), so where each rectangle begins
// decides what a textured fill looks like. None of this is PDFBox's: it is the
// JDK's pipes, and which of them a fill goes down.
//
// PDFBox's paints are custom paints to Java2D -- TilingPaint and SoftMask are
// not TexturePaint -- and every rectangle of one ends in
// sun.java2d.pipe.AlphaPaintPipe, which asks for at most TILE_SIZE, 32, pixels
// each way, from the corner of the box it is handed. What hands it the boxes
// depends on the hints and the clip:
//
//   - Antialiased, AAShapePipe asks Marlin for its tiles, 32 by 32, laid from
//     the top left of the box Marlin's Renderer.endRendering puts round the
//     shape: x in 1/256ths of a pixel, `ceil(x - 0.5)`, y in eighths shifted
//     by a half, `ceil(8y - 0.5)`, both no less than the clip's bounds. Under a
//     clip that is not a rectangle, SpanClipRenderer first shrinks each tile
//     to the part of the clip inside it, which moves the corner.
//   - Not antialiased, SpanShapeRenderer hands over the shape's spans, one
//     row high, each from where it begins; a span longer than 32 is asked for
//     32 at a time from there.
//   - Not antialiased, a Rectangle2D through a transform that keeps it upright
//     is one box, renderRect's, its corners truncated to ints and held inside
//     the clip's bounds, and asked for 32 by 32 from its corner.
//
// Recorded in testdata/texturepaint.txt, with the driver that recorded it.
//
// Two things here are not Java2D's, both of them the clip's. Java2D's clip is
// a Region, which ShapeSpanIterator makes of the clip's outline: a curve
// flattened to within a pixel, and each pixel held whole or not at all. The
// port's clip is the outline's own antialiased coverage, and inRegion reads a
// pixel covered half or more as held, so along a curved clip a span or a
// shrunk tile can begin a pixel away from Java2D's. And under a clip that is
// not a rectangle renderRect hands on the Region's own spans, which the port
// does not have, so it shrinks the tiles as it does for an antialiased fill.

import (
	goimage "image"
	"math"
	"slices"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"golang.org/x/image/math/fixed"
)

// tileSize is AlphaPaintPipe.TILE_SIZE, and Marlin's tile, which are the same.
const tileSize = 32

// requestMode says how the rectangles of one fill or stroke are laid out.
type requestMode int

const (
	// requestAlone asks for every pixel as a rectangle of its own, for a
	// caller that does not know how Java2D would have asked.
	requestAlone requestMode = iota
	// requestTiles lays tileSize squares from a corner.
	requestTiles
	// requestSpans asks for each run of pixels in a row from where it begins.
	requestSpans
)

// javaRequests is how Java2D asks for the pixels of one fill or stroke.
type javaRequests struct {
	mode requestMode
	// corner is where the first tile begins.
	corner goimage.Point
}

// requestWalker hands out the corner of the rectangle each pixel of one fill is
// asked in, visited a row at a time from the left, as the compositor visits
// them.
type requestWalker struct {
	requests javaRequests
	// clip is the clip's coverage, which shrinks each tile to the part of it
	// inside the tile; nil for none.
	clip *goimage.Alpha

	// last is the pixel visited before, which says where a span begins.
	last     goimage.Point
	runStart int
	visited  bool

	// bands remembers each tile's corner once shrunk.
	bands map[goimage.Point]goimage.Point
}

// cornerOf answers the corner of the rectangle the pixel at (x, y) is asked
// in, given the clip's coverage there. Pixels have to come in the order the
// compositor visits them.
func (w *requestWalker) cornerOf(x, y int, clipAlpha uint8) goimage.Point {
	switch w.requests.mode {
	case requestSpans:
		if !inRegion(w.clip, clipAlpha) {
			// Outside the clip as Java2D holds it, where it asks for nothing:
			// a pixel of its own, and the end of any span.
			w.visited = false
			return goimage.Pt(x, y)
		}
		if !w.visited || y != w.last.Y || x != w.last.X+1 {
			w.runStart = x
		}
		w.visited = true
		w.last = goimage.Pt(x, y)
		return goimage.Pt(w.runStart+floorTile(x-w.runStart), y)

	case requestTiles:
		c := w.requests.corner
		tile := goimage.Pt(c.X+floorTile(x-c.X), c.Y+floorTile(y-c.Y))
		if w.clip == nil {
			return tile
		}
		return w.band(tile)
	}
	return goimage.Pt(x, y)
}

// floorTile is the offset of the tile an offset from a corner falls in.
func floorTile(offset int) int {
	return offset >> 5 << 5
}

// band is the corner of a tile once SpanClipRenderer has shrunk it to the part
// of the clip inside it. A clip that is a rectangle leaves a tile laid inside
// its bounds as it was, so every clip goes through here.
func (w *requestWalker) band(tile goimage.Point) goimage.Point {
	if corner, known := w.bands[tile]; known {
		return corner
	}
	area := goimage.Rect(tile.X, tile.Y, tile.X+tileSize, tile.Y+tileSize).Intersect(w.clip.Bounds())
	corner := tile
	minX, minY := area.Max.X, area.Max.Y
	for y := area.Min.Y; y < area.Max.Y; y++ {
		row := w.clip.Pix[w.clip.PixOffset(area.Min.X, y):][:area.Dx()]
		for offset, alpha := range row {
			if !inRegion(w.clip, alpha) {
				continue
			}
			minY = min(minY, y)
			minX = min(minX, area.Min.X+offset)
			break
		}
	}
	if minX < area.Max.X {
		corner = goimage.Pt(max(tile.X, minX), max(tile.Y, minY))
	}
	if w.bands == nil {
		w.bands = map[goimage.Point]goimage.Point{}
	}
	w.bands[tile] = corner
	return corner
}

// inRegion says whether a pixel of the clip's coverage is inside the clip as
// Java2D holds it: a Region of whole pixels, each held where the clip holds its
// centre. The port's clip is antialiased, and a pixel covered half or more is
// the nearest it has to that.
func inRegion(clip *goimage.Alpha, alpha uint8) bool {
	return clip == nil || alpha >= 0x80
}

// marlinBox is the top left of the box Marlin's Renderer puts round the edges
// it is given, which is where AAShapePipe lays its first tile.
//
// Port of the parts of Renderer.addLine and Renderer.endRendering that decide
// it. An edge that crosses no row of subpixel centres is dropped and counts for
// nothing; the others lower the box's left to their lower end's x and its top
// to the first row they cross.
type marlinBox struct {
	boundsMinX, boundsMinY, boundsMaxY int
	edgeMinX                           float64
	edgeMinY                           int
	found                              bool
}

// Marlin's subpixels: 2^8 across a pixel and 2^3 down it.
const (
	subpixelsX = 256
	subpixelsY = 8
)

// newMarlinBox starts a box for the given clip bounds, which
// Renderer.init takes in whole pixels.
func newMarlinBox(clip goimage.Rectangle) *marlinBox {
	return &marlinBox{
		boundsMinX: clip.Min.X * subpixelsX,
		boundsMinY: clip.Min.Y * subpixelsY,
		boundsMaxY: clip.Max.Y * subpixelsY,
		edgeMinX:   math.Inf(1),
		edgeMinY:   math.MaxInt32,
	}
}

// addLine is Renderer.addLine's part in the box, for an edge in device space.
func (b *marlinBox) addLine(x1, y1, x2, y2 float64) {
	// tosubpixx and tosubpixy; y is shifted by a half "for fast ceil(y - 0.5)".
	x1, x2 = x1*subpixelsX, x2*subpixelsX
	y1, y2 = y1*subpixelsY-0.5, y2*subpixelsY-0.5
	if y2 < y1 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}
	first := max(javaIntOf(math.Ceil(y1)), b.boundsMinY)
	last := min(javaIntOf(math.Ceil(y2)), b.boundsMaxY)
	if first >= last {
		return
	}
	b.edgeMinY = min(b.edgeMinY, first)
	b.edgeMinX = math.Min(b.edgeMinX, math.Min(x1, x2))
	b.found = true
}

// corner is endRendering's pminX and pminY, and false where no edge counted.
func (b *marlinBox) corner() (goimage.Point, bool) {
	if !b.found {
		return goimage.Point{}, false
	}
	spminX := max(javaIntOf(math.Ceil(b.edgeMinX-0.5)), b.boundsMinX)
	return goimage.Pt(spminX>>8, b.edgeMinY>>3), true
}

// addPath gives the box the edges of a walked path, closing each subpath as
// Marlin's moveTo and closePath do. A curve counts by its extremes: Marlin
// flattens it into lines whose ends lie on it, and the lowest of those is its
// lowest point, give or take the flattening.
func (b *marlinBox) addPath(p *walkedPath) {
	var startX, startY, curX, curY float64
	open := false
	closeSubpath := func() {
		if open && (curX != startX || curY != startY) {
			b.addLine(curX, curY, startX, startY)
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
			b.addLine(curX, curY, x, y)
			curX, curY = x, y
		case geom.SegQuadTo, geom.SegCubicTo:
			points := []float64{curX, curY}
			points = append(points, p.coords[at:at+2*pointsOf(kind)]...)
			b.addCurve(points)
			curX, curY = points[len(points)-2], points[len(points)-1]
		case geom.SegClose:
			closeSubpath()
			curX, curY = startX, startY
		}
		at += 2 * pointsOf(kind)
	}
	closeSubpath()
}

// addCurve adds a quadratic or cubic Bézier, given as its points, as the lines
// through its ends and the points where it turns in x or in y.
func (b *marlinBox) addCurve(points []float64) {
	ts := []float64{0}
	for axis := 0; axis < 2; axis++ {
		ts = append(ts, curveTurns(points, axis)...)
	}
	ts = append(ts, 1)
	slices.Sort(ts)
	prevX, prevY := curvePoint(points, ts[0])
	for _, t := range ts[1:] {
		x, y := curvePoint(points, t)
		b.addLine(prevX, prevY, x, y)
		prevX, prevY = x, y
	}
}

// curveTurns answers the parameters inside (0, 1) where a Bézier's coordinate
// on the given axis stops rising or falling.
func curveTurns(points []float64, axis int) []float64 {
	p := make([]float64, 0, 4)
	for k := axis; k < len(points); k += 2 {
		p = append(p, points[k])
	}
	var roots []float64
	switch len(p) {
	case 3:
		// B'(t) = 2((p1-p0) + t(p0 - 2p1 + p2))
		if d := p[0] - 2*p[1] + p[2]; d != 0 {
			roots = append(roots, (p[0]-p[1])/d)
		}
	case 4:
		// B'(t)/3 = at^2 + bt + c
		a := -p[0] + 3*p[1] - 3*p[2] + p[3]
		bb := 2 * (p[0] - 2*p[1] + p[2])
		c := p[1] - p[0]
		if a == 0 {
			if bb != 0 {
				roots = append(roots, -c/bb)
			}
		} else if disc := bb*bb - 4*a*c; disc >= 0 {
			sq := math.Sqrt(disc)
			roots = append(roots, (-bb+sq)/(2*a), (-bb-sq)/(2*a))
		}
	}
	inside := roots[:0]
	for _, t := range roots {
		if t > 0 && t < 1 {
			inside = append(inside, t)
		}
	}
	return inside
}

// curvePoint is a quadratic or cubic Bézier at t.
func curvePoint(points []float64, t float64) (float64, float64) {
	u := 1 - t
	if len(points) == 6 {
		return u*u*points[0] + 2*u*t*points[2] + t*t*points[4],
			u*u*points[1] + 2*u*t*points[3] + t*t*points[5]
	}
	return u*u*u*points[0] + 3*u*u*t*points[2] + 3*u*t*t*points[4] + t*t*t*points[6],
		u*u*u*points[1] + 3*u*u*t*points[3] + 3*u*t*t*points[5] + t*t*t*points[7]
}

// addOutline gives the box the edges of a stroke's outline, each piece joined
// back to its start as the rasteriser joins it.
func (b *marlinBox) addOutline(polygons [][]fixed.Point26_6) {
	for _, polygon := range polygons {
		for k := range polygon {
			from := polygon[k]
			to := polygon[(k+1)%len(polygon)]
			b.addLine(float64(from.X)/64, float64(from.Y)/64, float64(to.X)/64, float64(to.Y)/64)
		}
	}
}

// fillRequests is how Java2D asks for the pixels of a fill of the given shape,
// walked into device space as path.
func (i *Image) fillRequests(shape geom.Shape, path *walkedPath) javaRequests {
	clip := i.javaClipBounds()
	if !i.antiAliasing {
		if rect, isRect := shape.(*geom.Rectangle2D); isRect && rectilinear(i.transform) {
			return renderRectRequests(rect, i.transform, clip)
		}
		return javaRequests{mode: requestSpans}
	}
	box := newMarlinBox(clip)
	box.addPath(path)
	corner, found := box.corner()
	if !found {
		return javaRequests{}
	}
	return javaRequests{mode: requestTiles, corner: corner}
}

// strokeRequests is fillRequests for a stroke, whose outline is what Marlin
// is given.
func (i *Image) strokeRequests(outline [][]fixed.Point26_6) javaRequests {
	if !i.antiAliasing {
		return javaRequests{mode: requestSpans}
	}
	box := newMarlinBox(i.javaClipBounds())
	box.addOutline(outline)
	corner, found := box.corner()
	if !found {
		return javaRequests{}
	}
	return javaRequests{mode: requestTiles, corner: corner}
}

// rectilinear is SpanShapeRenderer's test, that a transform has neither a
// general rotation nor a general transform: it keeps a rectangle's sides
// upright, as a scale, a flip or a quarter turn does.
func rectilinear(at *geom.AffineTransform) bool {
	if at == nil {
		return true
	}
	return (at.ShearX() == 0 && at.ShearY() == 0) || (at.ScaleX() == 0 && at.ScaleY() == 0)
}

// renderRectRequests is the box SpanShapeRenderer.renderRect hands on: the
// rectangle's corners through the transform, truncated to ints, and held
// inside the clip's bounds.
func renderRectRequests(rect *geom.Rectangle2D, at *geom.AffineTransform,
	clip goimage.Rectangle) javaRequests {
	corners := []float64{rect.X, rect.Y, rect.Width, rect.Height}
	corners[2] += corners[0]
	corners[3] += corners[1]
	if corners[2] <= corners[0] || corners[3] <= corners[1] {
		return javaRequests{}
	}
	if at != nil {
		at.TransformDoubles(corners, 0, corners, 0, 2)
	}
	x0, y0 := math.Min(corners[0], corners[2]), math.Min(corners[1], corners[3])
	corner := goimage.Pt(max(javaIntOf(x0), clip.Min.X), max(javaIntOf(y0), clip.Min.Y))
	return javaRequests{mode: requestTiles, corner: corner}
}

// needsRequests reports whether a paint reads the rectangle it is asked in.
// A texture does, and nothing else: see texturepaint.go.
func needsRequests(paint rendering.Paint) bool {
	switch p := paint.(type) {
	case rendering.TilingPaint, rendering.ImagePaint:
		return true
	case rendering.SoftMaskedPaint:
		return needsRequests(p.Paint)
	}
	return false
}
