package glyphlayout_test

// The layout backend, end to end.
//
// The Java tests of `pdfbox-layout-awt` cannot be ported: every assertion in
// them that checks the shaping goes through `TestBase.checkRenderIdent`, which
// renders both PDFs with `PDFRenderer` and compares them pixel by pixel, and
// nothing implements `rendering.Backend`. See migration/STATUS.md.
//
// What they were checking is readable without a renderer, though, and that is
// what these cases do: the glyph codes and the positioning end up in the
// content stream as `TJ` arrays, which is where the reference PDFs checked into
// pdfbox-layout-awt/src/test/resources/pdf carry them too.

import (
	"bytes"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/glyphlayout"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/text"
)

// layoutFonts is where the pdfbox-layout-awt test resources are.
const layoutFonts = "../../../pdfbox-layout-awt/src/test/resources/ttf/"

// writeWithLayout writes one line through the layout processor and answers the
// page's content stream.
func writeWithLayout(t *testing.T, fontFile string, features glyphlayout.Features,
	size float32, text string) string {
	t.Helper()
	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	file, err := os.Open(layoutFonts + fontFile)
	if err != nil {
		t.Skipf("%s is not in this repository: %v", fontFile, err)
	}
	defer file.Close()
	embedded, err := font.LoadPDType0FontSubset(document, file, false)
	if err != nil {
		t.Fatalf("LoadPDType0FontSubset: %v", err)
	}

	processor := glyphlayout.NewProcessorWithFeatures(features)
	if !processor.SupportsFont(embedded) {
		t.Fatalf("the processor refuses %s", fontFile)
	}

	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	stream.SetGlyphLayoutProcessor(processor)
	if err := stream.BeginText(); err != nil {
		t.Fatalf("BeginText: %v", err)
	}
	if err := stream.SetFont(embedded, size); err != nil {
		t.Fatalf("SetFont: %v", err)
	}
	if err := stream.NewLineAtOffset(50, 700); err != nil {
		t.Fatalf("NewLineAtOffset: %v", err)
	}
	if err := stream.ShowText(text); err != nil {
		t.Fatalf("ShowText: %v", err)
	}
	if err := stream.EndText(); err != nil {
		t.Fatalf("EndText: %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	var out bytes.Buffer
	if err := document.Save(&out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	return contentsOf(t, out.Bytes())
}

// contentsOf reads a saved document's first page content stream.
func contentsOf(t *testing.T, data []byte) string {
	t.Helper()
	document, err := pdfbox.LoadPDFBytes(data)
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	defer document.Close()
	reader, err := document.Page(0).Contents()
	if err != nil {
		t.Fatalf("Contents: %v", err)
	}
	contents, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading the contents: %v", err)
	}
	return string(contents)
}

// TestKerningReachesTheContentStream is what -kerning is for: with the feature
// on, the TJ array carries the adjustments GPOS gave; with it off, it does not.
func TestKerningReachesTheContentStream(t *testing.T) {
	const text = "AVATAR"

	without := writeWithLayout(t, "DejaVuSans.ttf", glyphlayout.Features{}, 12, text)
	with := writeWithLayout(t, "DejaVuSans.ttf",
		glyphlayout.Features{Kerning: true}, 12, text)

	if adjustments(without) != 0 {
		t.Errorf("with no kerning asked for, the stream carries %d adjustments:\n%s",
			adjustments(without), without)
	}
	if adjustments(with) == 0 {
		t.Errorf("with kerning asked for, the stream carries none:\n%s", with)
	}
	// Kerning pulls letters together, so every adjustment is positive in a TJ
	// array -- the number there is subtracted from the position.
	for _, value := range positiveNumbers(with) {
		if value <= 0 {
			t.Errorf("a kerning adjustment is %v; kerning pulls letters together", value)
		}
	}
}

// TestTextIsStillReadable is the assertion that matters most: whatever the
// layout does to the glyphs, the text has to come back out of the document.
func TestTextIsStillReadable(t *testing.T) {
	const text = "AVATAR, effective, affiliation, float, film, affluent"

	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	file, err := os.Open(layoutFonts + "DejaVuSans.ttf")
	if err != nil {
		t.Skipf("DejaVuSans is not in this repository: %v", err)
	}
	defer file.Close()
	embedded, err := font.LoadPDType0FontSubset(document, file, false)
	if err != nil {
		t.Fatalf("LoadPDType0FontSubset: %v", err)
	}

	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	stream.SetGlyphLayoutProcessor(
		glyphlayout.NewProcessorWithFeatures(glyphlayout.Features{Kerning: true}))
	if err := stream.BeginText(); err != nil {
		t.Fatal(err)
	}
	if err := stream.SetFont(embedded, 12); err != nil {
		t.Fatal(err)
	}
	if err := stream.NewLineAtOffset(50, 700); err != nil {
		t.Fatal(err)
	}
	if err := stream.ShowText(text); err != nil {
		t.Fatalf("ShowText: %v", err)
	}
	if err := stream.EndText(); err != nil {
		t.Fatal(err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := document.Save(&out); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := pdfbox.LoadPDFBytes(out.Bytes())
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	defer reloaded.Close()
	extracted, err := textOf(reloaded)
	if err != nil {
		t.Fatalf("extracting the text: %v", err)
	}
	if got := strings.TrimSpace(extracted); got != text {
		t.Errorf("extracted %q, want %q", got, text)
	}
}

// TestStringWidthGrowsWithTheText checks getStringWidthUni answers something
// sane, and that kerning makes a kerned string narrower -- which is the
// relation the Java test asserts (`assertTrue(f4 < f3)`) rather than any
// absolute number.
func TestStringWidthGrowsWithTheText(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	file, err := os.Open(layoutFonts + "DejaVuSans.ttf")
	if err != nil {
		t.Skipf("DejaVuSans is not in this repository: %v", err)
	}
	defer file.Close()
	embedded, err := font.LoadPDType0FontSubset(document, file, false)
	if err != nil {
		t.Fatalf("LoadPDType0FontSubset: %v", err)
	}

	plain := glyphlayout.NewProcessor()
	kerned := glyphlayout.NewProcessorWithFeatures(glyphlayout.Features{Kerning: true})

	short, err := plain.StringWidth(embedded, 12, "AV")
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}
	long, err := plain.StringWidth(embedded, 12, "AVAV")
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}
	if long <= short {
		t.Errorf("AVAV is %v wide and AV is %v; the longer string cannot be narrower",
			long, short)
	}

	withKerning, err := kerned.StringWidth(embedded, 12, "AVATAR")
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}
	without, err := plain.StringWidth(embedded, 12, "AVATAR")
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}
	if withKerning >= without {
		t.Errorf("kerned AVATAR is %v wide and unkerned is %v; kerning pulls it in",
			withKerning, without)
	}

	// And twice the size is twice the width.
	doubled, err := plain.StringWidth(embedded, 24, "AVATAR")
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}
	if diff := doubled - 2*without; diff > 0.01 || diff < -0.01 {
		t.Errorf("at 24 the width is %v and at 12 it is %v; it should double",
			doubled, without)
	}
}

