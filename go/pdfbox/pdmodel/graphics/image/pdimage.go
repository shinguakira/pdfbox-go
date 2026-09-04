// Package image holds the images a PDF carries: the image XObject, the inline
// image, and the reader that turns their samples into a picture.
//
// Port of org.apache.pdfbox.pdmodel.graphics.image.
package image

import (
	goimage "image"
	goimagecolor "image/color"
	"io"

	awtgeom "github.com/shinguakira/pdfbox-go/go/awt/geom"
	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// PDImage is an image in a PDF document.
//
// Port of the interface org.apache.pdfbox.pdmodel.graphics.image.PDImage.
type PDImage interface {
	// COSDictionary returns the dictionary below this image, which is what
	// Java's getCOSObject returns.
	COSDictionary() *cos.Dictionary

	// Image returns the content of this image as an RGB image.
	Image() (goimage.Image, error)

	// RawRaster returns the raw samples, undecoded and unconverted.
	RawRaster() (*awtimage.Raster, error)

	// RawImage returns an image in this image's own colour space, or nil where
	// there is no way to hold one.
	RawImage() (goimage.Image, error)

	// ImageOfRegion returns part of this image, subsampled.
	ImageOfRegion(region *awtgeom.Rectangle, subsampling int) (goimage.Image, error)

	// StencilImage returns the content of this stencil mask as an image filled
	// with the given colour.
	//
	// Java takes a java.awt.Paint, which may be a gradient or a pattern; the
	// port takes a solid colour, which is what a stencil in a content stream is
	// filled with unless the fill colour is itself a pattern, and a pattern
	// paint is slice 9's. See migration/STATUS.md.
	StencilImage(paint goimagecolor.Color) (goimage.Image, error)

	// CreateInputStream returns the decoded content of this image.
	CreateInputStream() (io.Reader, error)

	// CreateInputStreamStopping returns the content of this image, decoded
	// through every filter up to but not including the first named one.
	CreateInputStreamStopping(stopFilters []string) (io.Reader, error)

	// CreateInputStreamWithOptions returns the decoded content of this image,
	// letting a filter honour the given options.
	CreateInputStreamWithOptions(options *filter.DecodeOptions) (io.Reader, error)

	// IsEmpty reports whether this image has no data.
	IsEmpty() bool

	// IsStencil reports whether this image is a stencil mask.
	IsStencil() bool

	// SetStencil says whether this image is a stencil mask.
	SetStencil(isStencil bool)

	// BitsPerComponent returns how many bits one sample takes.
	BitsPerComponent() int

	// SetBitsPerComponent sets how many bits one sample takes.
	SetBitsPerComponent(bitsPerComponent int)

	// ColorSpace returns the colour space of this image.
	ColorSpace() (color.PDColorSpace, error)

	// SetColorSpace sets the colour space of this image.
	SetColorSpace(colorSpace color.PDColorSpace)

	// Height returns the height in pixels.
	Height() int

	// SetHeight sets the height in pixels.
	SetHeight(height int)

	// Width returns the width in pixels.
	Width() int

	// SetWidth sets the width in pixels.
	SetWidth(width int)

	// SetDecode sets the /Decode array.
	SetDecode(decode *cos.Array)

	// Decode returns the /Decode array, or nil where there is none.
	Decode() *cos.Array

	// Interpolate reports whether the image is to be interpolated when scaled.
	Interpolate() bool

	// SetInterpolate says whether the image is to be interpolated when scaled.
	SetInterpolate(value bool)

	// Suffix returns the file suffix the image data would have on its own,
	// "png" for a lossless one, or the empty string where it has none.
	Suffix() string
}
