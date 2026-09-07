package pdfbox

// Port of org.apache.pdfbox.cos.TestCOSIncrement.
//
// The Java class is in `cos`, but every step of it is Loader plus PDDocument,
// so it sits here with the other whole-document cases rather than in
// `pdfbox/cos`. One of its three cases is not ported:
//
//   - testConcurrentModification downloads a PDF from issues.apache.org, and
//     the port does not fetch anything in a test.
//
// That is recorded in migration/STATUS.md. testSubsetting was deferred with it
// while PDType0Font.load was unported; track/font-embedding ported the load,
// and the case is here.

import (
	"bytes"
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/image"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
)

// cosFixture is where the Java cos test resources are.
const cosFixture = "../../pdfbox/src/test/resources/org/apache/pdfbox/cos/"

// loadDocumentBytes is the Java helper loadDocument.
func loadDocumentBytes(t *testing.T, data []byte) *pdmodel.PDDocument {
	t.Helper()
	document, err := LoadPDFBytes(data)
	if err != nil {
		t.Fatalf("Loading the document failed: %v", err)
	}
	return document
}

// saveIncremental writes the document out incrementally and hands back the
// bytes, which is what each step of the Java case ends with.
func saveIncremental(t *testing.T, document *pdmodel.PDDocument) []byte {
	t.Helper()
	var out bytes.Buffer
	if err := document.SaveIncremental(&out); err != nil {
		t.Fatalf("SaveIncremental: %v", err)
	}
	return out.Bytes()
}

