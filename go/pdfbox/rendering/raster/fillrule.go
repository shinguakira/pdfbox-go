package raster

// Where a fill lands on pixels with antialiasing off.
//
// Java2D has no one answer. A fill with antialiasing off goes to one of two
// fillers, both of them native, and which one it goes to depends on the
// stroke in force -- sun.java2d.pipe.LoopPipe.fill sends the shape to
// FillPath where SunGraphics2D's strokeState is STROKE_THIN, and to a
// ShapeSpanIterator otherwise. The two do not fill the same pixels:
//
//   - FillPath is ProcessPath.c. Its own comment says it supports "pixels at
//     centers" and "pixels at corners", that corners are the default and mean
//     "straightforward mapping (x,y) --> (x,y)", and that VALUE_STROKE_PURE
//     takes half a pixel off every coordinate to put them at centres. So under
//     the hint PDFBox leaves alone, a pixel is filled where the shape covers
//     its top left corner. Its spans bear that out: a row is drawn for each
//     whole y, and from `ceil(xLeft)` to `ceil(xRight) - 1`.
//   - ShapeSpanIterator.c samples pixel centres, `ceil(y - 0.5)`, and unless
//     the hint is PURE it first rounds every coordinate to the nearest
//     quarter, `floor(v + 0.25) + 0.25`, which its own comment calls
//     "normalize to nearest (0.25, 0.25)". A curve's control points move with
//     the endpoints either side of them, as they do for a stroke.
//
// This backend rasterises at pixel centres, so each of those is something done
// to the path before it is rasterised: half a pixel down and to the right for
// the first, that rounding for the second, and nothing at all under PURE.
//
// It matters because PDFBox fills every rectangular path with antialiasing off
// -- PDFBOX-2302 -- so this decides the edge of every filled rectangle whose
// edge falls between pixel centres, and of every tile of a scaled tiling
// pattern. The cases named fillFractional in testdata/java2d.txt are Java2D's
// own answers to all four of these.
//
// A stroke drawn with antialiasing off is not this: Java2D draws a thin one
// with a line algorithm of its own, doDrawPath, and the port fills the
// outline. See migration/STATUS.md.

