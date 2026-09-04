package afm

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/util"
)

// Ported from fontbox/src/test/java/org/apache/fontbox/afm/, one Go test per
// Java test file: LigatureTest, KernPairTest, TrackKernTest, CompositePartTest,
// CompositeTest, CharMetricTest and FontMetricsTest.
//
// Java returns Collections.unmodifiableList from every list accessor, and each
// Java test asserts that adding to it throws UnsupportedOperationException. Go
// has no unmodifiable slice, so the port returns a copy and the tests assert
// what that buys instead: appending to what came out does not reach the object.

func TestLigature(t *testing.T) {
	ligature := NewLigature("successor", "ligature")
	if got := ligature.Successor(); got != "successor" {
		t.Errorf("Successor() = %q, want %q", got, "successor")
	}
	if got := ligature.Ligature(); got != "ligature" {
		t.Errorf("Ligature() = %q, want %q", got, "ligature")
	}
}

func TestKernPair(t *testing.T) {
	kernPair := NewKernPair("firstKernCharacter", "secondKernCharacter", 10, 20)
	if got := kernPair.FirstKernCharacter(); got != "firstKernCharacter" {
		t.Errorf("FirstKernCharacter() = %q, want %q", got, "firstKernCharacter")
	}
	if got := kernPair.SecondKernCharacter(); got != "secondKernCharacter" {
		t.Errorf("SecondKernCharacter() = %q, want %q", got, "secondKernCharacter")
	}
	if got := kernPair.X(); got != 10 {
		t.Errorf("X() = %v, want 10", got)
	}
	if got := kernPair.Y(); got != 20 {
		t.Errorf("Y() = %v, want 20", got)
	}
}

func TestTrackKern(t *testing.T) {
	trackKern := NewTrackKern(0, 1.0, 1.0, 10.0, 10.0)
	if got := trackKern.Degree(); got != 0 {
		t.Errorf("Degree() = %v, want 0", got)
	}
	if got := trackKern.MinPointSize(); got != 1.0 {
		t.Errorf("MinPointSize() = %v, want 1.0", got)
	}
	if got := trackKern.MinKern(); got != 1.0 {
		t.Errorf("MinKern() = %v, want 1.0", got)
	}
	if got := trackKern.MaxPointSize(); got != 10.0 {
		t.Errorf("MaxPointSize() = %v, want 10.0", got)
	}
	if got := trackKern.MaxKern(); got != 10.0 {
		t.Errorf("MaxKern() = %v, want 10.0", got)
	}
}

func TestCompositePart(t *testing.T) {
	compositePart := NewCompositePart("name", 10, 20)
	if got := compositePart.Name(); got != "name" {
		t.Errorf("Name() = %q, want %q", got, "name")
	}
	if got := compositePart.XDisplacement(); got != 10 {
		t.Errorf("XDisplacement() = %v, want 10", got)
	}
	if got := compositePart.YDisplacement(); got != 20 {
		t.Errorf("YDisplacement() = %v, want 20", got)
	}
}

func TestComposite(t *testing.T) {
	composite := NewComposite("name")
	if got := composite.Name(); got != "name" {
		t.Errorf("Name() = %q, want %q", got, "name")
	}
	if got := composite.Parts(); len(got) != 0 {
		t.Errorf("Parts() = %v, want none", got)
	}

	compositePart := NewCompositePart("name", 10, 20)
	composite.AddPart(compositePart)
	parts := composite.Parts()
	if len(parts) != 1 {
		t.Fatalf("Parts() has %d entries, want 1", len(parts))
	}
	if got := parts[0].Name(); got != "name" {
		t.Errorf("Parts()[0].Name() = %q, want %q", got, "name")
	}

	// Java hands back an unmodifiable list; the port hands back a copy.
	parts = append(parts, compositePart)
	if got := len(composite.Parts()); got != 1 {
		t.Errorf("appending to the returned slice changed the composite to %d parts", got)
	}
}

