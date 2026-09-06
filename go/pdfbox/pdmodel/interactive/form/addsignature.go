package form

// PDDocument.addSignature and saveIncrementalForExternalSigning.
//
// Both are functions over a *pdmodel.PDDocument rather than methods on it:
// addSignature names PDAcroForm, PDSignatureField and PDAnnotationWidget, and
// saveIncrementalForExternalSigning reads the signature dictionaries through
// them. pdmodel cannot import this package, so they live here, next to
// SignatureFieldsOfDocument. See AcroFormOfCatalog.

import (
	"errors"
	"io"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/digitalsignature"
)

// reserveByteRange is the /ByteRange addSignature writes, which COSWriter
// overwrites with the real one.
//
// Port of the private static PDDocument.RESERVE_BYTE_RANGE.
var reserveByteRange = []int{0, 1000000000, 1000000000, 1000000000}

// AddSignature adds a signature to be created using the given interface, with
// the default options.
//
// Only one signature may be added in a document. To sign several times, load
// the document, add the signature, save incrementally and close again.
//
// Port of PDDocument.addSignature(PDSignature, SignatureInterface). Java also
// declares addSignature(PDSignature) and
// addSignature(PDSignature, SignatureOptions), which are this with a nil
// interface; Go has no overloading, so a caller that wants those passes nil.
func AddSignature(document *pdmodel.PDDocument, sigObject *digitalsignature.PDSignature,
	signatureInterface pdfwriter.SignatureInterface) error {
	return AddSignatureOfOptions(document, sigObject, signatureInterface,
		digitalsignature.NewSignatureOptions())
}

