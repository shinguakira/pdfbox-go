package multipdf

// Adds an overlay to an existing PDF document.
//
// Port of org.apache.pdfbox.multipdf.Overlay. Java's class implements Closeable
// and the port keeps that as a Close method rather than an interface, because
// what Java closes is the documents it opened and nothing else.

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"strconv"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// Position is the possible location of the overlaid pages: foreground or
// background.
type Position int

const (
	// Background is Position.BACKGROUND, and is the default. Java's enum lists
	// FOREGROUND first; the port puts the default first so that the zero value
	// is the default, the way DocumentMergeMode does.
	Background Position = iota

	// Foreground is Position.FOREGROUND.
	Foreground
)

// layoutPage stores the overlay page information.
type layoutPage struct {
	overlayMediaBox  *common.PDRectangle
	overlayCOSStream *cos.Stream
	overlayResources *cos.Dictionary
	overlayRotation  int
}

// Overlay adds one document's first page over or under every page of another.
type Overlay struct {
	defaultOverlayPage            *layoutPage
	rotatedDefaultOverlayPagesMap map[int]*layoutPage
	firstPageOverlayPage          *layoutPage
	lastPageOverlayPage           *layoutPage
	oddPageOverlayPage            *layoutPage
	evenPageOverlayPage           *layoutPage

	openDocumentsSet                 []*pdmodel.PDDocument
	specificPageOverlayLayoutPageMap map[int]*layoutPage

	position Position

	inputFileName    string
	inputPDFDocument *pdmodel.PDDocument

	defaultOverlayFilename string
	defaultOverlayDocument *pdmodel.PDDocument

	firstPageOverlayFilename string
	firstPageOverlayDocument *pdmodel.PDDocument

	lastPageOverlayFilename string
	lastPageOverlayDocument *pdmodel.PDDocument

	allPagesOverlayFilename string
	allPagesOverlayDocument *pdmodel.PDDocument

	oddPageOverlayFilename string
	oddPageOverlayDocument *pdmodel.PDDocument

	evenPageOverlayFilename string
	evenPageOverlayDocument *pdmodel.PDDocument

	numberOfOverlayPages int
	useAllOverlayPages   bool
	adjustRotation       bool
}

// NewOverlay returns an overlay with nothing set on it.
//
// Java has no constructor; the maps it initialises at their declarations are
// initialised here.
func NewOverlay() *Overlay {
	return &Overlay{
		rotatedDefaultOverlayPagesMap:    map[int]*layoutPage{},
		specificPageOverlayLayoutPageMap: map[int]*layoutPage{},
	}
}

// ErrNoInputDocument is Java's `new IllegalArgumentException("No input
// document")`, which loadPDFs throws before anything is read.
var ErrNoInputDocument = errors.New("No input document")

// Overlay adds overlays to a document, taking a map of overlay file names for
// specific pages. The page numbers are 1-based, and the map must be empty (but
// not nil) if no specific mappings are used.
//
// It answers the modified input PDF document, which has to be saved and closed
// by the caller. If the input document was passed by SetInputPDF then it is
// that object that is returned.
func (o *Overlay) Overlay(specificPageOverlayMap map[int]string) (*pdmodel.PDDocument, error) {
	layouts := map[string]*layoutPage{}
	if err := o.loadPDFs(); err != nil {
		return nil, err
	}
	for pageNumber, path := range specificPageOverlayMap {
		page := layouts[path]
		if page == nil {
			doc, err := loadPDFFile(path)
			if err != nil {
				return nil, err
			}
			page, err = o.createLayoutPageFromDocument(doc)
			if err != nil {
				return nil, err
			}
			layouts[path] = page
			o.openDocumentsSet = append(o.openDocumentsSet, doc)
		}
		o.specificPageOverlayLayoutPageMap[pageNumber] = page
	}
	if err := o.processPages(o.inputPDFDocument); err != nil {
		return nil, err
	}
	return o.inputPDFDocument, nil
}

