package form

// Ported from
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/interactive/form/PDFieldTest.java.
//
// testSetPartialNameNull is not here: Java has its @Test commented out, and its
// own comment says the behaviour is undecided.

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
)

// newFieldFixture is PDFieldTest.setUp.
func newFieldFixture(t *testing.T) (*PDAcroForm, *PDTextField) {
	t.Helper()
	document := pdmodel.NewPDDocument()
	t.Cleanup(func() { document.Close() })
	acroForm := NewPDAcroForm(document)
	return acroForm, NewPDTextField(acroForm)
}

// TestPartialName is PDFieldTest.testPartialName. Java asserts the default
// partial name is null; the port answers the empty string.
func TestPartialName(t *testing.T) {
	_, textField := newFieldFixture(t)
	if got := textField.PartialName(); got != "" {
		t.Errorf("PartialName() = %q, want empty", got)
	}
	for _, testName := range []string{"testField", "anotherField"} {
		textField.SetPartialName(testName)
		if got := textField.PartialName(); got != testName {
			t.Errorf("PartialName() = %q, want %q", got, testName)
		}
	}
}

// TestPartialNameWithPeriodThrows is
// PDFieldTest.testPartialNameWithPeriodThrows. Java asserts
// IllegalArgumentException whose message names the period, which is unchecked,
// so the port panics.
func TestPartialNameWithPeriodThrows(t *testing.T) {
	_, textField := newFieldFixture(t)
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatal("SetPartialName() did not panic")
		}
		message, isString := recovered.(string)
		if !isString || !strings.Contains(message, "period character") {
			t.Errorf("panic = %v, want it to name the period character", recovered)
		}
	}()
	textField.SetPartialName("test.field")
}

// TestFullyQualifiedName is PDFieldTest.testFullyQualifiedName.
func TestFullyQualifiedName(t *testing.T) {
	_, textField := newFieldFixture(t)
	textField.SetPartialName("childField")
	if got, want := textField.FullyQualifiedName(), "childField"; got != want {
		t.Errorf("FullyQualifiedName() = %q, want %q", got, want)
	}
}

// TestFullyQualifiedNameNullPartialName is
// PDFieldTest.testFullyQualifiedNameNullPartialName. Java asserts null; the
// port answers the empty string.
func TestFullyQualifiedNameNullPartialName(t *testing.T) {
	_, textField := newFieldFixture(t)
	if got := textField.FullyQualifiedName(); got != "" {
		t.Errorf("FullyQualifiedName() = %q, want empty", got)
	}
}

// TestFullyQualifiedNameWithParent is
// PDFieldTest.testFullyQualifiedNameWithParent.
func TestFullyQualifiedNameWithParent(t *testing.T) {
	acroForm, _ := newFieldFixture(t)
	parentField := NewPDNonTerminalField(acroForm)
	parentField.SetPartialName("parentField")
	childField := NewPDTextFieldOf(acroForm, cos.NewDictionary(), parentField)
	childField.SetPartialName("childField")
	if got, want := childField.FullyQualifiedName(), "parentField.childField"; got != want {
		t.Errorf("FullyQualifiedName() = %q, want %q", got, want)
	}
}

// TestAlternateFieldName is PDFieldTest.testAlternateFieldName.
func TestAlternateFieldName(t *testing.T) {
	_, textField := newFieldFixture(t)
	if got := textField.AlternateFieldName(); got != "" {
		t.Errorf("AlternateFieldName() = %q, want empty", got)
	}
	for _, alternateName := range []string{"Alternate Name For Field", "New Alternate Name"} {
		textField.SetAlternateFieldName(alternateName)
		if got := textField.AlternateFieldName(); got != alternateName {
			t.Errorf("AlternateFieldName() = %q, want %q", got, alternateName)
		}
	}
}

// TestMappingName is PDFieldTest.testMappingName.
func TestMappingName(t *testing.T) {
	_, textField := newFieldFixture(t)
	if got := textField.MappingName(); got != "" {
		t.Errorf("MappingName() = %q, want empty", got)
	}
	for _, mappingName := range []string{"mappingName", "newMappingName"} {
		textField.SetMappingName(mappingName)
		if got := textField.MappingName(); got != mappingName {
			t.Errorf("MappingName() = %q, want %q", got, mappingName)
		}
	}
}

// TestReadOnlyFlag is PDFieldTest.testReadOnlyFlag.
func TestReadOnlyFlag(t *testing.T) {
	_, textField := newFieldFixture(t)
	assertFlagRoundTrip(t, "IsReadOnly", textField.IsReadOnly, textField.SetReadOnly)
}

