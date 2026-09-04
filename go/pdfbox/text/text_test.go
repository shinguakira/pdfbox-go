package text_test

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/awt/geom"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// lineFeed is what a page ends with by default.
const lineFeed = "\n"

// PDFTextStripperByAreaTest and BidiTest both open a PDF with Loader, which
// this slice does not carry; the tests below build the page in memory instead
// and run the same walk over it. The expected text of each is what the content
// stream draws, and the expected geometry is what the Java source computes.

// helveticaPage returns a one-page document whose resources name Helvetica as
// /F1, holding the given content stream.
func helveticaPage(t *testing.T, content string) *pdmodel.PDPage {
	t.Helper()
	stream := cos.NewStream(filter.Provider{})
	w, err := stream.CreateWriter()
	if err != nil {
		t.Fatalf("CreateWriter: %v", err)
	}
	if _, err := w.Write([]byte(content)); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	page := pdmodel.NewPDPageOfSize(common.NewPDRectangleOf(0, 0, 300, 400))
	page.Dictionary().SetItem(cos.Contents, stream)
	page.SetResources(pdmodel.NewPDResources())

	fontDict := cos.NewDictionary()
	fontDict.SetItem(cos.Type, cos.Font)
	fontDict.SetItem(cos.Subtype, cos.Type1)
	fontDict.SetName(cos.BaseFont, "Helvetica")
	fonts := cos.NewDictionary()
	fonts.SetItem(cos.GetPDFName("F1"), fontDict)
	page.Resources().Dictionary().SetItem(cos.Font, fonts)
	return page
}

// strip returns the text of a page holding the given content stream.
func strip(t *testing.T, content string) string {
	t.Helper()
	stripper := text.NewPDFTextStripper()
	var out strings.Builder
	stripper.SetOutput(&out)
	if err := stripper.ProcessPage(helveticaPage(t, content)); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	return out.String()
}

// TestStripperOneWord checks the simplest walk there is: one Tj at one place.
func TestStripperOneWord(t *testing.T) {
	got := strip(t, "BT /F1 12 Tf 10 300 Td (Hello) Tj ET")
	if strings.TrimSpace(got) != "Hello" {
		t.Errorf("text = %q, want %q", strings.TrimSpace(got), "Hello")
	}
}

// TestStripperTwoWordsOnOneLine checks that a gap wider than a space becomes a
// word separator, which is the whole of the stripper's spacing heuristic.
func TestStripperTwoWordsOnOneLine(t *testing.T) {
	// Helvetica at 12pt draws "Hello" 30.672 units wide; starting the second
	// word at x = 60 leaves a gap far wider than a space
	got := strip(t, "BT /F1 12 Tf 10 300 Td (Hello) Tj 50 0 Td (World) Tj ET")
	if strings.TrimSpace(got) != "Hello World" {
		t.Errorf("text = %q, want %q", strings.TrimSpace(got), "Hello World")
	}
}

// TestStripperTwoLines checks that a second line below the first is written on
// its own line.
func TestStripperTwoLines(t *testing.T) {
	got := strip(t, "BT /F1 12 Tf 10 300 Td (one) Tj 0 -20 Td (two) Tj ET")
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Fatalf("%d lines, want 2: %q", len(lines), got)
	}
	if lines[0] != "one" || lines[1] != "two" {
		t.Errorf("lines = %q, want [one two]", lines)
	}
}

// TestStripperPageEnd checks that a page ends with the page separator, which is
// a newline by default.
func TestStripperPageEnd(t *testing.T) {
	got := strip(t, "BT /F1 12 Tf 10 300 Td (x) Tj ET")
	if !strings.HasSuffix(got, "\n") {
		t.Errorf("text = %q, want it to end with a newline", got)
	}
}

// TestStripperEmptyPage checks that a page drawing nothing yields nothing.
func TestStripperEmptyPage(t *testing.T) {
	// Java still writes the page separator: writePage writes the page end
	// whether or not anything was drawn
	got := strip(t, "")
	if got != lineFeed {
		t.Errorf("text = %q, want %q", got, lineFeed)
	}
}

// TestStripperSuppressesDuplicateOverlappingText checks the default: text drawn
// twice in the same place is written once.
func TestStripperSuppressesDuplicateOverlappingText(t *testing.T) {
	got := strip(t, "BT /F1 12 Tf 10 300 Td (x) Tj 0 0 Td (x) Tj ET")
	if strings.TrimSpace(got) != "x" {
		t.Errorf("text = %q, want %q", strings.TrimSpace(got), "x")
	}
}

