package fdf_test

// JAVA-BUGS 44: `FDFAnnotationFreeText.setRotation` writes `/Rotate` as an
// integer and `getRotation` reads it with `getString`, which answers only for
// a COSString, so the getter answers null for everything the setter wrote and
// for every conforming file.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fdf"
)

// TestFreeTextRotationRoundTrip is the defect.
//
// The expected value is what was set, written out as the sibling
// `Justification` writes its own integer entry -- `/Rotate` is a number in the
// specification, and the XFDF `rotation` attribute the constructor reads is
// parsed with `Atoi`, so the setter is the half that is right.
func TestFreeTextRotationRoundTrip(t *testing.T) {
	annotation := fdf.NewFDFAnnotationFreeText()
	annotation.SetRotation(90)

	if got := annotation.Rotation(); got != "90" {
		t.Errorf("Rotation() = %q, want %q", got, "90")
	}
}

// TestFreeTextRotationOfNothing is the annotation that has no /Rotate, which
// reads as the zero the sibling getter also defaults to.
func TestFreeTextRotationOfNothing(t *testing.T) {
	if got := fdf.NewFDFAnnotationFreeText().Rotation(); got != "0" {
		t.Errorf("Rotation() of a fresh annotation = %q, want %q", got, "0")
	}
}
