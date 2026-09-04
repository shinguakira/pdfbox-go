package ttf

import (
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.fontbox.ttf.TestTTFParser.
//
// testParseVertical and testParseHeaders are not ported: the first reads a font
// this repository does not carry (target/fonts/ipag00303/ipag.ttf), and the
// second reads through FontHeaders, which a later slice ports. testParseMisc is
// ported for the part of it that this slice covers; the kerning, vertical and
// GSUB tables it also asserts on are read by a later slice. See
// migration/tasks/slice-3-text-simple-fonts.md.

// TestUTCDate checks whether the creation date is UTC.
func TestUTCDate(t *testing.T) {
	// Before PDFBOX-2122, TTFDataStream was using the default TimeZone. Go has
	// no settable default zone, and the port reads the date in UTC outright;
	// the assertion below is what pins that down.
	source, err := pdfio.OpenBufferedFile(ttfFixture + "LiberationSans-Regular.ttf")
	if err != nil {
		t.Fatalf("open font: %v", err)
	}
	parser := NewParser()
	ttf, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer ttf.Close()

	header, err := ttf.Header()
	if err != nil {
		t.Fatalf("header: %v", err)
	}
	created := header.Created()
	if got := created.Location(); got != time.UTC {
		t.Errorf("created zone = %v, want %v", got, time.UTC)
	}

	target := time.Date(2010, time.June, 18, 10, 23, 22, 0, time.UTC)
	if !created.Equal(target) {
		t.Errorf("created = %v, want %v", created, target)
	}
}

// TestPostTable tests the post table parser.
func TestPostTable(t *testing.T) {
	source, err := pdfio.OpenBufferedFile(ttfFixture + "LiberationSans-Regular.ttf")
	if err != nil {
		t.Fatalf("open font: %v", err)
	}
	parser := NewParser()
	font, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer font.Close()

	cmapTable, err := font.Cmap()
	if err != nil {
		t.Fatalf("cmap: %v", err)
	}
	if cmapTable == nil {
		t.Fatal("cmap table is nil")
	}

	cmaps := cmapTable.Cmaps()
	if cmaps == nil {
		t.Fatal("cmaps is nil")
	}

	var cmap *CmapSubtable
	for _, e := range cmaps {
		if e.PlatformID() == PlatformWindows &&
			e.PlatformEncodingID() == EncodingWindowsUnicodeBMP {
			cmap = e
			break
		}
	}
	if cmap == nil {
		t.Fatal("no Windows Unicode BMP cmap subtable")
	}

	post, err := font.PostScript()
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	if post == nil {
		t.Fatal("post table is nil")
	}

	glyphNames := post.GlyphNames()
	if glyphNames == nil {
		t.Fatal("glyph names is nil")
	}

	// test a WGL4 (Macintosh standard) name
	gid := cmap.GetGlyphID(0x2122) // TRADE MARK SIGN
	if got := glyphNames[gid]; got != "trademark" {
		t.Errorf("glyph name of U+2122 = %q, want %q", got, "trademark")
	}

	// test an additional name
	gid = cmap.GetGlyphID(0x20AC) // EURO SIGN
	if got := glyphNames[gid]; got != "Euro" {
		t.Errorf("glyph name of U+20AC = %q, want %q", got, "Euro")
	}
}

// TestParseMisc is the part of the Java test of the same name that this slice
// covers.
func TestParseMisc(t *testing.T) {
	source, err := pdfio.OpenBufferedFile(ttfFixture + "LiberationSans-Regular.ttf")
	if err != nil {
		t.Fatalf("open font: %v", err)
	}
	ttf, err := NewParser().Parse(source)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	defer ttf.Close()

	unitsPerEm, err := ttf.UnitsPerEm()
	if err != nil {
		t.Fatalf("units per em: %v", err)
	}
	if unitsPerEm != 2048 {
		t.Errorf("units per em = %d, want 2048", unitsPerEm)
	}
	advanceWidth, err := ttf.AdvanceWidth(19)
	if err != nil {
		t.Fatalf("advance width: %v", err)
	}
	if advanceWidth != 1139 {
		t.Errorf("advance width of glyph 19 = %d, want 1139", advanceWidth)
	}
	name, err := ttf.Name()
	if err != nil {
		t.Fatalf("name: %v", err)
	}
	if name != "LiberationSans" {
		t.Errorf("name = %q, want %q", name, "LiberationSans")
	}
}
