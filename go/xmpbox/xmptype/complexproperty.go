package xmptype

// The properties that hold other properties.
//
// Port of org.apache.xmpbox.type.ComplexPropertyContainer,
// AbstractComplexProperty, AbstractStructuredType and ArrayProperty.

import "time"

// ComplexPropertyContainer is the list of properties a complex property holds.
//
// Port of ComplexPropertyContainer.
type ComplexPropertyContainer struct {
	properties []AbstractField
}

// NewComplexPropertyContainer returns an empty container.
func NewComplexPropertyContainer() *ComplexPropertyContainer {
	return &ComplexPropertyContainer{}
}

// FirstEquivalentProperty returns the first property of the given local name
// whose type is the given one, or nil.
//
// Port of the protected getFirstEquivalentProperty(String, Class). Java compares
// getClass() with the Class it is handed; the port compares TypeName, which
// stands for getSimpleName because there is no reflection here.
func (c *ComplexPropertyContainer) FirstEquivalentProperty(localName, typeName string) AbstractField {
	list := c.PropertiesByLocalName(localName)
	if list != nil {
		for _, abstractField := range list {
			if abstractField.TypeName() == typeName {
				return abstractField
			}
		}
	}
	return nil
}

// AddProperty adds a property, removing any that is the same one first.
func (c *ComplexPropertyContainer) AddProperty(obj AbstractField) {
	c.RemoveProperty(obj)
	c.properties = append(c.properties, obj)
}

// AllProperties returns every property, in order.
func (c *ComplexPropertyContainer) AllProperties() []AbstractField { return c.properties }

// PropertiesByLocalName returns every property of the given name, and nil where
// there is none.
func (c *ComplexPropertyContainer) PropertiesByLocalName(localName string) []AbstractField {
	var list []AbstractField
	for _, abstractField := range c.AllProperties() {
		if abstractField.PropertyName() == localName {
			list = append(list, abstractField)
		}
	}
	if len(list) == 0 {
		return nil
	}
	return list
}

// IsSameProperty reports whether the two are the same property.
//
// Port of isSameProperty. AbstractField does not override equals, so Java's
// prop1.equals(prop2) is identity; the port compares the pointers. The port's
// property name is a string where Java's may be null, so the "both names are
// null" arm of the Java is "both names are empty" here.
func (c *ComplexPropertyContainer) IsSameProperty(prop1, prop2 AbstractField) bool {
	if prop1.TypeName() == prop2.TypeName() {
		pn1 := prop1.PropertyName()
		pn2 := prop2.PropertyName()
		if pn1 == "" {
			return pn2 == ""
		}
		if pn1 == pn2 {
			return prop1 == prop2
		}
	}
	return false
}

// ContainsProperty reports whether the container holds the given property.
func (c *ComplexPropertyContainer) ContainsProperty(property AbstractField) bool {
	for _, tmp := range c.AllProperties() {
		if c.IsSameProperty(tmp, property) {
			return true
		}
	}
	return false
}

// RemoveProperty removes the given property.
//
// Java's List.remove uses equals, which for a field is identity.
func (c *ComplexPropertyContainer) RemoveProperty(property AbstractField) {
	for i, tmp := range c.properties {
		if tmp == property {
			c.properties = append(c.properties[:i], c.properties[i+1:]...)
			return
		}
	}
}

// RemovePropertiesByName removes every property of the given name.
func (c *ComplexPropertyContainer) RemovePropertiesByName(localName string) {
	if len(c.properties) == 0 {
		return
	}
	propList := c.PropertiesByLocalName(localName)
	if propList == nil {
		return
	}
	for _, property := range propList {
		c.RemoveProperty(property)
	}
}

// AbstractComplexProperty is a property holding other properties.
//
// Port of the abstract AbstractComplexProperty.
type AbstractComplexProperty interface {
	AbstractField

	// Container returns the container of this property.
	Container() *ComplexPropertyContainer

	// AddProperty adds a property to this one.
	AddProperty(obj AbstractField)

	// RemoveProperty removes a property from this one.
	RemoveProperty(property AbstractField)

	// AllProperties returns every property this one holds.
	AllProperties() []AbstractField

	// Property returns the first property of the given name, or nil.
	Property(fieldName string) AbstractField

	// ArrayPropertyOf returns the first property of the given name as an array,
	// or nil.
	ArrayPropertyOf(fieldName string) *ArrayProperty

	// AddNamespace records the prefix a namespace is written with.
	AddNamespace(namespace, prefix string)

	// NamespacePrefix returns the prefix recorded for a namespace.
	NamespacePrefix(namespace string) string

	// AllNamespacesWithPrefix returns every namespace and its prefix.
	AllNamespacesWithPrefix() map[string]string
}

