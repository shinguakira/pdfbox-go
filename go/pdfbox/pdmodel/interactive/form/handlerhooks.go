package form

import (
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation/handlers"
)

// init gives the free text appearance handler the two things it reads off the
// form of a document. That package cannot name PDAcroForm -- pdmodel
// blank-imports it for the handler registry, and this package imports pdmodel --
// so it declares the two hooks and this fills them.
func init() {
	handlers.AcroFormDefaultAppearance = func(document common.COSDocumentLike) string {
		acroForm := acroFormOfDocumentLike(document)
		if acroForm == nil {
			return ""
		}
		return acroForm.DefaultAppearance()
	}
	handlers.AcroFormDefaultResourcesFont = func(document common.COSDocumentLike,
		fontName *cos.Name) font.PDFont {
		acroForm := acroFormOfDocumentLike(document)
		if acroForm == nil {
			return nil
		}
		defaultResources := acroForm.DefaultResources()
		if defaultResources == nil {
			return nil
		}
		defaultResourcesFont, err := defaultResources.GetFont(fontName)
		if err != nil {
			slog.Warn("form: font of the default resources not read",
				slog.String("font", fontName.Name()), slog.Any("err", err))
			return nil
		}
		return defaultResourcesFont
	}
}

// acroFormOfDocumentLike returns the form of the document the appearance
// handlers were given, and nil where it has none.
//
// The handlers hold the document as a common.COSDocumentLike, which is the COS
// document; Java holds a PDDocument and asks it for its catalogue. The port
// builds the PDDocument around the COS document, which is what Loader does with
// what the parser answers, and asks that.
func acroFormOfDocumentLike(document common.COSDocumentLike) *PDAcroForm {
	cosDocument, isCOSDocument := document.(*cos.Document)
	if !isCOSDocument {
		return nil
	}
	return AcroFormOfCatalog(pdmodel.NewPDDocumentOf(cosDocument, nil).DocumentCatalog())
}
