package raster

// Shrinking an image before it is drawn, as PDFBox does below half size.
//
// PageDrawer.drawBufferedImage draws an image it shrinks below
// imageDownscalingOptimizationThreshold -- 0.5 unless the caller sets it --
// under the quality and bicubic hints in two steps: getScaledInstance(w, h,
// SCALE_SMOOTH) averages it down to the size it will have on the page, and
// drawImage puts that down. The comment in PDFBox says why: drawImage alone
// "has terrible quality when scaling down".
//
// getScaledInstance is three pieces of the JDK, and each is here:
// OffScreenImageSource sends the image a row at a time, as its colour model's
// getRGB answers each pixel; AreaAveragingScaleFilter averages the rows into
// the smaller image, in floats, premultiplied, and un-premultiplies each
// result; and ImageRepresentation keeps what comes out in an image of the
// source's colour model where it can -- TYPE_INT_RGB for a TYPE_INT_RGB source,
// TYPE_INT_ARGB for anything else.

import (
	goimage "image"
	goimagecolor "image/color"
	"math"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// drawBufferedImage is the part of PageDrawer.drawBufferedImage that decides
// how an image is drawn: shrunk first where it is drawn below the threshold,
// drawn as it is otherwise.
func (i *Image) drawBufferedImage(source goimage.Image, at *geom.AffineTransform,
	paint rendering.Paint) error {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return nil
	}
	imageTransform := at.Clone()
	imageTransform.Scale(1.0/float64(width), -1.0/float64(height))
	imageTransform.Translate(0, -float64(height))

	if i.renderingQuality && i.interpolation == rendering.Bicubic {
		imageMatrix := util.NewMatrixFromAffineTransform(imageTransform)
		graphicsMatrix := util.NewMatrixFromAffineTransform(i.transform)
		scaleX := float32(math.Abs(float64(imageMatrix.ScalingFactorX() * graphicsMatrix.ScalingFactorX())))
		scaleY := float32(math.Abs(float64(imageMatrix.ScalingFactorY() * graphicsMatrix.ScalingFactorY())))
		if scaleX < i.downscalingThreshold || scaleY < i.downscalingThreshold {
			w := javaRoundFloat(float32(width) * scaleX)
			h := javaRoundFloat(float32(height) * scaleY)
			if w >= 1 && h >= 1 {
				scaled := scaledInstance(source, w, h)
				// remove the scale (extracted from w and h, to have it from
				// the rounded values hoping to reverse the rounding)
				imageTransform.Scale(float64(float32(1)/float32(w)*float32(width)),
					float64(float32(1)/float32(h)*float32(height)))
				full := i.transform.Clone()
				full.Concatenate(imageTransform)
				return i.drawThrough(scaled, full, paint)
			}
		}
	}
	full := i.transform.Clone()
	full.Concatenate(imageTransform)
	return i.drawThrough(source, full, paint)
}

// javaRoundFloat is Math.round(float): the nearest int, a half rounding up.
func javaRoundFloat(v float32) int {
	return javaIntOf(math.Floor(float64(v) + 0.5))
}

// scaledInstance is getScaledInstance(w, h, SCALE_SMOOTH) of the image, as the
// image drawImage then draws.
func scaledInstance(src goimage.Image, w, h int) goimage.Image {
	bounds := src.Bounds()
	filter := newAreaAveraging(bounds.Dx(), bounds.Dy(), w, h)
	row := make([]uint32, bounds.Dx())
	for y := 0; y < bounds.Dy(); y++ {
		for x := range row {
			row[x] = sentRGB(src, bounds.Min.X+x, bounds.Min.Y+y)
		}
		filter.accumPixels(y, row)
	}

	if isIntRGB(src) {
		// ImageRepresentation keeps a TYPE_INT_RGB source's colour model, and
		// an INT_RGB image has no alpha to keep.
		out := goimage.NewRGBA(goimage.Rect(0, 0, w, h))
		for k, argb := range filter.out {
			out.Pix[4*k], out.Pix[4*k+1], out.Pix[4*k+2], out.Pix[4*k+3] =
				uint8(argb>>16), uint8(argb>>8), uint8(argb), 0xFF
		}
		return out
	}
	out := goimage.NewNRGBA(goimage.Rect(0, 0, w, h))
	for k, argb := range filter.out {
		out.Pix[4*k], out.Pix[4*k+1], out.Pix[4*k+2], out.Pix[4*k+3] =
			uint8(argb>>16), uint8(argb>>8), uint8(argb), uint8(argb>>24)
	}
	return out
}

