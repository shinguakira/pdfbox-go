package form_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/form/TestFields.java
// and PDChoiceTest.java.

import (
	"slices"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	_ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fixup"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// formFixture is the directory the form test PDFs live in.
const formFixture = "../../../../../pdfbox/src/test/resources/org/apache/pdfbox/pdmodel/interactive/form/"

// pathOfPDF is the PATH_OF_PDF of TestFields.
const pathOfPDF = formFixture + "AcroFormsBasicFields.pdf"

// TestFlags is TestFields.testFlags.
func TestFlags(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	acroForm := form.NewPDAcroForm(doc)
	textBox := form.NewPDTextField(acroForm)

	// assert that default is false.
	assertFlag(t, "IsComb", textBox.IsComb(), false)

	// try setting and clearing a single field
	textBox.SetComb(true)
	assertFlag(t, "IsComb", textBox.IsComb(), true)
	textBox.SetComb(false)
	assertFlag(t, "IsComb", textBox.IsComb(), false)

	// try setting and clearing multiple fields
	textBox.SetComb(true)
	textBox.SetDoNotScroll(true)
	assertFlag(t, "IsComb", textBox.IsComb(), true)
	assertFlag(t, "DoNotScroll", textBox.DoNotScroll(), true)
	textBox.SetComb(false)
	textBox.SetDoNotScroll(false)
	assertFlag(t, "IsComb", textBox.IsComb(), false)
	assertFlag(t, "DoNotScroll", textBox.DoNotScroll(), false)

	// assert that setting a field to false multiple times works
	textBox.SetComb(false)
	assertFlag(t, "IsComb", textBox.IsComb(), false)
	textBox.SetComb(false)
	assertFlag(t, "IsComb", textBox.IsComb(), false)

	// assert that setting a field to true multiple times works
	textBox.SetComb(true)
	assertFlag(t, "IsComb", textBox.IsComb(), true)
	textBox.SetComb(true)
	assertFlag(t, "IsComb", textBox.IsComb(), true)
}

// assertFlag is the assertTrue and assertFalse of Java over a field flag.
func assertFlag(t *testing.T, what string, got, want bool) {
	t.Helper()
	if got != want {
		t.Errorf("%s() = %v, want %v", what, got, want)
	}
}

// TestAcroFormsBasicFields is TestFields.testAcroFormsBasicFields.
func TestAcroFormsBasicFields(t *testing.T) {
	doc, err := pdfbox.LoadPDF(pathOfPDF)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	// get and assert that there is a form
	acroForm := form.AcroFormOfCatalog(doc.DocumentCatalog())
	if acroForm == nil {
		t.Fatal("form.AcroFormOfCatalog() = nil, want a form")
	}

	// assert that there is no value, set the field value and ensure it has been
	// set
	textField := acroForm.Field("TextField").(*form.PDTextField)
	if got := textField.FieldDictionary().GetItem(cos.V); got != nil {
		t.Errorf("/V = %v, want nil", got)
	}
	if err := textField.SetValue("field value"); err != nil {
		t.Fatal(err)
	}
	if got := textField.FieldDictionary().GetItem(cos.V); got == nil {
		t.Error("/V = nil, want a value")
	}
	if got, want := textField.Value(), "field value"; got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}

	// Java then calls setValue(null) and asserts the key has been removed. The
	// port cannot: setValue takes a string, and cos.Dictionary.SetString
	// records the empty string as a value rather than as an absence, which its
	// comment says outright. The assertion is dropped rather than rewritten
	// into something setValue does not do; see migration/STATUS.md.

	// get the TextField with a DV entry
	textField = acroForm.Field("TextField-DefaultValue").(*form.PDTextField)
	if textField == nil {
		t.Fatal("Field(\"TextField-DefaultValue\") = nil, want a field")
	}
	if got, want := textField.DefaultValue(), "DefaultValue"; got != want {
		t.Errorf("DefaultValue() = %q, want %q", got, want)
	}
	if got, want := textField.DefaultValue(),
		textField.FieldDictionary().GetDictionaryObject(cos.DV).(*cos.StringObj).Value(); got != want {
		t.Errorf("DefaultValue() = %q, want %q", got, want)
	}
	if got, want := textField.DefaultAppearance(), "/Helv 12 Tf 0 g"; got != want {
		t.Errorf("DefaultAppearance() = %q, want %q", got, want)
	}

	// get a rich text field with a DV entry
	textField = acroForm.Field("RichTextField-DefaultValue").(*form.PDTextField)
	if textField == nil {
		t.Fatal("Field(\"RichTextField-DefaultValue\") = nil, want a field")
	}
	if got, want := textField.DefaultValue(), "DefaultValue"; got != want {
		t.Errorf("DefaultValue() = %q, want %q", got, want)
	}
	if got, want := textField.DefaultValue(),
		textField.FieldDictionary().GetDictionaryObject(cos.DV).(*cos.StringObj).Value(); got != want {
		t.Errorf("DefaultValue() = %q, want %q", got, want)
	}
	if got, want := textField.Value(), "DefaultValue"; got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if got, want := textField.DefaultAppearance(), "/Helv 12 Tf 0 g"; got != want {
		t.Errorf("DefaultAppearance() = %q, want %q", got, want)
	}
	if got, want := textField.DefaultStyleString(),
		"font: Helvetica,sans-serif 12.0pt; text-align:left; color:#000000 "; got != want {
		t.Errorf("DefaultStyleString() = %q, want %q", got, want)
	}
	// do not test for the full content as this is a rather long xml string
	//
	// Java measures the length in UTF-16 code units; the port measures it in
	// runes, which is the same count for this ASCII-only string.
	if got := len([]rune(textField.RichTextValue())); got != 338 {
		t.Errorf("RichTextValue() length = %d, want 338", got)
	}

	// get a rich text field with a text stream for the value
	textField = acroForm.Field("LongRichTextField").(*form.PDTextField)
	if textField == nil {
		t.Fatal("Field(\"LongRichTextField\") = nil, want a field")
	}
	if _, isStream := textField.FieldDictionary().
		GetDictionaryObject(cos.V).(*cos.Stream); !isStream {
		t.Errorf("/V = %T, want *cos.Stream",
			textField.FieldDictionary().GetDictionaryObject(cos.V))
	}
	if got := len([]rune(textField.Value())); got != 145396 {
		t.Errorf("Value() length = %d, want 145396", got)
	}
}

