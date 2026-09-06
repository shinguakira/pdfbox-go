package xml_test

// Port of org.apache.xmpbox.xml.DomXmpParserTest, third part.

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

func TestTypeInLiResourceElement(t *testing.T) {
	// <rdf:li xmlns:stEvt="..." rdf:parseType="Resource"
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?><x:xmpmeta xmlns:x="adobe:ns:meta/">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description xmlns:xmpMM="http://ns.adobe.com/xap/1.0/mm/" rdf:about="">
            <xmpMM:History>
                <rdf:Seq>
                    <rdf:li xmlns:stEvt="http://ns.adobe.com/xap/1.0/sType/ResourceEvent#" rdf:parseType="Resource">
                        <stEvt:action>created</stEvt:action>
                        <stEvt:parameters>original PDF file</stEvt:parameters>
                    </rdf:li>
                </rdf:Seq>
            </xmpMM:History>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`
	xmp2 := parse(t, s)
	mm := xmp2.XMPMediaManagementSchema()
	if mm == nil {
		t.Fatal("XMPMediaManagementSchema() = nil")
	}
	entries := mm.HistoryProperty().AllProperties()
	if len(entries) == 0 {
		t.Fatal("HistoryProperty() held nothing")
	}
	firstHistoryEntry, isEvent := entries[0].(*xmptype.ResourceEventType)
	if !isEvent {
		t.Fatalf("history[0] = %T, want a resource event", entries[0])
	}
	equal(t, "Action()", firstHistoryEntry.Action(), "created")
	equal(t, "Parameters()", firstHistoryEntry.Parameters(), "original PDF file")
}

func TestLenientPdfaExtension(t *testing.T) {
	// First bag in pdfaExtension is incomplete.
	s := `<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/"
           x:xmptk="Adobe XMP Core 4.2.1-c043 52.372728, 2009/01/18-15:08:04">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description rdf:about=""
		                 xmlns:xmpMM="http://ns.adobe.com/xap/1.0/mm/">
			<xmpMM:DocumentID>uuid:0b306144-6a43-dcbd-6b3e-c6b6b1df873d</xmpMM:DocumentID>
			<xmpMM:InstanceID>uuid:0b306144-6a43-dcbd-6b3e-c6b6b1df873d</xmpMM:InstanceID>
		</rdf:Description>
		<rdf:Description rdf:about=""
		                 xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/"
		                 xmlns:pdfaSchema="http://www.aiim.org/pdfa/ns/schema#"
		                 xmlns:pdfaProperty="http://www.aiim.org/pdfa/ns/property#">
			<pdfaExtension:schemas>
				<rdf:Bag>
					<rdf:li rdf:parseType="Resource">
						<pdfaSchema:namespaceURI>http://ns.adobe.com/pdf/1.3/</pdfaSchema:namespaceURI>
						<pdfaSchema:prefix>pdf</pdfaSchema:prefix>
						<pdfaSchema:schema>Adobe PDF Schema</pdfaSchema:schema>
					</rdf:li>
					<rdf:li rdf:parseType="Resource">
						<pdfaSchema:namespaceURI>http://ns.adobe.com/xap/1.0/mm/</pdfaSchema:namespaceURI>
						<pdfaSchema:prefix>xmpMM</pdfaSchema:prefix>
						<pdfaSchema:schema>XMP Media Management Schema</pdfaSchema:schema>
						<pdfaSchema:property>
							<rdf:Seq>
								<rdf:li rdf:parseType="Resource">
									<pdfaProperty:category>internal</pdfaProperty:category>
									<pdfaProperty:description>UUID based identifier for specific incarnation of a document</pdfaProperty:description>
									<pdfaProperty:name>InstanceID</pdfaProperty:name>
									<pdfaProperty:valueType>URI</pdfaProperty:valueType>
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
									<pdfaProperty:description>Amendment of PDF/A standard</pdfaProperty:description>
									<pdfaProperty:name>amd</pdfaProperty:name>
									<pdfaProperty:valueType>Text</pdfaProperty:valueType>
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
	parseFails(t, s, "Missing pdfaSchema:property in type definition")

	parser2 := xmpxml.NewDomXmpParser()
	if !parser2.IsStrictParsing() {
		t.Error("IsStrictParsing() = false on a new parser, want true")
	}
	parser2.SetStrictParsing(false)
	if parser2.IsStrictParsing() {
		t.Error("IsStrictParsing() = true after SetStrictParsing(false)")
	}
	xmp2, err := parser2.ParseBytes([]byte(s))
	noError(t, "parse", err)
	mm := xmp2.XMPMediaManagementSchema()
	if mm == nil {
		t.Fatal("XMPMediaManagementSchema() = nil")
	}
	equal(t, "InstanceID()", mm.InstanceID(), "uuid:0b306144-6a43-dcbd-6b3e-c6b6b1df873d")
	equal(t, "DocumentID()", mm.DocumentID(), "uuid:0b306144-6a43-dcbd-6b3e-c6b6b1df873d")
}

