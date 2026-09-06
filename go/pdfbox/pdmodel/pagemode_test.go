package pdmodel

// Ported from pdfbox/src/test/java/org/apache/pdfbox/pdmodel/PageModeTest.java
// and PageLayoutTest.java. Java gives them a file each; they are a few lines
// apiece and cover the two halves of the same shape, so the port keeps them
// together.
//
// Java asserts assertThrows(IllegalArgumentException.class, ...) for a value
// that is not one of the constants. The port's PageModeFromString answers an
// error rather than panicking -- the value comes out of a PDF, and
// conventions/java-to-go.md keeps a panic for a library bug -- so those two
// assertions read the error.

import (
	"testing"
)

// TestPageModeFromString is PageModeTest's six fromString cases.
func TestPageModeFromString(t *testing.T) {
	for _, want := range []struct {
		value string
		mode  PageMode
	}{
		{"FullScreen", PageModeFullScreen},
		{"UseThumbs", PageModeUseThumbs},
		{"UseOC", PageModeUseOptionalContent},
		{"UseNone", PageModeUseNone},
		{"UseAttachments", PageModeUseAttachments},
		{"UseOutlines", PageModeUseOutlines},
	} {
		retval, err := PageModeFromString(want.value)
		if err != nil {
			t.Errorf("PageModeFromString(%q) = %v", want.value, err)
			continue
		}
		if retval != want.mode {
			t.Errorf("PageModeFromString(%q) = %q, want %q", want.value, retval, want.mode)
		}
	}
}

// TestPageModeFromStringUnknown is
// fromStringInputNotNullOutputIllegalArgumentException and its second case.
func TestPageModeFromStringUnknown(t *testing.T) {
	for _, value := range []string{"", "Dulacb`ecj"} {
		if _, err := PageModeFromString(value); err == nil {
			t.Errorf("PageModeFromString(%q) = nil error, want one", value)
		}
	}
}

// TestPageModeStringValue is stringValueOutputNotNull.
func TestPageModeStringValue(t *testing.T) {
	objectUnderTest := PageModeUseOptionalContent
	if retval := objectUnderTest.StringValue(); retval != "UseOC" {
		t.Errorf("StringValue() = %q, want %q", retval, "UseOC")
	}
}

// TestPageLayoutValues is PageLayoutTest.testValues, a test for completeness
// (PDFBOX-3362): every constant has a distinct string value, and every string
// value maps back to a distinct constant.
func TestPageLayoutValues(t *testing.T) {
	pageLayoutSet := map[PageLayout]bool{}
	stringSet := map[string]bool{}
	for _, pl := range PageLayoutValues() {
		s := pl.StringValue()
		stringSet[s] = true
		layout, err := PageLayoutFromString(s)
		if err != nil {
			t.Fatalf("PageLayoutFromString(%q) = %v", s, err)
		}
		pageLayoutSet[layout] = true
	}
	if got, want := len(pageLayoutSet), len(PageLayoutValues()); got != want {
		t.Errorf("distinct layouts = %d, want %d", got, want)
	}
	if got, want := len(stringSet), len(PageLayoutValues()); got != want {
		t.Errorf("distinct strings = %d, want %d", got, want)
	}
}

// TestPageLayoutFromStringUnknown is PageLayoutTest's
// fromStringInputNotNullOutputIllegalArgumentException.
func TestPageLayoutFromStringUnknown(t *testing.T) {
	if _, err := PageLayoutFromString("SinglePag"); err == nil {
		t.Error(`PageLayoutFromString("SinglePag") = nil error, want one`)
	}
}
