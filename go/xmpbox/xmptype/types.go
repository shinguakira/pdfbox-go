package xmptype

// The XMP types, and what implements each of them.
//
// Port of the enum org.apache.xmpbox.type.Types.
//
// Java's constant carries a Class and instantiates it by reflection. The port
// has none, so a constant carries a constructor function instead, registered
// from the file that declares the type it makes: a simple type registers a
// SimpleConstructor and a structured type a StructuredConstructor. Everything
// Java does with getImplementingClass goes through those two registries.

import "fmt"

// Types is the type of an XMP property.
type Types int

// The types, in the order Java declares them.
const (
	// Structured is the base of every structured type.
	Structured Types = iota
	// DefinedType is a type a PDF/A extension schema defines.
	DefinedType

	// basic
	Text
	Date
	Boolean
	Integer
	Real
	GPSCoordinate
	ProperName
	Locale
	AgentName
	GUID
	XPath
	Part
	URL
	URI
	Choice
	MIMEType
	LangAlt
	RenditionClass
	Rational
	Colorant
	Font
	Layer
	Thumbnail
	ResourceEvent
	ResourceRef
	Version
	PDFASchema
	PDFAField
	PDFAProperty
	PDFAType
	Job
	OECF
	CFAPattern
	DeviceSettings
	Flash
	Dimensions

	// typesCount is one past the last type, which bounds the tables below.
	typesCount
)

// typeInfo is the three values a Java constant carries.
type typeInfo struct {
	name   string
	simple bool
	basic  Types

	// hasBasic stands for Java's null basic, which isBasic tests for.
	hasBasic bool
}

// typeInfos is the constant table, in declaration order.
var typeInfos = [typesCount]typeInfo{
	Structured:  {name: "Structured"},
	DefinedType: {name: "DefinedType"},

	Text:    {name: "Text", simple: true},
	Date:    {name: "Date", simple: true},
	Boolean: {name: "Boolean", simple: true},
	Integer: {name: "Integer", simple: true},
	Real:    {name: "Real", simple: true},

	GPSCoordinate:  {name: "GPSCoordinate", simple: true, basic: Text, hasBasic: true},
	ProperName:     {name: "ProperName", simple: true, basic: Text, hasBasic: true},
	Locale:         {name: "Locale", simple: true, basic: Text, hasBasic: true},
	AgentName:      {name: "AgentName", simple: true, basic: Text, hasBasic: true},
	GUID:           {name: "GUID", simple: true, basic: Text, hasBasic: true},
	XPath:          {name: "XPath", simple: true, basic: Text, hasBasic: true},
	Part:           {name: "Part", simple: true, basic: Text, hasBasic: true},
	URL:            {name: "URL", simple: true, basic: Text, hasBasic: true},
	URI:            {name: "URI", simple: true, basic: Text, hasBasic: true},
	Choice:         {name: "Choice", simple: true, basic: Text, hasBasic: true},
	MIMEType:       {name: "MIMEType", simple: true, basic: Text, hasBasic: true},
	LangAlt:        {name: "LangAlt", simple: true, basic: Text, hasBasic: true},
	RenditionClass: {name: "RenditionClass", simple: true, basic: Text, hasBasic: true},
	Rational:       {name: "Rational", simple: true, basic: Text, hasBasic: true},

	Colorant:       {name: "Colorant", basic: Structured, hasBasic: true},
	Font:           {name: "Font", basic: Structured, hasBasic: true},
	Layer:          {name: "Layer", basic: Structured, hasBasic: true},
	Thumbnail:      {name: "Thumbnail", basic: Structured, hasBasic: true},
	ResourceEvent:  {name: "ResourceEvent", basic: Structured, hasBasic: true},
	ResourceRef:    {name: "ResourceRef", basic: Structured, hasBasic: true},
	Version:        {name: "Version", basic: Structured, hasBasic: true},
	PDFASchema:     {name: "PDFASchema", basic: Structured, hasBasic: true},
	PDFAField:      {name: "PDFAField", basic: Structured, hasBasic: true},
	PDFAProperty:   {name: "PDFAProperty", basic: Structured, hasBasic: true},
	PDFAType:       {name: "PDFAType", basic: Structured, hasBasic: true},
	Job:            {name: "Job", basic: Structured, hasBasic: true},
	OECF:           {name: "OECF", basic: Structured, hasBasic: true},
	CFAPattern:     {name: "CFAPattern", basic: Structured, hasBasic: true},
	DeviceSettings: {name: "DeviceSettings", basic: Structured, hasBasic: true},
	Flash:          {name: "Flash", basic: Structured, hasBasic: true},
	Dimensions:     {name: "Dimensions", basic: Structured, hasBasic: true},
}

