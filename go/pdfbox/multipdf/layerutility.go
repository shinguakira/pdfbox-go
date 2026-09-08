package multipdf

// This class allows to import pages as Form XObjects into another document.
//
// Port of org.apache.pdfbox.multipdf.LayerUtility. "This is a utility class,
// not a Java Bean" -- everything it does is done through the target document it
// is constructed with.

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	fontutil "github.com/shinguakira/pdfbox-go/go/fontbox/util"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/form"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/optionalcontent"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// layerUtilityDebug is Java's `private static final boolean DEBUG = false`,
// which decides whether the layer's content stream is compressed.
const layerUtilityDebug = false

// LayerUtility imports pages as Form XObjects into another document, and
// places them as optional content groups.
type LayerUtility struct {
	targetDoc *pdmodel.PDDocument
	cloner    *PDFCloneUtility
}

// NewLayerUtility creates a new instance over the PDF document to modify.
func NewLayerUtility(targetDoc *pdmodel.PDDocument) *LayerUtility {
	return &LayerUtility{targetDoc: targetDoc, cloner: NewPDFCloneUtility(targetDoc)}
}

// Document returns the PDF document we work on.
func (u *LayerUtility) Document() *pdmodel.PDDocument { return u.targetDoc }

// WrapInSaveRestore adds a q/Q pair around the existing page's content.
//
// "Some applications may not wrap their page content in a save/restore (q/Q)
// pair which can lead to problems with coordinate system transformations when
// content is appended."
func (u *LayerUtility) WrapInSaveRestore(page *pdmodel.PDPage) error {
	saveGraphicsStateStream, err := u.writeStream("q\n")
	if err != nil {
		return err
	}
	restoreGraphicsStateStream, err := u.writeStream("Q\n")
	if err != nil {
		return err
	}

	// Wrap the existing page's content in a save/restore pair (q/Q) to have a
	// controlled environment to add additional content.
	pageDictionary := page.Dictionary()
	switch contents := pageDictionary.GetDictionaryObject(cos.Contents).(type) {
	case *cos.Stream:
		array := cos.NewArray()
		array.Add(saveGraphicsStateStream)
		array.Add(contents)
		array.Add(restoreGraphicsStateStream)
		pageDictionary.SetItem(cos.Contents, array)
		return nil
	case *cos.Array:
		contents.AddAt(0, saveGraphicsStateStream)
		contents.Add(restoreGraphicsStateStream)
		return nil
	default:
		return fmt.Errorf("Contents are unknown type: %T", contents)
	}
}

// writeStream is the four lines Java spends on each of the two wrapper streams.
func (u *LayerUtility) writeStream(content string) (*cos.Stream, error) {
	stream := u.Document().Document().CreateStream()
	out, err := stream.CreateWriter()
	if err != nil {
		return nil, err
	}
	// ISO-8859-1, which for these two bytes is the identity.
	if _, err := out.Write([]byte(content)); err != nil {
		out.Close()
		return nil, err
	}
	if err := out.Close(); err != nil {
		return nil, err
	}
	return stream, nil
}

// pageToFormFilter is the set of page entries importPageAsForm carries over to
// the form.
var pageToFormFilter = map[string]bool{"Group": true, "LastModified": true, "Metadata": true}

// ImportPageAsForm imports the given 0-based page of a PDF file as a Form
// XObject so it can be placed on another page in the target document.
//
// "You may want to call WrapInSaveRestore before invoking the Form XObject to
// make sure that the graphics state is reset."
func (u *LayerUtility) ImportPageAsForm(sourceDoc *pdmodel.PDDocument,
	pageNumber int) (*form.PDFormXObject, error) {
	return u.ImportPageAsFormOfPage(sourceDoc, sourceDoc.Page(pageNumber))
}

// ImportPageAsFormOfPage is the same for a page that is already in hand.
func (u *LayerUtility) ImportPageAsFormOfPage(sourceDoc *pdmodel.PDDocument,
	page *pdmodel.PDPage) (*form.PDFormXObject, error) {
	if err := u.importOcProperties(sourceDoc); err != nil {
		return nil, err
	}

	contents, err := page.Contents()
	if err != nil {
		return nil, err
	}
	newStream, err := common.NewPDStreamOfInput(u.targetDoc, contents, cos.FlateDecode)
	if err != nil {
		return nil, err
	}
	formXObject := form.NewPDFormXObjectOfPDStream(newStream)

	// Copy resources
	pageRes := page.Resources()
	formRes := pdmodel.NewPDResources()
	if pageRes != nil {
		if err := u.cloner.CloneMerge(pageRes, formRes); err != nil {
			return nil, err
		}
	}
	formXObject.SetResources(formRes)

	// Transfer some values from page to form
	if err := u.transferDict(page.Dictionary(),
		&formXObject.COSObject().(*cos.Stream).Dictionary, pageToFormFilter); err != nil {
		return nil, err
	}

	at := formXObject.Matrix().CreateAffineTransform()
	mediaBox := page.MediaBox()
	viewBox := page.CropBox()
	if viewBox == nil {
		viewBox = mediaBox
	}

	// Handle the /Rotation entry on the page dict
	rotation := page.Rotation()

	// Transform to FOP's user space
	// at.scale(1 / viewBox.getWidth(), 1 / viewBox.getHeight());
	at.Translate(float64(mediaBox.LowerLeftX()-viewBox.LowerLeftX()),
		float64(mediaBox.LowerLeftY()-viewBox.LowerLeftY()))
	switch rotation {
	case 90:
		at.Scale(float64(viewBox.Width()/viewBox.Height()),
			float64(viewBox.Height()/viewBox.Width()))
		at.Translate(0, float64(viewBox.Width()))
		at.QuadrantRotate(3) // 270
	case 180:
		at.Translate(float64(viewBox.Width()), float64(viewBox.Height()))
		at.QuadrantRotate(2) // 180
	case 270:
		at.Scale(float64(viewBox.Width()/viewBox.Height()),
			float64(viewBox.Height()/viewBox.Width()))
		at.Translate(float64(viewBox.Height()), 0)
		at.QuadrantRotate(1) // 90
	default:
		// no additional transformations necessary
	}
	// Compensate for Crop Boxes not starting at 0,0
	at.Translate(float64(-viewBox.LowerLeftX()), float64(-viewBox.LowerLeftY()))
	if !at.IsIdentity() {
		formXObject.SetMatrix(util.NewMatrixFromAffineTransform(at))
	}

	bbox := fontutil.NewBoundingBox()
	bbox.SetLowerLeftX(viewBox.LowerLeftX())
	bbox.SetLowerLeftY(viewBox.LowerLeftY())
	bbox.SetUpperRightX(viewBox.UpperRightX())
	bbox.SetUpperRightY(viewBox.UpperRightY())
	formXObject.SetBBox(common.NewPDRectangleOfBoundingBox(bbox))

	return formXObject, nil
}

