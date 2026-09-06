package xmptype

// The factory half of TypeMapping.
//
// Port of TypeMapping.createBoolean and the fourteen create methods beside it,
// plus createPropertyType and getSpecifiedPropertyType. They are kept out of
// typemapping.go the way Java keeps them at the end of the class.

import "fmt"

// CreateBoolean returns a boolean property.
func (t *TypeMapping) CreateBoolean(namespaceURI, prefix, propertyName string,
	value bool) (*BooleanType, error) {
	return NewBooleanType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateDate returns a date property.
func (t *TypeMapping) CreateDate(namespaceURI, prefix, propertyName string,
	value any) (*DateType, error) {
	return NewDateType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateInteger returns an integer property.
func (t *TypeMapping) CreateInteger(namespaceURI, prefix, propertyName string,
	value int) (*IntegerType, error) {
	return NewIntegerType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateReal returns a real property.
func (t *TypeMapping) CreateReal(namespaceURI, prefix, propertyName string,
	value float32) (*RealType, error) {
	return NewRealType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateText returns a text property.
func (t *TypeMapping) CreateText(namespaceURI, prefix, propertyName,
	value string) (*TextType, error) {
	return NewTextType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateProperName returns a proper name property.
func (t *TypeMapping) CreateProperName(namespaceURI, prefix, propertyName,
	value string) (*ProperNameType, error) {
	return NewProperNameType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateURI returns a URI property.
func (t *TypeMapping) CreateURI(namespaceURI, prefix, propertyName,
	value string) (*URIValueType, error) {
	return NewURIValueType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateURL returns a URL property.
func (t *TypeMapping) CreateURL(namespaceURI, prefix, propertyName,
	value string) (*URLValueType, error) {
	return NewURLValueType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateRenditionClass returns a rendition class property.
func (t *TypeMapping) CreateRenditionClass(namespaceURI, prefix, propertyName,
	value string) (*RenditionClassType, error) {
	return NewRenditionClassType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreatePart returns a part property.
func (t *TypeMapping) CreatePart(namespaceURI, prefix, propertyName,
	value string) (*PartValueType, error) {
	return NewPartValueType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateMIMEType returns a MIME type property.
func (t *TypeMapping) CreateMIMEType(namespaceURI, prefix, propertyName,
	value string) (*MIMEValueType, error) {
	return NewMIMEValueType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateLocale returns a locale property.
func (t *TypeMapping) CreateLocale(namespaceURI, prefix, propertyName,
	value string) (*LocaleType, error) {
	return NewLocaleType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateGUID returns a GUID property.
func (t *TypeMapping) CreateGUID(namespaceURI, prefix, propertyName,
	value string) (*GUIDType, error) {
	return NewGUIDType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateChoice returns a choice property.
func (t *TypeMapping) CreateChoice(namespaceURI, prefix, propertyName,
	value string) (*ChoiceType, error) {
	return NewChoiceType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateAgentName returns an agent name property.
func (t *TypeMapping) CreateAgentName(namespaceURI, prefix, propertyName,
	value string) (*AgentNameType, error) {
	return NewAgentNameType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateXPath returns an XPath property.
func (t *TypeMapping) CreateXPath(namespaceURI, prefix, propertyName,
	value string) (*XPathValueType, error) {
	return NewXPathValueType(t.metadata, namespaceURI, prefix, propertyName, value)
}

// CreateArrayProperty returns an array property of the given cardinality.
func (t *TypeMapping) CreateArrayProperty(namespace, prefix, propertyName string,
	cardinality Cardinality) *ArrayProperty {
	return NewArrayProperty(t.metadata, namespace, prefix, propertyName, cardinality)
}

// CreatePropertyType returns a property type of the given type and cardinality.
//
// Port of the static createPropertyType, which Java writes as an anonymous
// implementation of the annotation because an annotation cannot be constructed;
// the port's PropertyType is a plain value, so this is its constructor.
func CreatePropertyType(t Types, card Cardinality) PropertyType {
	return PropertyType{Type: t, Card: card}
}

// QName is an XML qualified name: a namespace URI and a local part.
//
// Port of javax.xml.namespace.QName, of which getSpecifiedPropertyType uses
// exactly these two accessors.
type QName struct {
	NamespaceURI string
	LocalPart    string
}

// String returns the Java QName.toString form.
func (q QName) String() string {
	if q.NamespaceURI == "" {
		return q.LocalPart
	}
	return "{" + q.NamespaceURI + "}" + q.LocalPart
}

// SpecifiedPropertyType returns the type declared for the given qualified name,
// and false where the caller should treat it as absent -- which is Java's null.
//
// Port of getSpecifiedPropertyType(QName, String).
//
// PDFBOX-6133: the method was rewritten because of photoshop and exif,
// because these namespaces exist as a schema and as a type
// "factory" is checked in the non-schema part to keep the pre PDFBOX-6133 behavior
func (t *TypeMapping) SpecifiedPropertyType(qName QName,
	parentTypeName string) (PropertyType, bool, error) {
	factory := t.SchemaFactory(qName.NamespaceURI)
	if factory != nil {
		// found in schema
		if propertyType, found := factory.PropertyDefinition().PropertyType(qName.LocalPart); found {
			return propertyType, true, nil
		}
	}

	// try in structured
	list, inStructured := t.structuredNamespaces2[qName.NamespaceURI]
	if inStructured {
		if len(list) == 1 {
			st := list[0]
			propDesc := t.structuredMappings[st]
			if factory == nil || containsName(propDesc.PropertiesNames(), qName.LocalPart) {
				return CreatePropertyType(st, Simple), true, nil
			}
			return PropertyType{}, false, nil
		}
		if len(list) > 1 {
			for _, structuredType := range list {
				if structuredType.String() == parentTypeName {
					return CreatePropertyType(structuredType, Simple), true, nil
				}
			}
			for _, structuredType := range list {
				propDesc := t.structuredMappings[structuredType]
				if containsName(propDesc.PropertiesNames(), qName.LocalPart) {
					return CreatePropertyType(structuredType, Simple), true, nil
				}
			}
		}
		return PropertyType{}, false, nil
	}

	// try in defined
	if _, inDefined := t.definedStructuredNamespaces2[qName.NamespaceURI]; !inDefined {
		// not found
		if factory != nil {
			return PropertyType{}, false, nil // pre PDFBOX-6133 behavior
		}
		return PropertyType{}, false,
			fmt.Errorf("%w: No descriptor found for %v", ErrBadFieldValue, qName)
	}
	return CreatePropertyType(DefinedType, Simple), true, nil
}

// containsName is Java's List.contains over the property names.
func containsName(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}
