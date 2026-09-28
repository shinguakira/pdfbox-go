package raster

// A texture as a paint.
//
// Port of java.awt.TexturePaintContext, the context java.awt.TexturePaint
// hands Java2D. PDFBox paints with one in two places: a tiling pattern, whose
// TilingPaint is a TexturePaint over one rendered tile, and an image drawn under
// the graphics state's soft mask, which drawBufferedImage fills as a
// TexturePaint over the image.
//
// The context does not map each device pixel back into the texture. Java2D asks
// it for a rectangle at a time, getRaster(x, y, w, h). It maps the rectangle's
// corner back through the inverse transform in double precision, and from
// there it walks: each step across a row adds the whole part of the inverse
// transform's step and its fraction, the fraction held as a 31-bit integer
// scaled by Integer.MAX_VALUE, and carries into the texel when the fraction
// passes 2^31. The scale is one short of 2^31, so a step that should land
// exactly on a texel boundary falls just short of it and takes the texel
// before. The walk starts again at every rectangle, so which texel a pixel
// shows depends on where its rectangle began; javaRequests says where that is.
//
// Filtered, the four texels around the point are blended with twelve-bit
// weights, the top twelve bits of the two fractions.
//
// TexturePaintContext has four subclasses, for four kinds of raster. They walk
// the same way. Int and Byte, and ByteFilter, read the texture's pixels as
// they are; Any, which is what a grey image gets when it is filtered, reads
// them through the image's colour model. What PDFBox paints with this is an
// INT_RGB or INT_ARGB image -- a tile, an image, a stencil filled with its
// colour -- and a one-bit grey image, TYPE_BYTE_GRAY, which is Byte unfiltered
// and Any filtered.