func TestCharMetricSimpleValues(t *testing.T) {
	charMetric := NewCharMetric()
	charMetric.SetCharacterCode(0)
	charMetric.SetName("name")
	charMetric.SetWx(10)
	charMetric.SetW0x(20)
	charMetric.SetW1x(30)
	charMetric.SetWy(40)
	charMetric.SetW0y(50)
	charMetric.SetW1y(60)

	if got := charMetric.CharacterCode(); got != 0 {
		t.Errorf("CharacterCode() = %v, want 0", got)
	}
	if got := charMetric.Name(); got != "name" {
		t.Errorf("Name() = %q, want %q", got, "name")
	}
	for _, c := range []struct {
		name string
		got  float32
		want float32
	}{
		{"Wx", charMetric.Wx(), 10},
		{"W0x", charMetric.W0x(), 20},
		{"W1x", charMetric.W1x(), 30},
		{"Wy", charMetric.Wy(), 40},
		{"W0y", charMetric.W0y(), 50},
		{"W1y", charMetric.W1y(), 60},
	} {
		if c.got != c.want {
			t.Errorf("%s() = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestCharMetricArrayValues(t *testing.T) {
	charMetric := NewCharMetric()
	charMetric.SetW([]float32{10, 20})
	charMetric.SetW0([]float32{30, 40})
	charMetric.SetW1([]float32{50, 60})
	charMetric.SetVv([]float32{70, 80})

	for _, c := range []struct {
		name string
		got  []float32
		want [2]float32
	}{
		{"W", charMetric.W(), [2]float32{10, 20}},
		{"W0", charMetric.W0(), [2]float32{30, 40}},
		{"W1", charMetric.W1(), [2]float32{50, 60}},
		{"Vv", charMetric.Vv(), [2]float32{70, 80}},
	} {
		if len(c.got) != 2 || c.got[0] != c.want[0] || c.got[1] != c.want[1] {
			t.Errorf("%s() = %v, want %v", c.name, c.got, c.want)
		}
	}
}

func TestCharMetricComplexValues(t *testing.T) {
	charMetric := NewCharMetric()
	charMetric.SetBoundingBox(util.NewBoundingBoxOf(10, 20, 30, 40))

	box := charMetric.BoundingBox()
	if box.LowerLeftX() != 10 || box.LowerLeftY() != 20 ||
		box.UpperRightX() != 30 || box.UpperRightY() != 40 {
		t.Errorf("BoundingBox() = %v, want [10.0,20.0,30.0,40.0]", box)
	}

	if got := charMetric.Ligatures(); len(got) != 0 {
		t.Errorf("Ligatures() = %v, want none", got)
	}
	ligature := NewLigature("successor", "ligature")
	charMetric.AddLigature(ligature)
	ligatures := charMetric.Ligatures()
	if len(ligatures) != 1 {
		t.Fatalf("Ligatures() has %d entries, want 1", len(ligatures))
	}
	if got := ligatures[0].Successor(); got != "successor" {
		t.Errorf("Ligatures()[0].Successor() = %q, want %q", got, "successor")
	}

	// Java hands back an unmodifiable list; the port hands back a copy.
	ligatures = append(ligatures, ligature)
	if got := len(charMetric.Ligatures()); got != 1 {
		t.Errorf("appending to the returned slice changed the metric to %d ligatures", got)
	}
}

func TestFontMetricsNames(t *testing.T) {
	fontMetrics := NewFontMetrics()
	fontMetrics.SetFontName("fontName")
	fontMetrics.SetFamilyName("familyName")
	fontMetrics.SetFullName("fullName")
	fontMetrics.SetFontVersion("fontVersion")
	fontMetrics.SetNotice("notice")

	for _, c := range []struct {
		name string
		got  string
		want string
	}{
		{"FontName", fontMetrics.FontName(), "fontName"},
		{"FamilyName", fontMetrics.FamilyName(), "familyName"},
		{"FullName", fontMetrics.FullName(), "fullName"},
		{"FontVersion", fontMetrics.FontVersion(), "fontVersion"},
		{"Notice", fontMetrics.Notice(), "notice"},
	} {
		if c.got != c.want {
			t.Errorf("%s() = %q, want %q", c.name, c.got, c.want)
		}
	}

	if got := fontMetrics.Comments(); len(got) != 0 {
		t.Errorf("Comments() = %v, want none", got)
	}
	fontMetrics.AddComment("comment")
	comments := fontMetrics.Comments()
	if len(comments) != 1 {
		t.Fatalf("Comments() has %d entries, want 1", len(comments))
	}

	comments = append(comments, "comment")
	if got := len(fontMetrics.Comments()); got != 1 {
		t.Errorf("appending to the returned slice changed the metrics to %d comments", got)
	}
}

func TestFontMetricsSimpleValues(t *testing.T) {
	fontMetrics := NewFontMetrics()
	fontMetrics.SetAFMVersion(4.3)
	fontMetrics.SetWeight("weight")
	fontMetrics.SetEncodingScheme("encodingScheme")
	fontMetrics.SetMappingScheme(0)
	fontMetrics.SetEscChar(0)
	fontMetrics.SetCharacterSet("characterSet")
	fontMetrics.SetCharacters(10)
	fontMetrics.SetIsBaseFont(true)
	fontMetrics.SetIsFixedV(true)
	fontMetrics.SetCapHeight(10)
	fontMetrics.SetXHeight(20)
	fontMetrics.SetAscender(30)
	fontMetrics.SetDescender(40)
	fontMetrics.SetStandardHorizontalWidth(50)
	fontMetrics.SetStandardVerticalWidth(60)
	fontMetrics.SetUnderlinePosition(70)
	fontMetrics.SetUnderlineThickness(80)
	fontMetrics.SetItalicAngle(90)
	fontMetrics.SetFixedPitch(true)

	if got := fontMetrics.AFMVersion(); got != 4.3 {
		t.Errorf("AFMVersion() = %v, want 4.3", got)
	}
	if got := fontMetrics.Weight(); got != "weight" {
		t.Errorf("Weight() = %q, want %q", got, "weight")
	}
	if got := fontMetrics.EncodingScheme(); got != "encodingScheme" {
		t.Errorf("EncodingScheme() = %q, want %q", got, "encodingScheme")
	}
	if got := fontMetrics.MappingScheme(); got != 0 {
		t.Errorf("MappingScheme() = %v, want 0", got)
	}
	if got := fontMetrics.EscChar(); got != 0 {
		t.Errorf("EscChar() = %v, want 0", got)
	}
	if got := fontMetrics.CharacterSet(); got != "characterSet" {
		t.Errorf("CharacterSet() = %q, want %q", got, "characterSet")
	}
	if got := fontMetrics.Characters(); got != 10 {
		t.Errorf("Characters() = %v, want 10", got)
	}
	if !fontMetrics.IsBaseFont() {
		t.Error("IsBaseFont() = false, want true")
	}
	if !fontMetrics.IsFixedV() {
		t.Error("IsFixedV() = false, want true")
	}
	for _, c := range []struct {
		name string
		got  float32
		want float32
	}{
		{"CapHeight", fontMetrics.CapHeight(), 10},
		{"XHeight", fontMetrics.XHeight(), 20},
		{"Ascender", fontMetrics.Ascender(), 30},
		{"Descender", fontMetrics.Descender(), 40},
		{"StandardHorizontalWidth", fontMetrics.StandardHorizontalWidth(), 50},
		{"StandardVerticalWidth", fontMetrics.StandardVerticalWidth(), 60},
		{"UnderlinePosition", fontMetrics.UnderlinePosition(), 70},
		{"UnderlineThickness", fontMetrics.UnderlineThickness(), 80},
		{"ItalicAngle", fontMetrics.ItalicAngle(), 90},
	} {
		if c.got != c.want {
			t.Errorf("%s() = %v, want %v", c.name, c.got, c.want)
		}
	}
	if !fontMetrics.IsFixedPitch() {
		t.Error("IsFixedPitch() = false, want true")
	}
}

func TestFontMetricsComplexValues(t *testing.T) {
	fontMetrics := NewFontMetrics()
	fontMetrics.SetFontBBox(util.NewBoundingBoxOf(10, 20, 30, 40))
	fontMetrics.SetVVector([]float32{10, 20})
	fontMetrics.SetCharWidth([]float32{30, 40})

	box := fontMetrics.FontBBox()
	if box.LowerLeftX() != 10 || box.LowerLeftY() != 20 ||
		box.UpperRightX() != 30 || box.UpperRightY() != 40 {
		t.Errorf("FontBBox() = %v, want [10.0,20.0,30.0,40.0]", box)
	}
	if got := fontMetrics.VVector(); len(got) != 2 || got[0] != 10 || got[1] != 20 {
		t.Errorf("VVector() = %v, want [10 20]", got)
	}
	if got := fontMetrics.CharWidth(); len(got) != 2 || got[0] != 30 || got[1] != 40 {
		t.Errorf("CharWidth() = %v, want [30 40]", got)
	}
}

// TestMetricSets pins that the metric sets value is checked. Java throws
// IllegalArgumentException outside 0..2; the port panics, since it is a
// caller's mistake and nothing in PDFBox catches it.
func TestMetricSets(t *testing.T) {
	fontMetrics := NewFontMetrics()
	fontMetrics.SetMetricSets(1)
	if got := fontMetrics.MetricSets(); got != 1 {
		t.Errorf("MetricSets() = %v, want 1", got)
	}

	for _, value := range []int{-1, 3} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("SetMetricSets(%d) did not panic", value)
				}
			}()
			fontMetrics.SetMetricSets(value)
		}()
	}
}

