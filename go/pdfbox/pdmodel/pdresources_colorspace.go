package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
)

// The colour space half of PDResources.
//
// Port of PDResources.getColorSpace, kept in a file of its own the way the
// encryption half of PDDocument is, and satisfying color.ResourcesLike so that
// a colour space can look a name up without the two packages importing each
// other.

var _ color.ResourcesLike = (*PDResources)(nil)

// colorSpaceCache is the colour space half of Java's ResourceCache.
//
// The port's ResourceCache interface lives in the font package, which the
// colour spaces must not pull in, so the two halves are separate interfaces and
// DefaultResourceCache satisfies both.
type colorSpaceCache interface {
	GetColorSpace(indirect *cos.Object) color.PDColorSpace
	PutColorSpace(indirect *cos.Object, space color.PDColorSpace)
}

// ColorSpace returns the colour space resource with the given name.
//
// Port of getColorSpace(COSName).
func (r *PDResources) ColorSpace(name *cos.Name) (color.PDColorSpace, error) {
	return r.ColorSpaceOfName(name, false)
}

// ColorSpaceOfName returns the colour space resource with the given name.
//
// Port of getColorSpace(COSName, boolean), which is for PDFBox internal use
// only; others should use ColorSpace. wasDefault says whether the current
// colour space was reached through a default colour space.
func (r *PDResources) ColorSpaceOfName(name *cos.Name, wasDefault bool) (color.PDColorSpace, error) {
	indirect := r.getIndirect(cos.ColorSpace, name)
	if cached := r.CachedColorSpace(indirect); cached != nil {
		return cached, nil
	}

	// get the instance
	var colorSpace color.PDColorSpace
	var err error
	if object := r.get(cos.ColorSpace, name); object != nil {
		colorSpace, err = color.CreateWithResources(object, r, wasDefault)
	} else {
		colorSpace, err = color.CreateWithResources(name, r, wasDefault)
	}
	if err != nil {
		return nil, err
	}

	// we can't cache PDPattern, because it holds page resources, see PDFBOX-2370
	//
	// It holds the resources it was built from and looks a pattern name up in
	// them, so another page handed it from the cache looks the name up in this
	// page's resources and does not find it. The line used to say /Pattern was
	// not ported and could not reach here; slice 9 ported it. See
	// patterncache_test.go.
	// Java tests `colorSpace instanceof PDPattern`; graphics/pattern is above
	// this package and cannot be imported here, and the only colour space whose
	// name is /Pattern is that one.
	if colorSpace.Name() != cos.Pattern.Name() {
		r.CacheColorSpace(indirect, colorSpace)
	}
	return colorSpace, nil
}

// CachedColorSpace returns the colour space cached for the given indirect
// object, or nil where there is none or there is no cache.
func (r *PDResources) CachedColorSpace(indirect *cos.Object) color.PDColorSpace {
	if indirect == nil || r.cache == nil {
		return nil
	}
	if cache, ok := r.cache.(colorSpaceCache); ok {
		return cache.GetColorSpace(indirect)
	}
	return nil
}

// CacheColorSpace records a colour space against its indirect object.
func (r *PDResources) CacheColorSpace(indirect *cos.Object, space color.PDColorSpace) {
	if indirect == nil || r.cache == nil || space == nil {
		return
	}
	if cache, ok := r.cache.(colorSpaceCache); ok {
		cache.PutColorSpace(indirect, space)
	}
}
