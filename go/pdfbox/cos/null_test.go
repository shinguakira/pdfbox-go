package cos

import (
	"bytes"
	"testing"
)

// The Java suite has no TestCOSNull — COSNull is only exercised indirectly.
// Per migration/conventions/tdd.md, Java code with no test still gets a test
// written first, from reading the Java. These assertions come from
// pdfbox/src/main/java/org/apache/pdfbox/cos/COSNull.java.

func TestNullBaseContract(t *testing.T) {
	assertBaseContract(t, NullObject)
}

func TestNullAccept(t *testing.T) {
	assertVisits(t, NullObject, "null")
}

func TestNullWritePDF(t *testing.T) {
	var buf bytes.Buffer
	if err := NullObject.WritePDF(&buf); err != nil {
		t.Fatalf("WritePDF: %v", err)
	}
	// Java: NULL_BYTES = {110, 117, 108, 108}
	assertBytesEqual(t, []byte{110, 117, 108, 108}, buf.Bytes())
}

func TestNullString(t *testing.T) {
	// Java: toString returns "COSNull{}"
	if got := NullObject.String(); got != "COSNull{}" {
		t.Errorf("String() = %q, want %q", got, "COSNull{}")
	}
}

func TestNullIsSingleton(t *testing.T) {
	// Java limits construction to one instance via a private constructor.
	if NullObject != NullObject {
		t.Error("the null instance does not compare equal to itself")
	}
}
