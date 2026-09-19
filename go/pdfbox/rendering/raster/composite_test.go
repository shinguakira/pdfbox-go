package raster

import (
	goimagecolor "image/color"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/blend"
)

// TestAnOpaqueNormalCompositeIsTheSource is what lets blendInto replace a pixel
// outright where the source is opaque and the blend mode is Normal.
//
// BlendComposite's arithmetic, which mix is, does not come to the source by
// algebra alone: the result alpha is dstAlpha + 1 - dstAlpha, in doubles, and
// the colour is dest + ratio * (source - dest), in floats, and neither is
// exact. What makes it the source is the rounding to a byte at the end. This
// runs mix over every source byte, every destination byte and every destination
// alpha, with no blend mode and with Normal, and holds each result to the
// source.
func TestAnOpaqueNormalCompositeIsTheSource(t *testing.T) {
	for _, mode := range []*blend.BlendMode{nil, blend.Normal} {
		i := &Image{blendMode: mode}
		pix := make([]uint8, 4)
		for dstAlpha := 0; dstAlpha < 256; dstAlpha++ {
			for dest := 0; dest < 256; dest++ {
				for source := 0; source < 256; source++ {
					pix[0], pix[1], pix[2], pix[3] = uint8(dest), uint8(255-dest), uint8(dest), uint8(dstAlpha)
					src := goimagecolor.NRGBA{R: uint8(source), G: uint8(source), B: uint8(255 - source), A: 0xFF}
					if !i.mix(pix, src, 1) {
						t.Fatalf("mode %v: mix cleared the pixel for source %d over %d at alpha %d",
							mode, source, dest, dstAlpha)
					}
					if pix[0] != src.R || pix[1] != src.G || pix[2] != src.B || pix[3] != 0xFF {
						t.Fatalf("mode %v: source %d over %d at alpha %d mixes to %v, not the source",
							mode, source, dest, dstAlpha, pix)
					}
				}
			}
		}
	}
}