// OverlayDocuments adds overlay documents to a document. "If you created the
// overlay documents with subsetted fonts, you need to save them first so that
// the subsetting gets done."
func (o *Overlay) OverlayDocuments(
	specificPageOverlayDocumentMap map[int]*pdmodel.PDDocument) (*pdmodel.PDDocument, error) {
	if err := o.loadPDFs(); err != nil {
		return nil, err
	}
	for pageNumber, doc := range specificPageOverlayDocumentMap {
		if doc == nil {
			continue
		}
		page, err := o.createLayoutPageFromDocument(doc)
		if err != nil {
			return nil, err
		}
		o.specificPageOverlayLayoutPageMap[pageNumber] = page
	}
	if err := o.processPages(o.inputPDFDocument); err != nil {
		return nil, err
	}
	return o.inputPDFDocument, nil
}

// Close closes all input documents which were used for the overlay and opened
// by this class.
func (o *Overlay) Close() error {
	for _, doc := range []*pdmodel.PDDocument{
		o.defaultOverlayDocument, o.firstPageOverlayDocument, o.lastPageOverlayDocument,
		o.allPagesOverlayDocument, o.oddPageOverlayDocument, o.evenPageOverlayDocument,
	} {
		if doc != nil {
			if err := doc.Close(); err != nil {
				return err
			}
		}
	}
	for _, doc := range o.openDocumentsSet {
		if err := doc.Close(); err != nil {
			return err
		}
	}
	o.openDocumentsSet = nil
	o.specificPageOverlayLayoutPageMap = map[int]*layoutPage{}
	o.rotatedDefaultOverlayPagesMap = map[int]*layoutPage{}
	return nil
}

// loadPDFs opens every file that was named and builds a layout page from every
// document there now is.
func (o *Overlay) loadPDFs() error {
	// input PDF
	if o.inputFileName != "" {
		doc, err := loadPDFFile(o.inputFileName)
		if err != nil {
			return err
		}
		o.inputPDFDocument = doc
	}
	if o.inputPDFDocument == nil {
		return ErrNoInputDocument
	}
	for _, each := range []struct {
		filename string
		document **pdmodel.PDDocument
		page     **layoutPage
	}{
		// default overlay PDF
		{o.defaultOverlayFilename, &o.defaultOverlayDocument, &o.defaultOverlayPage},
		// first page overlay PDF
		{o.firstPageOverlayFilename, &o.firstPageOverlayDocument, &o.firstPageOverlayPage},
		// last page overlay PDF
		{o.lastPageOverlayFilename, &o.lastPageOverlayDocument, &o.lastPageOverlayPage},
		// odd pages overlay PDF
		{o.oddPageOverlayFilename, &o.oddPageOverlayDocument, &o.oddPageOverlayPage},
		// even pages overlay PDF
		{o.evenPageOverlayFilename, &o.evenPageOverlayDocument, &o.evenPageOverlayPage},
	} {
		if each.filename != "" {
			doc, err := loadPDFFile(each.filename)
			if err != nil {
				return err
			}
			*each.document = doc
		}
		if *each.document != nil {
			page, err := o.createLayoutPageFromDocument(*each.document)
			if err != nil {
				return err
			}
			*each.page = page
		}
	}
	// all pages overlay PDF
	if o.allPagesOverlayFilename != "" {
		doc, err := loadPDFFile(o.allPagesOverlayFilename)
		if err != nil {
			return err
		}
		o.allPagesOverlayDocument = doc
	}
	if o.allPagesOverlayDocument != nil {
		pages, err := o.createPageOverlayLayoutPageMap(o.allPagesOverlayDocument)
		if err != nil {
			return err
		}
		o.specificPageOverlayLayoutPageMap = pages
		o.useAllOverlayPages = true
		o.numberOfOverlayPages = len(o.specificPageOverlayLayoutPageMap)
	}
	return nil
}

// createLayoutPageFromDocument creates a layoutPage from the first page of the
// given document.
func (o *Overlay) createLayoutPageFromDocument(doc *pdmodel.PDDocument) (*layoutPage, error) {
	return o.createLayoutPage(doc.Page(0))
}

