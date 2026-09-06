package xml_test

// Port of org.apache.xmpbox.parser.DeserializationTest.
//
// Java gives the parser tests a package of their own; the port keeps them with
// the parser they exercise, because Go has no split between a class's package
// and its test's.

import (
	"bytes"
	"crypto/sha256"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// TestMain fixes the zone every date in this package is read and written in.
//
// Port of DeserializationTest's @BeforeAll, which sets the default TimeZone to
// UTC because date values would otherwise differ depending on test location,
// and of the @Isolated and @ResourceLock(TIME_ZONE) that keep it from clashing
// with the rest of the suite. Go's default zone is time.Local, and a Go test
// binary runs one package's tests in one process, so setting it here covers the
// package the way @BeforeAll covers the class.
func TestMain(m *testing.M) {
	defaultTZ := time.Local
	time.Local = time.UTC
	code := m.Run()
	time.Local = defaultTZ
	os.Exit(code)
}

// checkTransform writes the metadata out, checks the digest of what was
// written, and reads it back to check the schema count.
//
// Port of DeserializationTest.checkTransform. Java normalizes CRLF to LF before
// hashing, because its transformer writes the platform's line separator; the
// port always writes LF, so the digests are over the same bytes.
func checkTransform(t *testing.T, metadata *xmpbox.XMPMetadata, expected string,
	expectedSchemaCount int) {
	t.Helper()
	var baos bytes.Buffer
	noError(t, "Serialize", xmpxml.NewXmpSerializer().Serialize(metadata, &baos, true))
	replaced := strings.ReplaceAll(baos.String(), "\r\n", "\n")
	digest := sha256.Sum256([]byte(replaced))
	result := new(big.Int).SetBytes(digest[:]).String()
	if result != expected {
		t.Errorf("digest = %s, want %s\noutput:\n%s", result, expected, replaced)
	}
	xmp, err := xmpxml.NewDomXmpParser().ParseBytes(baos.Bytes())
	noError(t, "parse", err)
	if got := len(xmp.AllSchemas()); got != expectedSchemaCount {
		t.Errorf("AllSchemas() held %d schemas, want %d", got, expectedSchemaCount)
	}
}

// parseFixture reads one of the Java test resources as a packet.
func parseFixture(t *testing.T, name string) *xmpbox.XMPMetadata {
	t.Helper()
	xmp, err := xmpxml.NewDomXmpParser().ParseBytes(readFixture(t, name))
	noError(t, "parse", err)
	return xmp
}

// fixtureErrorType reads one of the Java test resources and answers the kind of
// problem it is refused with.
func fixtureErrorType(t *testing.T, name string) xmpxml.ErrorType {
	t.Helper()
	_, err := xmpxml.NewDomXmpParser().ParseBytes(readFixture(t, name))
	if err == nil {
		t.Fatalf("parse of %s reported nothing, want a failure", name)
	}
	parsing, isParsing := err.(*xmpxml.XmpParsingError)
	if !isParsing {
		t.Fatalf("parse of %s = %T, want a parsing error", name, err)
	}
	return parsing.ErrorType
}

func TestStructuredRecursive(t *testing.T) {
	// not valid XMP according to the PDFLib free XMP validator
	metadata := parseFixture(t, "org/apache/xmpbox/parser/structured_recursive.xml")
	checkTransform(t, metadata,
		"62495942572014793625872774972947435765670563107818217447706375288846297812281",
		len(metadata.AllSchemas()))
}

func TestEmptyLi(t *testing.T) {
	metadata := parseFixture(t, "org/apache/xmpbox/parser/empty_list.xml")
	checkTransform(t, metadata,
		"95754993383010030299848397520773287413798669761891751126809013411187892693280",
		len(metadata.AllSchemas()))
}

func TestEmptyLi2(t *testing.T) {
	metadata := parseFixture(t, "validxmp/emptyli.xml")
	dc := metadata.DublinCoreSchema()
	if dc == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	dc.CreatorsProperty()
	checkTransform(t, metadata,
		"39450703080437563739186076111811684356424147071014681699119272065568305393521",
		len(metadata.AllSchemas()))
}

func TestGetTitle(t *testing.T) {
	metadata := parseFixture(t, "validxmp/emptyli.xml")
	dc := metadata.DublinCoreSchema()
	if dc == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	// Java passes null for the language, which the port spells as the empty
	// string.
	s, err := dc.TitleOfLanguage("")
	noError(t, "TitleOfLanguage", err)
	equal(t, "Title", s, "title value")
}

func TestAltBagSeq(t *testing.T) {
	metadata := parseFixture(t, "org/apache/xmpbox/parser/AltBagSeqTest.xml")
	checkTransform(t, metadata,
		"89123270336154452745819041017446278583816329940574853160909598044560152910018",
		len(metadata.AllSchemas()))
}

func TestIsartorStyleWithThumbs(t *testing.T) {
	metadata := parseFixture(t, "org/apache/xmpbox/parser/ThumbisartorStyle.xml")

	// <xmpMM:DocumentID>
	mm := metadata.XMPMediaManagementSchema()
	if mm == nil {
		t.Fatal("XMPMediaManagementSchema() = nil")
	}
	equal(t, "DocumentID()", mm.DocumentID(), "uuid:09C78666-2F91-3A9C-92AF-3691A6D594F7")

	// <xmp:CreateDate> <xmp:ModifyDate> <xmp:MetadataDate>
	basic := metadata.XMPBasicSchema()
	if basic == nil {
		t.Fatal("XMPBasicSchema() = nil")
	}
	want, err := xmpbox.ToCalendar("2008-01-18T16:59:54+01:00")
	noError(t, "ToCalendar", err)
	for _, dates := range []struct {
		what string
		read func() (time.Time, bool)
	}{
		{"CreateDate()", basic.CreateDate},
		{"ModifyDate()", basic.ModifyDate},
		{"MetadataDate()", basic.MetadataDate},
	} {
		got, held := dates.read()
		if !held || !got.Equal(want) {
			t.Errorf("%s = %v, %v, want %v", dates.what, got, held, want)
		}
	}

	// THUMBNAILS TEST
	thumbs, err := basic.ThumbnailsProperty()
	noError(t, "ThumbnailsProperty", err)
	if len(thumbs) != 2 {
		t.Fatalf("ThumbnailsProperty() held %d thumbnails, want 2", len(thumbs))
	}
	for i, thumb := range thumbs {
		height, held := thumb.Height()
		if !held || height != 162 {
			t.Errorf("thumbs[%d].Height() = %v, %v, want 162", i, height, held)
		}
		width, held := thumb.Width()
		if !held || width != 216 {
			t.Errorf("thumbs[%d].Width() = %v, %v, want 216", i, width, held)
		}
		equal(t, "Format()", thumb.Format(), "JPEG")
		equal(t, "Image()", thumb.Image(), "/9j/4AAQSkZJRgABAgEASABIAAD")
	}

	// Check the extension schema (also serves as example on how to retrieve)
	acmeMailSchema := metadata.Schema("http://www.acme.com/ns/email/1/")
	if acmeMailSchema == nil {
		t.Fatal("Schema(acme) = nil")
	}
	deliveryDate, isDate := acmeMailSchema.Base().Property("Delivery-Date").(*xmptype.DateType)
	if !isDate {
		t.Fatalf("Delivery-Date = %T, want a date",
			acmeMailSchema.Base().Property("Delivery-Date"))
	}
	equal(t, "Delivery-Date", deliveryDate.StringValue(), "2007-11-09T09:55:36+01:00")

	dst, isDefined := acmeMailSchema.Base().Property("From").(*xmptype.DefinedStructuredType)
	if !isDefined {
		t.Fatalf("From = %T, want a defined structured type",
			acmeMailSchema.Base().Property("From"))
	}
	equal(t, "name", propertyString(t, "name", dst.Property("name")),
		"[name=TextType:John Doe]")
	equal(t, "mailto", propertyString(t, "mailto", dst.Property("mailto")),
		"[mailto=TextType:john@acme.com]")

	checkTransform(t, metadata,
		"64755266855514150823517184659364700851455308334441170957883187622624192802093",
		len(metadata.AllSchemas()))
}

func TestWithNoXPacketStart(t *testing.T) {
	equal(t, "ErrorType", fixtureErrorType(t, "invalidxmp/noxpacket.xml"),
		xmpxml.XpacketBadStart)
}

func TestWithNoXPacketEnd(t *testing.T) {
	equal(t, "ErrorType", fixtureErrorType(t, "invalidxmp/noxpacketend.xml"),
		xmpxml.XpacketBadEnd)
}

func TestWithNoRDFElement(t *testing.T) {
	equal(t, "ErrorType", fixtureErrorType(t, "invalidxmp/noroot.xml"), xmpxml.Format)
}

func TestWithTwoRDFElement(t *testing.T) {
	equal(t, "ErrorType", fixtureErrorType(t, "invalidxmp/tworoot.xml"), xmpxml.Format)
}

func TestWithInvalidRDFElementPrefix(t *testing.T) {
	equal(t, "ErrorType", fixtureErrorType(t, "invalidxmp/invalidroot2.xml"), xmpxml.Format)
}

func TestWithRDFRootAsText(t *testing.T) {
	equal(t, "ErrorType", fixtureErrorType(t, "invalidxmp/invalidroot.xml"), xmpxml.Format)
}

func TestUndefinedSchema(t *testing.T) {
	equal(t, "ErrorType", fixtureErrorType(t, "invalidxmp/undefinedschema.xml"),
		xmpxml.NoSchema)
}

func TestUndefinedPropertyWithDefinedSchema(t *testing.T) {
	equal(t, "ErrorType",
		fixtureErrorType(t, "invalidxmp/undefinedpropertyindefinedschema.xml"),
		xmpxml.NoType)
}

func TestUndefinedStructuredWithDefinedSchema(t *testing.T) {
	equal(t, "ErrorType",
		fixtureErrorType(t, "invalidxmp/undefinedstructuredindefinedschema.xml"),
		xmpxml.NoValueType)
}

func TestRdfAboutFound(t *testing.T) {
	metadata := parseFixture(t, "validxmp/emptyli.xml")
	for _, xmpSchema := range metadata.AllSchemas() {
		if xmpSchema.Base().AboutAttribute() == nil {
			t.Errorf("%s has no rdf:about attribute", xmpSchema.Namespace())
		}
	}
}

func TestWithAttributesAsProperties(t *testing.T) {
	metadata := parseFixture(t, "validxmp/attr_as_props.xml")

	pdf := metadata.AdobePDFSchema()
	if pdf == nil {
		t.Fatal("AdobePDFSchema() = nil")
	}
	equal(t, "Producer()", pdf.Producer(), "GPL Ghostscript 8.64")

	dc := metadata.DublinCoreSchema()
	if dc == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	equal(t, "Format()", dc.Format(), "application/pdf")

	basic := metadata.XMPBasicSchema()
	if basic == nil {
		t.Fatal("XMPBasicSchema() = nil")
	}
	if _, held := basic.CreateDate(); !held {
		t.Error("CreateDate() held nothing, want a date")
	}

	pdfaid := metadata.PDFAIdentificationSchema()
	if pdfaid == nil {
		t.Fatal("PDFAIdentificationSchema() = nil")
	}
	equal(t, "Conformance()", pdfaid.Conformance(), "B")
	part, held := pdfaid.Part()
	if !held || part != 1 {
		t.Errorf("Part() = %v, %v, want 1", part, held)
	}

	mm := metadata.XMPMediaManagementSchema()
	if mm == nil {
		t.Fatal("XMPMediaManagementSchema() = nil")
	}
	equal(t, "DocumentID()", mm.DocumentID(), "e7127190-445c-11ea-0000-b3bc74086807")

	checkTransform(t, metadata,
		"27499224985683016678197540524065114038595582230834506941950503218519476041225",
		len(metadata.AllSchemas()))
}

// TestSpaceTextValues checks values with spaces at the start or the end, which
// must not be trimmed.
func TestSpaceTextValues(t *testing.T) {
	metadata := parseFixture(t, "validxmp/only_space_fields.xmp")

	// check producer
	pdf := metadata.AdobePDFSchema()
	if pdf == nil {
		t.Fatal("AdobePDFSchema() = nil")
	}
	equal(t, "Producer()", pdf.Producer(), " ")

	// check creator tool
	basic := metadata.XMPBasicSchema()
	if basic == nil {
		t.Fatal("XMPBasicSchema() = nil")
	}
	equal(t, "CreatorTool()", basic.CreatorTool(), "Canon ")

	checkTransform(t, metadata,
		"9220923061800113567693538810355030344095407871190202111473587642358933618073",
		len(metadata.AllSchemas()))
}

func TestMetadataParsing(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	dc, err := metadata.CreateAndAddDublinCoreSchema()
	noError(t, "CreateAndAddDublinCoreSchema", err)
	noError(t, "SetCoverage", dc.SetCoverage("coverage"))
	noError(t, "AddContributor", dc.AddContributor("contributor1"))
	noError(t, "AddContributor", dc.AddContributor("contributor2"))
	noError(t, "AddDescription", dc.AddDescription("x-default", "Description"))

	pdf, err := metadata.CreateAndAddAdobePDFSchema()
	noError(t, "CreateAndAddAdobePDFSchema", err)
	noError(t, "SetProducer", pdf.SetProducer("Producer"))
	noError(t, "SetPDFVersion", pdf.SetPDFVersion("1.4"))

	checkTransform(t, metadata,
		"24727341753942351260821151680330022244742411666459385225917195999704816908515",
		len(metadata.AllSchemas()))
}

// TestEmptyDate serializes an empty date property, which brought a
// NullPointerException before PDFBOX-6029.
func TestEmptyDate(t *testing.T) {
	xmpmeta := `<?xpacket begin="` + bom + `" id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta x:xmptk="Adobe XMP Core 4.2.1-c041 52.342996, 2008/05/07-20:48:00" xmlns:x="adobe:ns:meta/">
  <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
   <rdf:Description rdf:about="" xmlns:xmp="http://ns.adobe.com/xap/1.0/">
    <xmp:CreateDate></xmp:CreateDate>
   </rdf:Description>
  </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`
	metadata := parse(t, xmpmeta)
	checkTransform(t, metadata,
		"19030153876683461724958694183980892665426846590791273142114566290124997390122",
		len(metadata.AllSchemas()))
}
