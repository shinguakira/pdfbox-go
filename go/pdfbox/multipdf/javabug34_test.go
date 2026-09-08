package multipdf_test

// JAVA-BUGS 34: `PageExtractor.extract` guards only the first half of what its
// javadoc promises, so a start page past the end of the document reaches
// `Splitter.setEndPage` with a clamped end below it and throws instead of
// answering the blank document.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
)

// TestExtractBeyondTheDocumentIsBlank is the defect.
//
// The expected value is the extractor's own javadoc: "If startPage is greater
// than endPage **or greater than the number of pages in the source document**,
// a blank document will be returned". Java returns for the first of those and
// throws IllegalArgumentException from Splitter for the second, two frames
// down in a class the caller never named.
func TestExtractBeyondTheDocumentIsBlank(t *testing.T) {
	sourcePdf, err := pdfbox.LoadPDF(inputFixture + "cweb.pdf")
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer closeDoc(sourcePdf)
	pages := sourcePdf.NumberOfPages()

	result, err := multipdf.NewPageExtractorOfRange(sourcePdf, pages+2, pages+12).Extract()
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	defer closeDoc(result)
	if got := result.NumberOfPages(); got != 0 {
		t.Errorf("extracting pages %d to %d of a %d page document gave %d pages, "+
			"want the blank document the javadoc promises",
			pages+2, pages+12, pages, got)
	}
}

// TestExtractToBeyondTheDocumentStillClamps keeps the clamp the guard must not
// swallow: an end page past the last one is the case the javadoc says "will go
// to the end of the document", and it still extracts the rest of the file.
func TestExtractToBeyondTheDocumentStillClamps(t *testing.T) {
	sourcePdf, err := pdfbox.LoadPDF(inputFixture + "cweb.pdf")
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer closeDoc(sourcePdf)
	pages := sourcePdf.NumberOfPages()

	result, err := multipdf.NewPageExtractorOfRange(sourcePdf, pages-1, pages+10).Extract()
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	defer closeDoc(result)
	if got := result.NumberOfPages(); got != 2 {
		t.Errorf("extracting pages %d to %d of a %d page document gave %d pages, want 2",
			pages-1, pages+10, pages, got)
	}
}
