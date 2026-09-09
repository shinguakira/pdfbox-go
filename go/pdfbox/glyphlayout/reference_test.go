package glyphlayout_test

// The layout, measured against what the Java AWT backend actually produced.
//
// `pdfbox-layout-awt/src/test/resources/pdf/` holds the PDFs `TestBase.
// checkRenderIdent` compares against: they are the output of
// `GlyphLayoutProcessorAwt` over `java.awt.font.TextLayout`, checked into the
// Java repository. The Java tests use them by rendering both documents and
// comparing pixels, which this port cannot do -- nothing implements
// `rendering.Backend`. But the shaping is in the content stream, and that can
// be read without a rasteriser: which glyph, in which order, moved by how much.
//
// So `testdata/awt-*.txt` is those PDFs read back: one line per text object,
// holding the fonts it sets, the glyphs it shows as the characters their
// ToUnicode gives, the positioning adjustments and the text rises. This runs
// the same page through the Go layout and writes the same thing.
//
// The comparison is exact where it agrees and pinned where it does not: a line
// that differs has to be in knownDeviations with a reason, and a line listed
// there that stops differing fails as well, so a deviation cannot quietly
// appear or quietly disappear. migration/STATUS.md carries the same list in
// prose.

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/glyphlayout"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// The strings of GlyphLayoutLigaturesAndKerningTest, verbatim.
const (
	firacodeString = "!= == === >= <="
	dejavuString   = "AVATAR, effective, affiliation, float, film, affluent"
	bengaliString  = "আমি কোন পথে ক্ষীরের লক্ষ্মী ষন্ড পুতুল রুপো গঙ্গা ঋষি"
	thaiString     = "กูกินก้งปิ้งอยู่ในถ้ำ"
	bengaliString2 = "হ্যালো ওয়ার্ল্ড"
)

// The strings of GlyphLayoutBidiTest, verbatim.
const (
	bidiText1 = "نحن الآن في شهر رمضان 1447 هجري"
	bidiText2 = "Guten Tag "
	bidiText3 = "السلام عليكم"
	bidiText4 = " Good afternoon"
)

// TestLigaturesAndKerningAgainstTheAwtReference is
// GlyphLayoutLigaturesAndKerningTest.testLigaturesAndKerning, laid out the same
// way and compared operator by operator.
func TestLigaturesAndKerningAgainstTheAwtReference(t *testing.T) {
	compareWithReference(t, "awt-GlyphLayoutLigaturesAndKerning.txt",
		ligaturesAndKerningPage, map[int]string{
			0: "FiraCode draws != and >= with contextual alternates: 111 chained " +
				"contextual lookups, which is GSUB lookup type 6, and PDFBox's " +
				"reader drops every one of them. The port asks for the feature and " +
				"the substitutions it needs are not there.",
			1: "the same, on the line that asks for ligatures.",
			6: "Thai: the vowel and tone signs have contextual forms, and U+0E33 " +
				"decomposes into U+0E4D and U+0E32. NotoSansThai spells both as five " +
				"type 6 lookups and one type 5, which the reader drops. The " +
				"positioning of the glyphs it does produce agrees with the AWT " +
				"reference to the last design unit.",
			7: "Bengali: GsubWorkerForBengali substitutes a different set of " +
				"conjuncts and pre-base forms than the platform does. Lohit-Bengali " +
				"is the one layout font whose lookups the reader reads in full, so " +
				"this one is the worker's own choices, not a missing lookup type.",
			8: "the same, on the text object the Java test writes without the " +
				"helper, to reach the end-position branch of showTextUni.",
		})
}

// TestBidiAgainstTheAwtReference is GlyphLayoutBidiTest.testGlyphLayoutBidi.
func TestBidiAgainstTheAwtReference(t *testing.T) {
	compareWithReference(t, "awt-GlyphLayoutBidi.txt", bidiPage, map[int]string{
		0: "Arabic joining: a letter takes an initial, medial or final form from " +
			"the letters around it, which is the init/medi/fina features driven by " +
			"the Unicode joining types. PDFBox's GsubWorkerFactory has no Arabic " +
			"worker, so the port draws the isolated forms. The run order, which is " +
			"what this test is named for, is right.",
		1: "the same, on the second line.",
	})
}

