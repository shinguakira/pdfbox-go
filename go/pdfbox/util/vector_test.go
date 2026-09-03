package util

import "testing"

// Written from org.apache.pdfbox.util.Vector; the Java suite has no test for
// it, so per migration/conventions/tdd.md these come from the source.

func TestVector(t *testing.T) {
	v := NewVector(3, 4)
	if v.X() != 3 || v.Y() != 4 {
		t.Errorf("vector = (%v, %v), want (3, 4)", v.X(), v.Y())
	}

	scaled := v.Scale(2)
	if scaled.X() != 6 || scaled.Y() != 8 {
		t.Errorf("scaled = (%v, %v), want (6, 8)", scaled.X(), scaled.Y())
	}
	if v.X() != 3 || v.Y() != 4 {
		t.Error("Scale changed the vector it was called on")
	}
}

func TestVectorString(t *testing.T) {
	if got, want := NewVector(1, 2).String(), "(1.0, 2.0)"; got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}
