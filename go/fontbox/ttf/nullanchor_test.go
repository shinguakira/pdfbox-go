package ttf

// A NULL anchor is not an anchor at the origin.
//
// A mark attachment subtable holds one anchor per base glyph per mark class,
// and an offset of zero there is a NULL: this base takes no mark of that class.
// Reading it as an anchor at (0, 0) makes the subtable attach anyway, at minus
// the mark's own anchor -- which puts the mark at the far left of the letter,
// on the baseline, instead of leaving it where the advances put it.
//
// It is not a rare case. Of NotoSansArabic-Regular's 4665 base anchors, 3431
// are NULL and 6 are genuinely at the origin, so the two have to be told apart.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/table/common"
)

// TestNullBaseAnchorDoesNotAttach finds pairs the font declares no anchor for
// and checks the table leaves them alone.
func TestNullBaseAnchorDoesNotAttach(t *testing.T) {
	font := openFontFile(t, "NotoSansArabic-Regular.ttf")
	gpos, err := font.GPOS()
	if err != nil || gpos == nil {
		t.Skip("NotoSansArabic has no GPOS table")
	}
	glyphCount, err := font.NumberOfGlyphs()
	if err != nil {
		t.Fatalf("NumberOfGlyphs: %v", err)
	}

	declined, attached := gpos.anchorPairs(glyphCount)
	if len(declined) == 0 {
		t.Skip("this font declares no NULL base anchor")
	}
	if len(attached) == 0 {
		t.Fatal("this font attaches no mark at all, so nothing is being told apart")
	}

	// A pair no subtable has an anchor for must come back untouched.
	checked := 0
	for _, pair := range declined {
		if attached[pair] {
			// Another subtable does describe this pairing, so declining is not
			// the right answer for it.
			continue
		}
		positions := gpos.Position([]int{pair.base, pair.mark},
			[]string{"arab", "DFLT"}, []string{"mark", "mkmk"})
		if positions[1].AttachedTo != NotAttached {
			t.Errorf("glyph %d after glyph %d was attached, and no subtable of this "+
				"font has an anchor for the pair", pair.mark, pair.base)
		}
		if checked++; checked >= 200 {
			break
		}
	}
	if checked == 0 {
		t.Fatal("every NULL pair is described by another subtable, so nothing was checked")
	}
	t.Logf("%d pairs declared NULL, %d of them by every subtable that covers them",
		len(declined), checked)
}

// glyphPair is a base glyph and a mark glyph.
type glyphPair struct{ base, mark int }

// anchorPairs answers the pairs some mark attachment subtable declares a NULL
// anchor for, and the set of pairs some subtable does have an anchor for.
//
// A coverage table answers an index for a glyph and not a glyph for an index,
// so the glyphs are found by asking it about every glyph in the font.
func (t *GlyphPositioningTable) anchorPairs(glyphCount int) ([]glyphPair, map[glyphPair]bool) {
	var declined []glyphPair
	attached := map[glyphPair]bool{}
	for _, lookup := range t.lookupList {
		for _, subtable := range lookup.subTables {
			attachment, isAttachment := subtable.(*markAttachment)
			if !isAttachment {
				continue
			}
			bases := glyphsOfCoverage(attachment.baseCoverage, glyphCount)
			marks := glyphsOfCoverage(attachment.markCoverage, glyphCount)
			for baseIndex, base := range bases {
				if baseIndex >= len(attachment.baseAnchors) {
					continue
				}
				row := attachment.baseAnchors[baseIndex]
				for markIndex, mark := range marks {
					if markIndex >= len(attachment.marks) {
						continue
					}
					class := attachment.marks[markIndex].class
					if class >= len(row) {
						continue
					}
					pair := glyphPair{base: base, mark: mark}
					if row[class].present && attachment.marks[markIndex].anchor.present {
						attached[pair] = true
					} else {
						declined = append(declined, pair)
					}
				}
			}
		}
	}
	return declined, attached
}

// glyphsOfCoverage answers the glyph of each coverage index.
func glyphsOfCoverage(coverage common.CoverageTable, glyphCount int) map[int]int {
	glyphs := map[int]int{}
	for gid := 0; gid < glyphCount; gid++ {
		if index := coverage.CoverageIndex(gid); index >= 0 {
			glyphs[index] = gid
		}
	}
	return glyphs
}
