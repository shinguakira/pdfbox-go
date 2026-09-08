package logicalstructure_test

// JAVA-BUGS 39: `PDStructureNode.insertBefore` takes the index of the
// reference kid without checking it, and `indexOfObject` answers -1 when the
// kid is not in the array, so the insert throws IndexOutOfBoundsException.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
)

// TestInsertBeforeAKidThatIsNotThereDoesNothing is the defect.
//
// The expected behaviour is the single-kid branch's, five lines below: it
// compares the reference kid and does nothing when it does not match. The
// array branch must answer the same question the same way, since the two
// branches are one method's two shapes of `/K`. Java hands -1 to
// `List.add(int, E)` and throws.
func TestInsertBeforeAKidThatIsNotThereDoesNothing(t *testing.T) {
	parent := logicalstructure.NewPDStructureElement("Sect", nil)
	first := logicalstructure.NewPDStructureElement("P", parent)
	second := logicalstructure.NewPDStructureElement("P", parent)
	parent.AppendKid(first)
	parent.AppendKid(second)

	stranger := logicalstructure.NewPDStructureElement("P", nil)
	newKid := logicalstructure.NewPDStructureElement("Span", parent)

	parent.InsertBefore(newKid, stranger)

	if got := len(parent.Kids()); got != 2 {
		t.Errorf("the node has %d kids, want the 2 it had: inserting before a kid "+
			"that is not there must do nothing", got)
	}
}

// TestInsertBeforeAKidThatIsThereStillInserts keeps the case the index check
// must not swallow.
func TestInsertBeforeAKidThatIsThereStillInserts(t *testing.T) {
	parent := logicalstructure.NewPDStructureElement("Sect", nil)
	first := logicalstructure.NewPDStructureElement("P", parent)
	second := logicalstructure.NewPDStructureElement("P", parent)
	parent.AppendKid(first)
	parent.AppendKid(second)

	newKid := logicalstructure.NewPDStructureElement("Span", parent)
	parent.InsertBefore(newKid, second)

	kids := parent.Kids()
	if len(kids) != 3 {
		t.Fatalf("the node has %d kids, want 3", len(kids))
	}
	// the new kid went in front of the one it was inserted before
	inserted, ok := kids[1].(*logicalstructure.PDStructureElement)
	if !ok {
		t.Fatalf("the second kid is %T, want a structure element", kids[1])
	}
	if got := inserted.StandardStructureType(); got != "Span" {
		t.Errorf("the second kid is a %q, want the %q that was inserted", got, "Span")
	}
}
