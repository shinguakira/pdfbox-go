package afm

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/util"
)

// Ported from
// fontbox/src/test/java/org/apache/fontbox/afm/AFMParserTest.java.
//
// The fixtures are the Java ones, read from the Java tree where they sit. They
// are data — a real Helvetica.afm with 315 character metrics and 2705 kerning
// pairs — and there is nothing to be gained by copying them.

// afmFixture is the directory the Java test resources live in, relative to this
// package.
const afmFixture = "../../../fontbox/src/test/resources/afm/"

const helveticaAFM = afmFixture + "Helvetica.afm"

func openFixture(t *testing.T, path string) *os.File {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

func TestStartFontMetrics(t *testing.T) {
	_, err := NewAFMParser(strings.NewReader("huhu")).Parse()
	if err == nil {
		t.Fatal("parsing a file with no StartFontMetrics succeeded, want an error")
	}
	if !strings.Contains(err.Error(), StartFontMetrics) {
		t.Errorf("err = %v, want it to name %s", err, StartFontMetrics)
	}
}

func TestEndFontMetrics(t *testing.T) {
	_, err := NewAFMParser(openFixture(t, afmFixture+"NoEndFontMetrics.afm")).Parse()
	if err == nil {
		t.Fatal("parsing a file with no EndFontMetrics succeeded, want an error")
	}
	if !strings.Contains(err.Error(), "Unknown AFM key") {
		t.Errorf("err = %v, want it to say Unknown AFM key", err)
	}
}

func TestMalformedFloat(t *testing.T) {
	_, err := NewAFMParser(openFixture(t, afmFixture+"MalformedFloat.afm")).Parse()
	if err == nil {
		t.Fatal("parsing a malformed float succeeded, want an error")
	}
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Errorf("err = %v, want a number conversion error underneath", err)
	}
	if !strings.Contains(err.Error(), "4,1ab") {
		t.Errorf("err = %v, want it to quote the offending value 4,1ab", err)
	}
}

func TestMalformedInteger(t *testing.T) {
	_, err := NewAFMParser(openFixture(t, afmFixture+"MalformedInteger.afm")).Parse()
	if err == nil {
		t.Fatal("parsing a malformed integer succeeded, want an error")
	}
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Errorf("err = %v, want a number conversion error underneath", err)
	}
	if !strings.Contains(err.Error(), "3.4") {
		t.Errorf("err = %v, want it to quote the offending value 3.4", err)
	}
}

func TestHelveticaFontMetrics(t *testing.T) {
	fontMetrics, err := NewAFMParser(openFixture(t, helveticaAFM)).Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checkHelveticaFontMetrics(t, fontMetrics)
}

func TestHelveticaCharMetrics(t *testing.T) {
	fontMetrics, err := NewAFMParser(openFixture(t, helveticaAFM)).Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	checkHelveticaCharMetrics(t, fontMetrics.CharMetrics())
}

func TestHelveticaKernPairs(t *testing.T) {
	fontMetrics, err := NewAFMParser(openFixture(t, helveticaAFM)).Parse()
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	kernPairs := fontMetrics.KernPairs()
	if len(kernPairs) != 2705 {
		t.Errorf("got %d kern pairs, want 2705", len(kernPairs))
	}
	// check "KPX A Ucircumflex -50"
	checkKernPair(t, kernPairs, "A", "Ucircumflex", -50, 0)
	// check "KPX W agrave -40"
	checkKernPair(t, kernPairs, "W", "agrave", -40, 0)

	if got := fontMetrics.KernPairs0(); len(got) != 0 {
		t.Errorf("KernPairs0() = %v, want none", got)
	}
	if got := fontMetrics.KernPairs1(); len(got) != 0 {
		t.Errorf("KernPairs1() = %v, want none", got)
	}
	if got := fontMetrics.Composites(); len(got) != 0 {
		t.Errorf("Composites() = %v, want none", got)
	}
}

func TestHelveticaFontMetricsReducedDataset(t *testing.T) {
	fontMetrics, err := NewAFMParser(openFixture(t, helveticaAFM)).ParseReduced(true)
	if err != nil {
		t.Fatalf("ParseReduced: %v", err)
	}
	checkHelveticaFontMetrics(t, fontMetrics)
}

