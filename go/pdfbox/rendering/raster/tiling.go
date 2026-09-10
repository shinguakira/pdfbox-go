package raster

// Tiling patterns.
//
// Port of rendering/TilingPaint.java, which is a java.awt.TexturePaint over an
// image of one tile. Java renders that tile with the PageDrawer it was handed
// and then lets TexturePaint repeat it; the port renders the tile onto a
// backend of its own -- this one, recursively -- and repeats it here, because
// there is no TexturePaint to hand it to.
//
// TilingPaintFactory, the WeakHashMap in front of the constructor, is
// tilingcache.go. What it buys is one render of a tile per distinct pattern per
// page, and Go has no weak reference, so the lifetime is written down there
// instead of inferred from one.

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
	tile   *goimage.NRGBA
	anchor *geom.Rectangle2D
	// toAnchor maps a device pixel back into the space the anchor is in,
	// which is the inverse of what Java's createContext hands TexturePaint.
	toAnchor *geom.AffineTransform

	// filter is Java's TexturePaintContext `filter`, which KEY_INTERPOLATION
	// decides and PDFRenderer sets to BICUBIC.
	filter bool
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
	return &tilingSource{
		tile:     tile,
		anchor:   anchor,
		toAnchor: toAnchor,
		filter:   i.interpolation != rendering.NearestNeighbor,
	}, nil
}