func TestNoProcessingInstruction(t *testing.T) {
	// From file 000163.pdf, Coastal Services Magazine Volume 11_6
	// November/December
	s := `<x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Adobe XMP Core 4.1-c037 46.282696, Mon Apr 02 2007 18:36:42        ">
 <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <rdf:Description rdf:about=""
    xmlns:xapMM="http://ns.adobe.com/xap/1.0/mm/"
    xmlns:stRef="http://ns.adobe.com/xap/1.0/sType/ResourceRef#"
    xmlns:tiff="http://ns.adobe.com/tiff/1.0/"
    xmlns:xap="http://ns.adobe.com/xap/1.0/"
    xmlns:exif="http://ns.adobe.com/exif/1.0/"
    xmlns:dc="http://purl.org/dc/elements/1.1/"
    xmlns:photoshop="http://ns.adobe.com/photoshop/1.0/"
   xapMM:DocumentID="uuid:F1FEDA1D7D03DA11B0F6E4B4E63B0143"
   xapMM:InstanceID="uuid:7A28FBF56920DA11B4BBB356C0A5C72B"
   tiff:Orientation="1"
   tiff:XResolution="3050000/10000"
   tiff:YResolution="3050000/10000"
   tiff:ResolutionUnit="2"
   tiff:NativeDigest="123456"
   xap:ModifyDate="2005-09-08T09:13:10-04:00"
   xap:CreatorTool="Adobe Photoshop CS2 Windows"
   xap:CreateDate="2005-08-02T13:47:24-04:00"
   xap:MetadataDate="2005-09-08T09:13:10-04:00"
   exif:ColorSpace="-1"
   exif:PixelXDimension="1525"
   exif:PixelYDimension="387"
   exif:NativeDigest="12345678"
   dc:format="image/tiff"
   photoshop:ColorMode="4"
   photoshop:ICCProfile="U.S. Web Coated (SWOP) v2"
   photoshop:History="">
   <xapMM:DerivedFrom
    stRef:instanceID="adobe:docid:photoshop:28ff3dc5-4801-11d8-85d1-bb49d244e2ef"
    stRef:documentID="adobe:docid:photoshop:28ff3dc5-4801-11d8-85d1-bb49d244e2ef"/>
  </rdf:Description>
 </rdf:RDF>
</x:xmpmeta>`
	parseFails(t, s, "xmp should start with a processing instruction")

	xmp2 := parseLenient(t, s)
	assertNoProcessingInstruction(t, xmp2)

	written := serialized(t, xmp2)
	// check that there are no isolated properties
	// (Happened before the change at the bottom of loadAttributes())
	s2 := string(written)
	for _, isolated := range []string{
		" ColorMode=", " CreateDate=", " CreatorTool=", " DocumentID=",
	} {
		if strings.Contains(s2, isolated) {
			t.Errorf("serialized packet holds an isolated %q:\n%s", isolated, s2)
		}
	}

	// now make sure that parsing again still brings the same data
	parser3 := xmpxml.NewDomXmpParser()
	parser3.SetStrictParsing(false)
	xmp3, err := parser3.ParseBytes(written)
	noError(t, "parse", err)
	assertNoProcessingInstruction(t, xmp3)
}