func TestHelveticaCharMetricsReducedDataset(t *testing.T) {
	fontMetrics, err := NewAFMParser(openFixture(t, helveticaAFM)).ParseReduced(true)
	if err != nil {
		t.Fatalf("ParseReduced: %v", err)
	}
	checkHelveticaCharMetrics(t, fontMetrics.CharMetrics())
}

func TestHelveticaKernPairsReducedDataset(t *testing.T) {
	fontMetrics, err := NewAFMParser(openFixture(t, helveticaAFM)).ParseReduced(true)
	if err != nil {
		t.Fatalf("ParseReduced: %v", err)
	}

	// KernPairs, empty due to reducedDataset == true
	if got := fontMetrics.KernPairs(); len(got) != 0 {
		t.Errorf("KernPairs() = %d entries, want none under a reduced parse", len(got))
	}
	if got := fontMetrics.KernPairs0(); len(got) != 0 {
		t.Errorf("KernPairs0() = %v, want none", got)
	}
	if got := fontMetrics.KernPairs1(); len(got) != 0 {
		t.Errorf("KernPairs1() = %v, want none", got)
	}
	if got := fontMetrics.Composites(); len(got) != 0 {
		t.Errorf("Composites() = %v, want none", got)
	}
}

func checkHelveticaCharMetrics(t *testing.T, charMetrics []*CharMetric) {
	t.Helper()
	if len(charMetrics) != 315 {
		t.Fatalf("got %d char metrics, want 315", len(charMetrics))
	}

	for _, c := range []struct {
		name               string
		wx                 float32
		code               int
		llx, lly, urx, ury float32
	}{
		{"space", 278, 32, 0, 0, 0, 0},
		{"ring", 333, 202, 75, 572, 259, 756},
	} {
		metric := findCharMetric(charMetrics, c.name)
		if metric == nil {
			t.Fatalf("no char metric named %q", c.name)
		}
		if got := metric.Wx(); got != c.wx {
			t.Errorf("%s Wx() = %v, want %v", c.name, got, c.wx)
		}
		if got := metric.CharacterCode(); got != c.code {
			t.Errorf("%s CharacterCode() = %v, want %v", c.name, got, c.code)
		}
		checkBBox(t, metric.BoundingBox(), c.llx, c.lly, c.urx, c.ury)
		if got := metric.Ligatures(); len(got) != 0 {
			t.Errorf("%s Ligatures() = %v, want none", c.name, got)
		}
		if metric.W() != nil || metric.W0() != nil || metric.W1() != nil || metric.Vv() != nil {
			t.Errorf("%s carries a width vector it should not", c.name)
		}
	}
}

func findCharMetric(charMetrics []*CharMetric, name string) *CharMetric {
	for _, metric := range charMetrics {
		if metric.Name() == name {
			return metric
		}
	}
	return nil
}