// ErrLayerExists is Java's `new IllegalArgumentException("Optional group
// (layer) already exists: " + layerName)`.
var ErrLayerExists = errors.New("Optional group (layer) already exists")

// AppendFormAsLayer places the given form over the existing content of the
// indicated page, like an overlay.
//
// "The form is enveloped in a marked content section to indicate that it's part
// of an optional content group (OCG), here used as a layer. This optional group
// is returned and can be enabled and disabled through methods on
// PDOptionalContentProperties."
//
// The transform "controls the placement of your form. You'll need this if your
// page has a crop box different than the media box, or if these have negative
// coordinates, or if you want to scale or adjust your form."
func (u *LayerUtility) AppendFormAsLayer(targetPage *pdmodel.PDPage,
	formXObject *form.PDFormXObject, transform *geom.AffineTransform,
	layerName string) (*optionalcontent.PDOptionalContentGroup, error) {
	catalog := u.targetDoc.DocumentCatalog()
	ocprops := catalog.OCProperties()
	if ocprops == nil {
		ocprops = optionalcontent.NewPDOptionalContentProperties()
		catalog.SetOCProperties(ocprops)
	}
	if ocprops.HasGroup(layerName) {
		return nil, fmt.Errorf("%w: %s", ErrLayerExists, layerName)
	}

	cropBox := targetPage.CropBox()
	if (cropBox.LowerLeftX() < 0 || cropBox.LowerLeftY() < 0) && transform.IsIdentity() {
		// PDFBOX-4044
		slog.Warn("Negative cropBox and identity transform may make your form invisible",
			"cropBox", cropBox)
	}

	layer := optionalcontent.NewPDOptionalContentGroup(layerName)
	ocprops.AddGroup(layer)

	contentStream, err := pdmodel.NewPDPageContentStreamCompressed(u.targetDoc, targetPage,
		pdmodel.Append, !layerUtilityDebug)
	if err != nil {
		return nil, err
	}
	if err := appendLayerContent(contentStream, layer, formXObject, transform); err != nil {
		contentStream.Close()
		return nil, err
	}
	if err := contentStream.Close(); err != nil {
		return nil, err
	}
	return layer, nil
}

// appendLayerContent is the body of Java's try-with-resources.
func appendLayerContent(contentStream *pdmodel.PDPageContentStream,
	layer *optionalcontent.PDOptionalContentGroup, formXObject *form.PDFormXObject,
	transform *geom.AffineTransform) error {
	if err := contentStream.BeginMarkedContentWithProperties(cos.OC, layer); err != nil {
		return err
	}
	if err := contentStream.SaveGraphicsState(); err != nil {
		return err
	}
	if err := contentStream.Transform(util.NewMatrixFromAffineTransform(transform)); err != nil {
		return err
	}
	if err := contentStream.DrawForm(formXObject); err != nil {
		return err
	}
	if err := contentStream.RestoreGraphicsState(); err != nil {
		return err
	}
	return contentStream.EndMarkedContent()
}

// transferDict copies the entries the filter names from one dictionary to
// another, cloning each into the target document.
func (u *LayerUtility) transferDict(orgDict, targetDict *cos.Dictionary,
	filter map[string]bool) error {
	for _, key := range orgDict.KeySet() {
		if !filter[key.Name()] {
			continue
		}
		cloned, err := u.cloner.CloneForNewDocument(orgDict.GetItem(key))
		if err != nil {
			return err
		}
		targetDict.SetItem(key, cloned)
	}
	return nil
}

// importOcProperties imports /OCProperties from the source document to the
// target document "so hidden layers can still be hidden after import".
func (u *LayerUtility) importOcProperties(srcDoc *pdmodel.PDDocument) error {
	srcCatalog := srcDoc.DocumentCatalog()
	srcOCProperties := srcCatalog.OCProperties()
	if srcOCProperties == nil {
		return nil
	}

	dstCatalog := u.targetDoc.DocumentCatalog()
	dstOCProperties := dstCatalog.OCProperties()
	if dstOCProperties != nil {
		return u.cloner.CloneMerge(srcOCProperties, dstOCProperties)
	}
	cloned, err := u.cloner.CloneDictionaryForNewDocument(
		srcOCProperties.COSObject().(*cos.Dictionary))
	if err != nil {
		return err
	}
	dstCatalog.SetOCProperties(optionalcontent.NewPDOptionalContentPropertiesOf(cloned))
	return nil
}
