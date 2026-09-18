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
	"math"
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
