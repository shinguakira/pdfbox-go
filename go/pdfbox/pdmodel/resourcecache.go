package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
)

// ResourceCache keeps the objects read out of a resource dictionary, so that
// two pages sharing one indirect object share the object read from it.
//
// Port of org.apache.pdfbox.pdmodel.ResourceCache. The interface itself is
// declared in pdmodel/font, because it names PDFont and this package imports
// that one; this alias puts the Java name in the Java place. See
// migration/STATUS.md for the members whose types a later slice brings.
type ResourceCache = font.ResourceCache

// maxRemovals is how many times an object may be dropped from the cache before
// it is treated as stable and kept for good.
const maxRemovals = 3

// DefaultResourceCache is the cache a document uses unless it is given another.
//
// Port of org.apache.pdfbox.pdmodel.DefaultResourceCache. Java holds each entry
// through a SoftReference, so that the garbage collector may drop it under
// memory pressure; Go has no such reference, and the port holds them outright.
// The stable-cache bookkeeping is what keeps a repeatedly re-read object from
// being dropped, and it is ported as it stands.
type DefaultResourceCache struct {
	stableCacheEnabled bool

	fonts        map[*cos.Object]font.PDFont
	removedFonts map[int64]int
	stableFonts  map[int64]bool

	fontDescriptors map[*cos.Object]*font.PDFontDescriptor

	cidFonts map[*cos.Object]font.PDCIDFont

	colorSpaces *resourceMap[color.PDColorSpace]
	xobjects    *resourceMap[common.COSObjectable]
	shadings    *resourceMap[shading.Shading]
	patterns    *resourceMap[any]
	extGStates  *resourceMap[*state.PDExtendedGraphicsState]
	properties  *resourceMap[markedcontent.PropertyList]
}

var _ ResourceCache = (*DefaultResourceCache)(nil)

// NewDefaultResourceCache returns a cache that keeps an object it has had to
// drop too often.
func NewDefaultResourceCache() *DefaultResourceCache {
	return NewDefaultResourceCacheStable(true)
}

// NewDefaultResourceCacheStable returns a cache, enableStableCache saying
// whether an object dropped too often is kept for good.
func NewDefaultResourceCacheStable(enableStableCache bool) *DefaultResourceCache {
	return &DefaultResourceCache{
		stableCacheEnabled: enableStableCache,
		fonts:              map[*cos.Object]font.PDFont{},
		removedFonts:       map[int64]int{},
		stableFonts:        map[int64]bool{},
		fontDescriptors:    map[*cos.Object]*font.PDFontDescriptor{},
		cidFonts:           map[*cos.Object]font.PDCIDFont{},
		colorSpaces:        newResourceMap[color.PDColorSpace](),
		xobjects:           newResourceMap[common.COSObjectable](),
		shadings:           newResourceMap[shading.Shading](),
		patterns:           newResourceMap[any](),
		extGStates:         newResourceMap[*state.PDExtendedGraphicsState](),
		properties:         newResourceMap[markedcontent.PropertyList](),
	}
}

// GetFont returns the font read from the given indirect object, or nil.
func (c *DefaultResourceCache) GetFont(indirect *cos.Object) font.PDFont {
	return c.fonts[indirect]
}

// PutFont records the font read from the given indirect object.
func (c *DefaultResourceCache) PutFont(indirect *cos.Object, f font.PDFont) {
	c.fonts[indirect] = f
}

// RemoveFont drops the font read from the given indirect object and returns it.
//
// An object that has been dropped maxRemovals times is treated as stable: it is
// left in the cache and nothing is returned.
func (c *DefaultResourceCache) RemoveFont(indirect *cos.Object) font.PDFont {
	objectKey, hasKey := c.objectKey(indirect)
	if hasKey {
		if c.stableFonts[objectKey] {
			return nil
		}
		counter, ok := c.removedFonts[objectKey]
		if !ok {
			counter = 1
			c.removedFonts[objectKey] = counter
		}
		if counter < maxRemovals {
			counter++
			c.removedFonts[objectKey] = counter
		} else {
			c.stableFonts[objectKey] = true
			delete(c.removedFonts, objectKey)
			return nil
		}
	}
	f, ok := c.fonts[indirect]
	if !ok {
		return nil
	}
	delete(c.fonts, indirect)
	return f
}

