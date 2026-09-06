package xmpbox_test

// Port of org.apache.xmpbox.XMPMetaDataTest, DoubleSameTypeSchemaTest,
// TestXMPWithDefinedSchemas, TestXMPWithUndefinedSchemas and
// TestValidatePermitedMetadata.
//
// An external test: the metadata it builds needs the schema and xml packages,
// and the xml package imports this one.

import (
	"bufio"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// xmpFixture is where the Java test resources are, which the port reads rather
// than copying.
const xmpFixture = "../../xmpbox/src/test/resources/"

// noError fails the test where a step reported a problem.
func noError(t *testing.T, what string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}

// equal reports a value that is not what the Java test asserts.
func equal[T comparable](t *testing.T, what string, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %v, want %v", what, got, want)
	}
}

func TestAddingSchem(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	const tmpNsURI = "http://www.test.org/schem/"
	tmp, err := schema.NewXMPSchemaOfNamespace(metadata, tmpNsURI, "test")
	noError(t, "NewXMPSchemaOfNamespace", err)

	for _, value := range []string{"Value1", "Value2", "Value3"} {
		noError(t, "AddQualifiedBagValue", tmp.AddQualifiedBagValue("BagContainer", value))
		noError(t, "AddUnqualifiedSequenceValue",
			tmp.AddUnqualifiedSequenceValue("SeqContainer", value))
	}
	text, err := metadata.TypeMapping().CreateText("", "test", "simpleProperty", "YEP")
	noError(t, "CreateText", err)
	tmp.AddProperty(text)

	tmp2, err := schema.NewXMPSchemaFull(metadata, "http://www.space.org/schem/",
		"space", "space")
	noError(t, "NewXMPSchemaFull", err)
	for _, value := range []string{"ValueSpace1", "ValueSpace2", "ValueSpace3"} {
		noError(t, "AddUnqualifiedSequenceValue",
			tmp2.AddUnqualifiedSequenceValue("SeqSpContainer", value))
	}

	metadata.AddSchema(tmp)
	metadata.AddSchema(tmp2)

	// Check schema getting
	if metadata.Schema(tmpNsURI) != schema.Schema(tmp) {
		t.Errorf("Schema(%q) = %v, want the schema that was added", tmpNsURI,
			metadata.Schema(tmpNsURI))
	}
	if found := metadata.Schema("THIS URI NOT EXISTS !"); found != nil {
		t.Errorf("Schema of an unknown namespace = %v, want nothing", found)
	}
	vals := metadata.AllSchemas()
	if !holdsSchema(vals, tmp) || !holdsSchema(vals, tmp2) {
		t.Errorf("AllSchemas() = %v, want both schemas", vals)
	}
}

// holdsSchema reports whether the list holds the schema, which is Java's
// List.contains with the identity equals XMPSchema inherits.
func holdsSchema(all []schema.Schema, wanted schema.Schema) bool {
	for _, held := range all {
		if held == wanted {
			return true
		}
	}
	return false
}

func TestInitMetaDataWithInfo(t *testing.T) {
	const xpacketBegin, xpacketID = "TESTBEG", "TESTID"
	xpacketBytes := "TESTBYTES"
	const xpacketEncoding = "TESTENCOD"
	metadata := xmpbox.CreateXMPMetadataOfXpacket(xpacketBegin, xpacketID,
		&xpacketBytes, xpacketEncoding)
	equal(t, "XpacketBegin()", metadata.XpacketBegin(), xpacketBegin)
	equal(t, "XpacketID()", metadata.XpacketID(), xpacketID)
	if got := metadata.XpacketBytes(); got == nil || *got != xpacketBytes {
		t.Errorf("XpacketBytes() = %v, want %q", got, xpacketBytes)
	}
	equal(t, "XpacketEncoding()", metadata.XpacketEncoding(), xpacketEncoding)
}

