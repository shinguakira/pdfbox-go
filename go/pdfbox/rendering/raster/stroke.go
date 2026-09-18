package raster

// Turning a stroke into a shape.
//
// Java's Graphics2D.draw(shape) strokes with the current BasicStroke; the port
// asks rasterx's stroker for the outline and then fills it, which is what
// BasicStroke.createStrokedShape does and what Java2D does underneath.
//
// The stroke arrives in the user space of the Graphics2D, not in device space.
// PageDrawer.getStroke has put the line width and the dash phase through the
// CTM -- see its transformWidth calls -- but not through the transform of the
// Graphics2D, which is the page's own: the scale the page is rendered at, its
// rotation and the flip. Marlin puts the stroke through that transform as well
// as the path, so a page at 144 dpi is stroked twice as wide as one at 72, and
// so does this; see pen.

import (
	"image"
	"math"

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

	// outward takes the outline from the space it was stroked in to device
	// space, and is nil where that is device space already. See pen.
	outward *[4]float64
}

func newOutlineRecorder(outward *[4]float64) *outlineRecorder {
	return &outlineRecorder{first: true, outward: outward}
}

func (o *outlineRecorder) Start(a fixed.Point26_6) {
	a = o.toDevice(a)
	o.close()
	o.current = []fixed.Point26_6{a}
	o.grow(a)
}

func (o *outlineRecorder) Line(b fixed.Point26_6) {
	b = o.toDevice(b)
	o.current = append(o.current, b)
	o.grow(b)
}

