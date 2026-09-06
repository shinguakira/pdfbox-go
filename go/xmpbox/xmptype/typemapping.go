package xmptype

// The mapping from a type name to the thing that implements it.
//
// Port of org.apache.xmpbox.type.TypeMapping, which Java declares final.
//
// Java holds a Class per type and instantiates it by reflection; the port holds
// the constructor each type registered, and everything getImplementingClass
// reached goes through Types instead.
//
// Java also names the twelve schema classes here, which is the one place
// org.apache.xmpbox.type imports org.apache.xmpbox.schema while schema imports
// type back. Go forbids the cycle, so xmpbox/schema registers its factories
// through RegisterDefaultSchema from its own init and this package names only
// the shape it uses.

import "fmt"

// SchemaFactoryLike is what a TypeMapping keeps per namespace.
//
// Java's XMPSchemaFactory lives in xmpbox/schema, which imports this package;
// the port names what is used here and schema.XMPSchemaFactory satisfies it.
type SchemaFactoryLike interface {
	// Namespace returns the namespace the factory makes schemas for.
	Namespace() string

	// PropertyDefinition returns the properties the schema declares.
	PropertyDefinition() *PropertiesDescription
}

// SchemaFactoryConstructor makes a factory for a namespace whose schema is not
// one of the twelve, which Java writes as `new XMPSchemaFactory(ns,
// XMPSchema.class, mapping)`.
//
// xmpbox/schema sets it from its init. A program that never links that package
// gets no schema factories, which is the same "not linked in" the port's other
// registries have; see migration/STATUS.md.
var NewDefaultSchemaFactory func(namespace string,
	properties *PropertiesDescription) SchemaFactoryLike

// defaultSchemaFactories are the twelve namespaces Java's initialize registers,
// which xmpbox/schema adds from its init.
var defaultSchemaFactories []SchemaFactoryLike

// RegisterDefaultSchema records one of the schema factories every TypeMapping
// starts with.
//
// Port of the twelve addNameSpace calls in TypeMapping.initialize.
func RegisterDefaultSchema(factory SchemaFactoryLike) {
	defaultSchemaFactories = append(defaultSchemaFactories, factory)
}

// TypeMapping is what a piece of metadata reads its types through.
type TypeMapping struct {
	// type -> property
	structuredMappings map[Types]*PropertiesDescription

	// ns -> list of property descriptions
	definedStructuredNamespaces2 map[string][]*PropertiesDescription

	// typeName -> property
	definedStructuredMappings map[string]*PropertiesDescription

	// ns -> type
	// filled during init
	structuredNamespaces2 map[string][]Types

	metadata MetadataLike

	schemaMap map[string]SchemaFactoryLike
}

// NewTypeMapping returns the mapping of the given metadata.
func NewTypeMapping(metadata MetadataLike) *TypeMapping {
	t := &TypeMapping{metadata: metadata}
	t.initialize()
	return t
}

// initialize fills the four tables and the schema map.
func (t *TypeMapping) initialize() {
	// structured types
	t.structuredMappings = map[Types]*PropertiesDescription{}
	t.structuredNamespaces2 = map[string][]Types{}
	for _, structuredType := range AllTypes() {
		if !structuredType.IsStructured() {
			continue
		}
		st := structuredType.StructuredTypeInfo()
		ns := st.Namespace
		pm := structuredType.StructuredProperties()
		t.structuredNamespaces2[ns] = append(t.structuredNamespaces2[ns], structuredType)
		t.structuredMappings[structuredType] = pm
	}

	// define structured types
	t.definedStructuredMappings = map[string]*PropertiesDescription{}
	t.definedStructuredNamespaces2 = map[string][]*PropertiesDescription{}

	// schema
	t.schemaMap = map[string]SchemaFactoryLike{}
	for _, factory := range defaultSchemaFactories {
		t.schemaMap[factory.Namespace()] = factory
	}
}

// AddToDefinedStructuredTypes records a type a PDF/A extension schema defines.
func (t *TypeMapping) AddToDefinedStructuredTypes(typeName, ns string,
	pm *PropertiesDescription) {
	t.definedStructuredNamespaces2[ns] = append(t.definedStructuredNamespaces2[ns], pm)
	t.definedStructuredMappings[typeName] = pm
}

// DefinedDescriptionByNamespace returns the defined type of the given namespace
// that declares the given field, or nil.
func (t *TypeMapping) DefinedDescriptionByNamespace(namespace,
	pdfaFieldName string) *PropertiesDescription {
	propDescList := t.definedStructuredNamespaces2[namespace]
	for _, propDesc := range propDescList {
		// check whether one of these field names matches
		for _, name := range propDesc.PropertiesNames() {
			if name == pdfaFieldName {
				return propDesc
			}
		}
	}
	return nil
}

