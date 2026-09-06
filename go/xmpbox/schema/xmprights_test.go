package schema_test

// Port of org.apache.xmpbox.schema.XmpRightsSchemaTest.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
)

// freshXMPRights is the @BeforeEach: a new packet with the schema added.
func freshXMPRights(t *testing.T) *schema.XMPRightsManagementSchema {
	t.Helper()
	rights, err := xmpbox.CreateXMPMetadata().CreateAndAddXMPRightsManagementSchema()
	if err != nil {
		t.Fatalf("CreateAndAddXMPRightsManagementSchema: %v", err)
	}
	return rights
}

func TestXmpRightsSchema(t *testing.T) {
	// The two languages Java's parameter list gives the usage terms.
	usageTerms := map[string]string{
		"fr": "Termes d'utilisation",
		"en": "Usage Terms",
	}

	runFieldCases(t, freshXMPRights, []fieldCase[schema.XMPRightsManagementSchema]{
		{
			name: "Certificate",
			absent: func(s *schema.XMPRightsManagementSchema) bool {
				return s.CertificateProperty() == nil
			},
			exercise: []func(*testing.T, *schema.XMPRightsManagementSchema){
				func(t *testing.T, s *schema.XMPRightsManagementSchema) {
					const url = "http://une.url.vers.un.certificat/moncert.cer"
					noError(t, "SetCertificate", s.SetCertificate(url))
					equal(t, "Certificate()", s.Certificate(), url)
				},
			},
		},
		{
			name: "Marked",
			absent: func(s *schema.XMPRightsManagementSchema) bool {
				return s.MarkedProperty() == nil
			},
			exercise: []func(*testing.T, *schema.XMPRightsManagementSchema){
				func(t *testing.T, s *schema.XMPRightsManagementSchema) {
					noError(t, "SetMarked", s.SetMarked(true))
					marked, held := s.Marked()
					if !held || !marked {
						t.Errorf("Marked() = %v, %v, want true", marked, held)
					}
				},
			},
		},
		{
			name: "Owner",
			absent: func(s *schema.XMPRightsManagementSchema) bool {
				return s.OwnersProperty() == nil
			},
			exercise: []func(*testing.T, *schema.XMPRightsManagementSchema){
				func(t *testing.T, s *schema.XMPRightsManagementSchema) {
					noError(t, "AddOwner", s.AddOwner("OwnerName"))
					holdsAll(t, "Owners()", s.Owners(), []string{"OwnerName"})
					s.RemoveOwner("OwnerName")
					if owners := s.Owners(); len(owners) != 0 {
						t.Errorf("Owners() = %v, want none left", owners)
					}
					noError(t, "AddOwner", s.AddOwner("OwnerName"))
				},
			},
		},
		{
			name: "UsageTerms",
			absent: func(s *schema.XMPRightsManagementSchema) bool {
				return s.UsageTermsProperty() == nil
			},
			exercise: []func(*testing.T, *schema.XMPRightsManagementSchema){
				func(t *testing.T, s *schema.XMPRightsManagementSchema) {
					for language, terms := range usageTerms {
						noError(t, "AddUsageTerms", s.AddUsageTerms(language, terms))
					}
					languages, err := s.UsageTermsLanguages()
					noError(t, "UsageTermsLanguages", err)
					for _, language := range languages {
						got, err := s.UsageTermsOfLanguage(language)
						noError(t, "UsageTermsOfLanguage", err)
						equal(t, "UsageTermsOfLanguage("+language+")", got, usageTerms[language])
					}
				},
			},
		},
		{
			name: "WebStatement",
			absent: func(s *schema.XMPRightsManagementSchema) bool {
				return s.WebStatementProperty() == nil
			},
			exercise: []func(*testing.T, *schema.XMPRightsManagementSchema){
				func(t *testing.T, s *schema.XMPRightsManagementSchema) {
					const url = "http://une.url.vers.une.page.fr/"
					noError(t, "SetWebStatement", s.SetWebStatement(url))
					equal(t, "WebStatement()", s.WebStatement(), url)
				},
			},
		},
	})
}
