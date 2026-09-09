package raster

// Putting coverage, a paint and a clip onto the destination.
//
// Java hands all of this to Graphics2D: a Composite over a Paint, clipped.
// The port writes it, because no Go library composites the way PDF asks --
// the sixteen blend modes of ISO 32000-1 table 136 are not something a general
// 2D library carries.
//
// The blend functions themselves are ported: `blend.BlendMode` is slice 9's.
// What is here is the loop that applies one to a pixel.

import (
	goimage "image"
	goimagecolor "image/color"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// paintSource answers the colour a paint gives at a device pixel, and false
// where it paints nothing there.
//
// A solid colour ignores the point; a shading answers its context; a tiling
// pattern answers the tile. Java reaches all three through
// java.awt.PaintContext.
type paintSource interface {
	colorAt(x, y int) (goimagecolor.RGBA, bool)
}

// solidSource is a java.awt.Color: the same colour everywhere.
type solidSource struct {
	color goimagecolor.RGBA
}

func (s solidSource) colorAt(int, int) (goimagecolor.RGBA, bool) { return s.color, true }

// sourceOf returns what a paint paints with.
func (i *Image) sourceOf(paint rendering.Paint) (paintSource, float64, error) {
	switch p := paint.(type) {
	case rendering.ColorPaint:
		// The alpha of the paint multiplies the alpha constant of the
		// composite, which is what Java gets from an AlphaComposite over a
		// Color that carries one.
		return solidSource{color: goimagecolor.RGBA{
			R: clampToByte(p.Red),
			G: clampToByte(p.Green),
			B: clampToByte(p.Blue),
			A: 0xFF,
		}}, float64(p.Alpha), nil

	case rendering.SoftMaskedPaint:
		if p.Paint == nil {
			// applySoftMaskToPaint(nil, mask), which only
			// showTransparencyGroupOnGraphics makes: there is nothing to paint
			// with, and the mask is for PopGroup to apply to the group it is
			// about to composite.
			return nil, 0, ErrNotDrawn
		}
		under, alpha, err := i.sourceOf(p.Paint)
		if err != nil {
			return nil, 0, err
		}
		masked, err := i.newSoftMaskSource(p, under)
		if err != nil {
			return nil, 0, err
		}
		return masked, alpha, nil

	case rendering.TilingPaint:
		// A tiling pattern carries no alpha of its own either; the tile does,
		// per pixel, and colorAt answers it.
		source, err := i.cachedTilingSource(p)
		if err != nil {
			return nil, 0, err
		}
		return source, 1, nil

	case rendering.ShadingPaint:
		// A shading carries no alpha of its own; what it is drawn with comes
		// from the composite, as it does in Java.
		context, err := newShadingContext(p.Shading, p.Matrix, i.transform, i.dst.Bounds())
		if err != nil {
			return nil, 0, err
		}
		return context, 1, nil

	default:
		return nil, 0, ErrNotDrawn
	}
}

// clampToByte turns a component of 0..1 into a byte, rounding, and holds it
// inside the range.
//
// PageDrawer has already clamped what it hands over -- see clampColor -- so
// this is the conversion and not a second guard.
func clampToByte(v float32) uint8 {
	switch {
	case v <= 0:
		return 0
	case v >= 1:
		return 0xFF
	}
	return uint8(v*255 + 0.5)
}

// blendPixel puts one source colour onto the destination with the given
// alpha, which is the coverage, the constant alpha and the colour's own alpha
// already multiplied together.
//
// Port of BlendComposite.BlendCompositeContext.compose, for one pixel, with
// the parts that cannot arise here left out: this backend's surface is sRGB
// with alpha, so the source and the destination are always in the same colour
// space -- no conversion -- and `subtractive` is false, which it is for every
// space but CMYK.
//
// The three lines that do the work are Java's, in Java's order:
//
//	value = blend(srcValue, dstValue)
//	value = srcValue + dstAlpha * (value - srcValue)
//	value = dstValue + srcAlphaRatio * (value - dstValue)
//
// With no blend mode the first is the identity, and what is left is
// source-over. That is why Normal and Compatible need no special case.
func (i *Image) blendPixel(x, y int, src goimagecolor.RGBA, srcAlpha float64) {
	if srcAlpha <= 0 {
		return
	}
	if srcAlpha > 1 {
		srcAlpha = 1
	}
	i.blendInto(i.dst, x, y, src, srcAlpha)
	if i.secondary != nil {
		// The alpha-only surface of a group that can see its backdrop, drawn
		// in parallel so that PopGroup knows what alpha the group's own
		// contents have. Java draws the group twice for the same reason.
		i.blendInto(i.secondary, x, y, src, srcAlpha)
	}
}

// blendInto is the compose of one pixel onto one surface.
func (i *Image) blendInto(dst *goimage.RGBA, x, y int, src goimagecolor.RGBA,
	srcAlpha float64) {
	offset := dst.PixOffset(x, y)
	pix := dst.Pix[offset : offset+4 : offset+4]

	dstAlpha := float64(pix[3]) / 255
	resultAlpha := dstAlpha + srcAlpha - srcAlpha*dstAlpha
	if resultAlpha <= 0 {
		pix[0], pix[1], pix[2], pix[3] = 0, 0, 0, 0
		return
	}
	srcAlphaRatio := srcAlpha / resultAlpha

	source := [3]float32{float32(src.R) / 255, float32(src.G) / 255, float32(src.B) / 255}
	dest := [3]float32{float32(pix[0]) / 255, float32(pix[1]) / 255, float32(pix[2]) / 255}

	// blended is what the mode makes of the two, before either is mixed back
	// in by the alphas.
	blended := source
	if mode := i.blendMode; mode != nil {
		if mode.IsSeparableBlendMode() {
			channel := mode.BlendChannelFunction()
			for k := 0; k < 3; k++ {
				blended[k] = channel(source[k], dest[k])
			}
		} else {
			// A nonseparable mode reads all three channels at once, and Java
			// computes it in RGB, which is what these already are.
			result := make([]float32, 3)
			mode.BlendFunction()(source[:], dest[:], result)
			copy(blended[:], result)
		}
	}

	for k := 0; k < 3; k++ {
		value := source[k] + float32(dstAlpha)*(blended[k]-source[k])
		value = dest[k] + float32(srcAlphaRatio)*(value-dest[k])
		pix[k] = clampToByte(value)
	}
	pix[3] = uint8(resultAlpha*255 + 0.5)
}
