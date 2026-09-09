package raster

// Soft masks.
//
// Port of rendering/SoftMask.java and of the raster half of
// PageDrawer.applySoftMaskToPaint: the mask's transparency group is rendered,
// turned into a grey image, put back to the page's scale, and then every pixel
// the masked paint puts down has its alpha multiplied by the grey under it.
//
// The drawer's half -- the group's box, its size, its origin, its backdrop
// colour and the transform that puts it back -- is
// rendering.PageDrawer.DrawSoftMask, which this asks.

import (
	goimage "image"
	goimagecolor "image/color"
	"math"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/function"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// softMaskSource is a SoftMask paint: another source, seen through a mask.
//
// under is nil where there is nothing underneath, which is what
// applySoftMaskToPaint(nil, mask) makes and what PopGroup is handed -- there
// the group's own pixels are what the mask is applied to, and only alphaAt is
// asked for.
type softMaskSource struct {
	under paintSource

	mask *goimage.Alpha
	// originX and originY are where the mask's first pixel sits, from
	// TransparencyGroup.getOrigin, truncated the way SoftPaintContext.getRaster
	// truncates it: `x1 - (int) origin.getX()`.
	originX, originY int

	// backdrop is what a pixel outside the mask is multiplied by: Java's `bc`,
	// zero unless a Luminosity mask names a backdrop colour.
	backdrop uint8

	// transfer is the mask's /TR, nil for none and nil for the identity, which
	// SoftMask's constructor also treats as none.
	transfer function.PDFunction
	// transferred memoises it over the 256 greys, as Java's `map` does.
	transferred [256]float32
	known       [256]bool
}

// newSoftMaskSource renders the mask and answers the paint that sees through
// it, or under alone where the mask turns out to be empty.
func (i *Image) newSoftMaskSource(paint rendering.SoftMaskedPaint,
	under paintSource) (paintSource, error) {
	if paint.Drawer == nil || paint.Mask == nil {
		return under, nil
	}
	layer, err := paint.Drawer.DrawSoftMask(paint.Mask,
		func(width, height int) rendering.Backend {
			return NewImage(width, height, rendering.ARGB)
		})
	if err != nil {
		return nil, err
	}
	if layer == nil {
		// "Adobe Reader ignores empty softmasks instead of using bc color"
		// -- sample file PDFJS-6967_reduced_outside_softmask.pdf
		return under, nil
	}
	rendered, isImage := layer.Backend.(*Image)
	if !isImage {
		// DrawSoftMask made it with the function this handed it, so it is one.
		return nil, ErrNotDrawn
	}

	transfer, err := paint.Mask.TransferFunction()
	if err != nil {
		return nil, err
	}
	if _, isIdentity := transfer.(*function.PDFunctionTypeIdentity); isIdentity {
		transfer = nil
	}

	source := &softMaskSource{
		under:    under,
		mask:     adjustMask(grayOf(rendered.dst, layer), layer),
		originX:  int(layer.OriginX),
		originY:  int(layer.OriginY),
		transfer: transfer,
	}
	if layer.BackdropColor != nil {
		rgb, err := layer.BackdropColor.ToRGB()
		if err != nil {
			// "keep default", which is zero.
			rgb = 0
		}
		// http://stackoverflow.com/a/25463098/535646
		source.backdrop = uint8((299*((rgb>>16)&0xFF) + 587*((rgb>>8)&0xFF) +
			114*(rgb&0xFF)) / 1000)
	}
	return source, nil
}

// grayOf turns the rendered group into the grey image the mask is read from.
//
//	if (COSName.ALPHA.equals(subType))       gray.setData(image.getAlphaRaster());
//	else if (COSName.LUMINOSITY.equals(...)) g.drawImage(image, 0, 0, null);
//
// An Alpha mask reads the group's alpha channel straight. A Luminosity mask
// draws the group onto a TYPE_BYTE_GRAY surface, which is opaque and starts
// black, so a pixel the group did not reach reads as black -- unless
// DrawSoftMask painted a backdrop colour under it first.
//
// What that drawImage does to a colour depends on what the group was drawn
// into, which is why the layer carries isGray: a grey group is already a grey
// image and the channel goes straight across, and only a group in some other
// space is converted. See luma for what that conversion is worth.
func grayOf(rendered *goimage.RGBA, layer *rendering.SoftMaskLayer) *goimage.Alpha {
	bounds := rendered.Bounds()
	gray := goimage.NewAlpha(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			c := rendered.RGBAAt(x, y)
			if !layer.Luminosity {
				gray.SetAlpha(x, y, goimagecolor.Alpha{A: c.A})
				continue
			}
			// The group's colour at its own alpha, over black.
			alpha := float64(c.A) / 255
			red := uint8(float64(c.R)*alpha + 0.5)
			if layer.Gray {
				// A grey group's three channels are one channel; drawing a
				// grey image onto a grey image converts nothing.
				gray.SetAlpha(x, y, goimagecolor.Alpha{A: red})
				continue
			}
			gray.SetAlpha(x, y, goimagecolor.Alpha{A: luma(red,
				uint8(float64(c.G)*alpha+0.5),
				uint8(float64(c.B)*alpha+0.5))})
		}
	}
	return gray
}