// TestSupplementaryPlaneAgainstTheAwtReference is
// GlyphLayoutSMPTest.testGlyphLayoutSMP, which the Java calls "not a new
// functionality but a regression test that needs to be checked": every letter
// on the page is two chars in Java and two UTF-16 units in the PDF, and the
// layout must not take them apart.
//
// Nothing on this page is shaped, so it is expected to agree everywhere.
func TestSupplementaryPlaneAgainstTheAwtReference(t *testing.T) {
	compareWithReference(t, "awt-GlyphLayoutSMP.txt", supplementaryPlanePage, nil)
}

// mathematicalCodePoints is GlyphLayoutSMPTest.MATHEMATICAL_CODEPOINTS,
// verbatim.
var mathematicalCodePoints = []rune{0x1D504, 0x1D505, 0x212D, 0x1D507, 0x1D508,
	0x1D509, 0x1D50A, 0x210C, 0x2111, 0x1D50D, 0x1D50E, 0x1D50F, 0x1D510, 0x1D511,
	0x1D512, 0x1D513, 0x1D514, 0x211C, 0x1D516, 0x1D517, 0x1D518, 0x1D519, 0x1D51A,
	0x1D51B, 0x1D51C, 0x2128, 0x0A, 0x1D51E, 0x1D51F, 0x1D520, 0x1D521, 0x1D522,
	0x1D523, 0x1D524, 0x1D525, 0x1D526, 0x1D527, 0x1D528, 0x1D529, 0x1D52A, 0x1D52B,
	0x1D52C, 0x1D52D, 0x1D52E, 0x1D52F, 0x1D530, 0x1D531, 0x1D532, 0x1D533, 0x1D534,
	0x1D535, 0x1D536, 0x1D537, 0x0A, 0x0A,
	0x1D7D8, 0x1D7D9, 0x1D7DA, 0x1D7DB, 0x1D7DC, 0x1D7DD, 0x1D7DE, 0x1D7DF, 0x1D7E0,
	0x1D7E1, 0x0A, 0x0A,
	0x1D49C, 0x212C, 0x1D49E, 0x1D49F, 0x2130, 0x2131, 0x1D4A2, 0x210B, 0x2110,
	0x1D4A5, 0x1D4A6, 0x2112, 0x2133, 0x1D4A9, 0x1D4AA, 0x1D4AB, 0x1D4AC, 0x211B,
	0x1D4AE, 0x1D4AF, 0x1D4B0, 0x1D4B1, 0x1D4B2, 0x1D4B3, 0x1D4B4, 0x1D4B5, 0x0A,
	0x1D4B6, 0x1D4B7, 0x1D4B8, 0x1D4B9, 0x212F, 0x1D4BB, 0x210A, 0x1D4BD, 0x1D4BE,
	0x1D4BF, 0x1D4C0, 0x1D4C1, 0x1D4C2, 0x1D4C3, 0x2134, 0x1D4C5, 0x1D4C6, 0x1D4C7,
	0x1D4C8, 0x1D4C9, 0x1D4CA, 0x1D4CB, 0x1D4CC, 0x1D4CD, 0x1D4CE, 0x1D4CF, 0x0A, 0x0A}

// smpIntro is GlyphLayoutSMPTest.TEXT_INTRO.
const smpIntro = "Test of Letters from the Supplementary Multilingual Plane, " +
	"Mathematical Alphanumeric Symbols"

// supplementaryPlanePage lays out the intro, the name of the font, and the
// mathematical alphabets, one line per newline in them.
func supplementaryPlanePage(t *testing.T, document *pdmodel.PDDocument,
	stream *pdmodel.PDPageContentStream) {
	t.Helper()
	sans := loadLayoutFont(t, document, "Arimo-Regular.ttf")
	math := loadLayoutFont(t, document, "NotoSansMath-Regular.ttf")
	processor := glyphlayout.NewProcessor()

	const size = 12
	const x = float32(12)
	y := float32(780)

	y = showComposites(t, stream, processor, sans, size, x, y, smpIntro)
	// The Java writes `"Font used: " + mathFont.getName()`, which before
	// subsetting is the font's own name.
	y = showComposites(t, stream, processor, sans, size, x, y,
		"Font used: NotoSansMath-Regular")
	for _, line := range strings.Split(string(mathematicalCodePoints), "\n") {
		if line == "" {
			continue
		}
		y = showComposites(t, stream, processor, math, size, x, y, line)
	}
}

