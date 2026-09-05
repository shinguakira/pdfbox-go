package form

import (
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/digitalsignature"
)

// SignatureFieldsOfDocument returns the signature fields of the given document.
//
// Port of PDDocument.getSignatureFields. It is a function here rather than a
// method there because it names PDSignatureField, and this package imports
// pdmodel; see AcroFormOfCatalog.
func SignatureFieldsOfDocument(document *pdmodel.PDDocument) []*PDSignatureField {
	fields := []*PDSignatureField{}
	acroForm := AcroFormOfCatalogFixup(document.DocumentCatalog(), nil)
	if acroForm != nil {
		for field := range acroForm.FieldTree().All() {
			if signatureField, isSignatureField := field.(*PDSignatureField); isSignatureField {
				fields = append(fields, signatureField)
			}
		}
	}
	return fields
}

// SignatureDictionariesOfDocument returns the signature dictionaries of the
// given document.
//
// Port of PDDocument.getSignatureDictionaries; see SignatureFieldsOfDocument
// for why it is a function.
func SignatureDictionariesOfDocument(
	document *pdmodel.PDDocument) []*digitalsignature.PDSignature {
	signatures := []*digitalsignature.PDSignature{}
	for _, field := range SignatureFieldsOfDocument(document) {
		value := field.FieldDictionary().GetCOSDictionary(cos.V)
		if value != nil {
			signatures = append(signatures, digitalsignature.NewPDSignatureOf(value))
		}
	}
	return signatures
}
