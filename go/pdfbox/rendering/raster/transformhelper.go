package raster

// Drawing an image through a transform, as java.awt.Graphics2D.drawImage(image,
// transform, observer) does it under the bicubic hint.
//
// Java2D looks at the transform first. An image that lands one to one on whole
// pixels -- the transform comes to a translation by whole numbers, give or take
// DrawImage.MAX_TX_ERROR -- is copied. Anything else goes through
// TransformHelper, the JDK's native image transform, which is the one part of
// drawing an image the JDK does in C: for every device pixel whose centre falls
// inside the image, it steps the inverse transform in 32.32 fixed point, takes
// the 4 by 4 block of source pixels around that point, clamped at the image's
// edges, and weights it with a 256-step integer table of Keys' cubic, a = -0.5.
// The block is the same size whatever the scale; x/image/draw's CatmullRom, which
// this replaces, widens its kernel by the shrink factor, which is both more work
// than Java does and a different picture.
//
// Port of sun.java2d.pipe.DrawImage.transformImage, tryCopyOrScale and
// renderImageXform, and of TransformHelper.c -- Transform, calculateEdges,
// Transform_SafeHelper, BicubicInterp and init_bicubic_table -- and the
// TransformHelper macros of LoopMacros.h, JDK 17. What comes out is IntArgbPre,
// which is what image.RGBA holds; putting it on the page is the backend's
// compositing, as it is for everything else.

import (
	goimage "image"
	goimagecolor "image/color"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
)

// maxTxError is DrawImage.MAX_TX_ERROR: how far from whole a coordinate may be
// and still be taken for whole.
const maxTxError = .0001

// closeToInteger is DrawImage.closeToInteger.
func closeToInteger(i int, d float64) bool {
	return math.Abs(d-float64(i)) < maxTxError
}

// drawImageJava2D writes into sampled what Graphics2D.drawImage(src, tx, null)
// puts down under the bicubic hint, as premultiplied colour, and leaves every
// other pixel of it untouched. clipBounds is the bounds of Java's clip region,
// whose corner is where TransformHelper starts stepping.
func drawImageJava2D(sampled *goimage.RGBA, src goimage.Image, tx *geom.AffineTransform,
	clipBounds goimage.Rectangle) {
	source := newArgbPreSource(src)
	w, h := source.width, source.height

	// transformImage: three corners, to see whether the transform is
	// rectilinear, and tryCopyOrScale if it is.
	coords := []float64{0, 0, float64(w), float64(h), 0, float64(h)}
	tx.TransformDoubles(coords, 0, coords, 0, 3)
	if math.Abs(coords[0]-coords[4]) < maxTxError && math.Abs(coords[3]-coords[5]) < maxTxError &&
		copyImage(sampled, source, coords) {
		return
	}
	transformImage(sampled, source, tx, clipBounds)
}

// copyImage is the copy half of tryCopyOrScale: an image the transform puts
// down one to one, at whole pixels, is renderImageCopy. It answers false where
// Java goes on to transform it -- the scaling half is ScaledBlit, which only
// nearest neighbour takes.
func copyImage(sampled *goimage.RGBA, source *argbPreSource, coords []float64) bool {
	dx1, dy1, dx2, dy2 := coords[0], coords[1], coords[2], coords[3]
	for _, d := range []float64{dx1, dy1, dx2, dy2} {
		if d < math.MinInt32 || d > math.MaxInt32 {
			return false
		}
	}
	if !closeToInteger(source.width, dx2-dx1) || !closeToInteger(source.height, dy2-dy1) {
		return false
	}
	idx := int(math.Floor(dx1 + 0.5))
	idy := int(math.Floor(dy1 + 0.5))
	if !closeToInteger(idx, dx1) || !closeToInteger(idy, dy1) {
		return false
	}
	area := sampled.Bounds().Intersect(goimage.Rect(idx, idy, idx+source.width, idy+source.height))
	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			putArgbPre(sampled, x, y, source.at(x-idx, y-idy))
		}
	}
	return true
}