import (
	goimage "image"
	goimagecolor "image/color"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// textureContext is a TexturePaintContext: the texture, what the inverse of
// its transform steps by, and the rectangle it was asked for last.
type textureContext struct {
	tile            *goimage.NRGBA
	bWidth, bHeight int
	filter          bool
	// grey says the texture is a TYPE_BYTE_GRAY image. Filtered, its context
	// is Any, which reads each texel through the image's colour model, to
	// sRGB, blends, and stores the blend back through it, to a linear grey.
	grey bool

	xOrg, yOrg                                     float64
	incXAcross, incYAcross, incXDown, incYDown     float64
	colincx, colincy, rowincx, rowincy             int64
	colincxerr, colincyerr, rowincxerr, rowincyerr int64

	// The corner of the rectangle asked for last, and the texel and the two
	// fractions its walk starts from.
	requested            bool
	requestX, requestY   int
	startX, startY       int64
	startXErr, startYErr int64

	// The walk as far as the pixel asked for last, which the pixel after it
	// in the same row is one step on from, as it is in setRaster's loop.
	walked               bool
	walkedX, walkedY     int
	texelX, texelY       int64
	texelXErr, texelYErr int64
}

// newTextureContext is TexturePaint.createContext and the constructor of the
// context it answers.
//
// xform is the transform the paint is drawn through. The anchor is where one
// copy of the texture lands, in the space xform starts from, and the texture
// is stretched over it:
//
//	xform.translate(tx, ty);
//	xform.scale(sx, sy);
//
// where tx and ty are the anchor's corner and sx and sy its size over the
// image's.
func newTextureContext(tile *goimage.NRGBA, anchor *geom.Rectangle2D,
	xform *geom.AffineTransform, filter bool) *textureContext {
	bounds := tile.Bounds()
	t := &textureContext{tile: tile, bWidth: bounds.Dx(), bHeight: bounds.Dy(), filter: filter}
	if t.bWidth == 0 || t.bHeight == 0 {
		return t
	}

	at := xform.Clone()
	at.Translate(anchor.X, anchor.Y)
	at.Scale(anchor.Width/float64(t.bWidth), anchor.Height/float64(t.bHeight))
	inverse, err := at.CreateInverse()
	if err != nil {
		// xform.setToScale(0, 0): every step is nothing, so every pixel is
		// the first texel.
		inverse = geom.NewAffineTransform(0, 0, 0, 0, 0, 0)
	}

	width, height := float64(t.bWidth), float64(t.bHeight)
	t.incXAcross = textureMod(inverse.ScaleX(), width)
	t.incYAcross = textureMod(inverse.ShearY(), height)
	t.incXDown = textureMod(inverse.ShearX(), width)
	t.incYDown = textureMod(inverse.ScaleY(), height)
	t.xOrg = inverse.TranslateX()
	t.yOrg = inverse.TranslateY()
	t.colincx = int64(javaIntOf(t.incXAcross))
	t.colincy = int64(javaIntOf(t.incYAcross))
	t.colincxerr = fractAsInt(t.incXAcross)
	t.colincyerr = fractAsInt(t.incYAcross)
	t.rowincx = int64(javaIntOf(t.incXDown))
	t.rowincy = int64(javaIntOf(t.incYDown))
	t.rowincxerr = fractAsInt(t.incXDown)
	t.rowincyerr = fractAsInt(t.incYDown)
	return t
}

// fractAsInt is the fraction of a number as TexturePaintContext holds it:
// `(int) ((d % 1.0) * Integer.MAX_VALUE)`.
func fractAsInt(d float64) int64 {
	return int64(javaIntOf(math.Mod(d, 1) * math.MaxInt32))
}

// textureMod is TexturePaintContext.mod: the remainder of num over den, made
// not negative.
func textureMod(num, den float64) float64 {
	num = math.Mod(num, den)
	if num < 0 {
		num += den
		if num >= den {
			// For very small negative numerators, the answer might be such a
			// tiny bit less than den that the difference is smaller than the
			// mantissa of a double allows and the result would then be rounded
			// to den. If that is the case then we map that number to 0 as the
			// nearest modulus representation.
			num = 0
		}
	}
	return num
}

// request is the start of getRaster(x, y, w, h): where in the texture the
// rectangle's corner falls, and the two fractions the walk from it begins
// with.
//
//	double X = mod(xOrg + x * incXAcross + y * incXDown, bWidth);
//	double Y = mod(yOrg + x * incYAcross + y * incYDown, bHeight);
//
// The explicit conversions keep each product rounded on its own, as Java's
// are, where Go would be free to fuse a multiply and an add.
func (t *textureContext) request(x, y int) {
	t.requested = true
	t.walked = false
	t.requestX, t.requestY = x, y
	if t.bWidth == 0 || t.bHeight == 0 {
		return
	}
	across := t.xOrg + float64(float64(x)*t.incXAcross)
	X := textureMod(across+float64(float64(y)*t.incXDown), float64(t.bWidth))
	across = t.yOrg + float64(float64(x)*t.incYAcross)
	Y := textureMod(across+float64(float64(y)*t.incYDown), float64(t.bHeight))
	t.startX, t.startXErr = int64(javaIntOf(X)), fractAsInt(X)
	t.startY, t.startYErr = int64(javaIntOf(Y)), fractAsInt(Y)
}

// colorAt answers the texture's colour at a device pixel, walked to from the
// corner of the rectangle last asked for.
//
// A pixel above or to the left of that corner cannot be in the rectangle, so
// it starts one of its own rather than walking backwards. That is a guard and
// not a way to sample: a caller with no rectangle of Java2D's to follow asks
// through colorAlone, which gives each pixel one.
func (t *textureContext) colorAt(x, y int) (goimagecolor.NRGBA, bool) {
	if t.bWidth == 0 || t.bHeight == 0 {
		return goimagecolor.NRGBA{}, false
	}
	if !t.requested || x < t.requestX || y < t.requestY {
		t.request(x, y)
	}
	if t.walked && y == t.walkedY && x == t.walkedX+1 {
		t.texelX, t.texelXErr = textureStepOnce(t.texelX, t.texelXErr, t.colincx, t.colincxerr, t.bWidth)
		t.texelY, t.texelYErr = textureStepOnce(t.texelY, t.texelYErr, t.colincy, t.colincyerr, t.bHeight)
	} else {
		// setRaster: down the rows first, each row starting where the one
		// above did, then across.
		across, down := int64(x-t.requestX), int64(y-t.requestY)
		rowX, rowXErr := textureStep(t.startX, t.startXErr, down, t.rowincx, t.rowincxerr, t.bWidth)
		rowY, rowYErr := textureStep(t.startY, t.startYErr, down, t.rowincy, t.rowincyerr, t.bHeight)
		t.texelX, t.texelXErr = textureStep(rowX, rowXErr, across, t.colincx, t.colincxerr, t.bWidth)
		t.texelY, t.texelYErr = textureStep(rowY, rowYErr, across, t.colincy, t.colincyerr, t.bHeight)
	}
	t.walked, t.walkedX, t.walkedY = true, x, y
	texelX, texelY := int(t.texelX), int(t.texelY)
	xErr, yErr := t.texelXErr, t.texelYErr

	var c goimagecolor.NRGBA
	if t.filter {
		nextX, nextY := texelX+1, texelY+1
		if nextX >= t.bWidth {
			nextX = 0
		}
		if nextY >= t.bHeight {
			nextY = 0
		}
		texels := [4][]uint8{t.texel(texelX, texelY), t.texel(nextX, texelY),
			t.texel(texelX, nextY), t.texel(nextX, nextY)}
		if t.grey {
			var converted [4][4]uint8
			for k, texel := range texels {
				s := grayToSRGB[texel[0]]
				converted[k] = [4]uint8{s, s, s, texel[3]}
				texels[k] = converted[k][:]
			}
		}
		blended := textureBlend(texels, xErr, yErr)
		c = goimagecolor.NRGBA{R: blended[0], G: blended[1], B: blended[2], A: blended[3]}
		if t.grey {
			g := sRGBToGrey[c.R]
			c = goimagecolor.NRGBA{R: g, G: g, B: g, A: c.A}
		}
	} else {
		p := t.texel(texelX, texelY)
		c = goimagecolor.NRGBA{R: p[0], G: p[1], B: p[2], A: p[3]}
	}
	if c.A == 0 {
		// Nothing of the texture is here, and a texture is drawn on nothing:
		// what shows is whatever was already on the page.
		return goimagecolor.NRGBA{}, false
	}
	return c, true
}

// textureStep is n steps of the walk along one axis, from the texel at and
// the fraction err, each step adding inc texels and incErr of a fraction:
//
//	if ((xerr += colincxerr) < 0) {
//	    xerr &= Integer.MAX_VALUE;
//	    x++;
//	}
//	if ((x += colincx) >= bWidth) {
//	    x -= bWidth;
//	}
//
// The fraction is below 2^31 and so is its step, so their sum overflows an int
// exactly when it reaches 2^31, and the mask takes 2^31 off: the fraction is
// carried in 2^31ths, though it was made in 2^31-1ths. After n steps that is
// the sum of all of them, carried once, and the texel wraps as often as it
// passes the edge, which is what the running form does a step at a time.
func textureStep(at, err, n, inc, incErr int64, size int) (int64, int64) {
	total := err + n*incErr
	return (at + n*inc + total>>31) % int64(size), total & math.MaxInt32
}

// textureStepOnce is one step of the walk, as setRaster takes it, with no
// division: the fraction carries at 2^31 and the texel wraps at the edge.
func textureStepOnce(at, err, inc, incErr int64, size int) (int64, int64) {
	if err += incErr; err > math.MaxInt32 {
		err -= 1 << 31
		at++
	}
	if at += inc; at >= int64(size) {
		at -= int64(size)
	}
	return at, err
}

// texel is one texel's four bytes, red, green, blue and alpha, in place.
func (t *textureContext) texel(x, y int) []uint8 {
	at := t.tile.PixOffset(x, y)
	return t.tile.Pix[at : at+4 : at+4]
}

// textureBlend is TexturePaintContext.blend: the four texels around the point,
// the one it fell in, the one after it across, the one below and the one below
// that, weighted by how far into the first it fell, xErr and yErr being the
// walk's fractions.
//
// The weights are the fractions' top twelve bits and their complements to
// 2^12, so the four products add to 2^24, and each channel is summed in an
// unsigned 32-bit word -- 255 times 2^24 does not fit in Java's int, and
// `>>>` reads it as unsigned -- and rounded half up.
func textureBlend(texels [4][]uint8, xErr, yErr int64) [4]uint8 {
	x := uint32(xErr) >> 19
	y := uint32(yErr) >> 19
	f0, f1 := (4096-x)*(4096-y), x*(4096-y)
	f2, f3 := (4096-x)*y, x*y
	var out [4]uint8
	for k := range out {
		sum := uint32(texels[0][k])*f0 + uint32(texels[1][k])*f1 +
			uint32(texels[2][k])*f2 + uint32(texels[3][k])*f3
		out[k] = uint8((sum + 1<<23) >> 24)
	}
	return out
}

// sRGBToGrey is the grey TYPE_BYTE_GRAY's colour model makes of an sRGB grey
// (s, s, s): ComponentColorModel.getDataElements takes each channel through
// ColorModel.getsRGB8ToLinearRGB16LUT and weighs them in float, and a grey's
// three channels are the same. It is the way back from grayToSRGB, and what a
// filtered grey texture's blend is stored as. The values are the JDK's, which
// TexturePaintDrv.java prints and TestTheGreyTablesAreTheJDKs checks.
var sRGBToGrey = [256]uint8{
	0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 2, 2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 3, 3, 3,
	4, 4, 4, 4, 4, 5, 5, 5, 5, 6, 6, 6, 6, 7, 7, 7,
	8, 8, 8, 8, 9, 9, 9, 10, 10, 10, 11, 11, 12, 12, 12, 13,
	13, 13, 14, 14, 15, 15, 16, 16, 17, 17, 17, 18, 18, 19, 19, 20,
	20, 21, 22, 22, 23, 23, 24, 24, 25, 25, 26, 27, 27, 28, 29, 29,
	30, 30, 31, 32, 32, 33, 34, 35, 35, 36, 37, 37, 38, 39, 40, 41,
	41, 42, 43, 44, 45, 45, 46, 47, 48, 49, 50, 51, 51, 52, 53, 54,
	55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 69, 70,
	71, 72, 73, 74, 76, 77, 78, 79, 80, 81, 82, 84, 85, 86, 87, 88,
	90, 91, 92, 93, 95, 96, 97, 99, 100, 101, 103, 104, 105, 107, 108, 109,
	111, 112, 114, 115, 116, 118, 119, 121, 122, 124, 125, 127, 128, 130, 131, 133,
	134, 136, 138, 139, 141, 142, 144, 146, 147, 149, 151, 152, 154, 156, 157, 159,
	161, 163, 164, 166, 168, 170, 171, 173, 175, 177, 179, 181, 183, 184, 186, 188,
	190, 192, 194, 196, 198, 200, 202, 204, 206, 208, 210, 212, 214, 216, 218, 220,
	222, 224, 226, 229, 231, 233, 235, 237, 239, 242, 244, 246, 248, 250, 253, 255,
}
