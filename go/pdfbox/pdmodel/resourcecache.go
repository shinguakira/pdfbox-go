package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
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
