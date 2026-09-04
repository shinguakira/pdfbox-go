package image

import (
	goimage "image"
	goimagecolor "image/color"
	"math"
)

// The mask compositing of PDImageXObject.applyMask, and the scaling it needs.
//
// Java composes into a TYPE_INT_ARGB BufferedImage whose raster it addresses
// directly, and reaches three ways of doing it depending on whether the mask is
// soft and whether a /Matte is set. The port keeps the three, because the third
// is fixed point arithmetic whose rounding is visible, and works on an
// *image.RGBA, which is the same four bytes per pixel.

// applyMask composes an image and a mask.
//
// Port of PDImageXObject.applyMask.
func applyMask(img, mask goimage.Image, interpolateMask, isSoft bool, matte []float32,
	interpolateImage bool) goimage.Image {
	if mask == nil {
		return img
	}

	width := max(img.Bounds().Dx(), mask.Bounds().Dx())
	height := max(img.Bounds().Dy(), mask.Bounds().Dy())

	// scale mask to fit image, or image to fit mask, whichever is larger.
	// also make sure that mask is 8 bit gray and image is ARGB as this
	// is what needs to be returned.
	grayMask := toGray(mask, width, height, interpolateMask)
	argb := toRGBA(img, width, height, interpolateImage)

	switch {
	case !isSoft && matte == nil:
		// Java has a fast path here that combines the two buffers directly when
		// they are the same size; the port takes the sample loop below for it,
		// which computes the same thing -- an inverted mask sample written into
		// the alpha byte.
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				argb.Pix[argb.PixOffset(x, y)+3] = ^grayMask.Pix[grayMask.PixOffset(x, y)]
			}
		}

	case matte == nil:
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				sample := grayMask.Pix[grayMask.PixOffset(x, y)]
				if !isSoft {
					sample = ^sample
				}
				argb.Pix[argb.PixOffset(x, y)+3] = sample
			}
		}

	default:
		applyMatte(argb, grayMask, matte, width, height)
	}
	return argb
}

// applyMatte is the third arm of Java's applyMask, comment and arithmetic
// intact.
func applyMatte(argb *goimage.RGBA, grayMask *goimage.Gray, matte []float32, width, height int) {
	// Original code is to clamp component and alpha to [0f, 1f] as matte is,
	// and later expand to [0; 255] again (with rounding).
	// component = 255f * ((component / 255f - matte) / (alpha / 255f) + matte)
	//           = (255 * component - 255 * 255f * matte) / alpha + 255f * matte
	// There is a clearly visible factor 255 for most components in above formula,
	// i.e. max value is 255 * 255: 16 bits + sign.
	// Let's use faster fixed point integer arithmetics with Q16.15,
	// introducing neglible errors (0.001%).
	// Note: For "correct" rounding we increase the final matte value (m0h, m1h, m2h) by
	// a half an integer.
	const fraction = 15
	const factor = 255 << fraction
	m0 := javaRoundInt(factor*matte[0]) * 255
	m1 := javaRoundInt(factor*matte[1]) * 255
	m2 := javaRoundInt(factor*matte[2]) * 255
	m0h := m0/255 + (1 << (fraction - 1))
	m1h := m1/255 + (1 << (fraction - 1))
	m2h := m2/255 + (1 << (fraction - 1))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			a := int(grayMask.Pix[grayMask.PixOffset(x, y)])
			offset := argb.PixOffset(x, y)
			if a != 0 {
				argb.Pix[offset] = matteComponent(argb.Pix[offset], factor, m0, m0h, a)
				argb.Pix[offset+1] = matteComponent(argb.Pix[offset+1], factor, m1, m1h, a)
				argb.Pix[offset+2] = matteComponent(argb.Pix[offset+2], factor, m2, m2h, a)
			}
			argb.Pix[offset+3] = byte(a)
		}
	}
}

// matteComponent is one component of the matte arithmetic:
// clampColor(((component * factor - m) / a + mh) >> fraction).
func matteComponent(component byte, factor, m, mh, a int) byte {
	const fraction = 15
	return byte(clampColor(((int(component)*factor-m)/a + mh) >> fraction))
}

func clampColor(value int) int {
	switch {
	case value < 0:
		return 0
	case value > 255:
		return 255
	}
	return value
}

