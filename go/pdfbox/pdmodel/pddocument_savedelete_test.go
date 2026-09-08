package pdmodel_test

// Port of org.apache.pdfbox.pdmodel.TestPDDocument, "introduced with
// PDFBOX-1581".
//
// It was never ported and never recorded as deferred -- it is not in
// migration/STATUS.md under any reason, and no slice's task file names it. It
// was missed. Six cases, and every one of them is about the two operations a
// document is for: saving it and reading it back.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// TestSaveLoadStream is testSaveLoadStream: a one-page document saved
// uncompressed to a stream carries a 1.4 header and an %%EOF, and reads back as
// one page.
func TestSaveLoadStream(t *testing.T) {
	var saved bytes.Buffer
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPage())
	if err := document.SaveOfParameters(&saved, compress.NoCompression); err != nil {
		t.Fatalf("SaveOfParameters: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	checkSavedDocument(t, saved.Bytes(), "%PDF-1.4")
}

// TestSaveLoadFile is testSaveLoadFile, which is the same through a file.
func TestSaveLoadFile(t *testing.T) {
	target := filepath.Join(t.TempDir(), "pddocument-saveloadfile.pdf")

	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPage())
	if err := document.SaveToFileOfParameters(target, compress.NoCompression); err != nil {
		t.Fatalf("SaveToFileOfParameters: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	saved, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading the saved file: %v", err)
	}
	checkSavedDocument(t, saved, "%PDF-1.4")

	// The Java asserts the file's length as well as the bytes it read.
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() <= 200 {
		t.Errorf("the saved file is %d bytes, want more than 200", info.Size())
	}
}

// checkSavedDocument is the block both save cases repeat: the length, the
// header, the trailer, and one page when it is read back.
func checkSavedDocument(t *testing.T, saved []byte, header string) {
	t.Helper()
	if len(saved) <= 200 {
		t.Errorf("the document is %d bytes, want more than 200", len(saved))
	}
	if got := string(saved[:8]); got != header {
		t.Errorf("the header is %q, want %q", got, header)
	}
	if got := string(saved[len(saved)-6:]); got != "%%EOF\n" {
		t.Errorf("the document ends %q, want %q", got, "%%EOF\n")
	}

	reloaded, err := pdfbox.LoadPDFBytes(saved)
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	defer reloaded.Close()
	if got := reloaded.NumberOfPages(); got != 1 {
		t.Errorf("the reloaded document has %d pages, want 1", got)
	}
}

// TestVersions is testVersions: what a new document's version is, that it
// cannot be moved down, that it can be moved up, and that saving with the
// default compression takes everything to 1.6.
func TestVersions(t *testing.T) {
	// The default version.
	document := pdmodel.NewPDDocument()
	if got := document.Version(); got != 1.4 {
		t.Errorf("a new document is version %v, want 1.4", got)
	}
	if got := document.Document().Version(); got != 1.4 {
		t.Errorf("the COS document is version %v, want 1.4", got)
	}
	if got := document.DocumentCatalog().Version(); got != "1.4" {
		t.Errorf("the catalog is version %q, want %q", got, "1.4")
	}
	// Forcing the header down, which the catalog does not stop.
	document.Document().SetVersion(1.3)
	document.DocumentCatalog().SetVersion("")
	if got := document.Version(); got != 1.3 {
		t.Errorf("after forcing the header down the document is version %v, want 1.3", got)
	}
	if got := document.Document().Version(); got != 1.3 {
		t.Errorf("the COS document is version %v, want 1.3", got)
	}
	if got := document.DocumentCatalog().Version(); got != "" {
		t.Errorf("the catalog version is %q, want it cleared", got)
	}
	if err := document.Close(); err != nil {
		t.Fatal(err)
	}

	// setVersion refuses to move the version down and accepts moving it up.
	document = pdmodel.NewPDDocument()
	document.SetVersion(1.3)
	if got := document.Version(); got != 1.4 {
		t.Errorf("setVersion(1.3) left the document at %v, want 1.4: a version "+
			"is not moved down", got)
	}
	if got := document.Document().Version(); got != 1.4 {
		t.Errorf("the COS document is version %v, want 1.4", got)
	}
	if got := document.DocumentCatalog().Version(); got != "1.4" {
		t.Errorf("the catalog is version %q, want %q", got, "1.4")
	}
	document.SetVersion(1.5)
	if got := document.Version(); got != 1.5 {
		t.Errorf("setVersion(1.5) left the document at %v, want 1.5", got)
	}
	if got := document.Document().Version(); got != 1.4 {
		t.Errorf("the header moved to %v; setVersion changes the catalog, not "+
			"the header", got)
	}
	if got := document.DocumentCatalog().Version(); got != "1.5" {
		t.Errorf("the catalog is version %q, want %q", got, "1.5")
	}
	if err := document.Close(); err != nil {
		t.Fatal(err)
	}

	// PDFBOX-5265: saving with the default parameters compresses, and every
	// version then reads back as 1.6.
	var saved bytes.Buffer
	document = pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPage())
	if err := document.Save(&saved); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := pdfbox.LoadPDFBytes(saved.Bytes())
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	defer reloaded.Close()
	if got := reloaded.DocumentCatalog().Version(); got != "1.6" {
		t.Errorf("the catalog of a compressed document is version %q, want %q",
			got, "1.6")
	}
	if got := reloaded.Document().Version(); got != 1.6 {
		t.Errorf("the COS document is version %v, want 1.6", got)
	}
	if got := reloaded.Version(); got != 1.6 {
		t.Errorf("the document is version %v, want 1.6", got)
	}
	if got := string(saved.Bytes()[:8]); got != "%PDF-1.6" {
		t.Errorf("the header is %q, want %q", got, "%PDF-1.6")
	}
}

