package pdfwriter_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdfwriter/COSWriterTest.java.
//
// An external test package: this drives PDDocument.save, and pdmodel imports
// pdfwriter, so a test file in package pdfwriter could not reach it.
//
// Two of the four Java tests are here. The other two are recorded in
// migration/STATUS.md:
//   - testPDFBox5945 builds an AcroForm out of PDAcroForm, PDTextField and
//     PDAnnotationWidget, which slice 8 brings.
//   - testPDFBox6036 downloads two PDFs from issues.apache.org. The port's
//     tests do not reach the network.

import (
	"errors"
	"io"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

const inputFixture = "../../../pdfbox/src/test/resources/input/"

// closeRecordingWriter stands for the ByteArrayOutputStream of testPDFBox4321,
// whose close() throws. Go's io.Writer has no Close, so where Java asserts that
// the exception never surfaces this asserts that Close is never reached.
type closeRecordingWriter struct {
	written int
	closed  bool
}

func (w *closeRecordingWriter) Write(b []byte) (int, error) {
	w.written += len(b)
	return len(b), nil
}

func (w *closeRecordingWriter) Close() error {
	w.closed = true
	return errors.New("Stream was closed")
}

// TestPDFBox4321 is testPDFBox4321: check whether the output stream is closed
// after saving.
func TestPDFBox4321(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	page := pdmodel.NewPDPage()
	doc.AddPage(page)

	out := &closeRecordingWriter{}
	if err := doc.Save(out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if out.closed {
		t.Error("Save closed the stream it was given")
	}
	if out.written == 0 {
		t.Error("Save wrote nothing")
	}
}

// TestPDFBox5485 is testPDFBox5485.
func TestPDFBox5485(t *testing.T) {
	pdfDocument, err := pdfbox.LoadPDF(inputFixture + "PDFBOX-3110-poems-beads.pdf")
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer pdfDocument.Close()

	pageExtractor := multipdf.NewPageExtractorOfRange(pdfDocument, 2, 2)
	pdfPages, err := pageExtractor.Extract()
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	defer pdfPages.Close()

	if err := pdfPages.Save(io.Discard); err != nil {
		t.Fatalf("Save: %v", err)
	}
}
