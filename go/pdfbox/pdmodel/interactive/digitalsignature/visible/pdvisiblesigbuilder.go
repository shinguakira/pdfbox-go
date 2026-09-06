package visible

import (
	goimage "image"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	graphicsform "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/digitalsignature"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// PDVisibleSigBuilder is the builder that draws the appearance of a visible
// signature.
//
// Port of PDVisibleSigBuilder.
type PDVisibleSigBuilder struct {
	pdfStructure *PDFTemplateStructure
}

var _ PDFTemplateBuilder = (*PDVisibleSigBuilder)(nil)

// NewPDVisibleSigBuilder returns a builder over a fresh structure.
func NewPDVisibleSigBuilder() *PDVisibleSigBuilder {
	b := &PDVisibleSigBuilder{pdfStructure: &PDFTemplateStructure{}}
	slog.Info("visible: PDF Structure has been created")
	return b
}

// CreatePage creates the page of the template.
func (b *PDVisibleSigBuilder) CreatePage(properties *PDVisibleSignDesigner) {
	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(properties.PageWidth(),
		properties.PageHeight()))
	b.pdfStructure.SetPage(page)
	slog.Info("visible: PDF page has been created")
}

// CreateTemplate creates the document holding the given page.
func (b *PDVisibleSigBuilder) CreateTemplate(page *pdmodel.PDPage) error {
	template := pdmodel.NewPDDocument()
	template.AddPage(page)
	b.pdfStructure.SetTemplate(template)
	return nil
}

// CreateAcroForm creates the form of the template.
func (b *PDVisibleSigBuilder) CreateAcroForm(template *pdmodel.PDDocument) {
	theAcroForm := form.NewPDAcroForm(template)
	form.SetAcroFormOfCatalog(template.DocumentCatalog(), theAcroForm)
	b.pdfStructure.SetAcroForm(theAcroForm)
	slog.Info("visible: AcroForm has been created")
}

// Structure returns what has been built so far.
func (b *PDVisibleSigBuilder) Structure() *PDFTemplateStructure { return b.pdfStructure }

// CreateSignatureField creates the signature field of the form.
func (b *PDVisibleSigBuilder) CreateSignatureField(acroForm *form.PDAcroForm) error {
	sf := form.NewPDSignatureField(acroForm)
	b.pdfStructure.SetSignatureField(sf)
	slog.Info("visible: Signature field has been created")
	return nil
}

// CreateSignature creates the signature the field holds and puts the widget on
// the page.
func (b *PDVisibleSigBuilder) CreateSignature(pdSignatureField *form.PDSignatureField,
	page *pdmodel.PDPage, signerName string) error {
	pdSignature := digitalsignature.NewPDSignature()
	widget := pdSignatureField.Widgets()[0]
	if err := pdSignatureField.SetSignatureValue(pdSignature); err != nil {
		return err
	}
	widget.SetPage(page)
	annotations := page.Annotations()
	annotations.Add(widget)
	if signerName != "" {
		pdSignature.SetName(signerName)
	}
	b.pdfStructure.SetPDSignature(pdSignature)
	slog.Info("visible: PDSignature has been created")
	return nil
}

// CreateAcroFormDictionary fills in the /DR entry of the form.
func (b *PDVisibleSigBuilder) CreateAcroFormDictionary(acroForm *form.PDAcroForm,
	signatureField *form.PDSignatureField) error {
	acroFormFields := acroForm.Fields()
	acroFormDict := acroForm.Dictionary()
	acroForm.SetSignaturesExist(true)
	acroForm.SetAppendOnly(true)
	acroFormDict.SetDirect(true)
	acroFormFields = append(acroFormFields, signatureField)
	// Java adds to the live COSArrayList getFields answers, which writes the
	// field through to /Fields; the port's Fields answers a plain slice, so the
	// append is written back here.
	acroForm.SetFields(acroFormFields)
	acroForm.SetDefaultAppearance("/Helv 0 Tf 0 g")
	b.pdfStructure.SetAcroFormFields(acroFormFields)
	b.pdfStructure.SetAcroFormDictionary(acroFormDict)
	slog.Info("visible: AcroForm dictionary has been created")
	return nil
}

