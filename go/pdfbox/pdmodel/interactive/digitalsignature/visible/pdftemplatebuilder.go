package visible

import (
	goimage "image"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	graphicsform "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// PDFTemplateBuilder builds the parts of the appearance of a visible signature,
// one at a time, into a PDFTemplateStructure.
//
// Port of the interface PDFTemplateBuilder.
type PDFTemplateBuilder interface {
	// CreateAffineTransform sets the transform the image form is drawn
	// through.
	CreateAffineTransform(affineTransform *geom.AffineTransform)

	// CreatePage creates the page of the template.
	CreatePage(properties *PDVisibleSignDesigner)

	// CreateTemplate creates the document holding the given page.
	CreateTemplate(page *pdmodel.PDPage) error

	// CreateAcroForm creates the form of the template.
	CreateAcroForm(template *pdmodel.PDDocument)

	// CreateSignatureField creates the signature field of the form.
	CreateSignatureField(acroForm *form.PDAcroForm) error

	// CreateSignature creates the signature the field holds.
	CreateSignature(pdSignatureField *form.PDSignatureField, page *pdmodel.PDPage,
		signerName string) error

	// CreateAcroFormDictionary fills in the /DR entry of the form.
	CreateAcroFormDictionary(acroForm *form.PDAcroForm,
		signatureField *form.PDSignatureField) error

	// CreateSignatureRectangle sets the rectangle of the signature widget.
	CreateSignatureRectangle(signatureField *form.PDSignatureField,
		properties *PDVisibleSignDesigner) error

	// CreateProcSetArray creates the array of [Text, ImageB, ImageC, ImageI].
	CreateProcSetArray()

	// CreateSignatureImage embeds the image of the signature.
	CreateSignatureImage(template *pdmodel.PDDocument, img goimage.Image) error

	// CreateFormatterRectangle creates the bounding box of the forms.
	CreateFormatterRectangle(params []int)

	// CreateHolderFormStream creates the stream of the holder form.
	CreateHolderFormStream(template *pdmodel.PDDocument)

	// CreateHolderFormResources creates the resources of the holder form.
	CreateHolderFormResources()

	// CreateHolderForm creates the holder form.
	CreateHolderForm(holderFormResources *pdmodel.PDResources,
		holderFormStream *common.PDStream, bbox *common.PDRectangle)

	// CreateAppearanceDictionary creates the /AP entry of the widget.
	CreateAppearanceDictionary(holderForm *graphicsform.PDFormXObject,
		signatureField *form.PDSignatureField) error

	// CreateInnerFormStream creates the stream of the inner form.
	CreateInnerFormStream(template *pdmodel.PDDocument)

	// CreateInnerFormResource creates the resources of the inner form.
	CreateInnerFormResource()

	// CreateInnerForm creates the inner form, which sits inside the holder
	// form.
	CreateInnerForm(innerFormResources *pdmodel.PDResources,
		innerFormStream *common.PDStream, bbox *common.PDRectangle)

	// InsertInnerFormToHolderResources puts the inner form into the resources
	// of the holder form.
	InsertInnerFormToHolderResources(innerForm *graphicsform.PDFormXObject,
		holderFormResources *pdmodel.PDResources)

	// CreateImageFormStream creates the stream of the image form.
	CreateImageFormStream(template *pdmodel.PDDocument)

	// CreateImageFormResources creates the resources of the image form.
	CreateImageFormResources()

	// CreateImageForm creates the image form, which sits inside the inner form.
	CreateImageForm(imageFormResources, innerFormResource *pdmodel.PDResources,
		imageFormStream *common.PDStream, bbox *common.PDRectangle,
		affineTransform *geom.AffineTransform, img *image.PDImageXObject) error

	// CreateBackgroundLayerForm creates the blank n0 background layer form.
	CreateBackgroundLayerForm(innerFormResource *pdmodel.PDResources,
		bbox *common.PDRectangle) error

	// InjectProcSetArray puts the /ProcSet array into every resource
	// dictionary and into the page.
	InjectProcSetArray(innerForm *graphicsform.PDFormXObject, page *pdmodel.PDPage,
		innerFormResources, imageFormResources, holderFormResources *pdmodel.PDResources,
		procSet *cos.Array)

	// InjectAppearanceStreams writes the content of the three forms.
	InjectAppearanceStreams(holderFormStream, innerFormStream, imageFormStream *common.PDStream,
		imageFormName, imageName, innerFormName *cos.Name,
		properties *PDVisibleSignDesigner) error

	// CreateVisualSignature keeps the COS document of the template.
	CreateVisualSignature(template *pdmodel.PDDocument)

	// CreateWidgetDictionary fills in the /DR entry of the signature widget.
	CreateWidgetDictionary(signatureField *form.PDSignatureField,
		holderFormResources *pdmodel.PDResources) error

	// Structure returns what has been built so far.
	Structure() *PDFTemplateStructure

	// CloseTemplate closes the given document and the one of the structure.
	CloseTemplate(template *pdmodel.PDDocument) error
}
