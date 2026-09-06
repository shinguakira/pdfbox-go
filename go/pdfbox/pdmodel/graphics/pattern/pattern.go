// Package pattern holds the two pattern types a pattern colour space paints
// with: a tiling pattern, which is a content stream repeated over a grid, and a
// shading pattern, which is a shading.
//
// Port of org.apache.pdfbox.pdmodel.graphics.pattern. Java gives each class a
// file; they are short and name each other through the factory, so the port
// keeps them together.
package pattern

import (
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The two pattern types.
const (
	// TypeTilingPattern is a pattern painted by repeating a content stream.
	TypeTilingPattern = 1
	// TypeShadingPattern is a pattern painted by a shading.
	TypeShadingPattern = 2
)

// Pattern is what a concrete pattern supplies.
//
// PDAbstractPattern is an abstract class in Java; the port splits it into this
// interface for the one abstract method and the embedded struct below for the
// state and the concrete ones.
type Pattern interface {
	common.COSObjectable

	// Dictionary returns the pattern dictionary, which getCOSObject narrows to
	// in Java.
	Dictionary() *cos.Dictionary

	// PatternType returns which of the two this is.
	PatternType() int
}

// PDAbstractPattern carries the state and the concrete methods of a pattern.
//
// Port of the non-abstract half of PDAbstractPattern.
type PDAbstractPattern struct {
	patternDictionary *cos.Dictionary
}

// NewPDAbstractPattern creates the right pattern for the given dictionary.
//
// Port of the static PDAbstractPattern.create.
func NewPDAbstractPattern(dictionary *cos.Dictionary,
	resourceCache pdmodel.ResourceCache) (Pattern, error) {
	return newPattern(dictionary, nil, resourceCache)
}

// NewPDAbstractPatternOfStream is NewPDAbstractPattern for a pattern held as a
// stream, which a tiling pattern is; see PDTilingPattern.
func NewPDAbstractPatternOfStream(stream *cos.Stream,
	resourceCache pdmodel.ResourceCache) (Pattern, error) {
	return newPattern(&stream.Dictionary, stream, resourceCache)
}

// newPattern is the body the two constructors share.
func newPattern(dictionary *cos.Dictionary, stream *cos.Stream,
	resourceCache pdmodel.ResourceCache) (Pattern, error) {
	patternType := dictionary.GetIntDefault(cos.PatternType, 0)
	switch patternType {
	case TypeTilingPattern:
		return newPDTilingPattern(dictionary, stream, resourceCache), nil
	case TypeShadingPattern:
		return NewPDShadingPatternOf(dictionary), nil
	}
	return nil, fmt.Errorf("Error: Unknown pattern type %d", patternType)
}

// InitPattern is the protected PDAbstractPattern() constructor.
func (p *PDAbstractPattern) InitPattern() {
	p.patternDictionary = cos.NewDictionary()
	p.patternDictionary.SetName(cos.Type, cos.Pattern.Name())
}

// InitPatternOf is the protected PDAbstractPattern(COSDictionary) constructor.
func (p *PDAbstractPattern) InitPatternOf(dictionary *cos.Dictionary) {
	p.patternDictionary = dictionary
}

// COSObject returns the pattern dictionary.
func (p *PDAbstractPattern) COSObject() cos.Base { return p.patternDictionary }

// Dictionary returns the pattern dictionary, typed.
func (p *PDAbstractPattern) Dictionary() *cos.Dictionary { return p.patternDictionary }

// SetPaintType sets the /PaintType entry.
func (p *PDAbstractPattern) SetPaintType(paintType int) {
	p.patternDictionary.SetInt(cos.PaintType, paintType)
}

// Type returns the type of object that this is.
func (p *PDAbstractPattern) Type() string { return cos.Pattern.Name() }

// SetPatternType sets the /PatternType entry.
func (p *PDAbstractPattern) SetPatternType(patternType int) {
	p.patternDictionary.SetInt(cos.PatternType, patternType)
}

// Matrix returns the /Matrix entry, which maps the pattern's own space into the
// space it is used in.
func (p *PDAbstractPattern) Matrix() *util.Matrix {
	return util.CreateMatrix(p.patternDictionary.GetDictionaryObject(cos.Matrix))
}

// SetMatrix sets the /Matrix entry from an affine transform.
func (p *PDAbstractPattern) SetMatrix(transform *geom.AffineTransform) {
	matrix := cos.NewArray()
	values := make([]float64, 6)
	transform.GetMatrix(values)
	for _, v := range values {
		matrix.Add(cos.NewFloat(float32(v)))
	}
	p.patternDictionary.SetItem(cos.Matrix, matrix)
}

// The paint types of a tiling pattern.
const (
	// PaintColored means the pattern carries its own colour.
	PaintColored = 1
	// PaintUncolored means the pattern is painted in the colour the pattern
	// colour space was given.
	PaintUncolored = 2
)

// The tiling types of a tiling pattern.
const (
	// TilingConstantSpacing keeps the spacing constant.
	TilingConstantSpacing = 1
	// TilingNoDistortion does not distort the pattern cell.
	TilingNoDistortion = 2
	// TilingConstantSpacingFasterTiling keeps the spacing constant and tiles
	// faster.
	TilingConstantSpacingFasterTiling = 3
)

// PDTilingPattern is a pattern painted by repeating a content stream over a
// grid.
//
// Port of PDTilingPattern, which is a PDContentStream.
type PDTilingPattern struct {
	PDAbstractPattern

	// stream is the pattern dictionary as a stream, which holds the content.
	// Java needs no such field: COSStream extends COSDictionary there. See
	// shading.PDShading for the same arrangement.
	stream        *cos.Stream
	resourceCache pdmodel.ResourceCache
}

var _ Pattern = (*PDTilingPattern)(nil)

// NewPDTilingPattern creates an empty tiling pattern over a fresh stream.
func NewPDTilingPattern(codecs cos.CodecProvider) *PDTilingPattern {
	stream := cos.NewStream(codecs)
	p := &PDTilingPattern{stream: stream}
	p.InitPatternOf(&stream.Dictionary)
	p.Dictionary().SetName(cos.Type, cos.Pattern.Name())
	p.Dictionary().SetInt(cos.PatternType, TypeTilingPattern)
	// Resources required per PDF specification; when missing, pattern is not
	// displayed in Adobe Reader
	p.SetResources(pdmodel.NewPDResources())
	return p
}

// NewPDTilingPatternOf creates a tiling pattern over the given dictionary.
func NewPDTilingPatternOf(dictionary *cos.Dictionary) *PDTilingPattern {
	return newPDTilingPattern(dictionary, nil, nil)
}

// NewPDTilingPatternOfCache creates a tiling pattern over the given dictionary,
// reading its resources through the given cache.
func NewPDTilingPatternOfCache(dictionary *cos.Dictionary,
	resourceCache pdmodel.ResourceCache) *PDTilingPattern {
	return newPDTilingPattern(dictionary, nil, resourceCache)
}

// newPDTilingPattern is the body the three constructors share.
func newPDTilingPattern(dictionary *cos.Dictionary, stream *cos.Stream,
	resourceCache pdmodel.ResourceCache) *PDTilingPattern {
	p := &PDTilingPattern{stream: stream, resourceCache: resourceCache}
	p.InitPatternOf(dictionary)
	return p
}

// PatternType returns TypeTilingPattern.
func (p *PDTilingPattern) PatternType() int { return TypeTilingPattern }

// SetPaintType sets the /PaintType entry.
func (p *PDTilingPattern) SetPaintType(paintType int) {
	p.Dictionary().SetInt(cos.PaintType, paintType)
}

// PaintType returns the /PaintType entry, zero where there is none.
func (p *PDTilingPattern) PaintType() int {
	return p.Dictionary().GetIntDefault(cos.PaintType, 0)
}

// SetTilingType sets the /TilingType entry.
func (p *PDTilingPattern) SetTilingType(tilingType int) {
	p.Dictionary().SetInt(cos.TilingType, tilingType)
}

// TilingType returns the /TilingType entry, zero where there is none.
func (p *PDTilingPattern) TilingType() int {
	return p.Dictionary().GetIntDefault(cos.TilingType, 0)
}

// SetXStep sets the /XStep entry.
func (p *PDTilingPattern) SetXStep(xStep float32) {
	p.Dictionary().SetFloat(cos.XStep, xStep)
}

// XStep returns the /XStep entry, zero where there is none.
func (p *PDTilingPattern) XStep() float32 {
	return p.Dictionary().GetFloat(cos.XStep, 0)
}

// SetYStep sets the /YStep entry.
func (p *PDTilingPattern) SetYStep(yStep float32) {
	p.Dictionary().SetFloat(cos.YStep, yStep)
}

// YStep returns the /YStep entry, zero where there is none.
func (p *PDTilingPattern) YStep() float32 {
	return p.Dictionary().GetFloat(cos.YStep, 0)
}

// ContentStream returns the pattern as a stream.
//
// Java casts the dictionary to a COSStream, which throws where the pattern is
// not one; the port answers nil there, because the cast cannot be written.
func (p *PDTilingPattern) ContentStream() *common.PDStream {
	if p.stream == nil {
		return nil
	}
	return common.NewPDStream(p.stream)
}

// ContentsForRandomAccess returns the content of the pattern, or nil where the
// pattern dictionary is not a stream.
func (p *PDTilingPattern) ContentsForRandomAccess() (pdfio.RandomAccessRead, error) {
	if p.stream == nil {
		return nil, nil
	}
	return p.stream.CreateView()
}

// Resources returns the resources of the pattern, or nil where it has none.
func (p *PDTilingPattern) Resources() *pdmodel.PDResources {
	resources := p.Dictionary().GetCOSDictionary(cos.Resources)
	if resources == nil {
		return nil
	}
	return pdmodel.NewPDResourcesOfCache(resources, p.resourceCache)
}

// SetResources sets the resources of the pattern.
func (p *PDTilingPattern) SetResources(resources *pdmodel.PDResources) {
	if resources == nil {
		p.Dictionary().SetItem(cos.Resources, nil)
		return
	}
	p.Dictionary().SetItem(cos.Resources, resources.COSObject())
}

// BBox returns the /BBox of the pattern cell, or nil where there is none.
func (p *PDTilingPattern) BBox() *common.PDRectangle {
	bbox := p.Dictionary().GetCOSArray(cos.BBox)
	if bbox == nil {
		return nil
	}
	return common.NewPDRectangleOfCOSArray(bbox)
}

// SetBBox sets the /BBox of the pattern cell, and removes it for nil.
func (p *PDTilingPattern) SetBBox(bbox *common.PDRectangle) {
	if bbox == nil {
		p.Dictionary().RemoveItem(cos.BBox)
		return
	}
	p.Dictionary().SetItem(cos.BBox, bbox.COSArray())
}

// PDShadingPattern is a pattern painted by a shading.
//
// Port of PDShadingPattern.
type PDShadingPattern struct {
	PDAbstractPattern

	extendedGraphicsState *state.PDExtendedGraphicsState
	shading               shading.Shading
}

var _ Pattern = (*PDShadingPattern)(nil)

// NewPDShadingPattern creates an empty shading pattern.
func NewPDShadingPattern() *PDShadingPattern {
	p := &PDShadingPattern{}
	p.InitPattern()
	p.Dictionary().SetInt(cos.PatternType, TypeShadingPattern)
	return p
}

// NewPDShadingPatternOf creates a shading pattern over the given dictionary.
func NewPDShadingPatternOf(resourceDictionary *cos.Dictionary) *PDShadingPattern {
	p := &PDShadingPattern{}
	p.InitPatternOf(resourceDictionary)
	return p
}

// PatternType returns TypeShadingPattern.
func (p *PDShadingPattern) PatternType() int { return TypeShadingPattern }

// ExtendedGraphicsState returns the /ExtGState of the pattern, or nil where it
// has none.
func (p *PDShadingPattern) ExtendedGraphicsState() *state.PDExtendedGraphicsState {
	if p.extendedGraphicsState == nil {
		if base := p.Dictionary().GetCOSDictionary(cos.ExtGState); base != nil {
			p.extendedGraphicsState = state.NewPDExtendedGraphicsStateOf(base)
		}
	}
	return p.extendedGraphicsState
}

// SetExtendedGraphicsState sets the /ExtGState of the pattern.
func (p *PDShadingPattern) SetExtendedGraphicsState(
	extendedGraphicsState *state.PDExtendedGraphicsState) {
	p.extendedGraphicsState = extendedGraphicsState
	if extendedGraphicsState == nil {
		p.Dictionary().SetItem(cos.ExtGState, nil)
		return
	}
	p.Dictionary().SetItem(cos.ExtGState, extendedGraphicsState.COSObject())
}

// Shading returns the /Shading of the pattern, or nil where it has none.
func (p *PDShadingPattern) Shading() (shading.Shading, error) {
	if p.shading == nil {
		if base := p.Dictionary().GetDictionaryObject(cos.Shading); base != nil {
			// Java reads the entry with getCOSDictionary, which answers a
			// COSStream too since one extends the other; the port passes the
			// base along so that a mesh shading keeps its stream.
			created, err := shading.NewPDShading(base)
			if err != nil {
				return nil, err
			}
			p.shading = created
		}
	}
	return p.shading, nil
}

// SetShading sets the /Shading of the pattern.
func (p *PDShadingPattern) SetShading(shadingResources shading.Shading) {
	p.shading = shadingResources
	if shadingResources == nil {
		p.Dictionary().SetItem(cos.Shading, nil)
		return
	}
	p.Dictionary().SetItem(cos.Shading, shadingResources.COSObject())
}
