package xmptype_test

// Port of org.apache.xmpbox.type.TestSimpleMetadataProperties.
//
// It is an external test because it builds a real xmpbox.XMPMetadata, and
// xmpbox imports xmptype; an in-package test could not name it. D3's "does each
// test take the real path, with the real types" is why the metadata is the real
// one rather than a stand-in.
//
// Java asserts assertThrows(IllegalArgumentException.class, ...) for a value of
// the wrong shape. The port's constructors answer an error instead, which is
// what migration/conventions/java-to-go.md says an IllegalArgumentException out
// of a constructor becomes, so each of those becomes a check that the error is
// not nil.

import (
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// parent is Java's `private final XMPMetadata parent =
// XMPMetadata.createXMPMetadata()`, per test rather than per class because a Go
// test has no instance field.
func parent(t *testing.T) *xmpbox.XMPMetadata {
	t.Helper()
	return xmpbox.CreateXMPMetadata()
}

// TestBooleanBadTypeDetection is Java's test of the same name.
func TestBooleanBadTypeDetection(t *testing.T) {
	if _, err := xmptype.NewBooleanType(parent(t), "", "test", "boolean",
		"Not a Boolean"); err == nil {
		t.Error("NewBooleanType(\"Not a Boolean\") = nil error, want one")
	}
}

// TestDateBadTypeDetection is Java's test of the same name: a string that is
// not a date is refused, and so are a nil and an int handed to setValue.
func TestDateBadTypeDetection(t *testing.T) {
	if _, err := xmptype.NewDateType(parent(t), "", "test", "date", "Bad Date"); err == nil {
		t.Error("NewDateType(\"Bad Date\") = nil error, want one")
	}
	date, err := xmptype.NewDateType(parent(t), "", "test", "date", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := date.SetValue(nil); err == nil {
		t.Error("SetValue(nil) = nil error, want one")
	}
	if err := date.SetValue(3); err == nil {
		t.Error("SetValue(3) = nil error, want one")
	}
}

// TestIntegerBadTypeDetection is Java's test of the same name.
func TestIntegerBadTypeDetection(t *testing.T) {
	if _, err := xmptype.NewIntegerType(parent(t), "", "test", "integer",
		"Not an int"); err == nil {
		t.Error("NewIntegerType(\"Not an int\") = nil error, want one")
	}
}

// TestRealBadTypeDetection is Java's test of the same name.
func TestRealBadTypeDetection(t *testing.T) {
	if _, err := xmptype.NewRealType(parent(t), "", "test", "real", "Not a real"); err == nil {
		t.Error("NewRealType(\"Not a real\") = nil error, want one")
	}
}

// TestTextBadTypeDetection is Java's test of the same name, which hands a text
// property a Calendar.
func TestTextBadTypeDetection(t *testing.T) {
	calendar := time.Now()
	if _, err := xmptype.NewTextType(parent(t), "", "test", "text", calendar); err == nil {
		t.Error("NewTextType(a date) = nil error, want one")
	}
}

// TestElementAndObjectSynchronization is Java's test of the same name: each of
// the five factory methods gives back the value it was handed.
func TestElementAndObjectSynchronization(t *testing.T) {
	metadata := parent(t)
	boolv := true
	datev := time.Now()
	integerv := 1
	realv := float32(1.69)
	textv := "TEXTCONTENT"

	mapping := metadata.TypeMapping()
	boolean, err := mapping.CreateBoolean("", "test", "boolean", boolv)
	if err != nil {
		t.Fatal(err)
	}
	date, err := mapping.CreateDate("", "test", "date", datev)
	if err != nil {
		t.Fatal(err)
	}
	integer, err := mapping.CreateInteger("", "test", "integer", integerv)
	if err != nil {
		t.Fatal(err)
	}
	real, err := mapping.CreateReal("", "test", "real", realv)
	if err != nil {
		t.Fatal(err)
	}
	text, err := mapping.CreateText("", "test", "text", textv)
	if err != nil {
		t.Fatal(err)
	}

	if got := boolean.BooleanValue(); got != boolv {
		t.Errorf("BooleanValue() = %v, want %v", got, boolv)
	}
	if got := date.DateValue(); !got.Equal(datev) {
		t.Errorf("DateValue() = %v, want %v", got, datev)
	}
	if got := integer.IntegerValue(); got != integerv {
		t.Errorf("IntegerValue() = %v, want %v", got, integerv)
	}
	if got := real.RealValue(); got != realv {
		t.Errorf("RealValue() = %v, want %v", got, realv)
	}
	if got := text.StringValue(); got != textv {
		t.Errorf("StringValue() = %q, want %q", got, textv)
	}
}

// TestCreationFromString is Java's test of the same name: a property built from
// a string writes that same string back.
func TestCreationFromString(t *testing.T) {
	metadata := parent(t)
	boolv := "False"
	datev := "2010-03-22T14:33:11+01:00"
	integerv := "10"
	realv := "1.92"
	textv := "text"

	boolean, err := xmptype.NewBooleanType(metadata, "", "test", "boolean", boolv)
	if err != nil {
		t.Fatal(err)
	}
	date, err := xmptype.NewDateType(metadata, "", "test", "date", datev)
	if err != nil {
		t.Fatal(err)
	}
	integer, err := xmptype.NewIntegerType(metadata, "", "test", "integer", integerv)
	if err != nil {
		t.Fatal(err)
	}
	real, err := xmptype.NewRealType(metadata, "", "test", "real", realv)
	if err != nil {
		t.Fatal(err)
	}
	text, err := xmptype.NewTextType(metadata, "", "test", "text", textv)
	if err != nil {
		t.Fatal(err)
	}

	if got := boolean.StringValue(); got != boolv {
		t.Errorf("StringValue() = %q, want %q", got, boolv)
	}
	if got := date.StringValue(); got != datev {
		t.Errorf("StringValue() = %q, want %q", got, datev)
	}
	if got := integer.StringValue(); got != integerv {
		t.Errorf("StringValue() = %q, want %q", got, integerv)
	}
	if got := real.StringValue(); got != realv {
		t.Errorf("StringValue() = %q, want %q", got, realv)
	}
	if got := text.StringValue(); got != textv {
		t.Errorf("StringValue() = %q, want %q", got, textv)
	}
}

// TestObjectCreationWithNamespace is Java's test of the same name: the
// namespace a factory method is handed reaches the property.
func TestObjectCreationWithNamespace(t *testing.T) {
	metadata := parent(t)
	ns := "http://www.test.org/pdfa/"
	mapping := metadata.TypeMapping()

	boolean, err := mapping.CreateBoolean(ns, "test", "boolean", true)
	if err != nil {
		t.Fatal(err)
	}
	date, err := mapping.CreateDate(ns, "test", "date", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	integer, err := mapping.CreateInteger(ns, "test", "integer", 1)
	if err != nil {
		t.Fatal(err)
	}
	real, err := mapping.CreateReal(ns, "test", "real", float32(1.6))
	if err != nil {
		t.Fatal(err)
	}
	text, err := mapping.CreateText(ns, "test", "text", "TEST")
	if err != nil {
		t.Fatal(err)
	}

	for name, got := range map[string]string{
		"boolean": boolean.Namespace(),
		"date":    date.Namespace(),
		"integer": integer.Namespace(),
		"real":    real.Namespace(),
		"text":    text.Namespace(),
	} {
		if got != ns {
			t.Errorf("%s Namespace() = %q, want %q", name, got, ns)
		}
	}
}

// TestAttribute is Java's test of the same name: an attribute is stored under
// its local name, replaced by another of that name whatever its namespace, and
// removed by name.
func TestAttribute(t *testing.T) {
	integer, err := xmptype.NewIntegerType(parent(t), "", "test", "integer", 1)
	if err != nil {
		t.Fatal(err)
	}
	value := xmptype.NewAttribute("http://www.test.org/test/", "value1", "StringValue1")
	value2 := xmptype.NewAttribute("http://www.test.org/test/", "value2", "StringValue2")

	integer.SetAttribute(value)
	if got := integer.Attribute(value.Name()); got != value {
		t.Errorf("Attribute(%q) = %v, want the attribute that was set", value.Name(), got)
	}
	if !integer.ContainsAttribute(value.Name()) {
		t.Errorf("ContainsAttribute(%q) = false, want true", value.Name())
	}

	// Replacement check
	integer.SetAttribute(value2)
	if got := integer.Attribute(value2.Name()); got != value2 {
		t.Errorf("Attribute(%q) = %v, want the attribute that replaced it", value2.Name(), got)
	}
	integer.RemoveAttribute(value2.Name())
	if integer.ContainsAttribute(value2.Name()) {
		t.Errorf("ContainsAttribute(%q) = true after removal, want false", value2.Name())
	}

	// Attribute with namespace Creation checking
	valueNS := xmptype.NewAttribute("http://www.tefst2.org/test/", "value2", "StringValue.2")
	integer.SetAttribute(valueNS)
	valueNS2 := xmptype.NewAttribute("http://www.test2.org/test/", "value2", "StringValueTwo")
	integer.SetAttribute(valueNS2)

	atts := integer.AllAttributes()
	if containsAttribute(atts, valueNS) {
		t.Error("the attribute of the same local name was not replaced")
	}
	if !containsAttribute(atts, valueNS2) {
		t.Error("the replacing attribute is not among them")
	}
}

// containsAttribute is Java's List.contains, which for an Attribute is
// identity because the class does not override equals.
func containsAttribute(atts []*xmptype.Attribute, want *xmptype.Attribute) bool {
	for _, att := range atts {
		if att == want {
			return true
		}
	}
	return false
}
