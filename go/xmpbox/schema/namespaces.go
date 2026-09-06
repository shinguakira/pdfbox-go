package schema

// The namespace of each schema this package declares.
//
// Java reads a schema class's namespace off its @StructuredType annotation, and
// XMPMetadata.getSchema(Class) goes through the annotation to find the schema of
// a class. The port has no reflection, so each namespace is a constant and a
// caller that wants a schema of a known kind asks for its namespace.
const (
	AdobePDFNamespace           = "http://ns.adobe.com/pdf/1.3/"
	DublinCoreNamespace         = "http://purl.org/dc/elements/1.1/"
	PDFAExtensionNamespace      = "http://www.aiim.org/pdfa/ns/extension/"
	PDFAIdentificationNamespace = "http://www.aiim.org/pdfa/ns/id/"
	RightsManagementNamespace   = "http://ns.adobe.com/xap/1.0/rights/"
	PageTextNamespace           = "http://ns.adobe.com/xap/1.0/t/pg/"
	JobTicketNamespace          = "http://ns.adobe.com/xap/1.0/bj/"
	XMPBasicNamespace           = "http://ns.adobe.com/xap/1.0/"
	PhotoshopNamespace          = "http://ns.adobe.com/photoshop/1.0/"
	MediaManagementNamespace    = "http://ns.adobe.com/xap/1.0/mm/"
	TiffNamespace               = "http://ns.adobe.com/tiff/1.0/"
	ExifNamespace               = "http://ns.adobe.com/exif/1.0/"
)

// PDFAExtensionPreferedPrefix is the prefix PDFAExtensionSchema declares, which
// PdfaExtensionHelper insists on.
const PDFAExtensionPreferedPrefix = "pdfaExtension"