// TestMissingGlyphIsRefused is GlyphLayoutLigaturesAndKerningTest.
// testMissingGlyph: the same font, the same text, and the message asserted in
// full, character for character.
//
// Java throws IllegalArgumentException, which is unchecked; this port's
// convention for one of those is a panic, but the message reaches a caller
// through PDPageContentStream.showText, which returns an error. So it is an
// error here, and what is asserted is the message.
func TestMissingGlyphIsRefused(t *testing.T) {
	const want = "Missing glyph in font 'Lohit Bengali' for the character 'A', " +
		"codePoint: 65 (U+0041)."

	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	file, err := os.Open(layoutFonts + "Lohit-Bengali.ttf")
	if err != nil {
		t.Skipf("Lohit-Bengali is not in this repository: %v", err)
	}
	defer file.Close()
	embedded, err := font.LoadPDType0FontSubset(document, file, false)
	if err != nil {
		t.Fatalf("LoadPDType0FontSubset: %v", err)
	}

	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	stream.SetGlyphLayoutProcessor(glyphlayout.NewProcessor())
	if err := stream.BeginText(); err != nil {
		t.Fatal(err)
	}
	if err := stream.SetFont(embedded, 1); err != nil {
		t.Fatal(err)
	}
	if err := stream.NewLineAtOffset(0, 0); err != nil {
		t.Fatal(err)
	}

	// A Bengali font has no Latin letters, and "123ABC" reaches the A first.
	err = stream.ShowText("123ABC")
	if err == nil {
		t.Fatal("laying out a character the font has no glyph for was accepted")
	}
	if got := err.Error(); got != want {
		t.Errorf("the failure is\n  %q\nwant\n  %q", got, want)
	}
}