// javaRoundInt is java.lang.Math.round(float) returning an int.
func javaRoundInt(value float32) int { return javaRound(value) }

// toGray returns the mask as an 8 bit grey image of the given size.
//
// Java calls scaleImage with TYPE_BYTE_GRAY, which both converts and scales.
func toGray(img goimage.Image, width, height int, interpolate bool) *goimage.Gray {
	out := goimage.NewGray(goimage.Rect(0, 0, width, height))
	scaleInto(img, out.Bounds(), interpolate, func(x, y int, c goimagecolor.Color) {
		out.Set(x, y, goimagecolor.GrayModel.Convert(c))
	}, img.Bounds())
	return out
}

// toRGBA returns the image as an ARGB image of the given size.
func toRGBA(img goimage.Image, width, height int, interpolate bool) *goimage.RGBA {
	out := goimage.NewRGBA(goimage.Rect(0, 0, width, height))
	scaleInto(img, out.Bounds(), interpolate, func(x, y int, c goimagecolor.Color) {
		r, g, b, _ := c.RGBA()
		out.SetRGBA(x, y, goimagecolor.RGBA{
			R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: 255,
		})
	}, img.Bounds())
	return out
}

// scaleInto walks the destination and reads the source, nearest neighbour or
// bilinear.
//
// Java scales with an AffineTransformOp, bicubic for a small image and bilinear
// for a large one, and falls back to Graphics2D.drawImage. Go has neither, and
// the standard library has no resampler at all, so the port writes bilinear
// where Java interpolates and nearest neighbour where it does not. A scaled
// mask therefore differs from Java's on a bicubic path -- softly, in the
// gradient of the edge -- which is recorded in migration/STATUS.md.
func scaleInto(src goimage.Image, dst goimage.Rectangle, interpolate bool,
	set func(x, y int, c goimagecolor.Color), srcBounds goimage.Rectangle) {
	srcWidth := srcBounds.Dx()
	srcHeight := srcBounds.Dy()
	dstWidth := dst.Dx()
	dstHeight := dst.Dy()
	if srcWidth == 0 || srcHeight == 0 {
		return
	}
	interpolate = interpolate && (srcWidth != dstWidth || srcHeight != dstHeight)

	scaleX := float64(srcWidth) / float64(dstWidth)
	scaleY := float64(srcHeight) / float64(dstHeight)

	for y := 0; y < dstHeight; y++ {
		for x := 0; x < dstWidth; x++ {
			if !interpolate {
				sx := min(srcWidth-1, int(float64(x)*scaleX))
				sy := min(srcHeight-1, int(float64(y)*scaleY))
				set(x, y, src.At(srcBounds.Min.X+sx, srcBounds.Min.Y+sy))
				continue
			}
			set(x, y, bilinearAt(src, srcBounds,
				(float64(x)+0.5)*scaleX-0.5, (float64(y)+0.5)*scaleY-0.5))
		}
	}
}

// bilinearAt samples the source at a fractional position.
func bilinearAt(src goimage.Image, bounds goimage.Rectangle, fx, fy float64) goimagecolor.Color {
	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	dx := fx - float64(x0)
	dy := fy - float64(y0)

	clampX := func(v int) int { return min(bounds.Dx()-1, max(0, v)) }
	clampY := func(v int) int { return min(bounds.Dy()-1, max(0, v)) }

	var r, g, b, a float64
	for i := 0; i < 4; i++ {
		sx := clampX(x0 + i%2)
		sy := clampY(y0 + i/2)
		weight := (1 - dx) * (1 - dy)
		switch i {
		case 1:
			weight = dx * (1 - dy)
		case 2:
			weight = (1 - dx) * dy
		case 3:
			weight = dx * dy
		}
		sr, sg, sb, sa := src.At(bounds.Min.X+sx, bounds.Min.Y+sy).RGBA()
		r += float64(sr>>8) * weight
		g += float64(sg>>8) * weight
		b += float64(sb>>8) * weight
		a += float64(sa>>8) * weight
	}
	return goimagecolor.RGBA{
		R: uint8(clampColor(int(r + 0.5))),
		G: uint8(clampColor(int(g + 0.5))),
		B: uint8(clampColor(int(b + 0.5))),
		A: uint8(clampColor(int(a + 0.5))),
	}
}
