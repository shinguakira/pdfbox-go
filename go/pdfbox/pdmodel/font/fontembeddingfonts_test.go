package font_test

// The six cases of org.apache.pdfbox.pdmodel.font.TestFontEmbedding that read a
// font the Maven build downloads into `target/fonts`.
//
// They were deferred while that directory was empty. It is not:
// `migration/scripts/fetch-testdata.ps1` fills it, and all three fonts are
// there — `ipag00303/ipag.ttf`, `ipagp00303/ipagp.ttf` and
// `NotoSansCJKkr-VF.ttf`. Every expected value below is the Java's, copied.
//
// The cases skip rather than fail where the fetch has not been run, because a
// missing download is not a defect in the port.

import (
	"bytes"
	"os"
	"testing"

	"unicode"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// The three fonts, at the paths Java names.
const (
	ipagFont  = "../../../../pdfbox/target/fonts/ipag00303/ipag.ttf"
	ipagpFont = "../../../../pdfbox/target/fonts/ipagp00303/ipagp.ttf"
	notoCJK   = "../../../../pdfbox/target/fonts/NotoSansCJKkr-VF.ttf"
)

// requireFont skips the case where the fetch has not been run.
func requireFont(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s is not there; run migration/scripts/fetch-testdata.ps1", path)
	}
}

// verticalSubset is the body testCIDFontType2VerticalSubsetMonospace and
// testCIDFontType2VerticalSubsetProportional share.
//
// `「ABC」` written vertically comes back out one character per line, because
// that is what a vertical writing mode does to the extractor.
func verticalSubset(t *testing.T, fontPath string, wantCID int,
	wantW2 []int, wantStartCID int) {
	t.Helper()
	requireFont(t, fontPath)

	const message = "「ABC」"
	const wantExtracted = "「\nA\nB\nC\n」"

	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	vertical, err := font.LoadPDType0FontVerticalFile(document, fontPath)
	if err != nil {
		t.Fatalf("LoadPDType0FontVerticalFile: %v", err)
	}
	writeText(t, document, page, vertical, 20, 50, 700, message)

	// The font substitution: the vertical form of 「 is a different glyph.
	encoded, err := vertical.Encode(message)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if len(encoded) < 2 {
		t.Fatalf("Encode gave %d bytes", len(encoded))
	}
	if cid := int(encoded[0])<<8 + int(encoded[1]); cid != wantCID {
		t.Errorf("the first CID is %d, want %d", cid, wantCID)
	}

	fontDictionary := vertical.COSObject().(*cos.Dictionary)
	if got := fontDictionary.GetDictionaryObject(cos.Encoding); got != cos.IdentityV {
		t.Errorf("/Encoding is %v, want /Identity-V", got)
	}

	var saved bytes.Buffer
	if err := document.Save(&saved); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Vertical metrics are fixed during subsetting, so this comes after saving.
	descendant := vertical.DescendantFont().COSObject().(*cos.Dictionary)
	if dw2 := descendant.GetDictionaryObject(cos.DW2); dw2 != nil {
		t.Errorf("/DW2 is %v, want it absent: this font uses the default", dw2)
	}
	w2 := descendant.GetCOSArray(cos.W2)
	if w2 == nil {
		t.Fatal("/W2 is absent")
	}
	if got := w2.Size(); got != len(wantW2Array(wantW2, wantStartCID)) {
		t.Errorf("/W2 holds %d entries, want %d",
			got, len(wantW2Array(wantW2, wantStartCID)))
	}
	if wantW2 != nil {
		if got := w2.GetInt(0); got != wantStartCID {
			t.Errorf("/W2 starts at CID %d, want %d", got, wantStartCID)
		}
		metrics, isArray := w2.GetObject(1).(*cos.Array)
		if !isArray {
			t.Fatalf("/W2[1] is %T, want an array", w2.GetObject(1))
		}
		for i, want := range wantW2 {
			if got := metrics.GetInt(i); got != want {
				t.Errorf("/W2 metric %d is %d, want %d", i, got, want)
			}
		}
	}

	if got := trimmed(unicodeText(t, saved.Bytes())); got != wantExtracted {
		t.Errorf("the text extracts as %q, want %q", got, wantExtracted)
	}
}

// wantW2Array answers how many entries /W2 should hold: none for a monospaced
// font, the start CID and the metrics array for a proportional one.
func wantW2Array(metrics []int, startCID int) []int {
	if metrics == nil {
		return nil
	}
	return []int{startCID, 0}
}

// trimmed is Java's extracted.replaceAll("\r", "").trim().
func trimmed(extracted string) string {
	out := make([]rune, 0, len(extracted))
	for _, r := range extracted {
		if r != '\r' {
			out = append(out, r)
		}
	}
	text := string(out)
	for len(text) > 0 && (text[0] == '\n' || text[0] == ' ' || text[0] == '\t') {
		text = text[1:]
	}
	for len(text) > 0 {
		last := text[len(text)-1]
		if last != '\n' && last != ' ' && last != '\t' {
			break
		}
		text = text[:len(text)-1]
	}
	return text
}

