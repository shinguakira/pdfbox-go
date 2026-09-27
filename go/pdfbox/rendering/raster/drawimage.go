package raster

// Placing an image.
//
// Java hands the whole job to Graphics2D.drawImage(BufferedImage,
// AffineTransform, ImageObserver): the transform maps image pixels onto the
// page and Java2D samples. The port does the mapping itself, samples through
// golang.org/x/image/draw, and composites the result the way every other
// drawing here is composited -- through the clip, the alpha constant and the
// blend mode.

import (
	goimage "image"
	goimagecolor "image/color"
	"math"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/math/f64"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	pdimage "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
)

// aff3Of is a geom.AffineTransform as x/image wants it.
//
// Both are the same six numbers in the same roles: x' = m00 x + m01 y + m02.
func aff3Of(at *geom.AffineTransform) f64.Aff3 {
	return f64.Aff3{
		at.ScaleX(), at.ShearX(), at.TranslateX(),
		at.ShearY(), at.ScaleY(), at.TranslateY(),
	}
}

// imageTransform is the transform from a source image's pixels to the device,
// which is Java's
//
//	imageTransform = new AffineTransform(at);
//	imageTransform.scale(1.0 / width, -1.0 / height);
//	imageTransform.translate(0, -height);
//	full = graphics.getTransform(); full.concatenate(imageTransform);
//
// `at` maps the unit square onto where the image goes, so the two steps put
// the image's pixels into that square: the scale flips it, because an image's
// first row is its top and the unit square's y runs up.
func (i *Image) imageTransform(at *geom.AffineTransform, width, height int) *geom.AffineTransform {
	transform := at.Clone()
	transform.Scale(1/float64(width), -1/float64(height))
	transform.Translate(0, -float64(height))

	full := i.transform.Clone()
	full.Concatenate(transform)
	return full
}

// interpolator is the sampling the rendering hint asks for.
//
// Java's KEY_INTERPOLATION takes VALUE_INTERPOLATION_NEAREST_NEIGHBOR or
// VALUE_INTERPOLATION_BICUBIC, and PageDrawer sets the first for an image
// scaled up with /Interpolate false. CatmullRom is the bicubic of
// x/image/draw, which DrawSurface still samples with; an image's bicubic is
// Java2D's own, drawImageJava2D.
func (i *Image) interpolator() xdraw.Interpolator {
	if i.interpolation == rendering.NearestNeighbor {
		return xdraw.NearestNeighbor
	}
	return xdraw.CatmullRom
}

// javaClipBounds is the bounds of the clip region Java2D makes of the clip in
// force: the clip's box in device space with each side at the first pixel
// centre inside it, which is Region.clipRound, ceil(v - 0.5). TransformHelper
// starts stepping at its top left corner. Without a clip it is the surface.
func (i *Image) javaClipBounds() goimage.Rectangle {
	surface := i.dst.Bounds()
	if i.clip == nil {
		return surface
	}
	box := i.clip.Bounds2D()
	if box == nil {
		return surface
	}
	if i.clipTransform != nil {
		box = transformedRect(i.clipTransform, box)
	}
	return goimage.Rect(clipRound(box.X), clipRound(box.Y),
		clipRound(box.X+box.Width), clipRound(box.Y+box.Height)).Intersect(surface)
}

// clipRound is Region.clipRound.
func clipRound(v float64) int {
	return javaIntOf(math.Ceil(v - 0.5))
}

// DrawImage draws the given image through the given transform.
func (i *Image) DrawImage(pdImage pdimage.PDImage, at *geom.AffineTransform,
	subsampling int) error {
	source, err := pdImage.ImageOfRegion(nil, subsampling)
	if err != nil {
		return err
	}
	return i.drawBufferedImage(source, at, nil)
}

// DrawStencil draws the given stencil mask filled with the given paint.
//
// A stencil is one bit per pixel and carries no colour of its own: where it is
// set, the paint shows. Java asks PDImage for a stencil image already coloured
// -- getStencilImage(paint) -- because Graphics2D can only draw an image; the
// port keeps the mask and the paint apart, which is what lets a shading or a
// tiling pattern through a stencil work the same way as a colour.
//
// It asks for the stencil image all the same, in an opaque black: what it
// wants from it is the **alpha**, which getStencilImage sets from the mask's
// bits and from nothing else. ImageOfRegion would answer the wrong thing --
// an image mask has no colour space, so what comes back is opaque wherever it
// comes back at all, and a stencil would paint its whole rectangle.
func (i *Image) DrawStencil(pdImage pdimage.PDImage, at *geom.AffineTransform,
	paint rendering.Paint) error {
	mask, err := pdImage.StencilImage(goimagecolor.NRGBA{A: 0xFF})
	if err != nil {
		return err
	}
	switch p := paint.(type) {
	case rendering.ColorPaint:
		// A stencil filled with a colour is drawn as any other image is --
		// getStencilImage(paint), then drawBufferedImage -- and a stencil filled
		// with a pattern is not: Java renders the pattern and scales the mask
		// by a route of its own.
		return i.drawBufferedImage(mask, at, paint)
	case rendering.SoftMaskedPaint:
		if _, isColour := p.Paint.(rendering.ColorPaint); isColour {
			// The same, with the colour seen through the graphics state's soft
			// mask. PageDrawer draws a stencil under a soft mask as a texture
			// unless the stencil has a /Mask or an /SMask of its own; this is
			// the one that has, and getStencilImage fills it in its own pixels.
			filled, err := i.filledStencil(mask, paint)
			if err != nil {
				return err
			}
			return i.drawBufferedImage(filled, at, nil)
		}
	}
	return i.drawSampled(mask, at, paint)
}

