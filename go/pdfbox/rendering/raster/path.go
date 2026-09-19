package raster

// Turning a shape into a coverage mask.
//
// There is no Java here either: Graphics2D.fill takes a Shape and a winding
// rule and produces coverage, and this is that step written out.
//
// Two libraries do the work between them, and the split is not arbitrary:
//
//   - **github.com/golang/freetype/raster** scan-converts. It takes moves,
//     lines and both kinds of curve without flattening them first, and it
//     honours **both winding rules** -- `UseNonZeroWinding` -- which PDF needs
//     for `f` and `f*` alike.
//   - **github.com/srwiley/rasterx** strokes. Its stroker is the only pure-Go
//     one with PDF's whole model: miter, miter limit, round and bevel joins,
//     the three caps, and dashes with a phase.
//
// A0 chose rasterx for the stroking and did not notice that its scanner is
// nonzero only -- `ScannerGV.SetWinding` is a documented no-op, because
// x/image/vector underneath it counts only that way. freetype's rasteriser
// fills the gap, imports nothing but golang.org/x/image/math/fixed, and takes
// curves natively. See migration/STATUS.md.

import (
	"image"
	"math"
	"sync"

	"github.com/golang/freetype/raster"
	"golang.org/x/image/math/fixed"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// fixedPoint is a device-space point in freetype's 26.6 fixed point.
func fixedPoint(x, y float64) fixed.Point26_6 {
	return fixed.Point26_6{
		X: fixed.Int26_6(x * 64),
		Y: fixed.Int26_6(y * 64),
	}
}

// walkedPath is a shape walked through a transform into device space: its
// segments, the winding rule it asked for, and the box that every point it
// has falls inside, control points included.
type walkedPath struct {
	kinds  []int
	coords []float64
	rule   int

	minX, minY, maxX, maxY float64
}

// walkedPaths keeps the buffers of walked paths for the next shape.
var walkedPaths = sync.Pool{New: func() any { return new(walkedPath) }}

// walkShape walks a shape through a transform.
//
// The transform is applied by the iterator, which is what Java's
// Graphics2D.fill does with its own transform. Put the result back in
// walkedPaths once it is added.
func walkShape(shape geom.Shape, at *geom.AffineTransform) *walkedPath {
	p := walkedPaths.Get().(*walkedPath)
	p.kinds = p.kinds[:0]
	p.coords = p.coords[:0]
	p.minX, p.minY = math.Inf(1), math.Inf(1)
	p.maxX, p.maxY = math.Inf(-1), math.Inf(-1)

	iterator := shape.PathIterator(at)
	coords := make([]float64, 6)
	for ; !iterator.IsDone(); iterator.Next() {
		kind := iterator.CurrentSegment(coords)
		p.kinds = append(p.kinds, kind)
		for k := 0; k < 2*pointsOf(kind); k += 2 {
			x, y := coords[k], coords[k+1]
			p.coords = append(p.coords, x, y)
			// math.Min and math.Max answer NaN for a NaN, and reach sends a
			// path whose box is not finite down the whole surface.
			p.minX = math.Min(p.minX, x)
			p.maxX = math.Max(p.maxX, x)
			p.minY = math.Min(p.minY, y)
			p.maxY = math.Max(p.maxY, y)
		}
	}
	p.rule = iterator.WindingRule()
	return p
}

// pointsOf is how many points a segment of the given kind carries.
func pointsOf(kind int) int {
	switch kind {
	case geom.SegMoveTo, geom.SegLineTo:
		return 1
	case geom.SegQuadTo:
		return 2
	case geom.SegCubicTo:
		return 3
	}
	return 0
}

// addTo feeds the path to a rasteriser.
//
// A subpath that is not closed is still filled: both java.awt and PDF close an
// open subpath implicitly when filling it, and freetype's rasteriser does the
// same by joining the last point back to the first when the next Start arrives.
func (p *walkedPath) addTo(r *raster.Rasterizer) {
	point := func(at int) fixed.Point26_6 {
		return fixedPoint(p.coords[at], p.coords[at+1])
	}
	var start, current fixed.Point26_6
	started := false
	at := 0
	for _, kind := range p.kinds {
		switch kind {
		case geom.SegMoveTo:
			if started {
				r.Add1(start)
			}
			start = point(at)
			current = start
			r.Start(start)
			started = true

		case geom.SegLineTo:
			current = point(at)
			r.Add1(current)

		case geom.SegQuadTo:
			control := point(at)
			current = point(at + 2)
			r.Add2(control, current)

		case geom.SegCubicTo:
			first := point(at)
			second := point(at + 2)
			current = point(at + 4)
			r.Add3(first, second, current)

		case geom.SegClose:
			if started && current != start {
				r.Add1(start)
			}
			current = start
		}
		at += 2 * pointsOf(kind)
	}
	if started {
		// close the last subpath, as filling always does
		r.Add1(start)
	}
}

// reach answers the part of a width by height surface the path can put
// anything on: the box of its points, control points included, a pixel wider
// before and two after, as paddedRect pads. A path with a coordinate that is
// not a finite number reaches the whole surface.
func (p *walkedPath) reach(width, height int) image.Rectangle {
	if len(p.coords) == 0 || !finite(p.minX, p.minY, p.maxX, p.maxY) {
		return image.Rect(0, 0, width, height)
	}
	return reachOf(p.minX, p.minY, p.maxX, p.maxY, width, height)
}

// reachOf is reach for a box already worked out.
func reachOf(minX, minY, maxX, maxY float64, width, height int) image.Rectangle {
	return image.Rect(lowOf(minX, width), lowOf(minY, height),
		highOf(maxX, width), highOf(maxY, height)).Intersect(image.Rect(0, 0, width, height))
}

// lowOf is where the reach starts on one axis: a whole pixel before the least
// coordinate, and never before 0 or past the surface.
func lowOf(least float64, size int) int {
	if least < 1 {
		return 0
	}
	if least > float64(size) {
		return size
	}
	return int(math.Floor(least)) - 1
}

// highOf is where the reach ends on one axis: two pixels past the greatest
// coordinate, and never past the surface.
//
// A shape just off the top or the left edge still reaches pixel 0. freetype
// finds a coordinate's pixel by dividing its 26.6 value by 64, and Go's
// division truncates toward zero, so everything between -1 and 0 lands on
// pixel 0 and is drawn there. Only a shape wholly at -1 or beyond reaches
// nothing.
func highOf(greatest float64, size int) int {
	if greatest >= float64(size) {
		return size
	}
	if greatest <= -1 {
		return 0
	}
	return min(size, int(math.Ceil(greatest))+2)
}

// finite reports whether every value is a number and not an infinity.
func finite(values ...float64) bool {
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}

// sizedRasterizer is a rasteriser and the surface size it was last set to.
//
// freetype's SetBounds allocates a row index as tall as the surface each time
// it is called, and a page of many shapes called it for every one; Clear only
// marks the rows empty. The size also decides how finely the rasteriser splits
// curves, which is why a mask the size of a shape is still rasterised with the
// size of the surface it is for.
type sizedRasterizer struct {
	*raster.Rasterizer
	width, height int
}

// rasterizers keeps rasterisers for the next shape.
var rasterizers = sync.Pool{New: func() any {
	return &sizedRasterizer{Rasterizer: raster.NewRasterizer(0, 0)}
}}

// borrowRasterizer answers an empty rasteriser for a surface of the given
// size. Put it back in rasterizers when it is done.
func borrowRasterizer(width, height int) *sizedRasterizer {
	r := rasterizers.Get().(*sizedRasterizer)
	if r.width != width || r.height != height {
		r.SetBounds(width, height)
		r.width, r.height = width, height
	} else {
		r.Clear()
	}
	return r
}

// rasterizeInto scan-converts what the rasteriser holds into a mask over area
// of the surface. The spans are in the surface's own coordinates, and the
// painter keeps what lands inside the mask.
func rasterizeInto(r *raster.Rasterizer, area image.Rectangle, antiAliasing bool) *image.Alpha {
	mask := image.NewAlpha(area)
	if !area.Empty() {
		var painter raster.Painter = raster.NewAlphaSrcPainter(mask)
		if !antiAliasing {
			// Java turns anti-aliasing off for an axis-aligned rectangle and
			// for an image scaled up, and what it means by off is that a pixel
			// is in or out. That is what a MonochromePainter does to the spans
			// on their way through.
			painter = raster.NewMonochromePainter(painter)
		}
		r.Rasterize(painter)
	}
	return mask
}

// coverageOf rasterises a shape into an alpha mask for a surface of the given
// size, where 0 is untouched and 255 is fully covered.
//
// The mask covers the part of the surface the shape reaches, and its Rect says
// where; outside it the coverage is 0, which is what AlphaAt answers there. It
// used to be the whole surface for every shape: a page of 222,868 strokes
// allocated and cleared 4.1 MB for each, and that was two thirds of its render.
//
// The shape is rasterised where it is, not moved to the mask's corner.
// freetype decides how finely to split a cubic from a-3(b+c)+d, whose
// coefficients do not sum to zero, so the same curve moved by whole pixels is
// flattened differently and draws different edge pixels.
func coverageOf(shape geom.Shape, at *geom.AffineTransform, width, height int,
	antiAliasing bool) *image.Alpha {
	path := walkShape(shape, at)
	defer walkedPaths.Put(path)
	area := path.reach(width, height)

	r := borrowRasterizer(width, height)
	defer rasterizers.Put(r)
	path.addTo(r.Rasterizer)
	r.UseNonZeroWinding = path.rule == geom.WindNonZero
	return rasterizeInto(r.Rasterizer, area, antiAliasing)
}
