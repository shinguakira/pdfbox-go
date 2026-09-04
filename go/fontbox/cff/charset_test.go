package cff

import "testing"

// Port of org.apache.fontbox.cff.CFFCharsetTest.
//
// Java's IllegalStateException is unchecked, so the port panics where Java
// throws; assertPanics stands in for assertThrows.

func assertPanics(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", what)
		}
	}()
	f()
}

func TestEmbeddedCharset(t *testing.T) {
	// true -> CFFCharsetCID
	embeddedCharsetCID := NewEmbeddedCharset(true)
	if !embeddedCharsetCID.IsCIDFont() {
		t.Error("IsCIDFont() = false, want true")
	}
	embeddedCharsetCID.AddCID(10, 20)
	// test existing mapping
	if got := embeddedCharsetCID.GIDForCID(20); got != 10 {
		t.Errorf("GIDForCID(20) = %d, want 10", got)
	}
	if got := embeddedCharsetCID.CIDForGID(10); got != 20 {
		t.Errorf("CIDForGID(10) = %d, want 20", got)
	}
	// test not existing mapping
	if got := embeddedCharsetCID.GIDForCID(99); got != 0 {
		t.Errorf("GIDForCID(99) = %d, want 0", got)
	}
	if got := embeddedCharsetCID.CIDForGID(99); got != 0 {
		t.Errorf("CIDForGID(99) = %d, want 0", got)
	}
	// test not allowed method calls
	assertPanics(t, "SIDForGID", func() { embeddedCharsetCID.SIDForGID(0) })
	assertPanics(t, "GIDForSID", func() { embeddedCharsetCID.GIDForSID(0) })
	assertPanics(t, "AddSID", func() { embeddedCharsetCID.AddSID(0, 0, "test") })
	assertPanics(t, "SID", func() { embeddedCharsetCID.SID("test") })
	assertPanics(t, "NameForGID", func() { embeddedCharsetCID.NameForGID(0) })

	// false -> CFFCharsetType1
	embeddedCharsetType1 := NewEmbeddedCharset(false)
	if embeddedCharsetType1.IsCIDFont() {
		t.Error("IsCIDFont() = true, want false")
	}
	embeddedCharsetType1.AddSID(10, 20, "test")
	// test existing mapping
	if got := embeddedCharsetType1.SID("test"); got != 20 {
		t.Errorf("SID(test) = %d, want 20", got)
	}
	if got := embeddedCharsetType1.GIDForSID(20); got != 10 {
		t.Errorf("GIDForSID(20) = %d, want 10", got)
	}
	if got := embeddedCharsetType1.SIDForGID(10); got != 20 {
		t.Errorf("SIDForGID(10) = %d, want 20", got)
	}
	// test not existing mapping
	if got := embeddedCharsetType1.GIDForSID(99); got != 0 {
		t.Errorf("GIDForSID(99) = %d, want 0", got)
	}
	if got := embeddedCharsetType1.SIDForGID(99); got != 0 {
		t.Errorf("SIDForGID(99) = %d, want 0", got)
	}
	// test not allowed method calls
	assertPanics(t, "CIDForGID", func() { embeddedCharsetType1.CIDForGID(0) })
	assertPanics(t, "GIDForCID", func() { embeddedCharsetType1.GIDForCID(0) })
	assertPanics(t, "AddCID", func() { embeddedCharsetType1.AddCID(0, 0) })
}

func TestCFFCharsetCID(t *testing.T) {
	cffCharsetCID := NewCFFCharsetCID()
	if !cffCharsetCID.IsCIDFont() {
		t.Error("IsCIDFont() = false, want true")
	}
	cffCharsetCID.AddCID(10, 20)
	// test existing mapping
	if got := cffCharsetCID.GIDForCID(20); got != 10 {
		t.Errorf("GIDForCID(20) = %d, want 10", got)
	}
	if got := cffCharsetCID.CIDForGID(10); got != 20 {
		t.Errorf("CIDForGID(10) = %d, want 20", got)
	}
	// test not existing mapping
	if got := cffCharsetCID.GIDForCID(99); got != 0 {
		t.Errorf("GIDForCID(99) = %d, want 0", got)
	}
	if got := cffCharsetCID.CIDForGID(99); got != 0 {
		t.Errorf("CIDForGID(99) = %d, want 0", got)
	}
	// test not allowed method calls
	assertPanics(t, "SIDForGID", func() { cffCharsetCID.SIDForGID(0) })
	assertPanics(t, "GIDForSID", func() { cffCharsetCID.GIDForSID(0) })
	assertPanics(t, "AddSID", func() { cffCharsetCID.AddSID(0, 0, "test") })
	assertPanics(t, "SID", func() { cffCharsetCID.SID("test") })
	assertPanics(t, "NameForGID", func() { cffCharsetCID.NameForGID(0) })
}

