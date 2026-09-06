package xml_test

// Port of org.apache.xmpbox.xml.DomXmpParserTest, second part: the cases about
// what the parser refuses and what it makes of the same packet in lenient mode.

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// propertyString returns a property's Java toString, and "null" where there is
// none.
func propertyString(t *testing.T, what string, property xmptype.AbstractField) string {
	t.Helper()
	simple, isSimple := property.(xmptype.AbstractSimpleProperty)
	if !isSimple {
		t.Fatalf("%s = %T, want a simple property", what, property)
	}
	return stringOfProperty(simple)
}

func TestBadAttr(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
           <rdf:Description xmlns:xmpTPg="http://ns.adobe.com/xap/1.0/t/pg/"                            xmlns:stDim="http://ns.adobe.com/xap/1.0/sType/Dimensions#"		                 rdf:about="">
			<xmpTPg:MaxPageSize>
				<rdf:Description stDim:X="4" stDim:Y="3" stDim:Z="inch"/>
			</xmpTPg:MaxPageSize>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end="r"?>`
	parseFails(t, s,
		"No type defined for {http://ns.adobe.com/xap/1.0/sType/Dimensions#}X")

	xmp := parseLenient(t, s)
	pageText := xmp.PageTextSchema()
	if pageText == nil {
		t.Fatal("PageTextSchema() = nil")
	}
	dim, isDimensions := pageText.Property(schema.PageTextMaxPageSize).(*xmptype.DimensionsType)
	if !isDimensions {
		t.Fatalf("MaxPageSize = %T, want dimensions",
			pageText.Property(schema.PageTextMaxPageSize))
	}
	equal(t, "MaxPageSize", dim.String(), "DimensionsType{null x null null}")
	equal(t, "X", propertyString(t, "X", dim.Property("X")), "[X=TextType:4]")
	equal(t, "Y", propertyString(t, "Y", dim.Property("Y")), "[Y=TextType:3]")
	equal(t, "Z", propertyString(t, "Z", dim.Property("Z")), "[Z=TextType:inch]")
}

func TestBadType(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d'?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns:iX="http://ns.adobe.com/iX/1.0/">
	<rdf:Description xmlns="http://ns.adobe.com/pdf/1.3/"
	                 xmlns:pdf="http://ns.adobe.com/pdf/1.3/"
	                 about=""
	                 pdf:Author="edocslib"/>
</rdf:RDF>
<?xpacket end='r'?>`
	parseFails(t, s, "No type defined for {http://ns.adobe.com/pdf/1.3/}Author")

	xmp := parseLenient(t, s)
	pdf := xmp.AdobePDFSchema()
	if pdf == nil {
		t.Fatal("AdobePDFSchema() = nil")
	}
	equal(t, "Author", propertyString(t, "Author", pdf.Property("Author")),
		"[Author=TextType:edocslib]")
}

func TestBadType2(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/"
           x:xmptk="3.1.1-111">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description xmlns:pdf="http://ns.adobe.com/pdf/1.3/"
		                 rdf:about="">
			<pdf:Bad>Value</pdf:Bad>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end="r"?>`
	parseFails(t, s, "No type defined for {http://ns.adobe.com/pdf/1.3/}Bad")

	xmp := parseLenient(t, s)
	value, err := xmp.AdobePDFSchema().UnqualifiedTextPropertyValue("Bad")
	noError(t, "UnqualifiedTextPropertyValue", err)
	equal(t, "Bad", value, "Value")
}

func TestBadLocalName(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='` + bom + `' id='W5M0MpCehiHzreSzNTczkc9d'?><?adobe-xap-filters esc="CR"?>
<x:xapmeta xmlns:x="adobe:ns:meta/">
	<rdf:RDF xmlns:iX="http://ns.adobe.com/iX/1.0/" xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
	</rdf:RDF>
</x:xapmeta><?xpacket end='w'?>`
	parseFails(t, s, "Expecting local name 'xmpmeta' and found 'xapmeta'")

	xmp2 := parseLenient(t, s)
	if schemas := xmp2.AllSchemas(); len(schemas) != 0 {
		t.Errorf("AllSchemas() = %v, want none", schemas)
	}
}

