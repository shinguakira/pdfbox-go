package state

import "testing"

// Port of
// pdfbox/src/test/java/org/apache/pdfbox/pdmodel/graphics/state/RenderingIntentTest.java.
// testIsFill is in that file although it covers RenderingMode; it is kept here
// rather than moved, so that the two files still line up.

func TestFromStringInputNotNullOutputNotNull(t *testing.T) {
	// Arrange
	const value = "AbsoluteColorimetric"

	// Act
	retval := RenderingIntentFromString(value)

	// Assert result
	if retval != AbsoluteColorimetric {
		t.Errorf("got %v, want %v", retval, AbsoluteColorimetric)
	}
}

func TestFromStringInputNotNullOutputNotNull2(t *testing.T) {
	const value = "RelativeColorimetric"

	retval := RenderingIntentFromString(value)

	if retval != RelativeColorimetric {
		t.Errorf("got %v, want %v", retval, RelativeColorimetric)
	}
}

func TestFromStringInputNotNullOutputNotNull3(t *testing.T) {
	const value = "Perceptual"

	retval := RenderingIntentFromString(value)

	if retval != Perceptual {
		t.Errorf("got %v, want %v", retval, Perceptual)
	}
}

func TestFromStringInputNotNullOutputNotNull4(t *testing.T) {
	const value = "Saturation"

	retval := RenderingIntentFromString(value)

	if retval != Saturation {
		t.Errorf("got %v, want %v", retval, Saturation)
	}
}

func TestFromStringInputNotNullOutputNotNull5(t *testing.T) {
	const value = ""

	retval := RenderingIntentFromString(value)

	if retval != RelativeColorimetric {
		t.Errorf("got %v, want %v", retval, RelativeColorimetric)
	}
}

func TestStringValueOutputNotNull(t *testing.T) {
	objectUnderTest := AbsoluteColorimetric

	retval := objectUnderTest.StringValue()

	if retval != "AbsoluteColorimetric" {
		t.Errorf("got %q, want %q", retval, "AbsoluteColorimetric")
	}
}

func TestIsFill(t *testing.T) {
	objectUnderTest := Fill

	retval := objectUnderTest.IsFill()

	if retval != true {
		t.Errorf("got %v, want true", retval)
	}
}