// createLayoutPage creates a layoutPage from the given page.
func (o *Overlay) createLayoutPage(page *pdmodel.PDPage) (*layoutPage, error) {
	contents := page.Dictionary().GetDictionaryObject(cos.Contents)
	resources := page.Resources()
	if resources == nil {
		resources = pdmodel.NewPDResources()
	}
	combined, err := o.createCombinedContentStream(contents)
	if err != nil {
		return nil, err
	}
	return &layoutPage{
		overlayMediaBox:  page.MediaBox(),
		overlayCOSStream: combined,
		overlayResources: resources.Dictionary(),
		overlayRotation:  page.Rotation(),
	}, nil
}

func (o *Overlay) createPageOverlayLayoutPageMap(
	doc *pdmodel.PDDocument) (map[int]*layoutPage, error) {
	i := 0
	layoutPages := map[int]*layoutPage{}
	for page := range doc.Pages().All {
		built, err := o.createLayoutPage(page)
		if err != nil {
			return nil, err
		}
		layoutPages[i] = built
		i++
	}
	return layoutPages, nil
}

// createCombinedContentStream concatenates a page's content streams into one.
func (o *Overlay) createCombinedContentStream(contents cos.Base) (*cos.Stream, error) {
	contentStreams, err := createContentStreamList(contents)
	if err != nil {
		return nil, err
	}
	// concatenate streams
	concatStream := o.inputPDFDocument.Document().CreateStream()
	out, err := concatStream.CreateWriterWithFilters(cos.FlateDecode)
	if err != nil {
		return nil, err
	}
	for _, contentStream := range contentStreams {
		in, err := contentStream.CreateReader()
		if err != nil {
			out.Close()
			return nil, err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return nil, err
		}
	}
	if err := out.Close(); err != nil {
		return nil, err
	}
	return concatStream, nil
}

// createContentStreamList gets the content streams as a list.
func createContentStreamList(contents cos.Base) ([]*cos.Stream, error) {
	switch value := contents.(type) {
	case nil:
		return nil, nil
	case *cos.Stream:
		return []*cos.Stream{value}, nil
	case *cos.Array:
		var contentStreams []*cos.Stream
		for i := 0; i < value.Size(); i++ {
			nested, err := createContentStreamList(value.Get(i))
			if err != nil {
				return nil, err
			}
			contentStreams = append(contentStreams, nested...)
		}
		return contentStreams, nil
	case *cos.Object:
		return createContentStreamList(value.Object())
	}
	return nil, fmt.Errorf("Unknown content type: %T", contents)
}

// processPages puts an overlay on every page that has one.
func (o *Overlay) processPages(document *pdmodel.PDDocument) error {
	pageCounter := 0
	cloner := NewPDFCloneUtility(document)
	pageTree := document.Pages()
	numberOfPages := pageTree.Count()
	for page := range pageTree.All {
		pageCounter++
		overlayFor, err := o.getLayoutPage(pageCounter, numberOfPages)
		if err != nil {
			return err
		}
		if overlayFor == nil {
			continue
		}
		pageDictionary := page.Dictionary()
		originalContent := pageDictionary.GetDictionaryObject(cos.Contents)
		newContentArray := cos.NewArray()
		switch o.position {
		case Foreground:
			// save state
			saved, err := o.createStream("q\n")
			if err != nil {
				return err
			}
			newContentArray.Add(saved)
			if err := addOriginalContent(originalContent, newContentArray); err != nil {
				return err
			}
			// restore state
			restored, err := o.createStream("Q\n")
			if err != nil {
				return err
			}
			newContentArray.Add(restored)
			// overlay content last
			if err := o.overlayPage(page, overlayFor, newContentArray, cloner); err != nil {
				return err
			}
		case Background:
			// overlay content first
			if err := o.overlayPage(page, overlayFor, newContentArray, cloner); err != nil {
				return err
			}
			if err := addOriginalContent(originalContent, newContentArray); err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unknown type of position:%v", o.position)
		}
		pageDictionary.SetItem(cos.Contents, newContentArray)
	}
	return nil
}

func addOriginalContent(contents cos.Base, contentArray *cos.Array) error {
	switch value := contents.(type) {
	case nil:
		return nil
	case *cos.Stream:
		contentArray.Add(value)
		return nil
	case *cos.Array:
		contentArray.AddAll(value.ToList())
		return nil
	}
	return fmt.Errorf("Unknown content type: %T", contents)
}

