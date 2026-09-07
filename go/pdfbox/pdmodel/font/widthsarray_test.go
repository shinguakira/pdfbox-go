package font

// The two run-compressing width builders, checked against the running Java.
//
// getWidths and getVerticalMetrics are the same three-state machine over
// different tuple sizes, and both round every metric. The vertical one is the
// only place in this port where a *negative* value is rounded on a half
// boundary, which is where Java's round-half-up and Go's round-half-away-from-
// zero disagree: an advance height of 128 in a 2048-unit em is -62.5, and Java
// writes -62.
//
// No font in this repository has a `vhea` table -- the Java cases that write
// one download ipag.ttf -- so the vertical builder cannot be reached through a
// document. It is driven here with the array a font would produce, and the
// wanted values come from the same two methods in the running Java, reached by
// reflection because both are private.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// fakeEmbeddingDocument is what an embedder asks of a PDDocument. pdmodel
// imports this package, so a test in it cannot build a real PDDocument.
type fakeEmbeddingDocument struct {
	version float32
	closing []*ttf.TrueTypeFont
}

func (d *fakeEmbeddingDocument) CreateStream() *cos.Stream     { return cos.NewStream(nil) }
func (d *fakeEmbeddingDocument) Version() float32              { return d.version }
func (d *fakeEmbeddingDocument) SetVersion(newVersion float32) { d.version = newVersion }
func (d *fakeEmbeddingDocument) RegisterTrueTypeFontForClosing(font *ttf.TrueTypeFont) {
	d.closing = append(d.closing, font)
}

// TestWidthArraysMatchJava drives both builders with a synthetic array.
func TestWidthArraysMatchJava(t *testing.T) {
	source, err := pdfio.OpenBufferedFile(
		"../../../../pdfbox/src/main/resources/org/apache/pdfbox/resources/ttf/" +
			"LiberationSans-Regular.ttf")
	if err != nil {
		t.Fatalf("opening LiberationSans-Regular.ttf: %v", err)
	}
	font, err := ttf.NewParser().Parse(source)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	defer font.Close()

	header, err := font.Header()
	if err != nil {
		t.Fatalf("Header: %v", err)
	}
	if got := header.UnitsPerEm(); got != 2048 {
		t.Fatalf("LiberationSans reports %d units per em; the wanted values below "+
			"were measured at 2048", got)
	}

	embedder, err := newPDCIDFontType2Embedder(&fakeEmbeddingDocument{},
		cos.NewDictionary(), font, true, false)
	if err != nil {
		t.Fatalf("newPDCIDFontType2Embedder: %v", err)
	}

	// cid, advance height, advance width, yMax + top side bearing.
	//
	// The first three rows are identical, so the machine runs FIRST to SERIAL
	// and stays there; then a break to FIRST, two rows that differ only in
	// value so it goes to BRACKET, the MinInt32 row a glyph-less CID produces,
	// and a last row that breaks the run.
	//
	// 128 * 1000/2048 is exactly 62.5, and the entry is its negation.
	verticalInput := []int{
		10, 128, 1000, 1400,
		11, 128, 1000, 1400,
		12, 128, 1000, 1400,
		20, 256, 500, 700,
		21, 300, 500, 700,
		22, 400, 500, 700,
		minInt32Marker, 0, 0, 0,
		30, 100, 200, 300,
	}
	const wantVertical = "[ 10 12 -62 244 684 20 [ -125 122 342 -146 122 342 -195 122 342 ] " +
		"30 [ -49 49 146 ] ]"

	vertical, err := embedder.getVerticalMetrics(verticalInput)
	if err != nil {
		t.Fatalf("getVerticalMetrics: %v", err)
	}
	if got := renderCOS(vertical); got != wantVertical {
		t.Errorf("getVerticalMetrics wrote\n  %s\nwant Java's\n  %s", got, wantVertical)
	}

	// cid, advance width.
	horizontalInput := []int{
		0, 1000,
		1, 1000,
		2, 1000,
		3, 500,
		4, 600,
		5, 700,
		9, 900,
	}
	const wantHorizontal = "[ 0 2 488 3 [ 244 293 342 ] 9 [ 439 ] ]"

	horizontal, err := embedder.getWidths(horizontalInput)
	if err != nil {
		t.Fatalf("getWidths: %v", err)
	}
	if got := renderCOS(horizontal); got != wantHorizontal {
		t.Errorf("getWidths wrote\n  %s\nwant Java's\n  %s", got, wantHorizontal)
	}
}

// minInt32Marker is what buildVerticalMetrics writes for a CID with no glyph,
// and what getVerticalMetrics skips on.
const minInt32Marker = -1 << 31

// renderCOS prints an array of numbers and nested arrays the way the Java
// driver printed it.
func renderCOS(base cos.Base) string {
	var sb strings.Builder
	renderCOSInto(base, &sb)
	return strings.TrimSpace(sb.String())
}

func renderCOSInto(base cos.Base, sb *strings.Builder) {
	switch value := base.(type) {
	case *cos.Array:
		sb.WriteString("[ ")
		for i := 0; i < value.Size(); i++ {
			renderCOSInto(value.Get(i), sb)
		}
		sb.WriteString("] ")
	case *cos.Integer:
		fmt.Fprintf(sb, "%d ", value.LongValue())
	default:
		fmt.Fprintf(sb, "%v ", base)
	}
}
