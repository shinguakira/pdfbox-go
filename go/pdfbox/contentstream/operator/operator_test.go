package operator

import "testing"

// Written from
// pdfbox/src/main/java/org/apache/pdfbox/contentstream/operator/Operator.java.
// The Java suite has no test for this class.

func TestGetInterns(t *testing.T) {
	// Operators are cached, so the same name yields the same instance.
	if Get("Tj") != Get("Tj") {
		t.Error("Get returned distinct instances for the same operator")
	}
	if Get("Tj") == Get("TJ") {
		t.Error("distinct operators share an instance")
	}
	if got := Get("Tj").Name(); got != "Tj" {
		t.Errorf("Name() = %q, want %q", got, "Tj")
	}
}

// TestGetInlineImageOperatorsAreNotCached pins the exception the Java makes:
// BI and ID carry per-occurrence image data and parameters, so caching them
// would let one inline image overwrite another's.
func TestGetInlineImageOperatorsAreNotCached(t *testing.T) {
	for _, name := range []string{BeginInlineImage, BeginInlineImageData} {
		if Get(name) == Get(name) {
			t.Errorf("Get(%q) returned the same instance twice; inline image "+
				"operators carry per-occurrence data and must not be cached", name)
		}
	}
}

func TestGetRejectsNames(t *testing.T) {
	// Java throws IllegalArgumentException for an operator starting with '/',
	// because that is a name, not an operator.
	if _, err := GetChecked("/Type"); err == nil {
		t.Error("GetChecked accepted an operator starting with '/'")
	}
	if _, err := GetChecked("Tj"); err != nil {
		t.Errorf("GetChecked(%q): %v", "Tj", err)
	}
}

func TestString(t *testing.T) {
	// Java: toString returns "PDFOperator{" + theOperator + "}"
	if got := Get("Tj").String(); got != "PDFOperator{Tj}" {
		t.Errorf("String() = %q, want %q", got, "PDFOperator{Tj}")
	}
}

func TestImageDataAndParameters(t *testing.T) {
	op := Get(BeginInlineImage)

	if op.ImageData() != nil {
		t.Error("a fresh operator already carries image data")
	}
	op.SetImageData([]byte{1, 2, 3})
	if got := op.ImageData(); len(got) != 3 {
		t.Errorf("ImageData() has %d bytes, want 3", len(got))
	}

	if op.ImageParameters() != nil {
		t.Error("a fresh operator already carries image parameters")
	}
}
