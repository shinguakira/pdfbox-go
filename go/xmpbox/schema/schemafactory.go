package schema

// What a schema is, and how one is made for a namespace.
//
// Port of org.apache.xmpbox.schema.XMPSchemaFactory and XmpSchemaException.

import (
	"errors"
	"fmt"

	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// ErrXmpSchema reports a schema that could not be made or is not what was
// expected.
//
// Port of XmpSchemaException, which Java declares as a checked Exception.
var ErrXmpSchema = errors.New("schema: xmp schema")

// Schema is what a piece of metadata keeps a list of.
//
// Java keeps a List<XMPSchema> and narrows an element back to the concrete
// schema with a cast. Go cannot narrow an embedded struct, so a schema answers
// the XMPSchema it embeds through Base and a caller type-asserts the interface
// to the concrete schema.
type Schema interface {
	xmptype.AbstractStructuredType

	// Base returns the XMPSchema every schema is built on, which carries the
	// property accessors.
	Base() *XMPSchema
}

// Base returns the schema itself, so that XMPSchema is a Schema.
func (s *XMPSchema) Base() *XMPSchema { return s }

// MetadataHolder is what a factory adds the schema it made to.
//
// Java's XMPSchemaFactory calls metadata.addSchema, which is in the root
// package; that package imports this one for the schemas, so the port names
// what is used and xmpbox.XMPMetadata satisfies it.
type MetadataHolder interface {
	xmptype.MetadataLike

	// AddSchema adds a schema to the metadata.
	AddSchema(schema Schema)
}

// SchemaConstructor makes one schema of a known class.
//
// Port of what XMPSchemaFactory reaches through
// schemaClass.getDeclaredConstructor(...).newInstance(...). Java picks one of
// three constructors by the class and the prefix; each constructor here takes
// the prefix and ignores it where the Java would have called the one-argument
// form.
type SchemaConstructor func(metadata xmptype.MetadataLike, prefix string) (Schema, error)

// XMPSchemaFactory makes the schema of one namespace.
//
// Port of XMPSchemaFactory.
type XMPSchemaFactory struct {
	namespace   string
	construct   SchemaConstructor
	propDef     *xmptype.PropertiesDescription
	isBaseClass bool
}

var _ xmptype.SchemaFactoryLike = (*XMPSchemaFactory)(nil)

// NewXMPSchemaFactory returns the factory of a namespace whose schema has a
// class of its own.
//
// Port of XMPSchemaFactory(String, Class, PropertiesDescription).
func NewXMPSchemaFactory(namespace string, construct SchemaConstructor,
	propDef *xmptype.PropertiesDescription) *XMPSchemaFactory {
	return &XMPSchemaFactory{namespace: namespace, construct: construct, propDef: propDef}
}

// newBaseSchemaFactory returns the factory of a namespace with no schema class
// of its own, which Java writes as `new XMPSchemaFactory(ns, XMPSchema.class,
// mapping)`.
func newBaseSchemaFactory(namespace string,
	propDef *xmptype.PropertiesDescription) *XMPSchemaFactory {
	return &XMPSchemaFactory{namespace: namespace, propDef: propDef, isBaseClass: true}
}

// Namespace returns the namespace the factory makes schemas for.
func (f *XMPSchemaFactory) Namespace() string { return f.namespace }

// PropertyType returns the type of the named property, the second result being
// false where the schema does not declare it.
func (f *XMPSchemaFactory) PropertyType(name string) (xmptype.PropertyType, bool) {
	return f.propDef.PropertyType(name)
}

// PropertyDefinition returns the properties the schema declares.
func (f *XMPSchemaFactory) PropertyDefinition() *xmptype.PropertiesDescription {
	return f.propDef
}

// CreateXMPSchema makes the schema and adds it to the metadata.
//
// Port of createXMPSchema(XMPMetadata, String).
func (f *XMPSchemaFactory) CreateXMPSchema(metadata MetadataHolder,
	prefix string) (Schema, error) {
	var schema Schema
	var err error
	if f.isBaseClass {
		schema, err = NewXMPSchemaOfNamespace(metadata, f.namespace, prefix)
	} else {
		schema, err = f.construct(metadata, prefix)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: Cannot instantiate specified object schema: %w",
			ErrXmpSchema, err)
	}
	metadata.AddSchema(schema)
	return schema, nil
}

// init registers the twelve namespaces TypeMapping.initialize adds, and the
// constructor a namespace with no schema class of its own is made with.
//
// Java names the twelve in TypeMapping, which puts org.apache.xmpbox.type in a
// cycle with org.apache.xmpbox.schema; the port has this package push them
// instead. See xmptype.RegisterDefaultSchema.
func init() {
	xmptype.NewDefaultSchemaFactory = func(namespace string,
		properties *xmptype.PropertiesDescription) xmptype.SchemaFactoryLike {
		return newBaseSchemaFactory(namespace, properties)
	}

	for _, factory := range []*XMPSchemaFactory{
		NewXMPSchemaFactory(xmpBasicInfo.Namespace, newXMPBasicSchemaAs, xmpBasicProperties),
		NewXMPSchemaFactory(dublinCoreInfo.Namespace, newDublinCoreSchemaAs, dublinCoreProperties),
		NewXMPSchemaFactory(pdfaExtensionInfo.Namespace, newPDFAExtensionSchemaAs,
			pdfaExtensionProperties),
		NewXMPSchemaFactory(mediaManagementInfo.Namespace, newXMPMediaManagementSchemaAs,
			mediaManagementProperties),
		NewXMPSchemaFactory(adobePDFInfo.Namespace, newAdobePDFSchemaAs, adobePDFProperties),
		NewXMPSchemaFactory(pdfaIdentificationInfo.Namespace, newPDFAIdentificationSchemaAs,
			pdfaIdentificationProperties),
		NewXMPSchemaFactory(rightsManagementInfo.Namespace, newXMPRightsManagementSchemaAs,
			rightsManagementProperties),
		NewXMPSchemaFactory(photoshopInfo.Namespace, newPhotoshopSchemaAs, photoshopProperties),
		NewXMPSchemaFactory(jobTicketInfo.Namespace, newXMPBasicJobTicketSchemaAs,
			jobTicketProperties),
		NewXMPSchemaFactory(exifInfo.Namespace, newExifSchemaAs, exifProperties),
		NewXMPSchemaFactory(tiffInfo.Namespace, newTiffSchemaAs, tiffProperties),
		NewXMPSchemaFactory(pageTextInfo.Namespace, newXMPPageTextSchemaAs, pageTextProperties),
	} {
		xmptype.RegisterDefaultSchema(factory)
	}
}
