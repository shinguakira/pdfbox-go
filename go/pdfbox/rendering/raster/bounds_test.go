package raster

// The part of the surface a fill or a stroke is composed over.
//
// Fill and Draw do not walk the whole surface. They compose over the shape's
// bounding box through the transform, grown by what the stroke can add to it
// -- see deviceBounds -- and that box is only a bound: if it were ever smaller
// than the ink, the ink would be cut off at its edge with nothing to say so.
// Each case here is drawn twice, once through Fill or Draw and once composed
// over the whole surface from the same coverage, and the two are held to the
// same pixels.
//
// There is no Java to compare against, because Java2D composes over whatever
// the rasteriser covered; the bound is this backend's own.

import (
	goimage "image"
	"math"
	"strconv"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// TestAStrokeIsNotCutOffByItsBounds holds Draw's padded box to the stroke the
// stroker drew.
//
// The first case is the one a review of the bound asked about: a wide line
// through a transform that shrinks it, with the miter limit at its least, so
// that nothing but the width pads the box. The stroker and the pad have to
// agree about which space that width is in -- the stroker used to draw it in
// device space as it came, 100 pixels wide, while the pad took it through the
// transform to 10, and all but the middle of the line was lost.
func TestAStrokeIsNotCutOffByItsBounds(t *testing.T) {
	for _, c := range []struct {
		name      string
		transform *geom.AffineTransform
		stroke    *rendering.Stroke
		shape     geom.Shape
	}{
		{"shrunk", geom.NewAffineTransform(0.1, 0, 0, 0.1, 0, 0),
			&rendering.Stroke{LineWidth: 100, MiterLimit: 1}, lineShape(200, 300, 400, 300)},
		// A scale that differs each way, where the pen is an ellipse, with
		// square caps on a diagonal.
		{"nonUniform", geom.NewAffineTransform(4, 0, 0, 0.25, 0, 0),
			&rendering.Stroke{LineWidth: 4, LineCap: 2, MiterLimit: 1}, lineShape(4, 40, 10, 120)},
		// A rotation and a scale, and a miter join nearly as long as the limit
		// allows.
		{"rotatedMiter", rotation(30, 1.5, 32, 32),
			&rendering.Stroke{LineWidth: 2, MiterLimit: 10}, sharpAngle()},
		// No transform, and a square cap's corner, which reaches further than
		// half the width.
		{"squareCap", nil,
			&rendering.Stroke{LineWidth: 20, LineCap: 2, MiterLimit: 1}, lineShape(20, 20, 40, 40)},
	} {
		t.Run(c.name, func(t *testing.T) {
			drawn := boundsCase(c.transform)
			drawn.SetStroke(c.stroke)
			must(drawn.Draw(c.shape))

			whole := boundsCase(c.transform)
			bounds := whole.dst.Bounds()
			must(whole.compose(strokeCoverage(c.shape, c.transform, c.stroke,
				bounds.Dx(), bounds.Dy(), whole.antiAliasing, whole.strokeNormalization)))

			sameInk(t, drawn, whole)
		})
	}
}

// TestAFillIsNotCutOffByItsBounds is the same for Fill, whose box is not
// padded at all: a fill covers nothing outside its own outline.
func TestAFillIsNotCutOffByItsBounds(t *testing.T) {
	transform := geom.NewAffineTransform(2, 0.5, -1, 0.5, 30, 10)
	shape := geom.NewEllipse2D(0, 0, 16, 40)

	drawn := boundsCase(transform)
	must(drawn.Fill(shape))

	whole := boundsCase(transform)
	bounds := whole.dst.Bounds()
	must(whole.compose(coverageOf(shape, transform, bounds.Dx(), bounds.Dy(), whole.antiAliasing)))

	sameInk(t, drawn, whole)
}

// boundsCase is a white surface with black paint, anti-aliased and
// normalizing, as a page is drawn.
func boundsCase(transform *geom.AffineTransform) *Image {
	i := NewImage(64, 64, rendering.RGB)
	i.SetPaint(black)
	i.SetAntiAliasing(true)
	i.SetStrokeNormalization(true)
	i.SetTransform(transform)
	return i
}

// sameInk fails unless both surfaces hold the same pixels, and unless there is
// ink on them at all.
func sameInk(t *testing.T, drawn, whole *Image) {
	t.Helper()
	bounds := whole.dst.Bounds()
	inked, cut := 0, 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if whole.dst.NRGBAAt(x, y).R != 0xFF {
				inked++
			}
			if drawn.dst.NRGBAAt(x, y) != whole.dst.NRGBAAt(x, y) {
				cut++
			}
		}
	}
	if inked == 0 {
		t.Fatal("the case drew nothing, so it says nothing about the bound")
	}
	if cut != 0 {
		t.Errorf("%d of the %d inked pixels are not drawn inside the bound", cut, inked)
	}
}

