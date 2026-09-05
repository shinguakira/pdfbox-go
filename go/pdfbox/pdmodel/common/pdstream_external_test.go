package common_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/common/PDStreamTest.java and
// TestEmbeddedFiles.java. Both need a PDDocument, which sits above this
// package, and TestEmbeddedFiles also needs the Loader and
// PDEmbeddedFilesNameTreeNode.

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common/filespecification"
)

// commonFixture is the directory the two embedded file PDFs live in.
const commonFixture = "../../../../pdfbox/src/test/resources/org/apache/pdfbox/pdmodel/common/"

// dctStopFilters is the stop filter list all three PDStreamTest cases build.
//
// Java fills it with COSName.DCT_DECODE.toString(), which is "COSName{DCTDecode}",
// and createInputStream compares each filter's getName(), which is "DCTDecode".
// The two never match, so the stop list is inert -- a defect in the Java test,
// not in PDStream. The port copies the strings the Java builds rather than the
// ones it meant to; see migration/JAVA-BUGS.md.
func dctStopFilters() []string {
	return []string{cos.DCTDecode.String(), cos.DCT.String()}
}

// TestCreateInputStreamNullFilters is PDStreamTest.testCreateInputStreamNullFilters,
// the test for a null filter list (PDFBOX-2948).
func TestCreateInputStreamNullFilters(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	is := bytes.NewReader([]byte{12, 34, 56, 78})
	pdStream, err := common.NewPDStreamOfInput(doc.Document(), is, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := pdStream.Filters(); len(got) != 0 {
		t.Errorf("Filters() = %v, want none", got)
	}
	read, err := pdStream.CreateInputStreamStopping(dctStopFilters())
	if err != nil {
		t.Fatal(err)
	}
	assertBytes(t, read, []byte{12, 34, 56, 78})
}

// TestCreateInputStreamEmptyFilters is
// PDStreamTest.testCreateInputStreamEmptyFilters.
func TestCreateInputStreamEmptyFilters(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	is := bytes.NewReader([]byte{12, 34, 56, 78})
	pdStream, err := common.NewPDStreamOfInput(doc.Document(), is, cos.NewArray())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(pdStream.Filters()); got != 0 {
		t.Errorf("Filters() = %d, want 0", got)
	}
	read, err := pdStream.CreateInputStreamStopping(dctStopFilters())
	if err != nil {
		t.Fatal(err)
	}
	assertBytes(t, read, []byte{12, 34, 56, 78})
}

// TestCreateInputStreamNullStopFilters is
// PDStreamTest.testCreateInputStreamNullStopFilters.
func TestCreateInputStreamNullStopFilters(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	is := bytes.NewReader([]byte{12, 34, 56, 78})
	pdStream, err := common.NewPDStreamOfInput(doc.Document(), is, cos.NewArray())
	if err != nil {
		t.Fatal(err)
	}
	if got := len(pdStream.Filters()); got != 0 {
		t.Errorf("Filters() = %d, want 0", got)
	}
	read, err := pdStream.CreateInputStreamStopping(nil)
	if err != nil {
		t.Fatal(err)
	}
	assertBytes(t, read, []byte{12, 34, 56, 78})
}

// assertBytes is the run of `assertEquals(n, is.read())` calls the Java makes,
// ending with the -1 that says the stream is at its end.
func assertBytes(t *testing.T, read io.Reader, want []byte) {
	t.Helper()
	got, err := io.ReadAll(read)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("read = %v, want %v", got, want)
	}
}

