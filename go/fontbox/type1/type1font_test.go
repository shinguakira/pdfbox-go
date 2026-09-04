package type1

import (
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/encoding"
)

// Port of the part of org.apache.fontbox.pfb.PfbParserTest that goes through
// Type1Font. The three tests that read a real font need one the Java build
// downloads into target/fonts, so those skip when it is not present.

const fontFixture = "../../../fontbox/target/fonts/"

// TestEmpty tests a 0 length font.
func TestEmpty(t *testing.T) {
	_, err := CreateWithPFBBytes([]byte{})
	if err == nil {
		t.Fatal("an empty PFB was accepted")
	}
	if got, want := err.Error(), "Start marker missing"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

// TestMiscBadFonts tests some bad fonts.
func TestMiscBadFonts(t *testing.T) {
	ba := make([]byte, 18+1) // PfbParser.PFB_HEADER_LENGTH + 1
	_, err := CreateWithPFBBytes(ba)
	if err == nil {
		t.Fatal("a PFB without a start marker was accepted")
	}
	if got, want := err.Error(), "Start marker missing"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}

	ba[0] = 0x80 // PfbParser.START_MARKER
	ba[1] = 33
	_, err = CreateWithPFBBytes(ba)
	if err == nil {
		t.Fatal("a PFB with record type 33 was accepted")
	}
	if got, want := err.Error(), "Incorrect record type: 33"; got != want {
		t.Errorf("error = %q, want %q", got, want)
	}
}

// openFontFixture opens one of the fonts the Java build downloads.
func openFontFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(fontFixture + name)
	if err != nil {
		t.Skipf("the font the Java build downloads is not present: %v", err)
	}
	return data
}

// TestPfb tests parsing a PFB font.
func TestPfb(t *testing.T) {
	font, err := CreateWithPFBBytes(openFontFixture(t, "OpenSans-Regular.pfb"))
	if err != nil {
		t.Fatalf("CreateWithPFB: %v", err)
	}
	assertString(t, "Version", font.Version(), "1.10")
	assertString(t, "FontName", font.FontName(), "OpenSans-Regular")
	assertString(t, "FullName", font.FullName(), "Open Sans Regular")
	assertString(t, "FamilyName", font.FamilyName(), "Open Sans")
	assertString(t, "Notice", font.Notice(),
		"Digitized data copyright (c) 2010-2011, Google Corporation.")
	if font.IsFixedPitch() {
		t.Error("IsFixedPitch() = true, want false")
	}
	if font.IsForceBold() {
		t.Error("IsForceBold() = true, want false")
	}
	if got := font.ItalicAngle(); got != 0 {
		t.Errorf("ItalicAngle() = %v, want 0", got)
	}
	assertString(t, "Weight", font.Weight(), "Book")
	// Java asserts the encoding is a BuiltInEncoding. The port builds one with
	// encoding.NewBuiltInEncoding, which is an *Encoding like every other, so
	// what carries over is that it is not the shared StandardEncoding.
	if font.Encoding() == nil || font.Encoding() == encoding.StandardEncoding {
		t.Error("Encoding() is not a built-in encoding")
	}
	if got := len(font.ASCIISegment()); got != 4498 {
		t.Errorf("ASCIISegment length = %d, want 4498", got)
	}
	if got := len(font.BinarySegment()); got != 95911 {
		t.Errorf("BinarySegment length = %d, want 95911", got)
	}
	if got := len(font.CharStringsDict()); got != 938 {
		t.Errorf("CharStringsDict size = %d, want 938", got)
	}
	for _, s := range font.CharStringNames() {
		if _, err := font.GetPath(s); err != nil {
			t.Errorf("GetPath(%s): %v", s, err)
		}
		if has, _ := font.HasGlyph(s); !has {
			t.Errorf("HasGlyph(%s) = false, want true", s)
		}
	}
}

