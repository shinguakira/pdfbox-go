// Package color holds the PDF colour spaces and the colour values in them.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color. The package name shadows
// the standard library's image/color; a file here that needs that one imports
// it under an alias.
package color

import (
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