// isIntRGB reports whether the image is one the port holds for TYPE_INT_RGB:
// an image.RGBA, every pixel of it opaque.
func isIntRGB(src goimage.Image) bool {
	rgba, ok := src.(*goimage.RGBA)
	if !ok {
		return false
	}
	for k := 3; k < len(rgba.Pix); k += 4 {
		if rgba.Pix[k] != 0xFF {
			return false
		}
	}
	return true
}

// sentRGB is a pixel as OffScreenImageSource hands it to the filter and the
// filter reads it: model.getRGB of it, straight ARGB.
//
// A TYPE_INT_RGB or TYPE_INT_ARGB image goes with its own DirectColorModel,
// whose getRGB is the pixel, opaque where the model has no alpha. A
// TYPE_BYTE_GRAY image has a ComponentColorModel, which sendPixels converts
// with image.getRGB; that takes a linear grey to sRGB, which is grayToSRGB.
func sentRGB(src goimage.Image, x, y int) uint32 {
	switch img := src.(type) {
	case *goimage.RGBA:
		p := img.Pix[img.PixOffset(x, y):]
		if p[3] == 0xFF {
			return 0xFF000000 | uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2])
		}
	case *goimage.NRGBA:
		p := img.Pix[img.PixOffset(x, y):]
		return uint32(p[3])<<24 | uint32(p[0])<<16 | uint32(p[1])<<8 | uint32(p[2])
	case *goimage.Gray:
		g := uint32(grayToSRGB[img.Pix[img.PixOffset(x, y)]])
		return 0xFF000000 | g<<16 | g<<8 | g
	}
	c := goimagecolor.NRGBAModel.Convert(src.At(x, y)).(goimagecolor.NRGBA)
	return uint32(c.A)<<24 | uint32(c.R)<<16 | uint32(c.G)<<8 | uint32(c.B)
}

// areaAveraging is AreaAveragingScaleFilter, fed a row at a time as
// OffScreenImageSource feeds it, with the rows it hands on kept in out.
type areaAveraging struct {
	srcWidth, srcHeight   int
	destWidth, destHeight int

	reds, greens, blues, alphas []float32
	savedy, savedyrem           int

	out []uint32
}

func newAreaAveraging(srcWidth, srcHeight, destWidth, destHeight int) *areaAveraging {
	return &areaAveraging{
		srcWidth: srcWidth, srcHeight: srcHeight,
		destWidth: destWidth, destHeight: destHeight,
		out: make([]uint32, destWidth*destHeight),
	}
}

// calcRow is AreaAveragingScaleFilter.calcRow.
func (f *areaAveraging) calcRow() []uint32 {
	origmult := float32(f.srcWidth) * float32(f.srcHeight)
	outpix := make([]uint32, f.destWidth)
	for x := 0; x < f.destWidth; x++ {
		mult := origmult
		a := javaRoundFloat(f.alphas[x] / mult)
		if a <= 0 {
			a = 0
		} else if a >= 255 {
			a = 255
		} else {
			// un-premultiply the components (by modifying mult here, we
			// are effectively doing the divide by mult and divide by
			// alpha in the same step)
			mult = f.alphas[x] / 255
		}
		r := clampComponent(javaRoundFloat(f.reds[x] / mult))
		g := clampComponent(javaRoundFloat(f.greens[x] / mult))
		b := clampComponent(javaRoundFloat(f.blues[x] / mult))
		outpix[x] = uint32(a)<<24 | uint32(r)<<16 | uint32(g)<<8 | uint32(b)
	}
	return outpix
}

func clampComponent(v int) int {
	return min(255, max(0, v))
}