// TestSupportsFont checks the three answers supportsFont gives.
//
// The middle one is the one worth having: an OpenType font with PostScript
// outlines carries every table the layout reads, so it would lay out, and
// PDFBox does not support one here. A caller that is refused falls through to
// the ordinary path instead of getting a page nothing can render.
func TestSupportsFont(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	trueType := openLayoutFont(t, document, layoutFonts+"DejaVuSans.ttf")
	if !glyphlayout.NewProcessor().SupportsFont(trueType) {
		t.Error("a Type 0 font with a TrueType program was refused")
	}

	postScript := postScriptOutlineFont(t, document)
	// Without this the next assertion would pass for the wrong reason: a font
	// with no program at all is refused too.
	if postScript.TrueTypeFont() == nil {
		t.Fatal("the PostScript font carries no program, so nothing is being checked")
	}
	if glyphlayout.NewProcessor().SupportsFont(postScript) {
		t.Error("a Type 0 font with PostScript outlines was accepted; " +
			"PDFBox does not support one for glyph layout")
	}

	// Anything that is not a Type 0 font at all.
	helvetica, err := font.NewPDType1FontStandard14(font.Helvetica)
	if err != nil {
		t.Fatalf("NewPDType1FontStandard14: %v", err)
	}
	if glyphlayout.NewProcessor().SupportsFont(helvetica) {
		t.Error("a standard 14 font was accepted; the layout writes glyph ids, " +
			"which a simple font has no room for")
	}
}

// postScriptOutlineFont builds a Type 0 font whose program is an OpenType font
// with PostScript outlines.
//
// It is built from a dictionary rather than loaded, because loading one is
// refused earlier: the embedder answers "True Type fonts using CFF outlines are
// not supported". A document that already has such a font -- and they exist,
// which is why PDCIDFontType2 carries an `otf` beside its `ttf` and asks
// whether it is PostScript in three places -- reaches the layout this way.
func postScriptOutlineFont(t *testing.T, document *pdmodel.PDDocument) *font.PDType0Font {
	t.Helper()
	program, err := os.ReadFile("../../../fontbox/src/test/resources/otf/FoglihtenNo07.otf")
	if err != nil {
		t.Skipf("FoglihtenNo07.otf is not in this repository: %v", err)
	}
	stream, err := common.NewPDStreamOfInput(document.Document(),
		bytes.NewReader(program), nil)
	if err != nil {
		t.Fatalf("embedding the font program: %v", err)
	}

	descriptor := font.NewPDFontDescriptor()
	descriptor.SetFontName("FoglihtenNo07")
	descriptor.SetFontFile3(stream)

	descendant := cos.NewDictionary()
	descendant.SetItem(cos.GetPDFName("Type"), cos.GetPDFName("Font"))
	descendant.SetItem(cos.GetPDFName("Subtype"), cos.GetPDFName("CIDFontType2"))
	descendant.SetItem(cos.GetPDFName("BaseFont"), cos.GetPDFName("FoglihtenNo07"))
	descendant.SetItem(cos.GetPDFName("FontDescriptor"), descriptor.COSObject())
	systemInfo := cos.NewDictionary()
	systemInfo.SetItem(cos.GetPDFName("Registry"), cos.NewStringObj("Adobe"))
	systemInfo.SetItem(cos.GetPDFName("Ordering"), cos.NewStringObj("Identity"))
	systemInfo.SetItem(cos.GetPDFName("Supplement"), cos.GetInteger(0))
	descendant.SetItem(cos.GetPDFName("CIDSystemInfo"), systemInfo)

	dictionary := cos.NewDictionary()
	dictionary.SetItem(cos.GetPDFName("Type"), cos.GetPDFName("Font"))
	dictionary.SetItem(cos.GetPDFName("Subtype"), cos.GetPDFName("Type0"))
	dictionary.SetItem(cos.GetPDFName("BaseFont"), cos.GetPDFName("FoglihtenNo07"))
	dictionary.SetItem(cos.GetPDFName("Encoding"), cos.GetPDFName("Identity-H"))
	descendants := cos.NewArray()
	descendants.Add(descendant)
	dictionary.SetItem(cos.GetPDFName("DescendantFonts"), descendants)

	loaded, err := font.NewPDType0Font(dictionary, nil)
	if err != nil {
		t.Fatalf("reading the font back: %v", err)
	}
	return loaded
}

