package cmap

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Port of org.apache.fontbox.cmap.TestCMapParser.

const cmapFixture = "../../../fontbox/src/test/resources/cmap/"

// parseFixture parses one of the CMaps under the Java test resources.
func parseFixture(t *testing.T, parser *Parser, name string) *CMap {
	t.Helper()
	source, err := pdfio.OpenBufferedFile(cmapFixture + name)
	if err != nil {
		t.Fatalf("opening %s: %v", name, err)
	}
	cMap, err := parser.Parse(source)
	if err != nil {
		t.Fatalf("parsing %s: %v", name, err)
	}
	return cMap
}

// TestParserLookup checks whether the parser and the resulting mapping is
// working correct.
func TestParserLookup(t *testing.T) {
	cMap := parseFixture(t, NewParser(), "CMapTest")

	// char mappings
	charCases := []struct {
		bytes []byte
		want  string
		why   string
	}{
		{[]byte{0, 1}, "A", "bytes 00 01 from bfrange <0001> <0005> <0041>"},
		{[]byte{1, 0}, "0", "bytes 01 00 from bfrange <0100> <0109> <0030>"},
		{[]byte{1, 32}, "P", "bytes 01 00 from bfrange <0100> <0109> <0030>"},
		{[]byte{1, 33}, "R", "bytes 01 00 from bfrange <0100> <0109> <0030>"},
		{[]byte{0, 10}, "*", "bytes 00 0A from bfchar <000A> <002A>"},
		{[]byte{1, 10}, "+", "bytes 01 0A from bfchar <010A> <002B>"},
	}
	for _, c := range charCases {
		if got, _ := cMap.ToUnicodeBytes(c.bytes); got != c.want {
			t.Errorf("ToUnicode(%v) = %q, want %q -- %s", c.bytes, got, c.want, c.why)
		}
	}

	// CID mappings
	cidCases := []struct {
		bytes []byte
		want  int
		why   string
	}{
		{[]byte{0, 65}, 65, "CID 65 from cidrange <0000> <00ff> 0 "},
		{[]byte{1, 24}, 0x0118, "CID 280 from cidrange <0100> <01ff> 256"},
		{[]byte{2, 8}, 0x0208, "CID 520 from cidchar <0208> 520"},
		{[]byte{1, 0x2c}, 0x12C, "CID 300 from cidrange <0300> <0300> 300"},
	}
	for _, c := range cidCases {
		if got := cMap.ToCIDBytes(c.bytes); got != c.want {
			t.Errorf("ToCID(%v) = %d, want %d -- %s", c.bytes, got, c.want, c.why)
		}
	}
}

func TestIdentity(t *testing.T) {
	cMap, err := NewParser().ParsePredefined("Identity-H")
	if err != nil {
		t.Fatalf("ParsePredefined(Identity-H): %v", err)
	}
	cases := []struct {
		bytes []byte
		want  int
		why   string
	}{
		{[]byte{0, 65}, 65, "Indentity-H CID 65"},
		{[]byte{0x30, 0x39}, 12345, "Indentity-H CID 12345"},
		{[]byte{0xFF, 0xFF}, 0xFFFF, "Indentity-H CID 0xFFFF"},
	}
	for _, c := range cases {
		if got := cMap.ToCIDBytes(c.bytes); got != c.want {
			t.Errorf("ToCID(%v) = %d, want %d -- %s", c.bytes, got, c.want, c.why)
		}
	}
}