// filledStencil is getStencilImage(paint) for any paint: an image of the
// stencil's size filled with the paint, and clear where the stencil does not
// paint. stencil is opaque where it paints, as StencilImage makes it.
//
// Java fills it through the BufferedImage's own Graphics2D, whose device space
// is the image's pixels, so the paint is asked for its colour at (x, y) of the
// image, wherever the image is drawn afterwards. For a colour seen through a
// soft mask, which is what getNonStrokingPaint answers under one, that reads
// the mask at the top left of the page.
//
// A pixel the paint gives no alpha stays clear, as a SrcOver of nothing onto
// the clear BufferedImage leaves it.
func (i *Image) filledStencil(stencil goimage.Image, fill rendering.Paint) (*goimage.NRGBA, error) {
	source, alpha, err := i.sourceOf(fill)
	if err != nil {
		return nil, err
	}
	bounds := stencil.Bounds()
	filled := goimage.NewNRGBA(goimage.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			if _, _, _, a := stencil.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA(); a == 0 {
				continue
			}
			c, painted := source.colorAt(x, y)
			if !painted {
				continue
			}
			c.A = uint8(math.Round(float64(c.A) * alpha))
			if c.A == 0 {
				continue
			}
			filled.SetNRGBA(x, y, c)
		}
	}
	return filled, nil
}

// drawSampled maps a source image onto the destination through the transform
// that takes the unit square to where it goes, and composites it.
//
// Where paint is nil the source's own colours are drawn; where it is not, the
// source is a stencil and its coverage picks out where the paint shows.
func (i *Image) drawSampled(source goimage.Image, at *geom.AffineTransform,
	paint rendering.Paint) error {
	bounds := source.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return nil
	}
	return i.drawThrough(source, i.imageTransform(at, bounds.Dx(), bounds.Dy()), paint)
}

// drawThrough is drawSampled with the transform from the image's pixels to the
// device already worked out.
func (i *Image) drawThrough(source goimage.Image, transform *geom.AffineTransform,
	paint rendering.Paint) error {
	bounds := source.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return nil
	}

	// Sample into a scratch buffer rather than onto the destination itself:
	// what comes out has to go through the clip, the alpha constant and the
	// blend mode, and x/image/draw knows about none of them.
	// Premultiplied, because that is what x/image/draw writes and what
	// image.RGBA means. unpremultiply below turns each sample into the
	// straight colour the rest of this package speaks.
	//
	// The buffer and the loop cover where the image lands, not the whole
	// surface: a page of small images used to allocate and walk a page-sized
	// buffer for each of them, and the rows are read out of Pix rather than
	// through RGBAAt, which tests the bounds and works out the offset on every
	// call. What is drawn does not change.
	area := i.dst.Bounds().Intersect(paddedRect(transformedRect(transform,
		geom.NewRectangle2D(float64(bounds.Min.X), float64(bounds.Min.Y),
			float64(bounds.Dx()), float64(bounds.Dy())))))
	if area.Empty() {
		return nil
	}
	sampled := goimage.NewRGBA(area)
	if i.interpolation == rendering.NearestNeighbor {
		xdraw.NearestNeighbor.Transform(sampled, aff3Of(transform), source, bounds, xdraw.Src, nil)
	} else {
		// Java2D's own bicubic drawImage; see transformhelper.go.
		drawImageJava2D(sampled, source, transform, i.javaClipBounds())
	}

	var stencil paintSource
	if paint != nil {
		source, _, err := i.sourceOf(paint)
		if err != nil {
			return err
		}
		stencil = source
	}

	clip := i.clipCoverage()
	if clip != nil {
		area = area.Intersect(clip.Bounds())
	}
	for y := area.Min.Y; y < area.Max.Y; y++ {
		row := sampled.Pix[sampled.PixOffset(area.Min.X, y):][:4*area.Dx()]
		var clipRow []uint8
		if clip != nil {
			clipRow = clip.Pix[clip.PixOffset(area.Min.X, y):][:area.Dx()]
		}
		for offset := 0; offset+3 < len(row); offset += 4 {
			alpha := row[offset+3]
			if alpha == 0 {
				continue
			}
			coverage := float64(alpha) / 255
			if clipRow != nil {
				if clipRow[offset/4] == 0 {
					continue
				}
				coverage *= float64(clipRow[offset/4]) / 255
			}
			x := area.Min.X + offset/4
			colour := unpremultiply(goimagecolor.RGBA{
				R: row[offset], G: row[offset+1], B: row[offset+2], A: alpha,
			})
			if stencil != nil {
				painted := false
				if colour, painted = stencil.colorAt(x, y); !painted {
					continue
				}
			}
			i.blendPixel(x, y, colour, coverage*i.alphaConstant)
		}
	}
	return nil
}

// unpremultiply turns an image/color.RGBA, which is premultiplied, back into
// the straight colour the compositor works in.
func unpremultiply(c goimagecolor.RGBA) goimagecolor.NRGBA {
	if c.A == 0 || c.A == 0xFF {
		return goimagecolor.NRGBA(c)
	}
	return goimagecolor.NRGBA{
		R: uint8(int(c.R) * 255 / int(c.A)),
		G: uint8(int(c.G) * 255 / int(c.A)),
		B: uint8(int(c.B) * 255 / int(c.A)),
		A: c.A,
	}
}