// TestStripperKeepsDuplicateWhenAsked checks that turning the suppression off
// writes both.
func TestStripperKeepsDuplicateWhenAsked(t *testing.T) {
	stripper := text.NewPDFTextStripper()
	stripper.SetSuppressDuplicateOverlappingText(false)
	var out strings.Builder
	stripper.SetOutput(&out)
	page := helveticaPage(t, "BT /F1 12 Tf 10 300 Td (x) Tj 0 0 Td (x) Tj ET")
	if err := stripper.ProcessPage(page); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	if strings.TrimSpace(out.String()) != "xx" {
		t.Errorf("text = %q, want %q", strings.TrimSpace(out.String()), "xx")
	}
}

// TestStripperSortByPosition checks that sorting reads the page down and then
// across, whichever order the operators drew in.
func TestStripperSortByPosition(t *testing.T) {
	// "b" is drawn first but sits below "a"
	content := "BT /F1 12 Tf 10 200 Td (b) Tj ET BT /F1 12 Tf 10 300 Td (a) Tj ET"

	stripper := text.NewPDFTextStripper()
	stripper.SetSortByPosition(true)
	var out strings.Builder
	stripper.SetOutput(&out)
	if err := stripper.ProcessPage(helveticaPage(t, content)); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "b" {
		t.Errorf("sorted lines = %q, want [a b]", lines)
	}
}

// TestStripperPageRange checks that a page outside the range is skipped.
func TestStripperPageRange(t *testing.T) {
	stripper := text.NewPDFTextStripper()
	stripper.SetStartPage(2)
	var out strings.Builder
	stripper.SetOutput(&out)
	if err := stripper.ProcessPage(helveticaPage(t, "BT /F1 12 Tf 10 300 Td (x) Tj ET")); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	if out.String() != "" {
		t.Errorf("page 1 was written out with a start page of 2: %q", out.String())
	}
}

// TestStripperSeparators checks that the separators a caller sets are the ones
// written.
func TestStripperSeparators(t *testing.T) {
	stripper := text.NewPDFTextStripper()
	stripper.SetLineSeparator("|")
	stripper.SetWordSeparator("_")
	stripper.SetPageEnd("")
	var out strings.Builder
	stripper.SetOutput(&out)
	content := "BT /F1 12 Tf 10 300 Td (one) Tj 50 0 Td (two) Tj 0 -20 Td (three) Tj ET"
	if err := stripper.ProcessPage(helveticaPage(t, content)); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	if got := out.String(); got != "one_two|three" {
		t.Errorf("text = %q, want %q", got, "one_two|three")
	}
}

// TestStripperIgnoreContentStreamSpaceGlyphs checks PDFBOX-3774: a space the
// content stream draws can be dropped in favour of one worked out from the
// spacing.
func TestStripperIgnoreContentStreamSpaceGlyphs(t *testing.T) {
	stripper := text.NewPDFTextStripper()
	stripper.SetIgnoreContentStreamSpaceGlyphs(true)
	var out strings.Builder
	stripper.SetOutput(&out)
	page := helveticaPage(t, "BT /F1 12 Tf 10 300 Td (a b) Tj ET")
	if err := stripper.ProcessPage(page); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	// the drawn space is dropped; the gap it left is still wide enough to put a
	// separator back
	if got := strings.TrimSpace(out.String()); got != "a b" {
		t.Errorf("text = %q, want %q", got, "a b")
	}
}

// TestStripperByArea checks that only the text inside a region is written,
// which is what PDFTextStripperByAreaTest checks against a real document.
func TestStripperByArea(t *testing.T) {
	// "in" sits at y = 300, "out" at y = 100; the page is 400 high and the
	// stripper works in coordinates measured from the top, so "in" is at 100
	content := "BT /F1 12 Tf 10 300 Td (in) Tj ET BT /F1 12 Tf 10 100 Td (out) Tj ET"

	stripper := text.NewPDFTextStripperByArea()
	stripper.SetSortByPosition(true)
	stripper.SetLineSeparator("")
	stripper.AddRegion("region", geom.NewRectangle2D(0, 90, 300, 30))
	if err := stripper.ExtractRegions(helveticaPage(t, content)); err != nil {
		t.Fatalf("ExtractRegions: %v", err)
	}
	if got := strings.TrimSpace(stripper.GetTextForRegion("region")); got != "in" {
		t.Errorf("region text = %q, want %q", got, "in")
	}
	if len(stripper.Regions()) != 1 {
		t.Errorf("%d regions, want 1", len(stripper.Regions()))
	}

	stripper.RemoveRegion("region")
	if len(stripper.Regions()) != 0 {
		t.Errorf("%d regions after removing one, want 0", len(stripper.Regions()))
	}
}

