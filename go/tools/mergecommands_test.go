package tools_test

// The `merge` and `overlay` commands, which Java has no test for.
//
// `tools` has four test classes and none of them covers either, so these are
// written from the source the way `track/tools` wrote its eighteen and
// `track/imageio` wrote `export:images`. What they assert is what the two
// `call()` methods decide: which options are required, what is handed to
// `PDFMergerUtility` and `Overlay`, and the exit code each failure gets.
//
// The two utilities themselves are tested against the Java's own output in
// `go/pdfbox/multipdf`.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/tools"
)

// TestMergeJoinsItsInputs is the whole of PDFMerger.call: every -i in order,
// the -o as the destination, and a merge.
func TestMergeJoinsItsInputs(t *testing.T) {
	dir := t.TempDir()
	first := writePages(t, filepath.Join(dir, "one.pdf"), 1)
	second := writePages(t, filepath.Join(dir, "two.pdf"), 3)
	out := filepath.Join(dir, "merged.pdf")

	code, _, stderr := runCommand(tools.NewPDFMerger(),
		"-i", first, "-i", second, "-o", out)
	if code != 0 {
		t.Fatalf("exited %d; stderr is %q", code, stderr)
	}

	merged, err := pdfbox.LoadPDF(out)
	if err != nil {
		t.Fatalf("loading the merged file: %v", err)
	}
	defer merged.Close()
	if got := merged.NumberOfPages(); got != 4 {
		t.Errorf("the merged document has %d pages, want 1 + 3", got)
	}
}

// TestMergeReportsAMissingFile is the catch, which names the failure and
// answers 4.
func TestMergeReportsAMissingFile(t *testing.T) {
	out := filepath.Join(t.TempDir(), "merged.pdf")
	code, _, stderr := runCommand(tools.NewPDFMerger(),
		"-i", "nosuch.pdf", "-o", out)
	if code != 4 {
		t.Errorf("exited %d, want 4", code)
	}
	if !strings.Contains(stderr, "Error merging documents") {
		t.Errorf("stderr is %q, want it to say what failed", stderr)
	}
}

// TestMergeNeedsBothOptions is `required = true` on each of them.
func TestMergeNeedsBothOptions(t *testing.T) {
	dir := t.TempDir()
	input := writePages(t, filepath.Join(dir, "one.pdf"), 1)

	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"no input", []string{"-o", filepath.Join(dir, "merged.pdf")}, "input"},
		{"no output", []string{"-i", input}, "output"},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, _, stderr := runCommand(tools.NewPDFMerger(), c.args...)
			if code != 2 {
				t.Errorf("exited %d, want 2", code)
			}
			if !strings.Contains(stderr, c.want) {
				t.Errorf("stderr is %q, want it to name the option", stderr)
			}
		})
	}
}

// TestOverlayPutsTheDefaultOverlayOnEveryPage is the -default option, which
// reaches Overlay.setDefaultOverlayFile.
func TestOverlayPutsTheDefaultOverlayOnEveryPage(t *testing.T) {
	dir := t.TempDir()
	input := writePages(t, filepath.Join(dir, "input.pdf"), 3)
	overlay := writePages(t, filepath.Join(dir, "overlay.pdf"), 1)
	out := filepath.Join(dir, "out.pdf")

	code, _, stderr := runCommand(tools.NewOverlayPDF(),
		"-i", input, "-default", overlay, "-o", out)
	if code != 0 {
		t.Fatalf("exited %d; stderr is %q", code, stderr)
	}

	result, err := pdfbox.LoadPDF(out)
	if err != nil {
		t.Fatalf("loading the overlaid file: %v", err)
	}
	defer result.Close()
	if got := result.NumberOfPages(); got != 3 {
		t.Fatalf("the result has %d pages, want 3", got)
	}
	for page := range result.Pages().All {
		resources := page.Resources()
		if resources == nil {
			t.Fatal("an overlaid page has no resources")
		}
		if !namesAnOverlay(resources.XObjectNames()) {
			t.Errorf("a page carries %v, want one named OL...", resources.XObjectNames())
		}
	}
}

