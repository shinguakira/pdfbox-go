package pattern

// The /Pattern colour space.
//
// Port of org.apache.pdfbox.pdmodel.graphics.color.PDPattern, which lives in
// graphics.color there. It cannot live there here: it names PDAbstractPattern
// and PDResources, and both of those sit above graphics/color -- pattern
// imports pdmodel, which imports graphics/color. So the class lives in the
// package that can name every side of it, and graphics/color reaches it
// through the constructor this file registers from its init.
//
// PDSpecialColorSpace, which PDPattern extends in Java, adds nothing at all:
// it is an empty abstract class marking the colour spaces that add features to
// an underlying one. SpecialColorSpace in graphics/color is that marker.

import (
	"fmt"

	goimage "image"

	awtimage "github.com/shinguakira/pdfbox-go/go/awt/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

func init() {
	color.NewPatternColorSpace = func(resources color.ResourcesLike,
		underlying color.PDColorSpace) color.PDColorSpace {
		return newPDPattern(resources, underlying)
	}
}

// emptyPattern is a pattern which leaves no marks on the page.
//
// Port of the private static PDPattern.EMPTY_PATTERN.
var emptyPattern = color.NewPDColorOfComponents([]float32{}, nil)

// PDPattern is a colour space whose values name a pattern rather than giving
// components: either a tiling pattern or a shading pattern.
//
// Port of PDPattern.
type PDPattern struct {
	array                *cos.Array
	resources            color.ResourcesLike
	underlyingColorSpace color.PDColorSpace
}

var (
	_ color.PDColorSpace      = (*PDPattern)(nil)
	_ color.PatternColorSpace = (*PDPattern)(nil)
	_ color.SpecialColorSpace = (*PDPattern)(nil)
)

// NewPDPattern creates a new pattern colour space over the given resources.
//
// Port of PDPattern(PDResources).
func NewPDPattern(resources color.ResourcesLike) *PDPattern {
	return newPDPattern(resources, nil)
}

// NewPDPatternOfColorSpace creates a new uncoloured tiling pattern colour
// space over the given resources and underlying colour space.
//
// Port of PDPattern(PDResources, PDColorSpace).
func NewPDPatternOfColorSpace(resources color.ResourcesLike,
	colorSpace color.PDColorSpace) *PDPattern {
	return newPDPattern(resources, colorSpace)
}

// newPDPattern is the body the two constructors share; Java writes the array
// out in each.
func newPDPattern(resources color.ResourcesLike, colorSpace color.PDColorSpace) *PDPattern {
	p := &PDPattern{
		array:                cos.NewArray(),
		resources:            resources,
		underlyingColorSpace: colorSpace,
	}
	p.array.Add(cos.Pattern)
	if colorSpace != nil {
		p.array.Add(colorSpace.COSObject())
	}
	return p
}

// COSObject returns the colour space array.
func (p *PDPattern) COSObject() cos.Base { return p.array }

// Name returns "Pattern".
func (p *PDPattern) Name() string { return cos.Pattern.Name() }

// NumberOfComponents panics: a pattern has no components of its own.
//
// Java throws UnsupportedOperationException, which is unchecked.
func (p *PDPattern) NumberOfComponents() int {
	panic("color: a pattern colour space has no number of components")
}

// DefaultDecode panics, as NumberOfComponents does.
func (p *PDPattern) DefaultDecode(int) []float32 {
	panic("color: a pattern colour space has no default decode")
}

// InitialColor returns a pattern which leaves no marks on the page.
func (p *PDPattern) InitialColor() *color.PDColor { return emptyPattern }

// ToRGB panics, as NumberOfComponents does.
func (p *PDPattern) ToRGB([]float32) ([]float32, error) {
	panic("color: a pattern colour space cannot be converted to RGB")
}

// ToRGBImage panics, as NumberOfComponents does.
func (p *PDPattern) ToRGBImage(*awtimage.Raster) (goimage.Image, error) {
	panic("color: a pattern colour space cannot be converted to an RGB image")
}

// ToRawImage panics, as NumberOfComponents does.
func (p *PDPattern) ToRawImage(*awtimage.Raster) (goimage.Image, error) {
	panic("color: a pattern colour space cannot be converted to a raw image")
}

// IsSpecialColorSpace marks this as one of the colour spaces that add features
// to an underlying one; see the file comment.
func (p *PDPattern) IsSpecialColorSpace() {}

// Pattern returns the pattern the given colour names, and an error where the
// resources do not hold one under that name.
func (p *PDPattern) Pattern(c *color.PDColor) (Pattern, error) {
	lookup, canLookUp := p.resources.(patternResources)
	if !canLookUp {
		return nil, fmt.Errorf("pattern %v was not found", c.PatternName())
	}
	found, err := lookup.GetPattern(c.PatternName())
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, fmt.Errorf("pattern %v was not found", c.PatternName())
	}
	pattern, isPattern := found.(Pattern)
	if !isPattern {
		return nil, fmt.Errorf("pattern %v was not found", c.PatternName())
	}
	return pattern, nil
}

// patternResources is the half of PDResources that Pattern uses.
//
// color.ResourcesLike names only what a colour space asks of the resources, and
// a pattern lookup is not one of those, so this asks for it by shape -- the
// device PDResources already uses for its extended graphics state and property
// list caches.
type patternResources interface {
	// GetPattern returns the pattern resource of the given name, or nil.
	GetPattern(name *cos.Name) (any, error)
}

// UnderlyingColorSpace returns the underlying colour space where this is an
// uncoloured tiling pattern, and nil otherwise.
func (p *PDPattern) UnderlyingColorSpace() color.PDColorSpace {
	return p.underlyingColorSpace
}

// String returns "Pattern", which is PDPattern.toString.
func (p *PDPattern) String() string { return "Pattern" }

// patternOfResources is what PDResources.GetPattern answers through, so that
// pdmodel can build a pattern without importing this package.
func init() {
	pdmodel.NewPatternOfDictionary = func(dictionary *cos.Dictionary,
		stream *cos.Stream, cache pdmodel.ResourceCache) (any, error) {
		if stream != nil {
			return NewPDAbstractPatternOfStream(stream, cache)
		}
		return NewPDAbstractPattern(dictionary, cache)
	}
}
