package font_test

// What the embedder writes into the document, and the two book-keeping paths
// around it.
//
// TestFontEmbedding checks the round trip -- write text, read the same text
// back -- which is the right end-to-end assertion and passes even when a
// dictionary entry is wrong in a way the port's own reader forgives. These
// cases name the entries instead, so that a defect in one of them fails here
// rather than somewhere downstream.

import (
	"bytes"
	"io"
	"strings"
	"testing"

	pdfbox "github.com/shinguakira/pdfbox-go/go/pdfbox"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// writeSubsettedDocument writes one line with a subsetted LiberationSans and
// hands back the saved bytes. The callback runs after the text and before the
// save, which is where the subset is built.
func writeSubsettedDocument(t *testing.T, message string,
	before func(*font.PDType0Font)) []byte {
	t.Helper()
	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	embedded, err := font.LoadPDType0Font(document, openFont(t, liberationSans))
	if err != nil {
		t.Fatalf("LoadPDType0Font: %v", err)
	}
	writeText(t, document, page, embedded, 12, 50, 600, message)
	if before != nil {
		before(embedded)
	}

	var out bytes.Buffer
	if err := document.Save(&out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	return out.Bytes()
}

// reloadDescendant reloads a saved document and returns its Type 0 font and the
// descendant CIDFont's dictionary.
func reloadDescendant(t *testing.T, data []byte) (*pdmodel.PDDocument,
	*font.PDType0Font, *cos.Dictionary) {
	t.Helper()
	document, err := pdfbox.LoadPDFBytes(data)
	if err != nil {
		t.Fatalf("LoadPDFBytes: %v", err)
	}
	names := document.Page(0).Resources().FontNames()
	if len(names) == 0 {
		t.Fatal("the reloaded page has no font")
	}
	reloaded, err := document.Page(0).Resources().GetFont(names[0])
	if err != nil {
		t.Fatalf("GetFont: %v", err)
	}
	type0, ok := reloaded.(*font.PDType0Font)
	if !ok {
		t.Fatalf("the reloaded font is %T, want *font.PDType0Font", reloaded)
	}
	descendant, ok := type0.DescendantFont().COSObject().(*cos.Dictionary)
	if !ok {
		t.Fatal("the descendant font has no dictionary")
	}
	return document, type0, descendant
}

// TestSubsetWritesTheEntriesTheSpecificationAsksFor names each entry the
// subsetting path writes: the tagged /BaseFont from addNameTag, /W from
// buildWidths, /CIDToGIDMap from buildCIDToGIDMap and the descriptor's /CIDSet
// from buildCIDSet, which PDF/A requires.
func TestSubsetWritesTheEntriesTheSpecificationAsksFor(t *testing.T) {
	data := writeSubsettedDocument(t, "Unicode русский язык Tiếng Việt", nil)
	document, type0, descendant := reloadDescendant(t, data)
	defer document.Close()

	// addNameTag: six upper-case letters and a plus, on all three names.
	baseFont := type0.Name()
	if len(baseFont) < 7 || baseFont[6] != '+' {
		t.Fatalf("/BaseFont is %q; a subset name is six characters and a plus", baseFont)
	}
	for i := 0; i < 6; i++ {
		if baseFont[i] < 'A' || baseFont[i] > 'Z' {
			t.Errorf("/BaseFont is %q; the tag is not six upper-case letters", baseFont)
			break
		}
	}
	if !strings.HasSuffix(baseFont, "LiberationSans") {
		t.Errorf("/BaseFont is %q; the tag should prefix the font's own name", baseFont)
	}
	if got := descendant.GetNameAsString(cos.BaseFont, ""); got != baseFont {
		t.Errorf("the descendant's /BaseFont is %q, want the parent's %q", got, baseFont)
	}
	if fd := type0.DescendantFont().FontDescriptor(); fd == nil || fd.FontName() != baseFont {
		t.Error("the descriptor's /FontName does not carry the tag")
	}

	// buildWidths: /W is present and is the c [w...] shape.
	widths, ok := descendant.GetDictionaryObject(cos.W).(*cos.Array)
	if !ok || widths.Size() == 0 {
		t.Fatalf("/W is %v, want a non-empty array", descendant.GetDictionaryObject(cos.W))
	}
	if _, isInt := widths.GetObject(0).(*cos.Integer); !isInt {
		t.Errorf("/W starts with %T, want the first CID as an integer", widths.GetObject(0))
	}

	// buildCIDToGIDMap: a stream, not the /Identity name a whole font gets.
	if _, isStream := descendant.GetDictionaryObject(cos.CIDToGIDMap).(*cos.Stream); !isStream {
		t.Errorf("/CIDToGIDMap is %v, want a stream for a subsetted font",
			descendant.GetDictionaryObject(cos.CIDToGIDMap))
	}

	// buildCIDSet: PDF/A wants it, and every CID up to the last is set.
	descriptor := type0.DescendantFont().FontDescriptor()
	cidSet := descriptor.CIDSet()
	if cidSet == nil {
		t.Fatal("the descriptor has no /CIDSet")
	}
	bits, err := cidSet.ToByteArray()
	if err != nil {
		t.Fatalf("reading /CIDSet: %v", err)
	}
	if len(bits) == 0 {
		t.Fatal("/CIDSet is empty")
	}
	for i, b := range bits[:len(bits)-1] {
		if b != 0xFF {
			t.Errorf("/CIDSet byte %d is %#02x; every CID up to the last is set", i, b)
			break
		}
	}
}

// TestAWholeFontWritesIdentityCIDToGIDMap is the other side of the same entry:
// with subsetting off, CID is GID and Java writes the name /Identity rather
// than a stream, and there is no subset tag.
func TestAWholeFontWritesIdentityCIDToGIDMap(t *testing.T) {
	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)
	embedded, err := font.LoadPDType0FontSubset(document, openFont(t, liberationSans), false)
	if err != nil {
		t.Fatalf("LoadPDType0FontSubset: %v", err)
	}
	if embedded.WillBeSubset() {
		t.Error("WillBeSubset is true for a font loaded with subsetting off")
	}
	writeText(t, document, page, embedded, 12, 50, 600, "Unicode")
	var out bytes.Buffer
	if err := document.Save(&out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reloaded, type0, descendant := reloadDescendant(t, out.Bytes())
	defer reloaded.Close()

	if got := descendant.GetCOSName(cos.CIDToGIDMap); got != cos.Identity {
		t.Errorf("/CIDToGIDMap is %v, want /Identity for a whole font", got)
	}
	if strings.Contains(type0.Name(), "+") {
		t.Errorf("/BaseFont is %q; a font that is not subset carries no tag", type0.Name())
	}
}

// TestAddGlyphsToSubsetKeepsAGlyphNothingDrew covers addGlyphsToSubset, which
// PDAbstractContentStream calls with the glyphs a GSUB substitution produced --
// glyphs no code point in the text asked for, and which the subsetter would
// otherwise drop.
//
// It writes "A" and asks for the glyph of "Z" as well.
func TestAddGlyphsToSubsetKeepsAGlyphNothingDrew(t *testing.T) {
	original := mustParseFont(t, liberationSans)
	cmap, err := original.UnicodeCmapLookup(true)
	if err != nil {
		t.Fatalf("UnicodeCmapLookup: %v", err)
	}
	zed := cmap.GetGlyphID(int('Z'))
	if zed == 0 {
		t.Fatal("LiberationSans has no glyph for Z")
	}

	withZ := writeSubsettedDocument(t, "A", func(f *font.PDType0Font) {
		f.AddGlyphsToSubset(map[int]bool{zed: true})
	})
	withoutZ := writeSubsettedDocument(t, "A", nil)

	documentWith, _, descendantWith := reloadDescendant(t, withZ)
	defer documentWith.Close()
	documentWithout, _, descendantWithout := reloadDescendant(t, withoutZ)
	defer documentWithout.Close()

	// The CID is the original glyph id, so the map has to be long enough to
	// reach it and hold a glyph there.
	if got := cidToGIDEntry(t, descendantWith, zed); got == 0 {
		t.Errorf("/CIDToGIDMap has no glyph for CID %d; addGlyphsToSubset was ignored", zed)
	}
	if got := cidToGIDEntry(t, descendantWithout, zed); got != 0 {
		t.Errorf("/CIDToGIDMap has glyph %d for CID %d without addGlyphsToSubset; "+
			"the case proves nothing", got, zed)
	}
}

// cidToGIDEntry reads one entry out of a /CIDToGIDMap stream, answering 0 where
// the map is too short to reach it -- which is what a missing glyph looks like.
func cidToGIDEntry(t *testing.T, descendant *cos.Dictionary, cid int) int {
	t.Helper()
	stream, ok := descendant.GetDictionaryObject(cos.CIDToGIDMap).(*cos.Stream)
	if !ok {
		t.Fatalf("/CIDToGIDMap is %v, want a stream",
			descendant.GetDictionaryObject(cos.CIDToGIDMap))
	}
	reader, err := stream.CreateReader()
	if err != nil {
		t.Fatalf("reading /CIDToGIDMap: %v", err)
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading /CIDToGIDMap: %v", err)
	}
	if cid*2+1 >= len(data) {
		return 0
	}
	return int(data[cid*2])<<8 | int(data[cid*2+1])
}

// TestSubsettingDisabledPanics is the three IllegalStateExceptions PDType0Font
// raises for a font that is not being subset -- which is every font read out of
// a PDF, and every font loaded with embedSubset false.
func TestSubsettingDisabledPanics(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()
	whole, err := font.LoadPDType0FontSubset(document, openFont(t, liberationSans), false)
	if err != nil {
		t.Fatalf("LoadPDType0FontSubset: %v", err)
	}

	const want = "This font was created with subsetting disabled"
	for _, row := range []struct {
		name string
		call func()
	}{
		{"AddToSubset", func() { whole.AddToSubset(int('A')) }},
		{"AddGlyphsToSubset", func() { whole.AddGlyphsToSubset(map[int]bool{1: true}) }},
		{"Subset", func() { _ = whole.Subset() }},
	} {
		t.Run(row.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if got == nil {
					t.Errorf("%s returned; want the panic Java raises as "+
						"IllegalStateException", row.name)
					return
				}
				if message, _ := got.(string); message != want {
					t.Errorf("%s panicked with %v, want %q", row.name, got, want)
				}
			}()
			row.call()
		})
	}
}

// TestClosingTheDocumentClosesTheFontItRegistered covers
// registerTrueTypeFontForClosing: a subsetting embedder cannot close the font
// program when load returns, because the subset is not built until the document
// is saved, so the document owns it instead. PDType0Font.subset closes it too
// and sets its own reference to nil, so the document's close is the second one
// and has to be harmless -- which is what Java's HashSet plus a nulled field
// arrive at as well.
func TestClosingTheDocumentClosesTheFontItRegistered(t *testing.T) {
	document := pdmodel.NewPDDocument()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	// The load that parses the font is the one that registers it; the overload
	// that is handed a parsed font leaves it to the caller.
	embedded, err := font.LoadPDType0Font(document, openFont(t, liberationSans))
	if err != nil {
		t.Fatalf("LoadPDType0Font: %v", err)
	}
	writeText(t, document, page, embedded, 12, 50, 600, "A")

	var out bytes.Buffer
	if err := document.Save(&out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := document.Close(); err != nil {
		t.Fatalf("Close after the subset was built: %v", err)
	}
	// Closing twice must not double-close what it registered.
	if err := document.Close(); err != nil {
		t.Errorf("closing a closed document: %v", err)
	}
}
