package pdfwriter_test

// ContentStreamWriterTest.testPDFBox4750, as far as it goes here.
//
// Java parses a page's content stream, writes the tokens straight back out
// through ContentStreamWriter, renders both the original and the rewritten
// document, and asserts the images are identical. The port could not: there was
// no renderer, and the deferral in STATUS.md said "needs PDFRenderer and
// TestPDFToImage — slice 9". `track/raster` brings the renderer.
//
// **The Java's own file is not portable and the page is a different one.**
// `testPDFBox4750` reads `target/pdfs/PDFBOX-4750.pdf`, which the Java build
// downloads and which is not in this repository. What is used instead is
// `rendering/raster/testdata/graphics.pdf`, written by this port and already
// compared with PDFBox's own render elsewhere: fills under both winding rules,
// strokes with every join, cap and dash, a clip, a transform, an alpha
// constant, a blend mode and an axial shading.
//
// The assertion is Java's -- **identical**, not merely close -- because both
// sides go through the same renderer and the only thing between them is the
// writer.

import (
	goimage "image"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfwriter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/rendering/raster"
)

// graphicsPDF is the page written by rendering/raster/testdata/genpdf.go.
const graphicsPDF = "../rendering/raster/testdata/graphics.pdf"

// TestWritingTheTokensBackRendersTheSamePage is the round trip.
func TestWritingTheTokensBackRendersTheSamePage(t *testing.T) {
	before := renderGraphics(t, false)
	after := renderGraphics(t, true)

	if got, want := after.Bounds(), before.Bounds(); got != want {
		t.Fatalf("the rewritten page rendered %v and the original %v", got, want)
	}
	for y := before.Bounds().Min.Y; y < before.Bounds().Max.Y; y++ {
		for x := before.Bounds().Min.X; x < before.Bounds().Max.X; x++ {
			if before.At(x, y) != after.At(x, y) {
				t.Fatalf("(%d,%d) is %v after the round trip and was %v",
					x, y, after.At(x, y), before.At(x, y))
			}
		}
	}
}

// renderGraphics renders the page, first putting its content stream through the
// parser and the writer where rewrite says so.
func renderGraphics(t *testing.T, rewrite bool) goimage.Image {
	t.Helper()
	document, err := pdfbox.LoadPDF(graphicsPDF)
	if err != nil {
		t.Fatalf("loading the page: %v", err)
	}
	defer document.Close()

	if rewrite {
		page := document.Page(0)
		contents, err := page.ContentsForStreamParsing()
		if err != nil {
			t.Fatalf("reading the content stream: %v", err)
		}
		parser, err := pdfparser.NewStreamTokenParserSource(contents)
		if err != nil {
			t.Fatalf("parsing the content stream: %v", err)
		}
		tokens, err := parser.Parse()
		if err != nil {
			t.Fatalf("parsing the content stream: %v", err)
		}

		rewritten := common.NewPDStreamOfDocument(document.Document())
		out, err := rewritten.CreateOutputStreamOfFilter(cos.FlateDecode)
		if err != nil {
			t.Fatalf("opening the new content stream: %v", err)
		}
		if err := pdfwriter.NewContentStreamWriter(out).WriteTokens(tokens); err != nil {
			t.Fatalf("writing the tokens back: %v", err)
		}
		if err := out.Close(); err != nil {
			t.Fatalf("closing the new content stream: %v", err)
		}
		page.SetContents(rewritten)
	}

	image, err := raster.RenderPage(document, 0, 1, rendering.RGB)
	if err != nil {
		t.Fatalf("rendering the page: %v", err)
	}
	return image
}