// AddSignatureOfOptions adds a signature to the document. If the 0-based page
// number in the options parameter is smaller than 0 or larger than max, the
// nearest valid page number will be used (i.e. 0 or max) and no error is
// answered.
//
// Only one signature may be added in a document.
//
// Port of addSignature(PDSignature, SignatureInterface, SignatureOptions).
// Java throws IllegalStateException where one is already there, where the
// document has no page, and IllegalArgumentException from
// prepareVisibleSignature; all three are unchecked, so the port panics.
func AddSignatureOfOptions(document *pdmodel.PDDocument,
	sigObject *digitalsignature.PDSignature, signatureInterface pdfwriter.SignatureInterface,
	options *digitalsignature.SignatureOptions) error {
	if document.SignatureAdded() {
		panic("Only one signature may be added in a document")
	}
	document.SetSignatureAdded(true)

	// Reserve content
	// We need to reserve some space for the signature. Some signatures including
	// big certificate chain and we need enough space to store it.
	preferredSignatureSize := options.PreferredSignatureSize()
	if preferredSignatureSize > 0 {
		sigObject.SetContents(make([]byte, preferredSignatureSize))
	} else {
		sigObject.SetContents(make([]byte, digitalsignature.DefaultSignatureSize))
	}

	// Reserve ByteRange, will be overwritten in COSWriter
	sigObject.SetByteRange(reserveByteRange)

	document.SetSignInterface(signatureInterface)

	// Create SignatureForm for signature and append it to the document

	// Get the first valid page
	pageTree := document.Pages()
	pageCount := pageTree.Count()
	if pageCount == 0 {
		panic("Cannot sign an empty document")
	}

	// Get the AcroForm from the Root-Dictionary and append the annotation
	catalog := document.DocumentCatalog()
	acroForm := AcroFormOfCatalogFixup(catalog, nil)
	catalog.Dictionary().SetNeedToBeUpdated(true)

	if acroForm == nil {
		acroForm = NewPDAcroForm(document)
		SetAcroFormOfCatalog(catalog, acroForm)
	} else {
		acroForm.Dictionary().SetNeedToBeUpdated(true)
	}

	var signatureField *PDSignatureField
	fieldArray := acroForm.Dictionary().GetCOSArray(cos.Fields)
	if fieldArray != nil {
		fieldArray.SetNeedToBeUpdated(true)
		signatureField = findSignatureField(acroForm, sigObject)
	} else {
		acroForm.Dictionary().SetItem(cos.Fields, cos.NewArray())
	}
	var firstWidget *annotation.PDAnnotationWidget
	var page *pdmodel.PDPage
	if signatureField == nil {
		signatureField = NewPDSignatureField(acroForm)
		// append the signature object
		if err := signatureField.SetSignatureValue(sigObject); err != nil {
			return err
		}
		firstWidget = signatureField.Widgets()[0]
		startIndex := min(max(options.Page(), 0), pageCount-1)
		page = pageTree.Get(startIndex)
		// backward linking
		firstWidget.SetPage(page)
	} else {
		firstWidget = signatureField.Widgets()[0]
		sigObject.Dictionary().SetNeedToBeUpdated(true)
		page = nil
	}

	// TODO This "overwrites" the settings of the original signature field which might not be intended by the user
	// better make it configurable (not all users need/want PDF/A but their own setting):

	// to conform PDF/A-1 requirement:
	// The /F key's Print flag bit shall be set to 1 and
	// its Hidden, Invisible and NoView flag bits shall be set to 0
	firstWidget.SetPrinted(true)
	// This may be troublesome if several form fields are signed,
	// see thread from PDFBox users mailing list 17.2.2021 - 19.2.2021
	// https://mail-archives.apache.org/mod_mbox/pdfbox-users/202102.mbox/thread
	// better set the printed flag in advance

	// Set the AcroForm Fields
	acroFormFields := acroForm.Fields()
	acroForm.Dictionary().SetDirect(true)
	acroForm.SetSignaturesExist(true)
	acroForm.SetAppendOnly(true)

	checkFields := checkSignatureField(acroForm, signatureField)
	if checkFields {
		signatureField.FieldDictionary().SetNeedToBeUpdated(true)
	} else {
		acroFormFields = append(acroFormFields, signatureField)
		acroForm.SetFields(acroFormFields)
	}

	// Get the object from the visual signature
	visualSignature := options.VisualSignature()

	// Distinction of case for visual and non-visual signature
	if visualSignature == nil {
		prepareNonVisibleSignature(document, firstWidget)
	} else {
		prepareVisibleSignature(firstWidget, acroForm, visualSignature)
	}

	if page != nil {
		// Create Annotation / Field for signature
		annotations := page.Annotations()

		// Get the annotations of the page and append the signature-annotation to it
		// take care that page and acroforms do not share the same array
		// (if so, we don't need to add it twice)
		//
		// Java compares the two COSArrayLists by their backing lists; the port's
		// Fields answers a plain slice, so it compares the arrays the two sides
		// are stored in, which is the same question asked of the COS.
		sharesArray := checkFields &&
			annotations.COSObject() == cos.Base(acroForm.Dictionary().GetCOSArray(cos.Fields))
		if !sharesArray {
			// use check to prevent the annotation widget from appearing twice
			if checkSignatureAnnotation(annotations.ToSlice(), firstWidget) {
				firstWidget.AnnotationDictionary().SetNeedToBeUpdated(true)
			} else {
				annotations.Add(firstWidget)
			}
		}

		// Make /Annots a direct object by reassigning it,
		// to avoid problem if it is an existing indirect object:
		// it would not be updated in incremental save, and if we'd set the /Annots array "to be updated"
		// while keeping it indirect, Adobe Reader would claim that the document had been modified.
		page.SetAnnotations(annotations.ToSlice())

		page.Dictionary().SetNeedToBeUpdated(true)
	}
	return nil
}

// findSignatureField searches the form's fields for the signature field with
// the given signature dictionary, and answers nil where there is none.
//
// Port of the private PDDocument.findSignatureField, which takes the iterator
// the caller has already built.
func findSignatureField(acroForm *PDAcroForm,
	sigObject *digitalsignature.PDSignature) *PDSignatureField {
	for pdField := range acroForm.FieldIterator() {
		signatureField, isSignatureField := pdField.(*PDSignatureField)
		if !isSignatureField {
			continue
		}
		signature := signatureField.Signature()
		if signature != nil && signature.Dictionary() == sigObject.Dictionary() {
			return signatureField
		}
	}
	return nil
}