// TestRequiredFlag is PDFieldTest.testRequiredFlag.
func TestRequiredFlag(t *testing.T) {
	_, textField := newFieldFixture(t)
	assertFlagRoundTrip(t, "IsRequired", textField.IsRequired, textField.SetRequired)
}

// TestNoExportFlag is PDFieldTest.testNoExportFlag.
func TestNoExportFlag(t *testing.T) {
	_, textField := newFieldFixture(t)
	assertFlagRoundTrip(t, "IsNoExport", textField.IsNoExport, textField.SetNoExport)
}

// assertFlagRoundTrip is the false, true, false the three flag tests share.
func assertFlagRoundTrip(t *testing.T, what string, get func() bool, set func(bool)) {
	t.Helper()
	if get() {
		t.Errorf("%s() = true, want false", what)
	}
	set(true)
	if !get() {
		t.Errorf("%s() = false, want true", what)
	}
	set(false)
	if get() {
		t.Errorf("%s() = true, want false", what)
	}
}

// TestMultipleFlagsIndependently is PDFieldTest.testMultipleFlagsIndependently.
func TestMultipleFlagsIndependently(t *testing.T) {
	_, textField := newFieldFixture(t)
	textField.SetReadOnly(true)
	textField.SetRequired(true)
	textField.SetNoExport(false)
	assertFlagState(t, textField, true, true, false)

	// Change one flag and verify others remain unchanged
	textField.SetReadOnly(false)
	assertFlagState(t, textField, false, true, false)
}

// assertFlagState checks the three flags of a field.
func assertFlagState(t *testing.T, field *PDTextField, readOnly, required, noExport bool) {
	t.Helper()
	if got := field.IsReadOnly(); got != readOnly {
		t.Errorf("IsReadOnly() = %v, want %v", got, readOnly)
	}
	if got := field.IsRequired(); got != required {
		t.Errorf("IsRequired() = %v, want %v", got, required)
	}
	if got := field.IsNoExport(); got != noExport {
		t.Errorf("IsNoExport() = %v, want %v", got, noExport)
	}
}

// TestSetFieldFlagsZeroAndClearing is
// PDFieldTest.testSetFieldFlagsZeroAndClearing.
func TestSetFieldFlagsZeroAndClearing(t *testing.T) {
	_, textField := newFieldFixture(t)
	// Set all flags to true
	textField.SetReadOnly(true)
	textField.SetRequired(true)
	textField.SetNoExport(true)
	assertFlagState(t, textField, true, true, true)

	// Clear all flags by setting to 0
	textField.SetFieldFlags(0)
	assertFlagState(t, textField, false, false, false)
	if got := textField.FieldFlags(); got != 0 {
		t.Errorf("FieldFlags() = %d, want 0", got)
	}
}

// TestGetFieldType is PDFieldTest.testGetFieldType.
func TestGetFieldType(t *testing.T) {
	_, textField := newFieldFixture(t)
	if got, want := textField.FieldType(), "Tx"; got != want {
		t.Errorf("FieldType() = %q, want %q", got, want) // PDTextField has type "Tx"
	}
}

// TestSetValueAndGetValueAsString is
// PDFieldTest.testSetValueAndGetValueAsString.
//
// PDTextField requires proper form setup with /DR (Default Resources) to set
// values. This test verifies the method signatures and basic behavior without
// triggering appearance generation. For full integration testing, see
// PDTextFieldTest which uses a properly configured form.
func TestSetValueAndGetValueAsString(t *testing.T) {
	_, textField := newFieldFixture(t)
	// Verify getValueAsString returns empty string when no value is set
	if got := textField.ValueAsString(); got != "" {
		t.Errorf("ValueAsString() = %q, want empty", got)
	}
}

// TestGetWidgets is PDFieldTest.testGetWidgets.
func TestGetWidgets(t *testing.T) {
	_, textField := newFieldFixture(t)
	if textField.Widgets() == nil {
		t.Error("Widgets() = nil, want a list")
	}
}

// TestGetActionsNonNull is PDFieldTest.testGetActionsNonNull.
func TestGetActionsNonNull(t *testing.T) {
	_, textField := newFieldFixture(t)
	// First test that actions is null by default
	if got := textField.Actions(); got != nil {
		t.Errorf("Actions() = %v, want nil", got)
	}
	// Create a field with actions by adding to the dictionary
	textField.FieldDictionary().SetItem(cos.AA, cos.NewDictionary())
	// Now getActions should return a non-null PDFormFieldAdditionalActions
	if got := textField.Actions(); got == nil {
		t.Error("Actions() = nil, want the additional actions")
	}
}