// TestPfbPDFBox5713 covers PDFBOX-5713: a font with several binary segments.
func TestPfbPDFBox5713(t *testing.T) {
	font, err := CreateWithPFBBytes(openFontFixture(t, "DejaVuSerifCondensed.pfb"))
	if err != nil {
		t.Fatalf("CreateWithPFB: %v", err)
	}
	assertString(t, "Version", font.Version(), "Version 2.33")
	assertString(t, "FontName", font.FontName(), "DejaVuSerifCondensed")
	name, _ := font.Name()
	assertString(t, "Name", name, "DejaVuSerifCondensed")
	assertString(t, "FullName", font.FullName(), "DejaVu Serif Condensed")
	assertString(t, "FamilyName", font.FamilyName(), "DejaVu Serif Condensed")
	assertString(t, "Notice", font.Notice(),
		"Copyright [c] 2003 by Bitstream, Inc. All Rights Reserved.")
	if got := len(font.SubrsArray()); got != 11974 {
		t.Errorf("SubrsArray size = %d, want 11974", got)
	}
	if got := font.PaintType(); got != 0 {
		t.Errorf("PaintType() = %d, want 0", got)
	}
	if got := font.FontType(); got != 1 {
		t.Errorf("FontType() = %d, want 1", got)
	}
	assertString(t, "FontID", font.FontID(), "")
	if got := font.UniqueID(); got != 0 {
		t.Errorf("UniqueID() = %d, want 0", got)
	}
	if got := font.StrokeWidth(); got != 0 {
		t.Errorf("StrokeWidth() = %v, want 0", got)
	}
	if got := font.UnderlinePosition(); got != -63 {
		t.Errorf("UnderlinePosition() = %v, want -63", got)
	}
	if got := font.UnderlineThickness(); got != 44 {
		t.Errorf("UnderlineThickness() = %v, want 44", got)
	}
	assertNumbers(t, "BlueValues", font.BlueValues(),
		"[-15, 0, 729, 744, 512, 533, 785, 800, 760, 777, 829, 847, 920, 935]")
	assertNumbers(t, "OtherBlues", font.OtherBlues(), "[-222, -203]")
	assertNumbers(t, "FamilyBlues", font.FamilyBlues(), "[]")
	assertNumbers(t, "FamilyOtherBlues", font.FamilyOtherBlues(), "[]")
	if got := font.BlueScale(); got != 0 {
		t.Errorf("BlueScale() = %v, want 0", got)
	}
	if got := font.BlueShift(); got != 0 {
		t.Errorf("BlueShift() = %d, want 0", got)
	}
	if got := font.BlueFuzz(); got != 0 {
		t.Errorf("BlueFuzz() = %d, want 0", got)
	}
	if got := font.LanguageGroup(); got != 0 {
		t.Errorf("LanguageGroup() = %d, want 0", got)
	}
	assertNumbers(t, "StdHW", font.StdHW(), "[53]")
	assertNumbers(t, "StdVW", font.StdVW(), "[90]")
	assertNumbers(t, "StemSnapH", font.StemSnapH(), "[53]")
	assertNumbers(t, "StemSnapV", font.StemSnapV(), "[54, 66, 80, 90, 101, 146]")
	assertNumbers(t, "FontMatrix", font.FontMatrixNumbers(), "[0.001, 0, 0, 0.001, 0, 0]")
	bbox, err := font.FontBBox()
	if err != nil {
		t.Fatalf("FontBBox: %v", err)
	}
	want := "[-692.0,-347.0,1511.0,1109.0]"
	got := "[" + javaFloatString(bbox.LowerLeftX()) + "," + javaFloatString(bbox.LowerLeftY()) +
		"," + javaFloatString(bbox.UpperRightX()) + "," + javaFloatString(bbox.UpperRightY()) + "]"
	if got != want {
		t.Errorf("FontBBox = %s, want %s", got, want)
	}
	if font.IsFixedPitch() {
		t.Error("IsFixedPitch() = true, want false")
	}
	if font.IsForceBold() {
		t.Error("IsForceBold() = true, want false")
	}
	if got := font.ItalicAngle(); got != 0 {
		t.Errorf("ItalicAngle() = %v, want 0", got)
	}
	assertString(t, "Weight", font.Weight(), "Book")
	if font.Encoding() == nil || font.Encoding() == encoding.StandardEncoding {
		t.Error("Encoding() is not a built-in encoding")
	}
	if got := len(font.ASCIISegment()); got != 5959 {
		t.Errorf("ASCIISegment length = %d, want 5959", got)
	}
	if got := len(font.BinarySegment()); got != 1056090 {
		t.Errorf("BinarySegment length = %d, want 1056090", got)
	}
	if got := len(font.CharStringsDict()); got != 3399 {
		t.Errorf("CharStringsDict size = %d, want 3399", got)
	}
}

// TestPfbPDFBox3654 covers PDFBOX-3654: a font with a hex encoded binary
// segment.
func TestPfbPDFBox3654(t *testing.T) {
	ba := openFontFixture(t, "KIX-Barcode-Regular.pfb")
	font, err := CreateWithSegments(ba[0:1039], ba[1039:1039+26868])
	if err != nil {
		t.Fatalf("CreateWithSegments: %v", err)
	}
	assertString(t, "Version", font.Version(), "001.000")
	assertString(t, "FontName", font.FontName(), "KIX-Barcode-Regular")
	assertString(t, "FullName", font.FullName(), "KIX-Barcode-Regular")
	assertString(t, "FamilyName", font.FamilyName(), "KIX-Barcode")
	assertString(t, "Notice", font.Notice(), "")
	if font.IsFixedPitch() {
		t.Error("IsFixedPitch() = true, want false")
	}
	if font.IsForceBold() {
		t.Error("IsForceBold() = true, want false")
	}
	if got := font.ItalicAngle(); got != 0 {
		t.Errorf("ItalicAngle() = %v, want 0", got)
	}
	assertString(t, "Weight", font.Weight(), "Regular")
	if font.Encoding() != encoding.StandardEncoding {
		t.Error("Encoding() is not the StandardEncoding")
	}
	if got := len(font.ASCIISegment()); got != 1039 {
		t.Errorf("ASCIISegment length = %d, want 1039", got)
	}
	if got := len(font.BinarySegment()); got != 26868 {
		t.Errorf("BinarySegment length = %d, want 26868", got)
	}
	if got := len(font.CharStringsDict()); got != 257 {
		t.Errorf("CharStringsDict size = %d, want 257", got)
	}
}

func assertString(t *testing.T, what, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", what, got, want)
	}
}

func assertNumbers(t *testing.T, what string, got []any, want string) {
	t.Helper()
	if rendered := numbersString(got); rendered != want {
		t.Errorf("%s = %s, want %s", what, rendered, want)
	}
}
