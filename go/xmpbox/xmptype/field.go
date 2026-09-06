package xmptype

// The root of the property hierarchy.
//
// Port of org.apache.xmpbox.type.AbstractField.

// MetadataLike is what a field asks of the XMP metadata it belongs to.
//
// Java's AbstractField holds an org.apache.xmpbox.XMPMetadata, which is in the
// root package; that package holds a TypeMapping, which is here, so Java has a
// cycle between the two and Go cannot. The port names what is used and
// xmpbox.XMPMetadata satisfies it.
type MetadataLike interface {
	// TypeMapping returns the mapping from a type name to what implements it.
	TypeMapping() *TypeMapping
}

// AbstractField is the interface every property and field satisfies.
//
// Port of the abstract AbstractField. Java's two abstract methods are the two
// declared last; everything above them is the shared state, which Field holds.
type AbstractField interface {
	// Metadata returns the metadata this field belongs to.
	Metadata() MetadataLike

	// PropertyName returns the name of the property.
	PropertyName() string

	// SetPropertyName sets the name of the property.
	SetPropertyName(value string)

	// SetAttribute sets an attribute of the property, replacing any attribute
	// of the same name.
	SetAttribute(value *Attribute)

	// ContainsAttribute reports whether the property has an attribute of the
	// given name.
	ContainsAttribute(qualifiedName string) bool

	// Attribute returns the attribute of the given name, or nil.
	Attribute(qualifiedName string) *Attribute

	// AllAttributes returns every attribute of the property.
	AllAttributes() []*Attribute

	// RemoveAttribute removes the attribute of the given name.
	RemoveAttribute(qualifiedName string)

	// Namespace returns the namespace URI of the property.
	Namespace() string

	// Prefix returns the namespace prefix of the property.
	Prefix() string

	// TypeName returns the name of the Java class this field stands for.
	//
	// Java compares getClass() and prints getSimpleName() in half a dozen
	// places; the port has no reflection, so each type names itself.
	TypeName() string
}

// Field is the state and the concrete methods of AbstractField, which every
// implementation embeds.
//
// Java declares all of them final, so an implementation adds only Namespace and
// Prefix.
type Field struct {
	metadata     MetadataLike
	propertyName string
	attributes   map[string]*Attribute

	// attributeOrder keeps the order attributes were first set in, because Go
	// map iteration is random and Java's HashMap, however arbitrary its own
	// order, is at least the same on every walk of one map.
	attributeOrder []string
}

// InitField initialises the shared state, which is Java's protected
// AbstractField(XMPMetadata, String) constructor.
func (f *Field) InitField(metadata MetadataLike, propertyName string) {
	f.metadata = metadata
	f.propertyName = propertyName
	f.attributes = map[string]*Attribute{}
	f.attributeOrder = nil
}

// PropertyName returns the name of the property.
func (f *Field) PropertyName() string { return f.propertyName }

// SetPropertyName sets the name of the property.
func (f *Field) SetPropertyName(value string) { f.propertyName = value }

// SetAttribute sets an attribute of the property, replacing any attribute of
// the same name.
func (f *Field) SetAttribute(value *Attribute) {
	if _, replaced := f.attributes[value.Name()]; !replaced {
		f.attributeOrder = append(f.attributeOrder, value.Name())
	}
	f.attributes[value.Name()] = value
}

// ContainsAttribute reports whether the property has an attribute of the given
// name.
func (f *Field) ContainsAttribute(qualifiedName string) bool {
	_, found := f.attributes[qualifiedName]
	return found
}

// Attribute returns the attribute of the given name, or nil.
func (f *Field) Attribute(qualifiedName string) *Attribute {
	return f.attributes[qualifiedName]
}

// AllAttributes returns every attribute of the property, in the order they were
// first set.
func (f *Field) AllAttributes() []*Attribute {
	all := make([]*Attribute, 0, len(f.attributes))
	for _, name := range f.attributeOrder {
		if attribute, found := f.attributes[name]; found {
			all = append(all, attribute)
		}
	}
	return all
}

// RemoveAttribute removes the attribute of the given name.
func (f *Field) RemoveAttribute(qualifiedName string) {
	if _, found := f.attributes[qualifiedName]; !found {
		return
	}
	delete(f.attributes, qualifiedName)
	for i, name := range f.attributeOrder {
		if name == qualifiedName {
			f.attributeOrder = append(f.attributeOrder[:i], f.attributeOrder[i+1:]...)
			break
		}
	}
}

// Metadata returns the metadata this field belongs to.
func (f *Field) Metadata() MetadataLike { return f.metadata }
