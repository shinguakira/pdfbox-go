package visible

import (
	"bytes"
	"io"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// PDFTemplateCreator drives a PDFTemplateBuilder through the whole appearance
// of a visible signature, and hands back the one page document it built.
//
// Port of PDFTemplateCreator.
type PDFTemplateCreator struct {
	pdfBuilder PDFTemplateBuilder
}

// NewPDFTemplateCreator returns a creator driving the given builder.
func NewPDFTemplateCreator(templateBuilder PDFTemplateBuilder) *PDFTemplateCreator {
	return &PDFTemplateCreator{pdfBuilder: templateBuilder}
}

// PdfStructure returns what the builder has built so far.
func (c *PDFTemplateCreator) PdfStructure() *PDFTemplateStructure {
	return c.pdfBuilder.Structure()
}

// BuildPDF builds the one page document holding the appearance the given
// designer describes.
func (c *PDFTemplateCreator) BuildPDF(properties *PDVisibleSignDesigner) (io.Reader, error) {
	slog.Info("visible: pdf building has been started")
	pdfStructure := c.pdfBuilder.Structure()

	// we create array of [Text, ImageB, ImageC, ImageI]
	c.pdfBuilder.CreateProcSetArray()

	//create page
	c.pdfBuilder.CreatePage(properties)
	page := pdfStructure.Page()

	//create template
	if err := c.pdfBuilder.CreateTemplate(page); err != nil {
		return nil, err
	}

	template := pdfStructure.Template()
	defer template.Close()

	//create /AcroForm
	c.pdfBuilder.CreateAcroForm(template)
	acroForm := pdfStructure.AcroForm()

	// AcroForm contains signature fields
	if err := c.pdfBuilder.CreateSignatureField(acroForm); err != nil {
		return nil, err
	}
	pdSignatureField := pdfStructure.SignatureField()

	// create signature
	//TODO
	// The line below has no effect with the CreateVisibleSignature example.
	// The signature field is needed as a "holder" for the /AP tree,
	// but the /P and /V PDSignatureField entries are ignored by PDDocument.addSignature
	if err := c.pdfBuilder.CreateSignature(pdSignatureField, page, ""); err != nil {
		return nil, err
	}

	// that is /AcroForm/DR entry
	if err := c.pdfBuilder.CreateAcroFormDictionary(acroForm, pdSignatureField); err != nil {
		return nil, err
	}

	// create AffineTransform
	c.pdfBuilder.CreateAffineTransform(properties.Transform())
	transform := pdfStructure.AffineTransform()

	// rectangle, formatter, image. /AcroForm/DR/XObject contains that form
	if err := c.pdfBuilder.CreateSignatureRectangle(pdSignatureField, properties); err != nil {
		return nil, err
	}
	c.pdfBuilder.CreateFormatterRectangle(properties.FormatterRectangleParameters())
	bbox := pdfStructure.FormatterRectangle()
	if err := c.pdfBuilder.CreateSignatureImage(template, properties.Image()); err != nil {
		return nil, err
	}

	// create form stream, form and  resource.
	c.pdfBuilder.CreateHolderFormStream(template)
	holderFormStream := pdfStructure.HolderFormStream()
	c.pdfBuilder.CreateHolderFormResources()
	holderFormResources := pdfStructure.HolderFormResources()
	c.pdfBuilder.CreateHolderForm(holderFormResources, holderFormStream, bbox)

	// that is /AP entry the appearance dictionary.
	if err := c.pdfBuilder.CreateAppearanceDictionary(
		pdfStructure.HolderForm(), pdSignatureField); err != nil {
		return nil, err
	}

	// inner form stream, form and resource (holder form contains inner form)
	c.pdfBuilder.CreateInnerFormStream(template)
	c.pdfBuilder.CreateInnerFormResource()
	innerFormResource := pdfStructure.InnerFormResources()
	c.pdfBuilder.CreateInnerForm(innerFormResource, pdfStructure.InnerFormStream(), bbox)
	innerForm := pdfStructure.InnerForm()

	// inner form must be in the holder form as we wrote
	c.pdfBuilder.InsertInnerFormToHolderResources(innerForm, holderFormResources)

	//  Image form is in this structure: /AcroForm/DR/FRM/Resources/XObject/n2
	c.pdfBuilder.CreateImageFormStream(template)
	imageFormStream := pdfStructure.ImageFormStream()
	c.pdfBuilder.CreateImageFormResources()
	imageFormResources := pdfStructure.ImageFormResources()
	if err := c.pdfBuilder.CreateImageForm(imageFormResources, innerFormResource, imageFormStream,
		bbox, transform, pdfStructure.Image()); err != nil {
		return nil, err
	}

	if err := c.pdfBuilder.CreateBackgroundLayerForm(innerFormResource, bbox); err != nil {
		return nil, err
	}

	// now inject procSetArray
	c.pdfBuilder.InjectProcSetArray(innerForm, page, innerFormResource, imageFormResources,
		holderFormResources, pdfStructure.ProcSet())

	imageFormName := pdfStructure.ImageFormName()
	imageName := pdfStructure.ImageName()
	innerFormName := pdfStructure.InnerFormName()

	// now create Streams of AP
	if err := c.pdfBuilder.InjectAppearanceStreams(holderFormStream, imageFormStream,
		imageFormStream, imageFormName, imageName, innerFormName, properties); err != nil {
		return nil, err
	}
	c.pdfBuilder.CreateVisualSignature(template)
	if err := c.pdfBuilder.CreateWidgetDictionary(
		pdSignatureField, holderFormResources); err != nil {
		return nil, err
	}

	in, err := visualSignatureAsStream(pdfStructure.VisualSignature())
	if err != nil {
		return nil, err
	}
	slog.Info("visible: stream returning started", slog.Int("size", in.Len()))

	// return result of the stream
	return in, nil
}

// visualSignatureAsStream writes the given COS document out. Java declares it
// private.
func visualSignatureAsStream(visualSignature *cos.Document) (*bytes.Reader, error) {
	baos := &bytes.Buffer{}
	writer := pdfwriter.NewCOSWriter(baos)
	if err := pdmodel.WriteCOSDocument(writer, visualSignature); err != nil {
		return nil, err
	}
	return bytes.NewReader(baos.Bytes()), nil
}