// TestWidgetMissingRect is TestFields.testWidgetMissingRect.
func TestWidgetMissingRect(t *testing.T) {
	doc, err := pdfbox.LoadPDF(pathOfPDF)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	acroForm := form.AcroFormOfCatalog(doc.DocumentCatalog())
	textField := acroForm.Field("TextField-DefaultValue").(*form.PDTextField)
	widget := textField.Widgets()[0]

	// initially there is an Appearance Entry in the form
	if got := widget.AnnotationDictionary().GetDictionaryObject(cos.AP); got == nil {
		t.Error("/AP = nil, want an appearance")
	}
	widget.AnnotationDictionary().RemoveItem(cos.Rect)
	if err := textField.SetValue("field value"); err != nil {
		t.Fatal(err)
	}

	// There shall be no appearance entry if there is no /Rect to behave as
	// Adobe Acrobat does
	if got := widget.AnnotationDictionary().GetDictionaryObject(cos.AP); got != nil {
		t.Errorf("/AP = %v, want nil", got)
	}
}

// choiceOptions is the options list of PDChoiceTest.setUp.
var choiceOptions = []string{" ", "A", "B"}

// TestCreateListBox is PDChoiceTest.createListBox.
func TestCreateListBox(t *testing.T) {
	document := pdmodel.NewPDDocument()
	acroForm := form.NewPDAcroForm(document)
	choiceField := form.NewPDListBox(acroForm)
	if got, want := choiceField.FieldType(),
		choiceField.FieldDictionary().GetNameAsString(cos.FT, ""); got != want {
		t.Errorf("FieldType() = %q, want %q", got, want)
	}
	if got, want := choiceField.FieldType(), "Ch"; got != want {
		t.Errorf("FieldType() = %q, want %q", got, want)
	}
	if choiceField.IsCombo() {
		t.Error("IsCombo() = true, want false")
	}
}

// TestCreateComboBox is PDChoiceTest.createComboBox.
func TestCreateComboBox(t *testing.T) {
	document := pdmodel.NewPDDocument()
	acroForm := form.NewPDAcroForm(document)
	choiceField := form.NewPDComboBox(acroForm)
	if got, want := choiceField.FieldType(),
		choiceField.FieldDictionary().GetNameAsString(cos.FT, ""); got != want {
		t.Errorf("FieldType() = %q, want %q", got, want)
	}
	if got, want := choiceField.FieldType(), "Ch"; got != want {
		t.Errorf("FieldType() = %q, want %q", got, want)
	}
	if !choiceField.IsCombo() {
		t.Error("IsCombo() = false, want true")
	}
}

// TestGetOptionsFromStrings is PDChoiceTest.getOptionsFromStrings.
func TestGetOptionsFromStrings(t *testing.T) {
	document := pdmodel.NewPDDocument()
	acroForm := form.NewPDAcroForm(document)
	choiceField := form.NewPDComboBox(acroForm)
	choiceFieldOptions := cos.NewArray()
	choiceFieldOptions.Add(cos.NewStringObj(" "))
	choiceFieldOptions.Add(cos.NewStringObj("A"))
	choiceFieldOptions.Add(cos.NewStringObj("B"))

	// add the options using the low level COS model as the PD model will
	// abstract the COSArray
	choiceField.FieldDictionary().SetItem(cos.Opt, choiceFieldOptions)
	if got := choiceField.Options(); !slices.Equal(got, choiceOptions) {
		t.Errorf("Options() = %v, want %v", got, choiceOptions)
	}
}

// TestGetOptionsFromCOSArray is PDChoiceTest.getOptionsFromCOSArray.
func TestGetOptionsFromCOSArray(t *testing.T) {
	document := pdmodel.NewPDDocument()
	acroForm := form.NewPDAcroForm(document)
	choiceField := form.NewPDComboBox(acroForm)
	choiceFieldOptions := cos.NewArray()
	for _, value := range choiceOptions {
		// add entry to options
		entry := cos.NewArray()
		entry.Add(cos.NewStringObj(value))
		choiceFieldOptions.Add(entry)
	}

	// add the options using the low level COS model as the PD model will
	// abstract the COSArray
	choiceField.FieldDictionary().SetItem(cos.Opt, choiceFieldOptions)
	if got := choiceField.Options(); !slices.Equal(got, choiceOptions) {
		t.Errorf("Options() = %v, want %v", got, choiceOptions)
	}
}
