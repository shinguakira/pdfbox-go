package form_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/form/PDButtonTest.java.
//
// testRadioButtonWithOptions and testOptionsAndNamesNotNumbers are not here:
// both read a PDF out of target/pdfs, which the Maven build downloads from the
// issue tracker, and the port runs no test that reaches the network. See
// migration/STATUS.md.

import (
	"slices"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// acrobatForm is the acrobatAcroForm of PDButtonTest.setUp: the form of
// AcroFormsBasicFields.pdf, which Acrobat wrote.
func acrobatForm(t *testing.T) *form.PDAcroForm {
	t.Helper()
	acrobatDocument, err := pdfbox.LoadPDF(formFixture + "AcroFormsBasicFields.pdf")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { acrobatDocument.Close() })
	return form.AcroFormOfCatalog(acrobatDocument.DocumentCatalog())
}

// TestCreateCheckBox is PDButtonTest.createCheckBox.
func TestCreateCheckBox(t *testing.T) {
	acroForm := form.NewPDAcroForm(pdmodel.NewPDDocument())
	buttonField := form.NewPDCheckBox(acroForm)
	assertButtonType(t, buttonField, false, false)
}

// TestCreatePushButton is PDButtonTest.createPushButton.
func TestCreatePushButton(t *testing.T) {
	acroForm := form.NewPDAcroForm(pdmodel.NewPDDocument())
	buttonField := form.NewPDPushButton(acroForm)
	assertButtonType(t, buttonField, true, false)
}

// TestCreateRadioButton is PDButtonTest.createRadioButton.
func TestCreateRadioButton(t *testing.T) {
	acroForm := form.NewPDAcroForm(pdmodel.NewPDDocument())
	buttonField := form.NewPDRadioButton(acroForm)
	assertButtonType(t, buttonField, false, true)
}

// buttonField is the half of PDButton the three creation tests use.
type buttonField interface {
	form.PDField
	IsPushButton() bool
	IsRadioButton() bool
}

// assertButtonType is the body the three creation tests share.
func assertButtonType(t *testing.T, field buttonField, isPush, isRadio bool) {
	t.Helper()
	if got, want := field.FieldType(),
		field.FieldDictionary().GetNameAsString(cos.FT, ""); got != want {
		t.Errorf("FieldType() = %q, want %q", got, want)
	}
	if got, want := field.FieldType(), "Btn"; got != want {
		t.Errorf("FieldType() = %q, want %q", got, want)
	}
	if got := field.IsPushButton(); got != isPush {
		t.Errorf("IsPushButton() = %v, want %v", got, isPush)
	}
	if got := field.IsRadioButton(); got != isRadio {
		t.Errorf("IsRadioButton() = %v, want %v", got, isRadio)
	}
}

// TestRetrieveAcrobatCheckBoxProperties is
// PDButtonTest.retrieveAcrobatCheckBoxProperties.
func TestRetrieveAcrobatCheckBoxProperties(t *testing.T) {
	checkbox := acrobatForm(t).Field("Checkbox").(*form.PDCheckBox)
	if checkbox == nil {
		t.Fatal("Field(\"Checkbox\") = nil, want a field")
	}
	if got, want := checkbox.OnValue(), "Yes"; got != want {
		t.Errorf("OnValue() = %q, want %q", got, want)
	}
	if got := len(checkbox.OnValues()); got != 1 {
		t.Errorf("OnValues() size = %d, want 1", got)
	}
	if !slices.Contains(checkbox.OnValues(), "Yes") {
		t.Errorf("OnValues() = %v, want to contain %q", checkbox.OnValues(), "Yes")
	}
}

