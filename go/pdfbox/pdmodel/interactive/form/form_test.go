package form

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/form/PlainTextTest.java,
// PDTextFieldTest.java, PDDefaultAppearanceStringTest.java, TestCheckBox.java
// and PDSignatureFieldTest.java.
//
// PDFieldTreeTest is not here: it downloads a PDF from the issue tracker, and
// the port runs no test that reaches the network. See migration/STATUS.md.

import (
	"bytes"
	"math"
	"slices"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/graphics/color"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/digitalsignature"
)

// TestPlainTextParagraphs is the six tests of PlainTextTest, which differ only
// in the line break they hold.
func TestPlainTextParagraphs(t *testing.T) {
	for _, c := range []struct {
		name string
		text string
		want int
	}{
		{"characterCR", "CR\rCR", 2},
		{"characterLF", "LF\nLF", 2},
		{"characterCRLF", "CRLF\r\nCRLF", 2},
		{"characterLFCR", "LFCR\n\rLFCR", 3},
		{"characterUnicodeLinebreak", "linebreak linebreak", 2},
		{"characterUnicodeParagraphbreak", "paragraphbreak paragraphbreak", 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			text := interactive.NewPlainText(c.text)
			if got := len(text.Paragraphs()); got != c.want {
				t.Errorf("Paragraphs() = %d, want %d", got, c.want)
			}
		})
	}
}

// TestCreateDefaultTextField is PDTextFieldTest.createDefaultTextField.
func TestCreateDefaultTextField(t *testing.T) {
	document := pdmodel.NewPDDocument()
	acroForm := NewPDAcroForm(document)
	textField := NewPDTextField(acroForm)
	if got, want := textField.FieldType(),
		textField.FieldDictionary().GetNameAsString(cos.FT, ""); got != want {
		t.Errorf("FieldType() = %q, want %q", got, want)
	}
	if got, want := textField.FieldType(), "Tx"; got != want {
		t.Errorf("FieldType() = %q, want %q", got, want)
	}
}

// TestCreateWidgetForGet is PDTextFieldTest.createWidgetForGet.
func TestCreateWidgetForGet(t *testing.T) {
	document := pdmodel.NewPDDocument()
	acroForm := NewPDAcroForm(document)
	textField := NewPDTextField(acroForm)
	if got := textField.FieldDictionary().GetItem(cos.Type); got != nil {
		t.Errorf("/Type = %v, want nil", got)
	}
	if got := textField.FieldDictionary().GetNameAsString(cos.Subtype, ""); got != "" {
		t.Errorf("/Subtype = %q, want %q", got, "")
	}
	widget := textField.Widgets()[0]
	if got := textField.FieldDictionary().GetItem(cos.Type); got != cos.Base(cos.Annot) {
		t.Errorf("/Type = %v, want %v", got, cos.Annot)
	}
	if got, want := textField.FieldDictionary().GetNameAsString(cos.Subtype, ""),
		annotation.SubTypeWidget; got != want {
		t.Errorf("/Subtype = %q, want %q", got, want)
	}
	if widget.AnnotationDictionary() != textField.FieldDictionary() {
		t.Error("the widget dictionary is not the field dictionary")
	}
}

// TestParseDAString is PDDefaultAppearanceStringTest.testParseDAString.
func TestParseDAString(t *testing.T) {
	resources := pdmodel.NewPDResources()
	helvetica, err := font.NewPDType1FontStandard14(font.Helvetica)
	if err != nil {
		t.Fatal(err)
	}
	// the resource name is created when the font is added so need to capture
	// that
	fontResourceName := resources.AddFont(helvetica)

	sampleString := cos.NewStringObj("/" + fontResourceName.Name() + " 12 Tf 0.019 0.305 0.627 rg")
	defaultAppearanceString, err := newPDDefaultAppearanceString(sampleString, resources)
	if err != nil {
		t.Fatal(err)
	}
	if got := defaultAppearanceString.FontSize(); math.Abs(float64(got-12)) > 0.001 {
		t.Errorf("FontSize() = %v, want 12", got)
	}
	// Java asserts equality, and PDFont.equals compares the dictionaries.
	if got := defaultAppearanceString.Font(); !got.Equals(helvetica) {
		t.Errorf("Font() = %v, want %v", got, helvetica)
	}
	if got := defaultAppearanceString.FontColor().ColorSpace(); got !=
		color.PDColorSpace(color.DeviceRGB) {
		t.Errorf("ColorSpace() = %v, want %v", got, color.DeviceRGB)
	}
	for i, want := range []float32{0.019, 0.305, 0.627} {
		got := defaultAppearanceString.FontColor().Components()[i]
		if math.Abs(float64(got-want)) > 0.0001 {
			t.Errorf("Components()[%d] = %v, want %v", i, got, want)
		}
	}
}