func TestBadXPacketEnd1(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d" ?><x:xmpmeta xmlns:x="adobe:ns:meta/">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/" rdf:about="">
            <dc:format>application/pdf</dc:format>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket ends="w" ?>`
	parseFails(t, s,
		"Expected xpacket 'end' attribute (must be present and placed in first)")
}

func TestBadXPacketEnd2(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d" ?><x:xmpmeta xmlns:x="adobe:ns:meta/">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/" rdf:about="">
            <dc:format>application/pdf</dc:format>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket end="k" ?>`
	parseFails(t, s, "Expected xpacket 'end' attribute with value 'r' or 'w' ")
}

func TestNoRdfChildren(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d" ?>  <x:xmpmeta xmlns:x="adobe:ns:meta/"/>
<?xpacket end="w" ?>`
	parseFails(t, s, "No rdf description found in xmp")
}

func TestTextInsteadOfArray(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/"
           x:xmptk="3.1-701">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/"
		                 rdf:about="">
			<dc:title>Title</dc:title>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`
	parseFails(t, s,
		"Invalid array definition, expecting Alt and found Text [prefix=dc; name=title]")
}

func TestPropertyNotDefined(t *testing.T) {
	// While "Fired" does exist as a type, it's not the correct syntax, the
	// PDFLib XMP validator complains too. Surprisingly, it works since
	// PDFBOX-6133.
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='` + bom + `' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x="adobe:ns:meta/"
           x:xmptk="XMP toolkit 3.0-28, framework 1.6">
	<rdf:RDF xmlns:iX="http://ns.adobe.com/iX/1.0/"
	         xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description xmlns:exif="http://ns.adobe.com/exif/1.0/"
		                 rdf:about="uuid:d9974396-53ee-11d9-9542-81b7ec7f4613">
			<exif:Flash rdf:parseType="Resource">
				<exif:Fired>False</exif:Fired>
			</exif:Flash>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end='w'?>`
	xmp := parse(t, s)
	exif := xmp.Schema(schema.ExifNamespace)
	if exif == nil {
		t.Fatal("Schema(exif) = nil")
	}
	flash, isFlash := exif.Base().Property(schema.ExifFlash).(*xmptype.FlashType)
	if !isFlash {
		t.Fatalf("Flash = %T, want a flash", exif.Base().Property(schema.ExifFlash))
	}
	equal(t, "Fired", propertyString(t, "Fired", flash.Property(xmptype.FlashFired)),
		"[Fired=BooleanType:False]")
}

func TestBadAttr2(t *testing.T) {
	// File from image on page 14 from file 006054.pdf. exif:Flash is a
	// structured type; however the PDFLib XMP validator approves the file.
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='` + bom + `' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x="adobe:ns:meta/"
           x:xmptk="XMP toolkit 2.9.1-13, framework 1.6">
	<rdf:RDF xmlns:iX="http://ns.adobe.com/iX/1.0/"
	         xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description xmlns:exif="http://ns.adobe.com/exif/1.0/"
		                 exif:FNumber="36/10"
		                 exif:FileSource="3"
		                 exif:Flash="1"
		                 rdf:about="">
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end='r'?>`
	parseFails(t, s, "The type 'Flash' in 'exif:Flash=1' is a structured or array "+
		"type, but attributes are simple types")

	xmp := parseLenient(t, s)
	exif := xmp.Schema(schema.ExifNamespace)
	if exif == nil {
		t.Fatal("Schema(exif) = nil")
	}
	equal(t, "Flash", propertyString(t, "Flash", exif.Base().Property(schema.ExifFlash)),
		"[Flash=TextType:1]")
}

