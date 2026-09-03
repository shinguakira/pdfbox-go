package ttf

import "testing"

// Port of org.apache.fontbox.ttf.WGL4NamesTest.
//
// Java's getGlyphName returns null out of range and the port returns the empty
// string; getGlyphIndex returns null for a name it does not know and the port
// reports that through a second result.

func TestAllNames(t *testing.T) {
	allNames := AllGlyphNames()
	if allNames == nil {
		t.Fatal("all names is nil")
	}
	if len(allNames) != NumberOfMacGlyphs {
		t.Errorf("len(allNames) = %d, want %d", len(allNames), NumberOfMacGlyphs)
	}
}

func TestGetGlyphName(t *testing.T) {
	cases := []struct {
		index int
		want  string
	}{
		{0, ".notdef"},
		{32, "equal"},
		{75, "h"},
		{201, "Aacute"},
		{209, "Ocircumflex"},
		{256, "ccaron"},
		{NumberOfMacGlyphs + 1, ""},
		{-1, ""},
	}
	for _, c := range cases {
		if got := GlyphName(c.index); got != c.want {
			t.Errorf("GlyphName(%d) = %q, want %q", c.index, got, c.want)
		}
	}
}

func TestGlyphIndices(t *testing.T) {
	cases := []struct {
		name string
		want int
	}{
		{".notdef", 0},
		{"equal", 32},
		{"h", 75},
		{"Aacute", 201},
		{"Ocircumflex", 209},
		{"ccaron", 256},
	}
	for _, c := range cases {
		got, ok := GlyphIndex(c.name)
		if !ok {
			t.Errorf("GlyphIndex(%q) not found", c.name)
			continue
		}
		if got != c.want {
			t.Errorf("GlyphIndex(%q) = %d, want %d", c.name, got, c.want)
		}
	}
	if _, ok := GlyphIndex("INVALID"); ok {
		t.Error("GlyphIndex(\"INVALID\") found a glyph")
	}
}
