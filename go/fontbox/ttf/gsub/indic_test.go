package gsub_test

import (
	"slices"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/gsub"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.fontbox.ttf.gsub.GsubWorkerForBengaliTest.

const lohitBengaliTTF = ttfFixture + "ttf/Lohit-Bengali.ttf"

// bengaliWorker is the @BeforeEach of the Java class.
func bengaliWorker(t *testing.T) (ttf.CmapLookup, gsub.GsubWorker) {
	t.Helper()
	source, err := pdfio.OpenBufferedFile(lohitBengaliTTF)
	if err != nil {
		t.Fatalf("opening the font: %v", err)
	}
	font, err := ttf.NewParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing the font: %v", err)
	}
	defer font.Close()
	return workerFor(t, font)
}

func TestBengaliApplyTransforms(t *testing.T) {
	cmapLookup, gsubWorkerForBengali := bengaliWorker(t)
	cases := []struct {
		name  string
		word  string
		after []int
	}{
		{"simple_hosshoi_kar", "আমি", []int{56, 102, 91}},
		{"ja_phala", "ব্যাস", []int{89, 156, 101, 97}},
		{"e_kar", "বেলা", []int{438, 89, 94, 101}},
		{"o_kar", "বোস", []int{108, 89, 101, 97}},
		{"ou_kar", "মৌল", []int{108, 91, 114, 94}},
		{"oi_kar", "বৈর", []int{439, 89, 93}},
		{"kha_e_murddhana_swa_e_khiwa", "ক্ষীরের", []int{167, 103, 438, 93, 93}},
		{"ra_phala", "দ্রুত", []int{274, 82}},
		{"ref", "ধুর্ত", []int{85, 104, 440, 82}},
		{"ra_e_hosshu", "রুপো", []int{352, 108, 87, 101}},
		{"la_e_la_e", "কল্লোল", []int{67, 108, 369, 101, 94}},
		{"khanda_ta", "হঠাৎ", []int{98, 78, 101, 113}},
		// Java marks testApplyTransforms_o_kar_repeated_1_not_working_yet and
		// _2 @Disabled; they are left out here for the same reason.
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := gsubWorkerForBengali.ApplyTransforms(glyphIDs(t, c.word, cmapLookup))
			if !slices.Equal(result, c.after) {
				t.Errorf("ApplyTransforms(%q) = %v, want %v", c.word, result, c.after)
			}
		})
	}
}

// Port of org.apache.fontbox.ttf.gsub.GsubWorkerForDfltTest.
//
// JosefinSans-Italic.ttf (SIL Open Font License) uses the DFLT script and has
// standard ligatures.

const josefinSansTTF = ttfFixture + "ttf/JosefinSans-Italic.ttf"

func dfltWorker(t *testing.T) (ttf.CmapLookup, gsub.GsubWorker) {
	t.Helper()
	source, err := pdfio.OpenBufferedFile(josefinSansTTF)
	if err != nil {
		t.Fatalf("opening the font: %v", err)
	}
	font, err := ttf.NewParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing the font: %v", err)
	}
	defer font.Close()
	return workerFor(t, font)
}

func TestCorrectWorkerType(t *testing.T) {
	_, gsubWorkerForDflt := dfltWorker(t)
	if _, ok := gsubWorkerForDflt.(*gsub.GsubWorkerForDflt); !ok {
		t.Errorf("the worker is %T, want a GsubWorkerForDflt", gsubWorkerForDflt)
	}
}

func TestDfltApplyTransforms(t *testing.T) {
	cmapLookup, gsubWorkerForDflt := dfltWorker(t)
	cases := []struct {
		input       string
		expected    []int
		description string
	}{
		// No ligature - text passes through unchanged
		{"code", []int{229, 293, 235, 237}, "no ligature sequences"},
		// Simple ligature
		{"fi", []int{407}, "fi -> ligature"},
		// Ligature within word
		{"office", []int{293, 257, 407, 229, 237}, "ffi -> f + fi-ligature"},
		// Multi-f sequence
		{"ffl", []int{257, 408}, "ffl -> f + fl-ligature"},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			result := gsubWorkerForDflt.ApplyTransforms(glyphIDs(t, c.input, cmapLookup))
			if !slices.Equal(result, c.expected) {
				t.Errorf("%s: ApplyTransforms(%q) = %v, want %v", c.description, c.input,
					result, c.expected)
			}
		})
	}
}

// TestDfltApplyTransformsImmutableResult checks that the result does not share
// its array with the argument, which is what Java's read-only wrapper comes to
// here.
func TestDfltApplyTransformsImmutableResult(t *testing.T) {
	cmapLookup, gsubWorkerForDflt := dfltWorker(t)
	original := glyphIDs(t, "abc", cmapLookup)
	result := gsubWorkerForDflt.ApplyTransforms(original)
	if len(result) == 0 {
		t.Fatal("ApplyTransforms returned nothing")
	}
	before := original[0]
	result[0] = 999
	if original[0] != before {
		t.Error("the result shares its array with the argument")
	}
}
