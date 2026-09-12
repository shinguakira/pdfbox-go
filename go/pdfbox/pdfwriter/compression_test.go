package pdfwriter_test

// Ports of org.apache.pdfbox.pdfwriter.COSDocumentCompressionTest and
// COSWriterCompressionPoolTest.
//
// Both were deferred by slice 7 and neither deferral is still true.
// migration/STATUS.md recorded them as needing `PDAcroForm`,
// `PDComplexFileSpecification`, `PDPageContentStream`, `PDCheckBox`, `protect`,
// `PDDocumentOutline` and `PDOutlineItem`; every one of those has since been
// ported, by slices 5, 7 and 8, and nothing went back to look.
//
// What they check is that a document still says the same thing after it has
// been written out compressed -- the same pages, the same fields, the same
// attachment, the same content -- which is the one property object streams can
// quietly break.

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter/compress"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/filespecification"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/encryption"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/documentnavigation/outline"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// compressionInputs is the Java's INDIR.
const compressionInputs = "../../../pdfbox/src/test/resources/input/compression/"

// TestCompressAcroformDoc is testCompressAcroformDoc: the thirteen annotations
// of an AcroForm document, and the field name of each, after a compressed save.
//
// The names are the Java's, in the Java's order.
func TestCompressAcroformDoc(t *testing.T) {
	saved := compressAndReload(t, compressionInputs+"acroform.pdf")
	defer saved.Close()

	if got := saved.NumberOfPages(); got != 1 {
		t.Fatalf("the document has %d pages; the number should not have changed "+
			"during compression", got)
	}
	annotations := saved.Page(0).Annotations().ToSlice()
	if len(annotations) != 13 {
		t.Fatalf("the page has %d annotations; the number should not have changed",
			len(annotations))
	}

	// The field name of each annotation, or "" where the Java asserts a parent
	// instead.
	for i, want := range []string{
		"TextField", "Button", "CheckBox1", "CheckBox2",
		"TextFieldMultiLine", "TextFieldMultiLineRT",
		"", "",
		"ListBox", "ListBoxMultiSelect", "ComboBox", "ComboBoxEditable", "Signature",
	} {
		if want == "" {
			continue
		}
		got := annotations[i].COSObject().(*cos.Dictionary).GetNameAsString(cos.T, "")
		if got != want {
			t.Errorf("annotation %d is named %q, want %q", i+1, got, want)
		}
	}

	// The seventh and eighth are the two radio buttons, which carry their name
	// on a parent rather than on themselves.
	for _, i := range []int{6, 7} {
		dictionary := annotations[i].COSObject().(*cos.Dictionary)
		if dictionary.GetItem(cos.Parent) == nil {
			t.Errorf("annotation %d has no parent entry", i+1)
			continue
		}
		parent := dictionary.GetCOSDictionary(cos.Parent)
		if parent == nil {
			t.Errorf("annotation %d's parent is not a dictionary", i+1)
			continue
		}
		if got := parent.GetNameAsString(cos.T, ""); got != "GroupOption" {
			t.Errorf("annotation %d's parent is named %q, want %q", i+1, got,
				"GroupOption")
		}
	}
}

// TestCompressAttachmentsDoc is testCompressAttachmentsDoc: the attachment
// survives, under its name and at its length.
func TestCompressAttachmentsDoc(t *testing.T) {
	saved := compressAndReload(t, compressionInputs+"attachment.pdf")
	defer saved.Close()

	if got := saved.NumberOfPages(); got != 2 {
		t.Fatalf("the document has %d pages; the number should not have changed "+
			"during compression", got)
	}
	embeddedFiles, err := saved.DocumentCatalog().Names().EmbeddedFiles().Names()
	if err != nil {
		t.Fatalf("EmbeddedFiles: %v", err)
	}
	if len(embeddedFiles) != 1 {
		t.Fatalf("the document has %d attachments, want 1", len(embeddedFiles))
	}
	attachment, found := embeddedFiles["A4Unicode.pdf"]
	if !found {
		t.Fatalf("the document should have contained 'A4Unicode.pdf'; it has %v",
			keysOf(embeddedFiles))
	}
	if got := attachment.EmbeddedFile().Length(); got != 14997 {
		t.Errorf("the attachment is %d bytes, want 14997", got)
	}
}