// ComplexProperty is the state and the concrete methods of
// AbstractComplexProperty, which every implementation embeds.
type ComplexProperty struct {
	Field

	container         *ComplexPropertyContainer
	namespaceToPrefix map[string]string

	// isArray says whether the embedder is an ArrayProperty, which AddProperty
	// tests for. Java asks `this instanceof ArrayProperty`, which Go cannot do
	// from the embedded struct.
	isArray bool
}

// initComplex is Java's protected AbstractComplexProperty(XMPMetadata, String)
// constructor.
func (p *ComplexProperty) initComplex(metadata MetadataLike, propertyName string) {
	p.InitField(metadata, propertyName)
	p.container = NewComplexPropertyContainer()
	p.namespaceToPrefix = map[string]string{}
}

// AddNamespace records the prefix a namespace is written with.
func (p *ComplexProperty) AddNamespace(namespace, prefix string) {
	p.namespaceToPrefix[namespace] = prefix
}

// NamespacePrefix returns the prefix recorded for a namespace.
func (p *ComplexProperty) NamespacePrefix(namespace string) string {
	return p.namespaceToPrefix[namespace]
}

// AllNamespacesWithPrefix returns every namespace and its prefix.
func (p *ComplexProperty) AllNamespacesWithPrefix() map[string]string {
	return p.namespaceToPrefix
}

// AddProperty adds a property to this one, replacing any of the same name
// unless this is an array.
//
// https://www.adobe.com/content/dam/Adobe/en/devnet/xmp/pdfs/cs6/XMPSpecificationPart1.pdf
// "Each property name in an XMP packet shall be unique within that packet"
// "Multiple values are represented using an XMP array value"
// "The nested element's element content shall consist of zero or more rdf:li
// elements, one for each item in the array"
// thus delete existing elements of a property, except for arrays ("li")
func (p *ComplexProperty) AddProperty(obj AbstractField) {
	if !p.isArray {
		p.container.RemovePropertiesByName(obj.PropertyName())
	}
	p.container.AddProperty(obj)
}

// RemoveProperty removes a property from this one.
func (p *ComplexProperty) RemoveProperty(property AbstractField) {
	p.container.RemoveProperty(property)
}

// Container returns the container of this property.
func (p *ComplexProperty) Container() *ComplexPropertyContainer { return p.container }

// AllProperties returns every property this one holds.
func (p *ComplexProperty) AllProperties() []AbstractField { return p.container.AllProperties() }

// Property returns the first property of the given name, or nil.
func (p *ComplexProperty) Property(fieldName string) AbstractField {
	list := p.container.PropertiesByLocalName(fieldName)
	// return null if no property
	if list == nil {
		return nil
	}
	// return the first element of the list
	return list[0]
}

// ArrayPropertyOf returns the first property of the given name as an array, or
// nil.
//
// Java casts, which throws ClassCastException where the property is not an
// array; the port answers nil, because a Go type assertion that fails has no
// exception to raise from a getter.
func (p *ComplexProperty) ArrayPropertyOf(fieldName string) *ArrayProperty {
	list := p.container.PropertiesByLocalName(fieldName)
	// return null if no property
	if list == nil {
		return nil
	}
	// return the first element of the list
	array, isArray := list[0].(*ArrayProperty)
	if !isArray {
		return nil
	}
	return array
}

// FirstEquivalentProperty returns the first property of the given name and type.
func (p *ComplexProperty) FirstEquivalentProperty(localName, typeName string) AbstractField {
	return p.container.FirstEquivalentProperty(localName, typeName)
}

// StructureArrayName is the element name of an item of a structured array.
//
// Port of AbstractStructuredType.STRUCTURE_ARRAY_NAME.
const StructureArrayName = "li"

// AbstractStructuredType is a property whose fields are named by a schema.
//
// Port of the abstract AbstractStructuredType.
type AbstractStructuredType interface {
	AbstractComplexProperty

	// SetNamespace sets the namespace URI of the type.
	SetNamespace(ns string)

	// SetPrefix sets the namespace prefix of the type.
	SetPrefix(pf string)

	// PreferedPrefix returns the prefix the type would rather be written with,
	// spelled the way Java spells it.
	PreferedPrefix() string
}

// StructuredType is the state and the concrete methods of
// AbstractStructuredType, which every structured type embeds.
type StructuredType struct {
	ComplexProperty

	namespace      string
	preferedPrefix string
	prefix         string
}

// InitStructuredType is Java's protected AbstractStructuredType(XMPMetadata,
// String, String, String) constructor.
//
// Java reads the @StructuredType annotation off its own class; the port is
// handed it, because the embedder knows its own and the embedded struct cannot
// ask. An info with an empty namespace stands for Java's null annotation, which
// is what makes the namespaceURI parameter required.
func (s *StructuredType) InitStructuredType(metadata MetadataLike, info StructuredTypeInfo,
	namespaceURI, fieldPrefix, propertyName string) error {
	s.initComplex(metadata, propertyName)
	if info.Namespace != "" {
		// init with annotation
		s.namespace = info.Namespace
		s.preferedPrefix = info.PreferedPrefix
	} else {
		// init with parameters
		if namespaceURI == "" {
			return errBothNamespacesNil
		}
		s.namespace = namespaceURI
		s.preferedPrefix = fieldPrefix
	}
	if fieldPrefix == "" {
		s.prefix = s.preferedPrefix
	} else {
		s.prefix = fieldPrefix
	}
	return nil
}

