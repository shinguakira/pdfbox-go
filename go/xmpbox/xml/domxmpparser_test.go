package xml_test

// Port of org.apache.xmpbox.xml.DomXmpParserTest, first part.
//
// An external test: it builds metadata through the root package, which imports
// this one.

import (
	"bytes"
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// bom is the byte order mark an xpacket opens with, which the Java test writes
// into its string literals.
//
// Go forbids the character anywhere in a source file but its first byte, so it
// is spelled as its code point here.
var bom = string(rune(0xFEFF))

// xmlFixture is where the Java test resources are, which the port reads rather
// than copying.
const xmlFixture = "../../../xmpbox/src/test/resources/"

// parse reads a packet in strict mode and fails the test where it will not.
func parse(t *testing.T, s string) *xmpbox.XMPMetadata {
	t.Helper()
	xmp, err := xmpxml.NewDomXmpParser().ParseBytes([]byte(s))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return xmp
}

// parseLenient reads a packet in lenient mode and fails the test where it will
// not.
func parseLenient(t *testing.T, s string) *xmpbox.XMPMetadata {
	t.Helper()
	parser := xmpxml.NewDomXmpParser()
	parser.SetStrictParsing(false)
	xmp, err := parser.ParseBytes([]byte(s))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return xmp
}

// parseFails reads a packet in strict mode and checks the message it is
// refused with, which is Java's assertThrows plus assertEquals on getMessage.
func parseFails(t *testing.T, s, message string) {
	t.Helper()
	_, err := xmpxml.NewDomXmpParser().ParseBytes([]byte(s))
	if err == nil {
		t.Fatalf("parse reported nothing, want %q", message)
	}
	if err.Error() != message {
		t.Errorf("parse = %q, want %q", err.Error(), message)
	}
}

// parseFileFails is parseFails for a packet already in bytes.
func parseFileFails(t *testing.T, content []byte, message string) {
	t.Helper()
	_, err := xmpxml.NewDomXmpParser().ParseBytes(content)
	if err == nil {
		t.Fatalf("parse reported nothing, want %q", message)
	}
	if err.Error() != message {
		t.Errorf("parse = %q, want %q", err.Error(), message)
	}
}

// readFixture reads one of the Java test resources.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	content, err := os.ReadFile(xmlFixture + name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return content
}

