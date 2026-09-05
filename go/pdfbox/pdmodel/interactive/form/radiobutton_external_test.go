package form_test

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/form/TestRadioButtons.java.
//
// testPDFBox5831NumericValueForOpt and testPDFBox6178NonAsciiRadioButtonValue
// are not here: the first downloads a PDF from the issue tracker, and the second
// reads one out of target/pdfs, which the Maven build downloads. See
// migration/STATUS.md.

import (
	"slices"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/annotation"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// testFile3656 is the TESTFILE3656 of TestRadioButtons.
const testFile3656 = formFixture + "PDFBOX-3656-SF1199AEG (Complete).pdf"

// checkingSavings opens PDFBOX-3656 and answers its Checking/Savings field.
func checkingSavings(t *testing.T) *form.PDRadioButton {
	t.Helper()
	testPdf, err := pdfbox.LoadPDF(testFile3656)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { testPdf.Close() })
	acroForm := form.AcroFormOfCatalog(testPdf.DocumentCatalog())
	return acroForm.Field("Checking/Savings").(*form.PDRadioButton)
}

// assertNotInUnison is the assertFalse(isRadiosInUnison) the PDFBOX-3656 tests
// open with: the radio buttons can be selected individually although having the
// same ON value.
func assertNotInUnison(t *testing.T, field *form.PDRadioButton) {
	t.Helper()
	if field.IsRadiosInUnison() {
		t.Error("the radio buttons can be selected individually although " +
			"having the same ON value")
	}
}

// TestRadioButtonPDModel is TestRadioButtons.testRadioButtonPDModel.
func TestRadioButtonPDModel(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	acroForm := form.NewPDAcroForm(doc)
	radioButton := form.NewPDRadioButton(acroForm)

	// test that there are no nulls returned for an empty field
	// only specific methods are tested here. Java asserts each is not null; the
	// port answers a string or a slice, and a slice is empty rather than nil.
	if radioButton.SelectedExportValues() == nil {
		t.Error("SelectedExportValues() = nil, want an empty list")
	}
	if radioButton.ExportValues() == nil {
		t.Error("ExportValues() = nil, want an empty list")
	}

	// Test setting/getting option values - the dictionaries Opt entry
	options := []string{"Value01", "Value02"}
	radioButton.SetExportValues(options)

	// Test getSelectedExportValues()
	widgets := []*annotation.PDAnnotationWidget{}
	for i := range options {
		widget := annotation.NewPDAnnotationWidget()
		apNDict := cos.NewDictionary()
		apNDict.SetItem(cos.Off, annotation.NewPDAppearanceStream(doc.Document()).COSObject())
		apNDict.SetItem(cos.GetPDFName(options[i]),
			annotation.NewPDAppearanceStream(doc.Document()).COSObject())
		appearance := annotation.NewPDAppearanceDictionary()
		appearance.SetNormalAppearance(annotation.NewPDAppearanceEntryOf(apNDict))
		widget.SetAppearance(appearance)
		widget.SetAppearanceState("Off")
		widgets = append(widgets, widget)
	}
	radioButton.SetWidgets(widgets)

	for _, c := range []struct {
		value          string
		selectedValues []string
		states         []string
	}{
		{"Value01", []string{"Value01"}, []string{"Value01", "Off"}},
		{"Value02", []string{"Value02"}, []string{"Off", "Value02"}},
		{"Off", []string{}, []string{"Off", "Off"}},
	} {
		if err := radioButton.SetValue(c.value); err != nil {
			t.Fatal(err)
		}
		if got := radioButton.Value(); got != c.value {
			t.Errorf("Value() = %q, want %q", got, c.value)
		}
		if got := radioButton.SelectedExportValues(); !slices.Equal(got, c.selectedValues) {
			t.Errorf("SelectedExportValues() = %v, want %v", got, c.selectedValues)
		}
		for i, want := range c.states {
			if got := widgets[i].AppearanceState().Name(); got != want {
				t.Errorf("widget %d /AS = %q, want %q", i, got, want)
			}
		}
	}

	optItem, isArray := radioButton.FieldDictionary().GetItem(cos.Opt).(*cos.Array)
	// assert that the values have been correctly set
	if !isArray {
		t.Fatalf("/Opt = %T, want *cos.Array", radioButton.FieldDictionary().GetItem(cos.Opt))
	}
	if got := optItem.Size(); got != 2 {
		t.Errorf("/Opt size = %d, want 2", got)
	}
	if got, want := optItem.GetString(0, ""), options[0]; got != want {
		t.Errorf("/Opt[0] = %q, want %q", got, want)
	}

	// assert that the values can be retrieved correctly
	retrievedOptions := radioButton.ExportValues()
	if got := len(retrievedOptions); got != 2 {
		t.Errorf("ExportValues() size = %d, want 2", got)
	}
	if !slices.Equal(retrievedOptions, options) {
		t.Errorf("ExportValues() = %v, want %v", retrievedOptions, options)
	}

	// assert that the Opt entry is removed
	radioButton.SetExportValues(nil)
	if got := radioButton.FieldDictionary().GetItem(cos.Opt); got != nil {
		t.Errorf("/Opt = %v, want nil", got)
	}

	// if there is no Opt entry an empty List shall be returned
	if got := radioButton.ExportValues(); len(got) != 0 {
		t.Errorf("ExportValues() = %v, want empty", got)
	}
}

