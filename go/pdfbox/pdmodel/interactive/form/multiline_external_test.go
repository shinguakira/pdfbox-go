package form_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/form/AlignmentTest.java,
// AcroFormsRotationTest.java and MultilineFieldsTest.java.
//
// The rendering comparison each of the three fill tests ends with is not here:
// it renders the page and compares images, which is a later slice, and Java
// only prints when it differs rather than failing. See migration/STATUS.md.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdfparser"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// loadForm opens the named form fixture and answers it with its AcroForm.
func loadForm(t *testing.T, nameOfPDF string) (*pdmodel.PDDocument, *form.PDAcroForm) {
	t.Helper()
	document, err := pdfbox.LoadPDF(formFixture + nameOfPDF)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { document.Close() })
	return document, form.AcroFormOfCatalog(document.DocumentCatalog())
}

// saveTo writes the document into the temporary directory of the test, which is
// the OUT_DIR the Java tests save into.
func saveTo(t *testing.T, document *pdmodel.PDDocument, nameOfPDF string) {
	t.Helper()
	out, err := os.Create(filepath.Join(t.TempDir(), nameOfPDF))
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if err := document.Save(out); err != nil {
		t.Fatal(err)
	}
}

// setFieldValue sets the value of the named field.
func setFieldValue(t *testing.T, acroForm *form.PDAcroForm, name, value string) {
	t.Helper()
	field := acroForm.Field(name)
	if field == nil {
		t.Fatalf("Field(%q) = nil, want a field", name)
	}
	if err := field.SetValue(value); err != nil {
		t.Fatalf("Field(%q).SetValue(): %v", name, err)
	}
}

// TestAlignmentFillFields is AlignmentTest.fillFields.
func TestAlignmentFillFields(t *testing.T) {
	const nameOfPDF = "AlignmentTests.pdf"
	const testValue = "sdfASDF1234äöü"
	document, acroForm := loadForm(t, nameOfPDF)
	for _, name := range []string{
		"AlignLeft",
		"AlignLeft-Border_Small",
		"AlignLeft-Border_Medium",
		"AlignLeft-Border_Wide",
		"AlignLeft-Border_Wide_Clipped",
		"AlignLeft-Border_Small_Outside",
		"AlignMiddle",
		"AlignMiddle-Border_Small",
		"AlignMiddle-Border_Medium",
		"AlignMiddle-Border_Wide",
		"AlignMiddle-Border_Wide_Clipped",
		"AlignMiddle-Border_Medium_Outside",
		"AlignRight",
		"AlignRight-Border_Small",
		"AlignRight-Border_Medium",
		"AlignRight-Border_Wide",
		"AlignRight-Border_Wide_Clipped",
		"AlignRight-Border_Wide_Outside",
	} {
		setFieldValue(t, acroForm, name, testValue)
	}
	saveTo(t, document, nameOfPDF)
}

// TestRotationFillFields is AcroFormsRotationTest.fillFields.
func TestRotationFillFields(t *testing.T) {
	const nameOfPDF = "AcroFormsRotation.pdf"
	const testValue = "Lorem ipsum dolor sit amet, consetetur sadipscing elitr," +
		" sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam erat," +
		" sed diam voluptua."
	document, acroForm := loadForm(t, nameOfPDF)

	// portrait page
	// single line fields
	for _, name := range []string{
		"pdfbox.portrait.single.rotation0",
		"pdfbox.portrait.single.rotation90",
		"pdfbox.portrait.single.rotation180",
		"pdfbox.portrait.single.rotation270",
	} {
		field := acroForm.Field(name)
		setFieldValue(t, acroForm, name, field.FullyQualifiedName())
	}

	// multiline fields
	for _, name := range []string{
		"pdfbox.portrait.multi.rotation0",
		"pdfbox.portrait.multi.rotation90",
		"pdfbox.portrait.multi.rotation180",
		"pdfbox.portrait.multi.rotation270",
	} {
		field := acroForm.Field(name)
		setFieldValue(t, acroForm, name, field.FullyQualifiedName()+"\n"+testValue)
	}

	// 90 degrees rotated page
	// single line fields
	//
	// Java writes the name out rather than reading it back here, which comes to
	// the same thing.
	for _, name := range []string{
		"pdfbox.page90.single.rotation0",
		"pdfbox.page90.single.rotation90",
		"pdfbox.page90.single.rotation180",
		"pdfbox.page90.single.rotation270",
	} {
		setFieldValue(t, acroForm, name, name)
	}

	// multiline fields
	for _, name := range []string{
		"pdfbox.page90.multi.rotation0",
		"pdfbox.page90.multi.rotation90",
		"pdfbox.page90.multi.rotation180",
		"pdfbox.page90.multi.rotation270",
	} {
		field := acroForm.Field(name)
		setFieldValue(t, acroForm, name, field.FullyQualifiedName()+"\n"+testValue)
	}
	saveTo(t, document, nameOfPDF)
}

