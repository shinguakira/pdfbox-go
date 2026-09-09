package tools_test

// `pdfbox render`, the command `track/raster` exists to make possible.
//
// Java's PDFToImage has no test of its own -- nothing in `tools` does -- so
// what is asserted here is what the class does, option by option: which pages
// come out, what they are called, what format they are in, and what happens to
// a format nothing can write.

import (
	goimage "image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/tools"
)

// renderFixture copies hello3.pdf out of the Java tree so that the command can
// write its images beside it.
func renderFixture(t *testing.T) string {
	t.Helper()
	const fixture = "../../tools/src/test/resources/org/apache/pdfbox/hello3.pdf"
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("reading hello3.pdf: %v", err)
	}
	path := filepath.Join(t.TempDir(), "hello3.pdf")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
	return path
}

// renderedPage is a page of the file `render` wrote, decoded.
func renderedPage(t *testing.T, path string) goimage.Image {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening what render wrote: %v", err)
	}
	defer file.Close()
	image, _, err := goimage.Decode(file)
	if err != nil {
		t.Fatalf("decoding what render wrote: %v", err)
	}
	return image
}

// TestRenderWritesOnePNGPerPage is the loop: every page from startPage to
// endPage, named `<prefix>-<n>.<format>`.
func TestRenderWritesOnePNGPerPage(t *testing.T) {
	input := renderFixture(t)
	prefix := filepath.Join(t.TempDir(), "out")

	code, _, stderr := runCommand(tools.NewPDFToImage(),
		"-i", input, "-format", "png", "-prefix", prefix, "-dpi", "36")
	if code != tools.ExitOK {
		t.Fatalf("render = %d, stderr %q", code, stderr)
	}

	first := prefix + "-1.png"
	if _, err := os.Stat(first); err != nil {
		t.Fatalf("render wrote no %s: %v", first, err)
	}
	// hello3.pdf is one page, so there is no second.
	if _, err := os.Stat(prefix + "-2.png"); err == nil {
		t.Error("render wrote a second page for a one-page document")
	}

	// At 36 DPI a US Letter page is half its size in points.
	image := renderedPage(t, first)
	if got := image.Bounds().Dx(); got != 306 {
		t.Errorf("the page is %d pixels wide at 36 DPI, want 306", got)
	}
	if got := image.Bounds().Dy(); got != 396 {
		t.Errorf("the page is %d pixels tall at 36 DPI, want 396", got)
	}

	// It is a page with text on it, so it is not blank.
	inked := 0
	for y := image.Bounds().Min.Y; y < image.Bounds().Max.Y; y++ {
		for x := image.Bounds().Min.X; x < image.Bounds().Max.X; x++ {
			if r, _, _, _ := image.At(x, y).RGBA(); r>>8 < 0x80 {
				inked++
			}
		}
	}
	if inked == 0 {
		t.Error("the rendered page is blank")
	}
}

// TestRenderTakesOnePage is -page, which sets both ends of the range.
func TestRenderTakesOnePage(t *testing.T) {
	input := renderFixture(t)
	prefix := filepath.Join(t.TempDir(), "one")

	code, _, stderr := runCommand(tools.NewPDFToImage(),
		"-i", input, "-format", "png", "-prefix", prefix, "-page", "1", "-dpi", "18")
	if code != tools.ExitOK {
		t.Fatalf("render = %d, stderr %q", code, stderr)
	}
	if _, err := os.Stat(prefix + "-1.png"); err != nil {
		t.Errorf("render wrote no page 1: %v", err)
	}
}

// TestRenderDefaultsToJPEG is the `-format` default, and the default prefix,
// which is the input file without its extension.
func TestRenderDefaultsToJPEG(t *testing.T) {
	input := renderFixture(t)

	code, _, stderr := runCommand(tools.NewPDFToImage(), "-i", input, "-dpi", "18")
	if code != tools.ExitOK {
		t.Fatalf("render = %d, stderr %q", code, stderr)
	}
	written := strings.TrimSuffix(input, filepath.Ext(input)) + "-1.jpg"
	if _, err := os.Stat(written); err != nil {
		t.Errorf("render wrote no %s: %v", written, err)
	}
}

// TestRenderRejectsAFormatNothingWrites is the `getWriterFormatNames` check,
// which answers 2 and names the formats it does have.
func TestRenderRejectsAFormatNothingWrites(t *testing.T) {
	input := renderFixture(t)

	code, _, stderr := runCommand(tools.NewPDFToImage(),
		"-i", input, "-format", "webp")
	if code != 2 {
		t.Errorf("render = %d, want 2", code)
	}
	for _, want := range []string{"Invalid image format webp", "png", "jpg"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("the error does not mention %q:\n%s", want, stderr)
		}
	}
}

// TestRenderNeedsAnInput is the required option.
func TestRenderNeedsAnInput(t *testing.T) {
	code, _, stderr := runCommand(tools.NewPDFToImage())
	if code != tools.ExitUsage {
		t.Errorf("render = %d, want %d", code, tools.ExitUsage)
	}
	if !strings.Contains(stderr, "Missing required option") {
		t.Errorf("the error does not say what is missing:\n%s", stderr)
	}
}
