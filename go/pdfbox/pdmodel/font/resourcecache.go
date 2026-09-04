package font

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// ResourceCache keeps the objects read out of a resource dictionary, so that
// two pages sharing one indirect object share the object read from it.
//
// Port of org.apache.pdfbox.pdmodel.ResourceCache. Java puts it in pdmodel,
// which is the package that also holds PDResources; Go cannot, because pdmodel
// imports this package and the interface names types from it. It is declared
// here and aliased back into pdmodel under its Java name.
//
// The colour space, graphics state, shading, pattern, property list and XObject
// members of the Java interface are not here yet: the types they return belong
// to slices this one does not reach. See migration/STATUS.md.
type ResourceCache interface {
	// GetFont returns the font read from the given indirect object, or nil
	// where the cache has none.
	GetFont(indirect *cos.Object) PDFont

	// GetFontDescriptor returns the font descriptor read from the given
	// indirect object, or nil where the cache has none.
	GetFontDescriptor(indirect *cos.Object) *PDFontDescriptor

	// PutFont records the font read from the given indirect object.
	PutFont(indirect *cos.Object, font PDFont)

	// PutFontDescriptor records the font descriptor read from the given
	// indirect object.
	PutFontDescriptor(indirect *cos.Object, fontDescriptor *PDFontDescriptor)

	// RemoveFont drops the font read from the given indirect object and
	// returns it.
	RemoveFont(indirect *cos.Object) PDFont

	// RemoveFontDescriptor drops the font descriptor read from the given
	// indirect object and returns it.
	RemoveFontDescriptor(indirect *cos.Object) *PDFontDescriptor

	// GetCIDFont returns the CIDFont read from the given indirect object, or
	// nil where the cache has none.
	GetCIDFont(indirect *cos.Object) PDCIDFont

	// PutCIDFont records the CIDFont read from the given indirect object.
	PutCIDFont(indirect *cos.Object, cidFont PDCIDFont)

	// RemoveCIDFont drops the CIDFont read from the given indirect object and
	// returns it.
	RemoveCIDFont(indirect *cos.Object) PDCIDFont
}