// GetFontDescriptor returns the font descriptor read from the given indirect
// object, or nil.
func (c *DefaultResourceCache) GetFontDescriptor(indirect *cos.Object) *font.PDFontDescriptor {
	if len(c.fontDescriptors) == 0 {
		return nil
	}
	return c.fontDescriptors[indirect]
}

// PutFontDescriptor records the font descriptor read from the given indirect
// object.
func (c *DefaultResourceCache) PutFontDescriptor(indirect *cos.Object, fontDescriptor *font.PDFontDescriptor) {
	c.fontDescriptors[indirect] = fontDescriptor
}

// RemoveFontDescriptor drops the font descriptor read from the given indirect
// object and returns it.
func (c *DefaultResourceCache) RemoveFontDescriptor(indirect *cos.Object) *font.PDFontDescriptor {
	if len(c.fontDescriptors) == 0 {
		return nil
	}
	fd, ok := c.fontDescriptors[indirect]
	if !ok {
		return nil
	}
	delete(c.fontDescriptors, indirect)
	return fd
}

// objectKey returns the hash the stable-cache bookkeeping is keyed on, the
// second result being false where the cache does not do that bookkeeping or the
// object has no key.
func (c *DefaultResourceCache) objectKey(indirect *cos.Object) (int64, bool) {
	if !c.stableCacheEnabled {
		return 0, false
	}
	key := indirect.Key()
	if key == nil {
		return 0, false
	}
	return key.InternalHash(), true
}

// GetCIDFont returns the CIDFont read from the given indirect object, or nil.
func (c *DefaultResourceCache) GetCIDFont(indirect *cos.Object) font.PDCIDFont {
	return c.cidFonts[indirect]
}

// PutCIDFont records the CIDFont read from the given indirect object.
func (c *DefaultResourceCache) PutCIDFont(indirect *cos.Object, cidFont font.PDCIDFont) {
	c.cidFonts[indirect] = cidFont
}

// RemoveCIDFont drops the CIDFont read from the given indirect object and
// returns it.
func (c *DefaultResourceCache) RemoveCIDFont(indirect *cos.Object) font.PDCIDFont {
	cidFont, ok := c.cidFonts[indirect]
	if !ok {
		return nil
	}
	delete(c.cidFonts, indirect)
	return cidFont
}

// GetColorSpace returns the colour space read from the given indirect object,
// or nil where the cache has none.
//
// Port of DefaultResourceCache.getColorSpace.
func (c *DefaultResourceCache) GetColorSpace(indirect *cos.Object) color.PDColorSpace {
	return c.colorSpaces.get(indirect)
}

// PutColorSpace records the colour space read from the given indirect object.
//
// Port of DefaultResourceCache.put(COSObject, PDColorSpace).
func (c *DefaultResourceCache) PutColorSpace(indirect *cos.Object, space color.PDColorSpace) {
	c.colorSpaces.put(indirect, space)
}

// RemoveColorSpace drops the colour space read from the given indirect object
// and returns it.
//
// Port of DefaultResourceCache.removeColorSpace.
func (c *DefaultResourceCache) RemoveColorSpace(indirect *cos.Object) color.PDColorSpace {
	return c.colorSpaces.remove(c, indirect)
}

// resourceMap is one kind of cached resource, with the stable-cache
// bookkeeping Java repeats for each of them.
//
// Java writes getX, put and removeX out per kind, seven times over, differing
// only in the type and in which three maps they touch. The port writes the
// bookkeeping once; the fonts above keep their hand-written form, since they
// were ported before Go could express this and rewriting them would change no
// behaviour.
type resourceMap[T any] struct {
	entries map[*cos.Object]T
	removed map[int64]int
	stable  map[int64]bool
}

// newResourceMap returns an empty map of one kind of resource.
func newResourceMap[T any]() *resourceMap[T] {
	return &resourceMap[T]{
		entries: map[*cos.Object]T{},
		removed: map[int64]int{},
		stable:  map[int64]bool{},
	}
}

// get returns the resource read from the given indirect object, and the zero
// value where the cache has none.
func (m *resourceMap[T]) get(indirect *cos.Object) T {
	return m.entries[indirect]
}

// put records the resource read from the given indirect object.
func (m *resourceMap[T]) put(indirect *cos.Object, value T) {
	m.entries[indirect] = value
}