// TestFontResourceUnavailable is
// PDDefaultAppearanceStringTest.testFontResourceUnavailable.
func TestFontResourceUnavailable(t *testing.T) {
	resources := pdmodel.NewPDResources()
	sampleString := cos.NewStringObj("/Helvetica 12 Tf 0.019 0.305 0.627 rg")
	if _, err := newPDDefaultAppearanceString(sampleString, resources); err == nil {
		t.Error("newPDDefaultAppearanceString() = nil error, want an error")
	}
}

// TestWrongNumberOfColorArguments is
// PDDefaultAppearanceStringTest.testWrongNumberOfColorArguments.
func TestWrongNumberOfColorArguments(t *testing.T) {
	resources := pdmodel.NewPDResources()
	sampleString := cos.NewStringObj("/Helvetica 12 Tf 0.305 0.627 rg")
	if _, err := newPDDefaultAppearanceString(sampleString, resources); err == nil {
		t.Error("newPDDefaultAppearanceString() = nil error, want an error")
	}
}

// TestCheckboxPDModel is TestCheckBox.testCheckboxPDModel.
func TestCheckboxPDModel(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	acroForm := NewPDAcroForm(doc)
	checkBox := NewPDCheckBox(acroForm)

	// test that there are no nulls returned for an empty field
	// only specific methods are tested here
	if checkBox.ExportValues() == nil {
		t.Error("ExportValues() = nil, want an empty list")
	}
	// Java also asserts getValue() is not null; the port answers a string,
	// which cannot be.

	// Test setting/getting option values - the dictionaries Opt entry
	options := []string{"Value01", "Value02"}
	checkBox.SetExportValues(options)
	optItem, isArray := checkBox.FieldDictionary().GetItem(cos.Opt).(*cos.Array)

	// assert that the values have been correctly set
	if !isArray {
		t.Fatalf("/Opt = %T, want *cos.Array", checkBox.FieldDictionary().GetItem(cos.Opt))
	}
	if got := optItem.Size(); got != 2 {
		t.Errorf("/Opt size = %d, want 2", got)
	}
	if got, want := optItem.GetString(0, ""), options[0]; got != want {
		t.Errorf("/Opt[0] = %q, want %q", got, want)
	}

	// assert that the values can be retrieved correctly
	retrievedOptions := checkBox.ExportValues()
	if got := len(retrievedOptions); got != 2 {
		t.Errorf("ExportValues() size = %d, want 2", got)
	}
	if !slices.Equal(retrievedOptions, options) {
		t.Errorf("ExportValues() = %v, want %v", retrievedOptions, options)
	}

	// assert that the Opt entry is removed
	checkBox.SetExportValues(nil)
	if got := checkBox.FieldDictionary().GetItem(cos.Opt); got != nil {
		t.Errorf("/Opt = %v, want nil", got)
	}

	// if there is no Opt entry an empty List shall be returned
	if got := checkBox.ExportValues(); len(got) != 0 {
		t.Errorf("ExportValues() = %v, want empty", got)
	}
}