// transformInfo is TransformInfo: the inverse transform, device to source.
type transformInfo struct {
	dxdx, dxdy, tx float64
	dydx, dydy, ty float64
}

// transform is Transform_transform.
func (t *transformInfo) transform(x, y float64) (float64, float64) {
	return t.dxdx*x + t.dxdy*y + t.tx, t.dydx*x + t.dydy*y + t.ty
}

// longOneHalf is LongOneHalf, a half in 32.32 fixed point.
const longOneHalf = int64(1) << 31

// dblToLong is DblToLong: a double in 32.32 fixed point, truncated as C casts.
func dblToLong(d float64) int64 { return int64(d * (1 << 32)) }

// wholeOfLong is WholeOfLong.
func wholeOfLong(l int64) int32 { return int32(l >> 32) }

// txFixedUnsafe is TX_FIXED_UNSAFE: a coordinate 32.32 fixed point cannot hold.
func txFixedUnsafe(v float64) bool {
	return math.IsInf(v, 0) || math.IsNaN(v) || math.Abs(v) >= 1<<30
}

// transformImage is renderImageXform and the native Transform it calls, for
// the bicubic interpolation.
func transformImage(sampled *goimage.RGBA, source *argbPreSource, tx *geom.AffineTransform,
	clipBounds goimage.Rectangle) {
	itx, err := tx.CreateInverse()
	if err != nil {
		// Non-invertible transform means no output
		return
	}
	info := transformInfo{
		dxdx: itx.ScaleX(), dxdy: itx.ShearX(), tx: itx.TranslateX(),
		dydx: itx.ShearY(), dydy: itx.ScaleY(), ty: itx.TranslateY(),
	}
	if !finite(info.dxdx, info.dxdy, info.tx, info.dydx, info.dydy, info.ty) {
		return
	}

	// Find the maximum bounds on the destination that will be affected by the
	// transformed source.
	coords := []float64{0, 0, float64(source.width), 0, 0, float64(source.height),
		float64(source.width), float64(source.height)}
	tx.TransformDoubles(coords, 0, coords, 0, 4)
	ddx1, ddx2 := coords[0], coords[0]
	ddy1, ddy2 := coords[1], coords[1]
	for k := 2; k < len(coords); k += 2 {
		if d := coords[k]; ddx1 > d {
			ddx1 = d
		} else if ddx2 < d {
			ddx2 = d
		}
		if d := coords[k+1]; ddy1 > d {
			ddy1 = d
		} else if ddy2 < d {
			ddy2 = d
		}
	}
	bounds := goimage.Rect(
		max(javaIntOf(math.Floor(ddx1)), clipBounds.Min.X),
		max(javaIntOf(math.Floor(ddy1)), clipBounds.Min.Y),
		min(javaIntOf(math.Ceil(ddx2)), clipBounds.Max.X),
		min(javaIntOf(math.Ceil(ddy2)), clipBounds.Max.Y))
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return
	}

	if overflows(bounds, &info) {
		safeTransform(sampled, source, &info, bounds)
		return
	}

	dxdx, dydx := dblToLong(info.dxdx), dblToLong(info.dydx)
	dxdy, dydy := dblToLong(info.dxdy), dblToLong(info.dydy)
	xorig, yorig := info.transform(float64(bounds.Min.X)+0.5, float64(bounds.Min.Y)+0.5)
	xbase, ybase := dblToLong(xorig), dblToLong(yorig)
	edges := calculateEdges(bounds, dxdx, dydx, dxdy, dydy, xbase, ybase,
		uint32(source.width), uint32(source.height))

	rowx, rowy := xbase, ybase
	for dy := bounds.Min.Y; dy < bounds.Max.Y; dy++ {
		edge := edges[dy-bounds.Min.Y]
		for dx := edge[0]; dx < edge[1]; dx++ {
			xlong := rowx + int64(dx-bounds.Min.X)*dxdx
			ylong := rowy + int64(dx-bounds.Min.X)*dydx
			put(sampled, dx, dy, source.bicubic(xlong, ylong))
		}
		rowx += dxdy
		rowy += dydy
	}
}