// assertNoProcessingInstruction is the block of assertions the case makes
// before and after the round trip.
func assertNoProcessingInstruction(t *testing.T, xmp *xmpbox.XMPMetadata) {
	t.Helper()
	dc := xmp.DublinCoreSchema()
	if dc == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	equal(t, "Format()", dc.Format(), "image/tiff")

	mm := xmp.XMPMediaManagementSchema()
	if mm == nil {
		t.Fatal("XMPMediaManagementSchema() = nil")
	}
	equal(t, "DocumentID()", mm.DocumentID(), "uuid:F1FEDA1D7D03DA11B0F6E4B4E63B0143")

	tiff := xmp.Schema(schema.TiffNamespace)
	if tiff == nil {
		t.Fatal("Schema(tiff) = nil")
	}
	equal(t, "Orientation",
		propertyString(t, "Orientation", tiff.Base().Property(schema.TiffOrientation)),
		"[Orientation=IntegerType:1]")

	photoshop := xmp.PhotoshopSchema()
	if photoshop == nil {
		t.Fatal("PhotoshopSchema() = nil")
	}
	colorMode, held := photoshop.ColorMode()
	if !held || colorMode != 4 {
		t.Errorf("ColorMode() = %v, %v, want 4", colorMode, held)
	}

	exif := xmp.Schema(schema.ExifNamespace)
	if exif == nil {
		t.Fatal("Schema(exif) = nil")
	}
	equal(t, "PixelXDimension",
		propertyString(t, "PixelXDimension",
			exif.Base().Property(schema.ExifPixelXDimension)),
		"[PixelXDimension=IntegerType:1525]")

	basic := xmp.XMPBasicSchema()
	if basic == nil {
		t.Fatal("XMPBasicSchema() = nil")
	}
	equal(t, "CreatorTool()", basic.CreatorTool(), "Adobe Photoshop CS2 Windows")
}

func TestNoSchema(t *testing.T) {
	// From file 0075304.pdf, Centers for Medicare Medicaid Services: the file
	// uses "xml:ModifyDate" instead of "xmp:ModifyDate".
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?><x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Adobe XMP Core 5.6-c016 91.163616, 2018/10/29-16:58:49        ">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description xmlns:xmp="http://ns.adobe.com/xap/1.0/" xmlns:pdf="http://ns.adobe.com/pdf/1.3/" rdf:about="">
            <xml:ModifyDate>2019-07-26'T'19:28:53.000'-04:00'</xml:ModifyDate>
            <xmp:ModifyDate>2019-07-29T15:12:07-04:00</xmp:ModifyDate>
            <pdf:Producer>iTextSharp 4.0.3 (based on iText 2.0.2)</pdf:Producer>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`
	parseFails(t, s, "Schema is not set in this document : "+
		"http://www.w3.org/XML/1998/namespace, property: xml:ModifyDate")
}

func TestNoInstantiation(t *testing.T) {
	// Instantiation fails because of a bad date property
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='` + bom + `' id='W5M0MpCehiHzreSzNTczkc9d'?><?adobe-xap-filters esc="CRLF"?><x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="XMP toolkit 2.9.1-13, framework 1.6">
    <rdf:RDF xmlns:iX="http://ns.adobe.com/iX/1.0/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description xmlns:xmp="http://ns.adobe.com/xap/1.0/" rdf:about="uuid:f577a812-a531-11f4-0000-2eba1231b686">
            <xmp:CreateDate>2019-05-02T22:03:5Z</xmp:CreateDate>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket end='w'?>`
	parseFails(t, s, "Failed to instantiate DateType property with value "+
		"'2019-05-02T22:03:5Z' in xmp:CreateDate")
}

