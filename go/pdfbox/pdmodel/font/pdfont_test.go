package font

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
)

// Port of org.apache.pdfbox.pdmodel.font.PDFontTest.
//
// Only testPDFox4318 needs no document: every other test in that class builds a
// PDDocument, saves it, loads it back, or runs the text stripper, none of which
// this slice carries. See migration/tasks/slice-3-text-simple-fonts.md.

// TestPDFox4318 checks that a code point the encoding does not carry is
// rejected, and stays rejected after a code point it does carry was encoded.
//
// The second check is the point of the test: PDType1Font caches what it
// encodes, and the bug was that the cache made the second call succeed.
func TestPDFox4318(t *testing.T) {
	helveticaBold, err := NewPDType1FontStandard14(HelveticaBold)
	if err != nil {
		t.Fatalf("Helvetica-Bold: %v", err)
	}

	if _, err := helveticaBold.Encode(""); err == nil {
		t.Error("U+0080 was encoded, should have failed")
	}
	if _, err := helveticaBold.Encode("€"); err != nil {
		t.Errorf("euro sign: %v", err)
	}
	if _, err := helveticaBold.Encode(""); err == nil {
		t.Error("U+0080 was encoded after the euro sign, should have failed")
	}
}

// Java has no test for Standard14Fonts, PDFontDescriptor or PDFontFactory, so
// what follows is written from their sources and from the AFM files the
// standard 14 fonts are read out of.

func TestStandard14Names(t *testing.T) {
	// the names of the fourteen themselves, and a few of the aliases Java maps
	cases := []struct {
		name string
		want FontName
	}{
		{"Helvetica", Helvetica},
		{"Times-Roman", TimesRoman},
		{"ZapfDingbats", ZapfDingbatsFontName},
		{"Arial", Helvetica},
		{"Arial,BoldItalic", HelveticaBoldOblique},
		{"ArialMT", Helvetica},
		{"Arial-BoldItalicMT", HelveticaBoldOblique},
		{"TimesNewRoman,Bold", TimesBold},
		{"Times,Italic", TimesItalic},
		{"Symbol,Bold", SymbolFontName},
		{"CourierNew,BoldItalic", CourierBoldOblique},
	}
	for _, c := range cases {
		if !Standard14ContainsName(c.name) {
			t.Errorf("Standard14ContainsName(%q) = false", c.name)
			continue
		}
		got, ok := GetMappedFontName(c.name)
		if !ok || got != c.want {
			t.Errorf("GetMappedFontName(%q) = %v, want %v", c.name, got, c.want)
		}
	}

	if Standard14ContainsName("NotAFont") {
		t.Error("Standard14ContainsName(\"NotAFont\") = true")
	}
	if _, ok := GetMappedFontName("NotAFont"); ok {
		t.Error("GetMappedFontName(\"NotAFont\") found a font")
	}

	// Java maps 14 fonts plus 24 aliases
	if got := len(Standard14Names()); got != 38 {
		t.Errorf("len(Standard14Names()) = %d, want 38", got)
	}
}

// TestStandard14AFM checks that the AFM of a standard 14 font is read, using
// values taken from the AFM file itself.
func TestStandard14AFM(t *testing.T) {
	metrics := GetAFM("Helvetica")
	if metrics == nil {
		t.Fatal("no AFM for Helvetica")
	}
	if got := metrics.FontName(); got != "Helvetica" {
		t.Errorf("FontName() = %q, want %q", got, "Helvetica")
	}
	// Helvetica.afm: "C 32 ; WX 278 ; N space ; B 0 0 0 0 ;"
	if got := metrics.CharacterWidth("space"); got != 278 {
		t.Errorf("width of space = %v, want 278", got)
	}
	// Helvetica.afm: "C 65 ; WX 667 ; N A ; B 14 0 654 718 ;"
	if got := metrics.CharacterWidth("A"); got != 667 {
		t.Errorf("width of A = %v, want 667", got)
	}
	if got := metrics.EncodingScheme(); got != "AdobeStandardEncoding" {
		t.Errorf("EncodingScheme() = %q, want %q", got, "AdobeStandardEncoding")
	}

	// an alias reads the same metrics
	if GetAFM("Arial") != metrics {
		t.Error("Arial and Helvetica gave different metrics")
	}

	if GetAFM("NotAFont") != nil {
		t.Error("GetAFM(\"NotAFont\") returned metrics")
	}
}

