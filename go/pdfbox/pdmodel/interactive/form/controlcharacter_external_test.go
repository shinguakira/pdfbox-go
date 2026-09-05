package form_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/form/ControlCharacterTest.java,
// HandleDifferentDALevelsTest.java, CombAlignmentTest.java and the TestUtils
// they share.

import (
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// stringsFromStream is TestUtils.getStringsFromStream.
func stringsFromStream(t *testing.T, field form.PDField) []string {
	t.Helper()
	widget := field.Widgets()[0]
	content, err := widget.NormalAppearanceStream().ContentsForRandomAccess()
	if err != nil {
		t.Fatal(err)
	}
	parser, err := pdfparser.NewStreamTokenParserSource(content)
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := parser.Parse()
	if err != nil {
		t.Fatal(err)
	}
	// TODO: improve the string output to better match
	// trimming as Acrobat adds spaces to strings
	// where we don't
	out := []string{}
	for _, token := range tokens {
		if str, isString := token.(*cos.StringObj); isString {
			out = append(out, strings.TrimSpace(str.Value()))
		}
	}
	return out
}

// loadControlCharacters is the setUp of ControlCharacterTest.
func loadControlCharacters(t *testing.T) (*pdmodel.PDDocument, *form.PDAcroForm) {
	t.Helper()
	document, err := pdfbox.LoadPDF(formFixture + "ControlCharacters.pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { document.Close() })
	return document, form.AcroFormOfCatalog(document.DocumentCatalog())
}

// TestCharacterNUL is ControlCharacterTest.characterNUL.
//
// Java asserts IllegalArgumentException. The font encoders of this port answer
// that as an error rather than as a panic -- see PDType1Font.encodeCodePoint,
// which slice 4 wrote that way -- so what is asserted here is that the value is
// refused, not how.
func TestCharacterNUL(t *testing.T) {
	_, acroForm := loadControlCharacters(t)
	field := acroForm.Field("pdfbox-nul")
	if err := field.SetValue("NUL\x00NUL"); err == nil {
		t.Error("SetValue() = nil error, want an error")
	}
}

// TestCharacterTAB is ControlCharacterTest.characterTAB. There is no direct
// comparison to how Acrobat sets the value as we don't position with tabs.
func TestCharacterTAB(t *testing.T) {
	_, acroForm := loadControlCharacters(t)
	field := acroForm.Field("pdfbox-tab")
	if err := field.SetValue("TAB\tTAB"); err != nil {
		t.Fatal(err)
	}
	for _, token := range stringsFromStream(t, field) {
		if token != "TAB" {
			t.Errorf("token = %q, want %q", token, "TAB")
		}
	}
}

// TestControlCharacters is the parameterised ControlCharacterTest.testCharacter.
func TestControlCharacters(t *testing.T) {
	for _, c := range []struct {
		nameSuffix string
		value      string
	}{
		{"space", "SPACE SPACE"},
		{"cr", "CR\rCR"},
		{"lf", "LF\nLF"},
		{"crlf", "CRLF\r\nCRLF"},
		{"lfcr", "LFCR\n\rLFCR"},
		{"linebreak", "linebreak linebreak"},
		{"paragraphbreak", "paragraphbreak paragraphbreak"},
	} {
		t.Run(c.nameSuffix, func(t *testing.T) {
			_, acroForm := loadControlCharacters(t)
			field := acroForm.Field("pdfbox-" + c.nameSuffix)
			if err := field.SetValue(c.value); err != nil {
				t.Fatal(err)
			}
			pdfboxValues := stringsFromStream(t, field)
			acrobatValues := stringsFromStream(t, acroForm.Field("acrobat-"+c.nameSuffix))
			if !slices.Equal(pdfboxValues, acrobatValues) {
				t.Errorf("pdfbox = %q, acrobat = %q", pdfboxValues, acrobatValues)
			}
		})
	}
}

// loadDifferentDALevels is the setUp of HandleDifferentDALevelsTest, which
// prefills the fields to generate the appearance streams and saves the result.
func loadDifferentDALevels(t *testing.T) *form.PDAcroForm {
	t.Helper()
	const nameOfPDF = "DifferentDALevels.pdf"
	document, err := pdfbox.LoadPDF(formFixture + nameOfPDF)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { document.Close() })
	acroForm := form.AcroFormOfCatalog(document.DocumentCatalog())

	// prefill the fields to generate the appearance streams
	for _, prefill := range []struct{ name, value string }{
		{"SingleAnnotation", "single annotation"},
		{"MultipeAnnotations-SameLayout", "same layout"},
		{"MultipleAnnotations-DifferentLayout", "different layout"},
	} {
		field := acroForm.Field(prefill.name).(*form.PDTextField)
		if err := field.SetValue(prefill.value); err != nil {
			t.Fatal(err)
		}
	}
	out, err := os.Create(filepath.Join(t.TempDir(), nameOfPDF))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := document.Save(out); err != nil {
		t.Fatal(err)
	}
	return acroForm
}

// fontSettingFromFieldDA is the private getFontSettingFromDA(PDTextField).
func fontSettingFromFieldDA(field *form.PDTextField) string {
	defaultAppearance := field.DefaultAppearance()
	// get the font setting from the default appearance string
	return defaultAppearance[:strings.LastIndex(defaultAppearance, "Tf")+2]
}

// fontSettingFromWidgetDA is the private
// getFontSettingFromDA(PDAnnotationWidget). Java answers null where the widget
// has no /DA; the port answers the empty string, which is the same absence.
func fontSettingFromWidgetDA(widget *annotation.PDAnnotationWidget) string {
	defaultAppearance := widget.AnnotationDictionary().GetString(cos.DA, "")
	if defaultAppearance != "" {
		return defaultAppearance[:strings.LastIndex(defaultAppearance, "Tf")+2]
	}
	return defaultAppearance
}

// assertWidgetsCarryFontSetting is the loop the three checks of
// HandleDifferentDALevelsTest share.
func assertWidgetsCarryFontSetting(t *testing.T, field *form.PDTextField, perWidget bool) {
	t.Helper()
	fieldFontSetting := fontSettingFromFieldDA(field)
	for _, widget := range field.Widgets() {
		fontSetting := fieldFontSetting
		if perWidget {
			if widgetFontSetting := fontSettingFromWidgetDA(widget); widgetFontSetting != "" {
				fontSetting = widgetFontSetting
			}
		}
		reader, err := widget.NormalAppearanceStream().PDStream().CreateInputStream()
		if err != nil {
			t.Fatal(err)
		}
		contentAsBytes, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Index(string(contentAsBytes), fontSetting) <= 0 {
			t.Errorf("font setting in content stream shall be %q", fontSetting)
		}
	}
}

// TestCheckSingleAnnotation is HandleDifferentDALevelsTest.checkSingleAnnotation.
func TestCheckSingleAnnotation(t *testing.T) {
	acroForm := loadDifferentDALevels(t)
	field := acroForm.Field("SingleAnnotation").(*form.PDTextField)
	assertWidgetsCarryFontSetting(t, field, false)
}

// TestCheckSameLayout is HandleDifferentDALevelsTest.checkSameLayout.
func TestCheckSameLayout(t *testing.T) {
	acroForm := loadDifferentDALevels(t)
	field := acroForm.Field("MultipeAnnotations-SameLayout").(*form.PDTextField)
	assertWidgetsCarryFontSetting(t, field, false)
}

// TestCheckDifferentLayout is HandleDifferentDALevelsTest.checkDifferentLayout.
func TestCheckDifferentLayout(t *testing.T) {
	acroForm := loadDifferentDALevels(t)
	field := acroForm.Field("MultipleAnnotations-DifferentLayout").(*form.PDTextField)
	assertWidgetsCarryFontSetting(t, field, true)
}

// TestCombFields is CombAlignmentTest.testCombFields, which is PDFBOX-5256.
//
// The rendering comparison at the end is not here: it renders the page and
// compares images, which is a later slice, and Java only prints when it
// differs rather than failing. See migration/STATUS.md.
func TestCombFields(t *testing.T) {
	const nameOfPDF = "CombTest.pdf"
	const testValue = "1234567"
	document, err := pdfbox.LoadPDF(formFixture + nameOfPDF)
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	acroForm := form.AcroFormOfCatalog(document.DocumentCatalog())
	for _, name := range []string{"PDFBoxCombLeft", "PDFBoxCombMiddle", "PDFBoxCombRight"} {
		field := acroForm.Field(name)
		if err := field.SetValue(""); err != nil {
			t.Fatal(err)
		}
		if err := field.SetValue(testValue); err != nil {
			t.Fatal(err)
		}
	}
	out, err := os.Create(filepath.Join(t.TempDir(), nameOfPDF))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := document.Save(out); err != nil {
		t.Fatal(err)
	}
}

// TestPDFBOX5784 is CombAlignmentTest.testPDFBOX5784; see TestCombFields for
// the rendering comparison.
func TestPDFBOX5784(t *testing.T) {
	const nameOfPDF = "PDFBOX-5784.pdf"
	document, err := pdfbox.LoadPDF(formFixture + nameOfPDF)
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	acroForm := form.AcroFormOfCatalog(document.DocumentCatalog())
	for field := range acroForm.FieldTree().All() {
		if !strings.Contains(field.PartialName(), "acrobat") {
			if err := field.SetValue("WIaqg"); err != nil {
				t.Fatal(err)
			}
		}
	}
	out, err := os.Create(filepath.Join(t.TempDir(), nameOfPDF))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := document.Save(out); err != nil {
		t.Fatal(err)
	}
}

// TestCombFieldRefusesASupplementaryCharacter pins the one place where the
// port's comb layout could differ from PDFBox's. It is not a port: PDFBox has
// no test for it, and the slice 8 review feedback asked for the deviation to be
// pinned down rather than left as a comment.
//
// insertGeneratedCombAppearance walks the value one cell at a time. Java takes
// a cell with value.substring(i, i+1), which is one UTF-16 code unit, and the
// port takes one rune; the two differ only for a character outside the basic
// plane, where Java splits the surrogate pair across two cells and the port
// keeps it in one.
//
// The difference is not reachable. Java draws each cell through
// PDFont.getStringWidth, so its first half-a-pair cell asks the font for
// U+D83D; the port asks for U+1F600. Neither is in a standard 14 font, so both
// refuse the value rather than laying it out. This test asserts that the port
// refuses, so that the claim stops being true loudly rather than silently if
// the encoder ever starts accepting such a character.
func TestCombFieldRefusesASupplementaryCharacter(t *testing.T) {
	document, err := pdfbox.LoadPDF(formFixture + "CombTest.pdf")
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	acroForm := form.AcroFormOfCatalog(document.DocumentCatalog())
	field := acroForm.Field("PDFBoxCombRight")
	if field == nil {
		t.Fatal(`Field("PDFBoxCombRight") = nil, want the comb field`)
	}
	// U+1F600 GRINNING FACE, one rune and two UTF-16 code units.
	if err := field.SetValue("12\U0001F60034"); err == nil {
		t.Error("SetValue with a supplementary character was accepted, " +
			"so the comb layout is now reachable for one and the rune walk " +
			"in insertGeneratedCombAppearance differs from PDFBox")
	}
}
