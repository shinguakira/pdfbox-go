package logicalstructure_test

// JAVA-BUGS 37: `PDMarkInfo.setSuspect` ignores its argument and always writes
// false, so `/Suspects` can never be raised through it.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/documentinterchange/logicalstructure"
)

// TestSetSuspectWritesItsArgument is the defect.
//
// The expected behaviour is the setter's own name and the two setters beside
// it, `SetMarked` and `SetUserProperties`, which pass their argument through.
// PDF 32000-1:2008 table 321 gives /Suspects as the flag a producer raises
// when the tagged structure may not conform, so a producer that has reason to
// raise it must be able to.
func TestSetSuspectWritesItsArgument(t *testing.T) {
	markInfo := logicalstructure.NewPDMarkInfo()
	markInfo.SetSuspect(true)
	if !markInfo.IsSuspect() {
		t.Error("SetSuspect(true) left /Suspects false")
	}
	markInfo.SetSuspect(false)
	if markInfo.IsSuspect() {
		t.Error("SetSuspect(false) left /Suspects true")
	}
}

// TestMarkInfoSettersAreIndependent keeps the two flags apart: raising one
// must not touch the other, which is what makes the entry a slip in one
// method rather than a shape the class has.
func TestMarkInfoSettersAreIndependent(t *testing.T) {
	markInfo := logicalstructure.NewPDMarkInfo()
	markInfo.SetSuspect(true)
	markInfo.SetUserProperties(false)
	markInfo.SetMarked(true)

	if !markInfo.IsSuspect() {
		t.Error("/Suspects was cleared by another setter")
	}
	if markInfo.UsesUserProperties() {
		t.Error("/UserProperties is true")
	}
	if !markInfo.IsMarked() {
		t.Error("/Marked is false")
	}
}
