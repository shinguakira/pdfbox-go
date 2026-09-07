package font_test

// Port of org.apache.pdfbox.pdmodel.font.TestFontEmbedding.
//
// Eleven of its seventeen cases are here. The other six read a font from
// `target/fonts`, which the Maven build fills by downloading it, and the port
// fetches nothing in a test:
//
//	testCIDFontType2VerticalSubsetMonospace     ipag.ttf
//	testCIDFontType2VerticalSubsetProportional  ipagp.ttf
//	testMaxEntries                              ipag.ttf
//	testSurrogatePairCharacter                  ipag.ttf
//	testToUnicodePrefersUsedCodePoint           NotoSansCJKkr-VF.ttf
//	testToUnicodeCjkAndRadicalLookAlike         NotoSansCJKkr-VF.ttf
//
// Every case that is here writes a document, saves it, reads it back and
// extracts the text — which is the only way to check an embedded font end to
// end, and is what the Java does.

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// Where the Java test's fonts are. Java reaches them through the classpath;
// the port reads them by path.
const (
	// liberationSans is PDFont.class.getResourceAsStream(
	// "/org/apache/pdfbox/resources/ttf/LiberationSans-Regular.ttf")
	liberationSans = "../../../../pdfbox/src/main/resources/org/apache/pdfbox/resources/ttf/" +
		"LiberationSans-Regular.ttf"

	// indicFonts is getResourceAsStream("/org/apache/pdfbox/ttf/...")
	indicFonts = "../../../../pdfbox/src/test/resources/org/apache/pdfbox/ttf/"
)

// openFont is the getResourceAsStream every case opens with.
func openFont(t *testing.T, path string) *os.File {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	t.Cleanup(func() { file.Close() })
	return file
}

