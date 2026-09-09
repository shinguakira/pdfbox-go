package logicalstructure_test

// JAVA-BUGS 38: `PDUserAttributeObject` reads `/P` with `getCOSArray`, which
// answers null when the entry is absent, and all three of its methods use the
// result without a check -- so a fresh object, whose constructor writes only
// `/O`, throws from every one of them.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
)

// TestUserAttributeObjectWithNoPropertiesIsEmpty is the getter.
//
// The expected value is the empty list: an object with no `/P` has no user
// properties, which is the answer the rest of the class gives for an entry it
// does not have. Java dereferences the null and throws
// NullPointerException.
func TestUserAttributeObjectWithNoPropertiesIsEmpty(t *testing.T) {
	object := logicalstructure.NewPDUserAttributeObject()
	if got := object.OwnerUserProperties(); len(got) != 0 {
		t.Errorf("a fresh user attribute object has %d properties, want none", len(got))
	}
}

// TestAddUserPropertyToAFreshObject is the one the entry calls the way in.
//
// PDF 32000-1:2008 table 328 marks `/P` required, so a method that adds a
// property to an object without one writes the array it needs. Without it the
// object cannot be filled in through its own API at all.
func TestAddUserPropertyToAFreshObject(t *testing.T) {
	object := logicalstructure.NewPDUserAttributeObject()
	property := logicalstructure.NewPDUserProperty(object)
	property.SetName("colour")

	object.AddUserProperty(property)

	got := object.OwnerUserProperties()
	if len(got) != 1 {
		t.Fatalf("the object holds %d properties, want 1", len(got))
	}
	if name := got[0].Name(); name != "colour" {
		t.Errorf("the property is named %q, want %q", name, "colour")
	}
}

// TestRemoveUserPropertyFromAFreshObject is the third of the three.
//
// There is nothing to remove, so nothing is removed and nothing throws.
func TestRemoveUserPropertyFromAFreshObject(t *testing.T) {
	object := logicalstructure.NewPDUserAttributeObject()
	property := logicalstructure.NewPDUserProperty(object)
	property.SetName("colour")

	object.RemoveUserProperty(property)

	if got := object.OwnerUserProperties(); len(got) != 0 {
		t.Errorf("the object holds %d properties, want none", len(got))
	}
}