// TestPDFBOX3257 checks that setting CreateDate twice does not insert two
// elements, and that fixing that did not interfere with the handling of lists.
func TestPDFBOX3257(t *testing.T) {
	// taken from file test-landscape2.pdf
	xmpmeta := `<?xpacket id="W5M0MpCehiHzreSzNTczkc9d"?>
<x:xmpmeta xmlns:x="adobe:ns:meta/" x:xmptk="Adobe XMP Core 4.0-c316 44.253921, Sun Oct 01 2006 17:14:39">
   <rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
      <rdf:Description rdf:about=""
            xmlns:xap="http://ns.adobe.com/xap/1.0/">
         <xap:CreatorTool>Acrobat PDFMaker 8.1 for Word</xap:CreatorTool>
         <xap:ModifyDate>2008-11-12T15:29:43+01:00</xap:ModifyDate>
         <xap:CreateDate>2008-11-12T15:29:40+01:00</xap:CreateDate>
         <xap:MetadataDate>2008-11-12T15:29:43+01:00</xap:MetadataDate>
      </rdf:Description>
      <rdf:Description rdf:about=""
            xmlns:pdf="http://ns.adobe.com/pdf/1.3/">
         <pdf:Producer>Acrobat Distiller 8.1.0 (Windows)</pdf:Producer>
      </rdf:Description>
      <rdf:Description rdf:about=""
            xmlns:dc="http://purl.org/dc/elements/1.1/">
         <dc:format>application/pdf</dc:format>
         <dc:creator>
            <rdf:Seq>
               <rdf:li>R002325</rdf:li>
            </rdf:Seq>
         </dc:creator>
         <dc:subject>
            <rdf:Bag>
               <rdf:li>one</rdf:li>
               <rdf:li>two</rdf:li>
               <rdf:li>three</rdf:li>
               <rdf:li>four</rdf:li>
            </rdf:Bag>
         </dc:subject>
         <dc:title>
            <rdf:Alt>
               <rdf:li xml:lang="x-default"> </rdf:li>
            </rdf:Alt>
         </dc:title>
      </rdf:Description>
      <rdf:Description rdf:about=""
            xmlns:xapMM="http://ns.adobe.com/xap/1.0/mm/">
         <xapMM:DocumentID>uuid:31ae92cf-9a27-45e0-9371-0d2741e25919</xapMM:DocumentID>
         <xapMM:InstanceID>uuid:2c7eb5da-9210-4666-8cef-e02ef6631c5e</xapMM:InstanceID>
      </rdf:Description>
   </rdf:RDF>
</x:xmpmeta>
<?xpacket end="w"?>`
	parser := xmpxml.NewDomXmpParser()
	parser.SetStrictParsing(false)
	xmp, err := parser.ParseBytes([]byte(xmpmeta))
	noError(t, "parse", err)

	basicSchema := xmp.XMPBasicSchema()
	if basicSchema == nil {
		t.Fatal("XMPBasicSchema() = nil")
	}
	createDate1, held := basicSchema.CreateDate()
	if !held {
		t.Fatal("CreateDate() held nothing before it was set")
	}
	noError(t, "SetCreateDate", basicSchema.SetCreateDate(time.Now()))
	createDate2, held := basicSchema.CreateDate()
	if !held {
		t.Fatal("CreateDate() held nothing after it was set")
	}
	if createDate1.Equal(createDate2) {
		t.Errorf("CreateDate has not been set: still %v", createDate2)
	}

	// check that bugfix does not interfere with lists of properties with same
	// name
	dublinCoreSchema := xmp.DublinCoreSchema()
	if dublinCoreSchema == nil {
		t.Fatal("DublinCoreSchema() = nil")
	}
	if subjects := dublinCoreSchema.Subjects(); len(subjects) != 4 {
		t.Errorf("Subjects() = %v, want four", subjects)
	}
}

func TestDoubleDublinCore(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	dc1, err := metadata.CreateAndAddDublinCoreSchema()
	noError(t, "CreateAndAddDublinCoreSchema", err)
	const ownPrefix = "test"
	dc2, err := schema.NewDublinCoreSchemaPrefixed(metadata, ownPrefix)
	noError(t, "NewDublinCoreSchemaPrefixed", err)
	metadata.AddSchema(dc2)

	creators := []string{"creator1", "creator2"}
	const format = "application/pdf"
	noError(t, "SetFormat", dc1.SetFormat(format))
	noError(t, "AddCreator", dc1.AddCreator(creators[0]))
	noError(t, "AddCreator", dc1.AddCreator(creators[1]))

	const coverage = "Coverage"
	noError(t, "SetCoverage", dc2.SetCoverage(coverage))
	noError(t, "AddCreator", dc2.AddCreator(creators[0]))
	noError(t, "AddCreator", dc2.AddCreator(creators[1]))

	// We can't use metadata.DublinCoreSchema() due to specification of XMPBox
	// (see the doc comment of XMPMetadata)
	byPreferred, isDublinCore := metadata.SchemaOfPrefix("dc",
		schema.DublinCoreNamespace).(*schema.DublinCoreSchema)
	if !isDublinCore {
		t.Fatalf("SchemaOfPrefix(\"dc\") = %T, want a Dublin Core schema",
			metadata.SchemaOfPrefix("dc", schema.DublinCoreNamespace))
	}
	equal(t, "Format()", byPreferred.Format(), format)

	byOwn, isDublinCore := metadata.SchemaOfPrefix(ownPrefix,
		schema.DublinCoreNamespace).(*schema.DublinCoreSchema)
	if !isDublinCore {
		t.Fatalf("SchemaOfPrefix(%q) = %T, want a Dublin Core schema", ownPrefix,
			metadata.SchemaOfPrefix(ownPrefix, schema.DublinCoreNamespace))
	}
	equal(t, "Coverage()", byOwn.Coverage(), coverage)

	for _, xmpSchema := range metadata.AllSchemas() {
		dc, isDublinCore := xmpSchema.(*schema.DublinCoreSchema)
		if !isDublinCore {
			t.Fatalf("schema = %T, want a Dublin Core schema", xmpSchema)
		}
		held := dc.Creators()
		for _, creator := range creators {
			found := false
			for _, name := range held {
				if name == creator {
					found = true
				}
			}
			if !found {
				t.Errorf("Creators() = %v, which does not hold %q", held, creator)
			}
		}
	}
}

