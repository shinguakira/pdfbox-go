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

// addShape walks a shape through a transform and feeds it to a rasteriser,
// answering the winding rule the shape asked for.
//
// The transform is applied by the iterator, which is what Java's
// Graphics2D.fill does with its own transform. A subpath that is not closed is
// still filled: both java.awt and PDF close an open subpath implicitly when
// filling it, and freetype's rasteriser does the same by joining the last
// point back to the first when the next Start arrives.
func addShape(r *raster.Rasterizer, shape geom.Shape, at *geom.AffineTransform) int {
	iterator := shape.PathIterator(at)
	coords := make([]float64, 6)
	var start, current fixed.Point26_6
	started := false

	for ; !iterator.IsDone(); iterator.Next() {
		switch iterator.CurrentSegment(coords) {
		case geom.SegMoveTo:
			if started {
				r.Add1(start)
			}
			start = fixedPoint(coords[0], coords[1])
			current = start
			r.Start(start)
			started = true

		case geom.SegLineTo:
			current = fixedPoint(coords[0], coords[1])
			r.Add1(current)

		case geom.SegQuadTo:
			control := fixedPoint(coords[0], coords[1])
			current = fixedPoint(coords[2], coords[3])
			r.Add2(control, current)

		case geom.SegCubicTo:
			first := fixedPoint(coords[0], coords[1])
			second := fixedPoint(coords[2], coords[3])
			current = fixedPoint(coords[4], coords[5])
			r.Add3(first, second, current)

		case geom.SegClose:
			if started && current != start {
				r.Add1(start)
			}
			current = start
		}
	}
	if started {
		// close the last subpath, as filling always does
		r.Add1(start)
	}
	return iterator.WindingRule()
}

// coverageOf rasterises a shape into an alpha mask of the given size, where 0
// is untouched and 255 is fully covered.
//
// The mask's origin is (0,0), which is where the rasteriser's own coordinates
// start; every destination this package makes has its origin there too.
func coverageOf(shape geom.Shape, at *geom.AffineTransform, width, height int,
	antiAliasing bool) *image.Alpha {
	r := raster.NewRasterizer(width, height)
	r.UseNonZeroWinding = addShape(r, shape, at) == geom.WindNonZero

	mask := image.NewAlpha(image.Rect(0, 0, width, height))
	var painter raster.Painter = raster.NewAlphaSrcPainter(mask)
	if !antiAliasing {
		// Java turns anti-aliasing off for an axis-aligned rectangle and for
		// an image scaled up, and what it means by off is that a pixel is in
		// or out. That is what a MonochromePainter does to the spans on their
		// way through.
		painter = raster.NewMonochromePainter(painter)
	}
	r.Rasterize(painter)
	return mask
}