// TestIncrementallyCreateDocument is testIncrementallyCreateDocument: build a
// document one incremental save at a time, checking after each that the
// previous step survived the round trip.
func TestIncrementallyCreateDocument(t *testing.T) {
	var documentData []byte

	// Add page 1.
	func() {
		document := pdmodel.NewPDDocument()
		defer document.Close()
		document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 100)))
		var out bytes.Buffer
		if err := document.Save(&out); err != nil {
			t.Fatalf("Save: %v", err)
		}
		documentData = out.Bytes()
	}()

	// Add page 2 and 3.
	func() {
		document := loadDocumentBytes(t, documentData)
		defer document.Close()
		if got := document.NumberOfPages(); got != 1 {
			t.Fatalf("Document should have contained 1 page, has %d", got)
		}
		document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(200, 200)))
		document.AddPage(pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 100)))
		documentData = saveIncremental(t, document)
	}()

	// Remove page 2.
	func() {
		document := loadDocumentBytes(t, documentData)
		defer document.Close()
		if got := document.NumberOfPages(); got != 3 {
			t.Fatalf("Document should have contained 3 pages, has %d", got)
		}
		document.RemovePage(document.Page(1))
		documentData = saveIncremental(t, document)
	}()

	// Add an image to page 1.
	func() {
		document := loadDocumentBytes(t, documentData)
		defer document.Close()
		if got := document.Page(1).MediaBox().Width(); got == 200 {
			t.Fatal("Page 2 removal failed")
		}
		if got := document.NumberOfPages(); got != 2 {
			t.Fatalf("Document should have contained 2 pages, has %d", got)
		}
		if document.Page(0).HasContents() {
			t.Error("Page 1 should not have had contents")
		}
		if document.Page(0).Resources() != nil {
			t.Error("Page 1 should not have contained resources")
		}
		if document.Page(1).HasContents() {
			t.Error("Page 2 should not have had contents")
		}
		if document.Page(1).Resources() != nil {
			t.Error("Page 2 should not have contained resources")
		}

		contentStream, err := pdmodel.NewPDPageContentStream(document, document.Page(0))
		if err != nil {
			t.Fatalf("NewPDPageContentStream: %v", err)
		}
		picture, err := image.CreateFromFileByExtension(document, cosFixture+"simple.png")
		if err != nil {
			t.Fatalf("CreateFromFileByExtension: %v", err)
		}
		if err := contentStream.DrawImage(picture, 15, 20); err != nil {
			t.Fatalf("DrawImage: %v", err)
		}
		if err := contentStream.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		documentData = saveIncremental(t, document)
	}()

	// Write a text to page 2.
	func() {
		document := loadDocumentBytes(t, documentData)
		defer document.Close()
		if !document.Page(0).HasContents() {
			t.Error("Page 1 should have had contents")
		}
		if document.Page(0).Resources() == nil {
			t.Fatal("Page 1 should have contained resources")
		}
		if len(document.Page(0).Resources().FontNames()) != 0 {
			t.Error("Page 1 should not have contained a font")
		}
		if len(document.Page(0).Resources().XObjectNames()) == 0 {
			t.Error("Page 1 should have contained an XObject")
		}
		if document.Page(1).HasContents() {
			t.Error("Page 2 should not have had contents")
		}
		if document.Page(1).Resources() != nil {
			t.Error("Page 2 should not have contained resources")
		}

		contentStream, err := pdmodel.NewPDPageContentStream(document, document.Page(1))
		if err != nil {
			t.Fatalf("NewPDPageContentStream: %v", err)
		}
		if err := contentStream.BeginText(); err != nil {
			t.Fatalf("BeginText: %v", err)
		}
		if err := contentStream.SetFont(helvetica(t), 20); err != nil {
			t.Fatalf("SetFont: %v", err)
		}
		if err := contentStream.NewLineAtOffset(20, 50); err != nil {
			t.Fatalf("NewLineAtOffset: %v", err)
		}
		if err := contentStream.ShowText("Page 2"); err != nil {
			t.Fatalf("ShowText: %v", err)
		}
		if err := contentStream.EndText(); err != nil {
			t.Fatalf("EndText: %v", err)
		}
		if err := contentStream.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		documentData = saveIncremental(t, document)
	}()

	// Add an annotation to page 2.
	func() {
		document := loadDocumentBytes(t, documentData)
		defer document.Close()
		if !document.Page(0).HasContents() {
			t.Error("Page 1 should have had contents")
		}
		if document.Page(0).Resources() == nil {
			t.Error("Page 1 should have contained resources")
		}
		if document.Page(1).Resources() == nil {
			t.Fatal("Page 2 should have contained resources")
		}
		if got := document.Page(1).Annotations().Size(); got != 0 {
			t.Errorf("Page 2 should not have contained an annotation, has %d", got)
		}
		if !document.Page(1).HasContents() {
			t.Error("Page 2 should have had contents")
		}
		if len(document.Page(1).Resources().FontNames()) == 0 {
			t.Error("Page 2 should have contained a font")
		}
		if len(document.Page(1).Resources().XObjectNames()) != 0 {
			t.Error("Page 2 should not have contained an XObject")
		}

		textAnnotation := annotation.NewPDAnnotationText()
		textAnnotation.SetAnnotationName("text annotation")
		textAnnotation.SetContents("text annotation")
		textAnnotation.SetOpen(true)
		textAnnotation.SetColor(color.NewPDColorOfComponents(
			[]float32{1, 0, 0}, color.DeviceRGB))
		textAnnotation.SetRectangle(common.NewPDRectangleOf(4, 5, 10, 10))
		if err := textAnnotation.ConstructAppearancesInDocument(document); err != nil {
			t.Fatalf("ConstructAppearances: %v", err)
		}
		document.Page(1).SetAnnotations([]annotation.PDAnnotation{textAnnotation})
		documentData = saveIncremental(t, document)
	}()

	// Do nothing.
	func() {
		document := loadDocumentBytes(t, documentData)
		defer document.Close()
		if got := document.Page(1).Annotations().Size(); got != 1 {
			t.Errorf("Page 2 should have contained an annotation, has %d", got)
		}
		documentData = saveIncremental(t, document)
	}()

	// Check the result.
	document := loadDocumentBytes(t, documentData)
	defer document.Close()
	if got := document.NumberOfPages(); got != 2 {
		t.Fatalf("Document should have contained 2 pages, has %d", got)
	}
	if document.Page(0).Resources() == nil {
		t.Fatal("Page 1 should have contained resources")
	}
	if document.Page(1).Resources() == nil {
		t.Fatal("Page 2 should have contained resources")
	}
	if !document.Page(0).HasContents() {
		t.Error("Page 1 should have had contents")
	}
	if len(document.Page(0).Resources().FontNames()) != 0 {
		t.Error("Page 1 should not have contained a font")
	}
	if len(document.Page(0).Resources().XObjectNames()) == 0 {
		t.Error("Page 1 should have contained an XObject")
	}
	if !document.Page(1).HasContents() {
		t.Error("Page 2 should have had contents")
	}
	if got := document.Page(1).Annotations().Size(); got != 1 {
		t.Errorf("Page 2 should have contained an annotation, has %d", got)
	}
	if len(document.Page(1).Resources().FontNames()) == 0 {
		t.Error("Page 2 should have contained a font")
	}
}