// TestNullEmbeddedFile is TestEmbeddedFiles.testNullEmbeddedFile.
func TestNullEmbeddedFile(t *testing.T) {
	var embeddedFile *filespecification.PDEmbeddedFile
	ok := false

	doc, err := pdfbox.LoadPDF(filepath.Join(commonFixture, "null_PDComplexFileSpecification.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	catalog := doc.DocumentCatalog()
	names := catalog.Names()
	if names == nil {
		t.Fatal("Names() = nil, want the name dictionary")
	}
	embeddedFiles := names.EmbeddedFiles()
	if embeddedFiles == nil {
		t.Fatal("EmbeddedFiles() = nil, want the embedded files")
	}
	byName, err := embeddedFiles.Names()
	if err != nil {
		t.Fatal(err)
	}
	if got := len(byName); got != 2 {
		t.Errorf("expected two files, got %d", got)
	}

	if spec := byName["non-existent-file.docx"]; spec != nil {
		embeddedFile = spec.EmbeddedFile()
		ok = true
	}
	// now test for actual attachment
	spec := byName["My first attachment"]
	if spec == nil {
		t.Fatal("one attachment actually exists")
	}
	if got := spec.EmbeddedFile().Length(); got != 17660 {
		t.Errorf("existing file length = %d, want 17660", got)
	}
	spec = byName["non-existent-file.docx"]
	if spec == nil {
		t.Fatal("non-existent-file.docx = nil, want a specification")
	}
	if got := spec.File(); got != "" {
		t.Errorf("File() = %q, want the empty string, which is Java's null", got)
	}
	if got := spec.EmbeddedFile(); got != nil {
		t.Errorf("EmbeddedFile() = %v, want nil", got)
	}

	if !ok {
		t.Error("Was able to get file without exception")
	}
	if embeddedFile != nil {
		t.Error("EmbeddedFile was correctly null")
	}
}

// TestOSSpecificAttachments is TestEmbeddedFiles.testOSSpecificAttachments.
func TestOSSpecificAttachments(t *testing.T) {
	var nonOSFile, macFile, dosFile, unixFile *filespecification.PDEmbeddedFile

	doc, err := pdfbox.LoadPDF(filepath.Join(commonFixture, "testPDF_multiFormatEmbFiles.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	catalog := doc.DocumentCatalog()
	names := catalog.Names()
	if names == nil {
		t.Fatal("Names() = nil, want the name dictionary")
	}
	treeNode := names.EmbeddedFiles()
	if treeNode == nil {
		t.Fatal("EmbeddedFiles() = nil, want the embedded files")
	}
	kids := treeNode.Kids()
	if kids == nil {
		t.Fatal("Kids() = nil, want the child nodes")
	}
	for i := 0; i < kids.Size(); i++ {
		tmpNames, err := kids.Get(i).Names()
		if err != nil {
			t.Fatal(err)
		}
		spec := tmpNames["My first attachment"]
		if spec == nil {
			continue
		}
		nonOSFile = spec.EmbeddedFile()
		macFile = spec.EmbeddedFileMac()
		dosFile = spec.EmbeddedFileDos()
		unixFile = spec.EmbeddedFileUnix()
	}

	for _, want := range []struct {
		what, target string
		file         *filespecification.PDEmbeddedFile
	}{
		{"non os specific", "non os specific", nonOSFile},
		{"mac", "mac embedded", macFile},
		{"dos", "dos embedded", dosFile},
		{"unix", "unix embedded", unixFile},
	} {
		if want.file == nil {
			t.Errorf("%s: the embedded file is nil", want.what)
			continue
		}
		b, err := want.file.ToByteArray()
		if err != nil {
			t.Fatal(err)
		}
		if !byteArrayContainsLC(want.target, b) {
			t.Errorf("%s: the embedded file does not hold %q", want.what, want.target)
		}
	}
}

// byteArrayContainsLC is the private byteArrayContainsLC of the Java test,
// which reads the bytes as ISO-8859-1 and lower-cases them before looking for
// the target.
func byteArrayContainsLC(target string, b []byte) bool {
	runes := make([]rune, len(b))
	for i, c := range b {
		runes[i] = rune(c)
	}
	return strings.Contains(strings.ToLower(string(runes)), target)
}