// TestCIDFontType2VerticalSubsetMonospace is the monospaced half: IPA Gothic,
// whose /W2 has no entries because every glyph takes the default.
func TestCIDFontType2VerticalSubsetMonospace(t *testing.T) {
	// 7392 is the vertical form; it is 441 without the substitution.
	verticalSubset(t, ipagFont, 7392, nil, 0)
}

// TestCIDFontType2VerticalSubsetProportional is the proportional half: IPA
// P Gothic, whose /W2 carries the two glyphs that are not the default.
func TestCIDFontType2VerticalSubsetProportional(t *testing.T) {
	// 12607 is the vertical form; it is 12461 without the substitution.
	verticalSubset(t, ipagpFont, 12607,
		[]int{-570, 500, 450, -570, 500, 880}, 12607)
}

// TestMaxEntries is testMaxEntries, the corner case of PDFBOX-4302: a document
// whose ToUnicode CMap holds exactly MAX_ENTRIES_PER_OPERATOR entries, which is
// where the writer has to start a second operator.
func TestMaxEntries(t *testing.T) {
	requireFont(t, ipagFont)

	const message = "あいうえおかきくけこさしすせそたちつてとなにぬねのはひふへほまみむめもやゆよらりるれろわをん" +
		"アイウエオカキクケコサシスセソタチツテトナニヌネノハヒフヘホマミムメモヤユヨラリルレロワヲン" +
		"１２３４５６７８"

	// The case only works if the text has exactly that many distinct
	// characters, which Java asserts before writing anything.
	distinct := map[rune]bool{}
	for _, r := range message {
		distinct[r] = true
	}
	// ToUnicodeWriter.MAX_ENTRIES_PER_OPERATOR, which the port keeps unexported
	// as maxEntriesPerOperator in tounicodewriter.go.
	const maxEntriesPerOperator = 100
	if got := len(distinct); got != maxEntriesPerOperator {
		t.Fatalf("the message has %d distinct characters and the writer's "+
			"limit is %d; the case needs them equal",
			got, maxEntriesPerOperator)
	}

	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.A0)
	document.AddPage(page)

	embedded, err := font.LoadPDType0FontFile(document, ipagFont)
	if err != nil {
		t.Fatalf("LoadPDType0FontFile: %v", err)
	}
	writeText(t, document, page, embedded, 20, 50, 3000, message)

	var saved bytes.Buffer
	if err := document.Save(&saved); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := trimmed(unicodeText(t, saved.Bytes())); got != message {
		t.Errorf("the text does not survive the round trip at the operator "+
			"boundary:\n got %q\nwant %q", got, message)
	}
}

// renderAndExtract is the Java helper of the same name: write one code point
// with the Noto CJK font, save, read back, extract.
func renderAndExtract(t *testing.T, codePoint rune) string {
	t.Helper()
	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	embedded, err := font.LoadPDType0FontFile(document, notoCJK)
	if err != nil {
		t.Fatalf("LoadPDType0FontFile: %v", err)
	}
	writeText(t, document, page, embedded, 20, 50, 700, string(codePoint))

	var saved bytes.Buffer
	if err := document.Save(&saved); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return trimmed(unicodeText(t, saved.Bytes()))
}

// notoCJKLookup parses the font and answers its Unicode cmap.
func notoCJKLookup(t *testing.T) (ttf.CmapLookup, *ttf.TrueTypeFont) {
	t.Helper()
	source, err := pdfio.OpenBufferedFile(notoCJK)
	if err != nil {
		t.Fatalf("opening %s: %v", notoCJK, err)
	}
	parsed, err := ttf.NewParser().Parse(source)
	if err != nil {
		t.Fatalf("parsing %s: %v", notoCJK, err)
	}
	lookup, err := parsed.UnicodeCmapLookupStrict()
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	return lookup, parsed
}