// TestAcrobatCheckBoxProperties is PDButtonTest.testAcrobatCheckBoxProperties.
func TestAcrobatCheckBoxProperties(t *testing.T) {
	acroForm := acrobatForm(t)
	checkbox := acroForm.Field("Checkbox").(*form.PDCheckBox)
	if got, want := checkbox.Value(), "Off"; got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if checkbox.IsChecked() {
		t.Error("IsChecked() = true, want false")
	}

	if err := checkbox.Check(); err != nil {
		t.Fatal(err)
	}
	if got, want := checkbox.Value(), checkbox.OnValue(); got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if !checkbox.IsChecked() {
		t.Error("IsChecked() = false, want true")
	}

	if err := checkbox.SetValue("Yes"); err != nil {
		t.Fatal(err)
	}
	if got, want := checkbox.Value(), checkbox.OnValue(); got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if !checkbox.IsChecked() {
		t.Error("IsChecked() = false, want true")
	}
	if got := checkbox.FieldDictionary().GetDictionaryObject(cos.AS); got != cos.Base(cos.Yes) {
		t.Errorf("/AS = %v, want %v", got, cos.Yes)
	}

	if err := checkbox.SetValue("Off"); err != nil {
		t.Fatal(err)
	}
	if got, want := checkbox.Value(), cos.Off.Name(); got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if checkbox.IsChecked() {
		t.Error("IsChecked() = true, want false")
	}
	if got := checkbox.FieldDictionary().GetDictionaryObject(cos.AS); got != cos.Base(cos.Off) {
		t.Errorf("/AS = %v, want %v", got, cos.Off)
	}

	checkbox = acroForm.Field("Checkbox-DefaultValue").(*form.PDCheckBox)
	if got, want := checkbox.DefaultValue(), checkbox.OnValue(); got != want {
		t.Errorf("DefaultValue() = %q, want %q", got, want)
	}
	checkbox.SetDefaultValue("Off")
	if got, want := checkbox.DefaultValue(), cos.Off.Name(); got != want {
		t.Errorf("DefaultValue() = %q, want %q", got, want)
	}
}

// TestSetValueForAbstractedAcrobatCheckBox is
// PDButtonTest.setValueForAbstractedAcrobatCheckBox.
func TestSetValueForAbstractedAcrobatCheckBox(t *testing.T) {
	checkbox := acrobatForm(t).Field("Checkbox")
	if err := checkbox.SetValue("Yes"); err != nil {
		t.Fatal(err)
	}
	if got, want := checkbox.ValueAsString(),
		checkbox.(*form.PDCheckBox).OnValue(); got != want {
		t.Errorf("ValueAsString() = %q, want %q", got, want)
	}
	if !checkbox.(*form.PDCheckBox).IsChecked() {
		t.Error("IsChecked() = false, want true")
	}
	if got := checkbox.FieldDictionary().GetDictionaryObject(cos.AS); got != cos.Base(cos.Yes) {
		t.Errorf("/AS = %v, want %v", got, cos.Yes)
	}

	if err := checkbox.SetValue("Off"); err != nil {
		t.Fatal(err)
	}
	if got, want := checkbox.ValueAsString(), cos.Off.Name(); got != want {
		t.Errorf("ValueAsString() = %q, want %q", got, want)
	}
	if checkbox.(*form.PDCheckBox).IsChecked() {
		t.Error("IsChecked() = true, want false")
	}
	if got := checkbox.FieldDictionary().GetDictionaryObject(cos.AS); got != cos.Base(cos.Off) {
		t.Errorf("/AS = %v, want %v", got, cos.Off)
	}
}

// assertAppearanceStates checks the /AS of the widgets of the field.
func assertAppearanceStates(t *testing.T, field form.PDField, want ...string) {
	t.Helper()
	widgets := field.Widgets()
	for i, wantState := range want {
		if got := widgets[i].AppearanceState().Name(); got != wantState {
			t.Errorf("widget %d /AS = %q, want %q", i, got, wantState)
		}
	}
}

