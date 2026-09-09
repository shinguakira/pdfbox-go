package multipdf_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/multipdf/PageExtractorTest.java.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

const inputFixture = "../../../pdfbox/src/test/resources/input/"

// closeDoc is the helper of the same name in the Java test: a close whose
// failure is swallowed.
func closeDoc(doc *pdmodel.PDDocument) {
	if doc != nil {
		_ = doc.Close() /* Can't do much about this... */
	}
}

// TestExtract is testExtract: test of extract method, of class
// org.apache.pdfbox.util.PageExtractor.
func TestExtract(t *testing.T) {
	// this should work for most users
	sourcePdf, err := pdfbox.LoadPDF(inputFixture + "cweb.pdf")
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer closeDoc(sourcePdf)

	extract := func(instance *multipdf.PageExtractor) *pdmodel.PDDocument {
		t.Helper()
		result, err := instance.Extract()
		if err != nil {
			t.Fatalf("Extract: %v", err)
		}
		return result
	}
	assertPages := func(want int, result *pdmodel.PDDocument) {
		t.Helper()
		if got := result.NumberOfPages(); got != want {
			t.Errorf("NumberOfPages() = %d, want %d", got, want)
		}
	}

	instance := multipdf.NewPageExtractor(sourcePdf)
	result := extract(instance)
	assertPages(sourcePdf.NumberOfPages(), result)
	closeDoc(result)

	instance = multipdf.NewPageExtractorOfRange(sourcePdf, 1, 1)
	result = extract(instance)
	assertPages(1, result)
	closeDoc(result)

	instance = multipdf.NewPageExtractorOfRange(sourcePdf, 1, 5)
	result = extract(instance)
	assertPages(5, result)
	closeDoc(result)

	instance = multipdf.NewPageExtractorOfRange(sourcePdf, 5, 10)
	result = extract(instance)
	assertPages(6, result)
	closeDoc(result)

	instance = multipdf.NewPageExtractorOfRange(sourcePdf, 2, 1)
	result = extract(instance)
	assertPages(0, result)
	closeDoc(result)
}
