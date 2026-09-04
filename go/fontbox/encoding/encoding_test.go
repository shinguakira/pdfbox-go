package encoding

import "testing"

// Ported from
// fontbox/src/test/java/org/apache/fontbox/encoding/EncodingTest.java.

func TestStandardEncoding(t *testing.T) {
	// check some randomly chosen mappings
	for _, c := range []struct {
		code int
		name string
	}{
		{0, ".notdef"},
		{32, "space"},
		{112, "p"},
		{172, "guilsinglleft"},
	} {
		if got := StandardEncoding.Name(c.code); got != c.name {
			t.Errorf("Name(%d) = %q, want %q", c.code, got, c.name)
		}
	}
	for _, c := range []struct {
		name string
		code int
	}{
		{"space", 32},
		{"p", 112},
		{"guilsinglleft", 172},
	} {
		got, ok := StandardEncoding.Code(c.name)
		if !ok || got != c.code {
			t.Errorf("Code(%q) = %d, %v, want %d, true", c.name, got, ok, c.code)
		}
	}
}

func TestMacRomanEncoding(t *testing.T) {
	// check some randomly chosen mappings
	for _, c := range []struct {
		code int
		name string
	}{
		{0, ".notdef"},
		{32, "space"},
		{112, "p"},
		{167, "germandbls"},
	} {
		if got := MacRomanEncoding.Name(c.code); got != c.name {
			t.Errorf("Name(%d) = %q, want %q", c.code, got, c.name)
		}
	}
	for _, c := range []struct {
		name string
		code int
	}{
		{"space", 32},
		{"p", 112},
		{"germandbls", 167},
	} {
		got, ok := MacRomanEncoding.Code(c.name)
		if !ok || got != c.code {
			t.Errorf("Code(%q) = %d, %v, want %d, true", c.name, got, ok, c.code)
		}
	}
}

// TestUnmappedCode pins that a code the table does not carry comes back as
// .notdef rather than empty, and that its name maps to nothing.
func TestUnmappedCode(t *testing.T) {
	if got := StandardEncoding.Name(0xFFFF); got != NotDef {
		t.Errorf("Name of an unmapped code = %q, want %q", got, NotDef)
	}
	if _, ok := StandardEncoding.Code("no such glyph"); ok {
		t.Error("Code of an unmapped name reported a mapping")
	}
}

// TestBuiltInEncoding pins the encoding a font carries within itself.
func TestBuiltInEncoding(t *testing.T) {
	builtIn := NewBuiltInEncoding(map[int]string{65: "alpha"})
	if got := builtIn.Name(65); got != "alpha" {
		t.Errorf("Name(65) = %q, want %q", got, "alpha")
	}
	if got, ok := builtIn.Code("alpha"); !ok || got != 65 {
		t.Errorf("Code(alpha) = %d, %v, want 65, true", got, ok)
	}
}

// TestCodeToNameMapIsACopy pins that the map handed out cannot be used to
// change the encoding, which is what Java's unmodifiableMap buys.
func TestCodeToNameMapIsACopy(t *testing.T) {
	m := StandardEncoding.CodeToNameMap()
	m[32] = "changed"
	if got := StandardEncoding.Name(32); got != "space" {
		t.Errorf("changing the returned map changed the encoding to %q", got)
	}
}

// TestTableSizes guards the tables against a transcription loss. They were
// generated from the Java source, and these are the row counts of
// StandardEncoding.java and MacRomanEncoding.java.
func TestTableSizes(t *testing.T) {
	if got := len(standardEncodingTable); got != 149 {
		t.Errorf("the standard encoding table has %d rows, want 149", got)
	}
	if got := len(macRomanEncodingTable); got != 207 {
		t.Errorf("the mac roman encoding table has %d rows, want 207", got)
	}
	// Every row maps, so the encoding is as big as its table bar duplicate
	// names, which the reverse map collapses.
	if got := len(StandardEncoding.CodeToNameMap()); got != 149 {
		t.Errorf("the standard encoding maps %d codes, want 149", got)
	}
	if got := len(MacRomanEncoding.CodeToNameMap()); got != 207 {
		t.Errorf("the mac roman encoding maps %d codes, want 207", got)
	}
}