// TestOverlayTakesAPageMap is the repeatable -page option, whose value is
// `<pageNumber>=<file>`.
func TestOverlayTakesAPageMap(t *testing.T) {
	dir := t.TempDir()
	input := writePages(t, filepath.Join(dir, "input.pdf"), 2)
	overlay := writePages(t, filepath.Join(dir, "overlay.pdf"), 1)
	out := filepath.Join(dir, "out.pdf")

	code, _, stderr := runCommand(tools.NewOverlayPDF(),
		"-i", input, "-page", "2="+overlay, "-o", out)
	if code != 0 {
		t.Fatalf("exited %d; stderr is %q", code, stderr)
	}

	result, err := pdfbox.LoadPDF(out)
	if err != nil {
		t.Fatalf("loading the overlaid file: %v", err)
	}
	defer result.Close()
	if first := result.Page(0).Resources(); first != nil &&
		namesAnOverlay(first.XObjectNames()) {
		t.Error("the first page carries an overlay and only page 2 was named")
	}
	second := result.Page(1).Resources()
	if second == nil || !namesAnOverlay(second.XObjectNames()) {
		t.Error("the second page carries no overlay")
	}
}

// TestOverlayRefusesAnUnknownPosition is picocli's conversion of the -position
// value to the enum, which fails the command rather than defaulting.
func TestOverlayRefusesAnUnknownPosition(t *testing.T) {
	dir := t.TempDir()
	input := writePages(t, filepath.Join(dir, "input.pdf"), 1)

	code, _, stderr := runCommand(tools.NewOverlayPDF(),
		"-i", input, "-o", filepath.Join(dir, "out.pdf"), "-position", "SIDEWAYS")
	if code != 2 {
		t.Errorf("exited %d, want 2", code)
	}
	if !strings.Contains(stderr, "position") {
		t.Errorf("stderr is %q, want it to name the option", stderr)
	}
}

// TestOverlayReportsAMissingFile is the catch, which answers 4.
func TestOverlayReportsAMissingFile(t *testing.T) {
	dir := t.TempDir()
	code, _, stderr := runCommand(tools.NewOverlayPDF(),
		"-i", "nosuch.pdf", "-o", filepath.Join(dir, "out.pdf"))
	if code != 4 {
		t.Errorf("exited %d, want 4", code)
	}
	if !strings.Contains(stderr, "Error adding overlay(s) to PDF") {
		t.Errorf("stderr is %q, want it to say what failed", stderr)
	}
}

// TestOverlayNeedsBothOptions is `required = true` on the input and the output.
func TestOverlayNeedsBothOptions(t *testing.T) {
	dir := t.TempDir()
	input := writePages(t, filepath.Join(dir, "input.pdf"), 1)

	for _, c := range []struct {
		name string
		args []string
		want string
	}{
		{"no input", []string{"-o", filepath.Join(dir, "out.pdf")}, "input"},
		{"no output", []string{"-i", input}, "output"},
	} {
		t.Run(c.name, func(t *testing.T) {
			code, _, stderr := runCommand(tools.NewOverlayPDF(), c.args...)
			if code != 2 {
				t.Errorf("exited %d, want 2", code)
			}
			if !strings.Contains(stderr, c.want) {
				t.Errorf("stderr is %q, want it to name the option", stderr)
			}
		})
	}
}

// writePages saves a document of the given number of blank pages and answers
// its path.
func writePages(t *testing.T, path string, pages int) string {
	t.Helper()
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	for i := 0; i < pages; i++ {
		doc.AddPage(pdmodel.NewPDPage())
	}
	if err := doc.SaveToFile(path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	return path
}

// namesAnOverlay reports whether one of the names is an overlay form XObject,
// which Overlay adds under the prefix "OL".
func namesAnOverlay(names []*cos.Name) bool {
	for _, name := range names {
		if strings.HasPrefix(name.Name(), "OL") {
			return true
		}
	}
	return false
}
