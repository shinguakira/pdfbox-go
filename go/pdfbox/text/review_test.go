package text_test

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/util"
)

// The tests below pin the four defects the slice 3 adversarial review found.
// Each fails without its fix.

// TestDiacriticOutsideTheBasicPlaneIsNotMerged pins the first: Java measures a
// string in UTF-16 code units, so a character outside the basic plane has
// length 2 and mergeDiacritic returns without doing anything. Counting runes
// would give 1 and merge it.
func TestDiacriticOutsideTheBasicPlaneIsNotMerged(t *testing.T) {
	matrix := util.NewMatrixOf(12, 0, 0, 12, 10, 300)
	base := text.NewTextPosition(0, 300, 400, matrix, 16, 300, 8, 6, 4,
		"a", []int{97}, nil, 12, 12)

	diacriticMatrix := util.NewMatrixOf(12, 0, 0, 12, 11, 300)
	// U+1D165, MUSICAL SYMBOL COMBINING STEM: one rune, two UTF-16 units
	diacritic := text.NewTextPosition(0, 300, 400, diacriticMatrix, 15, 300, 8, 4, 4,
		"\U0001D165", []int{0}, nil, 12, 12)

	base.MergeDiacritic(diacritic)
	if got := base.Unicode(); got != "a" {
		t.Errorf("merged text = %q, want %q: a diacritic of UTF-16 length 2 is left alone",
			got, "a")
	}
}

// TestIsDiacriticOutsideTheBasicPlane pins the same rule in isDiacritic: Java's
// length check rejects anything but a single UTF-16 unit.
func TestIsDiacriticOutsideTheBasicPlane(t *testing.T) {
	matrix := util.NewMatrixOf(12, 0, 0, 12, 0, 0)
	// U+1D167, a combining mark outside the basic plane
	position := text.NewTextPosition(0, 300, 400, matrix, 6, 0, 8, 6, 4,
		"\U0001D167", []int{0}, nil, 12, 12)
	if position.IsDiacritic() {
		t.Error("a mark of UTF-16 length 2 is reported as a diacritic")
	}
}

// TestTwoUnnamedType3FontsCountAsAChange pins the second: where neither font
// has a name, Java falls back to comparing the fonts themselves. Comparing the
// two empty names would say the font had not changed.
func TestTwoUnnamedType3FontsCountAsAChange(t *testing.T) {
	// two Type 3 fonts, neither carrying a /Name
	first := type3FontWithoutName(t)
	second := type3FontWithoutName(t)
	if first.Name() != "" || second.Name() != "" {
		t.Fatal("the fonts have names, so the test proves nothing")
	}

	matrix := util.NewMatrixOf(12, 0, 0, 12, 10, 300)
	// the two positions sit far enough apart that only the font decides
	// whether the average character width is reset
	one := text.NewTextPosition(0, 300, 400, matrix, 16, 300, 8, 6, 4,
		"a", []int{97}, first, 12, 12)
	two := text.NewTextPosition(0, 300, 400, matrix, 16, 300, 8, 6, 4,
		"b", []int{98}, second, 12, 12)

	if !text.HasFontOrSizeChangedForTest(one, two) {
		t.Error("two different unnamed fonts are reported as the same font")
	}
	if text.HasFontOrSizeChangedForTest(one, one) {
		t.Error("one font is reported as having changed from itself")
	}
}

// type3FontWithoutName returns a Type 3 font whose dictionary carries no /Name.
func type3FontWithoutName(t *testing.T) font.PDFont {
	t.Helper()
	dict := cos.NewDictionary()
	dict.SetItem(cos.Type, cos.Font)
	dict.SetItem(cos.Subtype, cos.Type3)
	f, err := font.CreateFont(dict, nil)
	if err != nil {
		t.Fatalf("CreateFont: %v", err)
	}
	return f
}

// TestContainedSpaceIsRemovedFromTheArticle pins the third: Java removes the
// space through the iterator, so the list the article holds shrinks too. The
// port has to write the shortened slice back.
func TestContainedSpaceIsRemovedFromTheArticle(t *testing.T) {
	// a wide "W" at 40pt spans x = 10 to 47.8; the space is drawn at x = 20,
	// inside it. Td is relative to the line start, so 10 0 Td moves the line to
	// x = 20 rather than moving the pen on from where the W ended.
	content := "BT /F1 40 Tf 10 300 Td (W) Tj 10 0 Td ( ) Tj ET"

	stripper := text.NewPDFTextStripper()
	stripper.SetSortByPosition(true)
	var out strings.Builder
	stripper.SetOutput(&out)
	if err := stripper.ProcessPage(helveticaPage(t, content)); err != nil {
		t.Fatalf("ProcessPage: %v", err)
	}

	for _, article := range stripper.CharactersByArticle() {
		for _, position := range article {
			if position.Unicode() == " " {
				t.Fatal("the contained space is still in the article the stripper holds")
			}
		}
	}
}

// TestMultiplyFloatRoundsInFloat32 pins the fourth: Java multiplies in float
// and rounds that, so widening to float64 first can round the other way.
//
// 2.5 * 4.9 is 12.25 exactly in real arithmetic; in float32 it is 12.249999...,
// and 12.25 * 1000 lands either side of the half depending on the width the
// multiplication was done in.
func TestMultiplyFloatRoundsInFloat32(t *testing.T) {
	// 0.005 * 0.5 * 1000 is 2.5 exactly in float32 and 2.4999999...  in float64,
	// so the two widths round either side of the half. 0.012 * 42.375 goes the
	// other way. Both are the shapes the paragraph detection multiplies: a
	// threshold against a width or a height.
	cases := []struct {
		a, b float32
		want float32
	}{
		{0.005, 0.5, 0.003},
		{0.012, 42.375, 0.508},
		{2.0, 3.5, 7.0},
		{0.25, 6.0, 1.5},
	}
	for _, c := range cases {
		if got := text.MultiplyFloatForTest(c.a, c.b); got != c.want {
			t.Errorf("multiplyFloat(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
