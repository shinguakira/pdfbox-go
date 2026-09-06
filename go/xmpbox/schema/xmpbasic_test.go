package schema_test

// Port of org.apache.xmpbox.schema.XMPBasicTest.

import (
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
)

// freshXMPBasic is the @BeforeEach: a new packet with the schema added.
func freshXMPBasic(t *testing.T) *schema.XMPBasicSchema {
	t.Helper()
	basic, err := xmpbox.CreateXMPMetadata().CreateAndAddXMPBasicSchema()
	if err != nil {
		t.Fatalf("CreateAndAddXMPBasicSchema: %v", err)
	}
	return basic
}

func TestXMPBasic(t *testing.T) {
	createDate := time.Date(2005, time.March, 4, 11, 12, 13, 0, time.UTC)
	metadataDate := time.Date(2006, time.April, 5, 14, 15, 16, 0, time.UTC)
	modifyDate := time.Date(2007, time.May, 6, 17, 18, 19, 0, time.UTC)

	runFieldCases(t, freshXMPBasic, []fieldCase[schema.XMPBasicSchema]{
		{
			name:   "Advisory",
			absent: func(s *schema.XMPBasicSchema) bool { return s.AdvisoryProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "AddAdvisory", s.AddAdvisory("xpath1"))
					noError(t, "AddAdvisory", s.AddAdvisory("xpath2"))
					holdsAll(t, "Advisory()", s.Advisory(), []string{"xpath1", "xpath2"})
					s.RemoveAdvisory("xpath1")
					holdsAll(t, "Advisory()", s.Advisory(), []string{"xpath2"})
				},
			},
		},
		{
			name:   "BaseURL",
			absent: func(s *schema.XMPBasicSchema) bool { return s.BaseURLProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "SetBaseURL", s.SetBaseURL("URL"))
					equal(t, "BaseURL()", s.BaseURL(), "URL")
				},
				func(t *testing.T, s *schema.XMPBasicSchema) {
					url, err := s.Metadata().TypeMapping().CreateURL("", s.Prefix(),
						schema.BasicBaseURL, "another URL")
					noError(t, "CreateURL", err)
					s.SetBaseURLProperty(url)
					equal(t, "BaseURL()", s.BaseURL(), "another URL")
				},
			},
		},
		{
			name:   "CreateDate",
			absent: func(s *schema.XMPBasicSchema) bool { return s.CreateDateProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "SetCreateDate", s.SetCreateDate(createDate))
					got, held := s.CreateDate()
					if !held || !got.Equal(createDate) {
						t.Errorf("CreateDate() = %v, %v, want %v", got, held, createDate)
					}
				},
				func(t *testing.T, s *schema.XMPBasicSchema) {
					date, err := s.Metadata().TypeMapping().CreateDate("", s.Prefix(),
						schema.BasicCreateDate, metadataDate)
					noError(t, "CreateDate", err)
					s.SetCreateDateProperty(date)
					got, held := s.CreateDate()
					if !held || !got.Equal(metadataDate) {
						t.Errorf("CreateDate() = %v, %v, want %v", got, held, metadataDate)
					}
				},
			},
		},
		{
			name:   "CreatorTool",
			absent: func(s *schema.XMPBasicSchema) bool { return s.CreatorToolProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "SetCreatorTool", s.SetCreatorTool("CreatorTool"))
					equal(t, "CreatorTool()", s.CreatorTool(), "CreatorTool")
				},
				func(t *testing.T, s *schema.XMPBasicSchema) {
					agent, err := s.Metadata().TypeMapping().CreateAgentName("", s.Prefix(),
						schema.BasicCreatorTool, "another tool")
					noError(t, "CreateAgentName", err)
					s.SetCreatorToolProperty(agent)
					equal(t, "CreatorTool()", s.CreatorTool(), "another tool")
				},
			},
		},
		{
			name:   "Identifier",
			absent: func(s *schema.XMPBasicSchema) bool { return s.IdentifiersProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "AddIdentifier", s.AddIdentifier("id1"))
					noError(t, "AddIdentifier", s.AddIdentifier("id2"))
					holdsAll(t, "Identifiers()", s.Identifiers(), []string{"id1", "id2"})
					s.RemoveIdentifier("id1")
					holdsAll(t, "Identifiers()", s.Identifiers(), []string{"id2"})
				},
			},
		},
		{
			name:   "Label",
			absent: func(s *schema.XMPBasicSchema) bool { return s.LabelProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "SetLabel", s.SetLabel("label"))
					equal(t, "Label()", s.Label(), "label")
				},
				func(t *testing.T, s *schema.XMPBasicSchema) {
					text, err := s.CreateTextType(schema.BasicLabel, "another label")
					noError(t, "CreateTextType", err)
					s.SetLabelProperty(text)
					equal(t, "Label()", s.Label(), "another label")
				},
			},
		},
		{
			name:   "MetadataDate",
			absent: func(s *schema.XMPBasicSchema) bool { return s.MetadataDateProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "SetMetadataDate", s.SetMetadataDate(metadataDate))
					got, held := s.MetadataDate()
					if !held || !got.Equal(metadataDate) {
						t.Errorf("MetadataDate() = %v, %v, want %v", got, held, metadataDate)
					}
				},
			},
		},
		{
			name:   "ModifyDate",
			absent: func(s *schema.XMPBasicSchema) bool { return s.ModifyDateProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "SetModifyDate", s.SetModifyDate(modifyDate))
					got, held := s.ModifyDate()
					if !held || !got.Equal(modifyDate) {
						t.Errorf("ModifyDate() = %v, %v, want %v", got, held, modifyDate)
					}
				},
			},
		},
		{
			// ModifierDate is a field of its own, not another name for
			// ModifyDate; it is not in Java's parameter list, and is here so
			// that the other fields are checked against it too.
			name:   "ModifierDate",
			absent: func(s *schema.XMPBasicSchema) bool { return s.ModifierDateProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "SetModifierDate", s.SetModifierDate(modifyDate))
					got, held := s.ModifierDate()
					if !held || !got.Equal(modifyDate) {
						t.Errorf("ModifierDate() = %v, %v, want %v", got, held, modifyDate)
					}
				},
			},
		},
		{
			name:   "Nickname",
			absent: func(s *schema.XMPBasicSchema) bool { return s.NicknameProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "SetNickname", s.SetNickname("nick name"))
					equal(t, "Nickname()", s.Nickname(), "nick name")
				},
				func(t *testing.T, s *schema.XMPBasicSchema) {
					text, err := s.CreateTextType(schema.BasicNickname, "another nickname")
					noError(t, "CreateTextType", err)
					s.SetNicknameProperty(text)
					equal(t, "Nickname()", s.Nickname(), "another nickname")
				},
			},
		},
		{
			name:   "Rating",
			absent: func(s *schema.XMPBasicSchema) bool { return s.RatingProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					noError(t, "SetRating", s.SetRating(7))
					rating, held := s.Rating()
					if !held || rating != 7 {
						t.Errorf("Rating() = %v, %v, want 7", rating, held)
					}
				},
				func(t *testing.T, s *schema.XMPBasicSchema) {
					integer, err := s.Metadata().TypeMapping().CreateInteger("", s.Prefix(),
						schema.BasicRating, 3)
					noError(t, "CreateInteger", err)
					s.SetRatingProperty(integer)
					rating, held := s.Rating()
					if !held || rating != 3 {
						t.Errorf("Rating() = %v, %v, want 3", rating, held)
					}
				},
			},
		},
		{
			name: "Thumbnails",
			absent: func(s *schema.XMPBasicSchema) bool {
				thumbnails, err := s.ThumbnailsProperty()
				return err == nil && thumbnails == nil
			},
			exercise: []func(*testing.T, *schema.XMPBasicSchema){
				func(t *testing.T, s *schema.XMPBasicSchema) {
					const img = "/9j/4AAQSkZJRgABAgEASABIAAD"
					noError(t, "AddThumbnails", s.AddThumbnails(162, 400, "JPEG", img))
					found, err := s.ThumbnailsProperty()
					noError(t, "ThumbnailsProperty", err)
					if len(found) != 1 {
						t.Fatalf("ThumbnailsProperty() held %d thumbnails, want 1", len(found))
					}
					t1 := found[0]
					equal(t, "Height()", intOf(t1.Height()), 162)
					equal(t, "Width()", intOf(t1.Width()), 400)
					equal(t, "Format()", t1.Format(), "JPEG")
					equal(t, "Image()", t1.Image(), img)
				},
			},
		},
	})
}

// intOf drops the "held" result of an accessor that answers an optional
// integer, for a case that has just set one.
func intOf(value int, _ bool) int { return value }
