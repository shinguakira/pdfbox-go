package pdmodel

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	_ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation/handlers"
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

// The appearance handlers write through PDAppearanceContentStream, which lives
// here; annotation names what they use and takes this constructor.
func init() {
	annotation.NewAppearanceContentStream = func(appearance *annotation.PDAppearanceStream,
		compress bool) (annotation.AppearanceContentStream, error) {
		return NewPDAppearanceContentStreamCompressed(appearance, compress)
	}
}

// The highlight handler draws into a form XObject, which PDFormContentStream
// writes; annotation names what it uses and takes this constructor.
func init() {
	annotation.NewFormContentStream = func(
		formXObject *form.PDFormXObject) (annotation.FormContentStream, error) {
		return NewPDFormContentStream(formXObject)
	}
}