// CreateSignatureRectangle sets the rectangle of the signature widget.
func (b *PDVisibleSigBuilder) CreateSignatureRectangle(signatureField *form.PDSignatureField,
	properties *PDVisibleSignDesigner) error {
	rect := common.NewPDRectangle()
	rect.SetUpperRightX(properties.XAxis() + properties.Width())
	rect.SetUpperRightY(properties.TemplateHeight() - properties.YAxis())
	rect.SetLowerLeftY(properties.TemplateHeight() - properties.YAxis() - properties.Height())
	rect.SetLowerLeftX(properties.XAxis())
	signatureField.Widgets()[0].SetRectangle(rect)
	b.pdfStructure.SetSignatureRectangle(rect)
	slog.Info("visible: Signature rectangle has been created")
	return nil
}

// CreateAffineTransform sets the transform the image form is drawn through.
func (b *PDVisibleSigBuilder) CreateAffineTransform(affineTransform *geom.AffineTransform) {
	b.pdfStructure.SetAffineTransform(affineTransform)
	slog.Info("visible: Matrix has been added")
}

// CreateProcSetArray creates the array of [PDF, Text, ImageB, ImageC, ImageI].
func (b *PDVisibleSigBuilder) CreateProcSetArray() {
	procSetArr := cos.NewArray()
	procSetArr.Add(cos.GetPDFName("PDF"))
	procSetArr.Add(cos.GetPDFName("Text"))
	procSetArr.Add(cos.GetPDFName("ImageB"))
	procSetArr.Add(cos.GetPDFName("ImageC"))
	procSetArr.Add(cos.GetPDFName("ImageI"))
	b.pdfStructure.SetProcSet(procSetArr)
	slog.Info("visible: ProcSet array has been created")
}

// CreateSignatureImage embeds the image of the signature.
func (b *PDVisibleSigBuilder) CreateSignatureImage(template *pdmodel.PDDocument,
	img goimage.Image) error {
	embedded, err := image.CreateFromImage(template.Document(), img)
	if err != nil {
		return err
	}
	b.pdfStructure.SetImage(embedded)
	slog.Info("visible: Visible Signature Image has been created")
	return nil
}

// CreateFormatterRectangle creates the bounding box of the forms.
func (b *PDVisibleSigBuilder) CreateFormatterRectangle(params []int) {
	formatterRectangle := common.NewPDRectangle()
	formatterRectangle.SetLowerLeftX(float32(min(params[0], params[2])))
	formatterRectangle.SetLowerLeftY(float32(min(params[1], params[3])))
	formatterRectangle.SetUpperRightX(float32(max(params[0], params[2])))
	formatterRectangle.SetUpperRightY(float32(max(params[1], params[3])))
	b.pdfStructure.SetFormatterRectangle(formatterRectangle)
	slog.Info("visible: Formatter rectangle has been created")
}

// CreateHolderFormStream creates the stream of the holder form.
func (b *PDVisibleSigBuilder) CreateHolderFormStream(template *pdmodel.PDDocument) {
	holderForm := common.NewPDStreamOfDocument(template.Document())
	b.pdfStructure.SetHolderFormStream(holderForm)
	slog.Info("visible: Holder form stream has been created")
}

// CreateHolderFormResources creates the resources of the holder form.
func (b *PDVisibleSigBuilder) CreateHolderFormResources() {
	holderFormResources := pdmodel.NewPDResources()
	b.pdfStructure.SetHolderFormResources(holderFormResources)
	slog.Info("visible: Holder form resources have been created")
}

// CreateHolderForm creates the holder form.
func (b *PDVisibleSigBuilder) CreateHolderForm(holderFormResources *pdmodel.PDResources,
	holderFormStream *common.PDStream, bbox *common.PDRectangle) {
	holderForm := graphicsform.NewPDFormXObjectOfPDStream(holderFormStream)
	holderForm.SetResources(holderFormResources)
	holderForm.SetBBox(bbox)
	holderForm.SetFormType(1)
	b.pdfStructure.SetHolderForm(holderForm)
	slog.Info("visible: Holder form has been created")
}

// CreateAppearanceDictionary creates the /AP entry of the widget.
func (b *PDVisibleSigBuilder) CreateAppearanceDictionary(
	holderForm *graphicsform.PDFormXObject, signatureField *form.PDSignatureField) error {
	appearance := annotation.NewPDAppearanceDictionary()
	appearance.Dictionary().SetDirect(true)
	appearanceStream := annotation.NewPDAppearanceStreamOf(holderForm.Stream())
	appearance.SetNormalAppearanceStream(appearanceStream)
	signatureField.Widgets()[0].SetAppearance(appearance)
	b.pdfStructure.SetAppearanceDictionary(appearance)
	slog.Info("visible: PDF appearance dictionary has been created")
	return nil
}