// TestAcrobatCheckBoxGroupProperties is
// PDButtonTest.testAcrobatCheckBoxGroupProperties.
func TestAcrobatCheckBoxGroupProperties(t *testing.T) {
	checkbox := acrobatForm(t).Field("CheckboxGroup").(*form.PDCheckBox)
	if got, want := checkbox.Value(), "Off"; got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if checkbox.IsChecked() {
		t.Error("IsChecked() = true, want false")
	}
	if err := checkbox.Check(); err != nil {
		t.Fatal(err)
	}
	if got, want := checkbox.Value(), checkbox.OnValue(); got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if !checkbox.IsChecked() {
		t.Error("IsChecked() = false, want true")
	}
	if got := len(checkbox.OnValues()); got != 3 {
		t.Errorf("OnValues() size = %d, want 3", got)
	}
	for _, want := range []string{"Option1", "Option2", "Option3"} {
		if !slices.Contains(checkbox.OnValues(), want) {
			t.Errorf("OnValues() = %v, want to contain %q", checkbox.OnValues(), want)
		}
	}

	// test a value which sets one of the individual checkboxes within the group
	if err := checkbox.SetValue("Option1"); err != nil {
		t.Fatal(err)
	}
	if got, want := checkbox.Value(), "Option1"; got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if got, want := checkbox.ValueAsString(), "Option1"; got != want {
		t.Errorf("ValueAsString() = %q, want %q", got, want)
	}
	// ensure that for the widgets representing the individual checkboxes the AS
	// entry has been set
	assertAppearanceStates(t, checkbox, "Option1", "Off", "Off", "Off")

	// test a value which sets two of the individual chekboxes within the group
	// as the have the same name entry for being checked
	if err := checkbox.SetValue("Option3"); err != nil {
		t.Fatal(err)
	}
	if got, want := checkbox.Value(), "Option3"; got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	if got, want := checkbox.ValueAsString(), "Option3"; got != want {
		t.Errorf("ValueAsString() = %q, want %q", got, want)
	}
	assertAppearanceStates(t, checkbox, "Off", "Off", "Option3", "Option3")
}

// TestSetValueForAbstractedCheckBoxGroup is
// PDButtonTest.setValueForAbstractedCheckBoxGroup.
func TestSetValueForAbstractedCheckBoxGroup(t *testing.T) {
	checkbox := acrobatForm(t).Field("CheckboxGroup")

	// test a value which sets one of the individual checkboxes within the group
	if err := checkbox.SetValue("Option1"); err != nil {
		t.Fatal(err)
	}
	if got, want := checkbox.ValueAsString(), "Option1"; got != want {
		t.Errorf("ValueAsString() = %q, want %q", got, want)
	}
	assertAppearanceStates(t, checkbox, "Option1", "Off", "Off", "Off")

	// test a value which sets two of the individual chekboxes within the group
	// as the have the same name entry for being checked
	if err := checkbox.SetValue("Option3"); err != nil {
		t.Fatal(err)
	}
	if got, want := checkbox.ValueAsString(), "Option3"; got != want {
		t.Errorf("ValueAsString() = %q, want %q", got, want)
	}
	assertAppearanceStates(t, checkbox, "Off", "Off", "Option3", "Option3")
}

// assertInvalidValuePanics is the assertThrows of the four invalid value tests.
func assertInvalidValuePanics(t *testing.T, field form.PDField) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Error("SetValue() did not panic")
		}
	}()
	field.SetValue("InvalidValue") //nolint:errcheck // it panics
}

// TestSetCheckboxInvalidValue is PDButtonTest.setCheckboxInvalidValue. Java
// asserts IllegalArgumentException, which is unchecked, so the port panics.
func TestSetCheckboxInvalidValue(t *testing.T) {
	assertInvalidValuePanics(t, acrobatForm(t).Field("Checkbox"))
}

// TestSetCheckboxGroupInvalidValue is PDButtonTest.setCheckboxGroupInvalidValue.
func TestSetCheckboxGroupInvalidValue(t *testing.T) {
	assertInvalidValuePanics(t, acrobatForm(t).Field("CheckboxGroup"))
}

// TestSetAbstractedCheckboxInvalidValue is
// PDButtonTest.setAbstractedCheckboxInvalidValue.
func TestSetAbstractedCheckboxInvalidValue(t *testing.T) {
	assertInvalidValuePanics(t, acrobatForm(t).Field("Checkbox"))
}

// TestSetAbstractedCheckboxGroupInvalidValue is
// PDButtonTest.setAbstractedCheckboxGroupInvalidValue.
func TestSetAbstractedCheckboxGroupInvalidValue(t *testing.T) {
	assertInvalidValuePanics(t, acrobatForm(t).Field("CheckboxGroup"))
}