// InstanciateStructuredType returns a new value of the given structured type,
// under the given property name.
func (t *TypeMapping) InstanciateStructuredType(structuredType Types,
	propertyName string) (AbstractStructuredType, error) {
	construct := structuredType.StructuredConstructor()
	if construct == nil {
		return nil, fmt.Errorf("%w: Failed to instantiate structured type : %v",
			ErrBadFieldValue, structuredType)
	}
	tmp := construct(t.metadata)
	tmp.SetPropertyName(propertyName)
	return tmp, nil
}

// InstanciateDefinedType returns a new value of a type a PDF/A extension schema
// defines.
func (t *TypeMapping) InstanciateDefinedType(propertyName,
	namespace string) AbstractStructuredType {
	return NewDefinedStructuredType(t.metadata, namespace, "", propertyName)
}

// InstanciateSimpleProperty returns a new simple property of the given type.
func (t *TypeMapping) InstanciateSimpleProperty(nsuri, prefix, name string, value any,
	simpleType Types) (AbstractSimpleProperty, error) {
	construct := simpleType.SimpleConstructor()
	if construct == nil {
		return nil, fmt.Errorf("Failed to instantiate %v property with value '%v'",
			simpleType, value)
	}
	property, err := construct(t.metadata, nsuri, prefix, name, value)
	if err != nil {
		return nil, fmt.Errorf("Failed to instantiate %v property with value '%v': %w",
			simpleType, value, err)
	}
	return property, nil
}

// InstanciateSimpleField returns a new simple property of the type the given
// description gives the named property.
//
// Port of instanciateSimpleField(Class, ...). Java reads the annotations off the
// class it is handed; the port is handed the description that was built from
// them.
func (t *TypeMapping) InstanciateSimpleField(pm *PropertiesDescription,
	nsuri, prefix, propertyName string, value any) (AbstractSimpleProperty, error) {
	simpleType, found := pm.PropertyType(propertyName)
	if !found {
		return nil, fmt.Errorf("Failed to instantiate property '%s' with value '%v'",
			propertyName, value)
	}
	return t.InstanciateSimpleProperty(nsuri, prefix, propertyName, value, simpleType.Type)
}

// IsStructuredTypeNamespace reports whether the namespace is one a structured
// type declares.
func (t *TypeMapping) IsStructuredTypeNamespace(namespace string) bool {
	_, found := t.structuredNamespaces2[namespace]
	return found
}

// IsDefinedTypeNamespace reports whether the namespace is one a PDF/A extension
// schema defined a type in.
func (t *TypeMapping) IsDefinedTypeNamespace(namespace string) bool {
	_, found := t.definedStructuredNamespaces2[namespace]
	return found
}

// IsDefinedType reports whether the name is one a PDF/A extension schema
// defined.
func (t *TypeMapping) IsDefinedType(name string) bool {
	_, found := t.definedStructuredMappings[name]
	return found
}

// AddNewNameSpace records a namespace whose schema is not one of the twelve.
func (t *TypeMapping) AddNewNameSpace(ns, preferred string) {
	if NewDefaultSchemaFactory == nil {
		// xmpbox/schema is not linked in, so there is nothing to make a factory
		// with. See migration/STATUS.md.
		return
	}
	mapping := NewPropertiesDescription()
	t.schemaMap[ns] = NewDefaultSchemaFactory(ns, mapping)
}

// StructuredPropMapping returns the properties the given structured type
// declares.
func (t *TypeMapping) StructuredPropMapping(structuredType Types) *PropertiesDescription {
	return t.structuredMappings[structuredType]
}

// SchemaFactory returns the factory of the given namespace, or nil.
func (t *TypeMapping) SchemaFactory(namespace string) SchemaFactoryLike {
	return t.schemaMap[namespace]
}

// IsDefinedSchema reports whether the namespace has a schema factory.
func (t *TypeMapping) IsDefinedSchema(namespace string) bool {
	_, found := t.schemaMap[namespace]
	return found
}

// IsDefinedNamespace reports whether the namespace is known at all.
func (t *TypeMapping) IsDefinedNamespace(namespace string) bool {
	return t.IsDefinedSchema(namespace) || t.IsStructuredTypeNamespace(namespace) ||
		t.IsDefinedTypeNamespace(namespace)
}
