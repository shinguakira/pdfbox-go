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
	"bytes"
	"crypto/sha256"
	"errors"
	"io"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/multipdf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
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

// TestDocumentIDDigestUsesISO88591 pins the encoding the trailer /ID digest
// feeds on. Java writes
//
//	sha256.update(Long.toString(idTime).getBytes(StandardCharsets.ISO_8859_1));
//	... sha256.update(cosBase.toString().getBytes(StandardCharsets.ISO_8859_1));
//
// so a document whose /Info holds anything outside ASCII hashes different bytes
// than a UTF-8 conversion would, and gets a different /ID. The expected digest
// here is computed from that rule, not read off the port.
func TestDocumentIDDigestUsesISO88591(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	doc.AddPage(pdmodel.NewPDPage())
	idTime := int64(123456789)
	doc.SetDocumentId(&idTime)

	// e acute is in Latin-1 and survives; the snowman is not and becomes '?',
	// which is what Java's ISO-8859-1 encoder substitutes for an unmappable
	// character.
	title := "café ☃"
	doc.DocumentInformation().Dictionary().SetItem(cos.Title, cos.NewStringObj(title))

	var out bytes.Buffer
	if err := doc.SaveOfParameters(&out, compress.NoCompression); err != nil {
		t.Fatalf("Save: %v", err)
	}
	back, err := pdfbox.LoadPDFBytes(out.Bytes())
	if err != nil {
		t.Fatalf("reloading: %v", err)
	}
	defer back.Close()

	idArray := back.Document().Trailer().GetCOSArray(cos.ID)
	if idArray == nil || idArray.Size() != 2 {
		t.Fatalf("the trailer has no two element /ID: %v", idArray)
	}
	first, ok := idArray.GetObject(0).(*cos.StringObj)
	if !ok {
		t.Fatalf("/ID[0] is %T, want a string", idArray.GetObject(0))
	}

	digest := sha256.New()
	digest.Write(testISO88591("123456789"))
	// COSString.toString() is "COSString{" + value + "}"
	digest.Write(testISO88591("COSString{" + title + "}"))
	want := digest.Sum(nil)

	if got := first.Bytes(); !bytes.Equal(got, want) {
		t.Errorf("/ID[0] = %x, want %x", got, want)
	}
}

// testISO88591 is String.getBytes(StandardCharsets.ISO_8859_1): every code
// point up to 0xFF is its own byte, and anything above it is the encoder's
// replacement, '?'.
func testISO88591(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r > 0xFF {
			out = append(out, '?')
		} else {
			out = append(out, byte(r))
		}
	}
	return out
}
