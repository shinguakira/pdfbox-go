package encryption_test

// The cases of org.apache.pdfbox.encryption.TestSymmetricKeyEncryption that
// read a PDF the Maven build downloads into `target/pdfs`.
//
// They were deferred while that directory was empty;
// `migration/scripts/fetch-testdata.ps1` fills it. Every expected value is the
// Java's.

import (
	"os"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// encryptionPDFDir is Java's `new File("target/pdfs", ...)`.
const encryptionPDFDir = "../../../../pdfbox/target/pdfs/"

// openEncrypted loads one of them, with a password or without, skipping where
// the fetch has not been run.
func openEncrypted(t *testing.T, name, password string) *pdmodel.PDDocument {
	t.Helper()
	path := encryptionPDFDir + name
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s is not there; run migration/scripts/fetch-testdata.ps1", name)
	}
	var document *pdmodel.PDDocument
	var err error
	if password == "" {
		document, err = pdfbox.LoadPDF(path)
	} else {
		document, err = pdfbox.LoadPDFWithPassword(path, password)
	}
	if err != nil {
		t.Fatalf("loading %s with password %q: %v", name, password, err)
	}
	t.Cleanup(func() { document.Close() })
	return document
}

// extractedText is Java's `new PDFTextStripper().getText(doc)`.
func extractedText(t *testing.T, document *pdmodel.PDDocument) string {
	t.Helper()
	stripper := text.NewPDFTextStripper()
	extracted, err := stripper.GetTextOfPages(document.Pages())
	if err != nil {
		t.Fatalf("GetTextOfPages: %v", err)
	}
	return extracted
}

// TestPDFBox5955 is testPDFBox5955: RC4 with a key length that is neither 40
// nor 128 bits. Both files open with no password and with the owner's, and the
// text has to come out either way.
func TestPDFBox5955(t *testing.T) {
	for _, c := range []struct{ file, password, want string }{
		{"PDFBOX-5955-40bit.pdf", "", "0x0446615747"},
		{"PDFBOX-5955-40bit.pdf", "ownerpass", "0x0446615747"},
		{"PDFBOX-5955-48bit.pdf", "", "0x02988E82AFF8"},
		{"PDFBOX-5955-48bit.pdf", "ownerpass", "0x02988E82AFF8"},
	} {
		document := openEncrypted(t, c.file, c.password)
		if got := extractedText(t, document); !strings.Contains(got, c.want) {
			t.Errorf("%s with password %q does not contain %q",
				c.file, c.password, c.want)
		}
	}
}

// TestPDFBox5639 is testPDFBox5639: a document that opens only with its user
// password.
func TestPDFBox5639(t *testing.T) {
	document := openEncrypted(t, "PDFBOX-5639.pdf", "JUL2023rfi")
	if got := document.NumberOfPages(); got != 2 {
		t.Errorf("NumberOfPages() = %d, want 2", got)
	}
}
