package ttf

import (
	"bytes"
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/util/autodetect"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.fontbox.ttf.TTFSubsetterTest.
//
// testPDFBox3379, testPDFBox5728 and testPDFBox6015 read fonts Maven downloads
// into target/fonts (DejaVuSansMono, NotoMono-Regular, Keyboard); this
// repository does not carry them and the port skips those tests where the file
// is absent, the way testPDFBox3319 skips a missing SimHei.

// openFullFont parses a font file, failing the test where it cannot.
func openFullFont(t *testing.T, path string) *TrueTypeFont {
	t.Helper()
	source, err := pdfio.OpenBufferedFile(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	font, err := NewParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing %s: %v", path, err)
	}
	return font
}

// openDownloadedFont parses one of the fonts Maven downloads, skipping the test
// where it is not there.
func openDownloadedFont(t *testing.T, name string) *TrueTypeFont {
	t.Helper()
	path := "../../../fontbox/target/fonts/" + name
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s is not in target/fonts, test skipped", name)
	}
	return openFullFont(t, path)
}

// parseSubset parses the bytes the subsetter wrote, as an embedded font.
func parseSubset(t *testing.T, subsetBytes []byte, embedded bool) *TrueTypeFont {
	t.Helper()
	parser := NewParser()
	if embedded {
		parser = NewParserEmbedded(true)
	}
	subset, err := parser.Parse(pdfio.NewReadBufferBytes(subsetBytes))
	if err != nil {
		t.Fatalf("parsing the subset: %v", err)
	}
	return subset
}

func subsetBytes(t *testing.T, subsetter *TTFSubsetter) []byte {
	t.Helper()
	var baos bytes.Buffer
	if err := subsetter.WriteToStream(&baos); err != nil {
		t.Fatalf("writing the subset: %v", err)
	}
	return baos.Bytes()
}

func numberOfGlyphs(t *testing.T, font *TrueTypeFont) int {
	t.Helper()
	n, err := font.NumberOfGlyphs()
	if err != nil {
		t.Fatalf("NumberOfGlyphs: %v", err)
	}
	return n
}

func nameToGID(t *testing.T, font *TrueTypeFont, name string) int {
	t.Helper()
	gid, err := font.NameToGID(name)
	if err != nil {
		t.Fatalf("NameToGID(%q): %v", name, err)
	}
	return gid
}

func glyphOf(t *testing.T, font *TrueTypeFont, gid int) *GlyphData {
	t.Helper()
	glyphTable, err := font.Glyph()
	if err != nil {
		t.Fatalf("Glyph: %v", err)
	}
	glyph, err := glyphTable.GetGlyph(gid)
	if err != nil {
		t.Fatalf("GetGlyph(%d): %v", gid, err)
	}
	return glyph
}

func advanceWidthOf(t *testing.T, font *TrueTypeFont, gid int) int {
	t.Helper()
	width, err := font.AdvanceWidth(gid)
	if err != nil {
		t.Fatalf("AdvanceWidth(%d): %v", gid, err)
	}
	return width
}

func leftSideBearingOf(t *testing.T, font *TrueTypeFont, gid int) int16 {
	t.Helper()
	metrics, err := font.HorizontalMetrics()
	if err != nil {
		t.Fatalf("HorizontalMetrics: %v", err)
	}
	return metrics.LeftSideBearing(gid)
}

func postScriptOf(t *testing.T, font *TrueTypeFont) *PostScriptTable {
	t.Helper()
	post, err := font.PostScript()
	if err != nil {
		t.Fatalf("PostScript: %v", err)
	}
	return post
}

// pathIsEmpty is Java's font.getPath(name).getBounds2D().isEmpty().
func pathIsEmpty(t *testing.T, font *TrueTypeFont, name string) bool {
	t.Helper()
	path, err := font.GetPath(name)
	if err != nil {
		t.Fatalf("GetPath(%q): %v", name, err)
	}
	return path.Bounds2D().IsEmpty()
}

func widthOf(t *testing.T, font *TrueTypeFont, name string) float32 {
	t.Helper()
	width, err := font.GetWidth(name)
	if err != nil {
		t.Fatalf("GetWidth(%q): %v", name, err)
	}
	return width
}

// TestEmptySubset is PDFBOX-2854: empty subset with all tables.
func TestEmptySubset(t *testing.T) {
	x := openFullFont(t, ttfFixture+"LiberationSans-Regular.ttf")
	ttfSubsetter, err := NewTTFSubsetter(x)
	if err != nil {
		t.Fatalf("NewTTFSubsetter: %v", err)
	}

	subset := parseSubset(t, subsetBytes(t, ttfSubsetter), true)
	defer subset.Close()

	if got := numberOfGlyphs(t, subset); got != 1 {
		t.Errorf("NumberOfGlyphs = %d, want 1", got)
	}
	if got := nameToGID(t, subset, ".notdef"); got != 0 {
		t.Errorf("NameToGID(.notdef) = %d, want 0", got)
	}
	if glyphOf(t, subset, 0) == nil {
		t.Error("glyph 0 is missing")
	}
}