// ligaturesAndKerningPage lays the page out the way the Java test does: one
// font per combination of options, one line each.
//
// Two things in the Java test are left out because they cannot change the
// shaping: the y it walks down the page, which the operators do not carry, and
// `checkRenderIdent`.
func ligaturesAndKerningPage(t *testing.T, document *pdmodel.PDDocument,
	stream *pdmodel.PDPageContentStream) {
	t.Helper()
	fira := loadLayoutFont(t, document, "FiraCode-Regular.ttf")
	dejavu := loadLayoutFont(t, document, "DejaVuSans.ttf")
	thai := loadLayoutFont(t, document, "NotoSansThai-Regular.ttf")
	lohit := loadLayoutFont(t, document, "Lohit-Bengali.ttf")

	plain := glyphlayout.NewProcessor()
	ligatures := glyphlayout.NewProcessorWithFeatures(
		glyphlayout.Features{Ligatures: true})
	kerning := glyphlayout.NewProcessorWithFeatures(
		glyphlayout.Features{Kerning: true})
	both := glyphlayout.NewProcessorWithFeatures(
		glyphlayout.Features{Ligatures: true, Kerning: true})

	// x and y are the Java's: `page.getBBox().getLowerLeftX() + fontSize` and
	// `page.getBBox().getUpperRightY() - fontSize` over a default page.
	const size = 12
	const x = float32(12)
	y := float32(780)

	y = showComposites(t, stream, plain, fira, size, x, y, firacodeString)
	y = showComposites(t, stream, ligatures, fira, size, x, y, firacodeString+" (Ligatures)")
	y = showComposites(t, stream, plain, dejavu, size, x, y, dejavuString)
	y = showComposites(t, stream, ligatures, dejavu, size, x, y, dejavuString+" (Ligatures)")
	y = showComposites(t, stream, kerning, dejavu, size, x, y, dejavuString+" (Kerning)")
	y = showComposites(t, stream, both, dejavu, size, x, y,
		dejavuString+" (Ligatures and kerning)")
	y = showComposites(t, stream, both, thai, size, x, y, thaiString)
	y = showComposites(t, stream, plain, lohit, size, x, y-5, bengaliString+" (ভারত)")

	// The Java test writes this one without the helper, to reach the "adjust
	// the end position" branch at the end of showTextUni.
	stream.SetGlyphLayoutProcessor(plain)
	mustDo(t, stream.BeginText())
	mustDo(t, stream.SetFont(lohit, 20))
	mustDo(t, stream.NewLineAtOffset(x, y-20))
	mustDo(t, stream.ShowText(bengaliString2))
	mustDo(t, stream.ShowText(" "))
	mustDo(t, stream.ShowText(bengaliString2))
	mustDo(t, stream.EndText())

	// The two rules the Java draws under the plain and the kerned line, whose
	// lengths are the measured widths of the same string. They are the reason
	// the test measures the widths at all -- the assertions on f1 to f4 are
	// TestStringWidthGrowsWithTheText's, and these put the numbers on the page.
	f3, err := plain.StringWidth(dejavu, size, dejavuString)
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}
	f4, err := both.StringWidth(dejavu, size, dejavuString)
	if err != nil {
		t.Fatalf("StringWidth: %v", err)
	}
	rule(t, stream, x, 737, x+f3)
	rule(t, stream, x, 676, x+f4)
}

// rule strokes a horizontal line, which is the Java's moveTo/lineTo/stroke.
func rule(t *testing.T, stream *pdmodel.PDPageContentStream, x0, y, x1 float32) {
	t.Helper()
	mustDo(t, stream.MoveTo(x0, y))
	mustDo(t, stream.LineTo(x1, y))
	mustDo(t, stream.Stroke())
}

