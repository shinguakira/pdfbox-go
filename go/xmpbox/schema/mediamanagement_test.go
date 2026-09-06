package schema_test

// Port of org.apache.xmpbox.schema.XMPMediaManagementTest.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
)

// freshMediaManagement is the @BeforeEach: a new packet with the schema added.
func freshMediaManagement(t *testing.T) *schema.XMPMediaManagementSchema {
	t.Helper()
	mm, err := xmpbox.CreateXMPMetadata().CreateAndAddXMPMediaManagementSchema()
	if err != nil {
		t.Fatalf("CreateAndAddXMPMediaManagementSchema: %v", err)
	}
	return mm
}

// mediaManagementText is one of the fields the test sets and reads back as
// text.
func mediaManagementText(name, value string,
	get func(*schema.XMPMediaManagementSchema) string,
	set func(*schema.XMPMediaManagementSchema, string) error,
	property func(*schema.XMPMediaManagementSchema) bool,
) fieldCase[schema.XMPMediaManagementSchema] {
	return fieldCase[schema.XMPMediaManagementSchema]{
		name:   name,
		absent: property,
		exercise: []func(*testing.T, *schema.XMPMediaManagementSchema){
			func(t *testing.T, s *schema.XMPMediaManagementSchema) {
				noError(t, "Set"+name, set(s, value))
				equal(t, name+"()", get(s), value)
			},
		},
	}
}

func TestXMPMediaManagement(t *testing.T) {
	runFieldCases(t, freshMediaManagement, []fieldCase[schema.XMPMediaManagementSchema]{
		mediaManagementText("DocumentID", "uuid:FB031973-5E75-11B2-8F06-E7F5C101C07A",
			(*schema.XMPMediaManagementSchema).DocumentID,
			(*schema.XMPMediaManagementSchema).SetDocumentID,
			func(s *schema.XMPMediaManagementSchema) bool {
				return s.DocumentIDProperty() == nil
			}),
		mediaManagementText("Manager", "Raoul",
			(*schema.XMPMediaManagementSchema).Manager,
			(*schema.XMPMediaManagementSchema).SetManager,
			func(s *schema.XMPMediaManagementSchema) bool { return s.ManagerProperty() == nil }),
		mediaManagementText("ManageTo", "uuid:36",
			(*schema.XMPMediaManagementSchema).ManageTo,
			(*schema.XMPMediaManagementSchema).SetManageTo,
			func(s *schema.XMPMediaManagementSchema) bool { return s.ManageToProperty() == nil }),
		mediaManagementText("ManageUI", "uuid:3635",
			(*schema.XMPMediaManagementSchema).ManageUI,
			(*schema.XMPMediaManagementSchema).SetManageUI,
			func(s *schema.XMPMediaManagementSchema) bool { return s.ManageUIProperty() == nil }),
		mediaManagementText("InstanceID", "uuid:42",
			(*schema.XMPMediaManagementSchema).InstanceID,
			(*schema.XMPMediaManagementSchema).SetInstanceID,
			func(s *schema.XMPMediaManagementSchema) bool {
				return s.InstanceIDProperty() == nil
			}),
		mediaManagementText("OriginalDocumentID", "uuid:142",
			(*schema.XMPMediaManagementSchema).OriginalDocumentID,
			(*schema.XMPMediaManagementSchema).SetOriginalDocumentID,
			func(s *schema.XMPMediaManagementSchema) bool {
				return s.OriginalDocumentIDProperty() == nil
			}),
		mediaManagementText("RenditionParams", "my params",
			(*schema.XMPMediaManagementSchema).RenditionParams,
			(*schema.XMPMediaManagementSchema).SetRenditionParams,
			func(s *schema.XMPMediaManagementSchema) bool {
				return s.RenditionParamsProperty() == nil
			}),
		mediaManagementText("VersionID", "14",
			(*schema.XMPMediaManagementSchema).VersionID,
			(*schema.XMPMediaManagementSchema).SetVersionID,
			func(s *schema.XMPMediaManagementSchema) bool { return s.VersionIDProperty() == nil }),
		mediaManagementText("ManagerVariant", "the variant",
			(*schema.XMPMediaManagementSchema).ManagerVariant,
			(*schema.XMPMediaManagementSchema).SetManagerVariant,
			func(s *schema.XMPMediaManagementSchema) bool {
				return s.ManagerVariantProperty() == nil
			}),
		mediaManagementText("RenditionClass", "the class",
			(*schema.XMPMediaManagementSchema).RenditionClass,
			(*schema.XMPMediaManagementSchema).SetRenditionClass,
			func(s *schema.XMPMediaManagementSchema) bool {
				return s.RenditionClassProperty() == nil
			}),
		{
			name:   "SaveID",
			absent: func(s *schema.XMPMediaManagementSchema) bool { return s.SaveIDProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPMediaManagementSchema){
				func(t *testing.T, s *schema.XMPMediaManagementSchema) {
					noError(t, "SetSaveID", s.SetSaveID(36))
					saveID, held := s.SaveID()
					if !held || saveID != 36 {
						t.Errorf("SaveID() = %v, %v, want 36", saveID, held)
					}
				},
			},
		},
		{
			name:   "Versions",
			absent: func(s *schema.XMPMediaManagementSchema) bool { return s.VersionsProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPMediaManagementSchema){
				func(t *testing.T, s *schema.XMPMediaManagementSchema) {
					for _, version := range []string{"1", "2", "3"} {
						noError(t, "AddVersions", s.AddVersions(version))
					}
					holdsAll(t, "Versions()", s.Versions(), []string{"1", "2", "3"})
				},
			},
		},
		{
			name:   "History",
			absent: func(s *schema.XMPMediaManagementSchema) bool { return s.HistoryProperty() == nil },
			exercise: []func(*testing.T, *schema.XMPMediaManagementSchema){
				func(t *testing.T, s *schema.XMPMediaManagementSchema) {
					for _, action := range []string{"action 1", "action 2", "action 3"} {
						noError(t, "AddHistory", s.AddHistory(action))
					}
					// PDFBOX-6111: there is no getHistory, because the field is
					// an array of a structured type and not of a text value; the
					// array itself is what the test can read back.
					history := s.HistoryProperty()
					if history == nil {
						t.Fatal("HistoryProperty() = nil, want the array just added")
					}
					holdsAll(t, "History()", history.ElementsAsString(),
						[]string{"action 1", "action 2", "action 3"})
				},
			},
		},
		{
			name: "Ingredients",
			absent: func(s *schema.XMPMediaManagementSchema) bool {
				return s.IngredientsProperty() == nil
			},
			exercise: []func(*testing.T, *schema.XMPMediaManagementSchema){
				func(t *testing.T, s *schema.XMPMediaManagementSchema) {
					for _, resource := range []string{"resource1", "resource2"} {
						noError(t, "AddIngredients", s.AddIngredients(resource))
					}
					holdsAll(t, "Ingredients()", s.Ingredients(),
						[]string{"resource1", "resource2"})
				},
			},
		},
	})
}