// TestMultilineFillFields is MultilineFieldsTest.fillFields.
func TestMultilineFillFields(t *testing.T) {
	const nameOfPDF = "MultilineFields.pdf"
	const testValue = "Lorem ipsum dolor sit amet, consetetur sadipscing elitr, " +
		"sed diam nonumy eirmod tempor invidunt ut labore et dolore magna aliquyam"
	document, acroForm := loadForm(t, nameOfPDF)
	for _, name := range []string{
		"AlignLeft",
		"AlignMiddle",
		"AlignRight",
		"AlignLeft-Border_Small",
		"AlignMiddle-Border_Small",
		"AlignRight-Border_Small",
		"AlignLeft-Border_Medium",
		"AlignMiddle-Border_Medium",
		"AlignRight-Border_Medium",
		"AlignLeft-Border_Wide",
		"AlignMiddle-Border_Wide",
		"AlignRight-Border_Wide",
	} {
		setFieldValue(t, acroForm, name, testValue)
	}
	saveTo(t, document, nameOfPDF)
}

// TestMultilineAuto is MultilineFieldsTest.testMultilineAuto, which is
// PDFBOX-3812.
func TestMultilineAuto(t *testing.T) {
	_, acroForm := loadForm(t, "PDFBOX3812-acrobat-multiline-auto.pdf")

	// Get and store the field sizes in the original PDF
	fieldMultiline := acroForm.Field("Multiline").(*form.PDTextField)
	fontSizeMultiline := fontSizeFromAppearanceStream(t, fieldMultiline)
	fieldSingleline := acroForm.Field("Singleline").(*form.PDTextField)
	fontSizeSingleline := fontSizeFromAppearanceStream(t, fieldSingleline)
	fieldMultilineAutoscale := acroForm.Field("MultilineAutoscale").(*form.PDTextField)
	fontSizeMultilineAutoscale := fontSizeFromAppearanceStream(t, fieldMultilineAutoscale)
	fieldSinglelineAutoscale := acroForm.Field("SinglelineAutoscale").(*form.PDTextField)
	fontSizeSinglelineAutoscale := fontSizeFromAppearanceStream(t, fieldSinglelineAutoscale)

	setFieldValue(t, acroForm, "Multiline", "Multiline - Fixed")
	setFieldValue(t, acroForm, "Singleline", "Singleline - Fixed")
	setFieldValue(t, acroForm, "MultilineAutoscale", "Multiline - auto")
	setFieldValue(t, acroForm, "SinglelineAutoscale", "Singleline - auto")

	assertCloseWithin(t, "Multiline",
		fontSizeFromAppearanceStream(t, fieldMultiline), fontSizeMultiline, 0.001)
	assertCloseWithin(t, "Singleline",
		fontSizeFromAppearanceStream(t, fieldSingleline), fontSizeSingleline, 0.001)
	assertCloseWithin(t, "MultilineAutoscale",
		fontSizeFromAppearanceStream(t, fieldMultilineAutoscale),
		fontSizeMultilineAutoscale, 0.001)
	assertCloseWithin(t, "SinglelineAutoscale",
		fontSizeFromAppearanceStream(t, fieldSinglelineAutoscale),
		fontSizeSinglelineAutoscale, 0.025)
}

// assertCloseWithin is the assertEquals of Java with a delta.
func assertCloseWithin(t *testing.T, what string, got, want, delta float32) {
	t.Helper()
	difference := got - want
	if difference < 0 {
		difference = -difference
	}
	if difference > delta {
		t.Errorf("%s font size = %v, want %v", what, got, want)
	}
}

// TestMultilineBreak is MultilineFieldsTest.testMultilineBreak, which is
// PDFBOX-3835.
func TestMultilineBreak(t *testing.T) {
	const testPDF = "PDFBOX-3835-input-acrobat-wrap.pdf"
	_, localAcroForm := loadForm(t, testPDF)

	// Get and store the field sizes in the original PDF
	fieldInput := localAcroForm.Field("filled").(*form.PDTextField)
	fieldValue := fieldInput.Value()
	acrobatLines := textLinesFromAppearanceStream(t, fieldInput)
	if err := fieldInput.SetValue(fieldValue); err != nil {
		t.Fatal(err)
	}
	pdfboxLines := textLinesFromAppearanceStream(t, fieldInput)
	if len(acrobatLines) != len(pdfboxLines) {
		t.Fatalf("Number of lines generated by PDFBox shall match Acrobat: %d, got %d",
			len(acrobatLines), len(pdfboxLines))
	}
	for i := range acrobatLines {
		if got, want := len([]rune(pdfboxLines[i])), len([]rune(acrobatLines[i])); got != want {
			t.Errorf("Number of characters per lines generated by PDFBox shall match "+
				"Acrobat: line %d is %d, want %d", i, got, want)
		}
	}
}

// fontSizeFromAppearanceStream is the private getFontSizeFromAppearanceStream.
func fontSizeFromAppearanceStream(t *testing.T, field form.PDField) float32 {
	t.Helper()
	for i, token := range appearanceTokens(t, field) {
		name, isName := token.(*cos.Name)
		if !isName || name.Name() != "Helv" {
			continue
		}
		if number, isNumber := appearanceTokens(t, field)[i+1].(cos.Number); isNumber {
			return number.FloatValue()
		}
	}
	return 0
}

// textLinesFromAppearanceStream is the private getTextLinesFromAppearanceStream.
func textLinesFromAppearanceStream(t *testing.T, field form.PDField) []string {
	t.Helper()
	lines := []string{}
	for _, token := range appearanceTokens(t, field) {
		if str, isString := token.(*cos.StringObj); isString {
			lines = append(lines, str.Value())
		}
	}
	return lines
}

// appearanceTokens parses the normal appearance of the first widget of the
// field, which both private helpers of the Java test walk with a parser.
func appearanceTokens(t *testing.T, field form.PDField) []any {
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
	return tokens
}