func TestFontMetricsCharMetrics(t *testing.T) {
	fontMetrics := NewFontMetrics()
	if got := fontMetrics.CharMetrics(); len(got) != 0 {
		t.Errorf("CharMetrics() = %v, want none", got)
	}

	charMetric := NewCharMetric()
	fontMetrics.AddCharMetric(charMetric)
	charMetrics := fontMetrics.CharMetrics()
	if len(charMetrics) != 1 {
		t.Fatalf("CharMetrics() has %d entries, want 1", len(charMetrics))
	}

	charMetrics = append(charMetrics, charMetric)
	if got := len(fontMetrics.CharMetrics()); got != 1 {
		t.Errorf("appending to the returned slice changed the metrics to %d entries", got)
	}
}

func TestFontMetricsComposites(t *testing.T) {
	fontMetrics := NewFontMetrics()
	if got := fontMetrics.Composites(); len(got) != 0 {
		t.Errorf("Composites() = %v, want none", got)
	}

	composite := NewComposite("name")
	fontMetrics.AddComposite(composite)
	composites := fontMetrics.Composites()
	if len(composites) != 1 {
		t.Fatalf("Composites() has %d entries, want 1", len(composites))
	}

	composites = append(composites, composite)
	if got := len(fontMetrics.Composites()); got != 1 {
		t.Errorf("appending to the returned slice changed the metrics to %d entries", got)
	}
}