func TestXMPWithDefinedSchemas(t *testing.T) {
	for _, path := range []string{
		"validxmp/override_ns.rdf",
		"validxmp/ghost2.xmp",
		"validxmp/history2.rdf",
		"validxmp/Notepad++_A1b.xmp",
		"validxmp/metadata.rdf",
		"validxmp/PDFBOX-6099.xmp",
	} {
		t.Run(path, func(t *testing.T) {
			content, err := os.ReadFile(xmpFixture + path)
			noError(t, "read", err)
			rxmp, err := xmpxml.NewDomXmpParser().ParseBytes(content)
			noError(t, "parse", err)
			// ensure basic parsing was OK
			if len(rxmp.AllSchemas()) == 0 {
				t.Error("AllSchemas() is empty")
			}
		})
	}
}

func TestXMPWithUndefinedSchemas(t *testing.T) {
	for _, c := range []struct {
		path, namespace, propertyName, propertyValue string
	}{
		{"undefinedxmp/prism.xmp", "http://prismstandard.org/namespaces/basic/2.0/",
			"aggregationType", "journal"},
	} {
		t.Run(c.path, func(t *testing.T) {
			content, err := os.ReadFile(xmpFixture + c.path)
			noError(t, "read", err)
			builder := xmpxml.NewDomXmpParser()
			builder.SetStrictParsing(false)
			rxmp, err := builder.ParseBytes(content)
			noError(t, "parse", err)

			// ensure basic parsing was OK
			if len(rxmp.AllSchemas()) == 0 {
				t.Fatal("There should be a least one schema")
			}
			found := rxmp.Schema(c.namespace)
			if found == nil {
				t.Fatalf("The schema for {%s} should be available", c.namespace)
			}
			property := found.Base().Property(c.propertyName)
			if property == nil {
				t.Fatalf("The schema for {%s} should have a property {%s}",
					c.namespace, c.propertyName)
			}
			equal(t, "PropertyName()", property.PropertyName(), c.propertyName)
			value, err := found.Base().UnqualifiedTextPropertyValue(c.propertyName)
			noError(t, "UnqualifiedTextPropertyValue", err)
			equal(t, "value", value, c.propertyValue)
		})
	}
}

// TestValidatePermitedMetadata checks that every namespace, prefix and field
// name the specification permits is one this module declares.
//
// Java finds the field by walking the schema class for @PropertyType
// annotations; the port asks the schema factory's property description, which
// is what those annotations were read into.
func TestValidatePermitedMetadata(t *testing.T) {
	file, err := os.Open(xmpFixture + "permited_metadata.txt")
	noError(t, "open", err)
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "http://") {
			// else skip line
			continue
		}
		// this is a line to handle
		pos := strings.LastIndex(line, ":")
		spos := strings.LastIndex(line[:pos], "/")
		namespace := line[:spos+1]
		preferred := line[spos+1 : pos]
		fieldname := line[pos+1:]

		t.Run(namespace+preferred+":"+fieldname, func(t *testing.T) {
			// ensure schema exists
			xmpmd := xmpbox.CreateXMPMetadata()
			mapping := xmptype.NewTypeMapping(xmpmd)
			factory, isFactory := mapping.SchemaFactory(namespace).(*schema.XMPSchemaFactory)
			if !isFactory {
				t.Fatalf("Schema not existing: %s", namespace)
			}
			// ensure preferred is as expected
			created, err := factory.CreateXMPSchema(xmpmd, "aa")
			noError(t, "CreateXMPSchema", err)
			equal(t, "PreferedPrefix()", created.PreferedPrefix(), preferred)
			// ensure field is defined
			if _, found := factory.PropertyType(fieldname); !found {
				t.Errorf("Did not find field definition for '%s' in %s (%s)",
					fieldname, created.TypeName(), namespace)
			}
		})
	}
	noError(t, "scan", scanner.Err())
}
