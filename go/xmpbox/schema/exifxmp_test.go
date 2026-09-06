package schema_test

// Port of org.apache.xmpbox.schema.TestExifXmp.

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// xmpFixture is where the Java test resources are, which the port reads rather
// than copying.
const xmpFixture = "../../../xmpbox/src/test/resources/"

// openFixture opens one of the Java test resources.
func openFixture(t *testing.T, name string) io.ReadCloser {
	t.Helper()
	file, err := os.Open(xmpFixture + name)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	t.Cleanup(func() { file.Close() })
	return file
}

func TestExifNonStrict(t *testing.T) {
	is := openFixture(t, "validxmp/exif.xmp")
	builder := xmpxml.NewDomXmpParser()
	builder.SetStrictParsing(false)
	rxmp, err := builder.Parse(is)
	noError(t, "Parse", err)

	found := rxmp.Schema(schema.ExifNamespace)
	if found == nil {
		t.Fatal("Schema(exif) = nil, want the exif schema")
	}
	ss, isText := found.Base().Property(schema.ExifSpectralSensitivity).(*xmptype.TextType)
	if !isText {
		t.Fatalf("SpectralSensitivity = %T, want a text property",
			found.Base().Property(schema.ExifSpectralSensitivity))
	}
	value, isString := ss.Value().(string)
	if !isString {
		t.Fatalf("Value() = %T, want a string", ss.Value())
	}
	equal(t, "SpectralSensitivity", value, "spectral sens value")
}

func TestExifGenerate(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	tmapping := metadata.TypeMapping()
	exif, err := schema.NewExifSchema(metadata)
	noError(t, "NewExifSchema", err)
	metadata.AddSchema(exif)

	oecf := xmptype.NewOECFType(metadata)
	columns, err := tmapping.CreateInteger(oecf.Namespace(), oecf.Prefix(),
		xmptype.OECFColumns, 14)
	noError(t, "CreateInteger", err)
	oecf.AddProperty(columns)
	oecf.SetPropertyName(schema.ExifOECF)
	exif.AddProperty(oecf)

	var out bytes.Buffer
	noError(t, "Serialize", xmpxml.NewXmpSerializer().Serialize(metadata, &out, false))
}