func TestFontMetricsKernData(t *testing.T) {
	fontMetrics := NewFontMetrics()
	kernPair := NewKernPair("first", "second", 10, 20)

	cases := []struct {
		name string
		add  func()
		get  func() []*KernPair
	}{
		{"KernPairs", func() { fontMetrics.AddKernPair(kernPair) }, fontMetrics.KernPairs},
		{"KernPairs0", func() { fontMetrics.AddKernPair0(kernPair) }, fontMetrics.KernPairs0},
		{"KernPairs1", func() { fontMetrics.AddKernPair1(kernPair) }, fontMetrics.KernPairs1},
	}
	for _, c := range cases {
		if got := c.get(); len(got) != 0 {
			t.Errorf("%s() = %v, want none", c.name, got)
		}
		c.add()
		pairs := c.get()
		if len(pairs) != 1 {
			t.Fatalf("%s() has %d entries, want 1", c.name, len(pairs))
		}
		pairs = append(pairs, kernPair)
		if got := len(c.get()); got != 1 {
			t.Errorf("appending to %s() changed the metrics to %d entries", c.name, got)
		}
	}

	if got := fontMetrics.TrackKern(); len(got) != 0 {
		t.Errorf("TrackKern() = %v, want none", got)
	}
	trackKern := NewTrackKern(0, 1, 1, 10, 10)
	fontMetrics.AddTrackKern(trackKern)
	trackKerns := fontMetrics.TrackKern()
	if len(trackKerns) != 1 {
		t.Fatalf("TrackKern() has %d entries, want 1", len(trackKerns))
	}
	trackKerns = append(trackKerns, trackKern)
	if got := len(fontMetrics.TrackKern()); got != 1 {
		t.Errorf("appending to TrackKern() changed the metrics to %d entries", got)
	}
}

func TestCharMetricDimensions(t *testing.T) {
	fontMetrics := NewFontMetrics()
	if got := fontMetrics.AverageCharacterWidth(); got != 0 {
		t.Errorf("AverageCharacterWidth() of empty metrics = %v, want 0", got)
	}

	for _, c := range []struct {
		name   string
		wx, wy float32
	}{
		{"ten", 10, 20},
		{"twenty", 20, 40},
		{"thirty", 30, 60},
		{"forty", 40, 80},
	} {
		charMetric := NewCharMetric()
		charMetric.SetName(c.name)
		charMetric.SetWx(c.wx)
		charMetric.SetWy(c.wy)
		fontMetrics.AddCharMetric(charMetric)
	}

	if got := fontMetrics.CharacterWidth("ten"); got != 10 {
		t.Errorf("CharacterWidth(ten) = %v, want 10", got)
	}
	if got := fontMetrics.CharacterWidth("thirty"); got != 30 {
		t.Errorf("CharacterWidth(thirty) = %v, want 30", got)
	}
	if got := fontMetrics.CharacterWidth("unknown"); got != 0 {
		t.Errorf("CharacterWidth(unknown) = %v, want 0", got)
	}

	if got := fontMetrics.CharacterHeight("twenty"); got != 40 {
		t.Errorf("CharacterHeight(twenty) = %v, want 40", got)
	}
	if got := fontMetrics.CharacterHeight("forty"); got != 80 {
		t.Errorf("CharacterHeight(forty) = %v, want 80", got)
	}
	if got := fontMetrics.CharacterHeight("unknown"); got != 0 {
		t.Errorf("CharacterHeight(unknown) = %v, want 0", got)
	}

	if got := fontMetrics.AverageCharacterWidth(); got != 25 {
		t.Errorf("AverageCharacterWidth() = %v, want 25", got)
	}
}