// bidiPage lays out GlyphLayoutBidiTest's two lines, the second of which
// changes font twice inside one text object.
func bidiPage(t *testing.T, document *pdmodel.PDDocument,
	stream *pdmodel.PDPageContentStream) {
	t.Helper()
	arabic := loadLayoutFont(t, document, "NotoSansArabic-Regular.ttf")
	lgc := loadLayoutFont(t, document, "DejaVuSans.ttf")
	processor := glyphlayout.NewProcessor()
	stream.SetGlyphLayoutProcessor(processor)

	const size = 12
	const x = float32(12)
	y := float32(780)

	y = showComposites(t, stream, processor, arabic, size, x, y, bidiText1)

	mustDo(t, stream.BeginText())
	mustDo(t, stream.NewLineAtOffset(x, y))
	for _, part := range []struct {
		f    *font.PDType0Font
		text string
	}{{lgc, bidiText2}, {arabic, bidiText3}, {lgc, bidiText4}} {
		mustDo(t, stream.SetFont(part.f, size))
		mustDo(t, stream.ShowText(part.text))
	}
	mustDo(t, stream.EndText())
}

// showLine is TestBase.showCompositesLine.
func showLine(t *testing.T, stream *pdmodel.PDPageContentStream,
	processor *glyphlayout.Processor, f *font.PDType0Font, size, x, y float32, text string) {
	t.Helper()
	stream.SetGlyphLayoutProcessor(processor)
	mustDo(t, stream.BeginText())
	mustDo(t, stream.SetFont(f, size))
	mustDo(t, stream.NewLineAtOffset(x, y))
	mustDo(t, stream.ShowText(text))
	mustDo(t, stream.EndText())
}

// showComposites is TestBase.showComposites: draw the line and answer the y
// the next one starts at, a line height further down.
//
// The Java splits its argument on newlines and draws one line per piece; none
// of the strings these tests use has one, so this draws the one line.
//
// Until `track/raster` this took no y at all and every line went to 700, so
// they were drawn on top of each other. The operator comparison could not see
// it -- what it dumps is the fonts, the glyphs and the adjustments inside each
// text object, and never the `Td` that places it -- and the pixel comparison
// A1 added found it on its first run.
func showComposites(t *testing.T, stream *pdmodel.PDPageContentStream,
	processor *glyphlayout.Processor, f *font.PDType0Font, size, x, y float32,
	text string) float32 {
	t.Helper()
	showLine(t, stream, processor, f, size, x, y, text)
	bbox, err := f.BoundingBox()
	if err != nil {
		t.Fatalf("BoundingBox: %v", err)
	}
	return y - bbox.Height()/1000*size
}

// loadLayoutFont loads one of the fonts the Java tests use.
func loadLayoutFont(t *testing.T, document *pdmodel.PDDocument,
	name string) *font.PDType0Font {
	t.Helper()
	file, err := os.Open(layoutFonts + name)
	if err != nil {
		t.Skipf("%s is not in this repository: %v", name, err)
	}
	t.Cleanup(func() { file.Close() })
	embedded, err := font.LoadPDType0FontSubset(document, file, true)
	if err != nil {
		t.Fatalf("loading %s: %v", name, err)
	}
	return embedded
}

