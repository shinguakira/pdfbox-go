package filter_test

// org.apache.pdfbox.filter.TestFilters.testPDFBOX4517.
//
// It is here rather than in `testfilters_test.go` because it loads a document,
// and `pdfbox` imports `filter`, so the internal test package cannot reach it.
//
// It reads `target/pdfs/PDFBOX-4517-cryptfilter.pdf`, which the Maven build
// downloads; `migration/scripts/fetch-testdata.ps1` fills that directory.

import (
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
)

// TestPDFBOX4517 is a document whose crypt filter the parser has to follow to
// reach its one page.
func TestPDFBOX4517(t *testing.T) {
	const path = "../../../pdfbox/target/pdfs/PDFBOX-4517-cryptfilter.pdf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("PDFBOX-4517-cryptfilter.pdf is not there; " +
			"run migration/scripts/fetch-testdata.ps1")
	}
	document, err := pdfbox.LoadPDFWithPassword(path, "userpassword1234")
	if err != nil {
		t.Fatalf("LoadPDFWithPassword: %v", err)
	}
	defer document.Close()
	if got := document.NumberOfPages(); got != 1 {
		t.Errorf("NumberOfPages() = %d, want 1", got)
	}
}
