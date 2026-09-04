package gsub_test

import (
	"os"
	"runtime"
	"slices"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/gsub"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.fontbox.ttf.gsub.GsubWorkerForLatinTest.

const ttfFixture = "../../../../fontbox/src/test/resources/"

// TestApplyLigaturesCalibri covers the ligatures of the Calibri that ships with
// Windows.
func TestApplyLigaturesCalibri(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("calibri ligature test skipped")
	}
	const path = "c:/windows/fonts/calibri.ttf"
	if _, err := os.Stat(path); err != nil {
		t.Skip("calibri ligature test skipped")
	}
	source, err := pdfio.OpenBufferedFile(path)
	if err != nil {
		t.Fatalf("opening the font: %v", err)
	}
	font, err := ttf.NewParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing the font: %v", err)
	}
	cmapLookup, gsubWorkerForLatin := workerFor(t, font)
	font.Close()

	cases := []struct {
		word string
		want []int
	}{
		{"effective", []int{286, 299, 286, 272, 415, 448, 286}},
		{"attitude", []int{258, 427, 410, 437, 282, 286}},
		{"affiliate", []int{258, 312, 367, 349, 258, 410, 286}},
		{"film", []int{302, 367, 373}},
		{"float", []int{327, 381, 258, 410}},
		{"platform", []int{393, 367, 258, 414, 381, 396, 373}},
	}
	for _, c := range cases {
		got := gsubWorkerForLatin.ApplyTransforms(glyphIDs(t, c.word, cmapLookup))
		if !slices.Equal(got, c.want) {
			t.Errorf("ApplyTransforms(%q) = %v, want %v", c.word, got, c.want)
		}
	}
}

// TestApplyLigaturesFoglihtenNo07 covers the ligatures of the OTF in the Java
// test resources.
func TestApplyLigaturesFoglihtenNo07(t *testing.T) {
	source, err := pdfio.OpenBufferedFile(ttfFixture + "otf/FoglihtenNo07.otf")
	if err != nil {
		t.Fatalf("opening the font: %v", err)
	}
	font, err := ttf.NewOTFParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing the font: %v", err)
	}
	cmapLookup, gsubWorkerForLatin := workerFor(t, font.TrueTypeFont)
	font.Close()

	cases := []struct {
		word string
		want []int
	}{
		{"affine", []int{66, 1590, 645, 70}},
		{"attitude", []int{538, 633, 85, 86, 69, 70}},
		{"affiliate", []int{66, 1590, 525, 74, 683}},
		{"The film", []int{542, 1, 1591, 498}},
		{"The Last", []int{542, 1, 45, 703, 85}},
		{"platform", []int{81, 77, 538, 71, 80, 83, 78}},
	}
	for _, c := range cases {
		got := gsubWorkerForLatin.ApplyTransforms(glyphIDs(t, c.word, cmapLookup))
		if !slices.Equal(got, c.want) {
			t.Errorf("ApplyTransforms(%q) = %v, want %v", c.word, got, c.want)
		}
	}
}

func workerFor(t *testing.T, font *ttf.TrueTypeFont) (ttf.CmapLookup, gsub.GsubWorker) {
	t.Helper()
	cmapLookup, err := font.UnicodeCmapLookupStrict()
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	gsubData, err := font.GsubData()
	if err != nil {
		t.Fatalf("GsubData: %v", err)
	}
	return cmapLookup, gsub.NewFactory().GetGsubWorker(cmapLookup, gsubData)
}

func glyphIDs(t *testing.T, word string, cmapLookup ttf.CmapLookup) []int {
	t.Helper()
	var originalGlyphIDs []int
	for _, unicodeChar := range word {
		glyphID := cmapLookup.GetGlyphID(int(unicodeChar))
		if glyphID <= 0 {
			t.Fatalf("no glyph for %q in %q", unicodeChar, word)
		}
		originalGlyphIDs = append(originalGlyphIDs, glyphID)
	}
	return originalGlyphIDs
}
