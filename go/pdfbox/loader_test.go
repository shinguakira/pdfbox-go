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
	"io"
	"strings"
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

// TestBaseParserStackOverflow is
// org.apache.pdfbox.pdfparser.TestBaseParser.testBaseParserStackOverflow,
// PDFBOX-6041: a document whose object graph would send a recursive descent
// round for ever.
//
// Java asserts inside a catch, so the case passes either way and what it really
// checks is that the load *returns* rather than overflowing the stack. It goes
// further only where an IOException comes back, and then the message must be
// "Missing root object specification in trailer."
//
// Go cannot recover from a stack overflow at all -- it is a fatal error, not a
// panic -- so "it returned" is the whole assertion here too, and the error, if
// there is one, must be about the missing root rather than anything else.
func TestBaseParserStackOverflow(t *testing.T) {
	document, err := LoadPDF(parserFixture + "PDFBOX-6041-example.pdf")
	if err != nil {
		if !strings.Contains(err.Error(), "root object") {
			t.Errorf("LoadPDF = %v, want either no error or one about the "+
				"missing root object", err)
		}
		return
	}
	if err := document.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestPDFBox2079EmbeddedFile is
// org.apache.pdfbox.pdfparser.EndstreamFilterStreamTest.testPDFBox2079EmbeddedFile.
//
// The fixture's stream has had its /Length removed on purpose, so the length
// has to come from the endstream filter. PDFBox 1.8.5 appended a windows
// newline to it, giving 17662 bytes where the zip is 17660 — which broke
// java.util.zip. The count is the whole point of the case.
//
// Java writes the bytes to a file and measures the file; the port measures what
// it read, which is the same number without the detour through the disk.
func TestPDFBox2079EmbeddedFile(t *testing.T) {
	document, err := LoadPDF(parserFixture + "embedded_zip.pdf")
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer document.Close()

	names := document.DocumentCatalog().Names()
	if names == nil {
		t.Fatal("Names() = nil, want the document's name dictionary")
	}
	embedded := names.EmbeddedFiles()
	if embedded == nil {
		t.Fatal("EmbeddedFiles() = nil, want the embedded files node")
	}
	byName, err := embedded.Names()
	if err != nil {
		t.Fatalf("Names: %v", err)
	}
	if len(byName) != 1 {
		t.Fatalf("the document holds %d embedded files, want 1", len(byName))
	}

	spec := byName["My first attachment"]
	if spec == nil {
		t.Fatal(`no embedded file named "My first attachment"`)
	}
	file := spec.EmbeddedFile()
	if file == nil {
		t.Fatal("EmbeddedFile() = nil, want the attachment")
	}

	reader, err := file.CreateInputStream()
	if err != nil {
		t.Fatalf("CreateInputStream: %v", err)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(content) != 17660 {
		t.Errorf("the attachment is %d bytes, want 17660: a trailing newline "+
			"the endstream filter should drop is being counted", len(content))
	}
}