// remove drops the resource read from the given indirect object and returns it.
//
// An object that has been dropped maxRemovals times is treated as stable: it is
// left in the cache and nothing is returned.
func (m *resourceMap[T]) remove(cache *DefaultResourceCache, indirect *cos.Object) T {
	var zero T
	objectKey, hasKey := cache.objectKey(indirect)
	if hasKey {
		if m.stable[objectKey] {
			return zero
		}
		counter, ok := m.removed[objectKey]
		if !ok {
			counter = 1
			m.removed[objectKey] = counter
		}
		if counter < maxRemovals {
			counter++
			m.removed[objectKey] = counter
		} else {
			m.stable[objectKey] = true
			delete(m.removed, objectKey)
			return zero
		}
	}
	value, ok := m.entries[indirect]
	if !ok {
		return zero
	}
	delete(m.entries, indirect)
	return value
}

// GetXObject returns the XObject read from the given indirect object, or nil
// where the cache has none.
//
// Port of DefaultResourceCache.getXObject.
func (c *DefaultResourceCache) GetXObject(indirect *cos.Object) common.COSObjectable {
	return c.xobjects.get(indirect)
}

// PutXObject records the XObject read from the given indirect object.
//
// Port of DefaultResourceCache.put(COSObject, PDXObject).
func (c *DefaultResourceCache) PutXObject(indirect *cos.Object, xobject common.COSObjectable) {
	c.xobjects.put(indirect, xobject)
}

// RemoveXObject drops the XObject read from the given indirect object and
// returns it.
//
// Port of DefaultResourceCache.removeXObject.
func (c *DefaultResourceCache) RemoveXObject(indirect *cos.Object) common.COSObjectable {
	return c.xobjects.remove(c, indirect)
}

// GetShading returns the shading read from the given indirect object, or nil
// where the cache has none.
func (c *DefaultResourceCache) GetShading(indirect *cos.Object) shading.Shading {
	return c.shadings.get(indirect)
}

// PutShading records the shading read from the given indirect object.
func (c *DefaultResourceCache) PutShading(indirect *cos.Object, sh shading.Shading) {
	c.shadings.put(indirect, sh)
}

// RemoveShading drops the shading read from the given indirect object and
// returns it.
func (c *DefaultResourceCache) RemoveShading(indirect *cos.Object) shading.Shading {
	return c.shadings.remove(c, indirect)
}

// GetPattern returns the pattern read from the given indirect object, or nil
// where the cache has none.
//
// It answers an any, because graphics/pattern imports this package and so
// PDAbstractPattern cannot be named here; PDResources.PatternOfName says the
// same.
func (c *DefaultResourceCache) GetPattern(indirect *cos.Object) any {
	return c.patterns.get(indirect)
}

// PutPattern records the pattern read from the given indirect object.
func (c *DefaultResourceCache) PutPattern(indirect *cos.Object, pattern any) {
	c.patterns.put(indirect, pattern)
}

// RemovePattern drops the pattern read from the given indirect object and
// returns it.
func (c *DefaultResourceCache) RemovePattern(indirect *cos.Object) any {
	return c.patterns.remove(c, indirect)
}

// GetExtGState returns the extended graphics state read from the given indirect
// object, or nil where the cache has none.
func (c *DefaultResourceCache) GetExtGState(indirect *cos.Object) *state.PDExtendedGraphicsState {
	return c.extGStates.get(indirect)
}

// PutExtGState records the extended graphics state read from the given indirect
// object.
func (c *DefaultResourceCache) PutExtGState(indirect *cos.Object,
	extGState *state.PDExtendedGraphicsState) {
	c.extGStates.put(indirect, extGState)
}

// RemoveExtState drops the extended graphics state read from the given indirect
// object and returns it.
//
// Java names it removeExtState, not removeExtGState.
func (c *DefaultResourceCache) RemoveExtState(indirect *cos.Object) *state.PDExtendedGraphicsState {
	return c.extGStates.remove(c, indirect)
}

// GetProperties returns the property list read from the given indirect object,
// or nil where the cache has none.
func (c *DefaultResourceCache) GetProperties(indirect *cos.Object) markedcontent.PropertyList {
	return c.properties.get(indirect)
}

// PutProperties records the property list read from the given indirect object.
func (c *DefaultResourceCache) PutProperties(indirect *cos.Object,
	propertyList markedcontent.PropertyList) {
	c.properties.put(indirect, propertyList)
}

// RemoveProperties drops the property list read from the given indirect object
// and returns it.
func (c *DefaultResourceCache) RemoveProperties(indirect *cos.Object) markedcontent.PropertyList {
	return c.properties.remove(c, indirect)
}