func (o *Overlay) overlayPage(page *pdmodel.PDPage, overlayFor *layoutPage,
	array *cos.Array, cloner *PDFCloneUtility) error {
	resources := page.Resources()
	if resources == nil {
		resources = pdmodel.NewPDResources()
		page.SetResources(resources)
	}
	overlayFormXObject, err := o.createOverlayFormXObject(overlayFor, cloner)
	if err != nil {
		return err
	}
	formXObjectID := resources.AddXObject(overlayFormXObject, "OL")
	stream, err := o.createOverlayStream(page, overlayFor, formXObjectID)
	if err != nil {
		return err
	}
	array.Add(stream)
	return nil
}

// getLayoutPage answers the overlay that belongs on the given page, or nil.
func (o *Overlay) getLayoutPage(pageNumber, numberOfPages int) (*layoutPage, error) {
	specific, hasSpecific := o.specificPageOverlayLayoutPageMap[pageNumber]
	switch {
	case !o.useAllOverlayPages && hasSpecific:
		return specific, nil
	case pageNumber == 1 && o.firstPageOverlayPage != nil:
		return o.firstPageOverlayPage, nil
	case pageNumber == numberOfPages && o.lastPageOverlayPage != nil:
		return o.lastPageOverlayPage, nil
	case pageNumber%2 == 1 && o.oddPageOverlayPage != nil:
		return o.oddPageOverlayPage, nil
	case pageNumber%2 == 0 && o.evenPageOverlayPage != nil:
		return o.evenPageOverlayPage, nil
	case o.defaultOverlayPage != nil:
		if o.adjustRotation {
			// PDFBOX-6049: consider the rotation of the document page
			// Note that this segment is only the second best solution to the
			// problem. The best would be to make appropriate transforms in
			// calculateAffineTransform()
			page := o.inputPDFDocument.Page(pageNumber - 1)
			if rotation := page.Rotation(); rotation != 0 {
				return o.createAdjustedLayoutPage(rotation)
			}
		}
		return o.defaultOverlayPage, nil
	case o.useAllOverlayPages:
		usePageNum := (pageNumber - 1) % o.numberOfOverlayPages
		return o.specificPageOverlayLayoutPageMap[usePageNum], nil
	}
	return nil, nil
}

func (o *Overlay) createAdjustedLayoutPage(rotation int) (*layoutPage, error) {
	rotatedLayoutPage := o.rotatedDefaultOverlayPagesMap[rotation]
	if rotatedLayoutPage == nil {
		// createLayoutPage must be called because we can't reuse the COSStream
		built, err := o.createLayoutPage(o.defaultOverlayDocument.Page(0))
		if err != nil {
			return nil, err
		}
		rotatedLayoutPage = built
		newRotation := (rotatedLayoutPage.overlayRotation - rotation + 360) % 360
		rotatedLayoutPage.overlayRotation = newRotation
		o.rotatedDefaultOverlayPagesMap[rotation] = rotatedLayoutPage
	}
	return rotatedLayoutPage, nil
}

func (o *Overlay) createOverlayFormXObject(overlayFor *layoutPage,
	cloner *PDFCloneUtility) (*form.PDFormXObject, error) {
	xobjForm := form.NewPDFormXObjectOfStream(overlayFor.overlayCOSStream)
	clonedResources, err := cloner.CloneDictionaryForNewDocument(overlayFor.overlayResources)
	if err != nil {
		return nil, err
	}
	xobjForm.SetResources(pdmodel.NewPDResourcesOf(clonedResources))
	xobjForm.SetFormType(1)
	xobjForm.SetBBox(overlayFor.overlayMediaBox.CreateRetranslatedRectangle())
	at := geom.NewIdentityTransform()
	switch overlayFor.overlayRotation {
	case 90:
		at.Translate(0, float64(overlayFor.overlayMediaBox.Width()))
		at.QuadrantRotate(3) // 270
	case 180:
		at.Translate(float64(overlayFor.overlayMediaBox.Width()),
			float64(overlayFor.overlayMediaBox.Height()))
		at.QuadrantRotate(2) // 180
	case 270:
		at.Translate(float64(overlayFor.overlayMediaBox.Height()), 0)
		at.QuadrantRotate(1) // 90
	}
	xobjForm.SetMatrix(util.NewMatrixFromAffineTransform(at))
	return xobjForm, nil
}

