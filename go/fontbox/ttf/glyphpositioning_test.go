package ttf

// The GPOS reader, against the fonts the pdfbox-layout tests use.
//
// There is no Java to measure against here: PDFBox has no GPOS reader, which is
// why this one exists. What the cases assert instead is the table's own
// meaning -- that a kern pair the font declares comes back with the sign and
// size the font declares, and that a font with no GPOS answers nothing.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// layoutFonts is where the pdfbox-layout-awt test resources are.
const layoutFonts = "../../../pdfbox-layout-awt/src/test/resources/ttf/"

// openFontFile parses a font from the layout test resources.
func openFontFile(t *testing.T, name string) *TrueTypeFont {
	t.Helper()
	source, err := pdfio.OpenBufferedFile(layoutFonts + name)
	if err != nil {
		t.Skipf("%s is not in this repository: %v", name, err)
	}
	font, err := NewOTFParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	t.Cleanup(func() { font.Close() })
	return font.TrueTypeFont
}

// glyphsOf answers the glyph ids of a string, through the font's Unicode cmap.
func glyphsOf(t *testing.T, font *TrueTypeFont, text string) []int {
	t.Helper()
	cmap, err := font.UnicodeCmapLookup(true)
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	glyphs := make([]int, 0, len(text))
	for _, r := range text {
		gid := cmap.GetGlyphID(int(r))
		if gid == 0 {
			t.Fatalf("the font has no glyph for %q", r)
		}
		glyphs = append(glyphs, gid)
	}
	return glyphs
}

// TestGPOSIsRead checks the table parses at all, in the fonts that carry one.
func TestGPOSIsRead(t *testing.T) {
	for _, name := range []string{
		"DejaVuSans.ttf", "FiraCode-Regular.ttf", "Arimo-Regular.ttf",
		"NotoSansArabic-Regular.ttf", "NotoSansThai-Regular.ttf",
	} {
		t.Run(name, func(t *testing.T) {
			font := openFontFile(t, name)
			gpos, err := font.GPOS()
			if err != nil {
				t.Fatalf("GPOS: %v", err)
			}
			if gpos == nil {
				t.Skip("this font has no GPOS table")
			}
			if len(gpos.ScriptTags()) == 0 {
				t.Error("the table carries no script")
			}
			if len(gpos.lookupList) == 0 {
				t.Error("the table carries no lookup")
			}
		})
	}
}

// TestKerningPullsAVTogether is the case kerning exists for: in a font with a
// kern pair for A and V, the pair comes back with a negative x advance, and a
// pair the font does not kern comes back with none.
func TestKerningPullsAVTogether(t *testing.T) {
	font := openFontFile(t, "DejaVuSans.ttf")
	gpos, err := font.GPOS()
	if err != nil {
		t.Fatalf("GPOS: %v", err)
	}
	if gpos == nil {
		t.Skip("DejaVuSans has no GPOS table")
	}

	pair := glyphsOf(t, font, "AV")
	kerned := gpos.Position(pair, []string{"latn"}, []string{"kern"})
	if len(kerned) != 2 {
		t.Fatalf("Position gave %d results for two glyphs", len(kerned))
	}
	if kerned[0].XAdvance >= 0 {
		t.Errorf("A before V adjusted the advance by %d; kerning pulls them together, "+
			"so it has to be negative", kerned[0].XAdvance)
	}

	// "II" is not a kern pair in any font this port has seen.
	unkerned := glyphsOf(t, font, "II")
	plain := gpos.Position(unkerned,
		[]string{"latn"}, []string{"kern"})
	for i, position := range plain {
		if !position.IsZero() {
			t.Errorf("II glyph %d was adjusted by %+v; the font declares no pair for it",
				i, position)
		}
	}
}

// TestFeatureMustBeAskedFor checks a lookup only runs for a feature the caller
// turned on, which is what the -kerning option of the layout tests controls.
func TestFeatureMustBeAskedFor(t *testing.T) {
	font := openFontFile(t, "DejaVuSans.ttf")
	gpos, err := font.GPOS()
	if err != nil || gpos == nil {
		t.Skip("DejaVuSans has no GPOS table")
	}
	glyphs := glyphsOf(t, font, "AV")

	withKerning := gpos.Position(glyphs,
		[]string{"latn"}, []string{"kern"})
	without := gpos.Position(glyphs, []string{"latn"}, nil)

	if withKerning[0].IsZero() {
		t.Fatal("with kern asked for, nothing happened")
	}
	for i, position := range without {
		if !position.IsZero() {
			t.Errorf("with no feature asked for, glyph %d moved by %+v", i, position)
		}
	}
}

