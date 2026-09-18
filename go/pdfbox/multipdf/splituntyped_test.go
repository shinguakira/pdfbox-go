package multipdf_test

// What Splitter makes of a page tree whose /Count and whose iterator disagree.
//
// Java's processPages iterates `sourceDocument.getPages()`, and PDPageTree's
// iterator enqueues a leaf only where its /Type is /Page: a kid with no /Type at
// all is skipped with "Page skipped due to an invalid or missing type". So a tree
// whose /Count says three and whose three kids have no /Type splits into nothing
// at all, while getNumberOfPages still answers three and getPage(0) still
// answers a page. The port walked the pages by index, so it split the same
// document into three.
//
// Found by the corpus comparison of the write paths:
// itext-dotnet's PdfReaderTest/PagesDocument.pdf is written that way -- three
// page dictionaries with a /MediaBox, a /Parent and no /Type -- and PDFBox
// answers no parts for it where the port answered three.
//
// The expected values are PDFBox's, printed for this same page tree:
// getNumberOfPages=3, iterator pages=0, split parts=0, and getPage(0) answering
// the media box.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

func TestSplitAPageTreeWhoseKidsHaveNoType(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	pages := cos.NewDictionary()
	pages.SetItem(cos.Type, cos.Pages)
	pages.SetInt(cos.Count, 3)
	kids := cos.NewArray()
	for i := 0; i < 3; i++ {
		page := cos.NewDictionary()
		box := cos.NewArray()
		for _, value := range []int64{0, 0, 612, 792} {
			box.Add(cos.GetInteger(value))
		}
		page.SetItem(cos.MediaBox, box)
		page.SetItem(cos.Parent, pages)
		kids.Add(page)
	}
	pages.SetItem(cos.Kids, kids)
	document.DocumentCatalog().Dictionary().SetItem(cos.Pages, pages)

	if got := document.NumberOfPages(); got != 3 {
		t.Errorf("NumberOfPages() = %d, want PDFBox's 3", got)
	}
	iterated := 0
	for range document.Pages().All {
		iterated++
	}
	if iterated != 0 {
		t.Errorf("the page iterator handed out %d pages, want PDFBox's 0", iterated)
	}
	parts, err := multipdf.NewSplitter().Split(document)
	for _, part := range parts {
		part.Close()
	}
	if err != nil {
		t.Fatalf("Split: %v", err)
	}
	if len(parts) != 0 {
		t.Errorf("Split answered %d parts, want PDFBox's 0", len(parts))
	}

	// Last, because it writes the /Type the kid was missing: sanitizeType does
	// that in both, so a page read by index is a page to every later walk.
	if page := document.Page(0); page == nil {
		t.Error("Page(0) is nil, and PDFBox answers a page with the media box")
	} else if got := page.MediaBox(); got == nil || got.Width() != 612 {
		t.Errorf("Page(0)'s media box is %v, want 612 wide as PDFBox answers", got)
	}
}
