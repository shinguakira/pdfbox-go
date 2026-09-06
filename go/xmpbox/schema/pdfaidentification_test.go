package schema_test

// Port of org.apache.xmpbox.schema.PDFAIdentificationTest and
// PDFAIdentificationOthersTest, whose two error cases are the two
// AdobePDFErrorsTest declares and which adobepdf_test.go holds.

import (
	"bytes"
	"strconv"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	xmpxml "github.com/shinguakira/pdfbox-go/go/xmpbox/xml"
)

// freshPDFAIdentification is the @BeforeEach: a new packet with the schema
// added.
func freshPDFAIdentification(t *testing.T) *schema.PDFAIdentificationSchema {
	t.Helper()
	pdfaid, err := xmpbox.CreateXMPMetadata().CreateAndAddPDFAIdentificationSchema()
	if err != nil {
		t.Fatalf("CreateAndAddPDFAIdentificationSchema: %v", err)
	}
	return pdfaid
}

func TestPDFAIdentificationFields(t *testing.T) {
	runFieldCases(t, freshPDFAIdentification,
		[]fieldCase[schema.PDFAIdentificationSchema]{
			{
				name: "part",
				absent: func(s *schema.PDFAIdentificationSchema) bool {
					return s.PartProperty() == nil
				},
				exercise: []func(*testing.T, *schema.PDFAIdentificationSchema){
					func(t *testing.T, s *schema.PDFAIdentificationSchema) {
						noError(t, "SetPart", s.SetPart(1))
						part, held := s.Part()
						if !held || part != 1 {
							t.Errorf("Part() = %v, %v, want 1", part, held)
						}
					},
					func(t *testing.T, s *schema.PDFAIdentificationSchema) {
						noError(t, "SetPartValueWithString", s.SetPartValueWithString("2"))
						part, held := s.Part()
						if !held || part != 2 {
							t.Errorf("Part() = %v, %v, want 2", part, held)
						}
					},
				},
			},
			{
				name: "amd",
				absent: func(s *schema.PDFAIdentificationSchema) bool {
					return s.AmdProperty() == nil
				},
				exercise: []func(*testing.T, *schema.PDFAIdentificationSchema){
					func(t *testing.T, s *schema.PDFAIdentificationSchema) {
						noError(t, "SetAmd", s.SetAmd("2005"))
						equal(t, "Amendment()", s.Amendment(), "2005")
						equal(t, "Amd()", s.Amd(), "2005")
					},
				},
			},
			{
				name: "conformance",
				absent: func(s *schema.PDFAIdentificationSchema) bool {
					return s.ConformanceProperty() == nil
				},
				exercise: []func(*testing.T, *schema.PDFAIdentificationSchema){
					func(t *testing.T, s *schema.PDFAIdentificationSchema) {
						noError(t, "SetConformance", s.SetConformance("B"))
						equal(t, "Conformance()", s.Conformance(), "B")
					},
				},
			},
			{
				name: "rev",
				absent: func(s *schema.PDFAIdentificationSchema) bool {
					return s.RevProperty() == nil
				},
				exercise: []func(*testing.T, *schema.PDFAIdentificationSchema){
					func(t *testing.T, s *schema.PDFAIdentificationSchema) {
						noError(t, "SetRev", s.SetRev(2020))
						rev, held := s.Rev()
						if !held || rev != 2020 {
							t.Errorf("Rev() = %v, %v, want 2020", rev, held)
						}
					},
					func(t *testing.T, s *schema.PDFAIdentificationSchema) {
						noError(t, "SetRevValueWithString", s.SetRevValueWithString("2021"))
						rev, held := s.Rev()
						if !held || rev != 2021 {
							t.Errorf("Rev() = %v, %v, want 2021", rev, held)
						}
					},
				},
			},
		})
}

func TestPDFAIdentification(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	pdfaid, err := metadata.CreateAndAddPDFAIdentificationSchema()
	noError(t, "CreateAndAddPDFAIdentificationSchema", err)

	const versionID = 1
	const amdID = "2005"
	const conformance = "B"

	noError(t, "SetPartValueWithInt", pdfaid.SetPartValueWithInt(versionID))
	noError(t, "SetAmd", pdfaid.SetAmd(amdID))
	noError(t, "SetConformance", pdfaid.SetConformance(conformance))

	assertPDFAIdentification(t, pdfaid, versionID, amdID, conformance)

	// check retrieve this schema in metadata
	if metadata.PDFAIdentificationSchema() != pdfaid {
		t.Error("PDFAIdentificationSchema() is not the schema that was added")
	}

	var bos bytes.Buffer
	noError(t, "Serialize", xmpxml.NewXmpSerializer().Serialize(metadata, &bos, true))
	rxmp, err := xmpxml.NewDomXmpParser().ParseBytes(bos.Bytes())
	noError(t, "ParseBytes", err)
	assertPDFAIdentification(t, rxmp.PDFAIdentificationSchema(), versionID, amdID, conformance)
}

// assertPDFAIdentification is the block of assertions the test makes before and
// after the round trip.
func assertPDFAIdentification(t *testing.T, pdfaid *schema.PDFAIdentificationSchema,
	versionID int, amdID, conformance string) {
	t.Helper()
	if pdfaid == nil {
		t.Fatal("no PDF/A identification schema")
	}
	part, held := pdfaid.Part()
	if !held || part != versionID {
		t.Errorf("Part() = %v, %v, want %d", part, held, versionID)
	}
	equal(t, "Amendment()", pdfaid.Amendment(), amdID)
	equal(t, "Conformance()", pdfaid.Conformance(), conformance)
	equal(t, "PartProperty().StringValue()", pdfaid.PartProperty().StringValue(),
		strconv.Itoa(versionID))
	equal(t, "AmdProperty().StringValue()", pdfaid.AmdProperty().StringValue(), amdID)
	equal(t, "ConformanceProperty().StringValue()",
		pdfaid.ConformanceProperty().StringValue(), conformance)
}
