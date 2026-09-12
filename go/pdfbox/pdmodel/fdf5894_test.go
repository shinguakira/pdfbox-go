package pdmodel_test

// org.apache.pdfbox.pdmodel.TestFDF.testPDFBox5894, which reads
// `target/pdfs/PDFBOX-5894.fdf`. It was deferred while that directory was
// empty; `migration/scripts/fetch-testdata.ps1` fills it.

import (
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// TestFDFPDFBox5894 is the case: an FDF whose four annotations must all be
// found by type, and each must really carry /Type /Annot.
func TestFDFPDFBox5894(t *testing.T) {
	const path = "../../../pdfbox/target/pdfs/PDFBOX-5894.fdf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("PDFBOX-5894.fdf is not there; " +
			"run migration/scripts/fetch-testdata.ps1")
	}
	document, err := pdfbox.LoadFDF(path)
	if err != nil {
		t.Fatalf("LoadFDF: %v", err)
	}
	defer document.Close()

	annotations := document.Document().ObjectsByType(cos.Annot)
	if len(annotations) != 4 {
		t.Fatalf("the FDF holds %d annotations, want 4", len(annotations))
	}
	for i, object := range annotations {
		dictionary, isDictionary := object.Object().(*cos.Dictionary)
		if !isDictionary {
			t.Errorf("annotation %d is %T, want a dictionary", i, object.Object())
			continue
		}
		if got := dictionary.GetDictionaryObject(cos.Type); got != cos.Annot {
			t.Errorf("annotation %d has /Type %v, want /Annot", i, got)
		}
	}
}