// equal reports a value that is not what the Java test asserts.
func equal[T comparable](t *testing.T, what string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

// noError fails the test where a step reported a problem.
func noError(t *testing.T, what string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

func TestPDFBox5649(t *testing.T) {
	xmp, err := xmpxml.NewDomXmpParser().ParseBytes(
		readFixture(t, "org/apache/xmpbox/xml/PDFBOX-5649.xml"))
	noError(t, "parse", err)
	if xmp == nil {
		t.Fatal("parse answered nothing")
	}
}

func TestPDFBox5835(t *testing.T) {
	xmp, err := xmpxml.NewDomXmpParser().ParseBytes(
		readFixture(t, "org/apache/xmpbox/xml/PDFBOX-5835.xml"))
	noError(t, "parse", err)
	pdfaid := xmp.PDFAIdentificationSchema()
	if pdfaid == nil {
		t.Fatal("PDFAIdentificationSchema() = nil")
	}
	equal(t, "Conformance()", pdfaid.Conformance(), "A")
	part, held := pdfaid.Part()
	if !held || part != 3 {
		t.Errorf("Part() = %v, %v, want 3", part, held)
	}
}

func TestPDFBox5976(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="" id="W5M0MpCehiHzreSzNTczkc9d"?>
<rdf:RDF
	xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
	xmlns:pdf="http://ns.adobe.com/pdf/1.3/"
	xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/">
	    <rdf:Description pdfaid:conformance="B" pdfaid:part="3" rdf:about=""/>
	    <rdf:Description pdf:Producer="WeasyPrint 64.1" rdf:about=""/>
</rdf:RDF>
<?xpacket end="r"?>`
	xmp := parse(t, s)
	pdfaid := xmp.PDFAIdentificationSchema()
	if pdfaid == nil {
		t.Fatal("PDFAIdentificationSchema() = nil")
	}
	equal(t, "Conformance()", pdfaid.Conformance(), "B")
	part, held := pdfaid.Part()
	if !held || part != 3 {
		t.Errorf("Part() = %v, %v, want 3", part, held)
	}
}

// TestPDFBox6106 checks that "pdf:CreationDate='2004-01-30T17:21:50Z'" is
// detected as incorrect. Only Keywords, PDFVersion and Producer are allowed in
// strict mode.
func TestPDFBox6106(t *testing.T) {
	// from file 001358.pdf
	s := `<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d' bytes='647'?>
<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'
         xmlns:iX='http://ns.adobe.com/iX/1.0/'>
	<rdf:Description about=''
	                 xmlns='http://ns.adobe.com/pdf/1.3/'
	                 xmlns:pdf='http://ns.adobe.com/pdf/1.3/'
	                 pdf:CreationDate='2004-01-30T17:21:50Z'
	                 pdf:ModDate='2004-01-30T17:21:50Z'
	                 pdf:Producer='Acrobat Distiller 5.0.5 (Windows)'/>
	<rdf:Description about=''
	                 xmlns='http://ns.adobe.com/xap/1.0/'
	                 xmlns:xap='http://ns.adobe.com/xap/1.0/'
	                 xap:CreateDate='2004-01-30T17:21:50Z'
	                 xap:ModifyDate='2004-01-30T17:21:50Z'
	                 xap:MetadataDate='2004-01-30T17:21:50Z'/>
</rdf:RDF><?xpacket end='r'?>`
	parseFails(t, s, "No type defined for {http://ns.adobe.com/pdf/1.3/}CreationDate")
}

// TestPDFBox5288 checks that a namespace declaration within an rdf:li element
// is found.
func TestPDFBox5288(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?><x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Public XMP Toolkit Core 4.0  ">

 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">

  <rdf:Description xmlns:xmpMM="http://ns.adobe.com/xap/1.0/mm/" rdf:about="">
   <xmpMM:DocumentID>uidd:1f0e03977b90b6365a376454ffdf34a7</xmpMM:DocumentID>
   <xmpMM:History>
    <rdf:Seq>
     <rdf:li xmlns:stEvt="http://ns.adobe.com/xap/1.0/sType/ResourceEvent#">
      <rdf:Description>
       <stEvt:action>created</stEvt:action>
       <stEvt:parameters>iDRS PDF output engine 7</stEvt:parameters>
       <stEvt:when>2022-09-12T12:00:07+02:00</stEvt:when>
      </rdf:Description>
     </rdf:li>
    </rdf:Seq>
   </xmpMM:History>
  </rdf:Description>
 </rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`
	xmp := parse(t, s)
	mm := xmp.XMPMediaManagementSchema()
	if mm == nil {
		t.Fatal("XMPMediaManagementSchema() = nil")
	}
	equal(t, "DocumentID()", mm.DocumentID(), "uidd:1f0e03977b90b6365a376454ffdf34a7")
	history := mm.HistoryProperty()
	if history == nil {
		t.Fatal("HistoryProperty() = nil")
	}
	entries := history.AllProperties()
	if len(entries) == 0 {
		t.Fatal("HistoryProperty() held nothing")
	}
	firstHistoryEntry, isEvent := entries[0].(*xmptype.ResourceEventType)
	if !isEvent {
		t.Fatalf("history[0] = %T, want a resource event", entries[0])
	}
	equal(t, "Action()", firstHistoryEntry.Action(), "created")
	equal(t, "Parameters()", firstHistoryEntry.Parameters(), "iDRS PDF output engine 7")
}

// TestPageTextSchema tests XMPPageTextSchema and XMPMediaManagementSchema.
func TestPageTextSchema(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
           <rdf:Description xmlns:stRef="http://ns.adobe.com/xap/1.0/sType/ResourceRef#"
		                 xmlns:xmpMM="http://ns.adobe.com/xap/1.0/mm/"
		                 rdf:about="">
			<xmpMM:InstanceID>uuid:b429d411-e628-45ca-b932-d2c77fbe6cd3</xmpMM:InstanceID>
			<xmpMM:DocumentID>adobe:docid:indd:db084a4d-dbb2-11dc-ac34-beb3cc4028ec</xmpMM:DocumentID>
			<xmpMM:RenditionClass>proof:pdf</xmpMM:RenditionClass>
			<xmpMM:DerivedFrom rdf:parseType="Resource">
				<stRef:documentID>adobe:docid:indd:fa7c6589-9f4a-11dc-9641-af983df728d7</stRef:documentID>
			</xmpMM:DerivedFrom>
		</rdf:Description>		<rdf:Description xmlns:xmpTPg="http://ns.adobe.com/xap/1.0/t/pg/"
		                 rdf:about="">
			<xmpTPg:MaxPageSize>
				<rdf:Description xmlns:stDim="http://ns.adobe.com/xap/1.0/sType/Dimensions#">
					<stDim:w>4</stDim:w>
					<stDim:h>3</stDim:h>
					<stDim:unit>inch</stDim:unit>
				</rdf:Description>
			</xmpTPg:MaxPageSize>
			<xmpTPg:NPages>7</xmpTPg:NPages>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end="r"?>`
	xmp := parse(t, s)
	assertMaxPageSize(t, xmp)

	mm := xmp.XMPMediaManagementSchema()
	if mm == nil {
		t.Fatal("XMPMediaManagementSchema() = nil")
	}
	derivedFrom := mm.DerivedFromProperty()
	equal(t, "InstanceID()", mm.InstanceID(), "uuid:b429d411-e628-45ca-b932-d2c77fbe6cd3")
	equal(t, "RenditionClass()", mm.RenditionClass(), "proof:pdf")
	equal(t, "DocumentID()", mm.DocumentID(),
		"adobe:docid:indd:db084a4d-dbb2-11dc-ac34-beb3cc4028ec")
	if derivedFrom == nil {
		t.Fatal("DerivedFromProperty() = nil")
	}
	equal(t, "DerivedFrom.DocumentID()", derivedFrom.DocumentID(),
		"adobe:docid:indd:fa7c6589-9f4a-11dc-9641-af983df728d7")
}

// assertMaxPageSize is the pair of assertions the three page text cases share.
func assertMaxPageSize(t *testing.T, xmp *xmpbox.XMPMetadata) {
	t.Helper()
	pageText := xmp.PageTextSchema()
	if pageText == nil {
		t.Fatal("PageTextSchema() = nil")
	}
	dim, isDimensions := pageText.Property(schema.PageTextMaxPageSize).(*xmptype.DimensionsType)
	if !isDimensions {
		t.Fatalf("MaxPageSize = %T, want dimensions",
			pageText.Property(schema.PageTextMaxPageSize))
	}
	equal(t, "MaxPageSize", dim.String(), "DimensionsType{4.0 x 3.0 inch}")
	nPages, isSimple := pageText.Property(schema.PageTextNPages).(xmptype.AbstractSimpleProperty)
	if !isSimple {
		t.Fatalf("NPages = %T, want a simple property",
			pageText.Property(schema.PageTextNPages))
	}
	equal(t, "NPages", stringOfProperty(nPages), "[NPages=IntegerType:7]")
}

// stringOfProperty is Java's AbstractSimpleProperty.toString.
func stringOfProperty(property xmptype.AbstractSimpleProperty) string {
	return "[" + property.PropertyName() + "=" + property.TypeName() + ":" +
		property.StringValue() + "]"
}

// TestPageTextSchema2 tests the page text schema with dimensions mixed as
// children and attributes.
func TestPageTextSchema2(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
           <rdf:Description xmlns:xmpTPg="http://ns.adobe.com/xap/1.0/t/pg/"                            xmlns:stDim="http://ns.adobe.com/xap/1.0/sType/Dimensions#"		                 rdf:about="">
			<xmpTPg:MaxPageSize>
				<rdf:Description stDim:w="4" stDim:h="3">
					<stDim:unit>inch</stDim:unit>
				</rdf:Description>
			</xmpTPg:MaxPageSize>
			<xmpTPg:NPages>7</xmpTPg:NPages>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end="r"?>`
	assertMaxPageSize(t, parse(t, s))
}

// TestPageTextSchema3 tests the page text schema with dimensions as attributes
// only.
func TestPageTextSchema3(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
           <rdf:Description xmlns:xmpTPg="http://ns.adobe.com/xap/1.0/t/pg/"                            xmlns:stDim="http://ns.adobe.com/xap/1.0/sType/Dimensions#"		                 rdf:about="">
			<xmpTPg:MaxPageSize>
				<rdf:Description stDim:w="4" stDim:h="3" stDim:unit="inch"/>
			</xmpTPg:MaxPageSize>
			<xmpTPg:NPages>7</xmpTPg:NPages>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end="r"?>`
	assertMaxPageSize(t, parse(t, s))
}

// TestPDFBox3882 tests attributes being used as properties to define an
// extension schema, and checks the content of the extension schema itself.
func TestPDFBox3882(t *testing.T) {
	xmp, err := xmpxml.NewDomXmpParser().ParseBytes(
		readFixture(t, "org/apache/xmpbox/xml/PDFBOX-3882-dematbox.xml"))
	noError(t, "parse", err)

	extension := xmp.PDFExtensionSchema()
	if extension == nil {
		t.Fatal("PDFExtensionSchema() = nil")
	}
	allProperties := extension.SchemasProperty().AllProperties()
	if len(allProperties) != 1 {
		t.Fatalf("SchemasProperty() held %d schemas, want 1", len(allProperties))
	}
	pdfExtensionSchema, isSchemaType := allProperties[0].(*xmptype.PDFASchemaType)
	if !isSchemaType {
		t.Fatalf("schemas[0] = %T, want a PDF/A schema type", allProperties[0])
	}
	equal(t, "NamespaceURI()", pdfExtensionSchema.NamespaceURI(),
		"http://www.sagemcom.com/documents/xmlns/dematbox")
	equal(t, "PrefixValue()", pdfExtensionSchema.PrefixValue(), "dematbox")

	extensionSchema := xmp.Schema(pdfExtensionSchema.NamespaceURI())
	if extensionSchema == nil {
		t.Fatal("Schema(dematbox) = nil")
	}
	equal(t, "Namespace()", extensionSchema.Namespace(), pdfExtensionSchema.NamespaceURI())
	equal(t, "Prefix()", extensionSchema.Prefix(), pdfExtensionSchema.PrefixValue())

	pageInfoProp, isArray := extensionSchema.Base().Property("PageInfo").(*xmptype.ArrayProperty)
	if !isArray {
		t.Fatalf("PageInfo = %T, want an array", extensionSchema.Base().Property("PageInfo"))
	}
	dst, isDefined := pageInfoProp.AllProperties()[0].(*xmptype.DefinedStructuredType)
	if !isDefined {
		t.Fatalf("PageInfo[0] = %T, want a defined structured type",
			pageInfoProp.AllProperties()[0])
	}
	number, isSimple := dst.Property("number").(xmptype.AbstractSimpleProperty)
	if !isSimple {
		t.Fatalf("number = %T, want a simple property", dst.Property("number"))
	}
	equal(t, "number", stringOfProperty(number), "[number=IntegerType:1]")
	origNumber, isSimple := dst.Property("origNumber").(xmptype.AbstractSimpleProperty)
	if !isSimple {
		t.Fatalf("origNumber = %T, want a simple property", dst.Property("origNumber"))
	}
	equal(t, "origNumber", stringOfProperty(origNumber), "[origNumber=IntegerType:1]")
}

// TestPDFBox38822 tests ResourceEventType properties written as attributes
// rather than properties, which is the call of tryParseAttributesAsProperties
// at the end of parseLiElement.
func TestPDFBox38822(t *testing.T) {
	// data modified from XMP data in the JPEG file in Apache Tika
	// JpegParserTest.testJPEGXMPMM()
	s := `<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/"
           x:xmptk="Adobe XMP Core 5.0-c060 61.134777, 2010/02/12-17:32:00        ">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description rdf:about=""
		                 xmlns:xmp="http://ns.adobe.com/xap/1.0/"
		                 xmlns:dc="http://purl.org/dc/elements/1.1/"
		                 xmlns:xmpMM="http://ns.adobe.com/xap/1.0/mm/"
		                 xmlns:stEvt="http://ns.adobe.com/xap/1.0/sType/ResourceEvent#"
		                 xmlns:stRef="http://ns.adobe.com/xap/1.0/sType/ResourceRef#"
		                 xmlns:photoshop="http://ns.adobe.com/photoshop/1.0/"
		                 xmp:CreatorTool="Adobe Photoshop CS5 Macintosh"
		                 xmp:CreateDate="2012-04-30T12:52:07-04:00"
		                 xmp:MetadataDate="2012-05-03T13:36:11-04:00"
		                 xmp:ModifyDate="2012-05-03T13:36:11-04:00"
		                 dc:format="image/jpeg"
		                 xmpMM:InstanceID="xmp.iid:49E997338D4911E1AB62EBF9B374B234"
		                 xmpMM:DocumentID="xmp.did:49E997348D4911E1AB62EBF9B374B234"
		                 xmpMM:OriginalDocumentID="xmp.did:01801174072068118A6D9A879C818256"
		                 photoshop:History="2012-05-03T09:34:50-04:00&#x9;File i1222b.jpg opened&#xA;">
			<xmpMM:History>
				<rdf:Seq>
					<rdf:li stEvt:action="created"
					        stEvt:instanceID="xmp.iid:01801174072068118A6D9A879C818256"
					        stEvt:when="2012-04-30T12:52:07-04:00"
					        stEvt:softwareAgent="Adobe Photoshop CS5 Macintosh"/>
					<rdf:li stEvt:action="saved"
					        stEvt:instanceID="xmp.iid:02801174072068118A6D9A879C818256"
					        stEvt:when="2012-04-30T12:54:04-04:00"
					        stEvt:softwareAgent="Adobe Photoshop CS5 Macintosh"
					        stEvt:changed="/"/>
					<rdf:li stEvt:action="saved"
					        stEvt:instanceID="xmp.iid:03801174072068118A6D9A879C818256"
					        stEvt:when="2012-04-30T12:54:48-04:00"
					        stEvt:softwareAgent="Adobe Photoshop CS5 Macintosh"
					        stEvt:changed="/"/>
				</rdf:Seq>
			</xmpMM:History>
			<xmpMM:DerivedFrom stRef:instanceID="xmp.iid:21F0677BA22168118A6D9A879C818256"
			                   stRef:documentID="xmp.did:01801174072068118A6D9A879C818256"
			                   stRef:originalDocumentID="xmp.did:01801174072068118A6D9A879C818256"/>
			<photoshop:DocumentAncestors>
				<rdf:Bag>
					<rdf:li>adobe:docid:photoshop:11d3ec5a-c131-11d8-9274-ec65c7d7e0c6</rdf:li>
					<rdf:li>adobe:docid:photoshop:aadc7027-309c-11d8-9596-9cf45d2f630b</rdf:li>
					<rdf:li>adobe:docid:photoshop:c7961c59-6e0f-11d8-87b7-d67539df12d8</rdf:li>
				</rdf:Bag>
			</photoshop:DocumentAncestors>
			<photoshop:DateCreated>2012-04-30T12:54:48Z</photoshop:DateCreated>
			<photoshop:TextLayers>
				<rdf:Seq>
                               <rdf:li photoshop:LayerName="Name1" photoshop:LayerText="Text1"/>
                               <rdf:li photoshop:LayerName="Name2" photoshop:LayerText="Text2"/>
				</rdf:Seq>
			</photoshop:TextLayers>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`
	xmp := parse(t, s)
	mm := xmp.XMPMediaManagementSchema()
	if mm == nil {
		t.Fatal("XMPMediaManagementSchema() = nil")
	}
	historyProperties := mm.HistoryProperty().AllProperties()
	if len(historyProperties) != 3 {
		t.Fatalf("HistoryProperty() held %d entries, want 3", len(historyProperties))
	}
	events := make([]*xmptype.ResourceEventType, 3)
	for i, held := range historyProperties {
		event, isEvent := held.(*xmptype.ResourceEventType)
		if !isEvent {
			t.Fatalf("history[%d] = %T, want a resource event", i, held)
		}
		events[i] = event
	}
	equal(t, "events[0].Action()", events[0].Action(), "created")
	equal(t, "events[0].InstanceID()", events[0].InstanceID(),
		"xmp.iid:01801174072068118A6D9A879C818256")
	when, held := events[0].When()
	if !held {
		t.Fatal("events[0].When() held nothing")
	}
	equal(t, "events[0].When().Year()", when.Year(), 2012)
	equal(t, "events[0].When().Minute()", when.Minute(), 52)
	equal(t, "events[0].SoftwareAgent()", events[0].SoftwareAgent(),
		"Adobe Photoshop CS5 Macintosh")
	equal(t, "events[1].InstanceID()", events[1].InstanceID(),
		"xmp.iid:02801174072068118A6D9A879C818256")
	equal(t, "events[2].InstanceID()", events[2].InstanceID(),
		"xmp.iid:03801174072068118A6D9A879C818256")
	when, _ = events[1].When()
	equal(t, "events[1].When().Year()", when.Year(), 2012)
	equal(t, "events[1].When().Minute()", when.Minute(), 54)
	equal(t, "events[1].When().Second()", when.Second(), 4)
	when, _ = events[2].When()
	equal(t, "events[2].When().Year()", when.Year(), 2012)
	equal(t, "events[2].When().Minute()", when.Minute(), 54)
	equal(t, "events[2].When().Second()", when.Second(), 48)

	equal(t, "InstanceID()", mm.InstanceID(), "xmp.iid:49E997338D4911E1AB62EBF9B374B234")
	equal(t, "DocumentID()", mm.DocumentID(), "xmp.did:49E997348D4911E1AB62EBF9B374B234")
	equal(t, "OriginalDocumentID()", mm.OriginalDocumentID(),
		"xmp.did:01801174072068118A6D9A879C818256")

	photoshop := xmp.PhotoshopSchema()
	if photoshop == nil {
		t.Fatal("PhotoshopSchema() = nil")
	}
	textLayers, err := photoshop.TextLayers()
	noError(t, "TextLayers", err)
	if len(textLayers) != 2 {
		t.Fatalf("TextLayers() held %d layers, want 2", len(textLayers))
	}
	equal(t, "textLayers[0].LayerName()", textLayers[0].LayerName(), "Name1")
	equal(t, "textLayers[0].LayerText()", textLayers[0].LayerText(), "Text1")
	equal(t, "textLayers[1].LayerName()", textLayers[1].LayerName(), "Name2")
	equal(t, "textLayers[1].LayerText()", textLayers[1].LayerText(), "Text2")
	equal(t, "DateCreated()", photoshop.DateCreated(), "2012-04-30T12:54:48+00:00")
	equal(t, "History()", photoshop.History(),
		"2012-05-03T09:34:50-04:00\tFile i1222b.jpg opened\n")

	ancestors := photoshop.DocumentAncestorsProperty().AllProperties()
	if len(ancestors) != 3 {
		t.Fatalf("DocumentAncestors held %d values, want 3", len(ancestors))
	}
	for i, want := range []string{
		"adobe:docid:photoshop:11d3ec5a-c131-11d8-9274-ec65c7d7e0c6",
		"adobe:docid:photoshop:aadc7027-309c-11d8-9596-9cf45d2f630b",
		"adobe:docid:photoshop:c7961c59-6e0f-11d8-87b7-d67539df12d8",
	} {
		text, isText := ancestors[i].(*xmptype.TextType)
		if !isText {
			t.Fatalf("ancestors[%d] = %T, want text", i, ancestors[i])
		}
		equal(t, "ancestors", text.StringValue(), want)
	}
	// mm.DerivedFromProperty() doesn't work. However the PDFLib XMP validator
	// considers this file to be invalid, so lets not bother more.
}

// TestPDFBox5292 tests whether an inline extension schema is detected.
func TestPDFBox5292(t *testing.T) {
	s := `<?xpacket begin="ï»¿" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Adobe XMP Core 5.6-c015 84.159810, 2016/09/10-02:41:30        ">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description rdf:about=""
                         xmlns:xmp="http://ns.adobe.com/xap/1.0/"
                         xmlns:dc="http://purl.org/dc/elements/1.1/"
                         xmlns:pdf="http://ns.adobe.com/pdf/1.3/"
                         xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/"
                         xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/"
                         xmlns:pdfaSchema="http://www.aiim.org/pdfa/ns/schema#"
                         xmlns:pdfaProperty="http://www.aiim.org/pdfa/ns/property#"
                         xmlns:example="http://ns.example.org/default/1.0/">
            <xmp:CreateDate>2021-05-21T11:42:49+01:00</xmp:CreateDate>
            <xmp:ModifyDate>2021-05-21T11:47:16+02:00</xmp:ModifyDate>
            <xmp:MetadataDate>2021-05-21T11:47:16+02:00</xmp:MetadataDate>
            <dc:format>application/pdf</dc:format>
            <dc:title>
                <rdf:Alt>
                    <rdf:li xml:lang="x-default">Inline XMP Extension PoC</rdf:li>
                </rdf:Alt>
            </dc:title>
            <dc:creator>
                <rdf:Seq>
                    <rdf:li>DSO</rdf:li>
                </rdf:Seq>
            </dc:creator>
            <dc:description>
                <rdf:Alt>
                    <rdf:li xml:lang="x-default">Inline XMP Extension PoC</rdf:li>
                </rdf:Alt>
            </dc:description>
            <pdf:Keywords/>
            <pdfaid:part>2</pdfaid:part>
            <pdfaid:conformance>A</pdfaid:conformance>
            <example:Data>Example</example:Data>
            <pdfaExtension:schemas>
                <rdf:Bag>
                    <rdf:li rdf:parseType="Resource">
                        <pdfaSchema:schema>Simple Schema</pdfaSchema:schema>
                        <pdfaSchema:namespaceURI>http://ns.example.org/default/1.0/</pdfaSchema:namespaceURI>
                        <pdfaSchema:prefix>example</pdfaSchema:prefix>
                        <pdfaSchema:property>
                            <rdf:Seq>
                                <rdf:li rdf:parseType="Resource">
                                    <pdfaProperty:name>Data</pdfaProperty:name>
                                    <pdfaProperty:valueType>Text</pdfaProperty:valueType>
                                    <pdfaProperty:category>internal</pdfaProperty:category>
                                    <pdfaProperty:description>Example Data</pdfaProperty:description>
                                </rdf:li>
                            </rdf:Seq>
                        </pdfaSchema:property>
                    </rdf:li>
                    <rdf:li rdf:parseType="Resource">
                        <pdfaSchema:namespaceURI>http://www.aiim.org/pdfa/ns/id/</pdfaSchema:namespaceURI>
                        <pdfaSchema:prefix>pdfaid</pdfaSchema:prefix>
                        <pdfaSchema:schema>PDF/A ID Schema</pdfaSchema:schema>
                        <pdfaSchema:property>
                            <rdf:Seq>
                                <rdf:li rdf:parseType="Resource">
                                    <pdfaProperty:category>internal</pdfaProperty:category>
                                    <pdfaProperty:description>Part of PDF/A standard</pdfaProperty:description>
                                    <pdfaProperty:name>part</pdfaProperty:name>
                                    <pdfaProperty:valueType>Integer</pdfaProperty:valueType>
                                </rdf:li>
                                <rdf:li rdf:parseType="Resource">
                                    <pdfaProperty:category>internal</pdfaProperty:category>
                                    <pdfaProperty:description>Conformance level of PDF/A standard</pdfaProperty:description>
                                    <pdfaProperty:name>conformance</pdfaProperty:name>
                                    <pdfaProperty:valueType>Text</pdfaProperty:valueType>
                                </rdf:li>
                            </rdf:Seq>
                        </pdfaSchema:property>
                    </rdf:li>
                </rdf:Bag>
            </pdfaExtension:schemas>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta>

<?xpacket end="w"?>`
	xmp := parse(t, s)
	pdfaid := xmp.PDFAIdentificationSchema()
	if pdfaid == nil {
		t.Fatal("PDFAIdentificationSchema() = nil")
	}
	part, held := pdfaid.Part()
	if !held || part != 2 {
		t.Errorf("Part() = %v, %v, want 2", part, held)
	}
	example := xmp.Schema("http://ns.example.org/default/1.0/")
	if example == nil {
		t.Fatal("Schema(example) = nil")
	}
	dataValue, err := example.Base().UnqualifiedTextPropertyValue("Data")
	noError(t, "UnqualifiedTextPropertyValue", err)
	equal(t, "Data", dataValue, "Example")
}

// TestLenientBagSeqMixup checks that a Seq/Bag mixup is refused in strict mode
// and read in lenient mode.
func TestLenientBagSeqMixup(t *testing.T) {
	s := `<?xpacket begin='` + bom + `' id='W5M0MpCehiHzreSzNTczkc9d'?>
<?adobe-xap-filters esc="CRLF"?>
<x:xmpmeta xmlns:x='adobe:ns:meta/'>
	<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
		<rdf:Description xmlns:dc='http://purl.org/dc/elements/1.1/'
		                 dc:format='application/pdf'>
			<dc:subject>
				<rdf:Seq>
					<rdf:li>Important subject</rdf:li>
					<rdf:li>Unimportant subject</rdf:li>
				</rdf:Seq>
			</dc:subject>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta>
<?xpacket end='w'?>`
	parseFails(t, s,
		"Invalid array type, expecting Bag and found Seq [prefix=dc; name=subject]")

	xmp := parseLenient(t, s)
	dc := xmp.DublinCoreSchema()
	if dc == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	subjects := dc.Subjects()
	if len(subjects) != 2 {
		t.Fatalf("Subjects() = %v, want two", subjects)
	}
	equal(t, "subjects[0]", subjects[0], "Important subject")
	equal(t, "subjects[1]", subjects[1], "Unimportant subject")
}

// serialized writes the metadata out, which several cases do to check that
// nothing is lost.
func serialized(t *testing.T, xmp *xmpbox.XMPMetadata) []byte {
	t.Helper()
	var baos bytes.Buffer
	noError(t, "Serialize", xmpxml.NewXmpSerializer().Serialize(xmp, &baos, true))
	return baos.Bytes()
}