// CreateInnerFormStream creates the stream of the inner form.
func (b *PDVisibleSigBuilder) CreateInnerFormStream(template *pdmodel.PDDocument) {
	innerFormStream := common.NewPDStreamOfDocument(template.Document())
	b.pdfStructure.SetInnterFormStream(innerFormStream)
	slog.Info("visible: Stream of another form (inner form - it will be inside holder form) " +
		"has been created")
}

// CreateInnerFormResource creates the resources of the inner form.
func (b *PDVisibleSigBuilder) CreateInnerFormResource() {
	innerFormResources := pdmodel.NewPDResources()
	b.pdfStructure.SetInnerFormResources(innerFormResources)
	slog.Info("visible: Resources of another form (inner form - it will be inside holder form)" +
		"have been created")
}

// CreateInnerForm creates the inner form, which sits inside the holder form.
func (b *PDVisibleSigBuilder) CreateInnerForm(innerFormResources *pdmodel.PDResources,
	innerFormStream *common.PDStream, bbox *common.PDRectangle) {
	innerForm := graphicsform.NewPDFormXObjectOfPDStream(innerFormStream)
	innerForm.SetResources(innerFormResources)
	innerForm.SetBBox(bbox)
	innerForm.SetFormType(1)
	b.pdfStructure.SetInnerForm(innerForm)
	slog.Info("visible: Another form (inner form - it will be inside holder form) has been created")
}

// InsertInnerFormToHolderResources puts the inner form into the resources of
// the holder form.
func (b *PDVisibleSigBuilder) InsertInnerFormToHolderResources(
	innerForm *graphicsform.PDFormXObject, holderFormResources *pdmodel.PDResources) {
	holderFormResources.PutXObject(cos.FRM, innerForm)
	b.pdfStructure.SetInnerFormName(cos.FRM)
	slog.Info("visible: Now inserted inner form inside holder form")
}

// CreateImageFormStream creates the stream of the image form.
func (b *PDVisibleSigBuilder) CreateImageFormStream(template *pdmodel.PDDocument) {
	imageFormStream := common.NewPDStreamOfDocument(template.Document())
	b.pdfStructure.SetImageFormStream(imageFormStream)
	slog.Info("visible: Created image form stream")
}

// CreateImageFormResources creates the resources of the image form.
func (b *PDVisibleSigBuilder) CreateImageFormResources() {
	imageFormResources := pdmodel.NewPDResources()
	b.pdfStructure.SetImageFormResources(imageFormResources)
	slog.Info("visible: Created image form resources")
}

// CreateImageForm creates the image form, which sits inside the inner form.
func (b *PDVisibleSigBuilder) CreateImageForm(imageFormResources,
	innerFormResource *pdmodel.PDResources, imageFormStream *common.PDStream,
	bbox *common.PDRectangle, at *geom.AffineTransform, img *image.PDImageXObject) error {
	imageForm := graphicsform.NewPDFormXObjectOfPDStream(imageFormStream)
	imageForm.SetBBox(bbox)
	imageForm.SetMatrix(util.NewMatrixFromAffineTransform(at))
	imageForm.SetResources(imageFormResources)
	imageForm.SetFormType(1)
	imageFormResources.Dictionary().SetDirect(true)
	imageFormName := cos.GetPDFName("n2")
	innerFormResource.PutXObject(imageFormName, imageForm)
	imageName := imageFormResources.AddXObject(img, "img")
	b.pdfStructure.SetImageForm(imageForm)
	b.pdfStructure.SetImageFormName(imageFormName)
	b.pdfStructure.SetImageName(imageName)
	slog.Info("visible: Created image form")
	return nil
}