func mustDo(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// compareWithReference lays the page out, reads back what was written, and
// checks it against the recorded AWT output.
func compareWithReference(t *testing.T, reference string,
	page func(*testing.T, *pdmodel.PDDocument, *pdmodel.PDPageContentStream),
	knownDeviations map[int]string) {
	t.Helper()
	expected := readReference(t, "testdata/"+reference)
	actual := layOutAndRead(t, page)

	if len(actual) != len(expected) {
		t.Errorf("the page has %d text objects and the AWT reference has %d",
			len(actual), len(expected))
	}
	for index, reason := range knownDeviations {
		if index >= len(expected) {
			t.Errorf("text object %d is listed as a deviation and the reference has "+
				"only %d: %s", index, len(expected), reason)
		}
	}
	for i := 0; i < len(expected) && i < len(actual); i++ {
		reason, isKnown := knownDeviations[i]
		same := sameOperator(expected[i], actual[i])
		switch {
		case same && isKnown:
			t.Errorf("text object %d agrees with the AWT reference, and is listed as a "+
				"deviation: %s\nremove it from the list and from migration/STATUS.md",
				i, reason)
		case !same && isKnown:
			t.Logf("text object %d differs, as recorded: %s\n  AWT: %s\n  Go:  %s",
				i, reason, expected[i], actual[i])
		case !same:
			t.Errorf("text object %d does not match the AWT reference:\n  AWT: %s\n  Go:  %s",
				i, expected[i], actual[i])
		}
	}
}

// layOutAndRead writes the page, saves it, and reads its text operators back.
func layOutAndRead(t *testing.T,
	page func(*testing.T, *pdmodel.PDDocument, *pdmodel.PDPageContentStream)) []string {
	t.Helper()
	reloaded := layOutAndReload(t, page)
	defer reloaded.Close()
	return textOperatorsOf(t, reloaded)
}

// layOutAndReload writes the page, saves it and loads it back, which is what
// the Java tests do before they compare -- the file on disk is what is being
// checked, not the object graph that made it. The caller closes the document.
func layOutAndReload(t *testing.T,
	page func(*testing.T, *pdmodel.PDDocument, *pdmodel.PDPageContentStream)) *pdmodel.PDDocument {
	t.Helper()
	document := pdmodel.NewPDDocument()
	defer document.Close()
	pdPage := pdmodel.NewPDPage()
	document.AddPage(pdPage)

	stream, err := pdmodel.NewPDPageContentStream(document, pdPage)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	page(t, document, stream)
	if err := stream.Close(); err != nil {
		t.Fatalf("closing the content stream: %v", err)
	}

	var saved bytes.Buffer
	if err := document.Save(&saved); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reloaded, err := pdfbox.LoadPDFBytes(saved.Bytes())
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	return reloaded
}

// positionTolerance is how far two positions may differ and still be the same
// position.
//
// The numbers compared are thousandths of the font size, so this is a
// fifty-thousandth of an em: a hundredth of a pixel at 500 point. It is not
// zero because Java's numbers do not come from the font's design units the way
// this port's do -- AWT lays the run out in points at the size asked for, and
// `showTextUni` scales that back up by 1000/fontSize -- so a value the font
// declares as 9 units comes back out of the reference PDF as 8.99506.
const positionTolerance = 0.02

// sameOperator compares two dumped text objects, token by token, numerically
// where the token is a number.
func sameOperator(expected, actual string) bool {
	expectedFields, actualFields := movingFields(expected), movingFields(actual)
	if len(expectedFields) != len(actualFields) {
		return false
	}
	for i := range expectedFields {
		if expectedFields[i] == actualFields[i] {
			continue
		}
		left, leftOK := numberOf(expectedFields[i])
		right, rightOK := numberOf(actualFields[i])
		if !leftOK || !rightOK {
			return false
		}
		if difference := left - right; difference > positionTolerance ||
			difference < -positionTolerance {
			return false
		}
	}
	return true
}

// movingFields splits a dumped text object into tokens, dropping the movements
// that do not move anything.
//
// AWT lays a run out in points and this port in font design units, so a
// position AWT reaches by adding and subtracting floats can come out a
// millionth of an em away from where it started, and `showTextUni` writes that
// as an adjustment because Java's own threshold for writing one -- 1e-5 points
// -- is smaller still. A movement below the tolerance is not a movement, and
// one side writing it while the other does not is not a difference in shaping.
func movingFields(line string) []string {
	var fields []string
	for _, field := range strings.Fields(line) {
		if value, isNumber := numberOf(field); isNumber &&
			value < positionTolerance && value > -positionTolerance {
			continue
		}
		fields = append(fields, field)
	}
	// A space at the end of a line paints nothing, and the reference PDFs have
	// some the Java that renders them no longer produces: twenty lines of
	// GlyphLayoutDIN91379.pdf end with one, and no line of
	// LATIN_CHARS_DIN_91379 does. The PDF was rendered from an earlier
	// spelling of that string, and `checkRenderIdent` never noticed because it
	// compares pixels. Neither side's trailing space is a shaping difference.
	for len(fields) != 0 && fields[len(fields)-1] == "G:0020" {
		fields = fields[:len(fields)-1]
	}
	return fields
}

// numberOf answers the value of an adjustment or a rise, and false for a token
// that is neither.
func numberOf(field string) (float64, bool) {
	for _, prefix := range []string{"A:", "R:"} {
		if !strings.HasPrefix(field, prefix) {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimPrefix(field, prefix), 64)
		return value, err == nil
	}
	return 0, false
}

// readReference reads a recorded dump, dropping blank lines.
func readReference(t *testing.T, path string) []string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the AWT reference: %v", err)
	}
	var lines []string
	for _, line := range strings.Split(strings.ReplaceAll(
		string(content), "\r\n", "\n"), "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// textOperatorsOf writes a document's shaping in the format the recorded AWT
// dumps are in: one line per text object, holding the fonts it sets (F:), the
// glyphs it shows as the characters their ToUnicode gives (G:), the
// positioning adjustments (A:) and the text rises (R:), in the order they were
// written.
//
// One line per text object rather than per operator, because where the two
// break a run into operators is not the question: a text rise splits one run
// into three, and a shaping difference of one glyph would then shift every
// comparison after it.
func textOperatorsOf(t *testing.T, document *pdmodel.PDDocument) []string {
	t.Helper()
	contents := contentsOfPage(t, document)

	var lines []string
	var line strings.Builder
	var current font.PDFont
	scanner := &operatorScanner{contents: contents}
	for {
		operator, operands, ok := scanner.next()
		if !ok {
			break
		}
		switch operator {
		case "BT":
			line.Reset()
		case "ET":
			if text := strings.TrimSpace(line.String()); text != "" {
				lines = append(lines, text)
			}
			line.Reset()
		case "Tf":
			if len(operands) >= 2 {
				current = fontNamed(t, document, operands[len(operands)-2])
				line.WriteString("F:" + fontNameOf(current) + " ")
			}
		case "Ts":
			if len(operands) >= 1 {
				line.WriteString("R:" + operands[0] + " ")
			}
		case "Tj", "'":
			if len(operands) >= 1 {
				line.WriteString(decodeOperand(t, current, operands[len(operands)-1]))
			}
		case "TJ":
			if len(operands) >= 1 {
				line.WriteString(decodeArray(t, current, operands[len(operands)-1]))
			}
		}
	}
	return lines
}

// contentsOfPage answers the first page's content stream.
func contentsOfPage(t *testing.T, document *pdmodel.PDDocument) string {
	t.Helper()
	reader, err := document.Page(0).Contents()
	if err != nil {
		t.Fatalf("Contents: %v", err)
	}
	defer func() {
		if closer, ok := reader.(interface{ Close() error }); ok {
			closer.Close()
		}
	}()
	var buffer bytes.Buffer
	if _, err := buffer.ReadFrom(reader); err != nil {
		t.Fatalf("reading the contents: %v", err)
	}
	return buffer.String()
}

// fontNamed answers the page's font of the given resource name.
func fontNamed(t *testing.T, document *pdmodel.PDDocument, operand string) font.PDFont {
	t.Helper()
	found, err := document.Page(0).Resources().GetFont(cosNameOf(trimName(operand)))
	if err != nil {
		t.Fatalf("resolving the font %s: %v", operand, err)
	}
	return found
}

// fontNameOf answers a font's name without the subset prefix, which is six
// letters and a plus sign and differs between two runs.
func fontNameOf(f font.PDFont) string {
	if f == nil {
		return "?"
	}
	name := f.Name()
	if len(name) > 7 && name[6] == '+' {
		return name[7:]
	}
	return name
}

// decodeArray writes one TJ array: its strings decoded, its numbers as
// adjustments.
func decodeArray(t *testing.T, f font.PDFont, operand string) string {
	t.Helper()
	var out strings.Builder
	inner := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(operand), "["), "]")
	scanner := &operatorScanner{contents: inner}
	for _, element := range scanner.operands() {
		if strings.HasPrefix(element, "(") || strings.HasPrefix(element, "<") {
			out.WriteString(decodeOperand(t, f, element))
			continue
		}
		out.WriteString("A:" + element + " ")
	}
	return out.String()
}

// decodeOperand writes one string operand as its glyphs.
func decodeOperand(t *testing.T, f font.PDFont, operand string) string {
	t.Helper()
	if f == nil {
		return "G:? "
	}
	codes := bytes.NewReader(stringBytesOf(operand))
	var out strings.Builder
	for codes.Len() > 0 {
		code, err := f.ReadCode(codes)
		if err != nil {
			t.Fatalf("reading a code: %v", err)
		}
		unicode, err := f.ToUnicode(code)
		if err != nil {
			t.Fatalf("ToUnicode: %v", err)
		}
		out.WriteString("G:")
		if unicode == "" {
			out.WriteString(fmt.Sprintf("code%d", code))
		} else {
			for i, unit := range utf16UnitsOf(unicode) {
				if i > 0 {
					out.WriteString("+")
				}
				out.WriteString(fmt.Sprintf("%04X", unit))
			}
		}
		out.WriteString(" ")
	}
	return out.String()
}