// Namespace returns the namespace URI of the type.
func (s *StructuredType) Namespace() string { return s.namespace }

// SetNamespace sets the namespace URI of the type.
func (s *StructuredType) SetNamespace(ns string) { s.namespace = ns }

// Prefix returns the namespace prefix of the type.
func (s *StructuredType) Prefix() string { return s.prefix }

// SetPrefix sets the namespace prefix of the type.
func (s *StructuredType) SetPrefix(pf string) { s.prefix = pf }

// PreferedPrefix returns the prefix the type would rather be written with.
func (s *StructuredType) PreferedPrefix() string { return s.preferedPrefix }

// AddSimpleProperty adds a simple field of this type, read from the properties
// the type declares.
//
// Port of the protected addSimpleProperty. Java passes getClass(); the port
// passes the description the embedder registered, since there is no class to
// read the annotations off.
func (s *StructuredType) AddSimpleProperty(properties *PropertiesDescription,
	propertyName string, value any) error {
	tm := s.Metadata().TypeMapping()
	asp, err := tm.InstanciateSimpleField(properties, "", s.Prefix(), propertyName, value)
	if err != nil {
		return err
	}
	s.AddProperty(asp)
	return nil
}

// PropertyValueAsString returns the value of the named field as a string, and
// the empty string where the field is absent or is not simple -- which is
// Java's null.
func (s *StructuredType) PropertyValueAsString(fieldName string) string {
	absProp := s.Property(fieldName)
	if simple, isSimple := absProp.(AbstractSimpleProperty); isSimple {
		return simple.StringValue()
	}
	return ""
}

// DatePropertyAsCalendar returns the value of the named date field, the second
// result being false where there is none -- which is Java's null.
//
// Java answers getValue(), so a field that is there and holds no date is null
// too, not a date at the epoch.
func (s *StructuredType) DatePropertyAsCalendar(fieldName string) (time.Time, bool) {
	absProp := s.FirstEquivalentProperty(fieldName, "DateType")
	if date, isDate := absProp.(*DateType); isDate {
		return date.DateValue()
	}
	return time.Time{}, false
}

// CreateTextType returns a text property in this type's namespace.
func (s *StructuredType) CreateTextType(propertyName, value string) (*TextType, error) {
	return s.Metadata().TypeMapping().CreateText(s.Namespace(), s.Prefix(), propertyName, value)
}

// CreateArrayProperty returns an array property in this type's namespace.
func (s *StructuredType) CreateArrayProperty(propertyName string,
	cardinality Cardinality) *ArrayProperty {
	return s.Metadata().TypeMapping().CreateArrayProperty(
		s.Namespace(), s.Prefix(), propertyName, cardinality)
}

// ArrayProperty is a property holding an RDF array.
//
// Port of ArrayProperty.
type ArrayProperty struct {
	ComplexProperty

	arrayType Cardinality
	namespace string
	prefix    string
}

var _ AbstractComplexProperty = (*ArrayProperty)(nil)

// NewArrayProperty returns an array property of the given cardinality.
func NewArrayProperty(metadata MetadataLike, namespace, prefix, propertyName string,
	cardinality Cardinality) *ArrayProperty {
	a := &ArrayProperty{arrayType: cardinality, namespace: namespace, prefix: prefix}
	a.initComplex(metadata, propertyName)
	a.isArray = true
	return a
}

// ArrayType returns the cardinality of the array.
func (a *ArrayProperty) ArrayType() Cardinality { return a.arrayType }

// ElementsAsString returns every element as its string value.
//
// FIXME this will produce a ClassCastException if the elements are not of type
// AbstractSimpleProperty -- Java's comment. The port skips an element that is
// not simple rather than panicking, because a Go type assertion in a getter has
// no exception to raise; see migration/STATUS.md.
func (a *ArrayProperty) ElementsAsString() []string {
	allProperties := a.Container().AllProperties()
	retval := make([]string, 0, len(allProperties))
	for _, tmp := range allProperties {
		if simple, isSimple := tmp.(AbstractSimpleProperty); isSimple {
			retval = append(retval, simple.StringValue())
		}
	}
	return retval
}

// Namespace returns the namespace URI of the array.
func (a *ArrayProperty) Namespace() string { return a.namespace }

// Prefix returns the namespace prefix of the array.
func (a *ArrayProperty) Prefix() string { return a.prefix }

// TypeName returns the name of the Java class this stands for.
func (a *ArrayProperty) TypeName() string { return "ArrayProperty" }
