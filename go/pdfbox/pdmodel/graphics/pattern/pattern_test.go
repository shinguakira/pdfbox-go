package pattern

// PDFBox has no test for this package, so these are written from the Java
// source and from the pattern sections of PDF32000_2008.pdf.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/filter"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/shading"
)

// TestPatternDispatch checks that each /PatternType builds its own type, which
// is the switch of PDAbstractPattern.create.
func TestPatternDispatch(t *testing.T) {
	tiling := cos.NewDictionary()
	tiling.SetInt(cos.PatternType, TypeTilingPattern)
	created, err := NewPDAbstractPattern(tiling, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, isTiling := created.(*PDTilingPattern); !isTiling {
		t.Errorf("pattern type 1 = %T, want *PDTilingPattern", created)
	}
	if got := created.PatternType(); got != TypeTilingPattern {
		t.Errorf("PatternType() = %d, want %d", got, TypeTilingPattern)
	}

	shadingDict := cos.NewDictionary()
	shadingDict.SetInt(cos.PatternType, TypeShadingPattern)
	created, err = NewPDAbstractPattern(shadingDict, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, isShading := created.(*PDShadingPattern); !isShading {
		t.Errorf("pattern type 2 = %T, want *PDShadingPattern", created)
	}
	if got := created.PatternType(); got != TypeShadingPattern {
		t.Errorf("PatternType() = %d, want %d", got, TypeShadingPattern)
	}
}

// TestPatternRefusesAnUnknownType is the default arm of the same switch, which
// Java throws from. A dictionary with no /PatternType reads as type 0.
func TestPatternRefusesAnUnknownType(t *testing.T) {
	for _, patternType := range []int{0, 3} {
		dict := cos.NewDictionary()
		if patternType != 0 {
			dict.SetInt(cos.PatternType, patternType)
		}
		if _, err := NewPDAbstractPattern(dict, nil); err == nil {
			t.Errorf("pattern type %d = nil error, want one", patternType)
		}
	}
}

// TestTilingPatternAccessors round trips every entry a tiling pattern carries.
func TestTilingPatternAccessors(t *testing.T) {
	p := NewPDTilingPattern(filter.Provider{})
	if got := p.PatternType(); got != TypeTilingPattern {
		t.Errorf("PatternType() = %d, want %d", got, TypeTilingPattern)
	}
	// The constructor writes /Type, /PatternType and the resources the
	// specification requires.
	if got := p.Dictionary().GetNameAsString(cos.Type, ""); got != "Pattern" {
		t.Errorf("/Type = %q, want %q", got, "Pattern")
	}
	if p.Resources() == nil {
		t.Error("Resources() = nil, want the empty resources the constructor sets")
	}

	p.SetPaintType(PaintUncolored)
	if got := p.PaintType(); got != PaintUncolored {
		t.Errorf("PaintType() = %d, want %d", got, PaintUncolored)
	}
	p.SetTilingType(TilingNoDistortion)
	if got := p.TilingType(); got != TilingNoDistortion {
		t.Errorf("TilingType() = %d, want %d", got, TilingNoDistortion)
	}
	p.SetXStep(12.5)
	if got := p.XStep(); got != 12.5 {
		t.Errorf("XStep() = %v, want 12.5", got)
	}
	p.SetYStep(-3)
	if got := p.YStep(); got != -3 {
		t.Errorf("YStep() = %v, want -3", got)
	}

	// A pattern with no /BBox answers nil, and the entry round trips.
	if got := p.BBox(); got != nil {
		t.Errorf("BBox() = %v, want nil before one is set", got)
	}
	bbox := cos.NewArray()
	for _, v := range []float32{0, 0, 10, 20} {
		bbox.Add(cos.NewFloat(v))
	}
	p.Dictionary().SetItem(cos.BBox, bbox)
	if got := p.BBox(); got == nil || got.Width() != 10 || got.Height() != 20 {
		t.Errorf("BBox() = %v, want 10 by 20", got)
	}
}

// TestTilingPatternWithoutAStreamHasNoContents is Java's
// `dict instanceof COSStream` guard in getContentsForRandomAccess: a tiling
// pattern read from a plain dictionary carries no content.
func TestTilingPatternWithoutAStreamHasNoContents(t *testing.T) {
	dict := cos.NewDictionary()
	dict.SetInt(cos.PatternType, TypeTilingPattern)
	p := NewPDTilingPatternOf(dict)
	contents, err := p.ContentsForRandomAccess()
	if err != nil {
		t.Fatal(err)
	}
	if contents != nil {
		t.Error("ContentsForRandomAccess() = non-nil, want nil for a plain dictionary")
	}
	if p.ContentStream() != nil {
		t.Error("ContentStream() = non-nil, want nil for a plain dictionary")
	}
}

// TestTilingPatternOfAStreamHasContents is the other half: a pattern built over
// a stream answers that stream's bytes.
func TestTilingPatternOfAStreamHasContents(t *testing.T) {
	p := NewPDTilingPattern(filter.Provider{})
	w, err := p.ContentStream().Stream().CreateWriter()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("0 0 10 10 re f")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	contents, err := p.ContentsForRandomAccess()
	if err != nil {
		t.Fatal(err)
	}
	if contents == nil {
		t.Fatal("ContentsForRandomAccess() = nil, want the content that was written")
	}
	length, err := contents.Length()
	if err != nil {
		t.Fatal(err)
	}
	if length == 0 {
		t.Error("the content is empty")
	}
}

// TestShadingPatternShading checks that a shading pattern builds the shading
// its /Shading entry names, and caches it.
func TestShadingPatternShading(t *testing.T) {
	p := NewPDShadingPattern()
	if got := p.PatternType(); got != TypeShadingPattern {
		t.Errorf("PatternType() = %d, want %d", got, TypeShadingPattern)
	}
	got, err := p.Shading()
	if err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Errorf("Shading() = %v, want nil before one is set", got)
	}

	axial := cos.NewDictionary()
	axial.SetInt(cos.ShadingType, shading.ShadingType2)
	p.Dictionary().SetItem(cos.Shading, axial)
	got, err = p.Shading()
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ShadingType() != shading.ShadingType2 {
		t.Fatalf("Shading() = %v, want an axial shading", got)
	}
	// The second call answers the cached one, which is the field Java keeps.
	again, err := p.Shading()
	if err != nil {
		t.Fatal(err)
	}
	if again != got {
		t.Error("Shading() built a second shading, want the cached one")
	}
}

// TestPatternColorSpaceIsRegistered checks the registry this package fills from
// its init: graphics/color cannot name PDPattern, so Create reaches it through
// the constructor set there.
func TestPatternColorSpaceIsRegistered(t *testing.T) {
	created, err := color.Create(cos.Pattern)
	if err != nil {
		t.Fatalf("Create(/Pattern) = %v, want the pattern colour space", err)
	}
	if _, isPattern := created.(*PDPattern); !isPattern {
		t.Fatalf("Create(/Pattern) = %T, want *PDPattern", created)
	}
	if got := created.Name(); got != "Pattern" {
		t.Errorf("Name() = %q, want %q", got, "Pattern")
	}
	// A coloured pattern has no underlying colour space.
	if got := created.(color.PatternColorSpace).UnderlyingColorSpace(); got != nil {
		t.Errorf("UnderlyingColorSpace() = %v, want nil for a coloured pattern", got)
	}
	// The initial colour leaves no marks: no components and no pattern name.
	initial := created.InitialColor()
	if len(initial.Components()) != 0 {
		t.Errorf("InitialColor() has %d components, want none", len(initial.Components()))
	}
}

// TestPatternColorSpaceOfAnArrayCarriesTheUnderlyingSpace is the array arm:
// [/Pattern /DeviceRGB] is an uncoloured tiling pattern painted in DeviceRGB.
func TestPatternColorSpaceOfAnArrayCarriesTheUnderlyingSpace(t *testing.T) {
	array := cos.NewArray()
	array.Add(cos.Pattern)
	array.Add(cos.DeviceRGB)
	created, err := color.Create(array)
	if err != nil {
		t.Fatal(err)
	}
	underlying := created.(color.PatternColorSpace).UnderlyingColorSpace()
	if underlying == nil {
		t.Fatal("UnderlyingColorSpace() = nil, want DeviceRGB")
	}
	if got := underlying.Name(); got != "DeviceRGB" {
		t.Errorf("UnderlyingColorSpace().Name() = %q, want %q", got, "DeviceRGB")
	}
	// A one element array is the coloured form and carries none.
	oneElement := cos.NewArray()
	oneElement.Add(cos.Pattern)
	created, err = color.Create(oneElement)
	if err != nil {
		t.Fatal(err)
	}
	if got := created.(color.PatternColorSpace).UnderlyingColorSpace(); got != nil {
		t.Errorf("UnderlyingColorSpace() = %v, want nil", got)
	}
}

// TestPatternColorSpaceRefusesTheComponentMethods pins the six methods Java
// declares as UnsupportedOperationException: a pattern has no components, so
// nothing that reads them can answer.
func TestPatternColorSpaceRefusesTheComponentMethods(t *testing.T) {
	created, err := color.Create(cos.Pattern)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		what string
		call func()
	}{
		{"NumberOfComponents", func() { created.NumberOfComponents() }},
		{"DefaultDecode", func() { created.DefaultDecode(8) }},
		{"ToRGB", func() { _, _ = created.ToRGB([]float32{0}) }},
		{"ToRGBImage", func() { _, _ = created.ToRGBImage(nil) }},
		{"ToRawImage", func() { _, _ = created.ToRawImage(nil) }},
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", c.what)
				}
			}()
			c.call()
		}()
	}
}

// TestPatternNotFound is the error PDPattern.getPattern raises where the
// resources hold nothing under the colour's pattern name.
func TestPatternNotFound(t *testing.T) {
	p := NewPDPattern(nil)
	c := color.NewPDColorOfPattern(cos.GetPDFName("P0"), nil)
	if _, err := p.Pattern(c); err == nil {
		t.Error("Pattern() = nil error, want one naming the missing pattern")
	}
}
