package raster

// Turning a stroke into a shape.
//
// Java's Graphics2D.draw(shape) strokes with the current BasicStroke; the port
// asks rasterx's stroker for the outline and then fills it, which is what
// BasicStroke.createStrokedShape does and what Java2D does underneath.
//
// The stroke arrives in **device space**: PageDrawer.getStroke has already put
// the line width and the dash phase through the CTM -- see its transformWidth
// calls -- so the path is transformed first and stroked afterwards, at the
// width it was given.

import (
	"image"

	"github.com/golang/freetype/raster"
	"github.com/srwiley/rasterx"
	"golang.org/x/image/math/fixed"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// outlineRecorder is a rasterx.Scanner that keeps the polygons the stroker
// produces instead of scan-converting them.
//
// rasterx strokes into a Scanner, and its own scanner counts only the nonzero
// winding rule; freetype's rasteriser counts both and is what this package
// fills with. So the stroker's output is collected here and handed on. A
// stroke outline is filled nonzero either way -- the rule is the fill's, and
// the outline is a fill of its own -- so nothing is lost by the detour.
type outlineRecorder struct {
	polygons [][]fixed.Point26_6
	current  []fixed.Point26_6
	extent   fixed.Rectangle26_6
	first    bool
}

func newOutlineRecorder() *outlineRecorder { return &outlineRecorder{first: true} }

func (o *outlineRecorder) Start(a fixed.Point26_6) {
	o.close()
	o.current = []fixed.Point26_6{a}
	o.grow(a)
}

func (o *outlineRecorder) Line(b fixed.Point26_6) {
	o.current = append(o.current, b)
	o.grow(b)
}

// Draw ends the outline. rasterx calls it when the stroke is complete.
func (o *outlineRecorder) Draw() { o.close() }

func (o *outlineRecorder) close() {
	if len(o.current) > 1 {
		o.polygons = append(o.polygons, o.current)
	}
	o.current = nil
}

func (o *outlineRecorder) grow(p fixed.Point26_6) {
	if o.first {
		o.extent = fixed.Rectangle26_6{Min: p, Max: p}
		o.first = false
		return
	}
	if p.X < o.extent.Min.X {
		o.extent.Min.X = p.X
	}
	if p.Y < o.extent.Min.Y {
		o.extent.Min.Y = p.Y
	}
	if p.X > o.extent.Max.X {
		o.extent.Max.X = p.X
	}
	if p.Y > o.extent.Max.Y {
		o.extent.Max.Y = p.Y
	}
}

func (o *outlineRecorder) GetPathExtent() fixed.Rectangle26_6 { return o.extent }
func (o *outlineRecorder) SetBounds(w, h int)                 {}
func (o *outlineRecorder) SetColor(interface{})               {}
func (o *outlineRecorder) SetWinding(bool)                    {}
func (o *outlineRecorder) SetClip(image.Rectangle)            {}

func (o *outlineRecorder) Clear() {
	o.polygons = nil
	o.current = nil
	o.first = true
}

// rasterize fills the collected polygons into an alpha mask, nonzero.
func (o *outlineRecorder) rasterize(width, height int, antiAliasing bool) *image.Alpha {
	r := raster.NewRasterizer(width, height)
	r.UseNonZeroWinding = true
	// The pieces are not closed one by one. rasterx emits a stroke outline as
	// a stream of separate segments -- the two sides, the caps and the joins,
	// each arriving as its own Start and Line -- and it is their edges
	// together that enclose the stroke. Closing each piece back to its own
	// start would make every one of them a zero-area sliver and draw nothing;
	// a scanline rasteriser counts crossings, and the crossings of the pieces
	// already add up to the closed outline.
	for _, polygon := range o.polygons {
		r.Start(polygon[0])
		for _, point := range polygon[1:] {
			r.Add1(point)
		}
	}
	mask := image.NewAlpha(image.Rect(0, 0, width, height))
	var painter raster.Painter = raster.NewAlphaSrcPainter(mask)
	if !antiAliasing {
		painter = raster.NewMonochromePainter(painter)
	}
	r.Rasterize(painter)
	return mask
}

// capFunc is `J`: 0 butt, 1 round, 2 square.
func capFunc(lineCap int) rasterx.CapFunc {
	switch lineCap {
	case 1:
		return rasterx.RoundCap
	case 2:
		return rasterx.SquareCap
	default:
		return rasterx.ButtCap
	}
}

// joinMode is `j`: 0 miter, 1 round, 2 bevel.
//
// rasterx offers two more -- Arc and ArcClip, which are SVG 2's `arcs` -- and
// PDF has no such join.
func joinMode(lineJoin int) rasterx.JoinMode {
	switch lineJoin {
	case 1:
		return rasterx.Round
	case 2:
		return rasterx.Bevel
	default:
		return rasterx.Miter
	}
}

// strokeCoverage strokes a shape and answers the coverage of the outline.
func strokeCoverage(shape geom.Shape, at *geom.AffineTransform, stroke *rendering.Stroke,
	width, height int, antiAliasing, normalize bool) *image.Alpha {
	recorder := newOutlineRecorder()
	dasher := newDasherFor(recorder, width, height, stroke)
	addShapeToAdder(dasher, shape, at, newNormalizer(normalize, antiAliasing))
	dasher.Draw()
	return recorder.rasterize(width, height, antiAliasing)
}

// addShapeToAdder walks a shape through a transform into a rasterx path,
// normalizing each segment on the way.
//
// Unlike the fill, an open subpath stays open: the cap goes on its ends, and
// closing it would put a join there instead.
func addShapeToAdder(adder rasterx.Adder, shape geom.Shape, at *geom.AffineTransform,
	normalize *normalizer) {
	iterator := shape.PathIterator(at)
	coords := make([]float64, 6)
	started := false

	for ; !iterator.IsDone(); iterator.Next() {
		kind := iterator.CurrentSegment(coords)
		// The path is already in device space, which is where Marlin
		// normalizes: it wraps the iterator the device transform came out of.
		normalize.segment(kind, coords)
		switch kind {
		case geom.SegMoveTo:
			if started {
				adder.Stop(false)
			}
			adder.Start(fixedPoint(coords[0], coords[1]))
			started = true
		case geom.SegLineTo:
			adder.Line(fixedPoint(coords[0], coords[1]))
		case geom.SegQuadTo:
			adder.QuadBezier(fixedPoint(coords[0], coords[1]), fixedPoint(coords[2], coords[3]))
		case geom.SegCubicTo:
			adder.CubeBezier(fixedPoint(coords[0], coords[1]),
				fixedPoint(coords[2], coords[3]), fixedPoint(coords[4], coords[5]))
		case geom.SegClose:
			if started {
				adder.Stop(true)
				started = false
			}
		}
	}
	if started {
		adder.Stop(false)
	}
}

// newDasherFor builds the stroker a stroke describes.
func newDasherFor(scanner rasterx.Scanner, width, height int,
	stroke *rendering.Stroke) *rasterx.Dasher {
	dasher := rasterx.NewDasher(width, height, scanner)
	var dashes []float64
	for _, d := range stroke.DashArray {
		dashes = append(dashes, float64(d))
	}
	// The gap function is nil on purpose. rasterx picks RoundGap for a round
	// join and FlatGap otherwise, and passing one here overrides that -- a
	// round join then renders as a bevel. See rasterx_test.go.
	dasher.SetStroke(
		fixed.Int26_6(stroke.LineWidth*64),
		fixed.Int26_6(stroke.MiterLimit*64),
		capFunc(stroke.LineCap), nil, nil, joinMode(stroke.LineJoin),
		dashes, float64(stroke.DashPhase))
	return dasher
}