func TestBadAttr3(t *testing.T) {
	// test text in attribute which should have been an array property
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d' bytes='1064'?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
    <rdf:Description xmlns="http://purl.org/dc/elements/1.1/" xmlns:dc="http://purl.org/dc/elements/1.1/" about="" dc:creator="Creator" />
</rdf:RDF><?xpacket end='r'?>`
	parseFails(t, s, "The type 'Text' in 'dc:creator=Creator' is a structured or "+
		"array type, but attributes are simple types")

	xmp2 := parseLenient(t, s)
	// make sure that nothing is lost in serialization
	written := serialized(t, xmp2)
	parseFileFails(t, written,
		"Invalid array definition, expecting Seq and found Text [prefix=dc; name=creator]")

	parser4 := xmpxml.NewDomXmpParser()
	parser4.SetStrictParsing(false)
	xmp4, err := parser4.ParseBytes(written)
	noError(t, "parse", err)
	dc := xmp4.DublinCoreSchema()
	if dc == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	equal(t, "creator", propertyString(t, "creator", dc.Property(schema.DCCreator)),
		"[creator=TextType:Creator]")
}

// TestBadAttr4 tests an empty attribute where an array is expected, which is
// skipped in lenient mode.
func TestBadAttr4(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d' bytes='1206'?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" >
    <rdf:Description xmlns="http://purl.org/dc/elements/1.1/" xmlns:dc="http://purl.org/dc/elements/1.1/" about="" dc:creator="">
        <dc:coverage>Coverage</dc:coverage>
    </rdf:Description>
</rdf:RDF><?xpacket end='r'?>`
	parseFails(t, s, "The type 'Text' in 'dc:creator=' is a structured or array "+
		"type, but attributes are simple types")

	xmp2 := parseLenient(t, s)
	dc2 := xmp2.DublinCoreSchema()
	if dc2 == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	equal(t, "Coverage()", dc2.Coverage(), "Coverage")
	if creator := dc2.Property(schema.DCCreator); creator != nil {
		t.Errorf("creator = %v, want nothing", creator)
	}

	written := serialized(t, xmp2)
	parser3 := xmpxml.NewDomXmpParser()
	parser3.SetStrictParsing(false)
	xmp3, err := parser3.ParseBytes(written)
	noError(t, "parse", err)
	dc3 := xmp3.DublinCoreSchema()
	if dc3 == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	equal(t, "Coverage()", dc3.Coverage(), "Coverage")
	if creator := dc3.Property(schema.DCCreator); creator != nil {
		t.Errorf("creator = %v, want nothing", creator)
	}
}

// TestBadAttr5 tests an empty attribute where a language alternative is
// expected, which is skipped in lenient mode.
func TestBadAttr5(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d' bytes='987'?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:iX="http://ns.adobe.com/iX/1.0/">
    <rdf:Description xmlns="http://purl.org/dc/elements/1.1/" xmlns:dc="http://purl.org/dc/elements/1.1/" about="" dc:title="" dc:coverage="COVER"/>
</rdf:RDF><?xpacket end='r'?>`
	parseFails(t, s, "The type 'LangAlt' in 'dc:title=' is a structured or array "+
		"type, but attributes are simple types")

	xmp2 := parseLenient(t, s)
	dc2 := xmp2.DublinCoreSchema()
	if dc2 == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	title, err := dc2.Title()
	noError(t, "Title", err)
	equal(t, "Title()", title, "")
	if held := dc2.Property(schema.DCTitle); held != nil {
		t.Errorf("title = %v, want nothing", held)
	}
	equal(t, "Coverage()", dc2.Coverage(), "COVER")

	written := serialized(t, xmp2)
	parser3 := xmpxml.NewDomXmpParser()
	parser3.SetStrictParsing(false)
	xmp3, err := parser3.ParseBytes(written)
	noError(t, "parse", err)
	dc3 := xmp3.DublinCoreSchema()
	if dc3 == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	title, err = dc3.Title()
	noError(t, "Title", err)
	equal(t, "Title()", title, "")
	if held := dc3.Property(schema.DCTitle); held != nil {
		t.Errorf("title = %v, want nothing", held)
	}
	equal(t, "Coverage()", dc3.Coverage(), "COVER")
}

func TestBadSchema(t *testing.T) {
	// from file 130841.pdf: a structured type used like a schema
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='` + bom + `' id='W5M0MpCehiHzreSzNTczkc9d'?><?adobe-xap-filters esc="CRLF"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/"
           x:xmptk="XMP toolkit">
	<rdf:RDF xmlns:iX="http://ns.adobe.com/iX/1.0/"
	         xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description xmlns:stJob="http://ns.adobe.com/xap/1.0/sType/Job#"
		                 rdf:about="uuid"
		                 stJob:id="jobid"
		                 stJob:name="some name">
			<stJob:URL>https://pdfbox.apache.org</stJob:URL>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end='w'?>`
	parseFails(t, s,
		"This namespace is not from a schema: http://ns.adobe.com/xap/1.0/sType/Job#")
}