// TestStripperByAreaNeverSeparatesByBeads checks that the setting cannot be
// turned back on, which Java enforces by overriding the setter to do nothing.
func TestStripperByAreaNeverSeparatesByBeads(t *testing.T) {
	stripper := text.NewPDFTextStripperByArea()
	if stripper.SeparateByBeads() {
		t.Fatal("a stripper by area starts out separating by beads")
	}
	stripper.SetShouldSeparateByBeads(true)
	if stripper.SeparateByBeads() {
		t.Error("SetShouldSeparateByBeads(true) took effect")
	}
}

// TestMarkedContentExtractor checks that the extractor collects the text of
// each marked content sequence.
func TestMarkedContentExtractor(t *testing.T) {
	content := "/Span <</MCID 0>> BDC BT /F1 12 Tf 10 300 Td (inside) Tj ET EMC"

	extractor := text.NewPDFMarkedContentExtractor()
	if err := extractor.ProcessPage(helveticaPage(t, content)); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}
	contents := extractor.MarkedContents()
	if len(contents) != 1 {
		t.Fatalf("%d marked content sequences, want 1", len(contents))
	}
	tag, ok := contents[0].Tag()
	if !ok || tag != "Span" {
		t.Errorf("tag = %q, want %q", tag, "Span")
	}
	var collected strings.Builder
	for _, item := range contents[0].Contents() {
		if position, ok := item.(*text.TextPosition); ok {
			collected.WriteString(position.Unicode())
		}
	}
	if collected.String() != "inside" {
		t.Errorf("collected text = %q, want %q", collected.String(), "inside")
	}
}

// TestActualTextReplacesTheGlyphs checks that /ActualText on a marked content
// sequence stands in for whatever the glyphs would have said.
func TestActualTextReplacesTheGlyphs(t *testing.T) {
	content := "/Span <</ActualText (real)>> BDC BT /F1 12 Tf 10 300 Td (junk) Tj ET EMC"
	got := strings.TrimSpace(strip(t, content))
	if got != "real" {
		t.Errorf("text = %q, want %q", got, "real")
	}
}

// --- TextPosition, which Java tests only through the stripper ---

// textPositionAt returns a text position at the given place, with the geometry
// the stripper works in.
func textPositionAt(x, y, width, height float32) *text.TextPosition {
	matrix := util.NewMatrixOf(12, 0, 0, 12, x, y)
	return text.NewTextPosition(0, 300, 400, matrix, x+width, y, height, width, 4,
		"a", []int{97}, nil, 12, 12)
}

// TestTextPositionCoordinates checks the two coordinate systems a position
// reports in: the PDF one, and the one measured from the top of the page.
func TestTextPositionCoordinates(t *testing.T) {
	position := textPositionAt(10, 300, 6, 8)
	if got := position.X(); got != 10 {
		t.Errorf("X() = %v, want 10", got)
	}
	// the page is 400 high, so a baseline at y = 300 is 100 from the top
	if got := position.Y(); got != 100 {
		t.Errorf("Y() = %v, want 100", got)
	}
	if got := position.Width(); got != 6 {
		t.Errorf("Width() = %v, want 6", got)
	}
	if got := position.Height(); got != 8 {
		t.Errorf("Height() = %v, want 8", got)
	}
	if got := position.Dir(); got != 0 {
		t.Errorf("Dir() = %v, want 0", got)
	}
	if got := position.XDirAdj(); got != 10 {
		t.Errorf("XDirAdj() = %v, want 10", got)
	}
	if got := position.YDirAdj(); got != 100 {
		t.Errorf("YDirAdj() = %v, want 100", got)
	}
}