// rotation is a rotation by the given degrees and a uniform scale, about the
// origin, then moved to (tx, ty).
func rotation(degrees, scale, tx, ty float64) *geom.AffineTransform {
	sin, cos := math.Sincos(degrees * math.Pi / 180)
	return geom.NewAffineTransform(scale*cos, scale*sin, -scale*sin, scale*cos, tx, ty)
}

// sharpAngle is two arms meeting at an angle of about 19 degrees, whose miter
// is six times half the width long.
func sharpAngle() geom.Shape {
	path := geom.NewPathDouble()
	path.MoveTo(-6, -2)
	path.LineTo(6, 0)
	path.LineTo(-6, 2)
	return path
}

// TestACoverageMaskHoldsWhatTheWholeSurfaceWould holds the masks coverageOf and
// strokeCoverage make, which cover only what a shape reaches, to the mask the
// same rasteriser makes over the whole surface.
//
// Two things about freetype's rasteriser decide the bound, and each has a case
// here. It finds a coordinate's pixel by dividing by 64 with Go's truncating
// division, so a shape a fraction of a pixel off the top or the left edge
// still draws on the first row or column. And it decides how finely to split
// a cubic from where the curve is on the surface, not only from its shape,
// which is why the shape is rasterised where it is and never moved to the
// corner of its mask: moved, the circles below draw different edges.
func TestACoverageMaskHoldsWhatTheWholeSurfaceWould(t *testing.T) {
	const width, height = 64, 48
	shapes := map[string]geom.Shape{
		"offTheTop":   geom.NewRectangle2D(10.25, -0.75, 20, 0.5),
		"offTheLeft":  geom.NewEllipse2D(-0.9, 10.5, 0.8, 12),
		"pastTheEdge": geom.NewEllipse2D(50.3, 30.7, 30, 30),
		"acrossAll":   geom.NewRectangle2D(-10.5, -10.5, 90.25, 70.25),
		"offTheRight": geom.NewRectangle2D(64.5, 10, 5, 5),
	}
	for n := 0; n < 40; n++ {
		// circles at many places and sizes, for the cubics
		x := 3.1 + float64(n*37%53) + float64(n)/7
		y := 2.7 + float64(n*23%41) + float64(n)/11
		shapes["circle"+strconv.Itoa(n)] = geom.NewEllipse2D(x, y, 1.5+float64(n%9), 1.2+float64(n%7))
	}
	stroke := &rendering.Stroke{LineWidth: 1.7, LineCap: 1, LineJoin: 1, MiterLimit: 10}

	for name, shape := range shapes {
		path := walkShape(shape, nil)
		r := borrowRasterizer(width, height)
		path.addTo(r.Rasterizer)
		r.UseNonZeroWinding = path.rule == geom.WindNonZero
		whole := rasterizeInto(r.Rasterizer, goimage.Rect(0, 0, width, height), true)
		rasterizers.Put(r)
		walkedPaths.Put(path)
		sameCoverage(t, name+" filled", coverageOf(shape, nil, width, height, true), whole)

		recorder := newOutlineRecorder(nil)
		pen := penThrough(nil, stroke)
		dasher := newDasherFor(recorder, width, height, stroke, pen)
		addShapeToAdder(dasher, shape, nil, newNormalizer(true, true), nil)
		dasher.Draw()
		r = borrowRasterizer(width, height)
		r.UseNonZeroWinding = true
		for _, polygon := range recorder.polygons {
			r.Start(polygon[0])
			for _, point := range polygon[1:] {
				r.Add1(point)
			}
		}
		whole = rasterizeInto(r.Rasterizer, goimage.Rect(0, 0, width, height), true)
		rasterizers.Put(r)
		sameCoverage(t, name+" stroked",
			strokeCoverage(shape, nil, stroke, width, height, true, true), whole)
	}
}

// sameCoverage fails unless the bounded mask answers what the whole one does at
// every pixel of the surface.
func sameCoverage(t *testing.T, what string, bounded, whole *goimage.Alpha) {
	t.Helper()
	differing := 0
	for y := whole.Rect.Min.Y; y < whole.Rect.Max.Y; y++ {
		for x := whole.Rect.Min.X; x < whole.Rect.Max.X; x++ {
			if bounded.AlphaAt(x, y) != whole.AlphaAt(x, y) {
				differing++
			}
		}
	}
	if differing != 0 {
		t.Errorf("%s: %d pixels of the bounded mask are not the whole surface's", what, differing)
	}
}
