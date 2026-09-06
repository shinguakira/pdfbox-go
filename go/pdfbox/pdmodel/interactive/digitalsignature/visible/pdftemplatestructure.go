// Package visible builds the one page document that holds the appearance of a
// visible signature.
//
// Port of org.apache.pdfbox.pdmodel.interactive.digitalsignature.visible.
package visible

import (
	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	graphicsform "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/digitalsignature"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// PDFTemplateStructure holds every part of the template while the builder fills
// it in.
//
// Port of PDFTemplateStructure, which is a bag of fields with a getter and a
// setter each and no constructor.
type PDFTemplateStructure struct {
	page                 *pdmodel.PDPage
	template             *pdmodel.PDDocument
	acroForm             *form.PDAcroForm
	signatureField       *form.PDSignatureField
	pdSignature          *digitalsignature.PDSignature
	acroFormDictionary   *cos.Dictionary
	signatureRectangle   *common.PDRectangle
	affineTransform      *geom.AffineTransform
	procSet              *cos.Array
	image                *image.PDImageXObject
	formatterRectangle   *common.PDRectangle
	holderFormStream     *common.PDStream
	holderFormResources  *pdmodel.PDResources
	holderForm           *graphicsform.PDFormXObject
	appearanceDictionary *annotation.PDAppearanceDictionary
	innerFormStream      *common.PDStream
	innerFormResources   *pdmodel.PDResources
	innerForm            *graphicsform.PDFormXObject
	imageFormStream      *common.PDStream
	imageFormResources   *pdmodel.PDResources
	acroFormFields       []form.PDField
	innerFormName        *cos.Name
	imageFormName        *cos.Name
	imageName            *cos.Name
	visualSignature      *cos.Document
	imageForm            *graphicsform.PDFormXObject
	widgetDictionary     *cos.Dictionary
}

// Page returns the page of the template.
func (s *PDFTemplateStructure) Page() *pdmodel.PDPage { return s.page }

// SetPage sets the page of the template.
func (s *PDFTemplateStructure) SetPage(page *pdmodel.PDPage) { s.page = page }

// Template returns the document of the template.
func (s *PDFTemplateStructure) Template() *pdmodel.PDDocument { return s.template }

// SetTemplate sets the document of the template.
func (s *PDFTemplateStructure) SetTemplate(template *pdmodel.PDDocument) { s.template = template }

// AcroForm returns the form of the template.
func (s *PDFTemplateStructure) AcroForm() *form.PDAcroForm { return s.acroForm }

// SetAcroForm sets the form of the template.
func (s *PDFTemplateStructure) SetAcroForm(acroForm *form.PDAcroForm) { s.acroForm = acroForm }

// SignatureField returns the signature field of the template.
func (s *PDFTemplateStructure) SignatureField() *form.PDSignatureField { return s.signatureField }

// SetSignatureField sets the signature field of the template.
func (s *PDFTemplateStructure) SetSignatureField(signatureField *form.PDSignatureField) {
	s.signatureField = signatureField
}

// PDSignature returns the signature of the template.
func (s *PDFTemplateStructure) PDSignature() *digitalsignature.PDSignature { return s.pdSignature }

// SetPDSignature sets the signature of the template.
func (s *PDFTemplateStructure) SetPDSignature(pdSignature *digitalsignature.PDSignature) {
	s.pdSignature = pdSignature
}

// AcroFormDictionary returns the dictionary of the form.
func (s *PDFTemplateStructure) AcroFormDictionary() *cos.Dictionary { return s.acroFormDictionary }

// SetAcroFormDictionary sets the dictionary of the form.
func (s *PDFTemplateStructure) SetAcroFormDictionary(acroFormDictionary *cos.Dictionary) {
	s.acroFormDictionary = acroFormDictionary
}

// SignatureRectangle returns the rectangle of the signature widget.
func (s *PDFTemplateStructure) SignatureRectangle() *common.PDRectangle {
	return s.signatureRectangle
}

// SetSignatureRectangle sets the rectangle of the signature widget.
func (s *PDFTemplateStructure) SetSignatureRectangle(signatureRectangle *common.PDRectangle) {
	s.signatureRectangle = signatureRectangle
}

// AffineTransform returns the transform the image form is drawn through.
func (s *PDFTemplateStructure) AffineTransform() *geom.AffineTransform { return s.affineTransform }

// SetAffineTransform sets the transform the image form is drawn through.
func (s *PDFTemplateStructure) SetAffineTransform(affineTransform *geom.AffineTransform) {
	s.affineTransform = affineTransform
}

// ProcSet returns the /ProcSet array.
func (s *PDFTemplateStructure) ProcSet() *cos.Array { return s.procSet }

// SetProcSet sets the /ProcSet array.
func (s *PDFTemplateStructure) SetProcSet(procSet *cos.Array) { s.procSet = procSet }

// Image returns the image of the signature.
func (s *PDFTemplateStructure) Image() *image.PDImageXObject { return s.image }

// SetImage sets the image of the signature.
func (s *PDFTemplateStructure) SetImage(img *image.PDImageXObject) { s.image = img }

// FormatterRectangle returns the bounding box of the forms.
func (s *PDFTemplateStructure) FormatterRectangle() *common.PDRectangle {
	return s.formatterRectangle
}

// SetFormatterRectangle sets the bounding box of the forms.
func (s *PDFTemplateStructure) SetFormatterRectangle(formatterRectangle *common.PDRectangle) {
	s.formatterRectangle = formatterRectangle
}

// HolderFormStream returns the stream of the holder form.
func (s *PDFTemplateStructure) HolderFormStream() *common.PDStream { return s.holderFormStream }

// SetHolderFormStream sets the stream of the holder form.
func (s *PDFTemplateStructure) SetHolderFormStream(holderFormStream *common.PDStream) {
	s.holderFormStream = holderFormStream
}

// HolderForm returns the holder form.
func (s *PDFTemplateStructure) HolderForm() *graphicsform.PDFormXObject { return s.holderForm }

// SetHolderForm sets the holder form.
func (s *PDFTemplateStructure) SetHolderForm(holderForm *graphicsform.PDFormXObject) {
	s.holderForm = holderForm
}

// HolderFormResources returns the resources of the holder form.
func (s *PDFTemplateStructure) HolderFormResources() *pdmodel.PDResources {
	return s.holderFormResources
}

// SetHolderFormResources sets the resources of the holder form.
func (s *PDFTemplateStructure) SetHolderFormResources(holderFormResources *pdmodel.PDResources) {
	s.holderFormResources = holderFormResources
}

// AppearanceDictionary returns the /AP dictionary of the widget.
func (s *PDFTemplateStructure) AppearanceDictionary() *annotation.PDAppearanceDictionary {
	return s.appearanceDictionary
}

// SetAppearanceDictionary sets the /AP dictionary of the widget.
func (s *PDFTemplateStructure) SetAppearanceDictionary(
	appearanceDictionary *annotation.PDAppearanceDictionary) {
	s.appearanceDictionary = appearanceDictionary
}

// InnerFormStream returns the stream of the inner form.
func (s *PDFTemplateStructure) InnerFormStream() *common.PDStream { return s.innerFormStream }

// SetInnterFormStream sets the stream of the inner form.
//
// Java spells the name setInnterFormStream.
func (s *PDFTemplateStructure) SetInnterFormStream(innerFormStream *common.PDStream) {
	s.innerFormStream = innerFormStream
}

// InnerFormResources returns the resources of the inner form.
func (s *PDFTemplateStructure) InnerFormResources() *pdmodel.PDResources {
	return s.innerFormResources
}

// SetInnerFormResources sets the resources of the inner form.
func (s *PDFTemplateStructure) SetInnerFormResources(innerFormResources *pdmodel.PDResources) {
	s.innerFormResources = innerFormResources
}

// InnerForm returns the inner form.
func (s *PDFTemplateStructure) InnerForm() *graphicsform.PDFormXObject { return s.innerForm }

// SetInnerForm sets the inner form.
func (s *PDFTemplateStructure) SetInnerForm(innerForm *graphicsform.PDFormXObject) {
	s.innerForm = innerForm
}

// InnerFormName returns the name the inner form sits under.
func (s *PDFTemplateStructure) InnerFormName() *cos.Name { return s.innerFormName }

// SetInnerFormName sets the name the inner form sits under.
func (s *PDFTemplateStructure) SetInnerFormName(innerFormName *cos.Name) {
	s.innerFormName = innerFormName
}

// ImageFormStream returns the stream of the image form.
func (s *PDFTemplateStructure) ImageFormStream() *common.PDStream { return s.imageFormStream }

// SetImageFormStream sets the stream of the image form.
func (s *PDFTemplateStructure) SetImageFormStream(imageFormStream *common.PDStream) {
	s.imageFormStream = imageFormStream
}

// ImageFormResources returns the resources of the image form.
func (s *PDFTemplateStructure) ImageFormResources() *pdmodel.PDResources {
	return s.imageFormResources
}

// SetImageFormResources sets the resources of the image form.
func (s *PDFTemplateStructure) SetImageFormResources(imageFormResources *pdmodel.PDResources) {
	s.imageFormResources = imageFormResources
}

// ImageForm returns the image form.
func (s *PDFTemplateStructure) ImageForm() *graphicsform.PDFormXObject { return s.imageForm }

// SetImageForm sets the image form.
func (s *PDFTemplateStructure) SetImageForm(imageForm *graphicsform.PDFormXObject) {
	s.imageForm = imageForm
}

// ImageFormName returns the name the image form sits under.
func (s *PDFTemplateStructure) ImageFormName() *cos.Name { return s.imageFormName }

// SetImageFormName sets the name the image form sits under.
func (s *PDFTemplateStructure) SetImageFormName(imageFormName *cos.Name) {
	s.imageFormName = imageFormName
}

// ImageName returns the name the image sits under.
func (s *PDFTemplateStructure) ImageName() *cos.Name { return s.imageName }

// SetImageName sets the name the image sits under.
func (s *PDFTemplateStructure) SetImageName(imageName *cos.Name) { s.imageName = imageName }

// VisualSignature returns the COS document of the appearance.
func (s *PDFTemplateStructure) VisualSignature() *cos.Document { return s.visualSignature }

// SetVisualSignature sets the COS document of the appearance.
func (s *PDFTemplateStructure) SetVisualSignature(visualSignature *cos.Document) {
	s.visualSignature = visualSignature
}

// AcroFormFields returns the fields of the form.
func (s *PDFTemplateStructure) AcroFormFields() []form.PDField { return s.acroFormFields }

// SetAcroFormFields sets the fields of the form.
func (s *PDFTemplateStructure) SetAcroFormFields(acroFormFields []form.PDField) {
	s.acroFormFields = acroFormFields
}

// WidgetDictionary returns the dictionary of the signature widget.
func (s *PDFTemplateStructure) WidgetDictionary() *cos.Dictionary { return s.widgetDictionary }

// SetWidgetDictionary sets the dictionary of the signature widget.
func (s *PDFTemplateStructure) SetWidgetDictionary(widgetDictionary *cos.Dictionary) {
	s.widgetDictionary = widgetDictionary
}
