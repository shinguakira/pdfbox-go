package schema_test

// Port of org.apache.xmpbox.schema.PhotoshopSchemaTest.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
)

// freshPhotoshop is the @BeforeEach: a new packet with the schema added.
func freshPhotoshop(t *testing.T) *schema.PhotoshopSchema {
	t.Helper()
	photoshop, err := xmpbox.CreateXMPMetadata().CreateAndAddPhotoshopSchema()
	if err != nil {
		t.Fatalf("CreateAndAddPhotoshopSchema: %v", err)
	}
	return photoshop
}

// photoshopText is one of the fifteen fields the schema holds as plain text,
// which the test exercises the same way.
func photoshopText(name string, get func(*schema.PhotoshopSchema) string,
	set func(*schema.PhotoshopSchema, string) error,
	property func(*schema.PhotoshopSchema) bool) fieldCase[schema.PhotoshopSchema] {
	value := "value of " + name
	return fieldCase[schema.PhotoshopSchema]{
		name:   name,
		absent: property,
		exercise: []func(*testing.T, *schema.PhotoshopSchema){
			func(t *testing.T, s *schema.PhotoshopSchema) {
				noError(t, "Set"+name, set(s, value))
				equal(t, name+"()", get(s), value)
			},
		},
	}
}

func TestPhotoshop(t *testing.T) {
	runFieldCases(t, freshPhotoshop, []fieldCase[schema.PhotoshopSchema]{
		{
			name:   "AncestorID",
			absent: func(s *schema.PhotoshopSchema) bool { return s.AncestorIDProperty() == nil },
			exercise: []func(*testing.T, *schema.PhotoshopSchema){
				func(t *testing.T, s *schema.PhotoshopSchema) {
					noError(t, "SetAncestorID", s.SetAncestorID("uuid:ancestor"))
					equal(t, "AncestorID()", s.AncestorID(), "uuid:ancestor")
				},
			},
		},
		photoshopText("AuthorsPosition",
			(*schema.PhotoshopSchema).AuthorsPosition,
			(*schema.PhotoshopSchema).SetAuthorsPosition,
			func(s *schema.PhotoshopSchema) bool { return s.AuthorsPositionProperty() == nil }),
		photoshopText("CaptionWriter",
			(*schema.PhotoshopSchema).CaptionWriter,
			(*schema.PhotoshopSchema).SetCaptionWriter,
			func(s *schema.PhotoshopSchema) bool { return s.CaptionWriterProperty() == nil }),
		photoshopText("Category",
			(*schema.PhotoshopSchema).Category,
			(*schema.PhotoshopSchema).SetCategory,
			func(s *schema.PhotoshopSchema) bool { return s.CategoryProperty() == nil }),
		photoshopText("City",
			(*schema.PhotoshopSchema).City,
			(*schema.PhotoshopSchema).SetCity,
			func(s *schema.PhotoshopSchema) bool { return s.CityProperty() == nil }),
		{
			name:   "ColorMode",
			absent: func(s *schema.PhotoshopSchema) bool { return s.ColorModeProperty() == nil },
			exercise: []func(*testing.T, *schema.PhotoshopSchema){
				func(t *testing.T, s *schema.PhotoshopSchema) {
					// The only setter Java declares takes the integer's string
					// form.
					noError(t, "SetColorMode", s.SetColorMode("3"))
					mode, held := s.ColorMode()
					if !held || mode != 3 {
						t.Errorf("ColorMode() = %v, %v, want 3", mode, held)
					}
				},
			},
		},
		photoshopText("Country",
			(*schema.PhotoshopSchema).Country,
			(*schema.PhotoshopSchema).SetCountry,
			func(s *schema.PhotoshopSchema) bool { return s.CountryProperty() == nil }),
		photoshopText("Credit",
			(*schema.PhotoshopSchema).Credit,
			(*schema.PhotoshopSchema).SetCredit,
			func(s *schema.PhotoshopSchema) bool { return s.CreditProperty() == nil }),
		{
			name:   "DateCreated",
			absent: func(s *schema.PhotoshopSchema) bool { return s.DateCreatedProperty() == nil },
			exercise: []func(*testing.T, *schema.PhotoshopSchema){
				func(t *testing.T, s *schema.PhotoshopSchema) {
					// Java's getter answers the written string, not a Calendar.
					noError(t, "SetDateCreated", s.SetDateCreated("2005-01-02T12:00:00Z"))
					// The value goes through a Calendar and comes back written
					// the way DateConverter.toISO8601 writes one.
					equal(t, "DateCreated()", s.DateCreated(), "2005-01-02T12:00:00+00:00")
				},
			},
		},
		photoshopText("Headline",
			(*schema.PhotoshopSchema).Headline,
			(*schema.PhotoshopSchema).SetHeadline,
			func(s *schema.PhotoshopSchema) bool { return s.HeadlineProperty() == nil }),
		photoshopText("History",
			(*schema.PhotoshopSchema).History,
			(*schema.PhotoshopSchema).SetHistory,
			func(s *schema.PhotoshopSchema) bool { return s.HistoryProperty() == nil }),
		photoshopText("ICCProfile",
			(*schema.PhotoshopSchema).ICCProfile,
			(*schema.PhotoshopSchema).SetICCProfile,
			func(s *schema.PhotoshopSchema) bool { return s.ICCProfileProperty() == nil }),
		photoshopText("Instructions",
			(*schema.PhotoshopSchema).Instructions,
			(*schema.PhotoshopSchema).SetInstructions,
			func(s *schema.PhotoshopSchema) bool { return s.InstructionsProperty() == nil }),
		photoshopText("Source",
			(*schema.PhotoshopSchema).Source,
			(*schema.PhotoshopSchema).SetSource,
			func(s *schema.PhotoshopSchema) bool { return s.SourceProperty() == nil }),
		photoshopText("State",
			(*schema.PhotoshopSchema).State,
			(*schema.PhotoshopSchema).SetState,
			func(s *schema.PhotoshopSchema) bool { return s.StateProperty() == nil }),
		photoshopText("SupplementalCategories",
			(*schema.PhotoshopSchema).SupplementalCategories,
			(*schema.PhotoshopSchema).SetSupplementalCategories,
			func(s *schema.PhotoshopSchema) bool {
				return s.SupplementalCategoriesProperty() == nil
			}),
		photoshopText("TransmissionReference",
			(*schema.PhotoshopSchema).TransmissionReference,
			(*schema.PhotoshopSchema).SetTransmissionReference,
			func(s *schema.PhotoshopSchema) bool {
				return s.TransmissionReferenceProperty() == nil
			}),
		{
			name:   "Urgency",
			absent: func(s *schema.PhotoshopSchema) bool { return s.UrgencyProperty() == nil },
			exercise: []func(*testing.T, *schema.PhotoshopSchema){
				func(t *testing.T, s *schema.PhotoshopSchema) {
					noError(t, "SetUrgency", s.SetUrgency(5))
					urgency, held := s.Urgency()
					if !held || urgency != 5 {
						t.Errorf("Urgency() = %v, %v, want 5", urgency, held)
					}
				},
				func(t *testing.T, s *schema.PhotoshopSchema) {
					// Java overloads setUrgency on String and Integer.
					noError(t, "SetUrgencyString", s.SetUrgencyString("8"))
					urgency, held := s.Urgency()
					if !held || urgency != 8 {
						t.Errorf("Urgency() = %v, %v, want 8", urgency, held)
					}
				},
			},
		},
		{
			// The two array fields are not in Java's parameter list, because
			// SchemaTester has no case for a Bag of a structured type; they are
			// here so that the other fields are checked against them too.
			name: "DocumentAncestors",
			absent: func(s *schema.PhotoshopSchema) bool {
				return s.DocumentAncestorsProperty() == nil
			},
			exercise: []func(*testing.T, *schema.PhotoshopSchema){
				func(t *testing.T, s *schema.PhotoshopSchema) {
					noError(t, "AddDocumentAncestors", s.AddDocumentAncestors("ancestor one"))
					noError(t, "AddDocumentAncestors", s.AddDocumentAncestors("ancestor two"))
					holdsAll(t, "DocumentAncestors()", s.DocumentAncestors(),
						[]string{"ancestor one", "ancestor two"})
				},
			},
		},
		{
			name: "TextLayers",
			absent: func(s *schema.PhotoshopSchema) bool {
				layers, err := s.TextLayers()
				return err == nil && layers == nil
			},
			exercise: []func(*testing.T, *schema.PhotoshopSchema){
				func(t *testing.T, s *schema.PhotoshopSchema) {
					noError(t, "AddTextLayers", s.AddTextLayers("layer one", "the text"))
					layers, err := s.TextLayers()
					noError(t, "TextLayers", err)
					if len(layers) != 1 {
						t.Fatalf("TextLayers() held %d layers, want 1", len(layers))
					}
					equal(t, "LayerName()", layers[0].LayerName(), "layer one")
					equal(t, "LayerText()", layers[0].LayerText(), "the text")
				},
			},
		},
	})
}
