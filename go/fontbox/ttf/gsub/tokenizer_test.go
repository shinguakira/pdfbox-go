package gsub

import (
	"slices"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/model"
)

// Port of org.apache.fontbox.ttf.gsub.CompoundCharacterTokenizerTest.
//
// Java builds the tokenizer from a HashSet, whose iteration order it does not
// define; the port takes the words in the order the test lists them, and no
// case here depends on that order.

func TestTokenize(t *testing.T) {
	cases := []struct {
		name          string
		compoundWords []string
		text          string
		want          []string
	}{
		{
			"happyPath_2",
			[]string{"_84_93_", "_104_82_", "_104_87_"},
			"_84_112_93_104_82_61_96_102_93_104_87_110_",
			[]string{"_84_112_93", "_104_82_", "_61_96_102_93", "_104_87_", "_110_"},
		},
		{
			"happyPath_3",
			[]string{"_67_112_96_", "_74_112_76_"},
			"_67_112_96_103_93_108_93_",
			[]string{"_67_112_96_", "_103_93_108_93_"},
		},
		{
			"happyPath_4",
			[]string{"_67_112_96_", "_74_112_76_"},
			"_94_67_112_96_112_91_103_",
			[]string{"_94", "_67_112_96_", "_112_91_103_"},
		},
		{
			"happyPath_5",
			[]string{"_67_112_", "_76_112_"},
			"_94_167_112_91_103_",
			[]string{"_94_167_112_91_103_"},
		},
		{
			"happyPath_6",
			[]string{"_100_", "_101_", "_102_", "_103_", "_104_"},
			"_100_101_102_103_104_",
			[]string{"_100_", "_101_", "_102_", "_103_", "_104_"},
		},
		{
			"happyPath_7",
			[]string{"_100_101_", "_102_", "_103_104_"},
			"_100_101_102_103_104_",
			[]string{"_100_101_", "_102_", "_103_104_"},
		},
		{
			"happyPath_8",
			[]string{"_100_101_102_", "_101_102_", "_103_104_"},
			"_100_101_102_103_104_",
			[]string{"_100_101_102_", "_103_104_"},
		},
		{
			"happyPath_9",
			[]string{"_101_102_", "_101_102_"},
			"_100_101_102_103_104_",
			[]string{"_100", "_101_102_", "_103_104_"},
		},
		{
			"happyPath_10",
			[]string{"_201_", "_202_"},
			"_100_101_102_103_104_",
			[]string{"_100_101_102_103_104_"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tokenizer := NewCompoundCharacterTokenizer(c.compoundWords)
			got := tokenizer.Tokenize(c.text)
			if !slices.Equal(got, c.want) {
				t.Errorf("Tokenize(%q) = %q, want %q", c.text, got, c.want)
			}
		})
	}
}

// Port of org.apache.fontbox.ttf.gsub.GlyphArraySplitterRegexImplTest.

func TestSplit(t *testing.T) {
	cases := []struct {
		name     string
		matchers [][]int
		glyphIDs []int
		want     [][]int
	}{
		{
			"split_1",
			[][]int{{84, 93}, {102, 82}, {104, 87}},
			[]int{84, 112, 93, 104, 82, 61, 96, 102, 93, 104, 87, 110},
			[][]int{{84, 112, 93, 104, 82, 61, 96, 102, 93}, {104, 87}, {110}},
		},
		{
			"split_2",
			[][]int{{67, 112, 96}, {74, 112, 76}},
			[]int{67, 112, 96, 103, 93, 108, 93},
			[][]int{{67, 112, 96}, {103, 93, 108, 93}},
		},
		{
			"split_3",
			[][]int{{67, 112, 96}, {74, 112, 76}},
			[]int{94, 67, 112, 96, 112, 91, 103},
			[][]int{{94}, {67, 112, 96}, {112, 91, 103}},
		},
		{
			"split_4",
			[][]int{{67, 112}, {76, 112}},
			[]int{94, 167, 112, 91, 103},
			[][]int{{94, 167, 112, 91, 103}},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			matchers := make([]model.GlyphKey, len(c.matchers))
			for i, ids := range c.matchers {
				matchers[i] = model.NewGlyphKey(ids)
			}
			testClass := NewGlyphArraySplitterRegexImpl(matchers)
			tokens := testClass.Split(c.glyphIDs)
			if len(tokens) != len(c.want) {
				t.Fatalf("Split gave %d tokens, want %d: %v", len(tokens), len(c.want), tokens)
			}
			for i := range tokens {
				if !slices.Equal(tokens[i], c.want[i]) {
					t.Errorf("token %d = %v, want %v", i, tokens[i], c.want[i])
				}
			}
		})
	}
}

// Port of org.apache.fontbox.ttf.gsub.DefaultGsubWorkerTest.
//
// Java asserts the result is read-only by calling clear on it; a Go slice has
// no such wrapper, so what carries over is that the result is a copy, which the
// caller can change without touching the argument.
func TestDefaultGsubWorkerApplyTransforms(t *testing.T) {
	// given
	sut := &DefaultGsubWorker{}
	originalGlyphIDs := []int{1, 2, 3, 4, 5}

	// when
	pseudoTransformedIDs := sut.ApplyTransforms(originalGlyphIDs)

	// then
	if !slices.Equal(originalGlyphIDs, pseudoTransformedIDs) {
		t.Errorf("ApplyTransforms = %v, want %v", pseudoTransformedIDs, originalGlyphIDs)
	}
	pseudoTransformedIDs[0] = 999
	if originalGlyphIDs[0] != 1 {
		t.Error("the result shares its array with the argument")
	}
}
