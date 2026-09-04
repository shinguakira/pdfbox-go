package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/destination"
)

// The interactive packages name PDPage, PDPageTree and PDResources, which live
// here, and this package imports them back for the page's annotations. Go
// forbids that, so each of them declares what it needs and takes the
// constructors; these are them.
func init() {
	annotation.NewPageFromDictionary = func(dic *cos.Dictionary) annotation.PageLike {
		return NewPDPageOf(dic)
	}
	destination.NewPageFromDictionary = func(dic *cos.Dictionary) destination.PageLike {
		return NewPDPageOf(dic)
	}
	destination.IndexOfPageInTree = func(root, pageDict *cos.Dictionary) int {
		return NewPDPageTreeOf(root).IndexOf(NewPDPageOf(pageDict))
	}
	form.NewResourcesFromDictionary = func(dict *cos.Dictionary, cache form.CacheLike) form.ResourcesLike {
		resourceCache, _ := cache.(ResourceCache)
		return NewPDResourcesOfCache(dict, resourceCache)
	}
	form.NewEmptyResources = func() form.ResourcesLike { return NewPDResources() }
}
