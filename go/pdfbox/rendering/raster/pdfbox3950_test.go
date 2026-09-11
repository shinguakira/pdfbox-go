package raster_test

// The rendering half of org.apache.pdfbox.pdfparser.TestPDFParser.testPDFBox3950.
//
// The parsing half is `pdfbox/testpdfparser_test.go`, TestPDFBox3950, which
// asserts the four pages. Java does both in one method; the port cannot,
// because the loader's own tests are in `package pdfbox` and reaching the
// backend from there would pull the renderer into them.
//
// The file is a truncated one with missing pages, and what Java asserts is
// that every page renders -- with one exception, written into the test: page 3
// may fail with "Missing descendant font array", and only that. Anything else,
// from any page, is a failure.

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering/raster"
)

// TestPDFBox3950Renders is the loop Java runs after the page-count assertion.
func TestPDFBox3950Renders(t *testing.T) {
	const name = "../../../../pdfbox/target/pdfs/" +
		"PDFBOX-3950-23EGDHXSBBYQLKYOKGZUOVYVNE675PRD.pdf"

	document, err := pdfbox.LoadPDF(name)
	if err != nil {
		t.Fatalf("LoadPDF: %v\n"+
			"if the file is not there, run migration/scripts/fetch-testdata.ps1", err)
	}
	defer document.Close()

	pages := document.NumberOfPages()
	if pages != 4 {
		t.Fatalf("NumberOfPages() = %d, want 4", pages)
	}

	rendered := 0
	for i := 0; i < pages; i++ {
		image, err := raster.RenderPage(document, i, 1, rendering.RGB)
		if err != nil {
			// Java's one tolerated failure, and only on this page.
			if i == 3 && strings.Contains(err.Error(), "Missing descendant font array") {
				continue
			}
			t.Errorf("rendering page %d: %v", i, err)
			continue
		}
		if image == nil {
			t.Errorf("page %d rendered to nothing", i)
			continue
		}
		if bounds := image.Bounds(); bounds.Dx() <= 0 || bounds.Dy() <= 0 {
			t.Errorf("page %d rendered to %v", i, bounds)
			continue
		}
		rendered++
	}

	// Java tolerates one failure and would pass on four; asserting that at
	// least the three undamaged pages came out keeps the case from passing on
	// a renderer that fails everywhere with the tolerated message.
	if rendered < 3 {
		t.Errorf("%d of the 4 pages rendered, want at least 3", rendered)
	}
}
