package cff

import (
	"bytes"
	"os"
	"sync"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.fontbox.cff.CFFParserTest.
//
// The font is one the Java build downloads into target/fonts, so every test
// here skips when it is not present.

const sourceSansPath = "../../../fontbox/target/fonts/SourceSansProBold.otf"

// loadCFFFont is the @BeforeAll of the Java class, run per test here so that a
// missing font skips rather than failing the package.
func loadCFFFont(t *testing.T) *CFFType1Font {
	t.Helper()
	fonts := readFont(t, sourceSansPath)
	font, ok := fonts[0].(*CFFType1Font)
	if !ok {
		t.Fatalf("the first font is %T, want a CFFType1Font", fonts[0])
	}
	return font
}

func readFont(t *testing.T, filename string) []CFFFont {
	t.Helper()
	if _, err := os.Stat(filename); err != nil {
		t.Skipf("the font the Java build downloads is not present: %v", err)
	}
	randomAccessRead, err := pdfio.OpenBufferedFile(filename)
	if err != nil {
		t.Fatalf("opening %s: %v", filename, err)
	}
	fonts, err := NewCFFParser().Parse(randomAccessRead)
	if err != nil {
		t.Fatalf("parsing %s: %v", filename, err)
	}
	return fonts
}

func TestFontname(t *testing.T) {
	font := loadCFFFont(t)
	name, err := font.Name()
	if err != nil {
		t.Fatalf("Name: %v", err)
	}
	if name != "SourceSansPro-Bold" {
		t.Errorf("Name() = %q, want %q", name, "SourceSansPro-Bold")
	}
}

func TestFontBBox(t *testing.T) {
	font := loadCFFFont(t)
	fontBBox, err := font.FontBBox()
	if err != nil {
		t.Fatalf("FontBBox: %v", err)
	}
	if fontBBox == nil {
		t.Fatal("FontBBox must not be null")
	}
	if got := fontBBox.LowerLeftX(); got != -231.0 {
		t.Errorf("LowerLeftX = %v, want -231", got)
	}
	if got := fontBBox.LowerLeftY(); got != -384.0 {
		t.Errorf("LowerLeftY = %v, want -384", got)
	}
	if got := fontBBox.UpperRightX(); got != 1223.0 {
		t.Errorf("UpperRightX = %v, want 1223", got)
	}
	if got := fontBBox.UpperRightY(); got != 974.0 {
		t.Errorf("UpperRightY = %v, want 974", got)
	}
}

func TestFontMatrix(t *testing.T) {
	font := loadCFFFont(t)
	fontMatrix, err := font.FontMatrix()
	if err != nil {
		t.Fatalf("FontMatrix: %v", err)
	}
	if fontMatrix == nil {
		t.Fatal("FontMatrix must not be null")
	}
	want := []float32{0.001, 0.0, 0.0, 0.001, 0.0, 0.0}
	if len(fontMatrix) != len(want) {
		t.Fatalf("FontMatrix has %d values, want %d", len(fontMatrix), len(want))
	}
	for i, w := range want {
		if fontMatrix[i] != w {
			t.Errorf("FontMatrix[%d] = %v, want %v", i, fontMatrix[i], w)
		}
	}
}

func TestParsedCharset(t *testing.T) {
	font := loadCFFFont(t)
	charset := font.Charset()
	if charset == nil {
		t.Fatal("Charset must not be null")
	}
	if charset.IsCIDFont() {
		t.Error("isCIDFont has to be false")
	}
	if _, ok := charset.(*format1Charset); !ok {
		t.Errorf("Charset is %T, not the port of Format1Charset", charset)
	}
	// check some randomly chosen mappings
	cases := []struct {
		gid  int
		sid  int
		name string
	}{
		{0, 0, ".notdef"},
		{1, 1, "space"},
		{7, 39, "F"},
		{300, 585, "jcircumflex"},
		{700, 872, "infinity"},
	}
	for _, c := range cases {
		// gid2name
		if got, _ := charset.NameForGID(c.gid); got != c.name {
			t.Errorf("Unexpected value for gid2name mapping: NameForGID(%d) = %q, want %q",
				c.gid, got, c.name)
		}
		// gid2sid
		if got := charset.SIDForGID(c.gid); got != c.sid {
			t.Errorf("Unexpected value for gid2sid mapping: SIDForGID(%d) = %d, want %d",
				c.gid, got, c.sid)
		}
		// name2sid
		if got := charset.SID(c.name); got != c.sid {
			t.Errorf("Unexpected value for name2sid mapping: SID(%s) = %d, want %d",
				c.name, got, c.sid)
		}
	}
}

func TestParsedEncoding(t *testing.T) {
	font := loadCFFFont(t)
	encoding := font.CFFEncoding()
	if encoding == nil {
		t.Fatal("Encoding must not be null")
	}
	if encoding != CFFStandardEncoding {
		t.Error("Encoding is not the CFFStandardEncoding")
	}
}

func TestCharStringBytes(t *testing.T) {
	font := loadCFFFont(t)
	charStringBytes := font.CharStringBytes()
	if len(charStringBytes) == 0 {
		t.Fatal("CharStringBytes is empty")
	}
	if got := font.NumCharStrings(); got != 824 {
		t.Errorf("NumCharStrings = %d, want 824", got)
	}
	// check some randomly chosen values
	cases := []struct {
		index int
		want  []byte
	}{
		{1, []byte{0xFC, 15, 14}},
		{16, []byte{72, 29, 0xF3, 29, 0xF7, 0xB6, 0xF7, 43, 3, 33, 29, 14}},
		{195, []byte{0xD7, 88, 29, 0xD1, 0xF7, 12, 1, 0x85, 10, 3, 35, 29, 0xF7,
			0xCE, 0xF7, 62, 0xF7, 3, 10, 85, 0xC8, 61, 10}},
		{525, []byte{0xFB, 0xBB, 0xC3, 0xF8, 28, 1, 0xF7, 57, 0xD9, 0xBF, 29, 14}},
		{738, []byte{107, 0xD0, 10, 0xF7, 20, 0xF7, 123, 3, 0xF7, 0x90, 0xF8, 0xD2, 21,
			0xF6, 115, 10}},
	}
	for _, c := range cases {
		if !bytes.Equal(charStringBytes[c.index], c.want) {
			t.Errorf("Other char strings byte values than expected at %d: %v, want %v",
				c.index, charStringBytes[c.index], c.want)
		}
	}
}

func TestGlobalSubrIndex(t *testing.T) {
	font := loadCFFFont(t)
	globalSubrIndex := font.GlobalSubrIndex()
	if len(globalSubrIndex) == 0 {
		t.Fatal("GlobalSubrIndex is empty")
	}
	if len(globalSubrIndex) != 278 {
		t.Errorf("GlobalSubrIndex has %d entries, want 278", len(globalSubrIndex))
	}
	// check some randomly chosen values
	cases := []struct {
		index int
		want  []byte
	}{
		{12, []byte{21, 0xBA, 0xAD, 0xAB, 0xB8, 0xB8, 105, 0xAB, 92, 91, 105, 107,
			10, 0xAD, 0xF7, 62, 10}},
		{120, []byte{58, 122, 29, 0xFB, 48, 6, 11}},
		{253, []byte{68, 80, 29, 0xD3, 0xF7, 16, 0xF8, 0xA4, 119, 11}},
	}
	for _, c := range cases {
		if !bytes.Equal(globalSubrIndex[c.index], c.want) {
			t.Errorf("Other global subr index values than expected at %d: %v, want %v",
				c.index, globalSubrIndex[c.index], c.want)
		}
	}
}

// TestDeltaLists covers PDFBOX-4038: whether BlueValues and other delta encoded
// lists are read correctly. The test file is from FOP-2432.
func TestDeltaLists(t *testing.T) {
	font := loadCFFFont(t)
	cases := []struct {
		key  string
		want []int
	}{
		// Expected values found for this font
		{"BlueValues", []int{-12, 0, 496, 508, 578, 590, 635, 647, 652, 664, 701, 713}},
		{"OtherBlues", []int{-196, -184}},
		{"FamilyBlues", []int{-12, 0, 486, 498, 574, 586, 638, 650, 656, 668, 712, 724}},
		{"FamilyOtherBlues", []int{-217, -205}},
		{"StemSnapH", []int{115}},
		{"StemSnapV", []int{146, 150}},
	}
	for _, c := range cases {
		found, ok := font.PrivateDict()[c.key].([]any)
		if !ok {
			t.Errorf("%s is missing from the private dict", c.key)
			continue
		}
		if len(found) != len(c.want) {
			t.Errorf("%s has %d values, want %d", c.key, len(found), len(c.want))
			continue
		}
		for i, w := range c.want {
			if got := numberInt(found[i]); got != w {
				t.Errorf("%s[%d] = %d, want %d", c.key, i, got, w)
			}
		}
	}
}

// TestMultiThreadParse covers PDFBOX-5819: the charstring parse must survive
// being run from two goroutines at once.
func TestMultiThreadParse(t *testing.T) {
	font := loadCFFFont(t)
	var wg sync.WaitGroup
	for n := 0; n < 2; n++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 33; i < 126; i++ {
				if _, err := font.GetPath(string(rune(i))); err != nil {
					t.Errorf("GetPath(%q): %v", rune(i), err)
					return
				}
			}
		}()
	}
	wg.Wait()
}