func (o *Overlay) createOverlayStream(page *pdmodel.PDPage, overlayFor *layoutPage,
	xObjectID *cos.Name) (*cos.Stream, error) {
	// create a new content stream that executes the XObject content
	var overlayStream strings.Builder
	overlayStream.WriteString("q\nq\n")
	overlayMediaBox := common.NewPDRectangleOfCOSArray(overlayFor.overlayMediaBox.COSArray())
	if overlayFor.overlayRotation == 90 || overlayFor.overlayRotation == 270 {
		overlayMediaBox.SetLowerLeftX(overlayFor.overlayMediaBox.LowerLeftY())
		overlayMediaBox.SetLowerLeftY(overlayFor.overlayMediaBox.LowerLeftX())
		overlayMediaBox.SetUpperRightX(overlayFor.overlayMediaBox.UpperRightY())
		overlayMediaBox.SetUpperRightY(overlayFor.overlayMediaBox.UpperRightX())
	}
	at := o.CalculateAffineTransform(page, overlayMediaBox)
	flatmatrix := make([]float64, 6)
	at.GetMatrix(flatmatrix)
	for _, v := range flatmatrix {
		overlayStream.WriteString(float2String(float32(v)))
		overlayStream.WriteByte(' ')
	}
	overlayStream.WriteString(" cm\n")

	// if debugging, insert
	// 0 0 overlayMediaBox.getHeight() overlayMediaBox.getWidth() re\ns\n
	// into the content stream

	overlayStream.WriteString(" /")
	overlayStream.WriteString(xObjectID.Name())
	overlayStream.WriteString(" Do Q\nQ\n")
	return o.createStream(overlayStream.String())
}

// CalculateAffineTransform calculates the transform to be used when positioning
// the overlay.
//
// "The default implementation centers on the destination, and this is
// calculated from the lower left of the media box of the destination (this has
// been changed from 3.0 and 2.0, see PDFBOX-6048 for details). Override this
// method to do your own, e.g. move to a corner, rotate, or zoom." Java's method
// is protected and this is what Go has instead: a caller that wants another
// transform embeds Overlay and shadows it, which reaches only its own calls --
// see migration/conventions/java-to-go.md.
func (o *Overlay) CalculateAffineTransform(page *pdmodel.PDPage,
	overlayMediaBox *common.PDRectangle) *geom.AffineTransform {
	at := geom.NewIdentityTransform()
	pageMediaBox := page.MediaBox()
	hShift := pageMediaBox.LowerLeftX() +
		(pageMediaBox.Width()-overlayMediaBox.Width())/2.0
	vShift := pageMediaBox.LowerLeftY() +
		(pageMediaBox.Height()-overlayMediaBox.Height())/2.0
	slog.Debug("Overlay position", "h", hShift, "v", vShift)
	at.Translate(float64(hShift), float64(vShift))
	return at
}

// float2String renders a float the way the content stream wants it.
//
// "use a BigDecimal as intermediate state to avoid a floating point string
// representation of the float value" -- `new BigDecimal(String.valueOf(f))`
// takes Java's shortest round-tripping rendering of the float and then prints
// it without an exponent, which is what strconv and big.Float do here.
func float2String(floatValue float32) string {
	// 'g' with -1 precision gives the shortest representation that round-trips,
	// which is what String.valueOf(float) also produces; `toPlainString` then
	// re-renders it without an exponent. cos.Float does the same for the same
	// reason, and this is that code rather than a call to it because the
	// function there is unexported.
	stringValue := strconv.FormatFloat(float64(floatValue), 'g', -1, 32)
	if strings.ContainsAny(stringValue, "eE") {
		if plain, _, err := big.ParseFloat(stringValue, 10, 200, big.ToNearestEven); err == nil {
			stringValue = plain.Text('f', -1)
		}
	}
	if !strings.Contains(stringValue, ".") {
		// Java always renders a float with a fraction part
		stringValue += ".0"
	}
	// remove fraction digit "0" only
	if strings.Contains(stringValue, ".") && !strings.HasSuffix(stringValue, ".0") {
		for strings.HasSuffix(stringValue, "0") && !strings.HasSuffix(stringValue, ".0") {
			stringValue = stringValue[:len(stringValue)-1]
		}
	}
	return stringValue
}

