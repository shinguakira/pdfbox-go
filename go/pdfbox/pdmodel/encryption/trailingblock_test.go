package encryption_test

// A stream whose encrypted length is not a whole number of AES blocks.
//
// `javax.crypto.CipherInputStream` does not throw on one. Its contract is that
// it swallows the exception the final block raises and reports the end of the
// stream, having already written out every complete block it decrypted; Java
// therefore reads such a document and gets its content. The port raised
// "input length not a multiple of the block size" and answered nothing, so the
// page came back empty and every reader downstream saw a blank page.
//
// Found by porting PDAcroFormFlattenTest, where `test-2586.pdf` rendered
// differently before and after flattening. It was not a flattening defect: the
// file's one page content stream is 145 encrypted bytes, nine whole blocks and
// one byte over, and the port was dropping all of it.
//
// The expected values are read out of the running Java, JDK 17, through a
// driver that calls Loader.loadPDF and IOUtils.toByteArray(page.getContents()).

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
)

// trailingBlockFixture is a form whose page content stream has a trailing
// partial AES block. It is one of the documents PDAcroFormFlattenTest
// downloads; `migration/scripts/fetch-flatten.ps1` brings it down.
const trailingBlockFixture = "../../../../pdfbox/target/test-output/flatten/in/test-2586.pdf"

// TestAStreamWithATrailingPartialBlockStillDecrypts is the defect.
func TestAStreamWithATrailingPartialBlockStillDecrypts(t *testing.T) {
	if _, err := os.Stat(trailingBlockFixture); err != nil {
		t.Skip("test-2586.pdf is not there; run migration/scripts/fetch-flatten.ps1")
	}

	document, err := pdfbox.LoadPDF(trailingBlockFixture)
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	defer document.Close()

	reader, err := document.Page(0).Contents()
	if err != nil {
		t.Fatalf("Contents: %v", err)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading the page content: %v", err)
	}

	// 160 bytes, measured on the running Java.
	if len(content) != 160 {
		t.Errorf("the page content is %d bytes, want 160; a stream whose "+
			"encrypted length is not a whole number of AES blocks was being "+
			"refused instead of decrypted as far as it goes", len(content))
	}
	// And it is the page's text, not 160 bytes of anything.
	for _, want := range []string{"Accessible Combo Box", "Country: ", "/F1 12 Tf"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("the page content does not contain %q", want)
		}
	}
}