func TestPDFBox6126(t *testing.T) {
	// XMP originally from PDFBOX-4325, had this exception:
	// Cannot find a definition for the namespace
	// http://www.w3.org/1999/02/22-rdf-syntax-ns#, property: rdf:Description
	// Cause: "<rdf:Description" as child of <rdf:li .
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/"
           x:xmptk="Adobe XMP Core 5.1.0-jc003">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
		<rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/"
		                 xmlns:pdf="http://ns.adobe.com/pdf/1.3/"
		                 xmlns:pdfaExtension="http://www.aiim.org/pdfa/ns/extension/"
		                 xmlns:pdfaProperty="http://www.aiim.org/pdfa/ns/property#"
		                 xmlns:pdfaSchema="http://www.aiim.org/pdfa/ns/schema#"
		                 xmlns:pdfaid="http://www.aiim.org/pdfa/ns/id/"
		                 xmlns:pdfuaid="http://www.aiim.org/pdfua/ns/id/"
		                 xmlns:xmp="http://ns.adobe.com/xap/1.0/"
		                 dc:format="application/pdf"
		                 pdf:Producer="iText® 5.5.13 ©2000-2018 iText Group NV (AGPL-version)"
		                 pdfaid:conformance="B"
		                 pdfaid:part="1"
		                 rdf:about=""
		                 xmp:CreateDate="2018-09-24T09:00:57+02:00"
		                 xmp:ModifyDate="2018-09-24T09:00:57+02:00">
			<pdfaExtension:schemas>
				<rdf:Bag>
					<rdf:li rdf:parseType="Resource">
						<rdf:Description pdfaSchema:namespaceURI="http://www.aiim.org/pdfua/ns/id/"
						                 pdfaSchema:prefix="pdfuaid"
						                 pdfaSchema:schema="PDF/UA identification schema">
							<pdfaSchema:property>
								<rdf:Seq>
									<rdf:li pdfaProperty:category="internal"
									        pdfaProperty:description="PDF/UA version identifier"
									        pdfaProperty:name="part"
									        pdfaProperty:valueType="Integer"/>
									<rdf:li pdfaProperty:category="internal"
									        pdfaProperty:description="PDF/UA amendment identifier"
									        pdfaProperty:name="amd"
									        pdfaProperty:valueType="Text"/>
									<rdf:li pdfaProperty:category="internal"
									        pdfaProperty:description="PDF/UA corrigenda identifier"
									        pdfaProperty:name="corr"
									        pdfaProperty:valueType="Text"/>
								</rdf:Seq>
							</pdfaSchema:property>
						</rdf:Description>
					</rdf:li>
				</rdf:Bag>
			</pdfaExtension:schemas>
			<pdfuaid:part>1</pdfuaid:part>
		</rdf:Description>
	</rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`
	xmp1 := parse(t, s)
	assertUAPart(t, xmp1)

	// make sure that nothing is lost in serialization
	written := serialized(t, xmp1)
	xmp2, err := xmpxml.NewDomXmpParser().ParseBytes(written)
	noError(t, "parse", err)
	assertUAPart(t, xmp2)
}

// assertUAPart is the assertion PDFBOX-6126 makes before and after the round
// trip.
func assertUAPart(t *testing.T, xmp *xmpbox.XMPMetadata) {
	t.Helper()
	uaSchema := xmp.Schema("http://www.aiim.org/pdfua/ns/id/")
	if uaSchema == nil {
		t.Fatal("Schema(pdfuaid) = nil")
	}
	part, held, err := uaSchema.Base().IntegerPropertyValueAsSimple("part")
	noError(t, "IntegerPropertyValueAsSimple", err)
	if !held || part != 1 {
		t.Errorf("part = %v, %v, want 1", part, held)
	}
}

// TestNonStandardURIinRDF checks, for PDFBOX-6127, that a non-standard
// namespace is not recognised when it is declared on rdf:RDF, which happens
// because of the change XmpSerializer took in PDFBOX-2378.
func TestNonStandardURIinRDF(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?><x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Adobe XMP Core 4.2.1-c041 52.342996, 2008/05/07-20:48:00        ">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description xmlns:pdfx="http://ns.adobe.com/pdfx/1.3/" rdf:about="">
            <pdfx:XPressPrivate>private</pdfx:XPressPrivate>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`
	const message = "Cannot find a definition for the namespace " +
		"http://ns.adobe.com/pdfx/1.3/, property: pdfx:XPressPrivate"
	parseFails(t, s, message)

	xmp2 := parseLenient(t, s)
	assertXPressPrivate(t, xmp2)

	written := serialized(t, xmp2)
	// make sure that there is a non standard namespace in rdf:RDF
	if !strings.Contains(string(written), "<rdf:RDF xmlns:pdfx=") {
		t.Errorf("serialized packet has no non standard namespace on rdf:RDF:\n%s", written)
	}

	parseFileFails(t, written, message)

	parser4 := xmpxml.NewDomXmpParser()
	parser4.SetStrictParsing(false)
	xmp4, err := parser4.ParseBytes(written)
	noError(t, "parse", err)
	assertXPressPrivate(t, xmp4)
}