// javaIntOf is Java's (int) of a double: truncated, and held to the int range.
func javaIntOf(d float64) int {
	switch {
	case math.IsNaN(d):
		return 0
	case d >= math.MaxInt32:
		return math.MaxInt32
	case d <= math.MinInt32:
		return math.MinInt32
	}
	return int(d)
}

// overflows is checkOverflow: whether any corner pixel's centre, through the
// inverse transform, is out of reach of 32.32 fixed point.
func overflows(bounds goimage.Rectangle, info *transformInfo) bool {
	x1, y1 := float64(bounds.Min.X)+0.5, float64(bounds.Min.Y)+0.5
	x2, y2 := float64(bounds.Max.X)-0.5, float64(bounds.Max.Y)-0.5
	for _, corner := range [][2]float64{{x1, y1}, {x2, y1}, {x1, y2}, {x2, y2}} {
		x, y := info.transform(corner[0], corner[1])
		if txFixedUnsafe(x) || txFixedUnsafe(y) {
			return true
		}
	}
	return false
}

// calculateEdges answers, for each scanline of bounds, the first and the last
// pixel plus one whose centre maps back inside the source.
func calculateEdges(bounds goimage.Rectangle, dxdx, dydx, dxdy, dydy, xbase, ybase int64,
	sw, sh uint32) [][2]int {
	edges := make([][2]int, bounds.Dy())
	drowx := int64(bounds.Dx()-1) * dxdx
	drowy := int64(bounds.Dx()-1) * dydx
	outside := func(xlong, ylong int64) bool {
		return uint32(wholeOfLong(ylong)) >= sh || uint32(wholeOfLong(xlong)) >= sw
	}
	for row := range edges {
		dx1, dx2 := bounds.Min.X, bounds.Max.X

		xlong, ylong := xbase, ybase
		for dx1 < dx2 && outside(xlong, ylong) {
			dx1++
			xlong += dxdx
			ylong += dydx
		}

		xlong, ylong = xbase+drowx, ybase+drowy
		for dx2 > dx1 && outside(xlong, ylong) {
			dx2--
			xlong -= dxdx
			ylong -= dydx
		}

		edges[row] = [2]int{dx1, dx2}
		xbase += dxdy
		ybase += dydy
	}
	return edges
}

// safeTransform is Transform_SafeHelper, for a transform whose stepping would
// overflow: each pixel is taken back through the inverse transform on its own.
func safeTransform(sampled *goimage.RGBA, source *argbPreSource, info *transformInfo,
	bounds goimage.Rectangle) {
	sw, sh := float64(source.width), float64(source.height)
	for dy := bounds.Min.Y; dy < bounds.Max.Y; dy++ {
		for dx := bounds.Min.X; dx < bounds.Max.X; dx++ {
			x, y := info.transform(float64(dx)+0.5, float64(dy)+0.5)
			xlong, ylong := dblToLong(x), dblToLong(y)
			if x >= 0 && y >= 0 && x < sw && y < sh &&
				wholeOfLong(xlong) < int32(source.width) && wholeOfLong(ylong) < int32(source.height) {
				put(sampled, dx, dy, source.bicubic(xlong, ylong))
			}
		}
	}
}

// put writes one IntArgbPre result where sampled has room for it.
func put(sampled *goimage.RGBA, x, y int, argb uint32) {
	if goimage.Pt(x, y).In(sampled.Rect) {
		putArgbPre(sampled, x, y, argb)
	}
}

// putArgbPre writes an IntArgbPre value into a premultiplied image.
func putArgbPre(sampled *goimage.RGBA, x, y int, argb uint32) {
	offset := sampled.PixOffset(x, y)
	sampled.Pix[offset] = uint8(argb >> 16)
	sampled.Pix[offset+1] = uint8(argb >> 8)
	sampled.Pix[offset+2] = uint8(argb)
	sampled.Pix[offset+3] = uint8(argb >> 24)
}

