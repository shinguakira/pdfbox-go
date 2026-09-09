package schema_test

// JAVA-BUGS 55: `XMPMediaManagementSchema.addVersions` adds to a field
// declared `Seq` with `addQualifiedBagValue`, which makes a `Bag`.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/xmpbox"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/schema"
	"github.com/shinguakira/pdfbox-go/go/xmpbox/xmptype"
)

// TestVersionsIsASequence is the defect.
//
// The expected array type is the one the field is declared with,
// `Cardinality.Seq`, which is also what the neighbouring `AddHistory` writes
// for its own `Seq` field. A document written the other way carries an
// `rdf:Bag` where the schema says `rdf:Seq`, and reading it back in strict
// mode fails with "Invalid array type, expecting Seq and found Bag".
func TestVersionsIsASequence(t *testing.T) {
	metadata := xmpbox.CreateXMPMetadata()
	media, err := schema.NewXMPMediaManagementSchema(metadata)
	noError(t, "NewXMPMediaManagementSchema", err)

	noError(t, "AddVersions", media.AddVersions("1.0"))

	versions := media.VersionsProperty()
	if versions == nil {
		t.Fatal("VersionsProperty() answered nothing for a version that was added")
	}
	if got := versions.ArrayType(); got != xmptype.Seq {
		t.Errorf("the versions array is a %v, want a %v", got, xmptype.Seq)
	}
	holdsAll(t, "Versions", media.Versions(), []string{"1.0"})
}