// TestStandard14Widths checks the widths a standard 14 font reports, which come
// from its AFM through the WinAnsi encoding it is given.
func TestStandard14Widths(t *testing.T) {
	helvetica, err := NewPDType1FontStandard14(Helvetica)
	if err != nil {
		t.Fatalf("Helvetica: %v", err)
	}
	if !helvetica.IsStandard14() {
		t.Fatal("Helvetica is not a standard 14 font")
	}

	// code 32 is space in WinAnsiEncoding; Helvetica.afm gives it WX 278
	width, err := helvetica.Width(32)
	if err != nil {
		t.Fatalf("width of code 32: %v", err)
	}
	if width != 278 {
		t.Errorf("width of code 32 = %v, want 278", width)
	}
	// code 65 is A; Helvetica.afm gives it WX 667
	width, err = helvetica.Width(65)
	if err != nil {
		t.Fatalf("width of code 65: %v", err)
	}
	if width != 667 {
		t.Errorf("width of code 65 = %v, want 667", width)
	}

	if got := helvetica.SpaceWidth(); got != 278 {
		t.Errorf("SpaceWidth() = %v, want 278", got)
	}
}

// TestStandard14Encodings checks which encoding each of the three kinds of
// standard 14 font is given, which PDType1Font's constructor sets outright.
func TestStandard14Encodings(t *testing.T) {
	cases := []struct {
		font FontName
		want string
	}{
		{Helvetica, "WinAnsiEncoding"},
		{TimesRoman, "WinAnsiEncoding"},
		{SymbolFontName, "SymbolEncoding"},
		{ZapfDingbatsFontName, "ZapfDingbatsEncoding"},
	}
	for _, c := range cases {
		f, err := NewPDType1FontStandard14(c.font)
		if err != nil {
			t.Fatalf("%v: %v", c.font, err)
		}
		if got := f.Encoding().EncodingName(); got != c.want {
			t.Errorf("%v encoding = %q, want %q", c.font, got, c.want)
		}
		if got := f.Name(); got != c.font.Name() {
			t.Errorf("%v name = %q, want %q", c.font, got, c.font.Name())
		}
		if got := f.SubType(); got != "Type1" {
			t.Errorf("%v subtype = %q, want %q", c.font, got, "Type1")
		}
	}
}

// TestStandard14GlyphList checks that Zapf Dingbats is read through its own
// glyph list and the others through the Adobe Glyph List.
func TestStandard14GlyphList(t *testing.T) {
	zapf, err := NewPDType1FontStandard14(ZapfDingbatsFontName)
	if err != nil {
		t.Fatalf("ZapfDingbats: %v", err)
	}
	if zapf.GlyphList() != encoding.ZapfDingbats() {
		t.Error("ZapfDingbats is not read through the Zapf Dingbats glyph list")
	}

	helvetica, err := NewPDType1FontStandard14(Helvetica)
	if err != nil {
		t.Fatalf("Helvetica: %v", err)
	}
	if helvetica.GlyphList() != encoding.AdobeGlyphList() {
		t.Error("Helvetica is not read through the Adobe Glyph List")
	}
}

// TestStandard14ToUnicode checks the fallback path this slice relies on: with
// no ToUnicode CMap, a simple font maps its code through its encoding and then
// through the glyph list.
func TestStandard14ToUnicode(t *testing.T) {
	helvetica, err := NewPDType1FontStandard14(Helvetica)
	if err != nil {
		t.Fatalf("Helvetica: %v", err)
	}
	cases := []struct {
		code int
		want string
	}{
		{65, "A"},  // WinAnsi 65 -> "A" -> U+0041
		{32, " "},  // WinAnsi 32 -> "space" -> U+0020
		{128, "€"}, // WinAnsi 128 -> "Euro" -> U+20AC
	}
	for _, c := range cases {
		got, err := helvetica.ToUnicode(c.code)
		if err != nil {
			t.Fatalf("ToUnicode(%d): %v", c.code, err)
		}
		if got != c.want {
			t.Errorf("ToUnicode(%d) = %q, want %q", c.code, got, c.want)
		}
	}
}

// TestStandard14Encode checks that a standard 14 font encodes text back to the
// codes of its encoding.
func TestStandard14Encode(t *testing.T) {
	helvetica, err := NewPDType1FontStandard14(Helvetica)
	if err != nil {
		t.Fatalf("Helvetica: %v", err)
	}
	got, err := helvetica.Encode("AB ")
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if string(got) != "AB " {
		t.Errorf("Encode(%q) = %q, want %q", "AB ", got, "AB ")
	}

	// the widths of A, B and space in Helvetica.afm: 667 + 667 + 278
	width, err := helvetica.StringWidth("AB ")
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}
	if width != 667+667+278 {
		t.Errorf("StringWidth(%q) = %v, want %v", "AB ", width, 667+667+278)
	}
}