// SimpleConstructor makes one of the simple properties.
//
// Port of what Java reaches through
// getImplementingClass().getDeclaredConstructor(XMPMetadata.class, String.class,
// String.class, String.class, Object.class).
type SimpleConstructor func(metadata MetadataLike, namespaceURI, prefix, propertyName string,
	value any) (AbstractSimpleProperty, error)

// StructuredConstructor makes one of the structured types.
//
// Port of what Java reaches through
// getImplementingClass().getDeclaredConstructor(XMPMetadata.class).
type StructuredConstructor func(metadata MetadataLike) AbstractStructuredType

// simpleConstructors and structuredConstructors stand for the Class each Java
// constant carries. The file that declares a type registers it from its init,
// so that a type and the constant naming it stay together.
var (
	simpleConstructors     [typesCount]SimpleConstructor
	structuredConstructors [typesCount]StructuredConstructor
)

// RegisterSimpleType records the constructor of a simple type.
func RegisterSimpleType(t Types, construct SimpleConstructor) {
	simpleConstructors[t] = construct
}

// RegisterStructuredType records the constructor and the structured type
// annotation of a structured type.
func RegisterStructuredType(t Types, info StructuredTypeInfo, properties *PropertiesDescription,
	construct StructuredConstructor) {
	structuredConstructors[t] = construct
	structuredTypeInfos[t] = info
	structuredProperties[t] = properties
}

// structuredTypeInfos and structuredProperties are the @StructuredType
// annotation and the @PropertyType fields of each structured type, which Java
// reads off the class.
var (
	structuredTypeInfos  [typesCount]StructuredTypeInfo
	structuredProperties [typesCount]*PropertiesDescription
)

// IsSimple reports whether the type holds a single value.
func (t Types) IsSimple() bool { return typeInfos[t].simple }

// IsBasic reports whether the type is one of the five the specification builds
// everything else out of, which Java writes as a null basic.
func (t Types) IsBasic() bool { return !typeInfos[t].hasBasic }

// IsStructured reports whether the type is a structured one.
func (t Types) IsStructured() bool {
	return typeInfos[t].hasBasic && typeInfos[t].basic == Structured
}

// IsDefined reports whether the type is one a PDF/A extension schema defines.
func (t Types) IsDefined() bool { return t == DefinedType }

// Basic returns the type this one is built on, the second result being false
// where there is none -- which is Java's null.
func (t Types) Basic() (Types, bool) {
	return typeInfos[t].basic, typeInfos[t].hasBasic
}

// SimpleConstructor returns the constructor of a simple type, or nil.
//
// Port of getImplementingClass() for the simple half.
func (t Types) SimpleConstructor() SimpleConstructor { return simpleConstructors[t] }

// StructuredConstructor returns the constructor of a structured type, or nil.
//
// Port of getImplementingClass() for the structured half.
func (t Types) StructuredConstructor() StructuredConstructor { return structuredConstructors[t] }

// StructuredTypeInfo returns the @StructuredType annotation of a structured
// type.
func (t Types) StructuredTypeInfo() StructuredTypeInfo { return structuredTypeInfos[t] }

// ImplementingClassName returns the simple name of the class that implements
// the type, which is what the type mapping puts in its messages.
//
// Port of getImplementingClass().getSimpleName(). Every type but three names a
// class spelled as the constant plus "Type": GPSCoordinate and LangAlt are
// carried by a plain TextType, and MIMEType's class is MIMEType itself.
func (t Types) ImplementingClassName() string {
	switch t {
	case Structured, DefinedType:
		return ""
	case GPSCoordinate, LangAlt:
		return "TextType"
	case MIMEType:
		return "MIMEType"
	}
	return t.String() + "Type"
}

// StructuredProperties returns the properties a structured type declares.
func (t Types) StructuredProperties() *PropertiesDescription { return structuredProperties[t] }

// String returns the enum constant's name, which is Java's Enum.toString and
// which the type mapping reads a type name back from.
func (t Types) String() string {
	if t < 0 || t >= typesCount {
		return fmt.Sprintf("Types(%d)", int(t))
	}
	return typeInfos[t].name
}

// TypeOfName returns the type of the given name, the second result being false
// where there is none -- which is Java's IllegalArgumentException out of
// Types.valueOf.
func TypeOfName(name string) (Types, bool) {
	for t := Types(0); t < typesCount; t++ {
		if typeInfos[t].name == name {
			return t, true
		}
	}
	return 0, false
}

// AllTypes returns every type, which is Java's Types.values().
func AllTypes() []Types {
	all := make([]Types, 0, typesCount)
	for t := Types(0); t < typesCount; t++ {
		all = append(all, t)
	}
	return all
}
