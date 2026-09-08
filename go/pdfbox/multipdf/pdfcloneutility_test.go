package multipdf_test

// Port of org.apache.pdfbox.multipdf.PDFCloneUtilityTest.
//
// "Test suite for PDFCloneUtility, see PDFBOX-2052." `PDFCloneUtility` was
// ported by slice 7 and its test was not: all three cases needed
// `PDPageContentStream`, `PDFMergerUtility` or `PDOptionalContentProperties`,
// and this branch brings the last of the three. The whole class goes in.

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/optionalcontent"
)

// TestClonePDFWithCosArrayStream is the "original (minimal) test from
// PDFBOX-2052": a page whose /Contents is an array of two streams clones to a
// page with the same two.
func TestClonePDFWithCosArrayStream(t *testing.T) {
	srcDoc := pdmodel.NewPDDocument()
	defer srcDoc.Close()
	dstDoc := pdmodel.NewPDDocument()
	defer dstDoc.Close()

	pdPage := pdmodel.NewPDPage()
	srcDoc.AddPage(pdPage)
	appendEmptyContentStream(t, srcDoc, pdPage)
	appendEmptyContentStream(t, srcDoc, pdPage)

	cloner := multipdf.NewPDFCloneUtility(dstDoc)
	if cloner.Destination() != dstDoc {
		t.Error("getDestination() is not the document the cloner was made for")
	}
	clonedPageDictionary, err := cloner.CloneDictionaryForNewDocument(pdPage.Dictionary())
	if err != nil {
		t.Fatalf("cloneForNewDocument: %v", err)
	}
	clonedPage := pdmodel.NewPDPageOf(clonedPageDictionary)

	// Java walks an Iterator<PDStream> and asserts two non-null elements and no
	// third; the port answers a slice, so the same statement is its length.
	contentStreams := clonedPage.ContentStreams()
	if len(contentStreams) != 2 {
		t.Fatalf("the cloned page has %d content streams, want 2", len(contentStreams))
	}
	for i, stream := range contentStreams {
		if stream == nil {
			t.Errorf("content stream %d is nil", i)
		}
	}
}

// TestClonePDFWithCosArrayStream2 is the "broader test that saves to a real PDF
// document": three appended content streams, merged into a second document,
// and both files reload with one page.
func TestClonePDFWithCosArrayStream2(t *testing.T) {
	dir := t.TempDir()
	cloneSrc := filepath.Join(dir, "clone-src.pdf")
	cloneDst := filepath.Join(dir, "clone-dst.pdf")

	srcDoc := pdmodel.NewPDDocument()
	pdPage := pdmodel.NewPDPage()
	srcDoc.AddPage(pdPage)
	for _, c := range []struct {
		colour [3]float32
		y      float32
	}{
		{[3]float32{0, 0, 0}, 600}, // Color.black
		{[3]float32{1, 0, 0}, 500}, // Color.red
		{[3]float32{1, 1, 0}, 400}, // Color.yellow
	} {
		stream, err := pdmodel.NewPDPageContentStreamCompressed(srcDoc, pdPage,
			pdmodel.Append, false)
		if err != nil {
			t.Fatalf("NewPDPageContentStream: %v", err)
		}
		if err := stream.SetNonStrokingColorRGB(c.colour[0], c.colour[1], c.colour[2]); err != nil {
			t.Fatalf("setNonStrokingColor: %v", err)
		}
		if err := stream.AddRect(100, c.y, 300, 100); err != nil {
			t.Fatalf("addRect: %v", err)
		}
		if err := stream.Fill(); err != nil {
			t.Fatalf("fill: %v", err)
		}
		if err := stream.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := srcDoc.SaveToFile(cloneSrc); err != nil {
		t.Fatalf("save: %v", err)
	}

	merger := multipdf.NewPDFMergerUtility()
	dstDoc := pdmodel.NewPDDocument()
	// this calls PDFCloneUtility.cloneForNewDocument(),
	// which would fail before the fix in PDFBOX-2052
	if err := merger.AppendDocument(dstDoc, srcDoc); err != nil {
		t.Fatalf("appendDocument: %v", err)
	}
	// save and reload PDF, so that one can see that the files are legit
	if err := dstDoc.SaveToFile(cloneDst); err != nil {
		t.Fatalf("save: %v", err)
	}
	dstDoc.Close()
	srcDoc.Close()

	// Java loads each file twice, once with a password and once without; the
	// port has one loader and the password is a separate entry point, so each
	// file is loaded once.
	for _, path := range []string{cloneSrc, cloneDst} {
		doc, err := pdfbox.LoadPDF(path)
		if err != nil {
			t.Fatalf("loading %s: %v", path, err)
		}
		if got := doc.NumberOfPages(); got != 1 {
			t.Errorf("%s has %d pages, want 1", path, got)
		}
		doc.Close()
	}
}

// TestDirectIndirect is PDFBOX-4814: "this tests merging a direct and an
// indirect COSDictionary, when 'target' is indirect in cloneMerge()".
func TestDirectIndirect(t *testing.T) {
	doc1 := pdmodel.NewPDDocument()
	defer doc1.Close()
	doc1.AddPage(pdmodel.NewPDPage())
	doc1.DocumentCatalog().SetOCProperties(optionalcontent.NewPDOptionalContentProperties())

	var saved bytes.Buffer
	if err := doc1.Save(&saved); err != nil {
		t.Fatalf("save: %v", err)
	}
	doc2, err := pdfbox.LoadPDFBytes(saved.Bytes())
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	defer doc2.Close()

	merger := multipdf.NewPDFMergerUtility()
	// The OCProperties is a direct object here, but gets saved as an indirect object.
	if _, isDictionary := doc1.DocumentCatalog().COSObject().(*cos.Dictionary).GetItem(cos.OCProperties).(*cos.Dictionary); !isDictionary {
		t.Errorf("the source /OCProperties is %T, want a COSDictionary",
			doc1.DocumentCatalog().COSObject().(*cos.Dictionary).GetItem(cos.OCProperties))
	}
	if _, isReference := doc2.DocumentCatalog().COSObject().(*cos.Dictionary).GetItem(cos.OCProperties).(*cos.Object); !isReference {
		t.Errorf("the reloaded /OCProperties is %T, want a COSObject",
			doc2.DocumentCatalog().COSObject().(*cos.Dictionary).GetItem(cos.OCProperties))
	}
	if err := merger.AppendDocument(doc2, doc1); err != nil {
		t.Fatalf("appendDocument: %v", err)
	}
	if got := doc2.NumberOfPages(); got != 2 {
		t.Errorf("the merged document has %d pages, want 2", got)
	}
}

// appendEmptyContentStream is Java's
// `new PDPageContentStream(srcDoc, pdPage, AppendMode.APPEND, true).close()`.
func appendEmptyContentStream(t *testing.T, document *pdmodel.PDDocument, page *pdmodel.PDPage) {
	t.Helper()
	stream, err := pdmodel.NewPDPageContentStreamCompressed(document, page,
		pdmodel.Append, true)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
}
