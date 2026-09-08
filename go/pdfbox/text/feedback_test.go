package text_test

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// The tests below pin the defects the slice 3 review feedback found in this
// package. Each fails without its fix.

// TestStripperByAreaIsReusable pins the first: PDFTextStripperByArea overrides
// processPage, and Java's version clears characterListMapping before walking
// the page. The port's override left it holding the first extraction, so the
// duplicate suppression treated every glyph of the second as already drawn and
// the second result came back empty.
//
// The class documents itself as reusable: extractRegions "reset[s] the stored
// text for the region so this class can be reused".
func TestStripperByAreaIsReusable(t *testing.T) {
	content := "BT /F1 12 Tf 10 300 Td (reused) Tj ET"
	region := geom.NewRectangle2D(0, 90, 300, 30)

	stripper := text.NewPDFTextStripperByArea()
	stripper.SetSortByPosition(true)
	stripper.SetLineSeparator("")
	stripper.AddRegion("region", region)

	if err := stripper.ExtractRegions(helveticaPage(t, content)); err != nil {
		t.Fatalf("first ExtractRegions: %v", err)
	}
	first := strings.TrimSpace(stripper.GetTextForRegion("region"))
	if first != "reused" {
		t.Fatalf("first extraction = %q, want %q", first, "reused")
	}

	// the same stripper, the same region, the same page again
	if err := stripper.ExtractRegions(helveticaPage(t, content)); err != nil {
		t.Fatalf("second ExtractRegions: %v", err)
	}
	second := strings.TrimSpace(stripper.GetTextForRegion("region"))
	if second != first {
		t.Errorf("second extraction = %q, want %q", second, first)
	}
}

// TestGetTextOfPagesIsReusable pins the second: Java's writeText calls
// resetEngine first, which puts currentPageNo back to 1 and empties the
// per-page state. The port left the page number where the previous call had
// pushed it, so a second call ran with every page out of the range and returned
// nothing.
func TestGetTextOfPagesIsReusable(t *testing.T) {
	document, err := pdfbox.LoadPDF(corpusDir + "PDFBOX-3025.pdf")
	if err != nil {
		t.Skipf("the Java test corpus is not present: %v", err)
	}
	defer document.Close()

	stripper := text.NewPDFTextStripper()
	first, err := stripper.GetTextOfPages(document.Pages())
	if err != nil {
		t.Fatalf("first GetTextOfPages: %v", err)
	}
	if strings.TrimSpace(first) == "" {
		t.Fatal("the first call returned nothing, so the test proves nothing")
	}

	second, err := stripper.GetTextOfPages(document.Pages())
	if err != nil {
		t.Fatalf("second GetTextOfPages: %v", err)
	}
	if second != first {
		t.Errorf("the second call returned %d bytes, the first %d; they should match",
			len(second), len(first))
	}
}

// TestGetTextOfPagesResetsThePageNumber checks the same rule directly: the page
// number is back at 1 when the walk starts.
func TestGetTextOfPagesResetsThePageNumber(t *testing.T) {
	document, err := pdfbox.LoadPDF(corpusDir + "PDFBOX-3025.pdf")
	if err != nil {
		t.Skipf("the Java test corpus is not present: %v", err)
	}
	defer document.Close()

	stripper := text.NewPDFTextStripper()
	if _, err := stripper.GetTextOfPages(document.Pages()); err != nil {
		t.Fatalf("GetTextOfPages: %v", err)
	}
	if got := stripper.CurrentPageNo(); got <= 1 {
		t.Fatalf("the walk left the page number at %d, so the test proves nothing", got)
	}

	// the second walk starts over
	if _, err := stripper.GetTextOfPages(document.Pages()); err != nil {
		t.Fatalf("second GetTextOfPages: %v", err)
	}
	// the page number ends where the first walk left it, not twice as far
	if got := stripper.CurrentPageNo(); got != document.NumberOfPages()+1 {
		t.Errorf("after two walks the page number is %d, want %d",
			got, document.NumberOfPages()+1)
	}
}
