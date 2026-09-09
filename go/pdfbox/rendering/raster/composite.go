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
// B1 composites source-over, which is java.awt.AlphaComposite.SRC_OVER and
// what every PDF blend mode reduces to when the mode is Normal or Compatible.
// The blend modes themselves are B4's.
func (i *Image) blendPixel(x, y int, src goimagecolor.RGBA, alpha float64) {
	if alpha <= 0 {
		return
	}
	if alpha > 1 {
		alpha = 1
	}
	offset := i.dst.PixOffset(x, y)
	pix := i.dst.Pix[offset : offset+4 : offset+4]

	// The destination is not premultiplied here: it is built by NewImage as
	// opaque white or as fully transparent, and every write goes through this
	// function, so the invariant is kept rather than assumed.
	dstAlpha := float64(pix[3]) / 255
	outAlpha := alpha + dstAlpha*(1-alpha)
	if outAlpha <= 0 {
		pix[0], pix[1], pix[2], pix[3] = 0, 0, 0, 0
		return
	}
	over := func(s, d uint8) uint8 {
		result := (float64(s)*alpha + float64(d)*dstAlpha*(1-alpha)) / outAlpha
		return uint8(result + 0.5)
	}
	pix[0] = over(src.R, pix[0])
	pix[1] = over(src.G, pix[1])
	pix[2] = over(src.B, pix[2])
	pix[3] = uint8(outAlpha*255 + 0.5)
}
