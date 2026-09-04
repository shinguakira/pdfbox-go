// Package color holds the PDF colour spaces and the colour values in them.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color. The package name shadows
// the standard library's image/color; a file here that needs that one imports
// it under an alias.
package color

import (
	goimage "image"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
)

// PDColorSpace is a colour space, which says how the components of a colour
// value are to be read.
//
// Port of the abstract org.apache.pdfbox.pdmodel.graphics.color.PDColorSpace.
//
// The static create methods are not here: each dispatches to one of the dozen
// colour space classes, none of which this port has reached. Nor are
// toRGBImage and toRawImage, which return a java.awt.image.BufferedImage and
// belong with the image work of a later slice. See migration/STATUS.md.
type PDColorSpace interface {
	common.COSObjectable

	// Name returns the name of the colour space.
	Name() string

	// NumberOfComponents returns the number of components in this colour
	// space.
	NumberOfComponents() int

	// DefaultDecode returns the default decode array for an image with the
	// given number of bits per component.
	DefaultDecode(bitsPerComponent int) []float32

	// InitialColor returns the initial colour of this colour space, which is
	// what a content stream starts with.
	InitialColor() *PDColor

	// ToRGB converts a colour value of this colour space to RGB, with each
	// component between 0 and 1.
	ToRGB(value []float32) ([]float32, error)

	// ToRGBImage converts the given raster to an RGB image.
	//
	// Java returns a BufferedImage of TYPE_INT_RGB; the port returns an
	// *image.RGBA with every alpha byte 255, which is the nearest Go has.
	ToRGBImage(raster *awtimage.Raster) (goimage.Image, error)

	// ToRawImage returns an image in this colour space itself, or nil where
	// there is no way to hold one. Java can do it for an ICC based space and
	// for an indexed one over sRGB, because java.awt.image can carry an ICC
	// colour model; Go has no colour model beyond RGB and grey, so this port
	// returns nil throughout and the caller falls back to ToRGBImage, which is
	// what Java's callers do for the spaces that return null there.
	ToRawImage(raster *awtimage.Raster) (goimage.Image, error)
}

// newRGBImage returns the destination Java's `new BufferedImage(w, h,
// TYPE_INT_RGB)` gives, which is opaque.
func newRGBImage(width, height int) *goimage.RGBA {
	img := goimage.NewRGBA(goimage.Rect(0, 0, width, height))
	for i := 3; i < len(img.Pix); i += 4 {
		img.Pix[i] = 0xFF
	}
	return img
}

// setRGB writes one pixel of an image built by newRGBImage, clamping each
// component to a byte the way Java's raster does when a float is written into
// a TYPE_INT_RGB image.
func setRGB(img *goimage.RGBA, x, y int, r, g, b float32) {
	i := img.PixOffset(x, y)
	img.Pix[i] = clampToByte(r)
	img.Pix[i+1] = clampToByte(g)
	img.Pix[i+2] = clampToByte(b)
}

func clampToByte(value float32) byte {
	switch {
	case value <= 0:
		return 0
	case value >= 255:
		return 255
	}
	return byte(value)
}

// pixelAt reads one pixel of an image built by newRGBImage back as the three
// samples Java's getPixel returns.
func pixelAt(img *goimage.RGBA, x, y int, out []int) []int {
	i := img.PixOffset(x, y)
	out[0] = int(img.Pix[i])
	out[1] = int(img.Pix[i+1])
	out[2] = int(img.Pix[i+2])
	return out
}

// PatternColorSpace is a colour space whose values name a pattern rather than
// giving components directly.
//
// Port of the instanceof PDPattern checks in PDColor. PDPattern itself is not
// ported yet; this is the part of it PDColor asks about.
type PatternColorSpace interface {
	PDColorSpace

	// UnderlyingColorSpace returns the colour space the pattern paints in, or
	// nil for a coloured pattern that carries its own.
	UnderlyingColorSpace() PDColorSpace
}

// PDDeviceColorSpace is a colour space that directly specifies colours or
// shades of gray produced by an output device.
//
// Port of the abstract PDDeviceColorSpace. Java reaches the name through the
// dynamic dispatch of getName; Go has no such dispatch from an embedded type,
// so the name is held here and the concrete space passes it in.
type PDDeviceColorSpace struct {
	name string
}

// Name returns the name of the colour space.
func (c PDDeviceColorSpace) Name() string { return c.name }

// COSObject returns the name of the colour space as a COS name.
func (c PDDeviceColorSpace) COSObject() cos.Base { return cos.GetPDFName(c.name) }

// String returns the Java toString form, which is the name.
func (c PDDeviceColorSpace) String() string { return c.name }
