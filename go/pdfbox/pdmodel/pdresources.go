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
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/markedcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/optionalcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/state"
)

// PDResources is a set of resources available at the page/pages/stream level.
//
// Port of org.apache.pdfbox.pdmodel.PDResources.
//
// The colour space half is in pdresources_colorspace.go and the XObject and
// shading halves in pdresources_graphics.go, the way PDDocument splits off its
// encryption half. Everything else is here.
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

// NewPDResourcesOfCacheAndFontCache returns the resources held by the given
// dictionary, read through the given cache and sharing the given direct font
// cache.
//
// Port of the package-private PDResources(COSDictionary, ResourceCache,
// Map<COSName, SoftReference<PDFont>>), which the AcroForm uses so that its
// default resources keep one cache across the fields.
func NewPDResourcesOfCacheAndFontCache(resourceDictionary *cos.Dictionary,
	resourceCache ResourceCache, directFontCache map[*cos.Name]font.PDFont) *PDResources {
	if resourceDictionary == nil {
		panic("pdmodel: resourceDictionary is null")
	}
	return &PDResources{
		resources:       resourceDictionary,
		cache:           resourceCache,
		directFontCache: directFontCache,
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

// extGStateCache is the part of a resource cache that keeps extended graphics
// states.
//
// Java declares getExtGState and put(COSObject, PDExtendedGraphicsState) on
// ResourceCache itself. The port's ResourceCache is declared in pdmodel/font,
// because it names PDFont and this package imports that one, and it cannot name
// PDExtendedGraphicsState: graphics/state imports graphics, which imports font
// for PDFontSetting, so font naming state would close a cycle. A cache that
// keeps them says so by having these two methods, which DefaultResourceCache
// does; one that does not simply reads through.
type extGStateCache interface {
	GetExtGState(indirect *cos.Object) *state.PDExtendedGraphicsState
	PutExtGState(indirect *cos.Object, extGState *state.PDExtendedGraphicsState)
}

// GetExtGState returns the extended graphics state resource with the given
// name, or nil where the resources have none.
func (r *PDResources) GetExtGState(name *cos.Name) *state.PDExtendedGraphicsState {
	indirect := r.getIndirect(cos.ExtGState, name)
	cache, cacheKeepsThem := r.cache.(extGStateCache)
	if cacheKeepsThem && indirect != nil {
		if cached := cache.GetExtGState(indirect); cached != nil {
			return cached
		}
	}
	// get the instance
	var extGState *state.PDExtendedGraphicsState
	if base, isDictionary := asResourceDictionary(r.get(cos.ExtGState, name)); isDictionary {
		extGState = state.NewPDExtendedGraphicsStateOfCache(base, r.Cache())
	}
	if cacheKeepsThem && indirect != nil {
		cache.PutExtGState(indirect, extGState)
	}
	return extGState
}

// propertyListCache is the part of a resource cache that keeps property lists.
//
// Java declares getProperties, put(COSObject, PDPropertyList) and
// removeProperties on ResourceCache itself; the port's ResourceCache is
// declared in pdmodel/font, so this package asks the cache for them by shape,
// the way it asks for the extended graphics states above.
type propertyListCache interface {
	GetProperties(indirect *cos.Object) markedcontent.PropertyList
	PutProperties(indirect *cos.Object, propertyList markedcontent.PropertyList)
}

// GetProperties returns the property list resource with the given name, or nil
// where the resources have none.
//
// Port of PDResources.getProperties.
func (r *PDResources) GetProperties(name *cos.Name) markedcontent.PropertyList {
	indirect := r.getIndirect(cos.Properties, name)
	cache, cacheKeepsThem := r.cache.(propertyListCache)
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

// NewPatternOfDictionary builds a pattern resource. graphics/pattern sets it
// from its init.
//
// Java's PDResources.getPattern returns a PDAbstractPattern, which lives in
// graphics/pattern; that package imports this one, so the constructor is
// reached through here rather than named. It answers an any, because this
// package cannot name the return type either.
var NewPatternOfDictionary func(dictionary *cos.Dictionary, stream *cos.Stream,
	cache ResourceCache) (any, error)

// patternCache is the part of a resource cache that keeps patterns.
//
// Java declares getPattern, put(COSObject, PDAbstractPattern) and
// removePattern on ResourceCache itself; the port's ResourceCache is declared
// in pdmodel/font, so this package asks the cache for them by shape, the way it
// asks for the extended graphics states.
type patternCache interface {
	GetPattern(indirect *cos.Object) any
	PutPattern(indirect *cos.Object, pattern any)
}

// GetPattern returns the pattern resource with the given name, or nil where the
// resources have none.
//
// Port of PDResources.getPattern. It answers an any for the reason
// NewPatternOfDictionary gives; a caller narrows it to what it expects.
func (r *PDResources) GetPattern(name *cos.Name) (any, error) {
	if NewPatternOfDictionary == nil {
		// graphics/pattern is not linked in, so there is nothing to build a
		// pattern with. See migration/STATUS.md.
		return nil, nil
	}
	indirect := r.getIndirect(cos.Pattern, name)
	cache, cacheKeepsThem := r.cache.(patternCache)
	if cacheKeepsThem && indirect != nil {
		if cached := cache.GetPattern(indirect); cached != nil {
			return cached, nil
		}
	}
	base := r.get(cos.Pattern, name)
	dict, isDictionary := asResourceDictionary(base)
	if !isDictionary {
		return nil, nil
	}
	stream, _ := base.(*cos.Stream)
	pattern, err := NewPatternOfDictionary(dict, stream, r.Cache())
	if err != nil {
		return nil, err
	}
	if cacheKeepsThem && indirect != nil {
		cache.PutPattern(indirect, pattern)
	}
	return pattern, nil
}

// asResourceDictionary is Java's instanceof COSDictionary, which a COSStream
// also satisfies.
func asResourceDictionary(base cos.Base) (*cos.Dictionary, bool) {
	switch value := base.(type) {
	case *cos.Stream:
		return &value.Dictionary, true
	case *cos.Dictionary:
		return value, true
	}
	return nil, false
}

// AddFont adds the given font to the resources, and returns the name it went in
// under.
func (r *PDResources) AddFont(value font.PDFont) *cos.Name {
	return r.add(cos.Font, "F", value)
}

// AddColorSpace adds the given colour space to the resources.
func (r *PDResources) AddColorSpace(colorSpace color.PDColorSpace) *cos.Name {
	return r.add(cos.ColorSpace, "cs", colorSpace)
}

// AddExtGState adds the given extended graphics state to the resources.
func (r *PDResources) AddExtGState(extGState *state.PDExtendedGraphicsState) *cos.Name {
	return r.add(cos.ExtGState, "gs", extGState)
}

// AddPropertyList adds the given property list to the resources, under the
// prefix that says whether it is an optional content group.
func (r *PDResources) AddPropertyList(properties markedcontent.PropertyList) *cos.Name {
	if _, isGroup := properties.(*optionalcontent.PDOptionalContentGroup); isGroup {
		return r.add(cos.Properties, "oc", properties)
	}
	return r.add(cos.Properties, "Prop", properties)
}

// AddImageXObject adds the given image to the resources.
func (r *PDResources) AddImageXObject(value *image.PDImageXObject) *cos.Name {
	return r.add(cos.XObject, "Im", value)
}

// AddFormXObject adds the given form to the resources.
func (r *PDResources) AddFormXObject(value *form.PDFormXObject) *cos.Name {
	return r.add(cos.XObject, "Form", value)
}

// AddXObject adds the given XObject to the resources under the given prefix.
func (r *PDResources) AddXObject(xobject common.COSObjectable, prefix string) *cos.Name {
	return r.add(cos.XObject, prefix, xobject)
}

// PutFont sets the font resource with the given name.
func (r *PDResources) PutFont(name *cos.Name, value font.PDFont) {
	r.put(cos.Font, name, value)
}

// PutColorSpace sets the colour space resource with the given name.
func (r *PDResources) PutColorSpace(name *cos.Name, colorSpace color.PDColorSpace) {
	r.put(cos.ColorSpace, name, colorSpace)
}

// PutExtGState sets the extended graphics state resource with the given name.
func (r *PDResources) PutExtGState(name *cos.Name, extGState *state.PDExtendedGraphicsState) {
	r.put(cos.ExtGState, name, extGState)
}

// PutPropertyList sets the property list resource with the given name.
func (r *PDResources) PutPropertyList(name *cos.Name, properties markedcontent.PropertyList) {
	r.put(cos.Properties, name, properties)
}

// PutXObject sets the XObject resource with the given name.
func (r *PDResources) PutXObject(name *cos.Name, xobject common.COSObjectable) {
	r.put(cos.XObject, name, xobject)
}
