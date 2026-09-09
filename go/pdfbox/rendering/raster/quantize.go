package raster

// What a surface can hold.
//
// Java draws onto a BufferedImage of the type the caller asked
// `PDFRenderer.renderImage` for, and the image quantizes what is written into
// it: TYPE_BYTE_GRAY keeps one channel, TYPE_BYTE_BINARY keeps one bit. The
// port holds every surface as straight RGBA and applies the same quantization
// where Java's image would have, which is **as each pixel is written** -- so a
// later composite reads back what the surface really holds, as Java's does.
//
// The one exception is the same one Java makes: a page that blends at the top
// level is drawn onto ARGB whatever was asked for -- PDFBOX-4095 -- and the
// conversion happens once at the end, in onWhite.

import (
	goimagecolor "image/color"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// quantize turns a pixel into what the surface's type can hold, in place.
func (i *Image) quantize(pix []uint8) {
	switch i.imageType {
	case rendering.Gray:
		g := grayOfRGB(pix[0], pix[1], pix[2])
		pix[0], pix[1], pix[2] = g, g, g
	case rendering.Binary:
		if blackIsNearer(pix[0], pix[1], pix[2]) {
			pix[0], pix[1], pix[2] = 0, 0, 0
		} else {
			pix[0], pix[1], pix[2] = 0xFF, 0xFF, 0xFF
		}
	}
}

// quantizeColor is quantize for a colour rather than a pixel.
func (i *Image) quantizeColor(c goimagecolor.NRGBA) goimagecolor.NRGBA {
	pix := []uint8{c.R, c.G, c.B, c.A}
	i.quantize(pix)
	return goimagecolor.NRGBA{R: pix[0], G: pix[1], B: pix[2], A: pix[3]}
}

// grayOfRGB is what a TYPE_BYTE_GRAY surface makes of a colour.
//
// Measured, not assumed: a JDK 17 `renderImage(0, 1, ImageType.GRAY)` of the
// page this package renders gives 61 for (26,51,204), 174 for (230,179,0), 162
// for (102,204,102), 53 for (128,0,126) and 35 for (32,0,222), and
// `0.299R + 0.587G + 0.114B` rounded gives each of them. Those are the
// ITU-R BT.601 luma coefficients, which is what Java2D's ByteGray surface
// converts with -- not the CIE Y weights, and not the ICC transform that
// drawing an *image* onto a gray surface goes through. See the soft mask's
// `luma`, which is that other one.
func grayOfRGB(r, g, b uint8) uint8 {
	return uint8(0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b) + 0.5)
}

// blackIsNearer is what a TYPE_BYTE_BINARY surface asks: its colour model has
// two entries, black and white, and a colour becomes whichever is nearer in
// RGB.
//
//	R^2 + G^2 + B^2  <  (255-R)^2 + (255-G)^2 + (255-B)^2
func blackIsNearer(r, g, b uint8) bool {
	toBlack := square(int(r)) + square(int(g)) + square(int(b))
	toWhite := square(255-int(r)) + square(255-int(g)) + square(255-int(b))
	return toBlack < toWhite
}

func square(v int) int { return v * v }