// TestTextPositionDirection checks the four text directions, which the
// stripper sorts within.
func TestTextPositionDirection(t *testing.T) {
	cases := []struct {
		name   string
		matrix *util.Matrix
		want   float32
	}{
		// a b / c d, read as ScaleX ShearY / ShearX ScaleY
		{"left to right", util.NewMatrixOf(12, 0, 0, 12, 0, 0), 0},
		{"upside down", util.NewMatrixOf(-12, 0, 0, -12, 0, 0), 180},
		{"up", util.NewMatrixOf(0, 12, -12, 0, 0, 0), 90},
		{"down", util.NewMatrixOf(0, -12, 12, 0, 0, 0), 270},
	}
	for _, c := range cases {
		position := text.NewTextPosition(0, 300, 400, c.matrix, 0, 0, 8, 6, 4,
			"a", []int{97}, nil, 12, 12)
		if got := position.Dir(); got != c.want {
			t.Errorf("%s: Dir() = %v, want %v", c.name, got, c.want)
		}
	}
}

// TestTextPositionContains checks the overlap test the diacritic merging turns
// on.
func TestTextPositionContains(t *testing.T) {
	base := textPositionAt(10, 300, 10, 10)
	over := textPositionAt(12, 300, 4, 10)
	apart := textPositionAt(100, 300, 4, 10)

	if !base.Contains(over) {
		t.Error("a position sitting on top of another does not contain it")
	}
	if base.Contains(apart) {
		t.Error("a position far away is contained")
	}
	if !base.CompletelyContains(over) {
		t.Error("a position sitting inside another is not completely contained")
	}
	if over.CompletelyContains(base) {
		t.Error("a narrow position completely contains a wide one")
	}
}

// TestTextPositionIsDiacritic checks which characters count as diacritics.
func TestTextPositionIsDiacritic(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"́", true},   // combining acute accent, a non-spacing mark
		{"ˆ", true},   // modifier letter circumflex, a modifier symbol
		{"a", false},  // an ordinary letter
		{"ab", false}, // more than one character
		{"ー", false},  // PDFBOX-3833: the prolonged sound mark is not one
	}
	for _, c := range cases {
		matrix := util.NewMatrixOf(12, 0, 0, 12, 0, 0)
		position := text.NewTextPosition(0, 300, 400, matrix, 6, 0, 8, 6, 4,
			c.text, []int{0}, nil, 12, 12)
		if got := position.IsDiacritic(); got != c.want {
			t.Errorf("IsDiacritic(%q) = %v, want %v", c.text, got, c.want)
		}
	}
}

// TestTextPositionMergeDiacritic checks that a diacritic drawn over a letter
// ends up after it, in its combining form.
func TestTextPositionMergeDiacritic(t *testing.T) {
	matrix := util.NewMatrixOf(12, 0, 0, 12, 10, 300)
	base := text.NewTextPosition(0, 300, 400, matrix, 16, 300, 8, 6, 4,
		"a", []int{97}, nil, 12, 12)

	diacriticMatrix := util.NewMatrixOf(12, 0, 0, 12, 11, 300)
	// U+02CA, the modifier letter acute accent, which the diacritics table maps
	// onto U+0301
	diacritic := text.NewTextPosition(0, 300, 400, diacriticMatrix, 15, 300, 8, 4, 4,
		"ˊ", []int{0}, nil, 12, 12)

	base.MergeDiacritic(diacritic)
	if got := base.Unicode(); got != "á" {
		t.Errorf("merged text = %q, want %q", got, "á")
	}
}

// TestCompareTextPositions checks the order the stripper sorts in: down the
// page first, then across it.
func TestCompareTextPositions(t *testing.T) {
	// higher on the page means a smaller YDirAdj
	upper := textPositionAt(10, 300, 6, 8)
	lower := textPositionAt(10, 200, 6, 8)
	right := textPositionAt(100, 300, 6, 8)

	if text.CompareTextPositions(upper, lower) >= 0 {
		t.Error("the upper position does not sort before the lower one")
	}
	if text.CompareTextPositions(lower, upper) <= 0 {
		t.Error("the lower position does not sort after the upper one")
	}
	if text.CompareTextPositions(upper, right) >= 0 {
		t.Error("the left position does not sort before the right one on the same line")
	}
	if text.CompareTextPositions(upper, upper) != 0 {
		t.Error("a position does not compare equal to itself")
	}
}