// TestToUnicodePrefersUsedCodePoint is testToUnicodePrefersUsedCodePoint: when
// several code points reach one glyph, the /ToUnicode CMap must carry the one
// that was actually written, not the lowest that shares the glyph.
//
// The pair is found in the font rather than hard-coded, which is what the Java
// does; the assertion is that each of the two round-trips as itself. Before the
// fix the higher one came back as the lower.
func TestToUnicodePrefersUsedCodePoint(t *testing.T) {
	requireFont(t, notoCJK)

	lookup, parsed := notoCJKLookup(t)
	defer parsed.Close()
	profile, err := parsed.MaximumProfile()
	if err != nil {
		t.Fatalf("MaximumProfile: %v", err)
	}

	lowCp, highCp := -1, -1
	cjkPair := false
	for gid := 1; gid <= profile.NumGlyphs() && !cjkPair; gid++ {
		codes := lookup.GetCharCodes(gid) // sorted ascending
		if len(codes) < 2 {
			continue
		}
		lo, hi := -1, -1
		for _, cp := range codes {
			if cp <= 0xFFFF && !isJavaWhitespace(rune(cp)) && !isISOControl(rune(cp)) {
				if lo == -1 {
					lo = cp
				} else {
					hi = cp
					break
				}
			}
		}
		// 0x2E80 is where the CJK Radicals Supplement starts: prefer such a
		// pair, because it is the demonstration the issue was about.
		if hi != -1 && (lowCp == -1 || lo >= 0x2E80) {
			lowCp, highCp = lo, hi
			cjkPair = lo >= 0x2E80
		}
	}
	if highCp == -1 {
		t.Fatal("test font has no glyph shared between two printable code points")
	}

	if got, want := renderAndExtract(t, rune(highCp)), string(rune(highCp)); got != want {
		t.Errorf("U+%04X extracts as %q, want %q: the higher code point of a "+
			"shared glyph is coming back as the lower", highCp, got, want)
	}
	if got, want := renderAndExtract(t, rune(lowCp)), string(rune(lowCp)); got != want {
		t.Errorf("U+%04X extracts as %q, want %q", lowCp, got, want)
	}
}

// TestToUnicodeCjkAndRadicalLookAlike is testToUnicodeCjkAndRadicalLookAlike,
// the same thing with the pair named: 食 and the radical that looks like it and
// shares its glyph.
func TestToUnicodeCjkAndRadicalLookAlike(t *testing.T) {
	requireFont(t, notoCJK)

	const ideograph = rune(0x98DF) // 食 CJK Unified Ideograph
	const radical = rune(0x2EDD)   // ⻝ CJK RADICAL EAT ONE, shares the glyph

	lookup, parsed := notoCJKLookup(t)
	defer parsed.Close()

	// The precondition: both reach the same glyph, and the radical is the
	// lower, which is the entry the old reverse mapping picked for both.
	gid := lookup.GetGlyphID(int(ideograph))
	if gid <= 0 || gid != lookup.GetGlyphID(int(radical)) {
		t.Fatalf("the font maps 食 to glyph %d and ⻝ to %d; the case needs "+
			"them equal and non-zero", gid, lookup.GetGlyphID(int(radical)))
	}
	if codes := lookup.GetCharCodes(gid); len(codes) == 0 || codes[0] != int(radical) {
		t.Fatalf("the glyph's first char code is %v, want U+2EDD", codes)
	}

	if got, want := renderAndExtract(t, ideograph), string(ideograph); got != want {
		t.Errorf("食 extracts as %q, want %q: the ideograph is coming back as "+
			"its radical look-alike", got, want)
	}
	if got, want := renderAndExtract(t, radical), string(radical); got != want {
		t.Errorf("⻝ extracts as %q, want %q: a radical typed on purpose must "+
			"be kept", got, want)
	}
}

// isJavaWhitespace and isISOControl are Character.isWhitespace and
// Character.isISOControl, over the range this case looks at.
func isJavaWhitespace(r rune) bool {
	switch r {
	case ' ', '\t', '\n', '\v', '\f', '\r', 0x1C, 0x1D, 0x1E, 0x1F:
		return true
	}
	return unicode.IsSpace(r) && r != 0x00A0 && r != 0x2007 && r != 0x202F
}

func isISOControl(r rune) bool {
	return r <= 0x1F || (r >= 0x7F && r <= 0x9F)
}

// TestSurrogatePairCharacter is testSurrogatePairCharacter, PDFBOX-5812: Atka
// mackerel in Japanese kanji, which is a surrogate pair, written twice — once
// as the code point and once as the two UTF-16 units Java writes it with.
//
// Java also renders the file and compares it, but does not fail on a
// difference: its own comment says rendering differs between systems and the
// result has to be looked at. There is nothing to assert there, so the port
// asserts what Java asserts, which is the round trip through the extractor.
func TestSurrogatePairCharacter(t *testing.T) {
	requireFont(t, ipagFont)

	// U+29E3D twice. Java writes the second as the surrogate pair \uD867\uDE3D,
	// which is the same character; Go has no way to write half of one, so both
	// halves of the message are the code point.
	const message = "\U00029E3D\U00029E3D"

	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPage()
	document.AddPage(page)

	embedded, err := font.LoadPDType0FontFile(document, ipagFont)
	if err != nil {
		t.Fatalf("LoadPDType0FontFile: %v", err)
	}
	writeText(t, document, page, embedded, 64, 100, 700, message)

	var saved bytes.Buffer
	if err := document.Save(&saved); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if got := trimmed(unicodeText(t, saved.Bytes())); got != message {
		t.Errorf("the text extracts as %q, want %q: a character outside the "+
			"basic plane is not surviving the round trip", got, message)
	}
}