// keysOf is for the failure message above.
func keysOf(files map[string]*filespecification.PDComplexFileSpecification) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	return names
}

// TestCompressEncryptedDoc is testCompressEncryptedDoc: a document that is
// encrypted and compressed in the same save reads back.
func TestCompressEncryptedDoc(t *testing.T) {
	target := filepath.Join(t.TempDir(), "encrypted.pdf")

	document, err := pdfbox.LoadPDFWithPassword(compressionInputs+"unencrypted.pdf", "user")
	if err != nil {
		t.Fatalf("loading the source: %v", err)
	}
	policy := encryption.NewStandardProtectionPolicy("owner", "user",
		encryption.NewAccessPermissionOf(0))
	if err := document.Protect(policy); err != nil {
		t.Fatalf("Protect: %v", err)
	}
	if err := document.SaveToFile(target); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatal(err)
	}

	// If this does not fail, the encryption dictionary is present and working.
	reloaded, err := pdfbox.LoadPDFWithPassword(target, "user")
	if err != nil {
		t.Fatalf("the encrypted document did not read back: %v", err)
	}
	defer reloaded.Close()
	if got := reloaded.NumberOfPages(); got != 2 {
		t.Errorf("the reloaded document has %d pages, want 2", got)
	}
}

// TestAlteredDoc is testAlteredDoc: a page added to an existing document is
// still there, with its content stream, after a compressed save.
func TestAlteredDoc(t *testing.T) {
	target := filepath.Join(t.TempDir(), "altered.pdf")

	document, err := pdfbox.LoadPDF(compressionInputs + "unencrypted.pdf")
	if err != nil {
		t.Fatalf("loading the source: %v", err)
	}
	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOfSize(100, 100))
	document.AddPage(page)

	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	helvetica, err := font.NewPDType1FontStandard14(font.Helvetica)
	if err != nil {
		t.Fatalf("NewPDType1FontStandard14: %v", err)
	}
	if err := stream.BeginText(); err != nil {
		t.Fatal(err)
	}
	if err := stream.NewLineAtOffset(20, 80); err != nil {
		t.Fatal(err)
	}
	if err := stream.SetFont(helvetica, 12); err != nil {
		t.Fatal(err)
	}
	if err := stream.ShowText("Test"); err != nil {
		t.Fatal(err)
	}
	if err := stream.EndText(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
	if err := document.SaveToFile(target); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := pdfbox.LoadPDF(target)
	if err != nil {
		t.Fatalf("the altered document did not read back: %v", err)
	}
	defer reloaded.Close()
	if got := reloaded.NumberOfPages(); got != 3 {
		t.Fatalf("the document has %d pages; the number should not have changed "+
			"during compression", got)
	}
	streams := reloaded.Page(2).ContentStreams()
	if len(streams) == 0 {
		t.Fatal("the page added has no content stream")
	}

	// The Java asserts `getLength() == 43`, which is the /Length entry: the
	// stream **after** it was deflated. That number is not this port's to
	// match. Measured over the identical 35 bytes of content,
	// `java.util.zip.Deflater` answers 43 and Go's `compress/zlib` answers 47,
	// at every compression level -- it is the deflate implementation, and
	// nothing about PDFBox or this port. Both are valid Flate streams and both
	// inflate to the same bytes.
	//
	// So the case asserts the content instead, which is what the Java's number
	// stands for and what a reader of the page actually sees. See
	// migration/STATUS.md.
	const want = "BT\n20 80 Td\n/F1 12 Tf\n(Test) Tj\nET\n"
	reader, err := streams[0].CreateInputStream()
	if err != nil {
		t.Fatalf("reading the new page's stream: %v", err)
	}
	content, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading the new page's stream: %v", err)
	}
	if string(content) != want {
		t.Errorf("the new page's content is %q, want %q", content, want)
	}
	if got := streams[0].Length(); got != len(content) && got <= 0 {
		t.Errorf("the new page's stream reports a length of %d", got)
	}
}