// assertXPressPrivate is the assertion the non-standard namespace case makes
// before and after the round trip.
func assertXPressPrivate(t *testing.T, xmp *xmpbox.XMPMetadata) {
	t.Helper()
	found := xmp.Schema("http://ns.adobe.com/pdfx/1.3/")
	if found == nil {
		t.Fatal("Schema(pdfx) = nil")
	}
	equal(t, "XPressPrivate",
		propertyString(t, "XPressPrivate", found.Base().Property("XPressPrivate")),
		"[XPressPrivate=TextType:private]")
}

// TestBadProp tests an empty property where an array is expected, which is
// skipped in lenient mode.
func TestBadProp(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d' bytes='1506'?><rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#" xmlns:iX="http://ns.adobe.com/iX/1.0/">
    <rdf:Description xmlns="http://purl.org/dc/elements/1.1/" xmlns:dc="http://purl.org/dc/elements/1.1/" about="">
        <dc:creator/>
        <dc:coverage>Cover</dc:coverage>
    </rdf:Description>
</rdf:RDF><?xpacket end='r'?>`
	parseFails(t, s,
		"Invalid array definition, expecting Seq and found nothing [prefix=dc; name=creator]")

	xmp2 := parseLenient(t, s)
	assertCoverOnly(t, xmp2)

	written := serialized(t, xmp2)
	parser3 := xmpxml.NewDomXmpParser()
	parser3.SetStrictParsing(false)
	xmp3, err := parser3.ParseBytes(written)
	noError(t, "parse", err)
	assertCoverOnly(t, xmp3)
}

// assertCoverOnly is the assertion the empty-array case makes before and after
// the round trip.
func assertCoverOnly(t *testing.T, xmp *xmpbox.XMPMetadata) {
	t.Helper()
	dc := xmp.DublinCoreSchema()
	if dc == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	if creators := dc.Creators(); creators != nil {
		t.Errorf("Creators() = %v, want nothing", creators)
	}
	if held := dc.Property(schema.DCCreator); held != nil {
		t.Errorf("creator = %v, want nothing", held)
	}
	equal(t, "Coverage()", dc.Coverage(), "Cover")
}

func TestBadProp2(t *testing.T) {
	// PDFBOX-6135: from file 000316.pdf, stRef:documentName isn't defined
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?><x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="3.1-701">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description xmlns:stRef="http://ns.adobe.com/xap/1.0/sType/ResourceRef#" xmlns:xapMM="http://ns.adobe.com/xap/1.0/mm/" rdf:about="">
            <xapMM:DocumentID>uuid:CE03288B61A6DB11A55CA11F14F48514</xapMM:DocumentID>
            <xapMM:InstanceID>uuid:474647e9-680a-47dc-83d5-ba3f3a7e2a67</xapMM:InstanceID>
            <xapMM:DerivedFrom rdf:parseType="Resource">
                <stRef:documentName>uuid:8705447f-b80d-4cc8-82f7-0ec27187edfe</stRef:documentName>
                <stRef:documentID>uuid:b2f88223-2723-430d-b93c-3503ccb0e34b</stRef:documentID>
            </xapMM:DerivedFrom>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`
	parseFails(t, s, "Type 'stRef:documentName' not defined in "+
		"http://ns.adobe.com/xap/1.0/sType/ResourceRef#")

	xmp2 := parseLenient(t, s)
	mm := xmp2.XMPMediaManagementSchema()
	if mm == nil {
		t.Fatal("XMPMediaManagementSchema() = nil")
	}
	derived := mm.DerivedFromProperty()
	if derived == nil {
		t.Fatal("DerivedFromProperty() = nil")
	}
	equal(t, "DocumentID()", derived.DocumentID(),
		"uuid:b2f88223-2723-430d-b93c-3503ccb0e34b")
	equal(t, "documentName",
		propertyString(t, "documentName", derived.Property("documentName")),
		"[documentName=TextType:uuid:8705447f-b80d-4cc8-82f7-0ec27187edfe]")
}