// TestPDFontDescriptorFlags checks each flag bit against the value Java gives
// it.
func TestPDFontDescriptorFlags(t *testing.T) {
	fd := NewPDFontDescriptor()
	cases := []struct {
		name string
		set  func(bool)
		get  func() bool
		bit  int
	}{
		{"FixedPitch", fd.SetFixedPitch, fd.IsFixedPitch, 1},
		{"Serif", fd.SetSerif, fd.IsSerif, 2},
		{"Symbolic", fd.SetSymbolic, fd.IsSymbolic, 4},
		{"Script", fd.SetScript, fd.IsScript, 8},
		{"NonSymbolic", fd.SetNonSymbolic, fd.IsNonSymbolic, 32},
		{"Italic", fd.SetItalic, fd.IsItalic, 64},
		{"AllCap", fd.SetAllCap, fd.IsAllCap, 65536},
		{"SmallCap", fd.SetSmallCap, fd.IsSmallCap, 131072},
		{"ForceBold", fd.SetForceBold, fd.IsForceBold, 262144},
	}
	for _, c := range cases {
		c.set(true)
		if !c.get() {
			t.Errorf("%s was set but reads false", c.name)
		}
		if fd.Flags()&c.bit == 0 {
			t.Errorf("%s did not set bit %d", c.name, c.bit)
		}
		c.set(false)
		if c.get() {
			t.Errorf("%s was cleared but reads true", c.name)
		}
		if fd.Flags()&c.bit != 0 {
			t.Errorf("%s did not clear bit %d", c.name, c.bit)
		}
	}
}

// TestPDFontDescriptorHeightsAreAbsolute checks that a negative cap height or x
// height is read as its magnitude, which is what Java's Math.abs does.
func TestPDFontDescriptorHeightsAreAbsolute(t *testing.T) {
	dic := cos.NewDictionary()
	dic.SetFloat(cos.CapHeight, -700)
	dic.SetFloat(cos.XHeight, -500)
	fd := NewPDFontDescriptorFromDictionary(dic)
	if got := fd.CapHeight(); got != 700 {
		t.Errorf("CapHeight() = %v, want 700", got)
	}
	if got := fd.XHeight(); got != 500 {
		t.Errorf("XHeight() = %v, want 500", got)
	}
}

// TestPDFontDescriptorFromAFM checks the descriptor a standard 14 font is given,
// which is built from its AFM.
func TestPDFontDescriptorFromAFM(t *testing.T) {
	helvetica, err := NewPDType1FontStandard14(Helvetica)
	if err != nil {
		t.Fatalf("Helvetica: %v", err)
	}
	fd := helvetica.FontDescriptor()
	if fd == nil {
		t.Fatal("Helvetica has no font descriptor")
	}
	if got := fd.FontName(); got != "Helvetica" {
		t.Errorf("FontName() = %q, want %q", got, "Helvetica")
	}
	// Helvetica.afm: FamilyName Helvetica, EncodingScheme AdobeStandardEncoding
	if got := fd.FontFamily(); got != "Helvetica" {
		t.Errorf("FontFamily() = %q, want %q", got, "Helvetica")
	}
	if !fd.IsNonSymbolic() {
		t.Error("Helvetica is not marked non-symbolic")
	}
	if fd.IsSymbolic() {
		t.Error("Helvetica is marked symbolic")
	}
	// Helvetica.afm: FontBBox -166 -225 1000 931
	bbox := fd.FontBoundingBox()
	if bbox == nil {
		t.Fatal("no font bounding box")
	}
	if bbox.LowerLeftX() != -166 || bbox.LowerLeftY() != -225 ||
		bbox.UpperRightX() != 1000 || bbox.UpperRightY() != 931 {
		t.Errorf("FontBoundingBox() = %v, want [-166 -225 1000 931]", bbox)
	}
	// Helvetica.afm: CapHeight 718, XHeight 523, ItalicAngle 0
	if got := fd.CapHeight(); got != 718 {
		t.Errorf("CapHeight() = %v, want 718", got)
	}
	if got := fd.XHeight(); got != 523 {
		t.Errorf("XHeight() = %v, want 523", got)
	}
	if got := fd.ItalicAngle(); got != 0 {
		t.Errorf("ItalicAngle() = %v, want 0", got)
	}
	// for PDF/A
	if got := fd.StemV(); got != 0 {
		t.Errorf("StemV() = %v, want 0", got)
	}

	// Symbol is symbolic: its AFM says EncodingScheme FontSpecific
	symbol, err := NewPDType1FontStandard14(SymbolFontName)
	if err != nil {
		t.Fatalf("Symbol: %v", err)
	}
	if !symbol.FontDescriptor().IsSymbolic() {
		t.Error("Symbol is not marked symbolic")
	}
	if symbol.FontDescriptor().IsNonSymbolic() {
		t.Error("Symbol is marked non-symbolic")
	}
}

