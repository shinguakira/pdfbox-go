package schema_test

// Port of org.apache.xmpbox.schema.AdobePDFTest and AdobePDFErrorsTest, whose
// two error cases are the same two PDFAIdentificationOthersTest declares.

import (
	"bytes"
	"errors"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// freshAdobePDF is the @BeforeEach: a new packet with the schema added.
func freshAdobePDF(t *testing.T) *schema.AdobePDFSchema {
	t.Helper()
	pdf, err := xmpbox.CreateXMPMetadata().CreateAndAddAdobePDFSchema()
	if err != nil {
		t.Fatalf("CreateAndAddAdobePDFSchema: %v", err)
	}
	return pdf
}

func TestAdobePDF(t *testing.T) {
	runFieldCases(t, freshAdobePDF, []fieldCase[schema.AdobePDFSchema]{
		{
			name:   "Keywords",
			absent: func(s *schema.AdobePDFSchema) bool { return s.KeywordsProperty() == nil },
			exercise: []func(*testing.T, *schema.AdobePDFSchema){
				func(t *testing.T, s *schema.AdobePDFSchema) {
					noError(t, "SetKeywords", s.SetKeywords("kw1 kw2 kw3"))
					equal(t, "Keywords()", s.Keywords(), "kw1 kw2 kw3")
				},
				func(t *testing.T, s *schema.AdobePDFSchema) {
					text, err := s.CreateTextType(schema.PDFKeywords, "other keywords")
					noError(t, "CreateTextType", err)
					s.SetKeywordsProperty(text)
					equal(t, "Keywords()", s.Keywords(), "other keywords")
				},
			},
		},
		{
			name:   "PDFVersion",
			absent: func(s *schema.AdobePDFSchema) bool { return s.PDFVersionProperty() == nil },
			exercise: []func(*testing.T, *schema.AdobePDFSchema){
				func(t *testing.T, s *schema.AdobePDFSchema) {
					noError(t, "SetPDFVersion", s.SetPDFVersion("1.4"))
					equal(t, "PDFVersion()", s.PDFVersion(), "1.4")
				},
				func(t *testing.T, s *schema.AdobePDFSchema) {
					text, err := s.CreateTextType(schema.PDFPDFVersion, "1.7")
					noError(t, "CreateTextType", err)
					s.SetPDFVersionProperty(text)
					equal(t, "PDFVersion()", s.PDFVersion(), "1.7")
				},
			},
		},
		{
			name:   "Producer",
			absent: func(s *schema.AdobePDFSchema) bool { return s.ProducerProperty() == nil },
			exercise: []func(*testing.T, *schema.AdobePDFSchema){
				func(t *testing.T, s *schema.AdobePDFSchema) {
					noError(t, "SetProducer", s.SetProducer("testcase"))
					equal(t, "Producer()", s.Producer(), "testcase")
				},
				func(t *testing.T, s *schema.AdobePDFSchema) {
					text, err := s.CreateTextType(schema.PDFProducer, "another producer")
					noError(t, "CreateTextType", err)
					s.SetProducerProperty(text)
					equal(t, "Producer()", s.Producer(), "another producer")
				},
			},
		},
	})
}

func TestAdobePDFIdentification(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	schem, err := metadata.CreateAndAddAdobePDFSchema()
	noError(t, "CreateAndAddAdobePDFSchema", err)

	const keywords = "keywords ihih"
	const pdfVersion = "1.4"
	const producer = "producer"

	noError(t, "SetKeywords", schem.SetKeywords(keywords))
	noError(t, "SetPDFVersion", schem.SetPDFVersion(pdfVersion))

	// Check get null if property not defined
	equal(t, "Producer()", schem.Producer(), "")

	noError(t, "SetProducer", schem.SetProducer(producer))

	assertAdobePDF(t, schem, keywords, pdfVersion, producer)

	// check retrieve this schema in metadata
	if metadata.AdobePDFSchema() != schem {
		t.Errorf("AdobePDFSchema() is not the schema that was added")
	}

	var bos bytes.Buffer
	noError(t, "Serialize", xmpxml.NewXmpSerializer().Serialize(metadata, &bos, true))
	rxmp, err := xmpxml.NewDomXmpParser().ParseBytes(bos.Bytes())
	noError(t, "ParseBytes", err)
	assertAdobePDF(t, rxmp.AdobePDFSchema(), keywords, pdfVersion, producer)
}

// assertAdobePDF is the block of assertions the test makes before and after the
// round trip.
func assertAdobePDF(t *testing.T, schem *schema.AdobePDFSchema,
	keywords, pdfVersion, producer string) {
	t.Helper()
	if schem == nil {
		t.Fatal("no Adobe PDF schema")
	}
	equal(t, "KeywordsProperty().Prefix()", schem.KeywordsProperty().Prefix(), "pdf")
	equal(t, "KeywordsProperty().PropertyName()",
		schem.KeywordsProperty().PropertyName(), "Keywords")
	equal(t, "Keywords()", schem.Keywords(), keywords)

	equal(t, "PDFVersionProperty().Prefix()", schem.PDFVersionProperty().Prefix(), "pdf")
	equal(t, "PDFVersionProperty().PropertyName()",
		schem.PDFVersionProperty().PropertyName(), "PDFVersion")
	equal(t, "PDFVersion()", schem.PDFVersion(), pdfVersion)

	equal(t, "ProducerProperty().Prefix()", schem.ProducerProperty().Prefix(), "pdf")
	equal(t, "ProducerProperty().PropertyName()",
		schem.ProducerProperty().PropertyName(), "Producer")
	equal(t, "Producer()", schem.Producer(), producer)
}

func TestBadPDFAConformanceID(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	pdfaid, err := metadata.CreateAndAddPDFAIdentificationSchema()
	noError(t, "CreateAndAddPDFAIdentificationSchema", err)
	if err := pdfaid.SetConformance("kiohiohiohiohio"); !errors.Is(err, xmptype.ErrBadFieldValue) {
		t.Errorf("SetConformance = %v, want a bad field value", err)
	}
}

func TestBadVersionIDValueType(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	pdfaid, err := metadata.CreateAndAddPDFAIdentificationSchema()
	noError(t, "CreateAndAddPDFAIdentificationSchema", err)
	noError(t, "SetPartValueWithString", pdfaid.SetPartValueWithString("1"))
	// Java raises IllegalArgumentException, which the port reports as an error
	// out of the setter.
	if err := pdfaid.SetPartValueWithString("ojoj"); err == nil {
		t.Error("SetPartValueWithString(\"ojoj\") reported nothing, want an error")
	}
}
