package xmptype

// The two Java annotations that describe a property and a structured type.
//
// Port of org.apache.xmpbox.type.PropertyType and StructuredType, plus
// PropertiesDescription, which collects the first of them for one class.
//
// Java reads both off a class at run time with reflection. Go has none the port
// uses, so each is a value the type declares about itself: a structured type
// carries a StructuredType, and a class that has @PropertyType fields carries a
// PropertiesDescription built once in its package's init.

import "fmt"

// PropertyType says what a property holds and how many of them.
//
// Port of the @PropertyType annotation, whose card() defaults to Simple.
type PropertyType struct {
	// Type is the annotation's type().
	Type Types

	// Card is the annotation's card().
	Card Cardinality
}

// NewPropertyType returns a simple property of the given type, which is the
// annotation with card() left at its default.
func NewPropertyType(t Types) PropertyType {
	return PropertyType{Type: t, Card: Simple}
}

// NewPropertyTypeCard returns a property of the given type and cardinality.
func NewPropertyTypeCard(t Types, card Cardinality) PropertyType {
	return PropertyType{Type: t, Card: card}
}

// String returns a form of the annotation for a message.
func (p PropertyType) String() string {
	return fmt.Sprintf("@PropertyType(type=%v, card=%v)", p.Type, p.Card)
}

// StructuredType is the namespace and preferred prefix of a structured type or
// a schema.
//
// Port of the @StructuredType annotation.
type StructuredTypeInfo struct {
	// Namespace is the annotation's namespace().
	Namespace string

	// PreferedPrefix is the annotation's preferedPrefix(), spelled the way Java
	// spells it.
	PreferedPrefix string
}

// PropertiesDescription is the properties one class declares, by name.
//
// Port of PropertiesDescription.
type PropertiesDescription struct {
	types map[string]PropertyType

	// order keeps the order properties were added in, because Go map iteration
	// is random where Java's HashMap is at least stable within one map.
	order []string
}

// NewPropertiesDescription returns an empty description.
func NewPropertiesDescription() *PropertiesDescription {
	return &PropertiesDescription{types: map[string]PropertyType{}}
}

// PropertiesNames returns the names of every property described.
func (d *PropertiesDescription) PropertiesNames() []string {
	names := make([]string, 0, len(d.types))
	for _, name := range d.order {
		if _, found := d.types[name]; found {
			names = append(names, name)
		}
	}
	return names
}

// AddNewProperty records the type of the property of the given name.
func (d *PropertiesDescription) AddNewProperty(name string, propertyType PropertyType) {
	if _, replaced := d.types[name]; !replaced {
		d.order = append(d.order, name)
	}
	d.types[name] = propertyType
}

// PropertyType returns the type of the property of the given name, the second
// result being false where the description does not name it -- which is Java's
// null.
func (d *PropertiesDescription) PropertyType(name string) (PropertyType, bool) {
	propertyType, found := d.types[name]
	return propertyType, found
}

// String returns the Java toString form.
func (d *PropertiesDescription) String() string {
	return fmt.Sprintf("PropertiesDescription{types=%v}", d.types)
}
