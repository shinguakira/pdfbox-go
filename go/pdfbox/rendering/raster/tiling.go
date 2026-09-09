package raster

// Tiling patterns.
//
// Port of rendering/TilingPaint.java, which is a java.awt.TexturePaint over an
// image of one tile. Java renders that tile with the PageDrawer it was handed
// and then lets TexturePaint repeat it; the port renders the tile onto a
// backend of its own -- this one, recursively -- and repeats it here, because
// there is no TexturePaint to hand it to.
//
// TilingPaintFactory is not ported. All it is is a WeakHashMap in front of the
// constructor, keyed on the matrix, the pattern dictionary, the colour and the
// transform; Go has no weak reference, and a cache that never releases is
// worse than none. What it buys is one render of a tile per distinct pattern
// per page, and what it costs to leave out is one render per fill.

import (
	goimage "image"
	goimagecolor "image/color"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// maxEdge is TilingPaint.MAXEDGE, which PDFBOX-3653 added to stop a pattern
// asking for a surface of billions of pixels.
//
// Java reads it from the system property
// `pdfbox.rendering.tilingpaint.maxedge`, defaulting to 3000. There is no such
// thing to read here, so it is the default.
const maxEdge = 3000

// tilingSource is a TexturePaint: one rendered tile, repeated over an anchor
// rectangle.
type tilingSource struct {
	tile   *goimage.RGBA
	anchor *geom.Rectangle2D
	// toAnchor maps a device pixel back into the space the anchor is in,
	// which is the inverse of what Java's createContext hands TexturePaint.
	toAnchor *geom.AffineTransform
}

// newTilingSource renders one tile and answers the paint that repeats it.
func (i *Image) newTilingSource(paint rendering.TilingPaint) (*tilingSource, error) {
	if paint.Drawer == nil || paint.PatternMatrix == nil {
		// Nothing but PageDrawer builds one of these, and it fills both.
		return nil, ErrNotDrawn
	}
	anchor, err := anchorRect(paint)
	if err != nil {
		return nil, err
	}
	tile, err := i.tileImage(paint, anchor)
	if err != nil {
		return nil, err
	}

	// createContext concatenates the pattern matrix with its own scaling taken
	// out, because the scaling is already in the tile:
	//
	//	AffineTransform patternNoScale = patternMatrix.createAffineTransform();
	//	patternNoScale.scale(1 / patternMatrix.getScalingFactorX(),
	//	                     1 / patternMatrix.getScalingFactorY());
	//	xformPattern.concatenate(patternNoScale);
	patternNoScale := paint.PatternMatrix.CreateAffineTransform()
	patternNoScale.Scale(1/float64(paint.PatternMatrix.ScalingFactorX()),
		1/float64(paint.PatternMatrix.ScalingFactorY()))
	toDevice := i.transform.Clone()
	toDevice.Concatenate(patternNoScale)

	toAnchor, err := toDevice.CreateInverse()
	if err != nil {
		// A singular transform paints nothing, which is what Java's
		// TexturePaint does with one: its context answers a blank raster.
		return nil, ErrNotDrawn
	}
	return &tilingSource{tile: tile, anchor: anchor, toAnchor: toAnchor}, nil
}

// anchorRect is getAnchorRect: the box one tile lands in, with the pattern
// matrix's scaling applied and the steps standing in for a zero.
func anchorRect(paint rendering.TilingPaint) (*geom.Rectangle2D, error) {
	bbox := paint.Pattern.BBox()
	if bbox == nil {
		return nil, ErrNoPatternBBox
	}
	xStep := paint.Pattern.XStep()
	if xStep == 0 {
		// "/XStep is 0, using pattern /BBox width"
		xStep = bbox.Width()
	}
	yStep := paint.Pattern.YStep()
	if yStep == 0 {
		yStep = bbox.Height()
	}

	xScale := paint.PatternMatrix.ScalingFactorX()
	yScale := paint.PatternMatrix.ScalingFactorY()
	width := xStep * xScale
	height := yStep * yScale

	if math.Abs(float64(width*height)) > maxEdge*maxEdge {
		// PDFBOX-3653: prevent huge sizes
		width = float32(math.Min(maxEdge, math.Abs(float64(width)))) * signum(width)
		height = float32(math.Min(maxEdge, math.Abs(float64(height)))) * signum(height)
	}

	return geom.NewRectangle2D(float64(bbox.LowerLeftX()*xScale),
		float64(bbox.LowerLeftY()*yScale), float64(width), float64(height)), nil
}

// signum is Math.signum, which answers zero for zero.
func signum(v float32) float32 {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	}
	return v
}

