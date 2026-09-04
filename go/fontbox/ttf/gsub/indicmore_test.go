package gsub_test

import (
	"os"
	"slices"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/gsub"
	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/model"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Ports of the four remaining GsubWorker integration tests:
// GsubWorkerForDevanagariTest, GsubWorkerForGujaratiTest, GsubWorkerForTamilTest
// and GsubWorkerForSmcpTest, plus GsubWorkerForAaltTest.
//
// The @Disabled cases of the first two are left out, the way the Bengali port
// leaves its two out: Devanagari drops rkrf, cjct, abvs and psts, and Gujarati
// drops psts. Java's comment points at PDFBOX-5729 for why.

// workerForFile parses a font and returns its cmap lookup and the worker the
// factory picks for it, which is the @BeforeEach of each Java class.
func workerForFile(t *testing.T, path string) (ttf.CmapLookup, gsub.GsubWorker) {
	t.Helper()
	source, err := pdfio.OpenBufferedFile(path)
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

// TestDevanagariApplyTransforms is GsubWorkerForDevanagariTest.
func TestDevanagariApplyTransforms(t *testing.T) {
	cmapLookup, gsubWorkerForDevanagari := workerForFile(t,
		ttfFixture+"ttf/Lohit-Devanagari.ttf")
	cases := []struct {
		name  string
		word  string
		after []int
	}{
		{"locl", "प्त", []int{642}},
		{"nukt", "य़ज़क़", []int{400, 396, 393}},
		{"akhn", "क्षज्ञ", []int{520, 521}},
		{"rphf", "र्", []int{513}},
		{"blwf", "ह्रट्र", []int{602, 336, 516}},
		{"half", "ह्स्भ्त्", []int{558, 557, 546, 537}},
		{"vatu", "श्रत्रस्रघ्र", []int{517, 593, 601, 665}},
		{"pres", "शृक्तज्जह्ण", []int{603, 605, 617, 652}},
		{"blws", "दृहृट्रूट्रु", []int{660, 663, 336, 584, 336, 583}},
		{"haln", "द्", []int{539}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := gsubWorkerForDevanagari.ApplyTransforms(glyphIDs(t, c.word, cmapLookup))
			if !slices.Equal(result, c.after) {
				t.Errorf("ApplyTransforms(%q) = %v, want %v", c.word, result, c.after)
			}
		})
	}
}

// TestGujaratiApplyTransforms is GsubWorkerForGujaratiTest.
func TestGujaratiApplyTransforms(t *testing.T) {
	cmapLookup, gsubWorkerForGujarati := workerForFile(t, ttfFixture+"ttf/Lohit-Gujarati.ttf")
	cases := []struct {
		name  string
		word  string
		after []int
	}{
		{"akhn", "ક્ષજ્ઞત્તશ્ર", []int{330, 331, 304, 251}},
		{"rphf", "ર્સ", []int{98, 335}},
		{"rkrf", "પ્રક્રવ્ર", []int{242, 228, 250}},
		{"blwf", "ટ્ર", []int{76, 332}},
		{"half", "ત્ચ્થ્", []int{205, 195, 206}},
		{"vatu", "ત્રભ્રજ્ર", []int{237, 245, 233}},
		{"cjct", "દ્ધદ્નદ્ય", []int{309, 312, 305}},
		{"pres", "ગ્નટ્ટપ્તલ્લ", []int{284, 294, 314, 315}},
		{"abvs", "રેંરૈંર્યાં", []int{92, 255, 92, 258, 91, 102, 336}},
		{"blws", "હૃટ્રુણુરુ", []int{278, 76, 333, 337, 276}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			result := gsubWorkerForGujarati.ApplyTransforms(glyphIDs(t, c.word, cmapLookup))
			if !slices.Equal(result, c.after) {
				t.Errorf("ApplyTransforms(%q) = %v, want %v", c.word, result, c.after)
			}
		})
	}
}

// TestTamilDummy is GsubWorkerForTamilTest.testDummy, whose comment says to
// change the assertion to GsubWorkerForTamil once that worker is implemented.
func TestTamilDummy(t *testing.T) {
	_, gsubWorkerForTamil := workerForFile(t, ttfFixture+"ttf/Lohit-Tamil.ttf")
	if _, ok := gsubWorkerForTamil.(*gsub.DefaultGsubWorker); !ok {
		t.Errorf("the worker is %T, want a DefaultGsubWorker", gsubWorkerForTamil)
	}
}

