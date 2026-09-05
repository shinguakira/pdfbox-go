package pdmodel

// The drawing half of PDResources: the XObjects, shadings, extended graphics
// states and property lists a content stream draws through.
//
// Port of PDResources.getXObject, getShading, getExtGState and getProperties,
// kept out of pdresources.go the way the colour space half is. Each waited on
// the type it returns; slice 9 brings the last of them.

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
)

// xobjectCache is the part of a resource cache that keeps XObjects.
//
// Java declares getXObject, put(COSObject, PDXObject) and removeXObject on
// ResourceCache itself; the port's ResourceCache is declared in pdmodel/font,
// which cannot name an XObject, so this package asks the cache for them by
// shape, the way it asks for the patterns.
type xobjectCache interface {
	GetXObject(indirect *cos.Object) common.COSObjectable
	PutXObject(indirect *cos.Object, xobject common.COSObjectable)
}

// shadingCache is the part of a resource cache that keeps shadings.
type shadingCache interface {
	GetShading(indirect *cos.Object) shading.Shading
	PutShading(indirect *cos.Object, sh shading.Shading)
}

// extGStateCache is the part of a resource cache that keeps extended graphics
// states.
type extGStateCache interface {
	GetExtGState(indirect *cos.Object) *state.PDExtendedGraphicsState
	PutExtGState(indirect *cos.Object, extGState *state.PDExtendedGraphicsState)
}

// propertiesCache is the part of a resource cache that keeps property lists.
type propertiesCache interface {
	GetProperties(indirect *cos.Object) markedcontent.PropertyList
	PutProperties(indirect *cos.Object, propertyList markedcontent.PropertyList)
}

// XObject returns the XObject resource with the given name, or nil where the
// resources have none.
//
// Port of getXObject. Java's return type is PDXObject; the port has no
// interface over the four kinds, so it answers what CreateXObject does.
func (r *PDResources) XObject(name *cos.Name) (common.COSObjectable, error) {
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

// Shading returns the shading resource with the given name, or nil where the
// resources have none.
//
// Port of getShading.
func (r *PDResources) Shading(name *cos.Name) (shading.Shading, error) {
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

// ExtGState returns the extended graphics state resource with the given name,
// or nil where the resources have none.
//
// Port of getExtGState.
func (r *PDResources) ExtGState(name *cos.Name) *state.PDExtendedGraphicsState {
	indirect := r.getIndirect(cos.ExtGState, name)
	cache, cacheKeepsThem := r.cache.(extGStateCache)
	if cacheKeepsThem && indirect != nil {
		if cached := cache.GetExtGState(indirect); cached != nil {
			return cached
		}
	}

	// get the instance
	var extGState *state.PDExtendedGraphicsState
	if dict, isDictionary := asResourceDictionary(r.get(cos.ExtGState, name)); isDictionary {
		extGState = state.NewPDExtendedGraphicsStateOfCache(dict, r.Cache())
	}

	if cacheKeepsThem && indirect != nil {
		cache.PutExtGState(indirect, extGState)
	}
	return extGState
}

// Properties returns the property list resource with the given name, or nil
// where the resources have none.
//
// Port of getProperties.
func (r *PDResources) Properties(name *cos.Name) markedcontent.PropertyList {
	indirect := r.getIndirect(cos.Properties, name)
	cache, cacheKeepsThem := r.cache.(propertiesCache)
	if cacheKeepsThem && indirect != nil {
		if cached := cache.GetProperties(indirect); cached != nil {
			return cached
		}
	}

	// get the instance
	var propertyList markedcontent.PropertyList
	if dict, isDictionary := asResourceDictionary(r.get(cos.Properties, name)); isDictionary {
		propertyList = markedcontent.CreatePropertyList(dict)
	}

	if cacheKeepsThem && indirect != nil {
		cache.PutProperties(indirect, propertyList)
	}
	return propertyList
}
