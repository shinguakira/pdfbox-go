package cos

import (
	"bytes"
	"testing"
)

// Ported from pdfbox/src/test/java/org/apache/pdfbox/cos/TestCOSBoolean.java.

func TestBooleanBaseContract(t *testing.T) {
	// Java: setUp() assigns testCOSBase = COSBoolean.TRUE
	assertBaseContract(t, True)
}

func TestBooleanValue(t *testing.T) {
	if !True.Value() {
		t.Error("True.Value() = false, want true")
	}
	if False.Value() {
		t.Error("False.Value() = true, want false")
	}
}

func TestBooleanGetBoolean(t *testing.T) {
	// Java: testGetBoolean asserts the factory returns the shared instances
	if got := GetBoolean(true); got != True {
		t.Errorf("GetBoolean(true) = %v, want the True instance", got)
	}
	if got := GetBoolean(false); got != False {
		t.Errorf("GetBoolean(false) = %v, want the False instance", got)
	}
}

func TestBooleanEquals(t *testing.T) {
	// Java compares COSBoolean.TRUE against itself three times to check
	// reflexivity, symmetry and transitivity. Java's equals is identity
	// (`this == obj`, correct because only two instances exist); the Go port
	// keeps that by comparing pointers to the two package-level values.
	test1, test2, test3 := True, True, True

	if test1 != test1 {
		t.Error("reflexive: True != True")
	}
	if test2 != test1 || test1 != test2 {
		t.Error("symmetric: True and True do not compare equal both ways")
	}
	if test1 != test2 || test2 != test3 || test1 != test3 {
		t.Error("transitive: True instances do not all compare equal")
	}

	if True == False {
		t.Error("True == False, want them distinct")
	}
}

func TestBooleanAccept(t *testing.T) {
	// This asserts the double dispatch and the token bytes through WritePDF.
	// The COSWriter assertions the Java test makes are in
	// accept_external_test.go, which can import pdfwriter.
	assertVisits(t, True, "boolean")
	assertVisits(t, False, "boolean")

	var buf bytes.Buffer
	if err := True.WritePDF(&buf); err != nil {
		t.Fatalf("True.WritePDF: %v", err)
	}
	assertBytesEqual(t, []byte(True.String()), buf.Bytes())

	buf.Reset()
	if err := False.WritePDF(&buf); err != nil {
		t.Fatalf("False.WritePDF: %v", err)
	}
	assertBytesEqual(t, []byte(False.String()), buf.Bytes())
}

func TestBooleanString(t *testing.T) {
	// Java: toString returns String.valueOf(value)
	if got := True.String(); got != "true" {
		t.Errorf("True.String() = %q, want %q", got, "true")
	}
	if got := False.String(); got != "false" {
		t.Errorf("False.String() = %q, want %q", got, "false")
	}
}
