package tools_test

// Port of org.apache.pdfbox.tools.TestExtractText, all seven cases.
//
// Five of them go through PDFBox.main with **repeatable subcommands**: one
// invocation carrying `export:text` twice, each with its own options. Java
// replaces System.out to read the result; the port passes the streams in.

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/tools"
)

// The three fixtures the Java test names.
const (
	testFile1 = "../../tools/src/test/resources/org/apache/pdfbox/testPDFPackage.pdf"
	testFile2 = "../../tools/src/test/resources/org/apache/pdfbox/hello3.pdf"
	testFile3 = "../../tools/src/test/resources/org/apache/pdfbox/AngledExample.pdf"
)

// runExtractText runs the command on its own, which is what the first two cases
// do with `new CommandLine(app).execute(...)`.
func runExtractText(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := tools.Execute(tools.NewExtractText(), args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// runPDFBox runs the dispatcher, which is what the other five do with
// PDFBox.main.
func runPDFBox(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := tools.NewPDFBox(&stdout, &stderr).Run(args)
	return code, stdout.String(), stderr.String()
}

// checkText is the block of assertTrue/assertFalse every case ends with.
func checkText(t *testing.T, what, result string, wants map[string]bool) {
	t.Helper()
	for needle, want := range wants {
		if got := strings.Contains(result, needle); got != want {
			t.Errorf("%s: contains(%q) = %v, want %v\n%s", what, needle, got, want, result)
		}
	}
}

// TestEmbeddedPDFs is testEmbeddedPDFs: a PDF with embedded PDFs, whose text is
// extracted along with the outer document's.
func TestEmbeddedPDFs(t *testing.T) {
	code, out, errOut := runExtractText("-i", testFile1, "-console")
	if code != 0 {
		t.Fatalf("exited %d (%s), want 0", code, errOut)
	}
	checkText(t, "testEmbeddedPDFs", out, map[string]bool{
		"PDF1":                   true,
		"PDF2":                   true,
		"PDF file: " + testFile1: false,
		"Hello":                  false,
		"World.":                 false,
		"PDF file: " + testFile2: false,
	})
}

// TestAddFileName is testAddFileName: -addFileName prints the file name.
func TestAddFileName(t *testing.T) {
	code, out, errOut := runExtractText("-i", testFile1, "-console", "-addFileName")
	if code != 0 {
		t.Fatalf("exited %d (%s), want 0", code, errOut)
	}
	checkText(t, "testAddFileName", out, map[string]bool{
		"PDF1":                   true,
		"PDF2":                   true,
		"PDF file: " + testFile1: true,
		"Hello":                  false,
		"World.":                 false,
		"PDF file: " + testFile2: false,
	})
}

// TestPDFBoxRepeatableSubcommand is testPDFBoxRepeatableSubcommand: one
// invocation, the same subcommand twice.
func TestPDFBoxRepeatableSubcommand(t *testing.T) {
	_, out, _ := runPDFBox(
		"export:text", "-i", testFile1, "-console",
		"export:text", "-i", testFile2, "-console")

	checkText(t, "testPDFBoxRepeatableSubcommand", out, map[string]bool{
		"PDF1":                   true,
		"PDF2":                   true,
		"PDF file: " + testFile1: false,
		"Hello":                  true,
		"World.":                 true,
		"PDF file: " + testFile2: false,
	})
}

// TestPDFBoxRepeatableSubcommandAddFileName is the same with -addFileName on
// both.
func TestPDFBoxRepeatableSubcommandAddFileName(t *testing.T) {
	_, out, _ := runPDFBox(
		"export:text", "-i", testFile1, "-console", "-addFileName",
		"export:text", "-i", testFile2, "-console", "-addFileName")

	checkText(t, "testPDFBoxRepeatableSubcommandAddFileName", out, map[string]bool{
		"PDF1":                   true,
		"PDF2":                   true,
		"PDF file: " + testFile1: true,
		"Hello":                  true,
		"World.":                 true,
		"PDF file: " + testFile2: true,
	})
}

// TestPDFBoxRepeatableSubcommandAddFileNameOutfile writes both runs to the same
// file **without** -append, so the second truncates the first.
func TestPDFBoxRepeatableSubcommandAddFileNameOutfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outfile.txt")

	runPDFBox(
		"export:text", "-i", testFile1, "-encoding", "UTF-8", "-addFileName", "-o", path,
		"export:text", "-i", testFile2, "-encoding", "UTF-8", "-addFileName", "-o", path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	checkText(t, "testPDFBoxRepeatableSubcommandAddFileNameOutfile", string(data),
		map[string]bool{
			"PDF1":                   false,
			"PDF2":                   false,
			"PDF file: " + testFile1: false,
			"Hello":                  true,
			"World.":                 true,
			"PDF file: " + testFile2: true,
		})
}

// TestPDFBoxRepeatableSubcommandAddFileNameOutfileAppend is the same **with**
// -append on the second, so both runs survive.
func TestPDFBoxRepeatableSubcommandAddFileNameOutfileAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outfile.txt")

	runPDFBox(
		"export:text", "-i", testFile1, "-encoding", "UTF-8", "-addFileName", "-o", path,
		"export:text", "-i", testFile2, "-encoding", "UTF-8", "-addFileName", "-o", path,
		"-append")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	checkText(t, "testPDFBoxRepeatableSubcommandAddFileNameOutfileAppend", string(data),
		map[string]bool{
			"PDF1":                   true,
			"PDF2":                   true,
			"PDF file: " + testFile1: true,
			"Hello":                  true,
			"World.":                 true,
			"PDF file: " + testFile2: true,
		})
}

// TestRotationMagic is testRotationMagic: -rotationMagic finds the text at
// every angle on the page, one angle at a time.
func TestRotationMagic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "outfile.txt")

	runPDFBox("export:text", "-rotationMagic", "-i", testFile3, "-o", path)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	checkText(t, "testRotationMagic", string(data), map[string]bool{
		"Horizontal Text": true,
		"Vertical Text":   true,
	})
}
