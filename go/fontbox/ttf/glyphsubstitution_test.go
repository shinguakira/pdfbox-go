package ttf

import (
	"bufio"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf/model"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.fontbox.ttf.GlyphSubstitutionTableTest.

const dataPositionForGsubTable = 120544

var expectedFeatureNames = []string{"abvs", "akhn", "blwf", "blws", "half", "haln", "init",
	"nukt", "pres", "pstf", "rphf", "vatu"}

func TestGetGsubData(t *testing.T) {
	// given
	data, err := os.ReadFile(ttfFixture + "Lohit-Bengali.ttf")
	if err != nil {
		t.Fatalf("reading the font: %v", err)
	}
	rarbds, err := NewRandomAccessReadDataStream(pdfio.NewReadBufferBytes(data))
	if err != nil {
		t.Fatalf("opening the font: %v", err)
	}
	defer rarbds.Close()
	if err := rarbds.SeekTo(dataPositionForGsubTable); err != nil {
		t.Fatalf("seeking to the GSUB table: %v", err)
	}

	testClass := &GlyphSubstitutionTable{}

	// when
	if err := testClass.Read(nil, rarbds); err != nil {
		t.Fatalf("reading the GSUB table: %v", err)
	}

	// then
	gsubData := testClass.GsubData()
	if gsubData == nil {
		t.Fatal("GsubData is nil")
	}
	if gsubData == model.NoDataFound {
		t.Fatal("GsubData is NoDataFound")
	}
	if got := gsubData.Language(); got != model.Bengali {
		t.Errorf("Language = %v, want %v", got, model.Bengali)
	}
	if got := gsubData.ActiveScriptName(); got != "bng2" {
		t.Errorf("ActiveScriptName = %q, want %q", got, "bng2")
	}

	want := slices.Clone(expectedFeatureNames)
	sort.Strings(want)
	if got := gsubData.SupportedFeatures(); !slices.Equal(got, want) {
		t.Errorf("SupportedFeatures = %v, want %v", got, want)
	}

	for _, featureName := range expectedFeatureNames {
		expected := expectedGsubTableRawData(t,
			fmt.Sprintf("../../../fontbox/src/test/resources/gsub/lohit_bengali/bng2/%s.txt",
				featureName))
		scriptFeature := model.NewMapBackedScriptFeature(featureName, expected)
		actual, ok := gsubData.Feature(featureName).(*model.MapBackedScriptFeature)
		if !ok {
			t.Errorf("feature %s is %T, want a MapBackedScriptFeature", featureName,
				gsubData.Feature(featureName))
			continue
		}
		if !scriptFeature.Equals(actual) {
			t.Errorf("feature %s does not match the expected substitutions", featureName)
		}
	}
}

// expectedGsubTableRawData reads one of the Java test's expectation files,
// whose lines are "oldGlyphIds=newGlyphId".
func expectedGsubTableRawData(t *testing.T, pathToResource string) map[model.GlyphKey][]int {
	t.Helper()
	file, err := os.Open(pathToResource)
	if err != nil {
		t.Fatalf("opening %s: %v", pathToResource, err)
	}
	defer file.Close()

	gsubData := map[model.GlyphKey][]int{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "=")
		if len(parts) != 2 {
			t.Fatalf("invalid format in %s: %q", pathToResource, line)
		}
		var oldGlyphIDs []int
		for _, value := range strings.Split(parts[0], ",") {
			id, err := strconv.Atoi(value)
			if err != nil {
				t.Fatalf("invalid format in %s: %q", pathToResource, line)
			}
			oldGlyphIDs = append(oldGlyphIDs, id)
		}
		newGlyphID, err := strconv.Atoi(parts[1])
		if err != nil {
			t.Fatalf("invalid format in %s: %q", pathToResource, line)
		}
		gsubData[model.NewGlyphKey(oldGlyphIDs)] = []int{newGlyphID}
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("reading %s: %v", pathToResource, err)
	}
	return gsubData
}

// Port of org.apache.fontbox.ttf.GlyphSubstitutionTableLiberationFontTest.