func TestUniJISUTF16H(t *testing.T) {
	cMap, err := NewParser().ParsePredefined("UniJIS-UTF16-H")
	if err != nil {
		t.Fatalf("ParsePredefined(UniJIS-UTF16-H): %v", err)
	}

	// the next 3 cases demonstrate the issue of possible false result values
	// of CMap.toCID(int code)
	if got := cMap.ToCID(0xb1); got != 694 {
		t.Errorf("ToCID(0xb1) = %d, want 694 -- UniJIS-UTF16-H CID 0xb1 -> 694", got)
	}
	if got := cMap.ToCIDLength(0xb1, 1); got == 694 {
		t.Error("ToCID(0xb1, 1) = 694, want anything else -- UniJIS-UTF16-H CID 0xb1 -> 694")
	}
	if got := cMap.ToCIDLength(0xb1, 2); got != 694 {
		t.Errorf("ToCID(0xb1, 2) = %d, want 694 -- UniJIS-UTF16-H CID 0x00b1 -> 694", got)
	}

	cases := []struct {
		bytes []byte
		want  int
		why   string
	}{
		// 1:1 cid char mapping
		{[]byte{0x00, 0xb1}, 694, "UniJIS-UTF16-H CID 0x00b1 -> 694"},
		{[]byte{0xd8, 0x50, 0xdc, 0x4b}, 20168, "UniJIS-UTF16-H CID 0xd850dc4b -> 20168"},
		// cid range mapping
		{[]byte{0x54, 0x34}, 19223, "UniJIS-UTF16-H CID 0x5434 -> 19223"},
		{[]byte{0xd8, 0x3c, 0xdd, 0x12}, 10006, "UniJIS-UTF16-H CID 0xd83cdd12 -> 10006"},
	}
	for _, c := range cases {
		if got := cMap.ToCIDBytes(c.bytes); got != c.want {
			t.Errorf("ToCID(%v) = %d, want %d -- %s", c.bytes, got, c.want, c.why)
		}
	}
}

func TestUniJISUCS2H(t *testing.T) {
	cMap, err := NewParser().ParsePredefined("UniJIS-UCS2-H")
	if err != nil {
		t.Fatalf("ParsePredefined(UniJIS-UCS2-H): %v", err)
	}
	if got := cMap.ToCIDBytes([]byte{0, 65}); got != 34 {
		t.Errorf("ToCID([0 65]) = %d, want 34 -- UniJIS-UCS2-H CID 65 -> 34", got)
	}
}

func TestAdobeGB1UCS2(t *testing.T) {
	cMap, err := NewParser().ParsePredefined("Adobe-GB1-UCS2")
	if err != nil {
		t.Fatalf("ParsePredefined(Adobe-GB1-UCS2): %v", err)
	}
	if got, _ := cMap.ToUnicodeBytes([]byte{0, 0x11}); got != "0" {
		t.Errorf("ToUnicode([0 0x11]) = %q, want %q -- Adobe-GB1-UCS2 CID 0x11 maps to zero",
			got, "0")
	}
}

// TestParserWithPoorWhitespace tests the parser against a valid, but poorly
// formatted CMap file.
func TestParserWithPoorWhitespace(t *testing.T) {
	if cMap := parseFixture(t, NewParser(), "CMapNoWhitespace"); cMap == nil {
		t.Error("Failed to parse nasty CMap file")
	}
}

func TestParserWithMalformedbfrange1(t *testing.T) {
	cMap := parseFixture(t, NewParser(), "CMapMalformedbfrange1")
	if cMap == nil {
		t.Fatal("Failed to parse malformed CMap file")
	}

	if got, _ := cMap.ToUnicodeBytes([]byte{0, 1}); got != "A" {
		t.Errorf("ToUnicode([0 1]) = %q, want %q -- bytes 00 01 from bfrange <0001> <0009> <0041>",
			got, "A")
	}
	if got, ok := cMap.ToUnicodeBytes([]byte{1, 0}); ok {
		t.Errorf("ToUnicode([1 0]) = %q, want no mapping", got)
	}
}

