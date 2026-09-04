package cmap

import (
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.fontbox.cmap.TestCMap.

// TestLookup checks whether the mapping is working correct.
func TestLookup(t *testing.T) {
	bs := []byte{200}
	cMap := newCMap()
	cMap.addCharMapping(bs, "a")
	if got, _ := cMap.ToUnicodeBytes(bs); got != "a" {
		t.Errorf("ToUnicode(%v) = %q, want %q", bs, got, "a")
	}
}

// TestPDFBox3997 covers unicode that is above the basic multilingual plane,
// here: helicopter symbol, or D83D DE81 in the Noto Emoji font.
//
// Java's build downloads the font into target/fonts; the port skips when it is
// not there.
func TestPDFBox3997(t *testing.T) {
	const path = "../../../fontbox/target/fonts/NotoEmoji-Regular.ttf"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("the font the Java build downloads is not present: %v", err)
	}
	source, err := pdfio.OpenBufferedFile(path)
	if err != nil {
		t.Fatalf("opening the font: %v", err)
	}
	font, err := ttf.NewParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing the font: %v", err)
	}
	defer font.Close()

	cmap, err := font.UnicodeCmapLookup(false)
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	if got := cmap.GetGlyphID(0x1F681); got != 886 {
		t.Errorf("GetGlyphID(0x1F681) = %d, want 886", got)
	}
}