// TestCreateFontType1 checks that the factory reads a Type 1 dictionary as a
// Type 1 font.
func TestCreateFontType1(t *testing.T) {
	dict := cos.NewDictionary()
	dict.SetItem(cos.Type, cos.Font)
	dict.SetItem(cos.Subtype, cos.Type1)
	dict.SetName(cos.BaseFont, "Helvetica")

	font, err := CreateFont(dict, nil)
	if err != nil {
		t.Fatalf("CreateFont: %v", err)
	}
	if _, ok := font.(*PDType1Font); !ok {
		t.Fatalf("CreateFont gave a %T, want *PDType1Font", font)
	}
	if got := font.Name(); got != "Helvetica" {
		t.Errorf("Name() = %q, want %q", got, "Helvetica")
	}
	if !font.IsStandard14() {
		t.Error("Helvetica read from a dictionary is not a standard 14 font")
	}
}

// TestCreateFontUnknownSubtype checks the PDFBOX-1988 fallback: an unknown
// subtype is read as a Type 1 font, because Adobe Reader does that.
func TestCreateFontUnknownSubtype(t *testing.T) {
	dict := cos.NewDictionary()
	dict.SetItem(cos.Type, cos.Font)
	dict.SetItem(cos.Subtype, cos.GetPDFName("NoSuchSubtype"))
	dict.SetName(cos.BaseFont, "Helvetica")

	font, err := CreateFont(dict, nil)
	if err != nil {
		t.Fatalf("CreateFont: %v", err)
	}
	if _, ok := font.(*PDType1Font); !ok {
		t.Fatalf("CreateFont gave a %T, want *PDType1Font", font)
	}
}

// TestCreateFontDescendantNotAllowed checks that a descendant font on its own
// is rejected, which Java does with the same two messages.
func TestCreateFontDescendantNotAllowed(t *testing.T) {
	for _, c := range []struct {
		subType *cos.Name
		want    string
	}{
		{cos.CIDFontType0, "Type 0 descendant font not allowed"},
		{cos.CIDFontType2, "Type 2 descendant font not allowed"},
	} {
		dict := cos.NewDictionary()
		dict.SetItem(cos.Type, cos.Font)
		dict.SetItem(cos.Subtype, c.subType)
		_, err := CreateFont(dict, nil)
		if err == nil {
			t.Errorf("%v was accepted", c.subType)
			continue
		}
		if !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v error = %v, want it to say %q", c.subType, err, c.want)
		}
	}
}

// TestCreateFontType3 checks that the factory reads a Type 3 dictionary as a
// Type 3 font, and that the font reads its own font matrix.
func TestCreateFontType3(t *testing.T) {
	dict := cos.NewDictionary()
	dict.SetItem(cos.Type, cos.Font)
	dict.SetItem(cos.Subtype, cos.Type3)
	matrix := cos.NewArray()
	for _, v := range []float32{0.01, 0, 0, 0.01, 0, 0} {
		matrix.Add(cos.NewFloat(v))
	}
	dict.SetItem(cos.FontMatrix, matrix)

	font, err := CreateFont(dict, nil)
	if err != nil {
		t.Fatalf("CreateFont: %v", err)
	}
	type3, ok := font.(*PDType3Font)
	if !ok {
		t.Fatalf("CreateFont gave a %T, want *PDType3Font", font)
	}
	if !type3.IsEmbedded() {
		t.Error("a Type 3 font reports itself as not embedded")
	}
	if type3.IsStandard14() {
		t.Error("a Type 3 font reports itself as a standard 14 font")
	}
	if got := type3.FontMatrix().Value(0, 0); got != 0.01 {
		t.Errorf("font matrix [0][0] = %v, want 0.01", got)
	}
}
