package raster

// Stroke normalization.
//
// Port of `sun.java2d.marlin.MarlinRenderingEngine.NormalizingPathIterator`,
// which is the JDK's and not PDFBox's. It is here because PDFBox gets it
// whether it asks or not: `PDFRenderer.createDefaultRenderingHints` sets three
// hints and **not** `KEY_STROKE_CONTROL`, so a page renders under the
// Graphics2D default, and the default normalizes.
//
// What it does is move each segment's endpoint onto a pixel centre before the
// path is stroked, so that a thin line lands on whole pixels instead of
// straddling two. The control points of a curve move with the endpoints on
// either side of them, so the curve keeps its shape.
//
// It applies to **strokes only**. A fill goes through the rasteriser
// untouched, in Java and here -- see `NormMode`, which `strokeTo` chooses and
// nothing else does.
//
//	ON_WITH_AA -> NearestPixelCenter,  floor(c) + 0.5
//	ON_NO_AA   -> NearestPixelQuarter, floor(c + 0.25) + 0.25
//	OFF        -> the path as given
//
// The arithmetic is float32 because Marlin's is: it takes the `float[]`
// overload of `currentSegment`, so a double path is narrowed on the way in.

import (
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// normalizer is one walk of a path, adjusting each segment as Marlin's
// NormalizingPathIterator does.
//
// It is stateful across segments in two ways, both of which Java keeps in the
// same fields: the adjustment made to the previous endpoint, which a curve's
// first control point moves by, and the adjustment made at the last `moveTo`,
// which a `closePath` restores so that the subpath's last segment lands where
// its first began.
type normalizer struct {
	// mode says which pixel a coordinate is drawn to, and is nil where the
	// path is left alone.
	normCoord func(float32) float32

	curXAdjust, curYAdjust float32
	movXAdjust, movYAdjust float32
}

// newNormalizer answers the normalizer a stroke is drawn through, or one that
// does nothing where normalization is off.
//
//	final NormMode norm = (normalize) ?
//	        ((antialias) ? NormMode.ON_WITH_AA : NormMode.ON_NO_AA)
//	        : NormMode.OFF;
func newNormalizer(normalize, antiAliasing bool) *normalizer {
	if !normalize {
		return &normalizer{}
	}
	if antiAliasing {
		return &normalizer{normCoord: nearestPixelCenter}
	}
	return &normalizer{normCoord: nearestPixelQuarter}
}

// nearestPixelCenter is `FloatMath.floor_f(coord) + 0.5f`.
func nearestPixelCenter(coord float32) float32 {
	return float32(math.Floor(float64(coord))) + 0.5
}

// nearestPixelQuarter is `FloatMath.floor_f(coord + 0.25f) + 0.25f`.
func nearestPixelQuarter(coord float32) float32 {
	return float32(math.Floor(float64(coord+0.25))) + 0.25
}

// segment adjusts one segment in place and answers it.
//
// coords is what CurrentSegment filled in, and kind which of the five it is.
// The port passes float64 because that is what the path iterator speaks; the
// adjustments are worked out in float32, as Java's are.
func (n *normalizer) segment(kind int, coords []float64) {
	if n.normCoord == nil {
		return
	}

	// Which pair of the six is the endpoint. Java switches on the type and
	// throws InternalError on anything else; there is nothing else.
	var lastCoord int
	switch kind {
	case geom.SegMoveTo, geom.SegLineTo:
		lastCoord = 0
	case geom.SegQuadTo:
		lastCoord = 2
	case geom.SegCubicTo:
		lastCoord = 4
	case geom.SegClose:
		// "we don't want to deal with this case later. We just exit now"
		n.curXAdjust = n.movXAdjust
		n.curYAdjust = n.movYAdjust
		return
	default:
		return
	}

	// normalize endpoint
	x := float32(coords[lastCoord])
	xAdjust := n.normCoord(x) - x
	y := float32(coords[lastCoord+1])
	yAdjust := n.normCoord(y) - y

	coords[lastCoord] = float64(x + xAdjust)
	coords[lastCoord+1] = float64(y + yAdjust)

	// now that the end points are done, normalize the control points
	switch kind {
	case geom.SegMoveTo:
		n.movXAdjust = xAdjust
		n.movYAdjust = yAdjust
	case geom.SegLineTo:
		// nothing to move
	case geom.SegQuadTo:
		coords[0] += float64((n.curXAdjust + xAdjust) / 2)
		coords[1] += float64((n.curYAdjust + yAdjust) / 2)
	case geom.SegCubicTo:
		coords[0] += float64(n.curXAdjust)
		coords[1] += float64(n.curYAdjust)
		coords[2] += float64(xAdjust)
		coords[3] += float64(yAdjust)
	}

	n.curXAdjust = xAdjust
	n.curYAdjust = yAdjust
}