// argbPreSource reads an image's pixels as IntArgbPre, which is what the
// TransformHelper macros Copy<Type>ToIntArgbPre make of each source type.
type argbPreSource struct {
	image         goimage.Image
	min           goimage.Point
	width, height int
}

func newArgbPreSource(src goimage.Image) *argbPreSource {
	bounds := src.Bounds()
	return &argbPreSource{image: src, min: bounds.Min, width: bounds.Dx(), height: bounds.Dy()}
}

// at is one pixel as IntArgbPre.
//
// The types are the ones PDFBox draws, and the conversions are the macros':
// an image.RGBA is TYPE_INT_RGB where it is opaque, LoadIntRgbTo1IntArgb, and
// already premultiplied where it is not; an image.NRGBA is TYPE_INT_ARGB,
// CopyIntArgbToIntArgbPre, which premultiplies with MUL8; an image.Gray is
// TYPE_BYTE_GRAY, LoadByteGrayTo1IntArgb, which copies the grey into all three
// channels -- with none of the gamma TYPE_BYTE_GRAY's getRGB applies. Anything
// else is read as straight colour and premultiplied as TYPE_INT_ARGB is.
func (s *argbPreSource) at(x, y int) uint32 {
	x += s.min.X
	y += s.min.Y
	switch img := s.image.(type) {
	case *goimage.RGBA:
		p := img.Pix[img.PixOffset(x, y):]
		return uint32(p[3])<<24 | uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2])
	case *goimage.NRGBA:
		p := img.Pix[img.PixOffset(x, y):]
		return intArgbToPre(p[3], p[0], p[1], p[2])
	case *goimage.Gray:
		g := uint32(img.Pix[img.PixOffset(x, y)])
		return 0xFF000000 | g<<16 | g<<8 | g
	}
	c := goimagecolor.NRGBAModel.Convert(s.image.At(x, y)).(goimagecolor.NRGBA)
	return intArgbToPre(c.A, c.R, c.G, c.B)
}

// intArgbToPre is CopyIntArgbToIntArgbPre.
func intArgbToPre(a, r, g, b uint8) uint32 {
	switch {
	case a == 0:
		return 0
	case a < 0xFF:
		r, g, b = mul8(a, r), mul8(a, g), mul8(a, b)
	}
	return uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
}

// bicubic is the bicubic helper -- DEFINE_TRANSFORMHELPER_BC -- and
// BicubicInterp for one pixel: the 4 by 4 block around xlong, ylong, with the
// image's edge pixels standing in for what is past them, and the weighted sum.
func (s *argbPreSource) bicubic(xlong, ylong int64) uint32 {
	var block [16]uint32
	cw, ch := int32(s.width), int32(s.height)

	xl := xlong - longOneHalf
	yl := ylong - longOneHalf
	xwhole := wholeOfLong(xl)
	ywhole := wholeOfLong(yl)

	xdelta0 := (-xwhole) >> 31
	xdelta1 := int32(uint32(xwhole+1-cw) >> 31)
	xdelta2 := int32(uint32(xwhole+2-cw) >> 31)
	isneg := xwhole >> 31
	xwhole -= isneg
	xdelta1 += isneg
	xdelta2 += xdelta1

	// The row steps are in rows here, where the C has them in bytes.
	ydelta0 := ((-ywhole) >> 31) & -1
	ydelta1 := ((ywhole + 1 - ch) >> 31) & 1
	ydelta2 := ((ywhole + 2 - ch) >> 31) & 1
	isneg = ywhole >> 31
	ywhole -= isneg
	ydelta1 += isneg & -1

	columns := [4]int32{xwhole + xdelta0, xwhole, xwhole + xdelta1, xwhole + xdelta2}
	rows := [4]int32{ywhole + ydelta0, ywhole, ywhole + ydelta1, ywhole + ydelta1 + ydelta2}
	for r, row := range rows {
		for c, column := range columns {
			block[r*4+c] = s.at(int(column), int(row))
		}
	}

	xfactor := int32(uint32(xl) >> 24)
	yfactor := int32(uint32(yl) >> 24)
	return bicubicInterp(&block, xfactor, yfactor)
}

