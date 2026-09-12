package form_test

// Two cases that read a PDF the Maven build downloads, and one that downloads
// its own. All three were deferred while `target/pdfs` was empty;
// `migration/scripts/fetch-testdata.ps1` fills it.

import (
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// downloadedPDFDir is Java's `target/pdfs`.
const downloadedPDFDir = "../../../../../pdfbox/target/pdfs/"

// TestPDFieldTree5044 is
// org.apache.pdfbox.pdmodel.interactive.form.PDFieldTreeTest.test5044: a form
// whose field tree holds four fields and must be walked without repeating or
// dropping any of them.
//
// Java fetches the file from the issue tracker inside the test. The port reads
// it out of `target/pdfs`, where the same attachment is saved as
// PDFBOX-4131-0.pdf, because the port does not fetch in a test.
func TestPDFieldTree5044(t *testing.T) {
	path := downloadedPDFDir + "PDFBOX-4131-0.pdf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("PDFBOX-4131-0.pdf is not there; " +
			"run migration/scripts/fetch-testdata.ps1")
	}
	document, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer document.Close()

	acroForm := form.AcroFormOfCatalog(document.DocumentCatalog())
	if acroForm == nil {
		t.Fatal("the document has no AcroForm")
	}
	count := 0
	for range acroForm.FieldTree().All() {
		count++
	}
	if count != 4 {
		t.Errorf("the field tree walks %d fields, want 4", count)
	}
}

// TestCheckBoxPDFBox6207 is
// org.apache.pdfbox.pdmodel.interactive.form.TestCheckBox.testPDFBox6207:
// setting a valid value on a check box whose /Opt is invalid must not throw.
//
// Java returns early where the file is absent rather than skipping; the port
// skips, which says the same thing in the output.
func TestCheckBoxPDFBox6207(t *testing.T) {
	path := downloadedPDFDir + "PDFBOX-6207.pdf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("PDFBOX-6207.pdf is not there; " +
			"run migration/scripts/fetch-testdata.ps1")
	}
	document, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer document.Close()

	acroForm := form.AcroFormOfCatalog(document.DocumentCatalog())
	if acroForm == nil {
		t.Fatal("the document has no AcroForm")
	}
	field, isCheckBox := acroForm.Field("Check_Info_Post_andere").(*form.PDCheckBox)
	if !isCheckBox {
		t.Fatalf("Check_Info_Post_andere is %T, want a check box",
			acroForm.Field("Check_Info_Post_andere"))
	}
	if err := field.SetValue("Yes"); err != nil {
		t.Errorf("SetValue(Yes): %v -- setting a valid value on a check box "+
			"with an invalid /Opt entry must not fail", err)
	}
}