// accumPixels is AreaAveragingScaleFilter.accumPixels for one source row.
//
// The float arithmetic is Java's, operation for operation. Each product is
// converted to float32 on its own, which the Go specification says rounds it,
// so that no platform fuses it with the sum it goes into.
func (f *areaAveraging) accumPixels(y int, pixels []uint32) {
	if f.reds == nil {
		f.reds = make([]float32, f.destWidth)
		f.greens = make([]float32, f.destWidth)
		f.blues = make([]float32, f.destWidth)
		f.alphas = make([]float32, f.destWidth)
	}
	sy := y
	syrem := f.destHeight
	var dy, dyrem int
	if sy == 0 {
		dy = 0
		dyrem = 0
	} else {
		dy = f.savedy
		dyrem = f.savedyrem
	}
	for sy < y+1 {
		var amty int
		if dyrem == 0 {
			for i := 0; i < f.destWidth; i++ {
				f.alphas[i], f.reds[i], f.greens[i], f.blues[i] = 0, 0, 0, 0
			}
			dyrem = f.srcHeight
		}
		if syrem < dyrem {
			amty = syrem
		} else {
			amty = dyrem
		}
		sx := 0
		dx := 0
		sxrem := 0
		dxrem := f.srcWidth
		var a, r, g, b float32
		for sx < f.srcWidth {
			if sxrem == 0 {
				sxrem = f.destWidth
				rgb := pixels[sx]
				// getRGB() always returns non-premultiplied components
				a = float32(rgb >> 24)
				r = float32(rgb >> 16 & 0xff)
				g = float32(rgb >> 8 & 0xff)
				b = float32(rgb & 0xff)
				// premultiply the components if necessary
				if a != 255.0 {
					ascale := a / 255.0
					r = float32(r * ascale)
					g = float32(g * ascale)
					b = float32(b * ascale)
				}
			}
			var amtx int
			if sxrem < dxrem {
				amtx = sxrem
			} else {
				amtx = dxrem
			}
			mult := float32(amtx) * float32(amty)
			f.alphas[dx] += float32(mult * a)
			f.reds[dx] += float32(mult * r)
			f.greens[dx] += float32(mult * g)
			f.blues[dx] += float32(mult * b)
			if sxrem -= amtx; sxrem == 0 {
				sx++
			}
			if dxrem -= amtx; dxrem == 0 {
				dx++
				dxrem = f.srcWidth
			}
		}
		if dyrem -= amty; dyrem == 0 {
			outpix := f.calcRow()
			for {
				copy(f.out[dy*f.destWidth:(dy+1)*f.destWidth], outpix)
				dy++
				if syrem -= amty; !(syrem >= amty && amty == f.srcHeight) {
					break
				}
			}
		} else {
			syrem -= amty
		}
		if syrem == 0 {
			syrem = f.destHeight
			sy++
		}
	}
	f.savedyrem = dyrem
	f.savedy = dy
}

// grayToSRGB is what TYPE_BYTE_GRAY's getRGB answers for each grey: its
// ComponentColorModel holds a linear grey, and getRGB converts it to sRGB.
// The values are the JDK's, printed by getRGB of a one-pixel TYPE_BYTE_GRAY
// image for each sample from 0 to 255; TestTheGreyTableIsJavas holds a few.
var grayToSRGB = [256]uint8{
	0, 13, 22, 28, 34, 38, 42, 46, 50, 53, 56, 59, 61, 64, 66, 69,
	71, 73, 75, 77, 79, 81, 83, 85, 86, 88, 90, 92, 93, 95, 96, 98,
	99, 101, 102, 104, 105, 106, 108, 109, 110, 112, 113, 114, 115, 117, 118, 119,
	120, 121, 122, 124, 125, 126, 127, 128, 129, 130, 131, 132, 133, 134, 135, 136,
	137, 138, 139, 140, 141, 142, 143, 144, 145, 146, 147, 148, 148, 149, 150, 151,
	152, 153, 154, 155, 155, 156, 157, 158, 159, 159, 160, 161, 162, 163, 163, 164,
	165, 166, 167, 167, 168, 169, 170, 170, 171, 172, 173, 173, 174, 175, 175, 176,
	177, 178, 178, 179, 180, 180, 181, 182, 182, 183, 184, 185, 185, 186, 187, 187,
	188, 189, 189, 190, 190, 191, 192, 192, 193, 194, 194, 195, 196, 196, 197, 197,
	198, 199, 199, 200, 200, 201, 202, 202, 203, 203, 204, 205, 205, 206, 206, 207,
	208, 208, 209, 209, 210, 210, 211, 212, 212, 213, 213, 214, 214, 215, 215, 216,
	216, 217, 218, 218, 219, 219, 220, 220, 221, 221, 222, 222, 223, 223, 224, 224,
	225, 226, 226, 227, 227, 228, 228, 229, 229, 230, 230, 231, 231, 232, 232, 233,
	233, 234, 234, 235, 235, 236, 236, 237, 237, 238, 238, 238, 239, 239, 240, 240,
	241, 241, 242, 242, 243, 243, 244, 244, 245, 245, 246, 246, 246, 247, 247, 248,
	248, 249, 249, 250, 250, 251, 251, 251, 252, 252, 253, 253, 254, 254, 255, 255,
}
