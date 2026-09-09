package rendering

// JAVA-BUGS 50: `PageDrawer.showAnnotation` rotates the page's graphics for a
// NoRotate annotation and puts the transform back with a plain statement
// rather than a `finally`, so an error out of the annotation leaves every
// annotation after it drawn through this one's rotation.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// TestShowAnnotationRestoresTheTransformAfterAFailure is the defect.
//
// The expected behaviour is that the page's transform is what it was: the
// rotation belongs to the one annotation that asked for it, and the drawer's
// backend is the page's, drawn on again as soon as this returns.
func TestShowAnnotationRestoresTheTransformAfterAFailure(t *testing.T) {
	document, page := pageWithContent(t, 100, 100, "")
	page.SetRotation(90)

	// an appearance stream that cannot be read: it declares a filter no
	// decoder answers to
	appearanceStream := annotation.NewPDAppearanceStream(document)
	appearanceStream.SetBBox(common.NewPDRectangleOfSize(10, 10))
	appearanceStream.COSObject().(*cos.Stream).Dictionary.SetItem(cos.Filter,
		cos.GetPDFName("NoSuchFilter"))
	appearances := annotation.NewPDAppearanceDictionary()
	appearances.SetNormalAppearance(
		annotation.NewPDAppearanceEntryOf(appearanceStream.COSObject()))

	note := annotation.NewPDAnnotationText()
	note.SetRectangle(common.NewPDRectangleOf(10, 10, 20, 20))
	note.SetNoRotate(true)
	note.SetAppearance(appearances)
	page.SetAnnotations([]annotation.PDAnnotation{note})

	drawer, err := NewPageDrawer(newPageDrawerParameters(
		NewPDFRenderer(document), page, false, Export, RenderingHints{}, 0))
	if err != nil {
		t.Fatalf("NewPageDrawer: %v", err)
	}
	backend := newRecordingBackend()
	if err := drawer.DrawPage(backend, common.NewPDRectangleOfSize(100, 100)); err == nil {
		t.Fatal("the broken appearance drew without an error, so the test proves nothing")
	}

	// the page's transform is the flip drawPage installed, which has no shear;
	// the annotation's is a quarter turn, which is all shear
	if got := backend.Transform(); got.ShearX() != 0 || got.ShearY() != 0 {
		t.Errorf("the backend transform is %v, which still carries the "+
			"annotation's rotation", got)
	}
}
