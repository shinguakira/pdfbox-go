package xmpbox

// The schemas a piece of metadata carries.
//
// The other half of the XMPMetadata port; xmpmetadata.go holds the xpacket
// declaration and the type mapping.

import (
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
)

var _ schema.MetadataHolder = (*XMPMetadata)(nil)

// AllSchemas returns the schemas declared in this metadata.
//
// Java copies the list so that a caller cannot change it through the result.
func (m *XMPMetadata) AllSchemas() []schema.Schema {
	return append([]schema.Schema(nil), m.schemas...)
}

// AddSchema adds a schema to this metadata.
func (m *XMPMetadata) AddSchema(s schema.Schema) { m.schemas = append(m.schemas, s) }

// RemoveSchema removes a schema.
//
// Java's List.remove takes out the first element that equals the argument;
// XMPSchema does not override equals, so that is the first identical one.
func (m *XMPMetadata) RemoveSchema(s schema.Schema) {
	for i, declared := range m.schemas {
		if declared == s {
			m.schemas = append(m.schemas[:i], m.schemas[i+1:]...)
			return
		}
	}
}

// ClearSchemas removes every schema declared.
func (m *XMPMetadata) ClearSchemas() { m.schemas = m.schemas[:0] }

// Schema returns the first schema of the given namespace, and nil where there
// is none.
//
// Metadata should carry one schema per namespace; where it carries several,
// this answers the first and SchemaOfPrefix picks a particular one.
func (m *XMPMetadata) Schema(nsURI string) schema.Schema {
	for _, declared := range m.schemas {
		if declared.Namespace() == nsURI {
			return declared
		}
	}
	return nil
}

// SchemaOfPrefix returns the first schema of the given namespace written with
// the given prefix, and nil where there is none.
func (m *XMPMetadata) SchemaOfPrefix(prefix, nsURI string) schema.Schema {
	for _, declared := range m.AllSchemas() {
		if declared.Namespace() == nsURI && declared.Prefix() == prefix {
			return declared
		}
	}
	return nil
}