// createStream writes a content stream, compressing it only where it is long
// enough to be worth it.
func (o *Overlay) createStream(content string) (*cos.Stream, error) {
	stream := o.inputPDFDocument.Document().CreateStream()
	var filters cos.Base
	if len(content) > 20 {
		filters = cos.FlateDecode
	}
	out, err := stream.CreateWriterWithFilters(filters)
	if err != nil {
		return nil, err
	}
	// ISO-8859-1, which for the bytes a content stream holds is the identity.
	if _, err := out.Write([]byte(content)); err != nil {
		out.Close()
		return nil, err
	}
	if err := out.Close(); err != nil {
		return nil, err
	}
	return stream, nil
}

// SetOverlayPosition sets the overlay position.
func (o *Overlay) SetOverlayPosition(overlayPosition Position) { o.position = overlayPosition }

// SetInputFile sets the file to be overlaid.
func (o *Overlay) SetInputFile(inputFile string) { o.inputFileName = inputFile }

// SetInputPDF sets the PDF to be overlaid.
func (o *Overlay) SetInputPDF(inputPDF *pdmodel.PDDocument) { o.inputPDFDocument = inputPDF }

// InputFile returns the input file.
func (o *Overlay) InputFile() string { return o.inputFileName }

// SetDefaultOverlayFile sets the default overlay file.
func (o *Overlay) SetDefaultOverlayFile(file string) { o.defaultOverlayFilename = file }

// SetDefaultOverlayPDF sets the default overlay PDF.
func (o *Overlay) SetDefaultOverlayPDF(pdf *pdmodel.PDDocument) { o.defaultOverlayDocument = pdf }

// DefaultOverlayFile returns the default overlay file.
func (o *Overlay) DefaultOverlayFile() string { return o.defaultOverlayFilename }

// SetFirstPageOverlayFile sets the first page overlay file.
func (o *Overlay) SetFirstPageOverlayFile(file string) { o.firstPageOverlayFilename = file }

// SetFirstPageOverlayPDF sets the first page overlay PDF.
func (o *Overlay) SetFirstPageOverlayPDF(pdf *pdmodel.PDDocument) { o.firstPageOverlayDocument = pdf }

// SetLastPageOverlayFile sets the last page overlay file.
func (o *Overlay) SetLastPageOverlayFile(file string) { o.lastPageOverlayFilename = file }

// SetLastPageOverlayPDF sets the last page overlay PDF.
func (o *Overlay) SetLastPageOverlayPDF(pdf *pdmodel.PDDocument) { o.lastPageOverlayDocument = pdf }

// SetAllPagesOverlayFile sets the all pages overlay file.
func (o *Overlay) SetAllPagesOverlayFile(file string) { o.allPagesOverlayFilename = file }

// SetAllPagesOverlayPDF sets the all pages overlay PDF.
func (o *Overlay) SetAllPagesOverlayPDF(pdf *pdmodel.PDDocument) { o.allPagesOverlayDocument = pdf }

// SetOddPageOverlayFile sets the odd page overlay file.
func (o *Overlay) SetOddPageOverlayFile(file string) { o.oddPageOverlayFilename = file }

// SetOddPageOverlayPDF sets the odd page overlay PDF.
func (o *Overlay) SetOddPageOverlayPDF(pdf *pdmodel.PDDocument) { o.oddPageOverlayDocument = pdf }

// SetEvenPageOverlayFile sets the even page overlay file.
func (o *Overlay) SetEvenPageOverlayFile(file string) { o.evenPageOverlayFilename = file }

// SetEvenPageOverlayPDF sets the even page overlay PDF.
func (o *Overlay) SetEvenPageOverlayPDF(pdf *pdmodel.PDDocument) { o.evenPageOverlayDocument = pdf }

// SetAdjustRotation sets whether the overlay is rotated to match the page it
// goes on. See PDFBOX-6049.
func (o *Overlay) SetAdjustRotation(adjustRotation bool) { o.adjustRotation = adjustRotation }
