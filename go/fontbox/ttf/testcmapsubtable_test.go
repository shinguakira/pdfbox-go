package ttf

// Port of org.apache.fontbox.ttf.TestCMapSubtable.
//
// Both cases read a font the Maven build downloads into `target/fonts`, which
// is why neither was ported until now. `migration/scripts/fetch-testdata.ps1`
// fills it. Every expected value is the Java's.

import (
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

const (
	notoSansSC = "../../../fontbox/target/fonts/NotoSansSC-Regular.otf"
	ipaGothic  = "../../../pdfbox/target/fonts/ipag00303/ipag.ttf"
)

// openDownloadedRead opens one of the downloaded fonts as a source, skipping
// where the fetch has not run. ttfsubsetter_test.go has openDownloadedFont for
// the parsed form; these two cases need the raw source, and one of them is
// under pdfbox/target rather than fontbox/target.
func openDownloadedRead(t *testing.T, path string) pdfio.RandomAccessRead {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s is not there; run migration/scripts/fetch-testdata.ps1", path)
	}
	source, err := pdfio.OpenBufferedFile(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	t.Cleanup(func() { source.Close() })
	return source
}

// TestPDFBox5328 is testPDFBox5328: one glyph reachable from two character
// codes must answer both of them, and must answer the same pair through the
// (3, 1) BMP subtable and the (3, 10) full one.
func TestPDFBox5328(t *testing.T) {
	source := openDownloadedRead(t, notoSansSC)
	otf, err := NewOTFParserEmbedded(false).Parse(source)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	defer otf.Close()

	const gid = 8712
	want := []int{19981, 63847}

	lookup, err := otf.UnicodeCmapLookupStrict()
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	assertCharCodes(t, "the unicode lookup", lookup.GetCharCodes(gid), want)

	cmapTable, err := otf.Cmap()
	if err != nil {
		t.Fatalf("Cmap: %v", err)
	}
	full := cmapTable.GetSubtable(CmapPlatformUnicode, CmapEncodingUnicode20Full)
	bmp := cmapTable.GetSubtable(CmapPlatformUnicode, CmapEncodingUnicode20BMP)
	if full == nil || bmp == nil {
		t.Fatalf("the font has no (0,%d) or (0,%d) subtable",
			CmapEncodingUnicode20Full, CmapEncodingUnicode20BMP)
	}
	assertCharCodes(t, "the BMP subtable", bmp.GetCharCodes(gid), want)
	assertCharCodes(t, "the full subtable", full.GetCharCodes(gid), want)
}

func assertCharCodes(t *testing.T, what string, got, want []int) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s answers %v, want %v", what, got, want)
		return
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%s answers %v, want %v", what, got, want)
			return
		}
	}
}

// TestVerticalSubstitution is testVerticalSubstitution: turning vertical
// substitution on changes which glyph a character reaches.
//
// 「 and 」 are the demonstration, because their vertical forms are rotated
// rather than merely repositioned.
func TestVerticalSubstitution(t *testing.T) {
	source := openDownloadedRead(t, ipaGothic)
	font, err := NewParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	defer font.Close()

	horizontal, err := font.UnicodeCmapLookupStrict()
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	hgid1 := horizontal.GetGlyphID('「')
	hgid2 := horizontal.GetGlyphID('」')

	font.EnableVerticalSubstitutions()
	vertical, err := font.UnicodeCmapLookupStrict()
	if err != nil {
		t.Fatalf("UnicodeCmapLookup after enabling: %v", err)
	}
	vgid1 := vertical.GetGlyphID('「')
	vgid2 := vertical.GetGlyphID('」')

	for _, c := range []struct {
		what string
		got  int
		want int
	}{
		{"「 horizontal", hgid1, 441},
		{"」 horizontal", hgid2, 442},
		{"「 vertical", vgid1, 7392},
		{"」 vertical", vgid2, 7393},
	} {
		if c.got != c.want {
			t.Errorf("%s is glyph %d, want %d", c.what, c.got, c.want)
		}
	}
}