// TestCheckBoxNoAppearance is TestCheckBox.testCheckBoxNoAppearance.
func TestCheckBoxNoAppearance(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	page := pdmodel.NewPDPage()
	doc.AddPage(page)
	acroForm := NewPDAcroForm(doc)
	acroForm.SetNeedAppearances(true) // need this or it won't appear on Adobe Reader
	SetAcroFormOfCatalog(doc.DocumentCatalog(), acroForm)

	checkBox := NewPDCheckBox(acroForm)
	checkBox.SetPartialName("checkbox")
	widget := checkBox.Widgets()[0]
	widget.SetRectangle(common.NewPDRectangleOf(50, 600, 100, 100))
	bs := annotation.NewPDBorderStyleDictionary()
	bs.SetStyle(annotation.BorderStyleSolid)
	bs.SetWidth(1)
	acd := cos.NewDictionary()
	ac := annotation.NewPDAppearanceCharacteristicsDictionaryOf(acd)
	ac.SetBackground(color.NewPDColorOfComponents([]float32{1, 1, 0}, color.DeviceRGB))
	ac.SetBorderColour(color.NewPDColorOfComponents([]float32{1, 0, 0}, color.DeviceRGB))
	ac.SetNormalCaption("4") // 4 is checkmark, 8 is cross
	widget.SetAppearanceCharacteristics(ac)
	widget.SetBorderStyle(bs)
	if err := checkBox.SetValue("Off"); err != nil {
		t.Fatal(err)
	}
	page.Annotations().Add(widget)
	acroForm.SetFields([]PDField{checkBox})
	if got := checkBox.Value(); got != "Off" {
		t.Errorf("Value() = %q, want %q", got, "Off")
	}
}

// TestCreateDefaultSignatureField is
// PDSignatureFieldTest.createDefaultSignatureField.
func TestCreateDefaultSignatureField(t *testing.T) {
	document := pdmodel.NewPDDocument()
	acroForm := NewPDAcroForm(document)
	sigField := NewPDSignatureField(acroForm)
	sigField.SetPartialName("SignatureField")
	if got, want := sigField.FieldType(),
		sigField.FieldDictionary().GetNameAsString(cos.FT, ""); got != want {
		t.Errorf("FieldType() = %q, want %q", got, want)
	}
	if got, want := sigField.FieldType(), "Sig"; got != want {
		t.Errorf("FieldType() = %q, want %q", got, want)
	}
	if got := sigField.FieldDictionary().GetItem(cos.Type); got != cos.Base(cos.Annot) {
		t.Errorf("/Type = %v, want %v", got, cos.Annot)
	}
	if got, want := sigField.FieldDictionary().GetNameAsString(cos.Subtype, ""),
		annotation.SubTypeWidget; got != want {
		t.Errorf("/Subtype = %q, want %q", got, want)
	}

	// Add the field to the acroform
	acroForm.SetFields([]PDField{sigField})
	if got := acroForm.Field("SignatureField"); got == nil {
		t.Error("Field(\"SignatureField\") = nil, want a field")
	}
}

// TestSetValueForAbstractedSignatureField is
// PDSignatureFieldTest.setValueForAbstractedSignatureField. Java asserts
// UnsupportedOperationException, which is unchecked, so the port panics.
func TestSetValueForAbstractedSignatureField(t *testing.T) {
	document := pdmodel.NewPDDocument()
	acroForm := NewPDAcroForm(document)
	sigField := NewPDSignatureField(acroForm)
	sigField.SetPartialName("SignatureField")
	defer func() {
		if recover() == nil {
			t.Error("SetValue() did not panic")
		}
	}()
	sigField.SetValue("Can't set value using String") //nolint:errcheck // it panics
}

// TestGetContents is PDSignatureFieldTest.testGetContents.
func TestGetContents(t *testing.T) {
	// Normally, range0 + range1 = position of "<", and range2 = position after ">"
	signature := digitalsignature.NewPDSignature()
	signature.SetByteRange([]int{0, 10, 30, 10})
	by := []byte("AAAAAAAAAA<313233343536373839>BBBBBBBBBB")
	contents, err := signature.ContentsOfBytes(by)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), "123456789"; got != want {
		t.Errorf("ContentsOfBytes() = %q, want %q", got, want)
	}
	contents, err = signature.ContentsOfReader(bytes.NewReader(by))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(contents), "123456789"; got != want {
		t.Errorf("ContentsOfReader() = %q, want %q", got, want)
	}
}