// tileImage is getImage: the pattern's content stream, run once, onto a
// surface the size of one tile.
func (i *Image) tileImage(paint rendering.TilingPaint,
	anchor *geom.Rectangle2D) (*goimage.RGBA, error) {
	width := float32(math.Abs(anchor.Width))
	height := float32(math.Abs(anchor.Height))

	// device scale transform (i.e. DPI) (see PDFBOX-1466.pdf)
	xformMatrix := util.NewMatrixFromAffineTransform(paint.Transform)
	xScale := float32(math.Abs(float64(xformMatrix.ScalingFactorX())))
	yScale := float32(math.Abs(float64(xformMatrix.ScalingFactorY())))
	width *= xScale
	height *= yScale

	rasterWidth := maxInt(1, tilingCeiling(float64(width)))
	rasterHeight := maxInt(1, tilingCeiling(float64(height)))

	tile := NewImage(rasterWidth, rasterHeight, rendering.ARGB)
	at := geom.NewAffineTransform(1, 0, 0, 1, 0, 0)

	// flip a -ve YStep around its own axis (see gs-bugzilla694385.pdf)
	if paint.Pattern.YStep() < 0 {
		at.Translate(0, float64(rasterHeight))
		at.Scale(1, -1)
	}
	// flip a -ve XStep around its own axis
	if paint.Pattern.XStep() < 0 {
		at.Translate(float64(rasterWidth), 0)
		at.Scale(-1, 1)
	}
	// device scale transform (i.e. DPI)
	at.Scale(float64(xScale), float64(yScale))
	tile.SetTransform(at)

	// Only the scaling from the pattern transform, "doing scaling here
	// improves the image quality and prevents large scale-down factors from
	// creating huge tiling cells", and then the origin moved to (0,0).
	newPatternMatrix := util.ScaleInstance(
		float32(math.Abs(float64(paint.PatternMatrix.ScalingFactorX()))),
		float32(math.Abs(float64(paint.PatternMatrix.ScalingFactorY()))))
	bbox := paint.Pattern.BBox()
	newPatternMatrix.Translate(-bbox.LowerLeftX(), -bbox.LowerLeftY())

	if err := paint.Drawer.DrawTilingPattern(tile, paint.Pattern, paint.ColorSpace,
		paint.Color, newPatternMatrix); err != nil {
		return nil, err
	}
	return tile.dst, nil
}

// tilingCeiling is TilingPaint.ceiling, and it is not a ceiling.
//
//	BigDecimal decimal = BigDecimal.valueOf(num);
//	decimal = decimal.setScale(5, RoundingMode.CEILING);
//	return decimal.intValue();
//
// Rounding up at the fifth decimal place and then truncating to an int leaves
// every value below the next whole number where it was: `ceiling(3.9)` is 3.
// What the method does is take the floor, with a tolerance of 1e-5 so that a
// width that should have been whole and came out as 2.999999999 counts as 3.
// Its name and its javadoc -- "the closest integer which is larger than the
// given number" -- describe something else. See migration/JAVA-BUGS.md.
func tilingCeiling(num float64) int {
	rounded := math.Ceil(num*1e5) / 1e5
	return int(rounded)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// colorAt answers the tile's colour under a device pixel, repeating.
func (t *tilingSource) colorAt(x, y int) (goimagecolor.RGBA, bool) {
	point := []float64{float64(x), float64(y)}
	t.toAnchor.TransformDoubles(point, 0, point, 0, 1)

	column, inside := t.wrap(point[0]-t.anchor.X, t.anchor.Width, t.tile.Bounds().Dx())
	if !inside {
		return goimagecolor.RGBA{}, false
	}
	row, inside := t.wrap(point[1]-t.anchor.Y, t.anchor.Height, t.tile.Bounds().Dy())
	if !inside {
		return goimagecolor.RGBA{}, false
	}

	c := t.tile.RGBAAt(column, row)
	if c.A == 0 {
		// Nothing of the tile is here, and a tile is drawn on nothing: the
		// pattern's own background shows through, which for a PDF is whatever
		// was already on the page.
		return goimagecolor.RGBA{}, false
	}
	return c, true
}

// wrap turns an offset along one axis of the anchor into a column or row of
// the tile, repeating the tile in both directions.
func (t *tilingSource) wrap(offset, span float64, pixels int) (int, bool) {
	if span == 0 || pixels == 0 {
		return 0, false
	}
	fraction := math.Mod(offset/span, 1)
	if fraction < 0 {
		fraction++
	}
	index := int(fraction * float64(pixels))
	if index < 0 {
		index = 0
	}
	if index >= pixels {
		index = pixels - 1
	}
	return index, true
}
