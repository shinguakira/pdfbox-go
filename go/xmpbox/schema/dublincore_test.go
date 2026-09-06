package schema_test

// Port of org.apache.xmpbox.schema.DublinCoreTest.

import (
	"testing"
	"time"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
)

// freshDublinCore is the @BeforeEach: a new packet with the schema added.
func freshDublinCore(t *testing.T) *schema.DublinCoreSchema {
	t.Helper()
	dc, err := xmpbox.CreateXMPMetadata().CreateAndAddDublinCoreSchema()
	if err != nil {
		t.Fatalf("CreateAndAddDublinCoreSchema: %v", err)
	}
	return dc
}

func TestDublinCore(t *testing.T) {
	// The dates the date field is exercised with, which stand for the two
	// Calendars getJavaValue answers over SchemaTester's random loop.
	firstDate := time.Date(2004, time.February, 12, 15, 19, 21, 0, time.UTC)
	secondDate := time.Date(2011, time.September, 2, 8, 4, 5, 0, time.FixedZone("", 3600))

	runFieldCases(t, freshDublinCore, []fieldCase[schema.DublinCoreSchema]{
		{
			name:   "contributor",
			absent: func(s *schema.DublinCoreSchema) bool { return s.ContributorsProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddContributor", s.AddContributor("contributor one"))
					noError(t, "AddContributor", s.AddContributor("contributor two"))
					holdsAll(t, "Contributors()", s.Contributors(),
						[]string{"contributor one", "contributor two"})
					s.RemoveContributor("contributor one")
					holdsAll(t, "Contributors()", s.Contributors(),
						[]string{"contributor two"})
				},
			},
		},
		{
			name:   "coverage",
			absent: func(s *schema.DublinCoreSchema) bool { return s.CoverageProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "SetCoverage", s.SetCoverage("the coverage"))
					equal(t, "Coverage()", s.Coverage(), "the coverage")
				},
				func(t *testing.T, s *schema.DublinCoreSchema) {
					text, err := s.CreateTextType(schema.DCCoverage, "another coverage")
					noError(t, "CreateTextType", err)
					s.SetCoverageProperty(text)
					equal(t, "Coverage()", s.Coverage(), "another coverage")
				},
			},
		},
		{
			name:   "creator",
			absent: func(s *schema.DublinCoreSchema) bool { return s.CreatorsProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddCreator", s.AddCreator("creator one"))
					noError(t, "AddCreator", s.AddCreator("creator two"))
					holdsAll(t, "Creators()", s.Creators(),
						[]string{"creator one", "creator two"})
					s.RemoveCreator("creator one")
					holdsAll(t, "Creators()", s.Creators(), []string{"creator two"})
				},
			},
		},
		{
			name:   "date",
			absent: func(s *schema.DublinCoreSchema) bool { return s.DatesProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddDate", s.AddDate(firstDate))
					noError(t, "AddDate", s.AddDate(secondDate))
					dates := s.Dates()
					if len(dates) != 2 {
						t.Fatalf("Dates() = %v, want two dates", dates)
					}
					if !dates[0].Equal(firstDate) || !dates[1].Equal(secondDate) {
						t.Errorf("Dates() = %v, want %v and %v", dates, firstDate, secondDate)
					}
					s.RemoveDate(firstDate)
					if dates := s.Dates(); len(dates) != 1 || !dates[0].Equal(secondDate) {
						t.Errorf("Dates() = %v, want just %v", dates, secondDate)
					}
				},
			},
		},
		{
			name:   "format",
			absent: func(s *schema.DublinCoreSchema) bool { return s.FormatProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "SetFormat", s.SetFormat("application/pdf"))
					equal(t, "Format()", s.Format(), "application/pdf")
				},
			},
		},
		{
			name:   "identifier",
			absent: func(s *schema.DublinCoreSchema) bool { return s.IdentifierProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "SetIdentifier", s.SetIdentifier("the identifier"))
					equal(t, "Identifier()", s.Identifier(), "the identifier")
				},
				func(t *testing.T, s *schema.DublinCoreSchema) {
					text, err := s.CreateTextType(schema.DCIdentifier, "another identifier")
					noError(t, "CreateTextType", err)
					s.SetIdentifierProperty(text)
					equal(t, "Identifier()", s.Identifier(), "another identifier")
				},
			},
		},
		{
			name:   "language",
			absent: func(s *schema.DublinCoreSchema) bool { return s.LanguagesProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddLanguage", s.AddLanguage("en"))
					noError(t, "AddLanguage", s.AddLanguage("fr"))
					holdsAll(t, "Languages()", s.Languages(), []string{"en", "fr"})
					s.RemoveLanguage("en")
					holdsAll(t, "Languages()", s.Languages(), []string{"fr"})
				},
			},
		},
		{
			name:   "publisher",
			absent: func(s *schema.DublinCoreSchema) bool { return s.PublishersProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddPublisher", s.AddPublisher("publisher one"))
					noError(t, "AddPublisher", s.AddPublisher("publisher two"))
					holdsAll(t, "Publishers()", s.Publishers(),
						[]string{"publisher one", "publisher two"})
					s.RemovePublisher("publisher one")
					holdsAll(t, "Publishers()", s.Publishers(), []string{"publisher two"})
				},
			},
		},
		{
			name:   "relation",
			absent: func(s *schema.DublinCoreSchema) bool { return s.RelationsProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddRelation", s.AddRelation("relation one"))
					noError(t, "AddRelation", s.AddRelation("relation two"))
					holdsAll(t, "Relations()", s.Relations(),
						[]string{"relation one", "relation two"})
					s.RemoveRelation("relation one")
					holdsAll(t, "Relations()", s.Relations(), []string{"relation two"})
				},
			},
		},
		{
			name:   "source",
			absent: func(s *schema.DublinCoreSchema) bool { return s.SourceProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "SetSource", s.SetSource("the source"))
					equal(t, "Source()", s.Source(), "the source")
				},
				func(t *testing.T, s *schema.DublinCoreSchema) {
					text, err := s.CreateTextType(schema.DCSource, "another source")
					noError(t, "CreateTextType", err)
					s.SetSourceProperty(text)
					equal(t, "Source()", s.Source(), "another source")
				},
			},
		},
		{
			name:   "subject",
			absent: func(s *schema.DublinCoreSchema) bool { return s.SubjectsProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddSubject", s.AddSubject("subject one"))
					noError(t, "AddSubject", s.AddSubject("subject two"))
					holdsAll(t, "Subjects()", s.Subjects(),
						[]string{"subject one", "subject two"})
					s.RemoveSubject("subject one")
					holdsAll(t, "Subjects()", s.Subjects(), []string{"subject two"})
				},
			},
		},
		{
			name:   "type",
			absent: func(s *schema.DublinCoreSchema) bool { return s.TypesProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddType", s.AddType("type one"))
					noError(t, "AddType", s.AddType("type two"))
					holdsAll(t, "Types()", s.Types(), []string{"type one", "type two"})
					s.RemoveType("type one")
					holdsAll(t, "Types()", s.Types(), []string{"type two"})
				},
			},
		},
		{
			// The three language alternatives are not in Java's parameter list,
			// because SchemaTester has no case for a LangAlt; they are here so
			// that the other fields are checked against them too.
			name:   "description",
			absent: func(s *schema.DublinCoreSchema) bool { return s.DescriptionProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddDescription", s.AddDescription("en", "the description"))
					description, err := s.DescriptionOfLanguage("en")
					noError(t, "DescriptionOfLanguage", err)
					equal(t, "DescriptionOfLanguage(\"en\")", description, "the description")
				},
			},
		},
		{
			name:   "rights",
			absent: func(s *schema.DublinCoreSchema) bool { return s.RightsProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddRights", s.AddRights("en", "the rights"))
					rights, err := s.RightsOfLanguage("en")
					noError(t, "RightsOfLanguage", err)
					equal(t, "RightsOfLanguage(\"en\")", rights, "the rights")
				},
			},
		},
		{
			name:   "title",
			absent: func(s *schema.DublinCoreSchema) bool { return s.TitleProperty() == nil },
			exercise: []func(*testing.T, *schema.DublinCoreSchema){
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "SetTitle", s.SetTitle("the title"))
					title, err := s.Title()
					noError(t, "Title", err)
					equal(t, "Title()", title, "the title")
				},
				func(t *testing.T, s *schema.DublinCoreSchema) {
					noError(t, "AddTitle", s.AddTitle("fr", "le titre"))
					title, err := s.TitleOfLanguage("fr")
					noError(t, "TitleOfLanguage", err)
					equal(t, "TitleOfLanguage(\"fr\")", title, "le titre")
					languages, err := s.TitleLanguages()
					noError(t, "TitleLanguages", err)
					holdsAll(t, "TitleLanguages()", languages, []string{"fr"})
				},
			},
		},
	})
}
