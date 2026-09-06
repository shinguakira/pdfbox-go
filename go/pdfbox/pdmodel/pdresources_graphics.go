package pdmodel

// The drawing half of PDResources: the XObjects and shadings a content stream
// draws through.
//
// Port of PDResources.getXObject and getShading, kept out of pdresources.go the
// way the colour space half is. Each waited on the type it returns; slice 9
// brings both.

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
)

// xobjectCache is the part of a resource cache that keeps XObjects.
//
// Java declares getXObject, put(COSObject, PDXObject) and removeXObject on
// ResourceCache itself; the port's ResourceCache is declared in pdmodel/font,
// which cannot name an XObject, so this package asks the cache for them by
// shape, the way it asks for the extended graphics states.
type xobjectCache interface {
	GetXObject(indirect *cos.Object) common.COSObjectable
	PutXObject(indirect *cos.Object, xobject common.COSObjectable)
}

// shadingCache is the part of a resource cache that keeps shadings.
type shadingCache interface {
	GetShading(indirect *cos.Object) shading.Shading
	PutShading(indirect *cos.Object, sh shading.Shading)
}

// GetXObject returns the XObject resource with the given name, or nil where the
// resources have none.
//
// Port of getXObject. Java's return type is PDXObject; the port has no
// interface over the four kinds, so it answers what CreateXObject does.
func (r *PDResources) GetXObject(name *cos.Name) (common.COSObjectable, error) {
	indirect := r.getIndirect(cos.XObject, name)
	cache, cacheKeepsThem := r.cache.(xobjectCache)
	if cacheKeepsThem && indirect != nil {
		if cached := cache.GetXObject(indirect); cached != nil {
			return cached, nil
		}
	}

	// get the instance
	var xobject common.COSObjectable
	var err error
	value := r.get(cos.XObject, name)
	switch {
	case value == nil:
		xobject = nil
	case isCOSObject(value):
		xobject, err = CreateXObject(value.(*cos.Object).Object(), r)
	default:
		xobject, err = CreateXObject(value, r)
	}
	if err != nil {
		return nil, err
	}
	if cacheKeepsThem && indirect != nil && r.isAllowedCache(xobject) {
		cache.PutXObject(indirect, xobject)
	}
	return xobject, nil
}

// isCOSObject is Java's `value instanceof COSObject`.
func isCOSObject(value cos.Base) bool {
	_, ok := value.(*cos.Object)
	return ok
}

// isAllowedCache reports whether the given XObject may be cached, which an
// image whose colour space might be one of the page's is not.
//
// Port of the private isAllowedCache.
func (r *PDResources) isAllowedCache(xobject common.COSObjectable) bool {
	if img, isImage := xobject.(*image.PDImageXObject); isImage {
		colorSpaceName := img.Stream().GetCOSName(cos.ColorSpace)
		if colorSpaceName != nil {
			// don't cache if it might use page resources, see PDFBOX-2370 and
			// PDFBOX-3484
			if colorSpaceName == cos.DeviceCMYK && r.HasColorSpace(cos.DefaultCMYK) {
				return false
			}
			if colorSpaceName == cos.DeviceRGB && r.HasColorSpace(cos.DefaultRGB) {
				return false
			}
			if colorSpaceName == cos.DeviceGray && r.HasColorSpace(cos.DefaultGray) {
				return false
			}
			if r.HasColorSpace(colorSpaceName) {
				return false
			}
		}
	}
	return true
}

// GetShading returns the shading resource with the given name, or nil where the
// resources have none.
//
// Port of getShading.
func (r *PDResources) GetShading(name *cos.Name) (shading.Shading, error) {
	indirect := r.getIndirect(cos.Shading, name)
	cache, cacheKeepsThem := r.cache.(shadingCache)
	if cacheKeepsThem && indirect != nil {
		if cached := cache.GetShading(indirect); cached != nil {
			return cached, nil
		}
	}

	// get the instance
	var sh shading.Shading
	if dict, isDictionary := asResourceDictionary(r.get(cos.Shading, name)); isDictionary {
		created, err := shading.NewPDShading(dict)
		if err != nil {
			return nil, err
		}
		sh = created
	}

	if cacheKeepsThem && indirect != nil {
		cache.PutShading(indirect, sh)
	}
	return sh, nil
}

// AddShading adds the given shading to the resources, and returns the name it
// went in under.
//
// Port of add(PDShading).
func (r *PDResources) AddShading(sh shading.Shading) *cos.Name {
	return r.add(cos.Shading, "sh", sh)
}

// AddPattern adds the given pattern to the resources, and returns the name it
// went in under.
//
// Port of add(PDAbstractPattern). It takes the objectable rather than the
// pattern, because graphics/pattern imports this package; GetPattern says the
// same.
func (r *PDResources) AddPattern(pattern common.COSObjectable) *cos.Name {
	return r.add(cos.Pattern, "p", pattern)
}

// PutShading sets the shading resource with the given name.
//
// Port of put(COSName, PDShading).
func (r *PDResources) PutShading(name *cos.Name, sh shading.Shading) {
	r.put(cos.Shading, name, sh)
}

// PutPattern sets the pattern resource with the given name.
//
// Port of put(COSName, PDAbstractPattern), taking the objectable for the reason
// AddPattern gives.
func (r *PDResources) PutPattern(name *cos.Name, pattern common.COSObjectable) {
	r.put(cos.Pattern, name, pattern)
}
