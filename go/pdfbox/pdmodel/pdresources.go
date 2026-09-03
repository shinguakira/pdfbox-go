// Package pdmodel holds the PDF document model: the document, its pages, and
// the resources they draw with.
//
// Port of org.apache.pdfbox.pdmodel. As in pdmodel/common the PD prefix of the
// Java class names is kept, since it marks the document model across the whole
// tree rather than repeating the name of one package.
package pdmodel

import (
	"strconv"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// PDResources is a set of resources available at the page/pages/stream level.
//
// Port of org.apache.pdfbox.pdmodel.PDResources.
//
// getFont and the cache it reads through are here; getColorSpace, getExtGState,
// getShading, getPattern, getProperties and getXObject are not, because each
// needs a type this port has not reached. They arrive with those types. See
// migration/STATUS.md.
type PDResources struct {
	resources *cos.Dictionary
	cache     ResourceCache

	// directFontCache holds the fonts of resources written out directly rather
	// than as an indirect object, which the shared cache cannot key on.
	directFontCache map[*cos.Name]font.PDFont
}

var _ common.COSObjectable = (*PDResources)(nil)

// NewPDResources returns an empty set of resources, for embedding.
func NewPDResources() *PDResources {
	return &PDResources{
		resources:       cos.NewDictionary(),
		directFontCache: map[*cos.Name]font.PDFont{},
	}
}

// NewPDResourcesOf returns the resources held by the given dictionary, for
// reading.
//
// A nil dictionary is a caller's mistake, which Java reports with the unchecked
// IllegalArgumentException.
func NewPDResourcesOf(resourceDictionary *cos.Dictionary) *PDResources {
	if resourceDictionary == nil {
		panic("pdmodel: resourceDictionary is null")
	}
	return &PDResources{
		resources:       resourceDictionary,
		directFontCache: map[*cos.Name]font.PDFont{},
	}
}

// NewPDResourcesOfCache returns the resources held by the given dictionary,
// read through the given cache.
func NewPDResourcesOfCache(resourceDictionary *cos.Dictionary, resourceCache ResourceCache) *PDResources {
	if resourceDictionary == nil {
		panic("pdmodel: resourceDictionary is null")
	}
	return &PDResources{
		resources:       resourceDictionary,
		cache:           resourceCache,
		directFontCache: map[*cos.Name]font.PDFont{},
	}
}

// COSObject returns the underlying dictionary.
func (r *PDResources) COSObject() cos.Base { return r.resources }

// Dictionary returns the underlying dictionary.
func (r *PDResources) Dictionary() *cos.Dictionary { return r.resources }

// HasColorSpace reports whether the given color space name exists in these
// resources.
func (r *PDResources) HasColorSpace(name *cos.Name) bool {
	return r.get(cos.ColorSpace, name) != nil
}

// IsImageXObject tells whether the XObject resource with the given name is an
// image.
func (r *PDResources) IsImageXObject(name *cos.Name) bool {
	// get the instance
	value := r.get(cos.XObject, name)
	if value == nil {
		return false
	}
	if object, ok := value.(*cos.Object); ok {
		value = object.Object()
	}
	stream, ok := value.(*cos.Stream)
	if !ok {
		return false
	}
	return cos.Image == stream.GetCOSName(cos.Subtype)
}

// getIndirect returns the resource with the given name and kind as an indirect
// object, or nil.
func (r *PDResources) getIndirect(kind, name *cos.Name) *cos.Object {
	dict := r.resources.GetCOSDictionary(kind)
	if dict == nil {
		return nil
	}
	if object, ok := dict.GetItem(name).(*cos.Object); ok {
		return object
	}
	// not an indirect object. Resource may have been added at runtime.
	return nil
}

// get returns the resource with the given name and kind, or nil.
func (r *PDResources) get(kind, name *cos.Name) cos.Base {
	dict := r.resources.GetCOSDictionary(kind)
	if dict == nil {
		return nil
	}
	return dict.GetDictionaryObject(name)
}

// ColorSpaceNames returns the names of the color space resources, if any.
func (r *PDResources) ColorSpaceNames() []*cos.Name { return r.names(cos.ColorSpace) }

// XObjectNames returns the names of the XObject resources, if any.
func (r *PDResources) XObjectNames() []*cos.Name { return r.names(cos.XObject) }

// FontNames returns the names of the font resources, if any.
func (r *PDResources) FontNames() []*cos.Name { return r.names(cos.Font) }

// PropertiesNames returns the names of the property list resources, if any.
func (r *PDResources) PropertiesNames() []*cos.Name { return r.names(cos.Properties) }

// ShadingNames returns the names of the shading resources, if any.
func (r *PDResources) ShadingNames() []*cos.Name { return r.names(cos.Shading) }

// PatternNames returns the names of the pattern resources, if any.
func (r *PDResources) PatternNames() []*cos.Name { return r.names(cos.Pattern) }

// ExtGStateNames returns the names of the extended graphics state resources, if
// any.
func (r *PDResources) ExtGStateNames() []*cos.Name { return r.names(cos.ExtGState) }

// names returns the resource names of the given kind.
func (r *PDResources) names(kind *cos.Name) []*cos.Name {
	dict := r.resources.GetCOSDictionary(kind)
	if dict == nil {
		return nil
	}
	return dict.KeySet()
}

// add adds the given resource if it does not already exist, and returns the
// name it went in under. An item that is already there keeps the name it has.
func (r *PDResources) add(kind *cos.Name, prefix string, object common.COSObjectable) *cos.Name {
	// return the existing key if the item exists already
	dict := r.resources.GetCOSDictionary(kind)
	if dict != nil && dict.ContainsValue(object.COSObject()) {
		return dict.KeyForValue(object.COSObject())
	}

	// PDFBOX-4509: It could exist as an indirect object, happens when a font is
	// taken from the AcroForm default resources of a loaded PDF.
	if dict != nil && cos.Font == kind {
		for key, value := range dict.All {
			indirect, ok := value.(*cos.Object)
			if ok && object.COSObject() == indirect.Object() {
				return key
			}
		}
	}

	// add the item with a new key
	name := r.createKey(kind, prefix)
	r.put(kind, name, object)
	return name
}

// createKey returns a unique key for a new resource.
func (r *PDResources) createKey(kind *cos.Name, prefix string) *cos.Name {
	dict := r.resources.GetCOSDictionary(kind)
	if dict == nil {
		return cos.GetPDFName(prefix + "1")
	}

	// find a unique key
	var key string
	n := len(dict.KeySet())
	for {
		n++
		key = prefix + strconv.Itoa(n)
		if !dict.ContainsKey(cos.GetPDFName(key)) {
			break
		}
	}
	return cos.GetPDFName(key)
}

// put sets the value of a given named resource.
func (r *PDResources) put(kind, name *cos.Name, object common.COSObjectable) {
	dict := r.resources.GetCOSDictionary(kind)
	if dict == nil {
		dict = cos.NewDictionary()
		r.resources.SetItem(kind, dict)
	}
	dict.SetItem(name, object.COSObject())
}

// GetFont returns the font resource with the given name, or nil where the
// resources have none.
//
// Port of org.apache.pdfbox.pdmodel.PDResources.getFont. Java holds the direct
// cache through SoftReferences so the collector may drop an entry; Go has none,
// and the port holds them outright.
func (r *PDResources) GetFont(name *cos.Name) (font.PDFont, error) {
	indirect := r.getIndirect(cos.Font, name)
	if r.cache != nil && indirect != nil {
		if cached := r.cache.GetFont(indirect); cached != nil {
			return cached, nil
		}
	} else if indirect == nil {
		if cached, ok := r.directFontCache[name]; ok && cached != nil {
			return cached, nil
		}
	}

	var f font.PDFont
	base := r.get(cos.Font, name)
	if dict, ok := base.(*cos.Dictionary); ok {
		var err error
		f, err = font.CreateFont(dict, r.cache)
		if err != nil {
			return nil, err
		}
	}

	if r.cache != nil && indirect != nil {
		r.cache.PutFont(indirect, f)
	} else if indirect == nil {
		if r.directFontCache == nil {
			r.directFontCache = map[*cos.Name]font.PDFont{}
		}
		r.directFontCache[name] = f
	}
	return f, nil
}

// Cache returns the cache the resources are read through, or nil where they are
// read without one.
func (r *PDResources) Cache() ResourceCache { return r.cache }