// TestEmptySubset2 is PDFBOX-2854: empty subset with selected tables.
func TestEmptySubset2(t *testing.T) {
	x := openFullFont(t, ttfFixture+"LiberationSans-Regular.ttf")
	// List copied from TrueTypeEmbedder.java
	tables := []string{"head", "hhea", "loca", "maxp", "cvt ", "prep", "glyf", "hmtx", "fpgm",
		"gasp"}
	ttfSubsetter, err := NewTTFSubsetterTables(x, tables)
	if err != nil {
		t.Fatalf("NewTTFSubsetterTables: %v", err)
	}

	subset := parseSubset(t, subsetBytes(t, ttfSubsetter), true)
	defer subset.Close()

	if got := numberOfGlyphs(t, subset); got != 1 {
		t.Errorf("NumberOfGlyphs = %d, want 1", got)
	}
	if got := nameToGID(t, subset, ".notdef"); got != 0 {
		t.Errorf("NameToGID(.notdef) = %d, want 0", got)
	}
	if glyphOf(t, subset, 0) == nil {
		t.Error("glyph 0 is missing")
	}
}

// TestNonEmptySubset is PDFBOX-2854: subset with one glyph.
func TestNonEmptySubset(t *testing.T) {
	full := openFullFont(t, ttfFixture+"LiberationSans-Regular.ttf")
	ttfSubsetter, err := NewTTFSubsetter(full)
	if err != nil {
		t.Fatalf("NewTTFSubsetter: %v", err)
	}
	ttfSubsetter.Add('a')

	subset := parseSubset(t, subsetBytes(t, ttfSubsetter), true)
	defer subset.Close()

	if got := numberOfGlyphs(t, subset); got != 2 {
		t.Errorf("NumberOfGlyphs = %d, want 2", got)
	}
	if got := nameToGID(t, subset, ".notdef"); got != 0 {
		t.Errorf("NameToGID(.notdef) = %d, want 0", got)
	}
	if got := nameToGID(t, subset, "a"); got != 1 {
		t.Errorf("NameToGID(a) = %d, want 1", got)
	}
	if glyphOf(t, subset, 0) == nil {
		t.Error("glyph 0 is missing")
	}
	if glyphOf(t, subset, 1) == nil {
		t.Error("glyph 1 is missing")
	}
	if glyphOf(t, subset, 2) != nil {
		t.Error("glyph 2 should be missing")
	}
	wantWidth := advanceWidthOf(t, full, nameToGID(t, full, "a"))
	if got := advanceWidthOf(t, subset, nameToGID(t, subset, "a")); got != wantWidth {
		t.Errorf("AdvanceWidth(a) = %d, want %d", got, wantWidth)
	}
	wantLSB := leftSideBearingOf(t, full, nameToGID(t, full, "a"))
	if got := leftSideBearingOf(t, subset, nameToGID(t, subset, "a")); got != wantLSB {
		t.Errorf("LeftSideBearing(a) = %d, want %d", got, wantLSB)
	}
}

// TestPDFBox3319 checks that widths and left side bearings in a partially
// monospaced font are kept.
func TestPDFBox3319(t *testing.T) {
	t.Log("Searching for SimHei font...")
	var simhei string
	for _, path := range autodetect.NewFontFileFinder().Find() {
		if len(path) >= 10 && equalFoldASCII(path[len(path)-10:], "simhei.ttf") {
			simhei = path
			break
		}
	}
	if simhei == "" {
		t.Skip("SimHei font not available on this machine, test skipped")
	}
	t.Log("SimHei font found!")
	full := openFullFont(t, simhei)

	// List copied from TrueTypeEmbedder.java
	// Without it, the test would fail because of missing post table in source font
	tables := []string{"head", "hhea", "loca", "maxp", "cvt ", "prep", "glyf", "hmtx", "fpgm",
		"gasp"}

	ttfSubsetter, err := NewTTFSubsetterTables(full, tables)
	if err != nil {
		t.Fatalf("NewTTFSubsetterTables: %v", err)
	}

	for _, codePoint := range "中国你好!" {
		ttfSubsetter.Add(int(codePoint))
	}

	written := subsetBytes(t, ttfSubsetter)
	gidMap, err := ttfSubsetter.GetGIDMap()
	if err != nil {
		t.Fatalf("GetGIDMap: %v", err)
	}
	subset := parseSubset(t, written, true)
	defer subset.Close()

	if got := numberOfGlyphs(t, subset); got != 6 {
		t.Errorf("NumberOfGlyphs = %d, want 6", got)
	}

	for newGID, oldGID := range gidMap {
		if got, want := advanceWidthOf(t, subset, newGID), advanceWidthOf(t, full, oldGID); got != want {
			t.Errorf("AdvanceWidth(%d) = %d, want %d", newGID, got, want)
		}
		if got, want := leftSideBearingOf(t, subset, newGID),
			leftSideBearingOf(t, full, oldGID); got != want {
			t.Errorf("LeftSideBearing(%d) = %d, want %d", newGID, got, want)
		}
	}
}

