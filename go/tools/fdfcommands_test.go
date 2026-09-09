package tools_test

// The four FDF commands and ExtractXMP.
//
// **None has a Java test.** A2 asks for a decision either way, and these are
// worth one for a reason the others were not: two of the four differ from their
// twin in a way that is a defect rather than a design, and a test is what keeps
// the port carrying the difference rather than quietly tidying it away.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fdf"
	"github.com/shinguakira/pdfbox-go/go/tools"
)

// aFormDocument writes a document with an AcroForm and one text field, and
// answers its path.
func aFormDocument(t *testing.T) string {
	t.Helper()
	const fixture = "../../pdfbox/src/test/resources/org/apache/pdfbox/pdmodel/interactive/form/"
	for _, name := range []string{
		"AcroFormsBasicFields.pdf",
		"SimpleForm.pdf",
	} {
		if _, err := os.Stat(fixture + name); err == nil {
			return fixture + name
		}
	}
	t.Skip("no AcroForm fixture in this repository")
	return ""
}

// TestExportFDFRefusesADocumentWithNoForm is the branch entry 75 is about, on
// the side that gets it right.
func TestExportFDFRefusesADocumentWithNoForm(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.fdf")
	code, _, stderr := runCommand(tools.NewExportFDF(), "-i", testFile2, "-o", out)
	if code != 1 {
		t.Errorf("exited %d, want 1", code)
	}
	if want := "Error: This PDF does not contain a form."; !strings.Contains(stderr, want) {
		t.Errorf("stderr is %q, want %q", stderr, want)
	}
}

// TestExportXFDFRefusesADocumentWithNoForm is JAVA-BUGS 75: the same
// condition and the same message as its twin above, and Java exits 0 for it,
// so a script that checks the exit code is told the export succeeded while no
// file was written. The two commands differ only in the format the caller
// asked for, so they answer the same thing here.
func TestExportXFDFRefusesADocumentWithNoForm(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.xfdf")
	code, _, stderr := runCommand(tools.NewExportXFDF(), "-i", testFile2, "-o", out)
	if code != 1 {
		t.Errorf("exited %d, want 1 -- the same as exportfdf", code)
	}
	if want := "Error: This PDF does not contain a form."; !strings.Contains(stderr, want) {
		t.Errorf("stderr is %q, want %q", stderr, want)
	}
	if _, err := os.Stat(out); err == nil {
		t.Error("the file was written; the command reported success without writing one")
	}
}

// TestImportFDFOnADocumentWithNoFormSavesItUnchanged is the branch entry 76 is
// about, on the side that gets it right: the null check returns early and the
// document is saved as it was.
func TestImportFDFOnADocumentWithNoFormSavesItUnchanged(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "empty.fdf")
	writeEmptyFDF(t, data)
	out := filepath.Join(dir, "out.pdf")

	code, _, stderr := runCommand(tools.NewImportFDF(),
		"-i", testFile2, "--data", data, "-o", out)
	if code != 0 {
		t.Fatalf("exited %d (%s), want 0", code, stderr)
	}
	document, err := pdfbox.LoadPDF(out)
	if err != nil {
		t.Fatalf("the output is not a PDF: %v", err)
	}
	document.Close()
}

// TestImportXFDFOnADocumentWithNoFormPanics is JAVA-BUGS 76: the twin has no
// null check, so the same input dies rather than reporting anything.
//
// Java raises NullPointerException, which its catch(IOException) does not take;
// the port panics, and Execute answers ExitCode.SOFTWARE for it as picocli's
// execution exception handler does.
func TestImportXFDFOnADocumentWithNoFormPanics(t *testing.T) {
	dir := t.TempDir()
	data := filepath.Join(dir, "empty.xfdf")
	if err := os.WriteFile(data, []byte(
		`<?xml version="1.0" encoding="UTF-8"?>`+"\n"+
			`<xfdf xmlns="http://ns.adobe.com/xfdf/"><fields/></xfdf>`), 0o666); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := runCommand(tools.NewImportXFDF(),
		"-i", testFile2, "--data", data, "-o", filepath.Join(dir, "out.pdf"))
	if code != 1 {
		t.Errorf("exited %d, want 1 -- see JAVA-BUGS 76", code)
	}
	if stdout != "" {
		t.Errorf("the failure went to stdout as %q", stdout)
	}
}

// writeEmptyFDF writes the smallest FDF document that loads.
func writeEmptyFDF(t *testing.T, path string) {
	t.Helper()
	document := fdf.NewFDFDocument()
	defer document.Close()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := document.Save(file); err != nil {
		t.Fatal(err)
	}
}

// TestExtractXMPRefusesAPageThatIsNotThere is the one branch of ExtractXMP that
// is the command's own rather than the library's.
func TestExtractXMPRefusesAPageThatIsNotThere(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.xml")
	code, _, stderr := runCommand(tools.NewExtractXMP(),
		"-i", testFile2, "-o", out, "-page", "99")
	if code != 1 {
		t.Errorf("exited %d, want 1", code)
	}
	if want := "Page 99 doesn't exist"; !strings.Contains(stderr, want) {
		t.Errorf("stderr is %q, want %q", stderr, want)
	}
}

// TestExtractXMPReportsNoMetadata is the second: a document with no /Metadata.
//
// Every PDF fixture in the tools module carries XMP, so the document is built
// here rather than read.
func TestExtractXMPReportsNoMetadata(t *testing.T) {
	dir := t.TempDir()
	bare := filepath.Join(dir, "bare.pdf")
	document := pdmodel.NewPDDocument()
	document.AddPage(pdmodel.NewPDPage())
	if err := document.SaveToFile(bare); err != nil {
		t.Fatal(err)
	}
	document.Close()

	out := filepath.Join(dir, "out.xml")
	code, _, stderr := runCommand(tools.NewExtractXMP(), "-i", bare, "-o", out)
	if code != 1 {
		t.Errorf("exited %d, want 1", code)
	}
	if want := "No XMP metadata available"; !strings.Contains(stderr, want) {
		t.Errorf("stderr is %q, want %q", stderr, want)
	}
}

// TestExtractXMPWritesTheMetadata is the path that works: the fixtures of this
// module all carry XMP, so the stream comes back as XML.
func TestExtractXMPWritesTheMetadata(t *testing.T) {
	out := filepath.Join(t.TempDir(), "out.xml")
	code, _, stderr := runCommand(tools.NewExtractXMP(), "-i", testFile2, "-o", out)
	if code != 0 {
		t.Fatalf("exited %d (%s), want 0", code, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("the file was not written: %v", err)
	}
	if !strings.Contains(string(data), "<x:xmpmeta") && !strings.Contains(string(data), "<?xpacket") {
		t.Errorf("the file does not look like XMP:\n%.200s", data)
	}

	// -console writes the same bytes to stdout instead.
	code, stdout, stderr := runCommand(tools.NewExtractXMP(), "-i", testFile2, "-console")
	if code != 0 {
		t.Fatalf("-console exited %d (%s), want 0", code, stderr)
	}
	if stdout != string(data) {
		t.Errorf("-console wrote %d bytes and -o wrote %d; they should be the same",
			len(stdout), len(data))
	}
}

var _ = aFormDocument
