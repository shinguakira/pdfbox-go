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
// x/image/draw.
func (i *Image) interpolator() xdraw.Interpolator {
	if i.interpolation == rendering.NearestNeighbor {
		return xdraw.NearestNeighbor
	}
	return xdraw.CatmullRom
}

// DrawImage draws the given image through the given transform.
func (i *Image) DrawImage(pdImage pdimage.PDImage, at *geom.AffineTransform,
	subsampling int) error {
	source, err := pdImage.ImageOfRegion(nil, subsampling)
	if err != nil {
		return err
	}
	return i.drawSampled(source, at, nil)
}

// DrawStencil draws the given stencil mask filled with the given paint.
//
// A stencil is one bit per pixel and carries no colour of its own: where it is
// set, the paint shows. Java asks PDImage for a stencil image already coloured
// -- getStencilImage(paint) -- because Graphics2D can only draw an image; the
// port keeps the mask and the paint apart, which is what lets a shading or a
// tiling pattern through a stencil work the same way as a colour.
func (i *Image) DrawStencil(pdImage pdimage.PDImage, at *geom.AffineTransform,
	paint rendering.Paint) error {
	mask, err := pdImage.ImageOfRegion(nil, 1)
	if err != nil {
		return err
	}
	return i.drawSampled(mask, at, paint)
}

// drawSampled maps a source image onto the destination and composites it.
//
// Where paint is nil the source's own colours are drawn; where it is not, the
// source is a stencil and its coverage picks out where the paint shows.
func (i *Image) drawSampled(source goimage.Image, at *geom.AffineTransform,
	paint rendering.Paint) error {
	bounds := source.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return nil
	}

	// Sample into a scratch buffer the size of the destination rather than
	// onto the destination itself: what comes out has to go through the clip,
	// the alpha constant and the blend mode, and x/image/draw knows about none
	// of them.
	sampled := goimage.NewRGBA(i.dst.Bounds())
	i.interpolator().Transform(sampled,
		aff3Of(i.imageTransform(at, bounds.Dx(), bounds.Dy())),
		source, bounds, xdraw.Src, nil)

	var stencil paintSource
	if paint != nil {
		source, _, err := i.sourceOf(paint)
		if err != nil {
			return err
		}
		stencil = source
	}

	clip := i.clipCoverage()
	dstBounds := i.dst.Bounds()
	for y := dstBounds.Min.Y; y < dstBounds.Max.Y; y++ {
		for x := dstBounds.Min.X; x < dstBounds.Max.X; x++ {
			c := sampled.RGBAAt(x, y)
			if c.A == 0 {
				continue
			}
			coverage := float64(c.A) / 255
			if clip != nil {
				coverage *= float64(clip.AlphaAt(x, y).A) / 255
				if coverage == 0 {
					continue
				}
			}
			colour := unpremultiply(c)
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
func unpremultiply(c goimagecolor.RGBA) goimagecolor.RGBA {
	if c.A == 0 || c.A == 0xFF {
		return c
	}
	return goimagecolor.RGBA{
		R: uint8(int(c.R) * 255 / int(c.A)),
		G: uint8(int(c.G) * 255 / int(c.A)),
		B: uint8(int(c.B) * 255 / int(c.A)),
		A: c.A,
	}
}