// equalFoldASCII is Java's toLowerCase(Locale.US) comparison of a file suffix.
func equalFoldASCII(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		x, y := a[i], b[i]
		if 'A' <= x && x <= 'Z' {
			x += 'a' - 'A'
		}
		if 'A' <= y && y <= 'Z' {
			y += 'a' - 'A'
		}
		if x != y {
			return false
		}
	}
	return true
}

// TestPDFBox3379 checks that left side bearings in a partially monospaced font
// are kept.
func TestPDFBox3379(t *testing.T) {
	full := openDownloadedFont(t, "DejaVuSansMono.ttf")
	ttfSubsetter, err := NewTTFSubsetter(full)
	if err != nil {
		t.Fatalf("NewTTFSubsetter: %v", err)
	}
	ttfSubsetter.Add('A')
	ttfSubsetter.Add(' ')
	ttfSubsetter.Add('B')

	subset := parseSubset(t, subsetBytes(t, ttfSubsetter), false)
	defer subset.Close()

	if got := numberOfGlyphs(t, subset); got != 4 {
		t.Errorf("NumberOfGlyphs = %d, want 4", got)
	}
	for name, want := range map[string]int{".notdef": 0, "space": 1, "A": 2, "B": 3} {
		if got := nameToGID(t, subset, name); got != want {
			t.Errorf("NameToGID(%q) = %d, want %d", name, got, want)
		}
	}
	for _, name := range []string{"A", "B", "space"} {
		wantWidth := advanceWidthOf(t, full, nameToGID(t, full, name))
		if got := advanceWidthOf(t, subset, nameToGID(t, subset, name)); got != wantWidth {
			t.Errorf("AdvanceWidth(%q) = %d, want %d", name, got, wantWidth)
		}
		wantLSB := leftSideBearingOf(t, full, nameToGID(t, full, name))
		if got := leftSideBearingOf(t, subset, nameToGID(t, subset, name)); got != wantLSB {
			t.Errorf("LeftSideBearing(%q) = %d, want %d", name, got, wantLSB)
		}
	}
}

// TestPDFBox3757 is PDFBOX-3757: check that PostScript names that are not part
// of WGL4Names don't get shuffled in buildPostTable.
func TestPDFBox3757(t *testing.T) {
	font := openFullFont(t, ttfFixture+"LiberationSans-Regular.ttf")
	ttfSubsetter, err := NewTTFSubsetter(font)
	if err != nil {
		t.Fatalf("NewTTFSubsetter: %v", err)
	}
	ttfSubsetter.Add('Ö')
	ttfSubsetter.Add(' ')

	subset := parseSubset(t, subsetBytes(t, ttfSubsetter), true)
	defer subset.Close()

	if got := numberOfGlyphs(t, subset); got != 5 {
		t.Errorf("NumberOfGlyphs = %d, want 5", got)
	}

	wantGIDs := []struct {
		name string
		gid  int
	}{
		{".notdef", 0}, {"O", 1}, {"Odieresis", 2}, {"uni200A", 3}, {"dieresis.uc", 4},
	}
	for _, want := range wantGIDs {
		if got := nameToGID(t, subset, want.name); got != want.gid {
			t.Errorf("NameToGID(%q) = %d, want %d", want.name, got, want.gid)
		}
	}

	pst := postScriptOf(t, subset)
	for _, want := range wantGIDs {
		if got := pst.GetName(want.gid); got != want.name {
			t.Errorf("post name %d = %q, want %q", want.gid, got, want.name)
		}
	}

	if !pathIsEmpty(t, subset, "uni200A") {
		t.Error("Hair space path should be empty")
	}
	if pathIsEmpty(t, subset, "dieresis.uc") {
		t.Error("UC dieresis path should not be empty")
	}
}

// TestPDFBox5728 checks a font with a v3 PostScript table format and no glyph
// names.
func TestPDFBox5728(t *testing.T) {
	font := openDownloadedFont(t, "NotoMono-Regular.ttf")
	defer font.Close()
	postScript := postScriptOf(t, font)
	if got := postScript.FormatType(); got != 3.0 {
		t.Errorf("FormatType = %v, want 3.0", got)
	}
	if postScript.GlyphNames() != nil {
		t.Error("GlyphNames should be nil")
	}
	subsetter, err := NewTTFSubsetter(font)
	if err != nil {
		t.Fatalf("NewTTFSubsetter: %v", err)
	}
	subsetter.Add('a')
	var output bytes.Buffer
	if err := subsetter.WriteToStream(&output); err != nil {
		t.Fatalf("WriteToStream: %v", err)
	}
}