// CreateBackgroundLayerForm creates the blank n0 background layer form.
func (b *PDVisibleSigBuilder) CreateBackgroundLayerForm(innerFormResource *pdmodel.PDResources,
	bbox *common.PDRectangle) error {
	// create blank n0 background layer form
	n0Form := graphicsform.NewPDFormXObjectOfStream(
		b.pdfStructure.Template().Document().CreateStream())
	n0Form.SetBBox(bbox)
	n0Form.SetResources(pdmodel.NewPDResources())
	n0Form.SetFormType(1)
	innerFormResource.PutXObject(cos.GetPDFName("n0"), n0Form)
	slog.Info("visible: Created background layer form")
	return nil
}

// InjectProcSetArray puts the /ProcSet array into every resource dictionary and
// into the page.
func (b *PDVisibleSigBuilder) InjectProcSetArray(innerForm *graphicsform.PDFormXObject,
	page *pdmodel.PDPage, innerFormResources, imageFormResources,
	holderFormResources *pdmodel.PDResources, procSet *cos.Array) {
	innerFormResourcesOfForm, _ := innerForm.Resources().(*pdmodel.PDResources)
	innerFormResourcesOfForm.Dictionary().SetItem(cos.ProcSet, procSet)
	page.Dictionary().SetItem(cos.ProcSet, procSet)
	innerFormResources.Dictionary().SetItem(cos.ProcSet, procSet)
	imageFormResources.Dictionary().SetItem(cos.ProcSet, procSet)
	holderFormResources.Dictionary().SetItem(cos.ProcSet, procSet)
	slog.Info("visible: Inserted ProcSet to PDF")
}

// InjectAppearanceStreams writes the content of the three forms.
func (b *PDVisibleSigBuilder) InjectAppearanceStreams(holderFormStream, innerFormStream,
	imageFormStream *common.PDStream, imageFormName, imageName, innerFormName *cos.Name,
	properties *PDVisibleSignDesigner) error {
	// TODO remove unused parameter from interface??
	// Use width and height of BBox as values for transformation matrix.
	width := int(b.Structure().FormatterRectangle().Width())
	height := int(b.Structure().FormatterRectangle().Height())
	imgFormContent := "q " + itoa(width) + " 0 0 " + itoa(height) + " 0 0 cm /" +
		imageName.Name() + " Do Q\n"
	holderFormContent := "q 1 0 0 1 0 0 cm /" + innerFormName.Name() + " Do Q\n"
	innerFormContent := "q 1 0 0 1 0 0 cm /n0 Do Q q 1 0 0 1 0 0 cm /" +
		imageFormName.Name() + " Do Q\n"
	if err := b.WriteRawCommands(b.pdfStructure.HolderFormStream(), holderFormContent); err != nil {
		return err
	}
	if err := b.WriteRawCommands(b.pdfStructure.InnerFormStream(), innerFormContent); err != nil {
		return err
	}
	if err := b.WriteRawCommands(b.pdfStructure.ImageFormStream(), imgFormContent); err != nil {
		return err
	}
	slog.Info("visible: Injected appearance stream to pdf")
	return nil
}

// WriteRawCommands writes the given content stream operators into the stream.
func (b *PDVisibleSigBuilder) WriteRawCommands(stream *common.PDStream, commands string) error {
	os, err := stream.CreateOutputStream()
	if err != nil {
		return err
	}
	if _, err := os.Write([]byte(commands)); err != nil {
		os.Close()
		return err
	}
	return os.Close()
}

// CreateVisualSignature keeps the COS document of the template.
func (b *PDVisibleSigBuilder) CreateVisualSignature(template *pdmodel.PDDocument) {
	b.pdfStructure.SetVisualSignature(template.Document())
	slog.Info("visible: Visible signature has been created")
}

// CreateWidgetDictionary fills in the /DR entry of the signature widget.
func (b *PDVisibleSigBuilder) CreateWidgetDictionary(signatureField *form.PDSignatureField,
	holderFormResources *pdmodel.PDResources) error {
	widgetDict := signatureField.Widgets()[0].AnnotationDictionary()
	widgetDict.SetNeedToBeUpdated(true)
	widgetDict.SetItem(cos.DR, holderFormResources.COSObject())
	b.pdfStructure.SetWidgetDictionary(widgetDict)
	slog.Info("visible: WidgetDictionary has been created")
	return nil
}

// CloseTemplate closes the given document and the one of the structure.
func (b *PDVisibleSigBuilder) CloseTemplate(template *pdmodel.PDDocument) error {
	if err := template.Close(); err != nil {
		return err
	}
	return b.pdfStructure.Template().Close()
}