// TestRetrieveAcrobatRadioButtonProperties is
// PDButtonTest.retrieveAcrobatRadioButtonProperties.
func TestRetrieveAcrobatRadioButtonProperties(t *testing.T) {
	radioButton := acrobatForm(t).Field("RadioButtonGroup").(*form.PDRadioButton)
	if radioButton == nil {
		t.Fatal("Field(\"RadioButtonGroup\") = nil, want a field")
	}
	if got := len(radioButton.OnValues()); got != 2 {
		t.Errorf("OnValues() size = %d, want 2", got)
	}
	for _, want := range []string{"RadioButton01", "RadioButton02"} {
		if !slices.Contains(radioButton.OnValues(), want) {
			t.Errorf("OnValues() = %v, want to contain %q", radioButton.OnValues(), want)
		}
	}
}

// assertRadioAppearanceStates checks the /AS of the two radio widgets.
func assertRadioAppearanceStates(t *testing.T, field form.PDField, first, second *cos.Name) {
	t.Helper()
	widgets := field.Widgets()
	if got := widgets[0].AnnotationDictionary().GetDictionaryObject(cos.AS); got !=
		cos.Base(first) {
		t.Errorf("widget 0 /AS = %v, want %v", got, first)
	}
	if got := widgets[1].AnnotationDictionary().GetDictionaryObject(cos.AS); got !=
		cos.Base(second) {
		t.Errorf("widget 1 /AS = %v, want %v", got, second)
	}
}

// TestAcrobatRadioButtonProperties is
// PDButtonTest.testAcrobatRadioButtonProperties.
func TestAcrobatRadioButtonProperties(t *testing.T) {
	radioButton := acrobatForm(t).Field("RadioButtonGroup").(*form.PDRadioButton)

	// Set value so that first radio button option is selected
	if err := radioButton.SetValue("RadioButton01"); err != nil {
		t.Fatal(err)
	}
	if got, want := radioButton.Value(), "RadioButton01"; got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	// First option shall have /RadioButton01, second shall have /Off
	assertRadioAppearanceStates(t, radioButton, cos.GetPDFName("RadioButton01"), cos.Off)

	// Set value so that second radio button option is selected
	if err := radioButton.SetValue("RadioButton02"); err != nil {
		t.Fatal(err)
	}
	if got, want := radioButton.Value(), "RadioButton02"; got != want {
		t.Errorf("Value() = %q, want %q", got, want)
	}
	// First option shall have /Off, second shall have /RadioButton02
	assertRadioAppearanceStates(t, radioButton, cos.Off, cos.GetPDFName("RadioButton02"))
}

// TestSetValueForAbstractedAcrobatRadioButton is
// PDButtonTest.setValueForAbstractedAcrobatRadioButton.
func TestSetValueForAbstractedAcrobatRadioButton(t *testing.T) {
	radioButton := acrobatForm(t).Field("RadioButtonGroup")

	// Set value so that first radio button option is selected
	if err := radioButton.SetValue("RadioButton01"); err != nil {
		t.Fatal(err)
	}
	if got, want := radioButton.ValueAsString(), "RadioButton01"; got != want {
		t.Errorf("ValueAsString() = %q, want %q", got, want)
	}
	assertRadioAppearanceStates(t, radioButton, cos.GetPDFName("RadioButton01"), cos.Off)

	// Set value so that second radio button option is selected
	if err := radioButton.SetValue("RadioButton02"); err != nil {
		t.Fatal(err)
	}
	if got, want := radioButton.ValueAsString(), "RadioButton02"; got != want {
		t.Errorf("ValueAsString() = %q, want %q", got, want)
	}
	assertRadioAppearanceStates(t, radioButton, cos.Off, cos.GetPDFName("RadioButton02"))
}

// TestSetRadioButtonInvalidValue is PDButtonTest.setRadioButtonInvalidValue.
func TestSetRadioButtonInvalidValue(t *testing.T) {
	assertInvalidValuePanics(t, acrobatForm(t).Field("RadioButtonGroup"))
}

// TestSetAbstractedRadioButtonInvalidValue is
// PDButtonTest.setAbstractedRadioButtonInvalidValue.
func TestSetAbstractedRadioButtonInvalidValue(t *testing.T) {
	assertInvalidValuePanics(t, acrobatForm(t).Field("RadioButtonGroup"))
}
