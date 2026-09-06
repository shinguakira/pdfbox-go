package pdfbox

// Port of org.apache.pdfbox.pdfparser.TestPDFParser.
//
// Only one of its eighteen cases is here. The other seventeen read from
// `target/pdfs`, a directory the Maven build fills by downloading PDFs over the
// network; it does not exist in this repository and the port does not fetch
// anything in a test. They are listed in migration/STATUS.md with that reason.
//
// The one case that does not is testPDFParserMissingCatalog, whose fixture is
// checked in. It lives here rather than in `pdfparser` because it is a
// `Loader.loadPDF` case, and that is this package.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// parserFixture is where the Java pdfparser test resources are.
const parserFixture = "../../pdfbox/src/test/resources/org/apache/pdfbox/pdfparser/"

// TestPDFParserMissingCatalog is testPDFParserMissingCatalog, PDFBOX-3060: a
// document whose trailer has no /Root still opens.
//
// Java asserts only that nothing is thrown. The port asserts a little more --
// that the catalogue it built is reachable -- because a Go function that
// answers an error rather than throwing makes "did not throw" too easy to pass
// by accident.
func TestPDFParserMissingCatalog(t *testing.T) {
	document, err := LoadPDF(parserFixture + "MissingCatalog.pdf")
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer document.Close()

	catalog := document.DocumentCatalog()
	if catalog == nil {
		t.Fatal("DocumentCatalog() = nil, want a catalogue built for the " +
			"document that has none of its own")
	}
	root, ok := catalog.COSObject().(*cos.Dictionary)
	if !ok {
		t.Fatalf("the catalogue is %T, want a dictionary", catalog.COSObject())
	}
	if got := root.GetNameAsString(cos.Type, ""); got != "Catalog" {
		t.Errorf("/Type = %q, want %q", got, "Catalog")
	}
}