func TestCFFCharsetType1(t *testing.T) {
	cffCharsetType1 := NewCFFCharsetType1()
	if cffCharsetType1.IsCIDFont() {
		t.Error("IsCIDFont() = true, want false")
	}
	cffCharsetType1.AddSID(10, 20, "test")
	// test existing mapping
	if got := cffCharsetType1.SID("test"); got != 20 {
		t.Errorf("SID(test) = %d, want 20", got)
	}
	if got := cffCharsetType1.GIDForSID(20); got != 10 {
		t.Errorf("GIDForSID(20) = %d, want 10", got)
	}
	if got := cffCharsetType1.SIDForGID(10); got != 20 {
		t.Errorf("SIDForGID(10) = %d, want 20", got)
	}
	// test not existing mapping
	if got := cffCharsetType1.GIDForSID(99); got != 0 {
		t.Errorf("GIDForSID(99) = %d, want 0", got)
	}
	if got := cffCharsetType1.SIDForGID(99); got != 0 {
		t.Errorf("SIDForGID(99) = %d, want 0", got)
	}
	// test not allowed method calls
	assertPanics(t, "CIDForGID", func() { cffCharsetType1.CIDForGID(0) })
	assertPanics(t, "GIDForCID", func() { cffCharsetType1.GIDForCID(0) })
	assertPanics(t, "AddCID", func() { cffCharsetType1.AddCID(0, 0) })
}

// assertCharsetMapping checks the three ways round one charset row goes.
func assertCharsetMapping(t *testing.T, charset *CFFCharsetType1, gid, sid int, name string) {
	t.Helper()
	if got := charset.SIDForGID(gid); got != sid {
		t.Errorf("SIDForGID(%d) = %d, want %d", gid, got, sid)
	}
	if got := charset.SID(name); got != sid {
		t.Errorf("SID(%s) = %d, want %d", name, got, sid)
	}
	if got, ok := charset.NameForGID(gid); !ok || got != name {
		t.Errorf("NameForGID(%d) = %q, want %q", gid, got, name)
	}
}

func TestCFFExpertCharset(t *testing.T) {
	// check .notdef mapping
	assertCharsetMapping(t, CFFExpertCharset, 0, 0, ".notdef")
	// check some randomly chosen mappings
	assertCharsetMapping(t, CFFExpertCharset, 32, 253, "asuperior")
	assertCharsetMapping(t, CFFExpertCharset, 17, 240, "oneoldstyle")
	assertCharsetMapping(t, CFFExpertCharset, 134, 347, "Agravesmall")
}

func TestCFFExpertSubsetCharset(t *testing.T) {
	// check .notdef mapping
	assertCharsetMapping(t, CFFExpertSubsetCharset, 0, 0, ".notdef")
	// check some randomly chosen mappings
	assertCharsetMapping(t, CFFExpertSubsetCharset, 19, 246, "sevenoldstyle")
	assertCharsetMapping(t, CFFExpertSubsetCharset, 61, 324, "onethird")
	assertCharsetMapping(t, CFFExpertSubsetCharset, 85, 345, "periodinferior")
}

func TestCFFISOAdobeCharset(t *testing.T) {
	// check .notdef mapping
	assertCharsetMapping(t, CFFISOAdobeCharset, 0, 0, ".notdef")
	// check some randomly chosen mappings
	assertCharsetMapping(t, CFFISOAdobeCharset, 32, 32, "question")
	assertCharsetMapping(t, CFFISOAdobeCharset, 76, 76, "k")
	assertCharsetMapping(t, CFFISOAdobeCharset, 218, 218, "odieresis")
}
