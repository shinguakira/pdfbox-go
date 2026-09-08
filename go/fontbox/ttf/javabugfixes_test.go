package ttf_test

// The `fontbox/ttf` half of `track/java-bug-fixes`.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
)

// TestNameToGIDOfAUniNameWithoutACmap is JAVA-BUGS 12.
//
// `nameToGID` asks for the Unicode cmap without the strict flag, which answers
// null for a font that has none — a subsetted font usually has none — and then
// calls `getGlyphId` on it. The method answers 0 for every other kind of name
// it cannot resolve, three lines below and three lines above; this is the one
// path that dereferences instead.
func TestNameToGIDOfAUniNameWithoutACmap(t *testing.T) {
	font := ttf.NewTrueTypeFont(nil)

	// No cmap table at all, so UnicodeCmapLookup answers nil.
	gid, err := font.NameToGID("uni0041")
	if err != nil {
		t.Fatalf("NameToGID: %v", err)
	}
	if gid != 0 {
		t.Errorf("NameToGID(\"uni0041\") is %d, want 0: the font has no cmap", gid)
	}
}
