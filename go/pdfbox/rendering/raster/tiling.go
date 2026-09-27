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

// newTilingSource renders one tile and answers the paint that repeats it: a
// TexturePaint over the tile, stretched over the anchor rectangle.
func (i *Image) newTilingSource(paint rendering.TilingPaint) (*textureContext, error) {
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
	return newTextureContext(tile, anchor, toDevice,
		i.interpolation != rendering.NearestNeighbor), nil
}

// newImageSource is the TexturePaint PageDrawer.drawBufferedImage makes of an
// image it draws under a soft mask, `new TexturePaint(image, new
// Rectangle2D.Float(0, 0, width, height))`: the image is the tile, its anchor is
// its own rectangle, a pixel to a unit, and the transform is the one in force,
// which PageDrawer has made the image's.
//
// A stencil's tile is the stencil filled with its paint; see filledStencil.
func (i *Image) newImageSource(paint rendering.ImagePaint) (*textureContext, error) {
	var tile *goimage.NRGBA
	if paint.Fill == nil {
		tile = straightImage(paint.Image)
	} else {
		var err error
		if tile, err = i.filledStencil(paint.Image, paint.Fill); err != nil {
			return nil, err
		}
	}
	bounds := paint.Image.Bounds()
	anchor := geom.NewRectangle2D(0, 0, float64(float32(bounds.Dx())), float64(float32(bounds.Dy())))
	context := newTextureContext(tile, anchor, i.transform,
		i.interpolation != rendering.NearestNeighbor)
	// A one-bit grey image is the one TYPE_BYTE_GRAY image PDFBox makes, and
	// the decoder answers it, and only it, as an image.Gray.
	_, context.grey = paint.Image.(*goimage.Gray)
	context.grey = context.grey && paint.Fill == nil
	return context, nil
}

// straightImage is an image as the straight colour a tile holds, with its
// corner at the origin.
func straightImage(img goimage.Image) *goimage.NRGBA {
	bounds := img.Bounds()
	if nrgba, ok := img.(*goimage.NRGBA); ok && bounds.Min == (goimage.Point{}) {
		return nrgba
	}
	out := goimage.NewNRGBA(goimage.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			out.SetNRGBA(x, y, goimagecolor.NRGBAModel.Convert(
				img.At(bounds.Min.X+x, bounds.Min.Y+y)).(goimagecolor.NRGBA))
		}
	}
	return out
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
// TestAScaledTilingPatternRendersAsThePortMeansTo --
// TestTilingPatternsRenderAsPDFBoxRendersThem cannot show it, because its tiles
// are 1:1 and a whole number is its own ceiling either way.
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
