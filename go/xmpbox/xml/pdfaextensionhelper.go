package xml

// Reading a PDF/A extension schema, which declares namespaces and types the
// module does not know beforehand.
//
// Port of PdfaExtensionHelper.

import (
	"fmt"
	"strings"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// The four ways a value type names a choice.
const (
	closedChoice  = "closed Choice of "
	closedChoiceU = "Closed Choice of "
	openChoice    = "open Choice of "
	openChoiceU   = "Open Choice of "
)

// pdfaNamespaceDeclarations is the prefix and the namespace of each of the five
// PDF/A extension types, which validateNaming insists go together.
//
// Java reads them off each class's @StructuredType annotation; the port has no
// reflection, so it names them.
var pdfaNamespaceDeclarations = [...]struct{ prefix, namespace string }{
	{schema.PDFAExtensionPreferedPrefix, schema.PDFAExtensionNamespace},
	{xmptype.PDFAField.StructuredTypeInfo().PreferedPrefix,
		xmptype.PDFAField.StructuredTypeInfo().Namespace},
	{xmptype.PDFAProperty.StructuredTypeInfo().PreferedPrefix,
		xmptype.PDFAProperty.StructuredTypeInfo().Namespace},
	{xmptype.PDFASchema.StructuredTypeInfo().PreferedPrefix,
		xmptype.PDFASchema.StructuredTypeInfo().Namespace},
	{xmptype.PDFAType.StructuredTypeInfo().PreferedPrefix,
		xmptype.PDFAType.StructuredTypeInfo().Namespace},
}

// validateNaming checks that every namespace declaration on the description
// binds a PDF/A prefix to the namespace that goes with it.
//
// Port of PdfaExtensionHelper.validateNaming(XMPMetadata, Element).
func validateNaming(description *Element) error {
	for _, attribute := range description.Attributes() {
		for _, declaration := range pdfaNamespaceDeclarations {
			if err := checkNamespaceDeclaration(attribute,
				declaration.prefix, declaration.namespace); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkNamespaceDeclaration reports a prefix bound to the wrong namespace, or a
// namespace bound to the wrong prefix.
//
// Port of checkNamespaceDeclaration(Attr, Class).
func checkNamespaceDeclaration(attribute *Attr, cprefix, cnamespace string) error {
	if attribute.Prefix() == "" {
		// PDFBOX-6136: not relevant here
		return nil
	}
	prefix := attribute.LocalName()
	namespace := attribute.Value()
	// check extension
	if cprefix == prefix && cnamespace != namespace {
		return NewXmpParsingError(InvalidPdfaSchema,
			"Invalid PDF/A namespace definition, prefix: "+prefix+", namespace: "+namespace)
	}
	if cnamespace == namespace && cprefix != prefix {
		return NewXmpParsingError(InvalidPdfaSchema,
			"Invalid PDF/A namespace definition, prefix: "+prefix+", namespace: "+namespace)
	}
	return nil
}

// populateSchemaMapping teaches the type mapping every namespace and type the
// extension schemas describe.
//
// Port of PdfaExtensionHelper.populateSchemaMapping(XMPMetadata, boolean).
func populateSchemaMapping(meta *xmpbox.XMPMetadata, strictParsing bool) error {
	tm := meta.TypeMapping()
	for _, xmpSchema := range meta.AllSchemas() {
		if xmpSchema.Namespace() != schema.PDFAExtensionNamespace {
			continue
		}
		// ensure the prefix is the preferred one (cannot use other definition)
		if xmpSchema.Prefix() != schema.PDFAExtensionPreferedPrefix {
			return NewXmpParsingError(InvalidPrefix,
				"Found invalid prefix for PDF/A extension, found '"+xmpSchema.Prefix()+
					"', should be '"+schema.PDFAExtensionPreferedPrefix+"'")
		}
		// create schema and types
		pes, isExtension := xmpSchema.(*schema.PDFAExtensionSchema)
		if !isExtension {
			continue
		}
		sp := pes.SchemasProperty()
		for _, af := range sp.AllProperties() {
			if st, isSchemaType := af.(*xmptype.PDFASchemaType); isSchemaType {
				if err := populatePDFASchemaType(meta, st, tm, strictParsing); err != nil {
					return err
				}
			} // TODO unmanaged ?
		}
	}
	return nil
}

// populatePDFASchemaType adds one described schema, its types and its
// properties.
func populatePDFASchemaType(meta *xmpbox.XMPMetadata, st *xmptype.PDFASchemaType,
	tm *xmptype.TypeMapping, strictParsing bool) error {
	namespaceURI := st.NamespaceURI()
	// PDFBOX-5525
	if err := requirePresent(&st.StructuredType, xmptype.PDFASchemaNamespaceURI,
		"Missing pdfaSchema:namespaceURI in type definition"); err != nil {
		return err
	}
	namespaceURI = strings.TrimSpace(namespaceURI)
	prefix := st.PrefixValue()
	properties := st.PropertyArray()
	valueTypes := st.ValueType()
	xsf := tm.SchemaFactory(namespaceURI)
	// retrieve namespaces
	if xsf == nil {
		// create namespace with no field
		tm.AddNewNameSpace(namespaceURI, prefix)
		xsf = tm.SchemaFactory(namespaceURI)
	}
	// populate value type
	if valueTypes != nil {
		for _, af2 := range valueTypes.AllProperties() {
			if tt, isTypeType := af2.(*xmptype.PDFATypeType); isTypeType {
				if err := populatePDFAType(meta, tt, tm); err != nil {
					return err
				}
			}
		}
	}
	// populate properties
	if properties == nil && !strictParsing {
		return nil
	}
	if properties == nil {
		return NewXmpParsingError(RequiredProperty,
			"Missing pdfaSchema:property in type definition")
	}
	for _, af2 := range properties.AllProperties() {
		if pt, isPropertyType := af2.(*xmptype.PDFAPropertyType); isPropertyType {
			if err := populatePDFAPropertyType(pt, tm, xsf); err != nil {
				return err
			}
		} // TODO unmanaged ?
	}
	return nil
}

// populatePDFAPropertyType adds one described property to the schema factory.
func populatePDFAPropertyType(property *xmptype.PDFAPropertyType, tm *xmptype.TypeMapping,
	xsf xmptype.SchemaFactoryLike) error {
	pname := property.Name()
	ptype := property.ValueType()
	// check all mandatory fields are OK
	for _, required := range []string{
		xmptype.PDFAPropertyName,
		xmptype.PDFAPropertyValueType,
		xmptype.PDFAPropertyDescription,
		xmptype.PDFAPropertyCategory,
	} {
		if err := requirePresent(&property.StructuredType, required,
			fmt.Sprintf("Missing field '%s' in property definition", required)); err != nil {
			return err
		}
	}

	// check ptype existence
	pt, hasType, known := transformValueType(tm, ptype)
	if !known {
		return NewXmpParsingError(NoValueType, "Unknown property value type : "+ptype)
	}
	if !hasType {
		return NewXmpParsingError(NoValueType, "Type not defined : "+ptype)
	}
	if pt.Type.IsSimple() || pt.Type.IsStructured() || pt.Type == xmptype.DefinedType {
		xsf.PropertyDefinition().AddNewProperty(pname, pt)
		return nil
	}
	return NewXmpParsingError(NoValueType, "Type not defined : "+ptype)
}

// populatePDFAType adds one described value type to the type mapping.
func populatePDFAType(meta *xmpbox.XMPMetadata, valueType *xmptype.PDFATypeType,
	tm *xmptype.TypeMapping) error {
	ttype := valueType.Type()
	tns := valueType.NamespaceURI()
	tprefix := valueType.PrefixValue()
	// all fields are mandatory
	for _, required := range []string{
		xmptype.PDFATypeType_,
		xmptype.PDFATypeNSURI,
		xmptype.PDFATypePrefix,
		xmptype.PDFATypeDescription,
	} {
		if err := requirePresent(&valueType.StructuredType, required,
			fmt.Sprintf("Missing field '%s' in type definition", required)); err != nil {
			return err
		}
	}

	// create the structured type
	structuredType := xmptype.NewDefinedStructuredType(meta, tns, tprefix, "") // TODO
	// maybe a name exists
	if fields := valueType.Fields(); fields != nil {
		for _, af3 := range fields.AllProperties() {
			if ft, isFieldType := af3.(*xmptype.PDFAFieldType); isFieldType {
				if err := populatePDFAFieldType(ft, structuredType); err != nil {
					return err
				}
			}
			// else TODO
		}
	}
	// add the structured type to list
	pm := xmptype.NewPropertiesDescription()
	defined := structuredType.DefinedProperties()
	for _, name := range structuredType.DefinedPropertyNames() {
		pm.AddNewProperty(name, defined[name])
	}
	tm.AddToDefinedStructuredTypes(ttype, tns, pm)
	return nil
}

// populatePDFAFieldType adds one described field to the structured type.
func populatePDFAFieldType(field *xmptype.PDFAFieldType,
	structuredType *xmptype.DefinedStructuredType) error {
	fName := field.Name()
	fValueType := field.ValueType()
	for _, required := range []string{
		xmptype.PDFAFieldName,
		xmptype.PDFAFieldDescription,
		xmptype.PDFAFieldValueType,
	} {
		if err := requirePresent(&field.StructuredType, required,
			fmt.Sprintf("Missing field '%s' in field definition", required)); err != nil {
			return err
		}
	}

	fValue, named := xmptype.TypeOfName(fValueType)
	if !named {
		// Java catches the IllegalArgumentException Types.valueOf raises here.
		// TODO could fValueType be a structured type ?
		return NewXmpParsingError(NoValueType, "Type not defined : "+fValueType)
	}
	structuredType.AddDefinedProperty(fName,
		xmptype.CreatePropertyType(fValue, xmptype.Simple))
	return nil
}

// transformValueType reads the type and the cardinality out of the string a
// PDF/A extension schema writes a value type as.
//
// The last result is false where the string names a cardinality the module does
// not know, which is Java's null return; the second is false where it names no
// type, which is the null type inside the PropertyType Java built.
//
// Port of transformValueType(TypeMapping, String).
func transformValueType(tm *xmptype.TypeMapping,
	valueType string) (propertyType xmptype.PropertyType, hasType, known bool) {
	if valueType == "Lang Alt" {
		return xmptype.CreatePropertyType(xmptype.LangAlt, xmptype.Simple), true, true
	}
	// else all other cases
	switch {
	case strings.HasPrefix(valueType, closedChoice) || strings.HasPrefix(valueType, closedChoiceU):
		valueType = valueType[len(closedChoice):]
	case strings.HasPrefix(valueType, openChoice) || strings.HasPrefix(valueType, openChoiceU):
		valueType = valueType[len(openChoice):]
	}
	pos := strings.IndexByte(valueType, ' ')
	card := xmptype.Simple
	if pos > 0 {
		switch strings.ToLower(valueType[:pos]) {
		case "seq":
			card = xmptype.Seq
		case "bag":
			card = xmptype.Bag
		case "alt":
			card = xmptype.Alt
		default:
			return xmptype.PropertyType{}, false, false
		}
	}
	vt := valueType[pos+1:]
	name := vt
	if pos < 0 {
		name = valueType
	}
	valued, named := xmptype.TypeOfName(name)
	if !named && tm.IsDefinedType(vt) {
		valued, named = xmptype.DefinedType, true
	}
	if !named {
		// Java leaves the type null and builds a PropertyType around it, which
		// the caller then reports as "Type not defined".
		return xmptype.PropertyType{Card: card}, false, true
	}
	return xmptype.CreatePropertyType(valued, card), true, true
}

// requirePresent reports a field a described schema, type, property or field
// leaves out.
//
// Java reads the value and tests it for null; the port tests whether the
// property is there, because an absent property and one holding the empty
// string both read back as the empty string.
func requirePresent(structured *xmptype.StructuredType, fieldName, message string) error {
	if structured.Property(fieldName) == nil {
		return NewXmpParsingError(RequiredProperty, message)
	}
	return nil
}