func TestParseFailure(t *testing.T) {
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>`
	_, err := xmpxml.NewDomXmpParser().ParseBytes([]byte(s))
	if err == nil {
		t.Fatal("parse reported nothing, want a failure")
	}
	if !strings.HasPrefix(err.Error(), "Failed to parse: ") {
		t.Errorf("parse = %q, want it to start with %q", err.Error(), "Failed to parse: ")
	}
}

func TestNoXPacket(t *testing.T) {
	// must be "xpacket", not "packet"
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?packet begin="" id="W5M0MpCehiHzreSzNTczkc9d"?><x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="3.1-701">
</x:xmpmeta><?packet end="w"?>`
	parseFails(t, s, "Bad processing instruction name : packet")
}

func TestDoubleEnd(t *testing.T) {
	s := `<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d'?>
<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x="adobe:ns:meta/"
           x:xmptk="Adobe XMP Core 4.0-c316 44.253921, Sun Oct 01 2006 17:14:39">
	<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
	</rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>
<?xpacket end='r'?> `
	parseFails(t, s, "xmp should end after xpacket end processing instruction")
}

func TestBadInner(t *testing.T) {
	// file has "xmpMM:parseType". Changing this to "rdf:parseType" makes it
	// work.
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?><x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Adobe XMP Core 5.2-c001 63.139439, 2010/09/27-13:37:26        ">
    <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
        <rdf:Description xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:pdf="http://ns.adobe.com/pdf/1.3/" xmlns:photoshop="http://ns.adobe.com/photoshop/1.0/" xmlns:stEvt="http://ns.adobe.com/xap/1.0/sType/ResourceEvent#" xmlns:stRef="http://ns.adobe.com/xap/1.0/sType/ResourceRef#"  xmlns:xmpMM="http://ns.adobe.com/xap/1.0/mm/" xmlns:xmpRights="http://ns.adobe.com/xap/1.0/rights/">
            <xmpMM:DerivedFrom xmpMM:parseType="Resource">
                <stRef:instanceID>uuid:6b838c4d-07e2-0611-2333-558805f93988</stRef:instanceID>
                <stRef:documentID>uuid:6b838c4d-07e2-0611-2333-558805f93988</stRef:documentID>
            </xmpMM:DerivedFrom>
        </rdf:Description>
    </rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`
	parseFails(t, s,
		"inner element should contain child elements : [stRef:instanceID: null]")

	s2 := strings.ReplaceAll(s, "xmpMM:parseType", "rdf:parseType")
	xmp2 := parse(t, s2)
	mm := xmp2.XMPMediaManagementSchema()
	if mm == nil {
		t.Fatal("XMPMediaManagementSchema() = nil")
	}
	derivedFrom := mm.DerivedFromProperty()
	if derivedFrom == nil {
		t.Fatal("DerivedFromProperty() = nil")
	}
	equal(t, "InstanceID()", derivedFrom.InstanceID(),
		"uuid:6b838c4d-07e2-0611-2333-558805f93988")
	equal(t, "DocumentID()", derivedFrom.DocumentID(),
		"uuid:6b838c4d-07e2-0611-2333-558805f93988")
}

func TestBadRdfNameSpace(t *testing.T) {
	// has https in rdf namespace
	s := `<?xml version="1.0" encoding="UTF-8" standalone="no"?>
<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?><x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="XXX">
    <rdf:RDF xmlns:rdf="https://www.w3.org/1999/02/22-rdf-syntax-ns#">
    </rdf:RDF>
</x:xmpmeta><?xpacket end="w"?>`
	parseFails(t, s, "Expecting namespace 'http://www.w3.org/1999/02/22-rdf-syntax-ns#'"+
		" and found 'https://www.w3.org/1999/02/22-rdf-syntax-ns#'")
}