// featureWorker is the GSUB worker both GsubWorkerForAalt and GsubWorkerForSmcp
// are: the Latin worker with one feature of its own. Both live in the Java test
// tree rather than the library, and both copy applyGsubFeature out of the Latin
// worker rather than sharing it, so the port copies it too.
type featureWorker struct {
	featuresInOrder []string
	gsubData        model.GsubData
}

func (w *featureWorker) ApplyTransforms(originalGlyphIDs []int) []int {
	intermediateGlyphsFromGsub := originalGlyphIDs

	for _, feature := range w.featuresInOrder {
		if !w.gsubData.IsFeatureSupported(feature) {
			continue
		}
		scriptFeature := w.gsubData.Feature(feature)
		intermediateGlyphsFromGsub = applyGsubFeature(scriptFeature, intermediateGlyphsFromGsub)
	}

	return intermediateGlyphsFromGsub
}

func applyGsubFeature(scriptFeature model.ScriptFeature, originalGlyphs []int) []int {
	allGlyphIDsForSubstitution := scriptFeature.AllGlyphIDsForSubstitution()
	if len(allGlyphIDsForSubstitution) == 0 {
		return originalGlyphs
	}

	glyphArraySplitter := gsub.NewGlyphArraySplitterRegexImpl(allGlyphIDsForSubstitution)

	tokens := glyphArraySplitter.Split(originalGlyphs)
	var gsubProcessedGlyphs []int

	for _, chunk := range tokens {
		if scriptFeature.CanReplaceGlyphs(chunk) {
			// gsub system kicks in, you get the glyphId directly
			gsubProcessedGlyphs = append(gsubProcessedGlyphs,
				scriptFeature.ReplacementForGlyphs(chunk)...)
		} else {
			gsubProcessedGlyphs = append(gsubProcessedGlyphs, chunk...)
		}
	}

	return gsubProcessedGlyphs
}

// TestFoglihtenNo07 is GsubWorkerForAaltTest: the "aalt" type 3 tables of a
// font.
func TestFoglihtenNo07(t *testing.T) {
	source, err := pdfio.OpenBufferedFile(ttfFixture + "otf/FoglihtenNo07.otf")
	if err != nil {
		t.Fatalf("opening the font: %v", err)
	}
	font, err := ttf.NewOTFParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing the font: %v", err)
	}
	cmapLookup, err := font.UnicodeCmapLookupStrict()
	if err != nil {
		t.Fatalf("UnicodeCmapLookupStrict: %v", err)
	}
	gsubData, err := font.GsubData()
	if err != nil {
		t.Fatalf("GsubData: %v", err)
	}
	if err := font.Close(); err != nil {
		t.Fatalf("closing the font: %v", err)
	}
	gsubWorkerForAlt := &featureWorker{featuresInOrder: []string{"aalt"}, gsubData: gsubData}

	// Values should be the same you get by looking at the GSUB lookup lists 12
	// or 13 with a font tool
	want := []int{1139, 1562, 1477}
	got := gsubWorkerForAlt.ApplyTransforms(glyphIDs(t, "Abc", cmapLookup))
	if !slices.Equal(got, want) {
		t.Errorf("ApplyTransforms(\"Abc\") = %v, want %v", got, want)
	}
}

// TestCalibriSmcp is GsubWorkerForSmcpTest: the "smcp" type 2 tables of a font.
func TestCalibriSmcp(t *testing.T) {
	const file = `c:/windows/fonts/calibri.ttf`
	if _, err := os.Stat(file); err != nil {
		t.Skip("calibri smcp test skipped")
	}

	source, err := pdfio.OpenBufferedFile(file)
	if err != nil {
		t.Fatalf("opening the font: %v", err)
	}
	font, err := ttf.NewParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing the font: %v", err)
	}
	cmapLookup, err := font.UnicodeCmapLookupStrict()
	if err != nil {
		t.Fatalf("UnicodeCmapLookupStrict: %v", err)
	}
	gsubData, err := font.GsubData()
	if err != nil {
		t.Fatalf("GsubData: %v", err)
	}
	if err := font.Close(); err != nil {
		t.Fatalf("closing the font: %v", err)
	}
	gsubWorkerForSmcp := &featureWorker{featuresInOrder: []string{"smcp"}, gsubData: gsubData}

	// Values should be the same you get by looking at the GSUB lookup list 24
	// with a font tool. This one converts U+FB00, the single-ff-ligature glyph,
	// into "FF" small capitals.
	want := []int{165, 165}
	got := gsubWorkerForSmcp.ApplyTransforms(glyphIDs(t, "\ufb00", cmapLookup))
	if !slices.Equal(got, want) {
		t.Errorf("ApplyTransforms(U+FB00) = %v, want %v", got, want)
	}
}