// compressAndReload saves the document at the given path with the default
// parameters, which compress, and reads the result back.
func compressAndReload(t *testing.T, path string) *pdmodel.PDDocument {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s is not in this repository: %v", path, err)
	}
	target := filepath.Join(t.TempDir(), filepath.Base(path))

	document, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatalf("loading %s: %v", path, err)
	}
	if err := document.SaveToFile(target); err != nil {
		t.Fatalf("SaveToFile: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := pdfbox.LoadPDF(target)
	if err != nil {
		t.Fatalf("the compressed document did not read back: %v", err)
	}
	return reloaded
}

// TestCompressionPoolTakesADeepOutline is
// COSWriterCompressionPoolTest.testPDFBox6036.
//
// Java's old implementation collected the objects to compress by recursion and
// overflowed its stack on a deeply nested document; the fix was to iterate.
// A Go stack grows, so the failure this guards against does not look the same
// here -- what it would do instead is run until it is killed. The case is the
// Java's either way: build an outline of up to 131,072 items and ask the pool
// to be built over it.
func TestCompressionPoolTakesADeepOutline(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a quarter of a million outline items")
	}
	for i := 1; i <= 222222; i *= 2 {
		document := pdmodel.NewPDDocument()
		documentOutline := outline.NewPDDocumentOutline()
		document.DocumentCatalog().SetDocumentOutline(documentOutline)
		for j := 0; j < i; j++ {
			documentOutline.AddLast(outline.NewPDOutlineItem())
		}
		if _, err := compress.NewCompressionPool(document,
			compress.DefaultCompression); err != nil {
			t.Fatalf("building the pool over %d outline items: %v", i, err)
		}
		if err := document.Close(); err != nil {
			t.Fatal(err)
		}
	}
}

// TestPDFBox5927 is not ported: it loads `target/pdfs/PDFBOX-5927.pdf`, which
// the Maven build downloads and this repository does not carry. See
// migration/STATUS.md.

// TestPDFBox5927 is COSDocumentCompressionTest.testPDFBox5927, which reads
// `target/pdfs/PDFBOX-5927.pdf`: a checked check box must still be checked
// after the document is written out and read back.
//
// It was deferred while that directory was empty;
// `migration/scripts/fetch-testdata.ps1` fills it.
func TestPDFBox5927(t *testing.T) {
	const path = "../../../pdfbox/target/pdfs/PDFBOX-5927.pdf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("PDFBOX-5927.pdf is not there; " +
			"run migration/scripts/fetch-testdata.ps1")
	}

	original, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatalf("LoadPDF: %v", err)
	}
	var saved bytes.Buffer
	if err := original.Save(&saved); err != nil {
		original.Close()
		t.Fatalf("Save: %v", err)
	}
	original.Close()

	reloaded, err := pdfbox.LoadPDFBytes(saved.Bytes())
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	defer reloaded.Close()

	acroForm := form.AcroFormOfCatalog(reloaded.DocumentCatalog())
	if acroForm == nil {
		t.Fatal("the reloaded document has no AcroForm")
	}
	checkBox, isCheckBox := acroForm.Field("chkPrivacy1").(*form.PDCheckBox)
	if !isCheckBox {
		t.Fatalf("chkPrivacy1 is %T, want a check box", acroForm.Field("chkPrivacy1"))
	}
	if !checkBox.IsChecked() {
		t.Error("chkPrivacy1 is not checked after the round trip")
	}
}