// TestNoToggleToOff is TestRadioButtons.testNoToggleToOff.
func TestNoToggleToOff(t *testing.T) {
	doc := pdmodel.NewPDDocument()
	defer doc.Close()
	acroForm := form.NewPDAcroForm(doc)
	radioButton := form.NewPDRadioButton(acroForm)

	// default shall be false/unset
	if radioButton.IsNoToggleToOff() {
		t.Error("IsNoToggleToOff() = true, want false")
	}
	radioButton.SetNoToggleToOff(true)
	if !radioButton.IsNoToggleToOff() {
		t.Error("IsNoToggleToOff() = false, want true")
	}
	// spec bit 15 -> mask 1 << 14
	if got := radioButton.FieldDictionary().GetInt(cos.Ff) & (1 << 14); got != 1<<14 {
		t.Errorf("/Ff & 1<<14 = %d, want %d", got, 1<<14)
	}
	radioButton.SetNoToggleToOff(false)
	if radioButton.IsNoToggleToOff() {
		t.Error("IsNoToggleToOff() = true, want false")
	}
	if got := radioButton.FieldDictionary().GetInt(cos.Ff) & (1 << 14); got != 0 {
		t.Errorf("/Ff & 1<<14 = %d, want 0", got)
	}
}

// TestPDFBox3656NotInUnison is TestRadioButtons.testPDFBox3656NotInUnison.
func TestPDFBox3656NotInUnison(t *testing.T) {
	assertNotInUnison(t, checkingSavings(t))
}

// TestPDFBox3656ByValidExportValue is
// TestRadioButtons.testPDFBox3656ByValidExportValue.
func TestPDFBox3656ByValidExportValue(t *testing.T) {
	field := checkingSavings(t)
	// check defaults
	assertNotInUnison(t, field)
	if got := field.Value(); got != "Off" {
		t.Errorf("initially no option shall be selected: %q", got)
	}
	// set the field to a valid export value
	if err := field.SetValue("Checking"); err != nil {
		t.Fatal(err)
	}
	if got := field.Value(); got != "Checking" {
		t.Errorf("setting by the export value should also return that: %q", got)
	}
}

