package visible

import (
	goimage "image"
	"io"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// PDVisibleSignDesigner holds the size and placement of the appearance of a
// visible signature.
//
// Port of PDVisibleSignDesigner.
//
// Java's imageWidth and imageHeight are a boxed Float each, so a designer built
// from a document alone leaves them null and the first getWidth throws
// NullPointerException. Go has no boxed float, so the port holds them as float32
// and they start at zero; a caller that has not set an image or a size gets 0
// where Java throws.
type PDVisibleSignDesigner struct {
	imageWidth  float32
	imageHeight float32
	xAxis       float32
	yAxis       float32
	pageHeight  float32
	pageWidth   float32
	image       goimage.Image

	signatureFieldName           string
	formatterRectangleParameters []int
	affineTransform              *geom.AffineTransform
	imageSizeInPercents          float32
	rotation                     int
}

// newPDVisibleSignDesigner returns a designer with the defaults the field
// initialisers of Java give it.
func newPDVisibleSignDesigner() *PDVisibleSignDesigner {
	return &PDVisibleSignDesigner{
		signatureFieldName:           "sig",
		formatterRectangleParameters: []int{0, 0, 100, 50},
		affineTransform:              geom.NewIdentityTransform(),
	}
}

// NewPDVisibleSignDesignerOfFile returns a designer sized from the given page of
// the PDF at the given path, drawing the image the given reader holds.
//
// Port of PDVisibleSignDesigner(String, InputStream, int).
func NewPDVisibleSignDesignerOfFile(filename string, imageStream io.Reader,
	page int) (*PDVisibleSignDesigner, error) {
	d := newPDVisibleSignDesigner()
	// set visible signature image Input stream
	if err := d.readImageStream(imageStream); err != nil {
		return nil, err
	}
	// calculate height and width of document page
	if err := d.calculatePageSizeFromFile(filename, page); err != nil {
		return nil, err
	}
	return d, nil
}

// NewPDVisibleSignDesignerOfSource returns a designer sized from the given page
// of the PDF the given read holds, drawing the image the given reader holds.
//
// Port of PDVisibleSignDesigner(RandomAccessRead, InputStream, int).
func NewPDVisibleSignDesignerOfSource(documentSource pdfio.RandomAccessRead,
	imageStream io.Reader, page int) (*PDVisibleSignDesigner, error) {
	d := newPDVisibleSignDesigner()
	// set visible signature image Input stream
	if err := d.readImageStream(imageStream); err != nil {
		return nil, err
	}
	// calculate height and width of document page
	if err := d.calculatePageSizeFromRandomAccessRead(documentSource, page); err != nil {
		return nil, err
	}
	return d, nil
}

// NewPDVisibleSignDesignerOfDocument returns a designer sized from the given
// page of the given document, drawing the image the given reader holds.
//
// Port of PDVisibleSignDesigner(PDDocument, InputStream, int).
func NewPDVisibleSignDesignerOfDocument(document *pdmodel.PDDocument, imageStream io.Reader,
	page int) (*PDVisibleSignDesigner, error) {
	d := newPDVisibleSignDesigner()
	if err := d.readImageStream(imageStream); err != nil {
		return nil, err
	}
	d.calculatePageSize(document, page)
	return d, nil
}

// NewPDVisibleSignDesignerOfFileAndImage returns a designer sized from the given
// page of the PDF at the given path, drawing the given image.
//
// Port of PDVisibleSignDesigner(String, BufferedImage, int).
func NewPDVisibleSignDesignerOfFileAndImage(filename string, img goimage.Image,
	page int) (*PDVisibleSignDesigner, error) {
	d := newPDVisibleSignDesigner()
	// set visible signature image
	d.setImage(img)
	// calculate height and width of document page
	if err := d.calculatePageSizeFromFile(filename, page); err != nil {
		return nil, err
	}
	return d, nil
}

// NewPDVisibleSignDesignerOfSourceAndImage returns a designer sized from the
// given page of the PDF the given read holds, drawing the given image.
//
// Port of PDVisibleSignDesigner(RandomAccessRead, BufferedImage, int).
func NewPDVisibleSignDesignerOfSourceAndImage(documentSource pdfio.RandomAccessRead,
	img goimage.Image, page int) (*PDVisibleSignDesigner, error) {
	d := newPDVisibleSignDesigner()
	// set visible signature image
	d.setImage(img)
	// calculate height and width of document page
	if err := d.calculatePageSizeFromRandomAccessRead(documentSource, page); err != nil {
		return nil, err
	}
	return d, nil
}

// NewPDVisibleSignDesignerOfDocumentAndImage returns a designer sized from the
// given page of the given document, drawing the given image.
//
// Port of PDVisibleSignDesigner(PDDocument, BufferedImage, int).
func NewPDVisibleSignDesignerOfDocumentAndImage(document *pdmodel.PDDocument, img goimage.Image,
	page int) *PDVisibleSignDesigner {
	d := newPDVisibleSignDesigner()
	d.setImage(img)
	d.calculatePageSize(document, page)
	return d
}

// NewPDVisibleSignDesignerOfImageStream returns a designer drawing the image the
// given reader holds, with no page size of its own.
//
// Port of PDVisibleSignDesigner(InputStream).
func NewPDVisibleSignDesignerOfImageStream(imageStream io.Reader) (*PDVisibleSignDesigner, error) {
	d := newPDVisibleSignDesigner()
	// set visible signature image Input stream
	if err := d.readImageStream(imageStream); err != nil {
		return nil, err
	}
	return d, nil
}

// calculatePageSizeFromFile sizes the designer from the PDF at the given path.
// Java declares it private.
func (d *PDVisibleSignDesigner) calculatePageSizeFromFile(filename string, page int) error {
	document, err := pdfbox.LoadPDF(filename)
	if err != nil {
		return err
	}
	defer document.Close()
	// calculate height and width of document page
	d.calculatePageSize(document, page)
	return nil
}

// calculatePageSizeFromRandomAccessRead sizes the designer from the PDF the
// given read holds. Java declares it private.
func (d *PDVisibleSignDesigner) calculatePageSizeFromRandomAccessRead(
	documentSource pdfio.RandomAccessRead, page int) error {
	document, err := pdfbox.LoadPDFFrom(documentSource)
	if err != nil {
		return err
	}
	defer document.Close()
	// calculate height and width of document page
	d.calculatePageSize(document, page)
	return nil
}

// calculatePageSize sizes the designer from the given page of the given
// document. Java declares it private.
//
// Java throws IllegalArgumentException for a page below 1, which is unchecked,
// so the port panics.
func (d *PDVisibleSignDesigner) calculatePageSize(document *pdmodel.PDDocument, page int) {
	if page < 1 {
		panic("First page of pdf is 1, not " + itoa(page))
	}
	firstPage := document.Page(page - 1)
	mediaBox := firstPage.MediaBox()
	d.pageHeightOf(mediaBox.Height())
	d.pageWidth = mediaBox.Width()
	d.imageSizeInPercents = 100
	d.rotation = firstPage.Rotation() % 360
}

// AdjustForRotation turns the placement so that the appearance sits the right
// way up on a rotated page.
func (d *PDVisibleSignDesigner) AdjustForRotation() *PDVisibleSignDesigner {
	switch d.rotation {
	case 90:
		// https://stackoverflow.com/a/34359956/535646
		temp := d.yAxis
		d.yAxis = d.pageHeight - d.xAxis - d.imageWidth
		d.xAxis = temp
		d.affineTransform = geom.NewAffineTransform(
			0, float64(d.imageHeight/d.imageWidth),
			float64(-d.imageWidth/d.imageHeight), 0,
			float64(d.imageWidth), 0)
		temp = d.imageHeight
		d.imageHeight = d.imageWidth
		d.imageWidth = temp
	case 180:
		newX := d.pageWidth - d.xAxis - d.imageWidth
		newY := d.pageHeight - d.yAxis - d.imageHeight
		d.xAxis = newX
		d.yAxis = newY
		d.affineTransform = geom.NewAffineTransform(
			-1, 0, 0, -1, float64(d.imageWidth), float64(d.imageHeight))
	case 270:
		temp := d.xAxis
		d.xAxis = d.pageWidth - d.yAxis - d.imageHeight
		d.yAxis = temp
		d.affineTransform = geom.NewAffineTransform(
			0, float64(-d.imageHeight/d.imageWidth),
			float64(d.imageWidth/d.imageHeight), 0,
			0, float64(d.imageHeight))
		temp = d.imageHeight
		d.imageHeight = d.imageWidth
		d.imageWidth = temp
	}
	return d
}

// SignatureImage reads the image of the signature out of the file at the given
// path.
//
// Port of signatureImage(String).
func (d *PDVisibleSignDesigner) SignatureImage(path string) (*PDVisibleSignDesigner, error) {
	in, err := openFile(path)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	if err := d.readImageStream(in); err != nil {
		return nil, err
	}
	return d, nil
}

// Zoom grows or shrinks the appearance by the given percentage: 50 makes it
// half again as large, -50 halves it.
func (d *PDVisibleSignDesigner) Zoom(percent float32) *PDVisibleSignDesigner {
	d.imageHeight += (d.imageHeight * percent) / 100
	d.imageWidth += (d.imageWidth * percent) / 100
	d.formatterRectangleParameters[2] = int(d.imageWidth)
	d.formatterRectangleParameters[3] = int(d.imageHeight)
	return d
}

// Coordinates sets where the appearance sits on the page.
func (d *PDVisibleSignDesigner) Coordinates(x, y float32) *PDVisibleSignDesigner {
	d.XAxisOf(x)
	d.YAxisOf(y)
	return d
}

// XAxis returns where the appearance sits across the page.
func (d *PDVisibleSignDesigner) XAxis() float32 { return d.xAxis }

// XAxisOf sets where the appearance sits across the page.
//
// Java names it xAxis(float), overloading nothing; Go cannot have a method and
// an accessor of the same name.
func (d *PDVisibleSignDesigner) XAxisOf(xAxis float32) *PDVisibleSignDesigner {
	d.xAxis = xAxis
	return d
}

// YAxis returns where the appearance sits down the page.
func (d *PDVisibleSignDesigner) YAxis() float32 { return d.yAxis }

// YAxisOf sets where the appearance sits down the page.
func (d *PDVisibleSignDesigner) YAxisOf(yAxis float32) *PDVisibleSignDesigner {
	d.yAxis = yAxis
	return d
}

// Width returns how wide the appearance is.
func (d *PDVisibleSignDesigner) Width() float32 { return d.imageWidth }

// WidthOf sets how wide the appearance is.
func (d *PDVisibleSignDesigner) WidthOf(width float32) *PDVisibleSignDesigner {
	d.imageWidth = width
	d.formatterRectangleParameters[2] = int(width)
	return d
}

// Height returns how tall the appearance is.
func (d *PDVisibleSignDesigner) Height() float32 { return d.imageHeight }

// HeightOf sets how tall the appearance is.
func (d *PDVisibleSignDesigner) HeightOf(height float32) *PDVisibleSignDesigner {
	d.imageHeight = height
	d.formatterRectangleParameters[3] = int(height)
	return d
}

// TemplateHeight returns how tall the page is. Java declares it protected.
func (d *PDVisibleSignDesigner) TemplateHeight() float32 { return d.PageHeight() }

// pageHeightOf sets how tall the page is. Java declares it private.
func (d *PDVisibleSignDesigner) pageHeightOf(templateHeight float32) *PDVisibleSignDesigner {
	d.pageHeight = templateHeight
	return d
}

// SignatureFieldName returns the name of the signature field.
func (d *PDVisibleSignDesigner) SignatureFieldName() string { return d.signatureFieldName }

// SignatureFieldNameOf sets the name of the signature field.
func (d *PDVisibleSignDesigner) SignatureFieldNameOf(
	signatureFieldName string) *PDVisibleSignDesigner {
	d.signatureFieldName = signatureFieldName
	return d
}

// Image returns the image of the signature.
func (d *PDVisibleSignDesigner) Image() goimage.Image { return d.image }

// readImageStream decodes the image the given reader holds. Java declares it
// private, and turns the ImageIO cache off first because it writes to a
// temporary file; Go's image decoders have no such cache.
func (d *PDVisibleSignDesigner) readImageStream(stream io.Reader) error {
	img, _, err := goimage.Decode(stream)
	if err != nil {
		return err
	}
	d.setImage(img)
	return nil
}

// setImage keeps the image and takes its size. Java declares it private.
func (d *PDVisibleSignDesigner) setImage(img goimage.Image) {
	d.image = img
	bounds := img.Bounds()
	d.imageHeight = float32(bounds.Dy())
	d.imageWidth = float32(bounds.Dx())
	d.formatterRectangleParameters[2] = bounds.Dx()
	d.formatterRectangleParameters[3] = bounds.Dy()
}

// Transform returns the transform the image form is drawn through.
func (d *PDVisibleSignDesigner) Transform() *geom.AffineTransform { return d.affineTransform }

// TransformOf sets the transform the image form is drawn through, copying it as
// Java does.
func (d *PDVisibleSignDesigner) TransformOf(
	affineTransform *geom.AffineTransform) *PDVisibleSignDesigner {
	d.affineTransform = geom.NewAffineTransformOf(affineTransform)
	return d
}

// FormatterRectangleParameters returns the four corners of the bounding box of
// the forms.
func (d *PDVisibleSignDesigner) FormatterRectangleParameters() []int {
	return d.formatterRectangleParameters
}

// FormatterRectangleParametersOf sets the four corners of the bounding box of
// the forms.
func (d *PDVisibleSignDesigner) FormatterRectangleParametersOf(
	formatterRectangleParameters []int) *PDVisibleSignDesigner {
	d.formatterRectangleParameters = formatterRectangleParameters
	return d
}

// PageWidth returns how wide the page is.
func (d *PDVisibleSignDesigner) PageWidth() float32 { return d.pageWidth }

// PageWidthOf sets how wide the page is.
func (d *PDVisibleSignDesigner) PageWidthOf(pageWidth float32) *PDVisibleSignDesigner {
	d.pageWidth = pageWidth
	return d
}

// PageHeight returns how tall the page is.
func (d *PDVisibleSignDesigner) PageHeight() float32 { return d.pageHeight }

// ImageSizeInPercents returns how large the image is drawn, as a percentage.
func (d *PDVisibleSignDesigner) ImageSizeInPercents() float32 { return d.imageSizeInPercents }

// ImageSizeInPercentsOf sets how large the image is drawn, as a percentage.
func (d *PDVisibleSignDesigner) ImageSizeInPercentsOf(imageSizeInPercents float32) {
	d.imageSizeInPercents = imageSizeInPercents
}

// SignatureText panics: Java throws UnsupportedOperationException, which is
// unchecked.
func (d *PDVisibleSignDesigner) SignatureText() string {
	panic("That method is not yet implemented")
}

// SignatureTextOf panics: Java throws UnsupportedOperationException, which is
// unchecked.
func (d *PDVisibleSignDesigner) SignatureTextOf(signatureText string) *PDVisibleSignDesigner {
	panic("That method is not yet implemented")
}
