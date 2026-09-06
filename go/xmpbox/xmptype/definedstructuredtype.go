package xmptype

// A structured type a PDF/A extension schema defined at run time.
//
// Port of org.apache.xmpbox.type.DefinedStructuredType.

// DefinedStructuredType is a structured type whose fields are not compiled in
// but read out of an extension schema.
type DefinedStructuredType struct {
	StructuredType

	definedProperties map[string]PropertyType

	// order keeps the order properties were added in, for the reason
	// PropertiesDescription gives.
	order []string
}

var _ AbstractStructuredType = (*DefinedStructuredType)(nil)

// NewDefinedStructuredType returns a defined type in the given namespace.
//
// Port of DefinedStructuredType(XMPMetadata, String, String, String). The class
// carries no @StructuredType annotation, so the namespace argument is the one
// that counts and an empty one is the IllegalArgumentException Java raises.
func NewDefinedStructuredType(metadata MetadataLike, namespaceURI, fieldPrefix,
	propertyName string) *DefinedStructuredType {
	d := &DefinedStructuredType{definedProperties: map[string]PropertyType{}}
	// The error is the IllegalArgumentException of a null namespace with no
	// annotation, which this constructor cannot report; Java's caller passes a
	// namespace it has.
	_ = d.InitStructuredType(metadata, StructuredTypeInfo{}, namespaceURI, fieldPrefix, propertyName)
	return d
}

// AddDefinedProperty records the type of one of the fields the extension schema
// defined.
//
// Java names it addProperty, which collides with the addProperty every complex
// property has; Go has one namespace for the two, so the port spells this one
// out. Java's own javadoc calls them the defined properties.
func (d *DefinedStructuredType) AddDefinedProperty(name string, propertyType PropertyType) {
	if _, replaced := d.definedProperties[name]; !replaced {
		d.order = append(d.order, name)
	}
	d.definedProperties[name] = propertyType
}

// DefinedProperties returns the fields the extension schema defined.
func (d *DefinedStructuredType) DefinedProperties() map[string]PropertyType {
	return d.definedProperties
}

// DefinedPropertyNames returns the names of those fields, in the order they
// were added.
func (d *DefinedStructuredType) DefinedPropertyNames() []string {
	names := make([]string, 0, len(d.definedProperties))
	for _, name := range d.order {
		if _, found := d.definedProperties[name]; found {
			names = append(names, name)
		}
	}
	return names
}

// TypeName returns the name of the Java class this stands for.
func (d *DefinedStructuredType) TypeName() string { return "DefinedStructuredType" }