func TestParserWithMalformedbfrange2(t *testing.T) {
	cMap := parseFixture(t, NewParser(), "CMapMalformedbfrange2")
	if cMap == nil {
		t.Fatal("Failed to parse malformed CMap file")
	}

	if got, _ := cMap.ToUnicodeBytes([]byte{0, 1}); got != "0" {
		t.Errorf("ToUnicode([0 1]) = %q, want %q -- bytes 00 01 from bfrange <0001> <0009> <0030>",
			got, "0")
	}
	if got, _ := cMap.ToUnicodeBytes([]byte{2, 0x32}); got != "A" {
		t.Errorf("ToUnicode([2 0x32]) = %q, want %q -- bytes 02 32 from bfrange <0232> <0432> <0041>",
			got, "A")
	}

	// check border values for non strict mode
	if _, ok := cMap.ToUnicodeBytes([]byte{2, 0xF0}); !ok {
		t.Error("ToUnicode([2 0xF0]) has no mapping, want one")
	}
	if _, ok := cMap.ToUnicodeBytes([]byte{2, 0xF1}); !ok {
		t.Error("ToUnicode([2 0xF1]) has no mapping, want one")
	}

	// use strict mode
	cMap = parseFixture(t, NewParserStrict(true), "CMapMalformedbfrange2")
	// check border values for strict mode
	if _, ok := cMap.ToUnicodeBytes([]byte{2, 0xF0}); !ok {
		t.Error("in strict mode ToUnicode([2 0xF0]) has no mapping, want one")
	}
	if got, ok := cMap.ToUnicodeBytes([]byte{2, 0xF1}); ok {
		t.Errorf("in strict mode ToUnicode([2 0xF1]) = %q, want no mapping", got)
	}
}

func TestPredefinedMap(t *testing.T) {
	cMap, err := NewParser().ParsePredefined("Adobe-Korea1-UCS2")
	if err != nil {
		t.Fatalf("Failed to parse predefined CMap Adobe-Korea1-UCS2: %v", err)
	}

	if got := cMap.Name(); got != "Adobe-Korea1-UCS2" {
		t.Errorf("wrong CMap name: %q", got)
	}
	if got := cMap.WMode(); got != 0 {
		t.Errorf("wrong WMode: %d", got)
	}
	if cMap.HasCIDMappings() {
		t.Error("HasCIDMappings() = true, want false")
	}
	if !cMap.HasUnicodeMappings() {
		t.Error("HasUnicodeMappings() = false, want true")
	}

	if _, err := NewParser().ParsePredefined("Identity-V"); err != nil {
		t.Errorf("Failed to parse predefined CMap Identity-V: %v", err)
	}
}

func TestIdentitybfrange(t *testing.T) {
	// use strict mode
	cMap := parseFixture(t, NewParserStrict(true), "Identitybfrange")
	if got := cMap.Name(); got != "Adobe-Identity-UCS" {
		t.Errorf("wrong CMap name: %q", got)
	}

	cases := []struct {
		bytes []byte
		why   string
	}{
		{[]byte{0, 65}, "Indentity 0x0048"},
		{[]byte{0x30, 0x39}, "Indentity 0x3039"},
		// check border values for strict mode
		{[]byte{0x30, 0xFF}, "Indentity 0x30FF"},
		{[]byte{0x31, 0x00}, "Indentity 0x3100"},
		{[]byte{0xFF, 0xFF}, "Indentity 0xFFFF"},
	}
	for _, c := range cases {
		want := decodeUTF16BE(c.bytes)
		if got, _ := cMap.ToUnicodeBytes(c.bytes); got != want {
			t.Errorf("ToUnicode(%v) = %q, want %q -- %s", c.bytes, got, want, c.why)
		}
	}
}

// TestBadIncrement checks that parsing a CMap with empty byte arrays in bfrange
// does not fail. Empty hex strings produce zero-length byte arrays, causing
// increment to be called with position -1.
func TestBadIncrement(t *testing.T) {
	cmapData := []byte("1 beginbfrange\n<> <> <2223>\nendbfrange")
	cmap, err := NewParser().Parse(pdfio.NewReadBufferBytes(cmapData))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cmap == nil {
		t.Error("Parse returned no CMap")
	}
}