// writeText is the body every case shares: one page, one font, one run of text.
func writeText(t *testing.T, document *pdmodel.PDDocument, page *pdmodel.PDPage,
	embedded font.PDFont, size float32, x, y float32, message string) {
	t.Helper()
	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	if err := stream.BeginText(); err != nil {
		t.Fatalf("BeginText: %v", err)
	}
	if err := stream.SetFont(embedded, size); err != nil {
		t.Fatalf("SetFont: %v", err)
	}
	if err := stream.NewLineAtOffset(x, y); err != nil {
		t.Fatalf("NewLineAtOffset: %v", err)
	}
	if err := stream.ShowText(message); err != nil {
		t.Fatalf("ShowText: %v", err)
	}
	if err := stream.EndText(); err != nil {
		t.Fatalf("EndText: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

// unicodeText is the Java helper getUnicodeText: read the document back and
// extract its text.
func unicodeText(t *testing.T, data []byte) string {
	t.Helper()
	document, err := pdfbox.LoadPDFBytes(data)
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	defer document.Close()

	stripper := text.NewPDFTextStripper()
	extracted, err := stripper.GetTextOfPages(document.Pages())
	if err != nil {
		t.Fatalf("GetTextOfPages: %v", err)
	}
	return extracted
}

// TestCIDFontType2 and TestCIDFontType2Subset are the two cases that call
// validateCIDFontType2, with and without subsetting.
//
// This is the whole point of the branch in one assertion: embed a TrueType font
// as a CIDFontType2, write text with characters from three scripts, and read
// the same string back out.
func TestCIDFontType2(t *testing.T)       { validateCIDFontType2(t, false) }
func TestCIDFontType2Subset(t *testing.T) { validateCIDFontType2(t, true) }

func validateCIDFontType2(t *testing.T, useSubset bool) {
	t.Helper()
	const message = "Unicode русский язык Tiếng Việt"

	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	embedded, err := font.LoadPDType0FontSubset(document, openFont(t, liberationSans), useSubset)
	if err != nil {
		t.Fatalf("LoadPDType0FontSubset: %v", err)
	}
	writeText(t, document, page, embedded, 12, 50, 600, message)

	var out bytes.Buffer
	if err := document.Save(&out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// check that the extracted text matches what we wrote
	if got := strings.TrimSpace(unicodeText(t, out.Bytes())); got != message {
		t.Errorf("extracted %q, want %q", got, message)
	}
}

// TestIndicScripts is testBengali, testDevanagari, testDevanagari2 and
// testGujarati.
//
// **These four assert nothing in Java.** The render comparison prints to stderr
// rather than failing -- "don't fail, rendering is different on different
// systems, result must be viewed manually" -- and the text extraction assertion
// is commented out in the Java source. What they check is that a complex script
// font can be embedded, written, saved and read back without throwing, which is
// what the port checks too. The rendering half needs rendering.Backend and is
// not here.
func TestIndicScripts(t *testing.T) {
	for _, row := range []struct {
		name    string
		fontTTF string
		message string
	}{
		{"Bengali", "Lohit-Bengali.ttf", "বিতরণ শুরু হয়"},
		{"Devanagari", "Lohit-Devanagari.ttf", "हिन्दी अच्छा"},
		{"Devanagari2", "NotoSansDevanagari-Regular.ttf", "शक्ति"},
		{"Gujarati", "Lohit-Gujarati.ttf", "ગુજરાતી લખાણ"},
	} {
		t.Run(row.name, func(t *testing.T) {
			document := pdmodel.NewPDDocument()
			page := pdmodel.NewPDPageOfSize(common.A4)
			document.AddPage(page)

			embedded, err := font.LoadPDType0Font(document, openFont(t, indicFonts+row.fontTTF))
			if err != nil {
				t.Fatalf("LoadPDType0Font: %v", err)
			}
			writeText(t, document, page, embedded, 20, 50, 600, row.message)

			var out bytes.Buffer
			if err := document.Save(&out); err != nil {
				t.Fatalf("Save: %v", err)
			}
			if err := document.Close(); err != nil {
				t.Fatalf("Close: %v", err)
			}
			// reading it back has to work; what it extracts is not asserted,
			// because the Java does not assert it either
			unicodeText(t, out.Bytes())
		})
	}
}

// TestReuseEmbeddedSubsettedFont is testReuseEmbeddedSubsettedFont: write with
// an embedded subset, save, reopen, and write again with the *same* font read
// back out of the document. Both runs of text must survive.
func TestReuseEmbeddedSubsettedFont(t *testing.T) {
	const text1 = "The quick brown fox"
	const text2 = "xof nworb kciuq ehT"

	var out bytes.Buffer
	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPage()
	document.AddPage(page)

	embedded, err := font.LoadPDType0Font(document, openFont(t, liberationSans))
	if err != nil {
		t.Fatalf("LoadPDType0Font: %v", err)
	}
	writeText(t, document, page, embedded, 20, 50, 600, text1)
	if err := document.Save(&out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Append, while reusing the font subset
	appended, err := pdfbox.LoadPDFBytes(out.Bytes())
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	first := appended.Page(0)
	reused, err := first.Resources().GetFont(cos.GetPDFName("F1"))
	if err != nil {
		t.Fatalf("GetFont: %v", err)
	}
	if reused == nil {
		t.Fatal("the font F1 is not in the page's resources")
	}
	stream, err := pdmodel.NewPDPageContentStreamCompressed(appended, first,
		pdmodel.Append, true)
	if err != nil {
		t.Fatalf("NewPDPageContentStreamAppend: %v", err)
	}
	for _, step := range []struct {
		what string
		err  error
	}{
		{"BeginText", stream.BeginText()},
		{"SetFont", stream.SetFont(reused, 20)},
		{"NewLineAtOffset", stream.NewLineAtOffset(250, 600)},
		{"ShowText", stream.ShowText(text2)},
		{"EndText", stream.EndText()},
		{"Close", stream.Close()},
	} {
		if step.err != nil {
			t.Fatalf("%s: %v", step.what, step.err)
		}
	}
	out.Reset()
	if err := appended.Save(&out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := appended.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Test that both texts are there
	if got := strings.TrimSpace(unicodeText(t, out.Bytes())); got != text1+" "+text2 {
		t.Errorf("extracted %q, want %q", got, text1+" "+text2)
	}
}

// TestIsEmbeddingPermittedMultipleVersions is
// testIsEmbeddingPermittedMultipleVersions: every legal combination of the
// OS/2 fsType permission bits.
//
// Java mocks TrueTypeFont with Mockito and sets getOS2Windows().getFsType().
// Go has no Mockito and the port's TrueTypeFont is a struct, so the fsType goes
// into a real OS/2 table. The eight rows and their expected answers are Java's.
//
// The comments are Java's too, and they are the substance: bits 0x0006, 0x000A,
// 0x000C and 0x000E are illegal combinations for fsType version 3 and later,
// and PDFBox permits them anyway.
func TestIsEmbeddingPermittedMultipleVersions(t *testing.T) {
	for _, row := range []struct {
		fsType    int16
		permitted bool
		note      string
	}{
		{0x0000, true, "Installable embedding versions 0-3+"},
		// no test for 0001, since bit 0 is permanently reserved and its use is
		// deprecated
		{0x0002, false, "Restricted License embedding versions 0-3+"},
		// no test for 0011
		{0x0004, true, "Preview & Print embedding versions 0-3+"},
		// no test for 0101
		{0x0006, true, "Restricted License AND Preview & Print, versions 0-2"},
		// no test for 0111
		{0x0008, true, "Editable embedding versions 0-3+"},
		// no test for 1001
		{0x000A, true, "Restricted License AND Editable, versions 0-2"},
		// no test for 1011
		{0x000C, true, "Editable AND Preview & Print, versions 0-2"},
		// no test for 1101
		{0x000E, true, "Editable AND Preview & Print AND Restricted License, versions 0-2"},
		// no test for 1111
	} {
		t.Run(row.note, func(t *testing.T) {
			if got := font.IsEmbeddingPermittedForFsType(row.fsType); got != row.permitted {
				t.Errorf("fsType %#04x: embedding permitted = %v, want %v",
					uint16(row.fsType), got, row.permitted)
			}
		})
	}
}

// TestSurrogatePairCharacterExceptionIsBmpCodePoint is
// testSurrogatePairCharacterExceptionIsBmpCodePoint, PDFBOX-5812: a character
// the font has no glyph for is refused, and the failure names the character and
// its code point.
func TestSurrogatePairCharacterExceptionIsBmpCodePoint(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPage()
	document.AddPage(page)

	embedded, err := font.LoadPDType0Font(document, openFont(t, liberationSans))
	if err != nil {
		t.Fatalf("LoadPDType0Font: %v", err)
	}
	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	defer stream.Close()
	if err := stream.BeginText(); err != nil {
		t.Fatalf("BeginText: %v", err)
	}
	if err := stream.SetFont(embedded, 20); err != nil {
		t.Fatalf("SetFont: %v", err)
	}

	// U+3042 HIRAGANA LETTER A, inside the basic plane and not in this font
	wantShowTextPanic(t, stream, "あ",
		"could not find the glyphId for the character: あ, codePoint: 12354 (0x3042)")
}

// TestSurrogatePairCharacterExceptionIsValidCodePoint is
// testSurrogatePairCharacterExceptionIsValidCodePoint: the same, with the
// message Java asserts in full.
//
//	could not find the glyphId for the character: 𩸽, codePoint: 171581 (0x29E3D)
func TestSurrogatePairCharacterExceptionIsValidCodePoint(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPage()
	document.AddPage(page)

	embedded, err := font.LoadPDType0Font(document, openFont(t, liberationSans))
	if err != nil {
		t.Fatalf("LoadPDType0Font: %v", err)
	}
	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	defer stream.Close()
	if err := stream.BeginText(); err != nil {
		t.Fatalf("BeginText: %v", err)
	}
	if err := stream.SetFont(embedded, 20); err != nil {
		t.Fatalf("SetFont: %v", err)
	}

	wantShowTextPanic(t, stream, "𩸽",
		"could not find the glyphId for the character: 𩸽, codePoint: 171581 (0x29E3D)")
}

// TestEmbeddedFontWithZeroWidthChars is testEmbeddedFontWithZeroWidthChars: a
// zero width non-joiner survives the round trip and stays zero width.
func TestEmbeddedFontWithZeroWidthChars(t *testing.T) {
	// "abc" + U+200C ZERO WIDTH NON-JOINER + "def"
	const message = "abc‌def"

	var out bytes.Buffer
	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPage()
	document.AddPage(page)

	embedded, err := font.LoadPDType0Font(document, openFont(t, liberationSans))
	if err != nil {
		t.Fatalf("LoadPDType0Font: %v", err)
	}
	writeText(t, document, page, embedded, 20, 50, 600, message)
	if err := document.Save(&out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// verify that the text still contains zero-width characters
	extracted := strings.TrimSpace(unicodeText(t, out.Bytes()))
	if extracted != message {
		t.Errorf("extracted %q, want %q", extracted, message)
	}
	// Java counts UTF-16 units with String.length(); Go counts bytes, so the
	// count is over runes, which is the same seven characters.
	runes := []rune(extracted)
	if len(runes) != 7 {
		t.Errorf("extracted %d characters, want 7", len(runes))
	}
	if len(runes) > 3 && runes[3] != '‌' {
		t.Errorf("character 3 is %q, want the zero width non-joiner", runes[3])
	}
}

// wantShowTextPanic is `assertThrows(IllegalStateException.class, () ->
// contents.showText(message))` followed by the message assertion.
//
// IllegalStateException is unchecked, and this port renders an unchecked
// exception as a panic, so the recovery is where the Java's assertThrows is.
func wantShowTextPanic(t *testing.T, stream *pdmodel.PDPageContentStream,
	message, want string) {
	t.Helper()
	defer func() {
		got := recover()
		if got == nil {
			t.Errorf("ShowText(%q) returned; want the panic Java raises as "+
				"IllegalStateException", message)
			return
		}
		if fmt.Sprint(got) != want {
			t.Errorf("ShowText(%q) panicked with %q, want %q", message, got, want)
		}
	}()
	if err := stream.ShowText(message); err != nil {
		t.Errorf("ShowText(%q) = %v, want the panic Java raises", message, err)
	}
}

// TestSimpleTrueTypeFontEmbedding checks the other half of the branch, which
// TestFontEmbedding has no case for at all: PDTrueTypeFont.load embeds a font
// as a *simple* font -- one byte per character, an /Encoding and a /Widths
// array -- and never subsets, because PDTrueTypeFontEmbedder.buildSubset throws
// with the comment "use PDType0Font instead".
//
// Written from PDTrueTypeFontEmbedder and PDTrueTypeFont, since Java has no
// test that goes through them. It asserts what the Java writes: the subtype,
// the descriptor's symbolic flags, and that /FirstChar, /LastChar and /Widths
// agree with each other and with the encoding.
func TestSimpleTrueTypeFontEmbedding(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	embedded, err := font.LoadPDTrueTypeFont(document, openFont(t, liberationSans),
		encoding.WinAnsiEncodingInstance)
	if err != nil {
		t.Fatalf("LoadPDTrueTypeFont: %v", err)
	}

	dict := embedded.COSObject().(*cos.Dictionary)
	if got := dict.GetNameAsString(cos.Subtype, ""); got != "TrueType" {
		t.Errorf("/Subtype = %q, want %q", got, "TrueType")
	}

	descriptor := embedded.FontDescriptor()
	if descriptor == nil {
		t.Fatal("FontDescriptor() = nil, want the descriptor the embedder built")
	}
	// the embedder sets these two the opposite way round from the CID one
	if descriptor.IsSymbolic() {
		t.Error("the descriptor is symbolic, want non-symbolic for a simple font")
	}
	if !descriptor.IsNonSymbolic() {
		t.Error("the descriptor is not non-symbolic, want it to be")
	}

	firstChar := dict.GetIntDefault(cos.FirstChar, -1)
	lastChar := dict.GetIntDefault(cos.LastChar, -1)
	if firstChar < 0 || lastChar < firstChar {
		t.Fatalf("/FirstChar %d and /LastChar %d do not describe a range",
			firstChar, lastChar)
	}
	widths := dict.GetCOSArray(cos.Widths)
	if widths == nil {
		t.Fatal("/Widths is missing")
	}
	if widths.Size() != lastChar-firstChar+1 {
		t.Errorf("/Widths holds %d entries, want %d for /FirstChar %d to /LastChar %d",
			widths.Size(), lastChar-firstChar+1, firstChar, lastChar)
	}
	// 'A' is in WinAnsi and in this font, so it has a width
	if 'A' >= firstChar && 'A' <= lastChar {
		if got := widths.GetInt('A' - firstChar); got <= 0 {
			t.Errorf("the width of 'A' is %d, want a positive advance", got)
		}
	}
}

// TestSubsetKeepsTheGlyphsThatWereAskedFor is the other half of the branch's
// D8: read a document the port wrote back with the port's own fontbox parser,
// and check the glyphs the subset kept are the glyphs the text asked for.
//
// A subsetted CIDFontType2 has no cmap -- the subsetter keeps ten tables and
// cmap is not one of them -- so the way in is /CIDToGIDMap, which this branch
// writes: the CID is the original font's glyph id, and the entry it indexes is
// the id the same glyph has in the subset. The outline behind it has to be the
// outline the original font had for that character.
func TestSubsetKeepsTheGlyphsThatWereAskedFor(t *testing.T) {
	const message = "Unicode русский язык Tiếng Việt"

	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)
	embedded, err := font.LoadPDType0FontSubset(document, openFont(t, liberationSans), true)
	if err != nil {
		t.Fatalf("LoadPDType0FontSubset: %v", err)
	}
	writeText(t, document, page, embedded, 12, 50, 600, message)

	var out bytes.Buffer
	if err := document.Save(&out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// The original, to compare against.
	original, err := ttf.NewParser().Parse(mustOpenRead(t, liberationSans))
	if err != nil {
		t.Fatalf("parsing the original: %v", err)
	}
	defer original.Close()
	originalCmap, err := original.UnicodeCmapLookup(true)
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	originalGlyf, err := original.Glyph()
	if err != nil {
		t.Fatalf("Glyph: %v", err)
	}

	reloaded, err := pdfbox.LoadPDFBytes(out.Bytes())
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	defer reloaded.Close()

	names := reloaded.Page(0).Resources().FontNames()
	if len(names) != 1 {
		t.Fatalf("the reloaded page has %d fonts, want 1", len(names))
	}
	reloadedFont, err := reloaded.Page(0).Resources().GetFont(names[0])
	if err != nil {
		t.Fatalf("GetFont: %v", err)
	}
	type0, ok := reloadedFont.(*font.PDType0Font)
	if !ok {
		t.Fatalf("the reloaded font is %T, want *font.PDType0Font", reloadedFont)
	}
	if !type0.IsEmbedded() {
		t.Fatal("the reloaded font is not embedded")
	}
	descendant, ok := type0.DescendantFont().(*font.PDCIDFontType2)
	if !ok {
		t.Fatalf("the descendant is %T, want *font.PDCIDFontType2", type0.DescendantFont())
	}

	// Parse the /FontFile2 this branch wrote with the port's own parser.
	program := descendant.FontDescriptor().FontFile2()
	if program == nil {
		t.Fatal("the reloaded descendant has no /FontFile2")
	}
	programBytes, err := program.ToByteArray()
	if err != nil {
		t.Fatalf("reading /FontFile2: %v", err)
	}
	// The same call PDCIDFontType2 makes for an embedded program: a subsetted
	// font is missing post, cmap and name, and only the lenient parser takes it.
	subset, err := ttf.NewParserEmbedded(true).Parse(pdfio.NewReadBufferBytes(programBytes))
	if err != nil {
		t.Fatalf("parsing the embedded subset: %v", err)
	}
	defer subset.Close()
	subsetGlyf, err := subset.Glyph()
	if err != nil {
		t.Fatalf("the subset has no glyf table: %v", err)
	}

	seen := map[rune]bool{}
	for _, r := range message {
		if seen[r] {
			continue
		}
		seen[r] = true

		// Identity-H: the code is the CID, and for a subset the CID is the
		// original font's glyph id.
		cid := originalCmap.GetGlyphID(int(r))
		if cid == 0 {
			t.Errorf("the original font has no glyph for %q", r)
			continue
		}
		gid, err := descendant.CodeToGID(cid, type0)
		if err != nil {
			t.Errorf("CodeToGID(%d) for %q: %v", cid, r, err)
			continue
		}
		if gid == 0 {
			t.Errorf("/CIDToGIDMap maps CID %d (%q) to glyph 0; the subset dropped it", cid, r)
			continue
		}

		want, err := originalGlyf.GetGlyph(cid)
		if err != nil {
			t.Fatalf("the original glyph %d: %v", cid, err)
		}
		got, err := subsetGlyf.GetGlyph(gid)
		if err != nil {
			t.Fatalf("the subset glyph %d: %v", gid, err)
		}
		if got == nil {
			t.Errorf("the subset has no outline at glyph %d, for %q", gid, r)
			continue
		}
		if got.NumberOfContours() != want.NumberOfContours() ||
			got.XMinimum() != want.XMinimum() || got.YMinimum() != want.YMinimum() ||
			got.XMaximum() != want.XMaximum() || got.YMaximum() != want.YMaximum() {
			t.Errorf("the subset glyph %d for %q is %d contours in (%d,%d)-(%d,%d); "+
				"the original glyph %d is %d contours in (%d,%d)-(%d,%d)",
				gid, r, got.NumberOfContours(), got.XMinimum(), got.YMinimum(),
				got.XMaximum(), got.YMaximum(),
				cid, want.NumberOfContours(), want.XMinimum(), want.YMinimum(),
				want.XMaximum(), want.YMaximum())
		}
	}
}

// mustOpenRead opens a font file as a RandomAccessRead.
func mustOpenRead(t *testing.T, path string) pdfio.RandomAccessRead {
	t.Helper()
	source, err := pdfio.OpenBufferedFile(path)
	if err != nil {
		t.Fatalf("opening %s: %v", path, err)
	}
	return source
}