// TestDeleteBadFile is testDeleteBadFile: a file that fails to parse is not
// left open, so it can be deleted afterwards.
//
// On Windows this is the case that matters. A file with an open handle cannot
// be removed at all there, where a Unix unlink would succeed and hide the leak.
func TestDeleteBadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testDeleteBadFile.pdf")
	if err := os.WriteFile(path, []byte("<script language='JavaScript'>"), 0o644); err != nil {
		t.Fatalf("writing the bad file: %v", err)
	}

	if _, err := pdfbox.LoadPDF(path); err == nil {
		t.Fatal("parsing should fail")
	}
	if err := os.Remove(path); err != nil {
		t.Errorf("delete bad file failed after failed load: %v", err)
	}
}

// TestDeleteGoodFile is testDeleteGoodFile: a file that parses and is closed is
// not left open either.
func TestDeleteGoodFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "testDeleteGoodFile.pdf")

	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPage())
	if err := document.SaveToFile(path); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatal(err)
	}

	loaded, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	if err := loaded.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Errorf("delete good file failed after successful load and close: %v", err)
	}
}

// TestSaveWritesAsciiDigits is what testSaveArabicLocale asserts, in the shape
// this port can assert it.
//
// PDFBOX-3481 is that a cross-reference table written under an Arabic locale
// came out with Arabic-Indic digits, which nothing can parse. Java's
// `String.format` and `NumberFormat` follow `Locale.getDefault()`, so the Java
// test sets the locale to `ar-EG-u-nu-arab` and saves.
//
// **Go has no default locale**, and `strconv` and `fmt`'s `%d` write ASCII
// digits whatever the environment says; there is no setting the case could
// change to make the port fail. What is left of it is the property itself,
// asserted directly: every byte of the cross-reference section is a digit or a
// separator that a parser accepts. See migration/STATUS.md.
func TestSaveWritesAsciiDigits(t *testing.T) {
	var saved bytes.Buffer
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPage())
	if err := document.SaveOfParameters(&saved, compress.NoCompression); err != nil {
		t.Fatalf("SaveOfParameters: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatal(err)
	}

	contents := saved.String()
	start := strings.Index(contents, "xref")
	if start < 0 {
		t.Fatal("the saved document has no cross-reference table")
	}
	end := strings.Index(contents[start:], "trailer")
	if end < 0 {
		t.Fatal("the cross-reference table has no trailer after it")
	}
	for i, b := range []byte(contents[start : start+end]) {
		if b > 0x7F {
			t.Fatalf("byte %d of the cross-reference table is %#x, which is not "+
				"ASCII: a digit was written in some other numbering system",
				start+i, b)
		}
	}

	reloaded, err := pdfbox.LoadPDFBytes(saved.Bytes())
	if err != nil {
		t.Fatalf("the document did not read back: %v", err)
	}
	if err := reloaded.Close(); err != nil {
		t.Fatal(err)
	}
}
