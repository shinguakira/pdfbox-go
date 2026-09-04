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

// TestExtractBeyondTheDocumentPanics pins JAVA BUG 34. PageExtractor.extract
// promises, in its own javadoc, that "if startPage is greater than endPage or
// greater than the number of pages in the source document, a blank document
// will be returned". It does not: the clamped end page comes out below the
// start page, and Splitter.setEndPage throws IllegalArgumentException. The port
// panics, which is what an unchecked exception becomes.
func TestExtractBeyondTheDocumentPanics(t *testing.T) {
	sourcePdf, err := pdfbox.LoadPDF(inputFixture + "cweb.pdf")
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer closeDoc(sourcePdf)
	pages := sourcePdf.NumberOfPages()

	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Errorf("extracting pages %d to %d of a %d page document returned instead of"+
				" panicking", pages+2, pages+12, pages)
			return
		}
		if got, want := recovered, "End page is smaller than startPage"; got != want {
			t.Errorf("panicked with %v, want %q", got, want)
		}
	}()
	//nolint:errcheck // the panic is the assertion
	multipdf.NewPageExtractorOfRange(sourcePdf, pages+2, pages+12).Extract()
}