// TestPDFBox3656ByInvalidExportValue is
// TestRadioButtons.testPDFBox3656ByInvalidExportValue. Java asserts
// IllegalArgumentException with a message, which is unchecked, so the port
// panics with the same one.
func TestPDFBox3656ByInvalidExportValue(t *testing.T) {
	field := checkingSavings(t)
	// check defaults
	assertNotInUnison(t, field)
	if got := field.Value(); got != "Off" {
		t.Errorf("initially no option shall be selected: %q", got)
	}
	// set the field to an invalid value shall throw
	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatal("SetValue() did not panic")
			}
			// compare the messages
			expectedMessage := "value 'Invalid' is not a valid option for the field " +
				"Checking/Savings, valid values are: [Checking Savings] and Off"
			actualMessage, isString := recovered.(string)
			if !isString || !strings.Contains(actualMessage, expectedMessage) {
				t.Errorf("panic = %v, want it to contain %q", recovered, expectedMessage)
			}
		}()
		field.SetValue("Invalid") //nolint:errcheck // it panics
	}()
	if got := field.Value(); got != "Off" {
		t.Errorf("no option shall be selected: %q", got)
	}
	if got := field.SelectedExportValues(); len(got) != 0 {
		t.Errorf("no export values are selected: %v", got)
	}
}

// TestPDFBox3656ByValidIndex is TestRadioButtons.testPDFBox3656ByValidIndex.
func TestPDFBox3656ByValidIndex(t *testing.T) {
	field := checkingSavings(t)
	// check defaults
	assertNotInUnison(t, field)
	if got := field.Value(); got != "Off" {
		t.Errorf("initially no option shall be selected: %q", got)
	}
	// set the field to a valid index
	if err := field.SetValueIndex(4); err != nil {
		t.Fatal(err)
	}
	if got := field.Value(); got != "Checking" {
		t.Errorf("setting by the index value should return the corresponding export: %q", got)
	}
}

// TestPDFBox3656ByInvalidIndex is TestRadioButtons.testPDFBox3656ByInvalidIndex.
func TestPDFBox3656ByInvalidIndex(t *testing.T) {
	field := checkingSavings(t)
	// check defaults
	assertNotInUnison(t, field)
	if got := field.Value(); got != "Off" {
		t.Errorf("initially no option shall be selected: %q", got)
	}
	// set the field to an invalid index shall throw
	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatal("SetValueIndex() did not panic")
			}
			// compare the messages
			expectedMessage := "index '6' is not a valid index for the field " +
				"Checking/Savings, valid indices are from 0 to 5"
			actualMessage, isString := recovered.(string)
			if !isString || !strings.Contains(actualMessage, expectedMessage) {
				t.Errorf("panic = %v, want it to contain %q", recovered, expectedMessage)
			}
		}()
		field.SetValueIndex(6) //nolint:errcheck // it panics
	}()
	if got := field.Value(); got != "Off" {
		t.Errorf("no option shall be selected: %q", got)
	}
	if got := field.SelectedExportValues(); len(got) != 0 {
		t.Errorf("no export values are selected: %v", got)
	}
}

// TestPDFBox4617IndexNoneSelected is
// TestRadioButtons.testPDFBox4617IndexNoneSelected.
func TestPDFBox4617IndexNoneSelected(t *testing.T) {
	field := checkingSavings(t)
	if got := field.SelectedIndex(); got != -1 {
		t.Errorf("if there is no value set the index shall be -1: %d", got)
	}
}

// TestPDFBox4617IndexForSetByOption is
// TestRadioButtons.testPDFBox4617IndexForSetByOption.
func TestPDFBox4617IndexForSetByOption(t *testing.T) {
	field := checkingSavings(t)
	if err := field.SetValue("Checking"); err != nil {
		t.Fatal(err)
	}
	if got := field.SelectedIndex(); got != 0 {
		t.Errorf("the index shall be equal with the first entry of Checking "+
			"which is 0: %d", got)
	}
}

// TestPDFBox4617IndexForSetByIndex is
// TestRadioButtons.testPDFBox4617IndexForSetByIndex.
func TestPDFBox4617IndexForSetByIndex(t *testing.T) {
	field := checkingSavings(t)
	if err := field.SetValueIndex(4); err != nil {
		t.Fatal(err)
	}
	if got := field.Value(); got != "Checking" {
		t.Errorf("setting by the index value should return the corresponding export: %q", got)
	}
	if got := field.SelectedIndex(); got != 4 {
		t.Errorf("the index shall be equals with the set value of 4: %d", got)
	}
}