// TestPDFBox5230 is PDFBOX-5230: check that subsetting can be forced to use
// invisible glyphs.
func TestPDFBox5230(t *testing.T) {
	font := openFullFont(t, ttfFixture+"LiberationSans-Regular.ttf")
	ttfSubsetter, err := NewTTFSubsetter(font)
	if err != nil {
		t.Fatalf("NewTTFSubsetter: %v", err)
	}
	ttfSubsetter.Add('A')
	ttfSubsetter.Add('B')
	ttfSubsetter.Add(0x200C) // zero width non-joiner

	wantGIDs := []struct {
		name string
		gid  int
	}{
		{".notdef", 0}, {"A", 1}, {"B", 2}, {"uni200C", 3},
	}

	// verify results without forcing

	func() {
		subset := parseSubset(t, subsetBytes(t, ttfSubsetter), true)
		defer subset.Close()

		if got := numberOfGlyphs(t, subset); got != 4 {
			t.Errorf("NumberOfGlyphs = %d, want 4", got)
		}
		for _, want := range wantGIDs {
			if got := nameToGID(t, subset, want.name); got != want.gid {
				t.Errorf("NameToGID(%q) = %d, want %d", want.name, got, want.gid)
			}
		}

		pst := postScriptOf(t, subset)
		for _, want := range wantGIDs {
			if got := pst.GetName(want.gid); got != want.name {
				t.Errorf("post name %d = %q, want %q", want.gid, got, want.name)
			}
		}

		if pathIsEmpty(t, subset, "A") {
			t.Error("A path should not be empty")
		}
		if pathIsEmpty(t, subset, "B") {
			t.Error("B path should not be empty")
		}
		if pathIsEmpty(t, subset, "uni200C") {
			t.Error("ZWNJ path should not be empty")
		}
		if widthOf(t, subset, "A") == 0 {
			t.Error("A width should not be zero.")
		}
		if widthOf(t, subset, "B") == 0 {
			t.Error("B width should not be zero.")
		}
		if got := widthOf(t, subset, "uni200C"); got != 0 {
			t.Errorf("ZWNJ width should be zero, got %v", got)
		}
	}()

	// verify results while forcing B and ZWNJ to use invisible glyphs

	ttfSubsetter.ForceInvisible('B')
	ttfSubsetter.ForceInvisible(0x200C)

	subset := parseSubset(t, subsetBytes(t, ttfSubsetter), true)
	defer subset.Close()

	if got := numberOfGlyphs(t, subset); got != 4 {
		t.Errorf("NumberOfGlyphs = %d, want 4", got)
	}
	for _, want := range wantGIDs {
		if got := nameToGID(t, subset, want.name); got != want.gid {
			t.Errorf("NameToGID(%q) = %d, want %d", want.name, got, want.gid)
		}
	}

	pst := postScriptOf(t, subset)
	for _, want := range wantGIDs {
		if got := pst.GetName(want.gid); got != want.name {
			t.Errorf("post name %d = %q, want %q", want.gid, got, want.name)
		}
	}

	if pathIsEmpty(t, subset, "A") {
		t.Error("A path should not be empty")
	}
	if !pathIsEmpty(t, subset, "B") {
		t.Error("B path should be empty")
	}
	if !pathIsEmpty(t, subset, "uni200C") {
		t.Error("ZWNJ path should be empty")
	}
	if widthOf(t, subset, "A") == 0 {
		t.Error("A width should not be zero.")
	}
	if got := widthOf(t, subset, "B"); got != 0 {
		t.Errorf("B width should be zero, got %v", got)
	}
	if got := widthOf(t, subset, "uni200C"); got != 0 {
		t.Errorf("ZWNJ width should be zero, got %v", got)
	}
}

// TestPDFBox6015 checks a font with a 0/1 cmap.
func TestPDFBox6015(t *testing.T) {
	font := openDownloadedFont(t, "Keyboard.ttf")
	defer font.Close()
	unicodeCmapLookup, err := font.UnicodeCmapLookupStrict()
	if err != nil {
		t.Fatalf("UnicodeCmapLookupStrict: %v", err)
	}
	cases := []struct {
		codePoint int
		gid       int
	}{
		{'a', 185}, {'z', 210}, {'A', 159}, {'Z', 184}, {'0', 49}, {'9', 58},
	}
	for _, c := range cases {
		if got := unicodeCmapLookup.GetGlyphID(c.codePoint); got != c.gid {
			t.Errorf("GetGlyphID(%q) = %d, want %d", rune(c.codePoint), got, c.gid)
		}
	}
}