func TestNoInstantiation2(t *testing.T) {
	// Instantiation fails because of a bad date attribute
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='` + bom + `' id='W5M0MpCehiHzreSzNTczkc9d'?><?adobe-xap-filters esc="CRLF"?><x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="XMP toolkit 2.9.1-13, framework 1.6">
    <rdf:RDF xmlns:iX="http://ns.adobe.com/iX/1.0/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description xmlns:xap="http://ns.adobe.com/xap/1.0/" xap:CreateDate="2016-03-09T19:47:1Z">
            <xap:CreatorTool>PrimoPDF http://www.primopdf.com</xap:CreatorTool>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket end='w'?>`
	parseFails(t, s, "Failed to instantiate DateType property with value "+
		"'2016-03-09T19:47:1Z' in xap:CreateDate")
}

func TestPDFBox6131(t *testing.T) {
	// Contains "Open Choice of Integer" instead of "open Choice of Integer"
	xmp, err := xmpxml.NewDomXmpParser().ParseBytes(
		readFixture(t, "org/apache/xmpbox/xml/PDFBOX-6131-0015675.xml"))
	noError(t, "parse", err)
	assertUAPart(t, xmp)
}

// TestWrongType covers a packet that worked in 3.0.6 in lenient mode and
// temporarily stopped working while getSpecifiedPropertyType was changed in
// PDFBOX-6133. "photoshop:headline" should be a text, not a Seq, and it is
// "Headline" with a capital H; the cause is that the photoshop namespace exists
// both as a schema and as a type.
func TestWrongType(t *testing.T) {
	// from file 000367.pdf, USAID document
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?><x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Adobe XMP Core 4.0-c316 44.253921, Sun Oct 01 2006 17:14:39">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description xmlns:photoshop="http://ns.adobe.com/photoshop/1.0/" rdf:about="">
            <photoshop:headline>
                <rdf:Seq>
                    <rdf:li/>
                </rdf:Seq>
            </photoshop:headline>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`
	parseFails(t, s, "No type defined for {http://ns.adobe.com/photoshop/1.0/}headline")

	xmp2 := parseLenient(t, s)
	photoshop := xmp2.PhotoshopSchema()
	if photoshop == nil {
		t.Fatal("PhotoshopSchema() = nil")
	}
	equal(t, "Headline()", photoshop.Headline(), "")
	// non existant properties are treated as text, one might want to change
	// this in the future.
	equal(t, "headline", propertyString(t, "headline", photoshop.Property("headline")),
		"[headline=TextType:]")
}

func TestPDFBox61312(t *testing.T) {
	// Contains "Seq Text" instead of "seq Text" and "Bag Text" instead of
	// "bag Text", from file RMR6DEEUWZO6IM3A7WKRPX33SZMBTTQZ, Fairfax County
	// Office of Community Revitalization.
	xmp, err := xmpxml.NewDomXmpParser().ParseBytes(
		readFixture(t, "org/apache/xmpbox/xml/PDFBOX-6131-RMR6DEEUWZO6IM3A7WKRPX33SZMBTTQZ.xml"))
	noError(t, "parse", err)
	pdfaid := xmp.PDFAIdentificationSchema()
	if pdfaid == nil {
		t.Fatal("PDFAIdentificationSchema() = nil")
	}
	part, held := pdfaid.Part()
	if !held || part != 1 {
		t.Errorf("Part() = %v, %v, want 1", part, held)
	}
}

func TestPDFBox6133(t *testing.T) {
	// Namespace is used both for the schema and the type, and there are two
	// types with the same namespace.
	xmp, err := xmpxml.NewDomXmpParser().ParseBytes(
		readFixture(t, "org/apache/xmpbox/xml/PDFBOX-6133-0064638.xml"))
	noError(t, "parse", err)
	assertEPASchema(t, xmp)

	// Serialize and repeat to ensure nothing was broken in serialization
	written := serialized(t, xmp)
	again, err := xmpxml.NewDomXmpParser().ParseBytes(written)
	noError(t, "parse", err)
	assertEPASchema(t, again)
}

// assertEPASchema is the block of assertions PDFBOX-6133 makes before and after
// the round trip.
func assertEPASchema(t *testing.T, xmp *xmpbox.XMPMetadata) {
	t.Helper()
	epaSchema := xmp.Schema("http://www.epo.org/patent-bibliographic-data/1.0/")
	if epaSchema == nil {
		t.Fatal("Schema(epa) = nil")
	}
	base := epaSchema.Base()
	equal(t, "TotalNumberOfPages",
		propertyString(t, "TotalNumberOfPages", base.Property("TotalNumberOfPages")),
		"[TotalNumberOfPages=RealType:47.0]")

	pub, isDefined := base.Property("Publication").(*xmptype.DefinedStructuredType)
	if !isDefined {
		t.Fatalf("Publication = %T, want a defined structured type",
			base.Property("Publication"))
	}
	equal(t, "CountryCode",
		propertyString(t, "CountryCode", pub.Property("CountryCode")),
		"[CountryCode=TextType:EP]")

	classification, isArray := base.Property("Classification").(*xmptype.ArrayProperty)
	if !isArray {
		t.Fatalf("Classification = %T, want an array", base.Property("Classification"))
	}
	if len(classification.AllProperties()) != 4 {
		t.Fatalf("Classification held %d values, want 4",
			len(classification.AllProperties()))
	}
	class3, isText := classification.AllProperties()[3].(*xmptype.TextType)
	if !isText {
		t.Fatalf("Classification[3] = %T, want text", classification.AllProperties()[3])
	}
	equal(t, "Classification[3]", class3.StringValue(),
		"A61K 39/215 20060101ALI20160203BHEP")

	title, err := base.UnqualifiedLanguagePropertyValue("Title", "de")
	noError(t, "UnqualifiedLanguagePropertyValue", err)
	equal(t, "Title(de)", title, "CORONAVIRUS")

	documentStructure, isArray := base.Property("DocumentStructure").(*xmptype.ArrayProperty)
	if !isArray {
		t.Fatalf("DocumentStructure = %T, want an array", base.Property("DocumentStructure"))
	}
	if len(documentStructure.AllProperties()) != 5 {
		t.Fatalf("DocumentStructure held %d values, want 5",
			len(documentStructure.AllProperties()))
	}
	struct4, isDefined := documentStructure.AllProperties()[4].(*xmptype.DefinedStructuredType)
	if !isDefined {
		t.Fatalf("DocumentStructure[4] = %T, want a defined structured type",
			documentStructure.AllProperties()[4])
	}
	equal(t, "DocumentSection",
		propertyString(t, "DocumentSection", struct4.Property("DocumentSection")),
		"[DocumentSection=TextType:cited-references]")
	equal(t, "StartPage", propertyString(t, "StartPage", struct4.Property("StartPage")),
		"[StartPage=RealType:47.0]")
	equal(t, "NumberOfPages",
		propertyString(t, "NumberOfPages", struct4.Property("NumberOfPages")),
		"[NumberOfPages=RealType:1.0]")
}

func TestPropertyNotDefined2(t *testing.T) {
	// from file 089448.pdf, page 2, image 4
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='` + bom + `' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x="adobe:ns:meta/"
           x:xmptk="Adobe XMP Core 4.0-c006 1.236519, Wed Jun 14 2006 08:31:24">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/"
		                 xmlns:exif="http://ns.adobe.com/exif/1.0/">
			<exif:CFAPattern>
				<rdf:Description>
					<exif:Values>
						<rdf:Seq>
							<rdf:li>1</rdf:li>
							<rdf:li>2</rdf:li>
							<rdf:li>0</rdf:li>
							<rdf:li>1</rdf:li>
						</rdf:Seq>
					</exif:Values>
				</rdf:Description>
			</exif:CFAPattern>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end='w'?>`
	xmp := parse(t, s)
	exif := xmp.Schema(schema.ExifNamespace)
	if exif == nil {
		t.Fatal("Schema(exif) = nil")
	}
	cfa, isPattern := exif.Base().Property(schema.ExifCFAPattern).(*xmptype.CFAPatternType)
	if !isPattern {
		t.Fatalf("CFAPattern = %T, want a CFA pattern",
			exif.Base().Property(schema.ExifCFAPattern))
	}
	ap, isArray := cfa.Property(xmptype.CFAPatternValues).(*xmptype.ArrayProperty)
	if !isArray {
		t.Fatalf("Values = %T, want an array", cfa.Property(xmptype.CFAPatternValues))
	}
	// Java asserts the List.toString of the element values.
	equal(t, "Values", fmt.Sprintf("[%s]", strings.Join(ap.ElementsAsString(), ", ")),
		"[1, 2, 0, 1]")
}

