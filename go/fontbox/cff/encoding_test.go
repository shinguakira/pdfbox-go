package cff

import "testing"

// Port of org.apache.fontbox.cff.CFFEncodingTest.

func assertEncoding(t *testing.T, e *CFFEncoding, code int, name string) {
	t.Helper()
	if got := e.Name(code); got != name {
		t.Errorf("Name(%d) = %q, want %q", code, got, name)
	}
}

func assertEncodingCode(t *testing.T, e *CFFEncoding, name string, code int) {
	t.Helper()
	got, ok := e.Code(name)
	if !ok || got != code {
		t.Errorf("Code(%s) = %d, want %d", name, got, code)
	}
}

func TestCFFExpertEncoding(t *testing.T) {
	// check some randomly chosen mappings
	assertEncoding(t, CFFExpertEncoding, 0, ".notdef")
	assertEncoding(t, CFFExpertEncoding, 32, "space")
	assertEncoding(t, CFFExpertEncoding, 112, "Psmall")
	assertEncoding(t, CFFExpertEncoding, 251, "Ucircumflexsmall")
	assertEncodingCode(t, CFFExpertEncoding, "space", 32)
	assertEncodingCode(t, CFFExpertEncoding, "Psmall", 112)
	assertEncodingCode(t, CFFExpertEncoding, "Ucircumflexsmall", 251)
}

func TestCFFStandardEncoding(t *testing.T) {
	// check some randomly chosen mappings
	assertEncoding(t, CFFStandardEncoding, 0, ".notdef")
	assertEncoding(t, CFFStandardEncoding, 32, "space")
	assertEncoding(t, CFFStandardEncoding, 112, "p")
	assertEncoding(t, CFFStandardEncoding, 251, "germandbls")
	assertEncodingCode(t, CFFStandardEncoding, "space", 32)
	assertEncodingCode(t, CFFStandardEncoding, "p", 112)
	assertEncodingCode(t, CFFStandardEncoding, "germandbls", 251)
}