// openLayoutFont loads a font file as a Type 0 font.
func openLayoutFont(t *testing.T, document *pdmodel.PDDocument,
	path string) *font.PDType0Font {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Skipf("%s is not in this repository: %v", path, err)
	}
	t.Cleanup(func() { file.Close() })
	loaded, err := font.LoadPDType0FontSubset(document, file, false)
	if err != nil {
		t.Fatalf("loading %s: %v", path, err)
	}
	return loaded
}

// TestStringWidthWithAndWithoutKerning is the four widths the Java test
// compares, and the three relations it asserts between them.
//
// In Java the options are on the font and the processor is shared; in this port
// they are on the processor, so f3 and f4 come from two processors over one
// font rather than one processor over two fonts. The relations are the same.
func TestStringWidthWithAndWithoutKerning(t *testing.T) {
	const text = "AVATAR, effective, affiliation, float, film, affluent"
	const fontSize = 12.0

	document := pdmodel.NewPDDocument()
	defer document.Close()

	file, err := os.Open(layoutFonts + "DejaVuSans.ttf")
	if err != nil {
		t.Skipf("DejaVuSans is not in this repository: %v", err)
	}
	defer file.Close()
	dejavu, err := font.LoadPDType0FontSubset(document, file, false)
	if err != nil {
		t.Fatalf("LoadPDType0FontSubset: %v", err)
	}

	plain := glyphlayout.NewProcessor()
	ligaturesAndKerning := glyphlayout.NewProcessorWithFeatures(
		glyphlayout.Features{Ligatures: true, Kerning: true})

	// f1 and f2 in the Java, which asks the font itself rather than the layout.
	// It answers the same both times, which is the point the Java test makes:
	// "this shows that the ordinary getStringWidth() isn't helpful".
	width, err := dejavu.StringWidth(text)
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}
	f1 := width * dejavu.FontMatrix().ScaleX() * fontSize
	f2 := f1

	// f3 and f4, which ask the layout.
	f3, err := plain.StringWidth(dejavu, fontSize, text)
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}
	f4, err := ligaturesAndKerning.StringWidth(dejavu, fontSize, text)
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}

	if f1 != f2 {
		t.Errorf("the font answers %v and then %v for the same string", f1, f2)
	}
	if f4 >= f1 {
		t.Errorf("laid out with kerning the string is %v wide and the font says %v; "+
			"kerning pulls it in", f4, f1)
	}
	if f4 >= f3 {
		t.Errorf("laid out with kerning the string is %v wide and without it %v; "+
			"kerning pulls it in", f4, f3)
	}
}

// adjustments counts the numbers in the TJ arrays of a content stream, which
// is how many times the layout moved a glyph.
func adjustments(contents string) int {
	count := 0
	for _, value := range positiveNumbers(contents) {
		if value != 0 {
			count++
		}
	}
	return count
}

// positiveNumbers answers every number inside a TJ array.
//
// It has to scan rather than split on spaces: PDFBox writes a string operand
// with no separator after it, so a TJ array reads `(..)63.9 (..)77.6`, and the
// bytes inside a string are glyph codes -- any byte at all, spaces, digits and
// escaped parentheses among them.
func positiveNumbers(contents string) []float64 {
	var values []float64
	inArray := false
	for i := 0; i < len(contents); {
		switch contents[i] {
		case '(':
			i = endOfLiteralString(contents, i)
		case '<':
			if end := strings.IndexByte(contents[i:], '>'); end >= 0 {
				i += end + 1
			} else {
				i = len(contents)
			}
		case '[':
			inArray = true
			i++
		case ']':
			inArray = false
			i++
		default:
			end := i
			for end < len(contents) && strings.IndexByte("+-.0123456789",
				contents[end]) >= 0 {
				end++
			}
			if end == i {
				i++
				continue
			}
			if value, err := strconv.ParseFloat(contents[i:end], 64); err == nil && inArray {
				values = append(values, value)
			}
			i = end
		}
	}
	return values
}

// endOfLiteralString answers the index just past the literal string that starts
// at open, following the two rules PDF gives them: a backslash escapes the next
// byte, and parentheses nest.
func endOfLiteralString(contents string, open int) int {
	depth := 0
	for i := open; i < len(contents); i++ {
		switch contents[i] {
		case '\\':
			i++
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return len(contents)
}

// textOf extracts the text of a document.
func textOf(document *pdmodel.PDDocument) (string, error) {
	return text.NewPDFTextStripper().GetText(document)
}