// CreateAndAddDefaultSchema adds a schema of a namespace this module has no
// class for.
func (m *XMPMetadata) CreateAndAddDefaultSchema(nsPrefix, nsURI string) (*schema.XMPSchema, error) {
	s, err := schema.NewXMPSchemaOfNamespace(m, nsURI, nsPrefix)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// CreateAndAddPDFAExtensionSchemaWithDefaultNS adds the PDF/A extension schema
// with the namespaces PDFAExtensionSchema declares.
func (m *XMPMetadata) CreateAndAddPDFAExtensionSchemaWithDefaultNS() (
	*schema.PDFAExtensionSchema, error) {
	s, err := schema.NewPDFAExtensionSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// CreateAndAddPDFAExtensionSchemaWithNS adds the PDF/A extension schema with a
// given list of namespaces.
//
// Java takes the map and never reads it: the schema it builds is the default
// one, and the declared XmpSchemaException is never thrown. Kept, because the
// two ways to make it read the map -- deleting the method, or inventing which
// way the map runs -- are both decisions rather than fixes; nothing in either
// tree calls it or says. See migration/JAVA-BUGS.md 58.
func (m *XMPMetadata) CreateAndAddPDFAExtensionSchemaWithNS(
	namespaces map[string]string) (*schema.PDFAExtensionSchema, error) {
	s, err := schema.NewPDFAExtensionSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// PDFExtensionSchema returns the PDF/A extension schema, and nil where it is
// not declared.
func (m *XMPMetadata) PDFExtensionSchema() *schema.PDFAExtensionSchema {
	s, _ := m.Schema(schema.PDFAExtensionNamespace).(*schema.PDFAExtensionSchema)
	return s
}

// CreateAndAddPDFAIdentificationSchema adds the PDF/A identification schema.
func (m *XMPMetadata) CreateAndAddPDFAIdentificationSchema() (
	*schema.PDFAIdentificationSchema, error) {
	s, err := schema.NewPDFAIdentificationSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// PDFAIdentificationSchema returns the PDF/A identification schema, and nil
// where it is not declared.
func (m *XMPMetadata) PDFAIdentificationSchema() *schema.PDFAIdentificationSchema {
	s, _ := m.Schema(schema.PDFAIdentificationNamespace).(*schema.PDFAIdentificationSchema)
	return s
}

// CreateAndAddDublinCoreSchema adds the Dublin Core schema.
func (m *XMPMetadata) CreateAndAddDublinCoreSchema() (*schema.DublinCoreSchema, error) {
	s, err := schema.NewDublinCoreSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// DublinCoreSchema returns the Dublin Core schema, and nil where it is not
// declared.
func (m *XMPMetadata) DublinCoreSchema() *schema.DublinCoreSchema {
	s, _ := m.Schema(schema.DublinCoreNamespace).(*schema.DublinCoreSchema)
	return s
}

// CreateAndAddBasicJobTicketSchema adds the basic job ticket schema.
func (m *XMPMetadata) CreateAndAddBasicJobTicketSchema() (
	*schema.XMPBasicJobTicketSchema, error) {
	s, err := schema.NewXMPBasicJobTicketSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// BasicJobTicketSchema returns the basic job ticket schema, and nil where it is
// not declared.
func (m *XMPMetadata) BasicJobTicketSchema() *schema.XMPBasicJobTicketSchema {
	s, _ := m.Schema(schema.JobTicketNamespace).(*schema.XMPBasicJobTicketSchema)
	return s
}

// CreateAndAddXMPRightsManagementSchema adds the rights management schema.
func (m *XMPMetadata) CreateAndAddXMPRightsManagementSchema() (
	*schema.XMPRightsManagementSchema, error) {
	s, err := schema.NewXMPRightsManagementSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// XMPRightsManagementSchema returns the rights management schema, and nil where
// it is not declared.
func (m *XMPMetadata) XMPRightsManagementSchema() *schema.XMPRightsManagementSchema {
	s, _ := m.Schema(schema.RightsManagementNamespace).(*schema.XMPRightsManagementSchema)
	return s
}

// CreateAndAddXMPBasicSchema adds the XMP basic schema.
func (m *XMPMetadata) CreateAndAddXMPBasicSchema() (*schema.XMPBasicSchema, error) {
	s, err := schema.NewXMPBasicSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// XMPBasicSchema returns the XMP basic schema, and nil where it is not
// declared.
func (m *XMPMetadata) XMPBasicSchema() *schema.XMPBasicSchema {
	s, _ := m.Schema(schema.XMPBasicNamespace).(*schema.XMPBasicSchema)
	return s
}

// CreateAndAddXMPMediaManagementSchema adds the media management schema.
func (m *XMPMetadata) CreateAndAddXMPMediaManagementSchema() (
	*schema.XMPMediaManagementSchema, error) {
	s, err := schema.NewXMPMediaManagementSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// XMPMediaManagementSchema returns the media management schema, and nil where
// it is not declared.
func (m *XMPMetadata) XMPMediaManagementSchema() *schema.XMPMediaManagementSchema {
	s, _ := m.Schema(schema.MediaManagementNamespace).(*schema.XMPMediaManagementSchema)
	return s
}

// CreateAndAddPhotoshopSchema adds the photoshop schema.
func (m *XMPMetadata) CreateAndAddPhotoshopSchema() (*schema.PhotoshopSchema, error) {
	s, err := schema.NewPhotoshopSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// PhotoshopSchema returns the photoshop schema, and nil where it is not
// declared.
func (m *XMPMetadata) PhotoshopSchema() *schema.PhotoshopSchema {
	s, _ := m.Schema(schema.PhotoshopNamespace).(*schema.PhotoshopSchema)
	return s
}

// CreateAndAddAdobePDFSchema adds the Adobe PDF schema.
func (m *XMPMetadata) CreateAndAddAdobePDFSchema() (*schema.AdobePDFSchema, error) {
	s, err := schema.NewAdobePDFSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// AdobePDFSchema returns the Adobe PDF schema, and nil where it is not
// declared.
func (m *XMPMetadata) AdobePDFSchema() *schema.AdobePDFSchema {
	s, _ := m.Schema(schema.AdobePDFNamespace).(*schema.AdobePDFSchema)
	return s
}

// CreateAndAddPageTextSchema adds the page text schema.
func (m *XMPMetadata) CreateAndAddPageTextSchema() (*schema.XMPPageTextSchema, error) {
	s, err := schema.NewXMPPageTextSchema(m)
	if err != nil {
		return nil, err
	}
	s.SetAboutAsSimple("")
	m.AddSchema(s)
	return s, nil
}

// PageTextSchema returns the page text schema, and nil where it is not
// declared.
func (m *XMPMetadata) PageTextSchema() *schema.XMPPageTextSchema {
	s, _ := m.Schema(schema.PageTextNamespace).(*schema.XMPPageTextSchema)
	return s
}
