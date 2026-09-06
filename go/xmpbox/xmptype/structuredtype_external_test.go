package xmptype_test

// Port of org.apache.xmpbox.type.TestAbstractStructuredType and TestDerivedType.
//
// Both are external tests for the reason
// simplemetadataproperties_external_test.go gives.

import (
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// The namespace and prefix Java's TestAbstractStructuredType declares.
const (
	myNS     = "http://www.apache.org/test#"
	myPrefix = "test"
)

// The two fields Java's private MyStructuredType declares, with the
// @PropertyType annotations that go with them.
const (
	myText = "my-text"
	myDate = "my-date"
)

// myStructuredType is Java's private static class MyStructuredType: a
// structured type with no @StructuredType annotation, so the namespace and
// prefix come from its constructor arguments.
type myStructuredType struct {
	xmptype.StructuredType
}

// myStructuredProperties is what Java reads off MyStructuredType's two
// annotated fields.
var myStructuredProperties = func() *xmptype.PropertiesDescription {
	d := xmptype.NewPropertiesDescription()
	d.AddNewProperty(myText, xmptype.NewPropertyTypeCard(xmptype.Text, xmptype.Simple))
	d.AddNewProperty(myDate, xmptype.NewPropertyTypeCard(xmptype.Date, xmptype.Simple))
	return d
}()

// newMyStructuredType is Java's MyStructuredType(XMPMetadata, String, String).
func newMyStructuredType(t *testing.T, metadata xmptype.MetadataLike,
	namespaceURI, fieldPrefix string) *myStructuredType {
	t.Helper()
	s := &myStructuredType{}
	if err := s.InitStructuredType(metadata, xmptype.StructuredTypeInfo{},
		namespaceURI, fieldPrefix, "structuredPN"); err != nil {
		t.Fatal(err)
	}
	return s
}

// TypeName returns the name of the Java class this stands for.
func (s *myStructuredType) TypeName() string { return "MyStructuredType" }

// structuredFixture is Java's `protected final MyStructuredType st = new
// MyStructuredType(xmp, MY_NS, MY_PREFIX)`, built per test because a Go test
// has no instance field.
func structuredFixture(t *testing.T) *myStructuredType {
	t.Helper()
	return newMyStructuredType(t, xmpbox.CreateXMPMetadata(), myNS, myPrefix)
}

// TestValidate is Java's validate: the namespace and prefix a structured type
// with no annotation was built with come back.
func TestValidate(t *testing.T) {
	st := structuredFixture(t)
	if got := st.Namespace(); got != myNS {
		t.Errorf("Namespace() = %q, want %q", got, myNS)
	}
	if got := st.Prefix(); got != myPrefix {
		t.Errorf("Prefix() = %q, want %q", got, myPrefix)
	}
}

// TestNonExistingProperty is Java's testNonExistingProperty.
func TestNonExistingProperty(t *testing.T) {
	if got := structuredFixture(t).Property("NOT_EXISTING"); got != nil {
		t.Errorf("Property(\"NOT_EXISTING\") = %v, want nil", got)
	}
}

// TestNotValuatedPropertyProperty is Java's test of the same name: a field the
// type declares but that was never set is still absent.
func TestNotValuatedPropertyProperty(t *testing.T) {
	if got := structuredFixture(t).Property(myText); got != nil {
		t.Errorf("Property(%q) = %v, want nil", myText, got)
	}
}

// TestValuatedTextProperty is Java's test of the same name.
func TestValuatedTextProperty(t *testing.T) {
	st := structuredFixture(t)
	s := "my value"
	if err := st.AddSimpleProperty(myStructuredProperties, myText, s); err != nil {
		t.Fatal(err)
	}
	if got := st.PropertyValueAsString(myText); got != s {
		t.Errorf("PropertyValueAsString(%q) = %q, want %q", myText, got, s)
	}
	if got := st.PropertyValueAsString(myDate); got != "" {
		t.Errorf("PropertyValueAsString(%q) = %q, want the empty string, which is Java's null",
			myDate, got)
	}
	if got := st.Property(myText); got == nil {
		t.Errorf("Property(%q) = nil, want the property that was set", myText)
	}
}

// TestValuatedDateProperty is Java's test of the same name.
func TestValuatedDateProperty(t *testing.T) {
	st := structuredFixture(t)
	c := time.Now()
	if err := st.AddSimpleProperty(myStructuredProperties, myDate, c); err != nil {
		t.Fatal(err)
	}
	got, found := st.DatePropertyAsCalendar(myDate)
	if !found || !got.Equal(c) {
		t.Errorf("DatePropertyAsCalendar(%q) = %v, %t, want %v, true", myDate, got, found, c)
	}
	if _, found := st.DatePropertyAsCalendar(myText); found {
		t.Errorf("DatePropertyAsCalendar(%q) found a date, want none", myText)
	}
	if got := st.Property(myDate); got == nil {
		t.Errorf("Property(%q) = nil, want the property that was set", myDate)
	}
}

// The prefix, name and value Java's TestDerivedType declares.
const (
	derivedPrefix = "myprefix"
	derivedName   = "myname"
	derivedValue  = "myvalue"
)

// TestDerivedTypes is Java's parameterized test1, over the eleven text types it
// names. Java reaches each constructor by reflection; the port names them,
// because there is none.
func TestDerivedTypes(t *testing.T) {
	xmp := xmpbox.CreateXMPMetadata()

	for _, test := range []struct {
		typeName  string
		construct func(m xmptype.MetadataLike, ns, prefix, name string,
			value any) (xmptype.AbstractSimpleProperty, error)
	}{
		{"AgentNameType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewAgentNameType(m, ns, p, n, v)
		}},
		{"ChoiceType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewChoiceType(m, ns, p, n, v)
		}},
		{"GUIDType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewGUIDType(m, ns, p, n, v)
		}},
		{"LocaleType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewLocaleType(m, ns, p, n, v)
		}},
		{"MIMEType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewMIMEValueType(m, ns, p, n, v)
		}},
		{"PartType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewPartValueType(m, ns, p, n, v)
		}},
		{"ProperNameType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewProperNameType(m, ns, p, n, v)
		}},
		{"RenditionClassType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewRenditionClassType(m, ns, p, n, v)
		}},
		{"URIType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewURIValueType(m, ns, p, n, v)
		}},
		{"URLType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewURLValueType(m, ns, p, n, v)
		}},
		{"XPathType", func(m xmptype.MetadataLike, ns, p, n string, v any) (xmptype.AbstractSimpleProperty, error) {
			return xmptype.NewXPathValueType(m, ns, p, n, v)
		}},
	} {
		element, err := test.construct(xmp, "", derivedPrefix, derivedName, derivedValue)
		if err != nil {
			t.Fatalf("%s: %v", test.typeName, err)
		}
		if got := element.Namespace(); got != "" {
			t.Errorf("%s: Namespace() = %q, want the empty string, which is Java's null",
				test.typeName, got)
		}
		value, isString := element.Value().(string)
		if !isString {
			t.Errorf("%s: Value() = %T, want a string", test.typeName, element.Value())
			continue
		}
		if value != derivedValue {
			t.Errorf("%s: Value() = %q, want %q", test.typeName, value, derivedValue)
		}
		if got := element.TypeName(); got != test.typeName {
			t.Errorf("TypeName() = %q, want %q", got, test.typeName)
		}
	}
}