func checkHelveticaFontMetrics(t *testing.T, fontMetrics *FontMetrics) {
	t.Helper()

	if got := fontMetrics.AFMVersion(); got != 4.1 {
		t.Errorf("AFMVersion() = %v, want 4.1", got)
	}
	for _, c := range []struct {
		name string
		got  string
		want string
	}{
		{"FontName", fontMetrics.FontName(), "Helvetica"},
		{"FullName", fontMetrics.FullName(), "Helvetica"},
		{"FamilyName", fontMetrics.FamilyName(), "Helvetica"},
		{"Weight", fontMetrics.Weight(), "Medium"},
		{"FontVersion", fontMetrics.FontVersion(), "002.000"},
		{"EncodingScheme", fontMetrics.EncodingScheme(), "AdobeStandardEncoding"},
		{"CharacterSet", fontMetrics.CharacterSet(), "ExtendedRoman"},
	} {
		if c.got != c.want {
			t.Errorf("%s() = %q, want %q", c.name, c.got, c.want)
		}
	}

	checkBBox(t, fontMetrics.FontBBox(), -166, -225, 1000, 931)

	const notice = "Copyright (c) 1985, 1987, 1989, 1990, 1997 Adobe Systems Incorporated.  " +
		"All Rights Reserved.Helvetica is a trademark of Linotype-Hell AG and/or its subsidiaries."
	if got := fontMetrics.Notice(); got != notice {
		t.Errorf("Notice() = %q, want %q", got, notice)
	}

	if got := fontMetrics.MappingScheme(); got != 0 {
		t.Errorf("MappingScheme() = %v, want 0", got)
	}
	if got := fontMetrics.EscChar(); got != 0 {
		t.Errorf("EscChar() = %v, want 0", got)
	}
	if got := fontMetrics.Characters(); got != 0 {
		t.Errorf("Characters() = %v, want 0", got)
	}
	if !fontMetrics.IsBaseFont() {
		t.Error("IsBaseFont() = false, want true")
	}
	if got := fontMetrics.VVector(); got != nil {
		t.Errorf("VVector() = %v, want nil", got)
	}
	if fontMetrics.IsFixedV() {
		t.Error("IsFixedV() = true, want false")
	}

	for _, c := range []struct {
		name string
		got  float32
		want float32
	}{
		{"CapHeight", fontMetrics.CapHeight(), 718},
		{"XHeight", fontMetrics.XHeight(), 523},
		{"Ascender", fontMetrics.Ascender(), 718},
		{"Descender", fontMetrics.Descender(), -207},
		{"StandardHorizontalWidth", fontMetrics.StandardHorizontalWidth(), 76},
		{"StandardVerticalWidth", fontMetrics.StandardVerticalWidth(), 88},
		{"UnderlinePosition", fontMetrics.UnderlinePosition(), -100},
		{"UnderlineThickness", fontMetrics.UnderlineThickness(), 50},
		{"ItalicAngle", fontMetrics.ItalicAngle(), 0},
	} {
		if c.got != c.want {
			t.Errorf("%s() = %v, want %v", c.name, c.got, c.want)
		}
	}

	comments := fontMetrics.Comments()
	if len(comments) != 4 {
		t.Fatalf("got %d comments, want 4", len(comments))
	}
	const firstComment = "Copyright (c) 1985, 1987, 1989, 1990, 1997 Adobe Systems " +
		"Incorporated.  All Rights Reserved."
	if comments[0] != firstComment {
		t.Errorf("comments[0] = %q, want %q", comments[0], firstComment)
	}
	if comments[2] != "UniqueID 43054" {
		t.Errorf("comments[2] = %q, want %q", comments[2], "UniqueID 43054")
	}

	if got := fontMetrics.CharWidth(); got != nil {
		t.Errorf("CharWidth() = %v, want nil", got)
	}
	if fontMetrics.IsFixedPitch() {
		t.Error("IsFixedPitch() = true, want false")
	}
}

func checkBBox(t *testing.T, bBox *util.BoundingBox, lowerX, lowerY, upperX, upperY float32) {
	t.Helper()
	if bBox == nil {
		t.Fatal("the bounding box is nil")
	}
	if bBox.LowerLeftX() != lowerX || bBox.LowerLeftY() != lowerY ||
		bBox.UpperRightX() != upperX || bBox.UpperRightY() != upperY {
		t.Errorf("bounding box = %v, want [%v,%v,%v,%v]",
			bBox, lowerX, lowerY, upperX, upperY)
	}
}

func checkKernPair(t *testing.T, kernPairs []*KernPair, firstKernChar, secondKernChar string, x, y float32) {
	t.Helper()
	for _, pair := range kernPairs {
		if pair.FirstKernCharacter() != firstKernChar ||
			pair.SecondKernCharacter() != secondKernChar {
			continue
		}
		if pair.X() != x {
			t.Errorf("kern pair %s %s X() = %v, want %v", firstKernChar, secondKernChar, pair.X(), x)
		}
		if pair.Y() != y {
			t.Errorf("kern pair %s %s Y() = %v, want %v", firstKernChar, secondKernChar, pair.Y(), y)
		}
		return
	}
	t.Errorf("no kern pair for %s %s", firstKernChar, secondKernChar)
}