// luma is what drawing an sRGB image onto a TYPE_BYTE_GRAY one does, which is
// a colour conversion and not a weighted sum of the bytes: Java hands it to a
// ColorConvertOp between sRGB and CS_GRAY, so each component is linearised,
// weighted by CIE Y and put back through the sRGB transfer curve.
//
// It is a different conversion from the one SoftMask uses for the backdrop
// colour, which really is a weighted sum of the bytes. Both are Java's.
func luma(r, g, b uint8) uint8 {
	y := 0.2126*linearise(r) + 0.7152*linearise(g) + 0.0722*linearise(b)
	return uint8(delinearise(y)*255 + 0.5)
}

func linearise(v uint8) float64 {
	c := float64(v) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func delinearise(c float64) float64 {
	if c <= 0.0031308 {
		return c * 12.92
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

// adjustMask is the redraw at the end of PageDrawer.adjustImage: the mask was
// drawn at the device's scale, and this puts it back to the page's.
//
// DrawSoftMask has already worked out the transform and the size, and answers
// no transform for the case Java short-circuits -- a page whose own transform
// is a plain scale, which is every page rendered by renderImage.
func adjustMask(gray *goimage.Alpha, layer *rendering.SoftMaskLayer) *goimage.Alpha {
	if layer.Adjust == nil {
		return gray
	}
	adjusted := goimage.NewAlpha(goimage.Rect(0, 0, layer.AdjustWidth, layer.AdjustHeight))
	inverse, err := layer.Adjust.CreateInverse()
	if err != nil {
		return gray
	}
	point := make([]float64, 2)
	for y := 0; y < layer.AdjustHeight; y++ {
		for x := 0; x < layer.AdjustWidth; x++ {
			// The destination pixel's centre, mapped back, which is what a
			// nearest-neighbour drawImage samples at.
			point[0], point[1] = float64(x)+0.5, float64(y)+0.5
			inverse.TransformDoubles(point, 0, point, 0, 1)
			source := goimage.Point{
				X: int(math.Floor(point[0])),
				Y: int(math.Floor(point[1])),
			}
			if !source.In(gray.Bounds()) {
				continue
			}
			adjusted.SetAlpha(x, y, gray.AlphaAt(source.X, source.Y))
		}
	}
	return adjusted
}

// colorAt answers the masked colour under a device pixel.
func (s *softMaskSource) colorAt(x, y int) (goimagecolor.RGBA, bool) {
	c, painted := s.under.colorAt(x, y)
	if !painted {
		return c, false
	}
	c.A = uint8(math.Round(float64(c.A) * float64(s.alphaAt(x, y))))
	return c, true
}

// alphaAt is the fraction SoftPaintContext.getRaster multiplies a pixel's
// alpha by: the grey under it, through the transfer function, and the backdrop
// where the pixel falls outside the mask.
func (s *softMaskSource) alphaAt(x, y int) float32 {
	point := goimage.Point{X: x - s.originX, Y: y - s.originY}
	if !point.In(s.mask.Bounds()) {
		return float32(s.backdrop) / 255
	}
	grey := s.mask.AlphaAt(point.X, point.Y).A
	if s.transfer == nil {
		return float32(grey) / 255
	}
	if !s.known[grey] {
		result, err := s.transfer.Eval([]float32{float32(grey) / 255})
		if err != nil || len(result) == 0 {
			// "ignore exception, treat as outside"
			return float32(s.backdrop) / 255
		}
		s.transferred[grey] = result[0]
		s.known[grey] = true
	}
	return s.transferred[grey]
}