// TestPositionIsStableAndSized checks the two properties every caller depends
// on: one result per glyph, and nothing for an empty run.
func TestPositionIsStableAndSized(t *testing.T) {
	font := openFontFile(t, "DejaVuSans.ttf")
	gpos, err := font.GPOS()
	if err != nil || gpos == nil {
		t.Skip("DejaVuSans has no GPOS table")
	}

	if got := gpos.Position(nil, []string{"latn"}, []string{"kern"}); len(got) != 0 {
		t.Errorf("an empty run gave %d results", len(got))
	}

	glyphs := glyphsOf(t, font, "AVATAR")
	first := gpos.Position(glyphs,
		[]string{"latn"}, []string{"kern"})
	if len(first) != len(glyphs) {
		t.Fatalf("Position gave %d results for %d glyphs", len(first), len(glyphs))
	}
	// Running it twice must answer the same thing: the table holds no state.
	second := gpos.Position(glyphs,
		[]string{"latn"}, []string{"kern"})
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("glyph %d gave %+v then %+v", i, first[i], second[i])
		}
	}
}

// TestArabicMarksAttach checks lookup types 4 and 6: a vowel mark is placed
// relative to the letter before it rather than after it.
func TestArabicMarksAttach(t *testing.T) {
	font := openFontFile(t, "NotoSansArabic-Regular.ttf")
	gpos, err := font.GPOS()
	if err != nil || gpos == nil {
		t.Skip("NotoSansArabic has no GPOS table")
	}

	// U+0644 ARABIC LETTER LAM followed by U+064E ARABIC FATHA, a vowel mark
	// that sits above the letter.
	glyphs := glyphsOf(t, font, "لَ")
	positions := gpos.Position(glyphs,
		[]string{"arab"}, []string{"mark", "mkmk"})
	if len(positions) != 2 {
		t.Fatalf("Position gave %d results for two glyphs", len(positions))
	}
	if positions[1].IsZero() {
		t.Error("the fatha was not moved; a mark attaches to the letter before it")
	}
	if positions[1].YPlacement <= 0 {
		t.Errorf("the fatha was placed at y %d; it sits above the letter",
			positions[1].YPlacement)
	}
}

// TestGPOSKerningAgreesWithTheKernTable is the independent check on the
// numbers.
//
// There is no Java GPOS to measure against, but a font that carries both the
// old `kern` table and a GPOS `kern` feature is saying the same thing twice,
// and fontbox reads the old table already. Where both are present the values
// should agree.
//
// They are allowed to differ: `kern` is the older mechanism, GPOS can say more,
// and a font may update one and not the other. What the case asserts is that
// they agree wherever the old table has a pair at all -- a disagreement means
// the GPOS reader is reading the wrong bytes, not that the font is subtle.
func TestGPOSKerningAgreesWithTheKernTable(t *testing.T) {
	for _, name := range []string{
		"DejaVuSans.ttf", "Arimo-Regular.ttf", "FiraCode-Regular.ttf",
	} {
		t.Run(name, func(t *testing.T) {
			font := openFontFile(t, name)
			gpos, err := font.GPOS()
			if err != nil || gpos == nil {
				t.Skip("no GPOS table")
			}
			kerning, err := font.Kerning()
			if err != nil || kerning == nil {
				t.Skip("no kern table")
			}
			subtable := kerning.HorizontalKerningSubtable()
			if subtable == nil {
				t.Skip("no horizontal kern subtable")
			}

			// Every two-letter pair of a sample alphabet, which is where a
			// Latin font puts its kerning.
			const alphabet = "AVWTYLoacdefgijklmnprstuvwxyz.,"
			checked, agreed := 0, 0
			for _, left := range alphabet {
				for _, right := range alphabet {
					glyphs := glyphsOf(t, font, string(left)+string(right))
					fromKern := subtable.KerningPair(glyphs[0], glyphs[1])
					if fromKern == 0 {
						continue
					}
					checked++
					fromGPOS := gpos.Position(glyphs, []string{"latn"},
						[]string{"kern"})[0].XAdvance
					if fromGPOS == fromKern {
						agreed++
						continue
					}
					if checked-agreed <= 3 {
						t.Errorf("%q%q: kern says %d, GPOS says %d",
							left, right, fromKern, fromGPOS)
					}
				}
			}
			if checked == 0 {
				t.Skip("the kern table has no pair in the sample alphabet")
			}
			t.Logf("%d pairs in the kern table, %d agree with GPOS", checked, agreed)
			if agreed*10 < checked*9 {
				t.Errorf("only %d of %d pairs agree; the GPOS reader is reading "+
					"the wrong bytes", agreed, checked)
			}
		})
	}
}