// TestPDFBox6136 covers the corner case of an extension schema declared with
// both "xmlns=" and "xmlns:pdfaExtension=".
func TestPDFBox6136(t *testing.T) {
	// File 0018804.pdf (Italian parliament)
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d' bytes='6865'?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:iX="http://ns.adobe.com/iX/1.0/">
    <rdf:Description xmlns="http://www.aiim.org/pdfa/ns/extension/" xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/" xmlns:pdfaProperty="http://www.aiim.org/pdfa/ns/property#" xmlns:pdfaSchema="http://www.aiim.org/pdfa/ns/schema#" about="">
        <pdfaExtension:schemas>
            <rdf:Bag>
                <rdf:li rdf:parseType="Resource">
                    <pdfaSchema:namespaceURI>http://ns.adobe.com/pdfx/1.3/</pdfaSchema:namespaceURI>
                    <pdfaSchema:prefix>pdfx</pdfaSchema:prefix>
                    <pdfaSchema:schema>Adobe Document Info PDF eXtension Schema</pdfaSchema:schema>
                    <pdfaSchema:property>
                        <rdf:Seq>
                            <rdf:li rdf:parseType="Resource">
                                <pdfaProperty:category>internal</pdfaProperty:category>
                                <pdfaProperty:description>ID of PDF/X standard</pdfaProperty:description>
                                <pdfaProperty:name>GTS_PDFXVersion</pdfaProperty:name>
                                <pdfaProperty:valueType>Text</pdfaProperty:valueType>
                            </rdf:li>
                            <rdf:li rdf:parseType="Resource">
                                <pdfaProperty:category>internal</pdfaProperty:category>
                                <pdfaProperty:description>Conformance level of PDF/X standard</pdfaProperty:description>
                                <pdfaProperty:name>GTS_PDFXConformance</pdfaProperty:name>
                                <pdfaProperty:valueType>Text</pdfaProperty:valueType>
                            </rdf:li>
                            <rdf:li rdf:parseType="Resource">
                                <pdfaProperty:category>internal</pdfaProperty:category>
                                <pdfaProperty:description>Company creating the PDF</pdfaProperty:description>
                                <pdfaProperty:name>Company</pdfaProperty:name>
                                <pdfaProperty:valueType>Text</pdfaProperty:valueType>
                            </rdf:li>
                            <rdf:li rdf:parseType="Resource">
                                <pdfaProperty:category>internal</pdfaProperty:category>
                                <pdfaProperty:description>Date when document was last modified</pdfaProperty:description>
                                <pdfaProperty:name>SourceModified</pdfaProperty:name>
                                <pdfaProperty:valueType>Text</pdfaProperty:valueType>
                            </rdf:li>
                        </rdf:Seq>
                    </pdfaSchema:property>
                </rdf:li>
                <rdf:li rdf:parseType="Resource">
                    <pdfaSchema:namespaceURI>http://ns.adobe.com/xap/1.0/mm/</pdfaSchema:namespaceURI>
                    <pdfaSchema:prefix>xmpMM</pdfaSchema:prefix>
                    <pdfaSchema:schema>XMP Media Management Schema</pdfaSchema:schema>
                    <pdfaSchema:property>
                        <rdf:Seq>
                            <rdf:li rdf:parseType="Resource">
                                <pdfaProperty:category>internal</pdfaProperty:category>
                                <pdfaProperty:description>UUID based identifier for specific incarnation of a document</pdfaProperty:description>
                                <pdfaProperty:name>InstanceID</pdfaProperty:name>
                                <pdfaProperty:valueType>URI</pdfaProperty:valueType>
                            </rdf:li>
                            <rdf:li rdf:parseType="Resource">
                                <pdfaProperty:category>internal</pdfaProperty:category>
                                <pdfaProperty:description>The common identifier for all versions and renditions of a document.</pdfaProperty:description>
                                <pdfaProperty:name>OriginalDocumentID</pdfaProperty:name>
                                <pdfaProperty:valueType>URI</pdfaProperty:valueType>
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
                                <pdfaProperty:description>Amendment of PDF/A standard</pdfaProperty:description>
                                <pdfaProperty:name>amd</pdfaProperty:name>
                                <pdfaProperty:valueType>Text</pdfaProperty:valueType>
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
    <rdf:Description xmlns="http://www.aiim.org/pdfa/ns/id/" xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/" about="">
        <pdfaid:part>1</pdfaid:part>
        <pdfaid:conformance>B</pdfaid:conformance>
    </rdf:Description>
</rdf:RDF><?xpacket end='r'?>`
	xmp := parseLenient(t, s)
	pdfaid := xmp.PDFAIdentificationSchema()
	if pdfaid == nil {
		t.Fatal("PDFAIdentificationSchema() = nil")
	}
	equal(t, "Conformance()", pdfaid.Conformance(), "B")
	part, held := pdfaid.Part()
	if !held || part != 1 {
		t.Errorf("Part() = %v, %v, want 1", part, held)
	}
}

// TestNamespaceInRoot covers PDFBOX-6138: the namespaces are declared on the
// root element rather than on rdf:RDF or deeper.
func TestNamespaceInRoot(t *testing.T) {
	s := `<?xml version="1.0" encoding="utf-8" standalone="no"?>
<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d'?><x:xmpmeta xmlns:x="adobe:ns:meta/" xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/" xmlns:pdfaProperty="http://www.aiim.org/pdfa/ns/property#" xmlns:pdfaSchema="http://www.aiim.org/pdfa/ns/schema#" xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" x:xmptk="Adobe XMP Core 5.6-c015 91.163280, 2018/06/22-11:31:03        ">
    <rdf:RDF>
        <rdf:Description rdf:about="">
            <pdfaExtension:schemas>
                <rdf:Bag>
                    <rdf:li rdf:parseType="Resource">
                        <pdfaSchema:schema>PDF/UA Universal Accessibility Schema</pdfaSchema:schema>
                        <pdfaSchema:namespaceURI>http://www.aiim.org/pdfua/ns/id/</pdfaSchema:namespaceURI>
                        <pdfaSchema:prefix>pdfuaid</pdfaSchema:prefix>
                        <pdfaSchema:property>
                            <rdf:Seq>
                                <rdf:li rdf:parseType="Resource">
                                    <pdfaProperty:name>part</pdfaProperty:name>
                                    <pdfaProperty:valueType>Integer</pdfaProperty:valueType>
                                    <pdfaProperty:category>internal</pdfaProperty:category>
                                    <pdfaProperty:description>Indicates, which part of ISO 14289 standard is followed</pdfaProperty:description>
                                </rdf:li>
                            </rdf:Seq>
                        </pdfaSchema:property>
                    </rdf:li>
                </rdf:Bag>
            </pdfaExtension:schemas>
            <pdfuaid:part>1</pdfuaid:part>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket end='w'?>`
	assertUAPart(t, parse(t, s))
}