// checkSignatureField reports whether the field already exists in the field
// list.
//
// Port of the private PDDocument.checkSignatureField.
func checkSignatureField(acroForm *PDAcroForm, signatureField *PDSignatureField) bool {
	for field := range acroForm.FieldIterator() {
		if _, isSignatureField := field.(*PDSignatureField); !isSignatureField {
			continue
		}
		if field.FieldDictionary() == signatureField.FieldDictionary() {
			return true
		}
	}
	return false
}

// checkSignatureAnnotation reports whether the widget already exists in the
// annotation list.
//
// Port of the private PDDocument.checkSignatureAnnotation.
func checkSignatureAnnotation(annotations []annotation.PDAnnotation,
	widget *annotation.PDAnnotationWidget) bool {
	for _, annot := range annotations {
		if annot.AnnotationDictionary() == widget.AnnotationDictionary() {
			return true
		}
	}
	return false
}

// prepareNonVisibleSignature is the private PDDocument.prepareNonVisibleSignature.
func prepareNonVisibleSignature(document *pdmodel.PDDocument,
	firstWidget *annotation.PDAnnotationWidget) {
	// "Signature fields that are not intended to be visible shall
	// have an annotation rectangle that has zero height and width."
	// Set rectangle for non-visual signature to rectangle array [ 0 0 0 0 ]
	firstWidget.SetRectangle(common.NewPDRectangle())

	// The visual appearance must also exist for an invisible signature but may be empty.
	appearanceDictionary := annotation.NewPDAppearanceDictionary()
	appearanceStream := annotation.NewPDAppearanceStream(document.Document())
	appearanceStream.SetBBox(common.NewPDRectangle())
	appearanceDictionary.SetNormalAppearanceStream(appearanceStream)
	firstWidget.SetAppearance(appearanceDictionary)
}

// prepareVisibleSignature is the private PDDocument.prepareVisibleSignature.
//
// Java throws IllegalArgumentException where the template is missing an object,
// which is unchecked, so the port panics.
func prepareVisibleSignature(firstWidget *annotation.PDAnnotationWidget, acroForm *PDAcroForm,
	visualSignature *cos.Document) {
	// Obtain visual signature object
	annotFound := false
	sigFieldFound := false
	// get all objects
	// Java maps the xref table keys through getObjectFromPool; the port walks
	// the same table.
	for key := range visualSignature.XRefTable() {
		cosObject := visualSignature.ObjectFromPool(key)
		if cosObject == nil {
			continue
		}
		cosBaseDict, isDictionary := cosObject.Object().(*cos.Dictionary)
		if !isDictionary {
			continue
		}
		// Search for signature annotation
		if !annotFound && cosBaseDict.GetCOSName(cos.Type) == cos.Annot {
			assignSignatureRectangle(firstWidget, cosBaseDict)
			annotFound = true
		}
		// Search for signature field
		apDict := cosBaseDict.GetCOSDictionary(cos.AP)
		if apDict != nil && !sigFieldFound && cosBaseDict.GetCOSName(cos.FT) == cos.Sig {
			assignAppearanceDictionary(firstWidget, apDict)
			assignAcroFormDefaultResource(acroForm, cosBaseDict)
			sigFieldFound = true
		}
		if annotFound && sigFieldFound {
			break
		}
	}
	if !annotFound || !sigFieldFound {
		panic("Template is missing required objects")
	}
}

// assignSignatureRectangle is the private PDDocument.assignSignatureRectangle.
func assignSignatureRectangle(firstWidget *annotation.PDAnnotationWidget,
	annotDict *cos.Dictionary) {
	// Read and set the rectangle for visual signature
	existingRectangle := firstWidget.Rectangle()

	// in case of an existing field keep the original rect
	if existingRectangle == nil || existingRectangle.COSArray().Size() != 4 {
		rectArray := annotDict.GetCOSArray(cos.Rect)
		firstWidget.SetRectangle(common.NewPDRectangleOfCOSArray(rectArray))
	}
}