// liberationFont opens the font the Java test reads in its @BeforeEach.
func liberationFont(t *testing.T) *OpenTypeFont {
	t.Helper()
	fontFile, err := pdfio.OpenBufferedFile(ttfFixture + "LiberationSans-Regular.ttf")
	if err != nil {
		t.Fatalf("opening the font: %v", err)
	}
	font, err := NewOTFParser().Parse(fontFile)
	if err != nil {
		t.Fatalf("parsing the font: %v", err)
	}
	t.Cleanup(func() { font.Close() })
	return font
}

// TestGetGsubDataDefault checks that GsubData with no args yields latn.
func TestGetGsubDataDefault(t *testing.T) {
	font := liberationFont(t)
	gsubData, err := font.GsubData()
	if err != nil {
		t.Fatalf("GsubData: %v", err)
	}
	if got := gsubData.ActiveScriptName(); got != "latn" {
		t.Errorf("ActiveScriptName = %q, want %q", got, "latn")
	}
}

// TestGetGsubDataForUnsupportedScriptTag checks that GsubData for an
// unsupported script yields nil.
func TestGetGsubDataForUnsupportedScriptTag(t *testing.T) {
	font := liberationFont(t)
	gsub, err := font.GSUB()
	if err != nil {
		t.Fatalf("GSUB: %v", err)
	}
	if gsubData := gsub.GsubDataForScript("<some_non_existent_script_tag>"); gsubData != nil {
		t.Errorf("GsubDataForScript = %v, want nil", gsubData)
	}
}

// TestGetGsubDataForCyrillic checks that GsubData for the 'cyrl' tag yields the
// GSUB features of the Cyrillic script.
func TestGetGsubDataForCyrillic(t *testing.T) {
	font := liberationFont(t)
	gsub, err := font.GSUB()
	if err != nil {
		t.Fatalf("GSUB: %v", err)
	}
	const cyrillicScriptTag = "cyrl"
	expectedFeatures := []string{"subs", "sups"}

	cyrillicGsubData := gsub.GsubDataForScript(cyrillicScriptTag)

	if cyrillicGsubData == nil {
		t.Fatal("GsubDataForScript(cyrl) is nil")
	}
	if got := cyrillicGsubData.ActiveScriptName(); got != cyrillicScriptTag {
		t.Errorf("ActiveScriptName = %q, want %q", got, cyrillicScriptTag)
	}
	if got := cyrillicGsubData.SupportedFeatures(); !slices.Equal(got, expectedFeatures) {
		t.Errorf("SupportedFeatures = %v, want %v", got, expectedFeatures)
	}
}

// TestGetSupportedScriptTags checks that all the script tags are loaded from
// GSUB as is.
func TestGetSupportedScriptTags(t *testing.T) {
	font := liberationFont(t)
	gsub, err := font.GSUB()
	if err != nil {
		t.Fatalf("GSUB: %v", err)
	}
	expectedSet := []string{"DFLT", "bopo", "copt", "cyrl", "grek", "hebr", "latn"}

	supportedScriptTags := gsub.SupportedScriptTags()

	got := slices.Clone(supportedScriptTags)
	sort.Strings(got)
	want := slices.Clone(expectedSet)
	sort.Strings(want)
	if !slices.Equal(got, want) {
		t.Errorf("SupportedScriptTags = %v, want %v", got, want)
	}
}

// TestGsubDataLoadingForAllSupportedScripts checks that GSUB data is loaded for
// all scripts supported by the font.
func TestGsubDataLoadingForAllSupportedScripts(t *testing.T) {
	for _, scriptTag := range []string{"DFLT", "bopo", "copt", "cyrl", "grek", "hebr", "latn"} {
		t.Run(scriptTag, func(t *testing.T) {
			font := liberationFont(t)
			gsub, err := font.GSUB()
			if err != nil {
				t.Fatalf("GSUB: %v", err)
			}

			gsubData := gsub.GsubDataForScript(scriptTag)

			if gsubData == nil {
				t.Fatal("GsubDataForScript is nil")
			}
			if gsubData == model.NoDataFound {
				t.Fatal("GsubDataForScript is NoDataFound")
			}
			if got := gsubData.Language(); got != model.Unspecified {
				t.Errorf("Language = %v, want %v", got, model.Unspecified)
			}
			if got := gsubData.ActiveScriptName(); got != scriptTag {
				t.Errorf("ActiveScriptName = %q, want %q", got, scriptTag)
			}
		})
	}
}
