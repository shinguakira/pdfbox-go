package tools

// The two slice 1 commands, DecompressObjectstreams and WriteDecodedDoc.
//
// **Neither has a Java test.** A2 asks for a decision either way, and it is
// worth writing these: what each command adds over the library call underneath
// is a default (the output file name), a filter (-skipImages), and a flag
// surface, and none of the three is covered by any library test.
//
// The fixture is tools/src/test/resources/org/apache/pdfbox/hello3.pdf, which
// the Java tests of this module use.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
)

// toolsFixture is where this module's Java test resources are.
const toolsFixture = "../../tools/src/test/resources/org/apache/pdfbox/"

// copyFixture puts a fixture in a temporary directory, for the cases that write
// over their input.
func copyFixture(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(toolsFixture + name)
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

// TestDecompressObjectstreamsWritesToTheOutputFile is -i with -o.
func TestDecompressObjectstreamsWritesToTheOutputFile(t *testing.T) {
	in := toolsFixture + "hello3.pdf"
	out := filepath.Join(t.TempDir(), "out.pdf")

	code, stdout, stderr := run(NewDecompressObjectstreams(), "-i", in, "-o", out)
	if code != ExitOK {
		t.Fatalf("exited %d (%s), want %d", code, stderr, ExitOK)
	}
	if stdout != "" {
		t.Errorf("wrote %q to stdout; this command prints nothing when it works", stdout)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("the output file was not written: %v", err)
	}
	document, err := pdfbox.LoadPDF(out)
	if err != nil {
		t.Fatalf("the output is not a PDF: %v", err)
	}
	defer document.Close()
	if got := document.NumberOfPages(); got != 1 {
		t.Errorf("the output has %d pages, want 1", got)
	}
}

// TestDecompressObjectstreamsOverwritesItsInput is the -o default, which is the
// one thing this command does that the library does not: "If omitted the
// original file is overwritten."
func TestDecompressObjectstreamsOverwritesItsInput(t *testing.T) {
	path := copyFixture(t, "hello3.pdf")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	code, _, stderr := run(NewDecompressObjectstreams(), "-i", path)
	if code != ExitOK {
		t.Fatalf("exited %d (%s), want %d", code, stderr, ExitOK)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) == string(after) {
		t.Error("the input file is unchanged; it should have been overwritten")
	}
	document, err := pdfbox.LoadPDF(path)
	if err != nil {
		t.Fatalf("the overwritten file is not a PDF: %v", err)
	}
	document.Close()
}

// TestDecompressObjectstreamsHasNoVersionOption is the flag surface. Java's
// @Command for this one is not mixinStandardHelpOptions -- it declares
// -h/--help itself with usageHelp = true -- so -V is not one of its options and
// accepting it would be accepting an argument the Java refuses.
func TestDecompressObjectstreamsHasNoVersionOption(t *testing.T) {
	code, out, _ := run(NewDecompressObjectstreams(), "-V")
	if code != ExitUsage {
		t.Errorf("-V exited %d, want %d: this command has no version option",
			code, ExitUsage)
	}
	if out != "" {
		t.Errorf("-V printed %q to stdout", out)
	}

	// -h is one of its options, and prints the header.
	code, out, _ = run(NewDecompressObjectstreams(), "-h")
	if code != ExitOK {
		t.Errorf("-h exited %d, want %d", code, ExitOK)
	}
	if !strings.Contains(out, "Decompresses object streams") {
		t.Errorf("-h printed %q, want the header", out)
	}
}

// TestVersionCommandHasNeitherHelpNorVersionOption is the same check for
// `version`, whose @Command carries neither mixin.
func TestVersionCommandHasNeitherHelpNorVersionOption(t *testing.T) {
	for _, name := range []string{"-h", "--help", "-V", "--version"} {
		code, _, _ := run(NewVersion(), name)
		if code != ExitUsage {
			t.Errorf("version %s exited %d, want %d: it declares no such option",
				name, code, ExitUsage)
		}
	}
}

// TestWriteDecodedDocNamesItsOutput is calculateOutputFilename, which is the
// command's own rule and not the library's.
func TestWriteDecodedDocNamesItsOutput(t *testing.T) {
	for _, row := range []struct{ in, want string }{
		{"doc.pdf", "doc_unc.pdf"},
		{"doc.PDF", "doc_unc.pdf"},
		{"doc.Pdf", "doc_unc.pdf"},
		{"doc", "doc_unc.pdf"},
		{"doc.txt", "doc.txt_unc.pdf"},
		{"a.pdf.pdf", "a.pdf_unc.pdf"},
	} {
		if got := calculateOutputFilename(row.in); got != row.want {
			t.Errorf("calculateOutputFilename(%q) = %q, want %q", row.in, got, row.want)
		}
	}
}

// TestWriteDecodedDocDecodesEveryStream runs the command and checks that no
// stream in the result still carries a /Filter.
func TestWriteDecodedDocDecodesEveryStream(t *testing.T) {
	in := toolsFixture + "hello3.pdf"
	out := filepath.Join(t.TempDir(), "decoded.pdf")

	code, stdout, stderr := run(NewWriteDecodedDoc(), in, out)
	if code != ExitOK {
		t.Fatalf("exited %d (%s), want %d", code, stderr, ExitOK)
	}
	if stdout != "" {
		t.Errorf("wrote %q to stdout", stdout)
	}

	document, err := pdfbox.LoadPDF(out)
	if err != nil {
		t.Fatalf("the output is not a PDF: %v", err)
	}
	defer document.Close()

	filtered := 0
	for key := range document.Document().XRefTable() {
		object := document.Document().ObjectFromPool(key)
		if object == nil {
			continue
		}
		if stream, ok := object.Object().(*cos.Stream); ok {
			if stream.GetItem(cos.Filter) != nil {
				filtered++
			}
		}
	}
	if filtered != 0 {
		t.Errorf("%d streams still carry a /Filter; every one should be decoded", filtered)
	}
}

// TestWriteDecodedDocSkipImages is -skipImages, the one option that changes
// what the command does.
func TestWriteDecodedDocSkipImages(t *testing.T) {
	// A document with an image XObject. This one is in the module's own
	// fixtures for the ImageIOUtil tests.
	const withImage = "../../tools/src/test/resources/input/ImageIOUtil/png_demo.pdf"

	countFilteredImages := func(path string) int {
		t.Helper()
		document, err := pdfbox.LoadPDF(path)
		if err != nil {
			t.Fatalf("loading %s: %v", path, err)
		}
		defer document.Close()
		filtered := 0
		for key := range document.Document().XRefTable() {
			object := document.Document().ObjectFromPool(key)
			if object == nil {
				continue
			}
			stream, ok := object.Object().(*cos.Stream)
			if !ok {
				continue
			}
			if stream.GetItem(cos.Subtype) == cos.Image && stream.GetItem(cos.Filter) != nil {
				filtered++
			}
		}
		return filtered
	}

	if countFilteredImages(withImage) == 0 {
		t.Skip("the fixture has no compressed image XObject, so the option cannot be told apart")
	}

	decoded := filepath.Join(t.TempDir(), "decoded.pdf")
	if code, _, stderr := run(NewWriteDecodedDoc(), withImage, decoded); code != ExitOK {
		t.Fatalf("without -skipImages: exited %d (%s)", code, stderr)
	}
	skipped := filepath.Join(t.TempDir(), "skipped.pdf")
	if code, _, stderr := run(NewWriteDecodedDoc(), "-skipImages", withImage, skipped); code != ExitOK {
		t.Fatalf("with -skipImages: exited %d (%s)", code, stderr)
	}

	if got := countFilteredImages(decoded); got != 0 {
		t.Errorf("without -skipImages, %d image streams still carry a /Filter", got)
	}
	if got := countFilteredImages(skipped); got == 0 {
		t.Error("with -skipImages, no image stream carries a /Filter; they should be untouched")
	}
}

// TestWriteDecodedDocPositionalArity is the @Parameters arity: one file is
// required, a second is optional, and a third is a usage error.
func TestWriteDecodedDocPositionalArity(t *testing.T) {
	in := toolsFixture + "hello3.pdf"

	code, out, errOut := run(NewWriteDecodedDoc())
	if code != ExitUsage {
		t.Errorf("with no file, exited %d, want %d", code, ExitUsage)
	}
	if out != "" {
		t.Errorf("the error went to stdout as %q", out)
	}
	if !strings.Contains(errOut, "<inputfile>") {
		t.Errorf("stderr is %q, want it to name the missing parameter", errOut)
	}

	code, _, errOut = run(NewWriteDecodedDoc(), in, filepath.Join(t.TempDir(), "a.pdf"), "extra")
	if code != ExitUsage {
		t.Errorf("with three files, exited %d, want %d", code, ExitUsage)
	}
	if !strings.Contains(errOut, "index 2") {
		t.Errorf("stderr is %q, want it to name the unmatched argument", errOut)
	}
}
