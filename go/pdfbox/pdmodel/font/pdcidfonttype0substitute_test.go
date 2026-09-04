package font_test

import (
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/cff"
	"github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// Port of org.apache.pdfbox.pdmodel.font.PDCIDFontType0SubstituteTest.
//
// A non-embedded Adobe-CNS1 CIDFontType0 must render real glyphs when a
// CID-keyed substitute is available, even when the substitute uses a different
// character collection (modern CJK fonts such as Noto CJK are
// Adobe-Identity-0). PDFBOX-6249.
//
// The test loads a PDF and so cannot sit in the font package, which the loader
// imports; every name it reaches for is exported.

// fontResourceDir is where the Java test resources of this package live,
// relative to it.
const fontResourceDir = "../../../../pdfbox/src/test/resources/org/apache/pdfbox/pdmodel/font/"

func TestLatinViaCns1NonEmbedded(t *testing.T) {
	doc, err := pdfbox.LoadPDF(fontResourceDir + "PDFBOX-6249-cns1-nonembedded-latin.pdf")
	if err != nil {
		t.Fatalf("loading the document: %v", err)
	}
	defer doc.Close()

	page := doc.Page(0)
	pdFont, err := page.Resources().GetFont(cos.GetPDFName("F1"))
	if err != nil {
		t.Fatalf("reading the font: %v", err)
	}
	type0Font, ok := pdFont.(*font.PDType0Font)
	if !ok {
		t.Fatalf("the font is %T, want a PDType0Font", pdFont)
	}
	cidFont, ok := type0Font.DescendantFont().(*font.PDCIDFontType0)
	if !ok {
		t.Fatalf("the descendant font is %T, want a PDCIDFontType0", type0Font.DescendantFont())
	}

	mapping := font.FontMappersInstance().GetCIDFont(cidFont.BaseFont(),
		cidFont.FontDescriptor(), cidFont.CIDSystemInfo())
	if !mapping.IsCIDFont() {
		t.Skip("no CID-keyed substitute for Adobe-CNS1 installed, can't test")
	}

	// 0x48 is "H": the CMap maps it to CID 41, which is not a valid glyph index
	// in an Adobe-Identity-0 substitute; it must be resolved via Unicode
	hasGlyph, err := cidFont.HasGlyph(0x48, type0Font)
	if err != nil {
		t.Fatalf("HasGlyph: %v", err)
	}
	if !hasGlyph {
		t.Error("glyph for 'H' not found in substitute")
	}
	path, err := cidFont.GetPath(0x48, type0Font)
	if err != nil {
		t.Fatalf("GetPath: %v", err)
	}
	if path.PathIterator(nil).IsDone() {
		t.Error("glyph for 'H' has an empty path")
	}

	// the repro descriptor carries no Panose or FontWeight: a regular weight
	// must outrank Bold among otherwise-tied candidates
	name, err := mapping.Font().Name()
	if err != nil {
		t.Fatalf("Name: %v", err)
	}
	if strings.HasSuffix(name, "-Bold") {
		t.Errorf("regular weight should be preferred over Bold on a weightless descriptor, got %s",
			name)
	}

	cffTable, err := mapping.Font().CFF()
	if err != nil {
		t.Fatalf("CFF: %v", err)
	}
	if cidKeyed, ok := cffTable.Font().(*cff.CFFCIDFont); ok && cidKeyed.Ordering() == "Identity" {
		// ASCII GIDs happen to line up with Adobe CIDs in Noto/Source Han, so
		// assert on an ideograph, where using the CID as a GID yields the wrong
		// glyph
		cmapLookup, err := mapping.Font().UnicodeCmapLookupStrict()
		if err != nil {
			t.Fatalf("UnicodeCmapLookupStrict: %v", err)
		}
		expected := cmapLookup.GetGlyphID(0x4E2D)
		if expected <= 0 {
			t.Fatalf("the substitute has no glyph for U+4E2D, got GID %d", expected)
		}
		gid, err := cidFont.CodeToGID(0x4E2D, type0Font)
		if err != nil {
			t.Fatalf("CodeToGID: %v", err)
		}
		if gid != expected {
			t.Errorf("GID must be resolved via Unicode, not used as a CID: got %d, want %d",
				gid, expected)
		}
	}
}
