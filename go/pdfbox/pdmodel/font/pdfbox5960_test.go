package font_test

// PDFBOX-5960, Apache 6849822cf, 49855d4f9 and a1f50ab4b, in the sync of
// 2026-09-07. The only change in that sync that moves a glyph.
//
// PDTrueTypeFont.codeToGID branches on the Symbolic flag: a non-symbolic font
// is resolved by glyph name, through the (3, 1) cmap, the (1, 0) cmap and then
// the 'post' table; a symbolic one is resolved by character code, straight into
// the cmap. Some fonts set both the Symbolic and the NonSymbolic flag, which
// says nothing, and codeToGID took the symbolic path for them. Where such a
// font also carries an Encoding dictionary with a /BaseEncoding that is one of
// the three named ones, the file has said what the codes mean, and Apache now
// resolves by name first and falls back to the code only if that finds nothing.
//
// The Java test that came with it reads a JIRA attachment,
// PDFBOX-5960-reduced1.pdf, downloaded by pom.xml at build time; this
// repository has no copy and never fetches one. What is below is the same case
// over LiberationSans, which is in the Java's own resources: /Differences maps
// code 71 to "ccedilla" while the code itself is 'G' in the (3, 1) cmap, so the
// two paths answer two different glyphs and which one is taken is visible.
//
// The two GIDs are read out of LiberationSans-Regular.ttf and are what its
// 'post' table and (3, 1) cmap both answer:
//
//	G         GID 42
//	ccedilla  GID 169

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font/encoding"
)

const (
	gidOfG        = 42
	gidOfCcedilla = 169

	flagSymbolic    = 4
	flagNonSymbolic = 32
)

// aTrueTypeFontWith embeds LiberationSans, puts the given /Flags on its
// descriptor and the given /Encoding on the font, and reads the result back the
// way PDFontFactory does.
func aTrueTypeFontWith(t *testing.T, flags int, fontEncoding cos.Base) *font.PDTrueTypeFont {
	t.Helper()
	document := pdmodel.NewPDDocument()
	t.Cleanup(func() { document.Close() })

	embedded, err := font.LoadPDTrueTypeFont(document, openFont(t, liberationSans),
		encoding.WinAnsiEncodingInstance)
	if err != nil {
		t.Fatalf("LoadPDTrueTypeFont: %v", err)
	}
	dictionary := embedded.COSObject().(*cos.Dictionary)
	dictionary.GetCOSDictionary(cos.FontDescriptor).SetInt(cos.Flags, flags)
	if fontEncoding == nil {
		dictionary.RemoveItem(cos.Encoding)
	} else {
		dictionary.SetItem(cos.Encoding, fontEncoding)
	}

	reloaded, err := font.NewPDTrueTypeFontFromDictionary(dictionary, nil)
	if err != nil {
		t.Fatalf("NewPDTrueTypeFontFromDictionary: %v", err)
	}
	return reloaded
}

// anEncodingDictionary is /Encoding as a dictionary: a /BaseEncoding, where
// base is not empty, and 71 -> ccedilla in /Differences.
func anEncodingDictionary(base string) *cos.Dictionary {
	differences := cos.NewArray()
	differences.Add(cos.GetInteger(71))
	differences.Add(cos.GetPDFName("ccedilla"))

	dictionary := cos.NewDictionary()
	dictionary.SetItem(cos.Type, cos.GetPDFName("Encoding"))
	if base != "" {
		dictionary.SetItem(cos.BaseEncoding, cos.GetPDFName(base))
	}
	dictionary.SetItem(cos.Differences, differences)
	return dictionary
}

func gidOf(t *testing.T, f *font.PDTrueTypeFont, code int) int {
	t.Helper()
	gid, err := f.CodeToGID(code)
	if err != nil {
		t.Fatalf("CodeToGID(%d): %v", code, err)
	}
	return gid
}

