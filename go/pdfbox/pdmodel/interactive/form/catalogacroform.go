package form

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// DocumentFixup is a repair applied to a document before it is read.
//
// Port of the interface pdmodel.fixup.PDDocumentFixup, which is named here as
// well because AcroFormOfCatalogFixup takes one: pdmodel/fixup names PDAcroForm,
// so this package cannot import it back. Go matches interfaces structurally, so
// a fixup.PDDocumentFixup is one of these.
type DocumentFixup interface {
	// Apply performs the repair.
	Apply()
}

// NewAcroFormDefaultFixup builds the fixup AcroFormOfCatalog applies. It is the
// AcroFormDefaultFixup of pdmodel/fixup, which sets this from its init, because
// that package names PDAcroForm and so cannot be imported from here.
//
// It is nil until pdmodel/fixup is linked in; AcroFormOfCatalog then reads the
// form with no fixup applied, which is getAcroForm(null) of Java.
var NewAcroFormDefaultFixup func(document *pdmodel.PDDocument) DocumentFixup

// AcroFormOfCatalog returns the form of the given catalogue with the default
// fixup applied, or nil where the document has no form.
//
// Port of PDDocumentCatalog.getAcroForm. It is a function rather than a method
// because it names PDAcroForm, and this package imports pdmodel.
func AcroFormOfCatalog(catalog *pdmodel.PDDocumentCatalog) *PDAcroForm {
	if NewAcroFormDefaultFixup == nil {
		return AcroFormOfCatalogFixup(catalog, nil)
	}
	return AcroFormOfCatalogFixup(catalog, NewAcroFormDefaultFixup(catalog.Document()))
}

// AcroFormOfCatalogFixup returns the form of the given catalogue, or nil where
// the document has no form.
//
// Dependent on the acroFormFixup given, some fixing and changing will be done to
// the form. To be sure that no fixes are applied, pass nil.
//
// Passing a fixup might change the original content, and later calls with nil
// will return the changed content.
//
// Port of getAcroForm(PDDocumentFixup).
func AcroFormOfCatalogFixup(catalog *pdmodel.PDDocumentCatalog,
	acroFormFixup DocumentFixup) *PDAcroForm {
	if acroFormFixup != nil && acroFormFixup != catalog.AcroFormFixupApplied() {
		acroFormFixup.Apply()
		catalog.SetCachedAcroForm(nil)
		catalog.SetAcroFormFixupApplied(acroFormFixup)
	} else if catalog.AcroFormFixupApplied() != nil {
		slog.Debug("form: AcroForm content has already been retrieved with fixes applied " +
			"- original content changed because of that")
	}
	cached, _ := catalog.CachedAcroForm().(*PDAcroForm)
	if cached == nil {
		dict := catalog.Dictionary().GetCOSDictionary(cos.AcroForm)
		if dict != nil {
			cached = NewPDAcroFormOf(catalog.Document(), dict)
			catalog.SetCachedAcroForm(cached)
		} else {
			catalog.SetCachedAcroForm(nil)
		}
	}
	return cached
}

// SetAcroFormOfCatalog sets the form of the given catalogue.
//
// Port of PDDocumentCatalog.setAcroForm; see AcroFormOfCatalog for why it is a
// function.
func SetAcroFormOfCatalog(catalog *pdmodel.PDDocumentCatalog, acroForm *PDAcroForm) {
	catalog.Dictionary().SetItem(cos.AcroForm, acroForm.COSObject())
	catalog.SetCachedAcroForm(nil)
}