// TestToStringWithValue is PDFieldTest.testToStringWithValue.
func TestToStringWithValue(t *testing.T) {
	_, textField := newFieldFixture(t)
	textField.SetPartialName("fieldWithValue")
	assertStringNames(t, textField.String(), "PDTextField", "fieldWithValue")
}

// TestFieldToString is PDFieldTest.testToString.
func TestFieldToString(t *testing.T) {
	_, textField := newFieldFixture(t)
	textField.SetPartialName("myField")
	assertStringNames(t, textField.String(), "PDTextField", "myField")
}

// assertStringNames checks the rendering of a field names both the type and the
// field.
func assertStringNames(t *testing.T, stringRepresentation string, want ...string) {
	t.Helper()
	for _, name := range want {
		if !strings.Contains(stringRepresentation, name) {
			t.Errorf("String() = %q, want it to contain %q", stringRepresentation, name)
		}
	}
}

// TestGetAcroForm is PDFieldTest.testGetAcroForm.
func TestGetAcroForm(t *testing.T) {
	acroForm, textField := newFieldFixture(t)
	if got := textField.AcroForm(); got != acroForm {
		t.Errorf("AcroForm() = %v, want %v", got, acroForm)
	}
}

// TestGetParent is PDFieldTest.testGetParent.
func TestGetParent(t *testing.T) {
	acroForm, textField := newFieldFixture(t)
	// Test parent is null for field without parent
	if got := textField.Parent(); got != nil {
		t.Errorf("Parent() = %v, want nil", got)
	}
	// Test parent is set when field has parent
	parent := NewPDNonTerminalField(acroForm)
	childField := NewPDTextFieldOf(acroForm, cos.NewDictionary(), parent)
	if got := childField.Parent(); got != parent {
		t.Errorf("Parent() = %v, want %v", got, parent)
	}
}

// TestGetCOSObject is PDFieldTest.testGetCOSObject.
func TestGetCOSObject(t *testing.T) {
	_, textField := newFieldFixture(t)
	if _, isDictionary := textField.COSObject().(*cos.Dictionary); !isDictionary {
		t.Errorf("COSObject() = %T, want *cos.Dictionary", textField.COSObject())
	}
}

// TestFieldEquals is PDFieldTest.testEquals.
func TestFieldEquals(t *testing.T) {
	acroForm, _ := newFieldFixture(t)
	field1 := NewPDTextField(acroForm)
	field2 := NewPDTextField(acroForm)

	// Fields with same COS dictionary should be equal
	field1.SetPartialName("testField")
	field3 := NewPDTextFieldOf(acroForm, field1.FieldDictionary(), nil)
	if !field1.Equals(field3) {
		t.Error("field1.Equals(field3) = false, want true")
	}

	// Different fields should not be equal
	field2.SetPartialName("differentField")
	if field1.Equals(field2) {
		t.Error("field1.Equals(field2) = true, want false")
	}

	// Field should equal itself
	if !field1.Equals(field1) {
		t.Error("field1.Equals(field1) = false, want true")
	}

	// Java also asserts a field equals neither null nor a string; the port's
	// Equals takes a PDField, so only the nil half can be written.
	if field1.Equals(nil) {
		t.Error("field1.Equals(nil) = true, want false")
	}
}

// TestGetActions is PDFieldTest.testGetActions.
func TestGetActions(t *testing.T) {
	_, textField := newFieldFixture(t)
	// By default, actions should be null
	if got := textField.Actions(); got != nil {
		t.Errorf("Actions() = %v, want nil", got)
	}
}

// TestMultiplePropertiesTogether is PDFieldTest.testMultiplePropertiesTogether.
func TestMultiplePropertiesTogether(t *testing.T) {
	_, textField := newFieldFixture(t)
	textField.SetPartialName("complexField")
	textField.SetAlternateFieldName("Complex Field")
	textField.SetMappingName("complex_field")
	textField.SetReadOnly(true)
	textField.SetRequired(true)
	if got, want := textField.PartialName(), "complexField"; got != want {
		t.Errorf("PartialName() = %q, want %q", got, want)
	}
	if got, want := textField.AlternateFieldName(), "Complex Field"; got != want {
		t.Errorf("AlternateFieldName() = %q, want %q", got, want)
	}
	if got, want := textField.MappingName(), "complex_field"; got != want {
		t.Errorf("MappingName() = %q, want %q", got, want)
	}
	if !textField.IsReadOnly() {
		t.Error("IsReadOnly() = false, want true")
	}
	if !textField.IsRequired() {
		t.Error("IsRequired() = false, want true")
	}
}