// helvetica is `new PDType1Font(Standard14Fonts.FontName.HELVETICA)`, which in
// Go reports whether the standard font could be built.
func helvetica(t *testing.T) *font.PDType1Font {
	t.Helper()
	f, err := font.NewPDType1FontStandard14(font.Helvetica)
	if err != nil {
		t.Fatalf("NewPDType1FontStandard14: %v", err)
	}
	return f
}

// TestSubsetting is testSubsetting: PDFBOX-5627. A subsetted font is added to a
// document that already exists on disk, and the incremental save has to carry
// the font program with it — the subset is only built when the document is
// saved, so an incremental save that ran the subsetter too late, or not at all,
// writes a font dictionary whose /FontFile2 is missing or empty.
//
// Java writes the increment to target/test-output/PDFBOX-5627.pdf and reloads
// it from there; the port keeps it in memory, as its other cases do.
func TestSubsetting(t *testing.T) {
	var created bytes.Buffer
	func() {
		document := pdmodel.NewPDDocument()
		defer document.Close()

		page := pdmodel.NewPDPageOfSize(common.A4)
		document.AddPage(page)
		if err := document.Save(&created); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}()

	var incremented []byte
	func() {
		document := loadDocumentBytes(t, created.Bytes())
		defer document.Close()

		page := document.Page(0)

		file, err := os.Open("../../pdfbox/src/main/resources/org/apache/pdfbox/resources/" +
			"ttf/LiberationSans-Regular.ttf")
		if err != nil {
			t.Fatalf("opening LiberationSans-Regular.ttf: %v", err)
		}
		defer file.Close()

		embedded, err := font.LoadPDType0Font(document, file)
		if err != nil {
			t.Fatalf("LoadPDType0Font: %v", err)
		}

		func() {
			contentStream, err := pdmodel.NewPDPageContentStream(document, page)
			if err != nil {
				t.Fatalf("NewPDPageContentStream: %v", err)
			}
			if err := contentStream.BeginText(); err != nil {
				t.Fatalf("BeginText: %v", err)
			}
			if err := contentStream.SetFont(embedded, 12); err != nil {
				t.Fatalf("SetFont: %v", err)
			}
			if err := contentStream.NewLineAtOffset(75, 750); err != nil {
				t.Fatalf("NewLineAtOffset: %v", err)
			}
			if err := contentStream.ShowText("Apache PDFBox"); err != nil {
				t.Fatalf("ShowText: %v", err)
			}
			if err := contentStream.EndText(); err != nil {
				t.Fatalf("EndText: %v", err)
			}
			if err := contentStream.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
		}()

		catalog, ok := document.DocumentCatalog().COSObject().(*cos.Dictionary)
		if !ok {
			t.Fatalf("the catalog is %T, want *cos.Dictionary", document.DocumentCatalog().COSObject())
		}
		catalog.SetNeedToBeUpdated(true)
		pages := catalog.GetCOSDictionary(cos.Pages)
		pages.SetNeedToBeUpdated(true)
		page.Dictionary().SetNeedToBeUpdated(true)

		incremented = saveIncremental(t, document)
	}()

	document := loadDocumentBytes(t, incremented)
	defer document.Close()

	page := document.Page(0)
	names := page.Resources().FontNames()
	if len(names) == 0 {
		t.Fatalf("the reloaded page has no font")
	}
	reloaded, err := page.Resources().GetFont(names[0])
	if err != nil {
		t.Fatalf("GetFont(%v): %v", names[0], err)
	}
	if !reloaded.IsEmbedded() {
		t.Errorf("%v is not embedded; the incremental save dropped the font program", names[0])
	}
}