// anchorRect is getAnchorRect: the box one tile lands in, with the pattern
// matrix's scaling applied and the steps standing in for a zero.
func anchorRect(paint rendering.TilingPaint) (*geom.Rectangle2D, error) {
	bbox := paint.Pattern.BBox()
	if bbox == nil {
		return nil, ErrNoPatternBBox
	}
	xStep := paint.Pattern.XStep()
	if isPositiveZero(xStep) {
		// "/XStep is 0, using pattern /BBox width"
		xStep = bbox.Width()
	}
	yStep := paint.Pattern.YStep()
	if isPositiveZero(yStep) {
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

// isPositiveZero is `Float.compare(v, 0) == 0`, which is not `v == 0`.
//
// Float.compare orders -0.0 below +0.0 and NaN above everything, so it answers
// zero for +0.0 alone. `v == 0` in Go is true for -0.0 as well, and a pattern
// whose /XStep is written `-0` would then take the bbox width here and keep the
// negative zero in Java -- which getImage reads again, as `getXStep() < 0`,
// where -0.0 is not less than zero either. The difference is one document in a
// million and one line to be faithful about.
func isPositiveZero(v float32) bool {
	return v == 0 && !math.Signbit(float64(v))
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
	anchor *geom.Rectangle2D) (*goimage.NRGBA, error) {
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

// tilingCeiling is TilingPaint.ceiling, **corrected**, and it is the one place
// in this package where the Go deliberately does not do what the Java does.
// See migration/JAVA-BUGS.md 85.
//
// Java is
//
//	BigDecimal decimal = BigDecimal.valueOf(num);
//	decimal = decimal.setScale(5, RoundingMode.CEILING);
//	return decimal.intValue();
//
// which rounds up at the fifth decimal place and then **truncates**, so every
// value below the next whole number stays where it was and `ceiling(3.9)` is 3.
// Its javadoc asks for two things --
//
//	Returns the closest integer which is larger than the given number.
//	Uses BigDecimal to avoid floating point error which would cause gaps in
//	the tiling.
//
// -- and the truncation satisfies the second and not the first. Rounding up at
// the fifth decimal place and then taking the ceiling satisfies both: 3.9
// becomes 4, and a width that should have been whole and came out as
// 2.999999999 stays 3 rather than buying an extra pixel from a float error.
//
// What it costs is that a tiling pattern is rasterized at the size it is drawn
// at rather than up to a pixel smaller in each direction, so a page of them
// does not match PDFBox's pixel for pixel. That is measured, in
// TestTilingPatternsRenderAsPDFBoxRendersThem.
func tilingCeiling(num float64) int {
	tolerated := math.Ceil(num*1e5) / 1e5
	return int(math.Ceil(tolerated))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// colorAt answers the tile's colour under a device pixel, repeating.
//
// The pixel's **corner** is what is mapped back, and not its centre: Java's
// PaintContext is handed `getRaster(x1, y1, w, h)` in device coordinates and
// TexturePaint reads each at the coordinate it is given. Sampling the centre
// instead was tried and takes the patterns page from 525 differing pixels
// against PDFBox to 3876.
//
// Whether the four texels around that point are blended is Java's `filter`
// flag: TexturePaintContext.getContext takes it from KEY_INTERPOLATION, and
// PDFRenderer.createDefaultRenderingHints sets that to BICUBIC, so a page
// renders with it on. It shows only where a tile does not land on whole pixels
// -- at 1:1 the sample falls on a texel corner and the blend answers that
// texel -- which is why a page of unscaled patterns is exact either way.
func (t *tilingSource) colorAt(x, y int) (goimagecolor.NRGBA, bool) {
	point := []float64{float64(x), float64(y)}
	t.toAnchor.TransformDoubles(point, 0, point, 0, 1)

	column, columnWeight, inside := t.wrap(point[0]-t.anchor.X, t.anchor.Width,
		t.tile.Bounds().Dx())
	if !inside {
		return goimagecolor.NRGBA{}, false
	}
	row, rowWeight, inside := t.wrap(point[1]-t.anchor.Y, t.anchor.Height,
		t.tile.Bounds().Dy())
	if !inside {
		return goimagecolor.NRGBA{}, false
	}

	c := t.tile.NRGBAAt(column, row)
	if t.filter {
		c = t.blend(column, columnWeight, row, rowWeight)
	}
	if c.A == 0 {
		// Nothing of the tile is here, and a tile is drawn on nothing: the
		// pattern's own background shows through, which for a PDF is whatever
		// was already on the page.
		return goimagecolor.NRGBA{}, false
	}
	return c, true
}

// blend is the four-texel average TexturePaintContext.Any makes when its
// `filter` is on: the texel the sample landed in, the one after it along each
// axis, and the one diagonally after, weighted by how far into its own texel
// the sample fell.
//
// "The one after" wraps, because the texture repeats -- the texel after the
// last column is the first column of the next tile, and it is the tile's own
// first column.
func (t *tilingSource) blend(column int, columnWeight float64,
	row int, rowWeight float64) goimagecolor.NRGBA {
	width, height := t.tile.Bounds().Dx(), t.tile.Bounds().Dy()
	nextColumn := (column + 1) % width
	nextRow := (row + 1) % height

	topLeft := t.tile.NRGBAAt(column, row)
	topRight := t.tile.NRGBAAt(nextColumn, row)
	bottomLeft := t.tile.NRGBAAt(column, nextRow)
	bottomRight := t.tile.NRGBAAt(nextColumn, nextRow)

	across := func(left, right uint8) float64 {
		return float64(left) + columnWeight*(float64(right)-float64(left))
	}
	down := func(top, bottom float64) uint8 {
		return uint8(top + rowWeight*(bottom-top) + 0.5)
	}
	return goimagecolor.NRGBA{
		R: down(across(topLeft.R, topRight.R), across(bottomLeft.R, bottomRight.R)),
		G: down(across(topLeft.G, topRight.G), across(bottomLeft.G, bottomRight.G)),
		B: down(across(topLeft.B, topRight.B), across(bottomLeft.B, bottomRight.B)),
		A: down(across(topLeft.A, topRight.A), across(bottomLeft.A, bottomRight.A)),
	}
}

// wrap turns an offset along one axis of the anchor into a column or row of
// the tile, repeating the tile in both directions, and how far into that texel
// the sample fell.
func (t *tilingSource) wrap(offset, span float64, pixels int) (int, float64, bool) {
	if span == 0 || pixels == 0 {
		return 0, 0, false
	}
	fraction := math.Mod(offset/span, 1)
	if fraction < 0 {
		fraction++
	}
	position := fraction * float64(pixels)
	index := int(position)
	if index < 0 {
		index = 0
	}
	if index >= pixels {
		index = pixels - 1
	}
	return index, position - float64(index), true
}