// bicubicInterp is BicubicInterp's body for one pixel, with the integer
// arithmetic TransformHelper is compiled with: BICUBIC_USE_INT_MATH.
func bicubicInterp(block *[16]uint32, xfactor, yfactor int32) uint32 {
	table := bicubicCoefficients
	accumA, accumR, accumG, accumB := int32(1<<15), int32(1<<15), int32(1<<15), int32(1<<15)
	xc := [4]int32{xfactor + 256, xfactor, 256 - xfactor, 512 - xfactor}
	yc := [4]int32{yfactor + 256, yfactor, 256 - yfactor, 512 - yfactor}
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			factor := table[xc[c]] * table[yc[r]]
			rgb := block[r*4+c]
			accumB += int32(rgb&0xff) * factor
			accumG += int32(rgb>>8&0xff) * factor
			accumR += int32(rgb>>16&0xff) * factor
			accumA += int32(rgb>>24&0xff) * factor
		}
	}
	accumA >>= 16
	accumR >>= 16
	accumG >>= 16
	accumB >>= 16
	accumA = saturate(accumA, 255)
	accumR = saturate(accumR, accumA)
	accumG = saturate(accumG, accumA)
	accumB = saturate(accumB, accumA)
	return uint32(accumA)<<24 | uint32(accumR)<<16 | uint32(accumG)<<8 | uint32(accumB)
}

// saturate is SAT: the value held to 0..max.
func saturate(v, limit int32) int32 {
	v &= ^(v >> 31) // negatives become 0
	v -= limit      // only overflows are now positive
	v &= v >> 31    // positives become 0
	v += limit      // range is now [0 -> max]
	return v
}

// bicubicCoefficients is bicubic_coeff: init_bicubic_table(-0.5), with
// BC_DblToCoeff truncating to 256ths as the integer build does.
//
// It is built at load, as mul8Table below is. Building it on first use would
// be a write to a package variable, and this is a library: two pages rendered
// at once in one process would race on it, which the race detector proves in
// TestTwoPagesRenderAtOnce.
var bicubicCoefficients = func() *[513]int32 {
	var table [513]int32
	const a = -0.5
	i := 0
	for ; i < 256; i++ {
		// r(x) = (A + 2)|x|^3 - (A + 3)|x|^2 + 1 , 0 <= |x| < 1
		x := float64(i) / 256.0
		x = ((a+2)*x-(a+3))*x*x + 1
		table[i] = int32(x * 256)
	}
	for ; i < 384; i++ {
		// r(x) = A|x|^3 - 5A|x|^2 + 8A|x| - 4A , 1 <= |x| < 2
		x := float64(i) / 256.0
		x = ((a*x-5*a)*x+8*a)*x - 4*a
		table[i] = int32(x * 256)
	}
	table[384] = (256 - table[128]*2) / 2
	for i++; i <= 512; i++ {
		table[i] = 256 - (table[512-i] + table[i-256] + table[768-i])
	}
	return &table
}()

// mul8Table is AlphaMath.c's mul8table.
var mul8Table = func() *[256][256]uint8 {
	var table [256][256]uint8
	for i := uint32(1); i < 256; i++ { // SCALE == (1 << 24)
		inc := i<<16 + i<<8 + i // approx. SCALE * (i/255.0)
		val := inc + 1<<23      // inc + SCALE*0.5
		for j := 1; j < 256; j++ {
			table[i][j] = uint8(val >> 24) // val / SCALE
			val += inc
		}
	}
	return &table
}()

// mul8 is MUL8.
func mul8(a, b uint8) uint8 { return mul8Table[a][b] }