// assignAppearanceDictionary is the private PDDocument.assignAppearanceDictionary.
func assignAppearanceDictionary(firstWidget *annotation.PDAnnotationWidget,
	apDict *cos.Dictionary) {
	// read and set Appearance Dictionary
	ap := annotation.NewPDAppearanceDictionaryOf(apDict)
	apDict.SetDirect(true)
	firstWidget.SetAppearance(ap)
}

// assignAcroFormDefaultResource is the private
// PDDocument.assignAcroFormDefaultResource.
func assignAcroFormDefaultResource(acroForm *PDAcroForm, newDict *cos.Dictionary) {
	// read and set/update AcroForm default resource dictionary /DR if available
	newDR := newDict.GetCOSDictionary(cos.DR)
	if newDR == nil {
		return
	}
	defaultResources := acroForm.DefaultResources()
	if defaultResources == nil {
		acroForm.Dictionary().SetItem(cos.DR, newDR)
		newDR.SetDirect(true)
		newDR.SetNeedToBeUpdated(true)
		return
	}
	oldDR := defaultResources.Dictionary()
	newXObject := newDR.GetCOSDictionary(cos.XObject)
	oldXObject := oldDR.GetCOSDictionary(cos.XObject)
	if newXObject != nil && oldXObject != nil {
		oldXObject.AddAll(newXObject)
		oldDR.SetNeedToBeUpdated(true)
	}
}

// ErrNotLoadedFromAFile is what saving a document for external signing reports
// where it was built in memory. Java throws IllegalStateException.
var ErrNotLoadedFromAFile = errors.New("form: document was not loaded from a file or a stream")

// ErrNoSignatureField is what saving a document for external signing reports
// where it holds no signature dictionary.
var ErrNoSignatureField = errors.New("form: document does not contain signature fields")

// ErrByteRangeChanged is what saving a document for external signing reports
// where the reserved byte range was overwritten between AddSignature and here.
var ErrByteRangeChanged = errors.New(
	"form: signature reserve byte range has been changed after AddSignature, " +
		"please set the byte range that existed after AddSignature")

// SaveIncrementalForExternalSigning saves the document for signing by something
// outside this library, and returns the support the caller sets the signature
// through.
//
// Port of PDDocument.saveIncrementalForExternalSigning; it is a function here
// because it reads the signature dictionaries through the form. Java throws
// IllegalStateException for each of the three refusals, which is unchecked; the
// port answers an error, because every one of them is a fact about the file
// rather than about the caller.
func SaveIncrementalForExternalSigning(document *pdmodel.PDDocument,
	output io.Writer) (*digitalsignature.SigningSupport, error) {
	// Java calls subsetDesignatedFonts() first. That method is unexported here
	// and its body is a no-op -- font subsetting is font embedding, which the
	// port does not have -- so the set it walks is always empty. See
	// migration/STATUS.md.
	if document.PDFSource() == nil {
		return nil, ErrNotLoadedFromAFile
	}
	// PDFBOX-3978: getLastSignatureDictionary() not helpful if signing into a template
	// that is not the last signature. So give higher priority to signature with update flag.
	var foundSignature *digitalsignature.PDSignature
	for _, sig := range SignatureDictionariesOfDocument(document) {
		foundSignature = sig
		if sig.Dictionary().IsNeedToBeUpdated() {
			break
		}
	}

	if foundSignature == nil {
		return nil, ErrNoSignatureField
	}

	byteRange := foundSignature.ByteRange()
	if len(byteRange) != len(reserveByteRange) {
		return nil, ErrByteRangeChanged
	}
	for i, want := range reserveByteRange {
		if byteRange[i] != want {
			return nil, ErrByteRangeChanged
		}
	}
	writer, err := pdfwriter.NewCOSWriterIncremental(output, document.PDFSource())
	if err != nil {
		return nil, err
	}
	if err := writer.Write(document); err != nil {
		return nil, err
	}
	signingSupport := digitalsignature.NewSigningSupport(writer)
	document.SetSigningSupport(signingSupport)
	return signingSupport, nil
}