// TestContradictorySymbolicFlagsResolveByNameFirst is the change.
func TestContradictorySymbolicFlagsResolveByNameFirst(t *testing.T) {
	for _, base := range []string{"WinAnsiEncoding", "MacRomanEncoding", "StandardEncoding"} {
		f := aTrueTypeFontWith(t, flagSymbolic|flagNonSymbolic, anEncodingDictionary(base))
		if !f.FontDescriptor().IsSymbolic() || !f.FontDescriptor().IsNonSymbolic() {
			t.Fatalf("/%s: the descriptor does not carry both flags", base)
		}
		if got := gidOf(t, f, 71); got != gidOfCcedilla {
			t.Errorf("/BaseEncoding /%s: CodeToGID(71) = %d, want %d; %d is the "+
				"glyph the code reaches through the cmap, which is what a font "+
				"whose Symbolic flag could be trusted would answer",
				base, got, gidOfCcedilla, gidOfG)
		}
	}
}

// TestSymbolicOnlyStillResolvesByTheCode is the first guard: a font whose
// Symbolic flag says one thing and nothing contradicts it keeps the code-based
// path, /Differences or no /Differences. This is the case the new branch must
// not reach.
func TestSymbolicOnlyStillResolvesByTheCode(t *testing.T) {
	f := aTrueTypeFontWith(t, flagSymbolic, anEncodingDictionary("WinAnsiEncoding"))
	if got := gidOf(t, f, 71); got != gidOfG {
		t.Errorf("CodeToGID(71) = %d for a font that is only symbolic, want %d",
			got, gidOfG)
	}
}

// TestNonSymbolicIsUnchanged is the second guard: the non-symbolic path already
// resolved by name, and the refactor that pulled it out into codeToGIDByName
// must not have moved it.
func TestNonSymbolicIsUnchanged(t *testing.T) {
	f := aTrueTypeFontWith(t, flagNonSymbolic, anEncodingDictionary("WinAnsiEncoding"))
	if got := gidOf(t, f, 71); got != gidOfCcedilla {
		t.Errorf("CodeToGID(71) = %d for a non-symbolic font, want %d",
			got, gidOfCcedilla)
	}
}

// TestContradictoryFlagsWithNoRecognizedBaseEncoding is the third guard, and
// the reason the check is on the /BaseEncoding rather than on the flags alone.
// An Encoding dictionary with no /BaseEncoding has not said what the codes
// mean: what it falls back to is the font's own built-in encoding, which is
// the cmap the code path reads anyway.
func TestContradictoryFlagsWithNoRecognizedBaseEncoding(t *testing.T) {
	f := aTrueTypeFontWith(t, flagSymbolic|flagNonSymbolic, anEncodingDictionary(""))
	if got := gidOf(t, f, 71); got != gidOfG {
		t.Errorf("CodeToGID(71) = %d with no /BaseEncoding to read, want %d",
			got, gidOfG)
	}
}

// TestContradictoryFlagsWithNoEncodingDictionary is the fourth: /Encoding as a
// name is not a DictionaryEncoding, so there are no /Differences to resolve and
// the branch does not apply.
func TestContradictoryFlagsWithNoEncodingDictionary(t *testing.T) {
	f := aTrueTypeFontWith(t, flagSymbolic|flagNonSymbolic,
		cos.GetPDFName("WinAnsiEncoding"))
	if got := gidOf(t, f, 71); got != gidOfG {
		t.Errorf("CodeToGID(71) = %d with /Encoding a name, want %d", got, gidOfG)
	}
}

// TestContradictoryFlagsFallBackToTheCodeWhenTheNameFindsNothing is the last
// one, and it is what "first" in the change means: the name path is tried and
// the code path still runs when it answers 0. A /Differences that names a glyph
// LiberationSans does not have leaves the code to answer.
func TestContradictoryFlagsFallBackToTheCodeWhenTheNameFindsNothing(t *testing.T) {
	differences := cos.NewArray()
	differences.Add(cos.GetInteger(71))
	differences.Add(cos.GetPDFName("NoSuchGlyphInLiberationSans"))
	encodingDictionary := cos.NewDictionary()
	encodingDictionary.SetItem(cos.Type, cos.GetPDFName("Encoding"))
	encodingDictionary.SetItem(cos.BaseEncoding, cos.GetPDFName("WinAnsiEncoding"))
	encodingDictionary.SetItem(cos.Differences, differences)

	f := aTrueTypeFontWith(t, flagSymbolic|flagNonSymbolic, encodingDictionary)
	if got := gidOf(t, f, 71); got != gidOfG {
		t.Errorf("CodeToGID(71) = %d for a name the font does not have, want %d "+
			"from the code path", got, gidOfG)
	}
}
