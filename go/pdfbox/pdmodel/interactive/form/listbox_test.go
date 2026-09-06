package form

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/form/TestListBox.java.

import (
	"slices"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// listBoxFixture is the setUp of TestListBox.
type listBoxFixture struct {
	// exportValues are the export values.
	exportValues []string

	// displayValues are the display values, not sorted on purpose as this will
	// be used to test the sort option of the list box.
	displayValues []string

	doc    *pdmodel.PDDocument
	choice *PDListBox
}

// newListBoxFixture is TestListBox.setUp.
func newListBoxFixture(t *testing.T) *listBoxFixture {
	t.Helper()
	f := &listBoxFixture{
		exportValues:  []string{"export01", "export02", "export03"},
		displayValues: []string{"display02", "display01", "display03"},
	}
	f.doc = pdmodel.NewPDDocument()
	t.Cleanup(func() { f.doc.Close() })
	page := pdmodel.NewPDPageOfSize(common.A4)
	f.doc.AddPage(page)
	acroForm := NewPDAcroForm(f.doc)

	// Adobe Acrobat uses Helvetica as a default font and
	// stores that under the name '/Helv' in the resources dictionary
	helvetica, err := font.NewPDType1FontStandard14(font.Helvetica)
	if err != nil {
		t.Fatal(err)
	}
	resources := pdmodel.NewPDResources()
	resources.PutFont(cos.Helv, helvetica)

	// Add and set the resources and default appearance at the form level
	acroForm.SetDefaultResources(resources)

	// Acrobat sets the font size on the form level to be auto sized as default.
	// This is done by setting the font size to '0'
	acroForm.SetDefaultAppearance("/Helv 0 Tf 0 g")

	// the choice field for testing
	f.choice = NewPDListBox(acroForm)
	f.choice.SetDefaultAppearance("/Helv 12 Tf 0g")

	// Specify the annotation associated with the field
	widget := f.choice.Widgets()[0]
	widget.SetRectangle(common.NewPDRectangleOf(50, 750, 200, 50))
	widget.SetPage(page)

	// Add the annotation to the page
	page.Annotations().Add(widget)
	return f
}

// TestNoNullsReturned is TestListBox.testNoNullsReturned.
//
// Java asserts that neither getOptions nor getValue answers null; both answer a
// slice in the port, which is empty rather than nil for an empty field.
func TestNoNullsReturned(t *testing.T) {
	f := newListBoxFixture(t)
	if f.choice.Options() == nil {
		t.Error("Options() = nil, want an empty list")
	}
	if f.choice.Value() == nil {
		t.Error("Value() = nil, want an empty list")
	}
}

// TestExportValuesGetterSetter is TestListBox.testExportValuesGetterSetter.
func TestExportValuesGetterSetter(t *testing.T) {
	f := newListBoxFixture(t)
	// setting/getting option values - the dictionaries Opt entry
	f.choice.SetOptions(f.exportValues)
	if got := f.choice.OptionsDisplayValues(); !slices.Equal(got, f.exportValues) {
		t.Errorf("OptionsDisplayValues() = %v, want %v", got, f.exportValues)
	}
	if got := f.choice.OptionsExportValues(); !slices.Equal(got, f.exportValues) {
		t.Errorf("OptionsExportValues() = %v, want %v", got, f.exportValues)
	}

	// Test bug 1 of PDFBOX-4252 when top index is not null
	topIndex := 1
	f.choice.SetTopIndex(&topIndex)
	if err := f.choice.SetValue(f.exportValues[2]); err != nil {
		t.Fatal(err)
	}
	if got, want := f.choice.Value()[0], f.exportValues[2]; got != want {
		t.Errorf("Value()[0] = %q, want %q", got, want)
	}
	f.choice.SetTopIndex(nil) // reset

	// assert that the option values have been correctly set
	optItem, isArray := f.choice.FieldDictionary().GetItem(cos.Opt).(*cos.Array)
	if !isArray {
		t.Fatalf("/Opt = %T, want *cos.Array", f.choice.FieldDictionary().GetItem(cos.Opt))
	}
	if got, want := optItem.Size(), len(f.exportValues); got != want {
		t.Errorf("/Opt size = %d, want %d", got, want)
	}
	if got, want := optItem.GetString(0, ""), f.exportValues[0]; got != want {
		t.Errorf("/Opt[0] = %q, want %q", got, want)
	}

	// assert that the option values can be retrieved correctly
	retrievedOptions := f.choice.Options()
	if got, want := len(retrievedOptions), len(f.exportValues); got != want {
		t.Errorf("Options() size = %d, want %d", got, want)
	}
	if !slices.Equal(retrievedOptions, f.exportValues) {
		t.Errorf("Options() = %v, want %v", retrievedOptions, f.exportValues)
	}
}

// TestFieldValueSetterGetter is TestListBox.testFieldValueSetterGetter.
func TestFieldValueSetterGetter(t *testing.T) {
	f := newListBoxFixture(t)
	// add test data
	f.choice.SetOptions(f.exportValues)
	f.choice.SetMultiSelect(true)
	if err := f.choice.SetValues(f.exportValues); err != nil {
		t.Fatal(err)
	}

	// assert that the option values have been correctly set
	valueItems, isArray := f.choice.FieldDictionary().GetItem(cos.V).(*cos.Array)
	if !isArray {
		t.Fatalf("/V = %T, want *cos.Array", f.choice.FieldDictionary().GetItem(cos.V))
	}
	if got, want := valueItems.Size(), len(f.exportValues); got != want {
		t.Errorf("/V size = %d, want %d", got, want)
	}
	if got, want := valueItems.GetString(0, ""), f.exportValues[0]; got != want {
		t.Errorf("/V[0] = %q, want %q", got, want)
	}

	// assert that the index values have been correctly set
	indexItems, isArray := f.choice.FieldDictionary().GetItem(cos.I).(*cos.Array)
	if !isArray {
		t.Fatalf("/I = %T, want *cos.Array", f.choice.FieldDictionary().GetItem(cos.I))
	}
	if got, want := indexItems.Size(), len(f.exportValues); got != want {
		t.Errorf("/I size = %d, want %d", got, want)
	}

	// setting a single value shall remove the indices
	if err := f.choice.SetValue("export01"); err != nil {
		t.Fatal(err)
	}
	if got := f.choice.FieldDictionary().GetItem(cos.I); got != nil {
		t.Errorf("/I = %v, want nil", got)
	}
}

// TestMultiselect is TestListBox.testMultiselect. Java asserts
// IllegalArgumentException with a message, which is unchecked, so the port
// panics with the same one.
func TestMultiselect(t *testing.T) {
	f := newListBoxFixture(t)
	// add test data
	f.choice.SetOptions(f.exportValues)

	// ensure that the choice field doesn't allow multiple selections
	f.choice.SetMultiSelect(false)

	// without multiselect setting multiple items shall fail
	func() {
		defer func() {
			recovered := recover()
			if recovered == nil {
				t.Fatal("SetValues() did not panic")
			}
			if got, want := recovered, "The list box does not allow multiple selections."; got != want {
				t.Errorf("panic = %v, want %q", got, want)
			}
		}()
		f.choice.SetValues(f.exportValues) //nolint:errcheck // it panics
	}()

	// ensure that the choice field does allow multiple selections
	f.choice.SetMultiSelect(true)

	// now this call must succeed
	if err := f.choice.SetValues(f.exportValues); err != nil {
		t.Fatal(err)
	}
}

// TestOptIsRemovedForNull is TestListBox.testOptIsRemovedForNull.
func TestOptIsRemovedForNull(t *testing.T) {
	f := newListBoxFixture(t)
	// add test data
	f.choice.SetOptions(f.exportValues)
	if got := f.choice.FieldDictionary().GetItem(cos.Opt); got == nil {
		t.Error("/Opt = nil, want an array")
	}

	// assert that the Opt entry is removed
	f.choice.SetOptions(nil)
	if got := f.choice.FieldDictionary().GetItem(cos.Opt); got != nil {
		t.Errorf("/Opt = %v, want nil", got)
	}

	// if there is no Opt entry an empty List shall be returned
	if got := f.choice.Options(); len(got) != 0 {
		t.Errorf("Options() = %v, want empty", got)
	}
}

// TestSetExportAndDisplay is TestListBox.testSetExportAndDisplay.
func TestSetExportAndDisplay(t *testing.T) {
	f := newListBoxFixture(t)
	// setting display and export value
	f.choice.SetOptionsWithDisplayValues(f.exportValues, f.displayValues)
	if got := f.choice.OptionsDisplayValues(); !slices.Equal(got, f.displayValues) {
		t.Errorf("OptionsDisplayValues() = %v, want %v", got, f.displayValues)
	}
	if got := f.choice.OptionsExportValues(); !slices.Equal(got, f.exportValues) {
		t.Errorf("OptionsExportValues() = %v, want %v", got, f.exportValues)
	}
}

// TestSortOption is TestListBox.testSortOption.
func TestSortOption(t *testing.T) {
	f := newListBoxFixture(t)
	// add test data
	f.choice.SetOptionsWithDisplayValues(f.exportValues, f.displayValues)
	if got, want := f.choice.OptionsDisplayValues()[0], "display02"; got != want {
		t.Errorf("OptionsDisplayValues()[0] = %q, want %q", got, want)
	}

	// test the sort option
	f.choice.SetSort(true)
	f.choice.SetOptionsWithDisplayValues(f.exportValues, f.displayValues)
	for i, want := range []string{"display01", "display02", "display03"} {
		if got := f.choice.OptionsDisplayValues()[i]; got != want {
			t.Errorf("OptionsDisplayValues()[%d] = %q, want %q", i, got, want)
		}
	}
}

// TestEmptyOptionsNotNull is TestListBox.testEmptyOptionsNotNull.
func TestEmptyOptionsNotNull(t *testing.T) {
	f := newListBoxFixture(t)
	// assert that the Opt entry is removed
	f.choice.SetOptionsWithDisplayValues(nil, f.displayValues)
	if got := f.choice.FieldDictionary().GetItem(cos.Opt); got != nil {
		t.Errorf("/Opt = %v, want nil", got)
	}

	// if there is no Opt entry an empty list shall be returned
	if got := f.choice.Options(); len(got) != 0 {
		t.Errorf("Options() = %v, want empty", got)
	}
	if got := f.choice.OptionsDisplayValues(); len(got) != 0 {
		t.Errorf("OptionsDisplayValues() = %v, want empty", got)
	}
	if got := f.choice.OptionsExportValues(); len(got) != 0 {
		t.Errorf("OptionsExportValues() = %v, want empty", got)
	}
}

// TestExceptionForDifferentNumberOfEntries is
// TestListBox.testExceptionForDifferentNumberOfEntries: an
// IllegalArgumentException is thrown when export and display value lists have
// different sizes.
func TestExceptionForDifferentNumberOfEntries(t *testing.T) {
	f := newListBoxFixture(t)
	exportValues := slices.Delete(slices.Clone(f.exportValues), 1, 2)
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("SetOptionsWithDisplayValues() did not panic")
		}
		want := "The number of entries for exportValue and displayValue shall be the same."
		if got := recovered; got != want {
			t.Errorf("panic = %v, want %q", got, want)
		}
	}()
	f.choice.SetOptionsWithDisplayValues(exportValues, f.displayValues)
}