// toDevice is Marlin's deltaTransformConsumer: the linear part of the
// transform, applied to the stroker's output.
func (o *outlineRecorder) toDevice(p fixed.Point26_6) fixed.Point26_6 {
	if o.outward == nil {
		return p
	}
	x, y := deltaTransform(o.outward, float64(p.X)/64, float64(p.Y)/64)
	return fixedPoint(x, y)
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

// strokeCoverage strokes a shape and answers the coverage of the outline, or
// nil where the transform lets nothing be drawn.
func strokeCoverage(shape geom.Shape, at *geom.AffineTransform, stroke *rendering.Stroke,
	width, height int, antiAliasing, normalize bool) *image.Alpha {
	pen := penThrough(at, stroke)
	if pen == nil {
		return nil
	}
	recorder := newOutlineRecorder(pen.outward)
	dasher := newDasherFor(recorder, width, height, stroke, pen)
	addShapeToAdder(dasher, shape, at, newNormalizer(normalize, antiAliasing), pen.inward)
	dasher.Draw()
	return recorder.rasterize(width, height, antiAliasing)
}

// pen is a stroke as the stroker is handed it.
//
// Port of the transform half of DMarlinRenderingEngine.strokeTo. Marlin looks
// at the transform the path is drawn through and does one of three things:
//
//   - through a transform that flattens the plane onto a line, it strokes
//     nothing, because whatever it drew would be squashed back to a line;
//   - through one that multiplies every length by the same amount -- a
//     uniform scale, with or without a rotation or a flip -- it strokes the
//     transformed path, with the width, the dashes and the phase multiplied by
//     that amount;
//   - through any other, it strokes the path in user space, where the pen is
//     round, and transforms the outline, which makes the pen an ellipse.
//
// The last is done here in a space that is not quite Marlin's, to the same
// result. rasterx strokes in 26.6 fixed point, and a unit of user space can be
// many pixels, so the stroke is made in user space enlarged by the largest
// factor the transform scales anything by: the same stroke, a similarity
// larger, in which a sixty-fourth of a unit is never more than a sixty-fourth
// of a pixel.
type pen struct {
	width  float64
	dashes []float64
	phase  float64

	// inward takes a point from device space into the space the stroke is made
	// in, and outward brings the outline back. They are the linear parts only,
	// m00 m01 m10 m11, as Marlin's inverseDeltaTransformConsumer and
	// deltaTransformConsumer are, and both are nil where the stroke is made in
	// device space.
	inward, outward *[4]float64
}

// penThrough answers the pen a stroke is drawn with through the given
// transform, or nil where nothing can be drawn through it.
func penThrough(at *geom.AffineTransform, stroke *rendering.Stroke) *pen {
	p := &pen{width: float64(stroke.LineWidth), phase: float64(stroke.DashPhase)}
	for _, d := range stroke.DashArray {
		p.dashes = append(p.dashes, float64(d))
	}
	if at == nil || at.IsIdentity() {
		return p
	}
	a := at.ScaleX()
	b := at.ShearX()
	c := at.ShearY()
	d := at.ScaleY()
	det := a*d - c*b

	if math.Abs(det) <= 2*math.SmallestNonzeroFloat64 {
		// this rendering engine takes one dimensional curves and turns
		// them into 2D shapes by giving them width.
		// However, if everything is to be passed through a singular
		// transformation, these 2D shapes will be squashed down to 1D
		// again so, nothing can be drawn.
		return nil
	}

	// If the transform is a constant multiple of an orthogonal transformation
	// then every length is just multiplied by a constant, so we just
	// need to transform input paths to stroker and tell stroker
	// the scaled width. This condition is satisfied if
	// a*b == -c*d && a*a+c*c == b*b+d*d. In the actual check below, we
	// leave a bit of room for error.
	if nearZero(a*b+c*d) && nearZero(a*a+c*c-(b*b+d*d)) {
		p.scale(math.Sqrt(a*a + c*c))
		return p
	}

	s := maximumScale(at)
	p.scale(s)
	p.outward = &[4]float64{a / s, b / s, c / s, d / s}
	p.inward = &[4]float64{d * s / det, -b * s / det, -c * s / det, a * s / det}
	return p
}

// scale multiplies every length of the pen by the same factor.
func (p *pen) scale(factor float64) {
	p.width *= factor
	if p.dashes != nil {
		for i := range p.dashes {
			p.dashes[i] *= factor
		}
		p.phase *= factor
	}
}

// maximumScale answers the most the transform lengthens anything, which is the
// largest of its singular values; 1 for no transform.
//
// It is the "maximum scale" DMarlinRenderingEngine.userSpaceLineWidth and
// SunGraphics2D.validateBasicStroke work out, with the square root taken.
func maximumScale(at *geom.AffineTransform) float64 {
	if at == nil {
		return 1
	}
	A := at.ScaleX() // m00
	C := at.ShearX() // m01
	B := at.ShearY() // m10
	D := at.ScaleY() // m11

	EA := A*A + B*B         // x^2 coefficient
	EB := 2.0 * (A*C + B*D) // xy coefficient
	EC := C*C + D*D         // y^2 coefficient

	hypot := math.Sqrt(EB*EB + (EA-EC)*(EA-EC))
	return math.Sqrt((EA + EC + hypot) / 2.0)
}

// nearZero is Marlin's: |num| < 2 * Math.ulp(num).
func nearZero(num float64) bool {
	return math.Abs(num) < 2*ulp(num)
}

// ulp is java.lang.Math.ulp(double): the distance from the magnitude of x to
// the next larger double.
func ulp(x float64) float64 {
	x = math.Abs(x)
	switch {
	case math.IsNaN(x):
		return x
	case math.IsInf(x, 0):
		return math.Inf(1)
	case x == math.MaxFloat64:
		// Java answers the spacing below it, 2^971, rather than infinity.
		return math.Ldexp(1, 971)
	}
	return math.Nextafter(x, math.Inf(1)) - x
}

// deltaTransform applies the linear part m00 m01 m10 m11 to a point.
func deltaTransform(m *[4]float64, x, y float64) (float64, float64) {
	return m[0]*x + m[1]*y, m[2]*x + m[3]*y
}

// addShapeToAdder walks a shape through a transform into a rasterx path,
// normalizing each segment on the way, and through inward where the stroke is
// not made in device space.
//
// Unlike the fill, an open subpath stays open: the cap goes on its ends, and
// closing it would put a join there instead.
func addShapeToAdder(adder rasterx.Adder, shape geom.Shape, at *geom.AffineTransform,
	normalize *normalizer, inward *[4]float64) {
	iterator := shape.PathIterator(at)
	coords := make([]float64, 6)
	started := false

	for ; !iterator.IsDone(); iterator.Next() {
		kind := iterator.CurrentSegment(coords)
		// The path is already in device space, which is where Marlin
		// normalizes: it wraps the iterator the device transform came out of.
		normalize.segment(kind, coords)
		if inward != nil {
			// Marlin's inverseDeltaTransformConsumer, which comes after the
			// normalizing iterator.
			for k := 0; k+1 < len(coords); k += 2 {
				coords[k], coords[k+1] = deltaTransform(inward, coords[k], coords[k+1])
			}
		}
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
//
// The lengths are the pen's -- the width, the dashes and the phase as the
// transform left them -- and the rest is the stroke's: the miter limit is a
// ratio, and no transform changes it.
func newDasherFor(scanner rasterx.Scanner, width, height int,
	stroke *rendering.Stroke, pen *pen) *rasterx.Dasher {
	dasher := rasterx.NewDasher(width, height, scanner)
	// The gap function is nil on purpose. rasterx picks RoundGap for a round
	// join and FlatGap otherwise, and passing one here overrides that -- a
	// round join then renders as a bevel. See rasterx_test.go.
	dasher.SetStroke(
		fixed.Int26_6(pen.width*64),
		fixed.Int26_6(stroke.MiterLimit*64),
		capFunc(stroke.LineCap), nil, nil, joinMode(stroke.LineJoin),
		pen.dashes, pen.phase)
	return dasher
}