import (
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// fillAdjustment is what a fill does to its path before rasterising it.
type fillAdjustment int

const (
	// fillAsGiven is the path as it is: antialiased, or VALUE_STROKE_PURE,
	// where both fillers sample pixel centres.
	fillAsGiven fillAdjustment = iota
	// fillAtCorners is FillPath's default, a pixel taken at its corner.
	fillAtCorners
	// fillAtQuarters is ShapeSpanIterator's rounding.
	fillAtQuarters
)

// fillAdjustmentOf is which of them a fill gets here and now.
func (i *Image) fillAdjustmentOf() fillAdjustment {
	if i.antiAliasing || !i.strokeNormalization {
		return fillAsGiven
	}
	if i.thinStroke() {
		return fillAtCorners
	}
	return fillAtQuarters
}

// The pen sizes SunGraphics2D compares a stroke against.
const (
	minPenSizeSquared = 1.000000001
	// MinPenSizeAA is RenderingEngine.getMinimumAAPenSize, which Marlin
	// answers as one over the fewest subpixels it has across a pixel, 8.
	minPenSizeAA        = 1.0 / 8
	minPenSizeAASquared = minPenSizeAA * minPenSizeAA
)

// thinStroke is SunGraphics2D.validateBasicStroke's STROKE_THIN, which is the
// state, and only the state, that sends a fill to FillPath.
//
// A dashed stroke of any width is STROKE_THINDASHED, which is not it. A stroke
// through a transform that scales is measured against the most that transform
// lengthens anything, which is what the EA, EC and hypot terms below work out;
// for a transform that scales the same both ways that is the determinant, and
// the two agree, so the port takes the one expression.
//
// The stroke is whatever was last set, because that is what Java2D has: a fill
// asks the graphics for its stroke state, and PDFBox sets a stroke only when
// it strokes. A page that strokes a wide line and then fills a rectangle fills
// it by the other rule.
func (i *Image) thinStroke() bool {
	width, dashed := 1.0, false
	// Java's defaultStroke, which a Graphics2D starts with, is a BasicStroke
	// of width 1 with no dashes.
	isDefault := i.stroke == nil
	if i.stroke != nil {
		width = float64(i.stroke.LineWidth)
		dashed = len(i.stroke.DashArray) > 0
	}
	if dashed {
		return false
	}

	if !scalesAnything(i.transform) {
		// transformState < TRANSFORM_TRANSLATESCALE
		if i.antiAliasing {
			return width <= minPenSizeAA
		}
		return isDefault || width <= 1
	}

	widthSquared := maximumScaleSquared(i.transform)
	if !isDefault {
		widthSquared *= width * width
	}
	if i.antiAliasing {
		return widthSquared <= minPenSizeAASquared
	}
	return widthSquared <= minPenSizeSquared
}

// scalesAnything reports whether the transform is more than a translation,
// which is where SunGraphics2D stops measuring a stroke by its width alone.
func scalesAnything(at *geom.AffineTransform) bool {
	if at == nil {
		return false
	}
	return at.ScaleX() != 1 || at.ScaleY() != 1 || at.ShearX() != 0 || at.ShearY() != 0
}

// maximumScaleSquared is the square of the most a transform lengthens
// anything, as validateBasicStroke works it out:
//
//	double EA = A*A + B*B;          // x^2 coefficient
//	double EB = 2*(A*C + B*D);      // xy coefficient
//	double EC = C*C + D*D;          // y^2 coefficient
//	double hypot = Math.sqrt(EB*EB + (EA-EC)*(EA-EC));
//	widthsquared = ((EA + EC + hypot)/2.0);
func maximumScaleSquared(at *geom.AffineTransform) float64 {
	if at == nil {
		return 1
	}
	a, b := at.ScaleX(), at.ShearY()
	c, d := at.ShearX(), at.ScaleY()
	ea := a*a + b*b
	eb := 2 * (a*c + b*d)
	ec := c*c + d*d
	hypot := math.Sqrt(eb*eb + (ea-ec)*(ea-ec))
	return (ea + ec + hypot) / 2
}

// adjust moves a walked path to where the filler of the moment reads it.
//
// The rounding is float32 because Java2D's is: ProcessPath is handed a
// Path2D.Float, and ShapeSpanIterator reads a path iterator's float
// coordinates, so a coordinate is already narrowed before either sees it.
func (p *walkedPath) adjust(adjustment fillAdjustment) {
	switch adjustment {
	case fillAtCorners:
		for k := range p.coords {
			p.coords[k] += 0.5
		}
		p.minX, p.minY = p.minX+0.5, p.minY+0.5
		p.maxX, p.maxY = p.maxX+0.5, p.maxY+0.5

	case fillAtQuarters:
		p.adjustToQuarters()
	}
}

// adjustToQuarters is ShapeSpanIterator's _ADJUST, ADJUST1, ADJUST2 and
// ADJUST3 over a whole path: each endpoint to the nearest quarter, a
// quadratic's control point by the average of the adjustment either side of
// it, and a cubic's two by the adjustment of the endpoint each belongs to.
func (p *walkedPath) adjustToQuarters() {
	// pd->adjx and pd->adjy, the adjustment the endpoint before this one was
	// given.
	var adjX, adjY float64
	at := 0
	for _, kind := range p.kinds {
		switch kind {
		case geom.SegMoveTo, geom.SegLineTo:
			adjX, adjY = p.quarterPoint(at)

		case geom.SegQuadTo:
			newAdjX, newAdjY := p.quarterPoint(at + 2)
			// x1 += (pd->adjx + newadjy) / 2;
			// y1 += (pd->adjy + newadjy) / 2;
			//
			// The x of a quadratic's control point moves by the *y* of the
			// new adjustment. That is the JDK's, not a slip here: the macro
			// reads newadjy on both lines. A quadratic is rare in a fill with
			// antialiasing off, and the port carries what Java2D does.
			p.coords[at] += (adjX + newAdjY) / 2
			p.coords[at+1] += (adjY + newAdjY) / 2
			adjX, adjY = newAdjX, newAdjY

		case geom.SegCubicTo:
			// The first control point moves with the endpoint before it and
			// the second with the endpoint after, which keeps both tangents
			// parallel to what they were.
			priorX, priorY := adjX, adjY
			newAdjX, newAdjY := p.quarterPoint(at + 4)
			p.coords[at] += priorX
			p.coords[at+1] += priorY
			p.coords[at+2] += newAdjX
			p.coords[at+3] += newAdjY
			adjX, adjY = newAdjX, newAdjY
		}
		at += 2 * pointsOf(kind)
	}
	p.remeasure()
}

// quarterPoint rounds the point at the given offset to the nearest quarter and
// answers how far it moved.
func (p *walkedPath) quarterPoint(at int) (float64, float64) {
	x, y := p.coords[at], p.coords[at+1]
	newX, newY := nearestQuarter(x), nearestQuarter(y)
	p.coords[at], p.coords[at+1] = newX, newY
	return newX - x, newY - y
}

// nearestQuarter is `(jfloat) floor(x + 0.25f) + 0.25f`.
func nearestQuarter(v float64) float64 {
	shifted := float32(v) + 0.25
	return float64(float32(math.Floor(float64(shifted))) + 0.25)
}

// remeasure puts the path's box back round its points, which an adjustment
// that moves them one at a time can leave behind.
func (p *walkedPath) remeasure() {
	p.minX, p.minY = math.Inf(1), math.Inf(1)
	p.maxX, p.maxY = math.Inf(-1), math.Inf(-1)
	for k := 0; k+1 < len(p.coords); k += 2 {
		p.minX = math.Min(p.minX, p.coords[k])
		p.maxX = math.Max(p.maxX, p.coords[k])
		p.minY = math.Min(p.minY, p.coords[k+1])
		p.maxY = math.Max(p.maxY, p.coords[k+1])
	}
}
