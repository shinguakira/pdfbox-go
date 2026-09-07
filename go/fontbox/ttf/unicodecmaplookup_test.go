package ttf

// getUnicodeCmapLookup when the font has no cmap at all.
//
// The lenient form answers Java's null there, and a *CmapSubtable that is nil
// boxed in a Go interface is not == nil, so the port has to hand back an
// untyped nil or every caller's null check silently passes. But that
// normalisation belongs where Java returns the cmap, not before the GSUB
// branch: with a feature enabled and a GSUB table present, Java wraps the null
// cmap in a SubstitutingCmapLookup and returns *that*, which is not null.
//
// The wanted values below were printed by the running Java, driving fontbox
// with the cmap table removed from the font by reflection.

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// lohitDevanagari has a GSUB table, which the case needs.
const lohitDevanagari = "../../../pdfbox/src/test/resources/org/apache/pdfbox/ttf/" +
	"Lohit-Devanagari.ttf"

// parseWithoutCmap parses the font and removes its cmap table, which is the
// state a lenient lookup answers null for. Java's own driver did it the same
// way, by reaching into the table map.
func parseWithoutCmap(t *testing.T, removeCmap bool) *TrueTypeFont {
	t.Helper()
	source, err := pdfio.OpenBufferedFile(lohitDevanagari)
	if err != nil {
		t.Fatalf("opening %s: %v", lohitDevanagari, err)
	}
	font, err := NewParser().Parse(source)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	t.Cleanup(func() { font.Close() })
	if removeCmap {
		delete(font.tables, CmapTag)
		cmap, err := font.Cmap()
		if err != nil {
			t.Fatalf("Cmap after removing it: %v", err)
		}
		if cmap != nil {
			t.Fatalf("the cmap table is still there after removing it")
		}
	}
	return font
}

// TestUnicodeCmapLookupKeepsTheGsubBranch checks the four states Java
// distinguishes.
func TestUnicodeCmapLookupKeepsTheGsubBranch(t *testing.T) {
	// With a cmap and a feature: a SubstitutingCmapLookup, as ever.
	withCmap := parseWithoutCmap(t, false)
	withCmap.EnableGsubFeature("abvs")
	lookup, err := withCmap.UnicodeCmapLookup(false)
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	if _, ok := lookup.(*SubstitutingCmapLookup); !ok {
		t.Errorf("with a cmap and a feature the lookup is %T, want *SubstitutingCmapLookup",
			lookup)
	}

	// No cmap and no feature: Java returns null.
	plain := parseWithoutCmap(t, true)
	lookup, err = plain.UnicodeCmapLookup(false)
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	if lookup != nil {
		t.Errorf("with no cmap and no feature the lookup is %#v, want nil", lookup)
	}

	// No cmap but a feature enabled: Java returns a SubstitutingCmapLookup
	// around the null cmap. It is *not* null, and a caller's null check has to
	// see that.
	substituting := parseWithoutCmap(t, true)
	substituting.EnableGsubFeature("abvs")
	lookup, err = substituting.UnicodeCmapLookup(false)
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	if lookup == nil {
		t.Fatal("with no cmap but a feature enabled the lookup is nil; Java returns " +
			"a SubstitutingCmapLookup wrapping the null cmap")
	}
	if _, ok := lookup.(*SubstitutingCmapLookup); !ok {
		t.Fatalf("the lookup is %T, want *SubstitutingCmapLookup", lookup)
	}

	// And using it fails, because both sides dereference the cmap unguarded.
	// Java raises NullPointerException; the port panics on the nil pointer.
	func() {
		defer func() {
			got := recover()
			if got == nil {
				t.Error("GetGlyphID on a lookup with no cmap returned; Java raises " +
					"NullPointerException")
				return
			}
			if !strings.Contains(strings.ToLower(takeString(got)), "nil pointer") {
				t.Errorf("GetGlyphID panicked with %v; want the nil dereference "+
					"Java raises as NullPointerException", got)
			}
		}()
		lookup.GetGlyphID(0x0915)
	}()
}

// takeString renders a recovered value.
func takeString(v any) string {
	if err, ok := v.(error); ok {
		return err.Error()
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
