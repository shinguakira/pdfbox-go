package glyphlayout_test

// JAVA-BUGS 77: `checkMissingGlyphs` formats `'%c'` from `text.charAt(index)`,
// a UTF-16 code unit, beside a code point read with `codePointAt`. For a
// character outside the basic plane the two disagree: the message carries the
// high surrogate on its own, which is ill-formed and prints as a replacement
// character or as nothing.

import (
	"os"
	"strings"
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/glyphlayout"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/font"
)

// TestMissingGlyphNamesTheWholeCharacter is the defect.
//
// The expected message is the one the method is written to give -- the
// character that could not be drawn, and its code point -- for a character
// outside the basic plane. The `%04x` half is already right in the Java; it is
// the `'%c'` half that carries half a surrogate pair. This module's own
// `GlyphLayoutSMPTest` is entirely about that plane, so a font missing one of
// its characters is not a hypothetical.
func TestMissingGlyphNamesTheWholeCharacter(t *testing.T) {
	const emoji = "\U0001F600"
	const want = "Missing glyph in font 'Lohit Bengali' for the character '" +
		emoji + "', codePoint: 128512 (U+1f600)."

	document := pdmodel.NewPDDocument()
	defer document.Close()
	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	file, err := os.Open(layoutFonts + "Lohit-Bengali.ttf")
	if err != nil {
		t.Skipf("Lohit-Bengali is not in this repository: %v", err)
	}
	defer file.Close()
	embedded, err := font.LoadPDType0FontSubset(document, file, false)
	if err != nil {
		t.Fatalf("LoadPDType0FontSubset: %v", err)
	}

	stream, err := pdmodel.NewPDPageContentStream(document, page)
	if err != nil {
		t.Fatalf("NewPDPageContentStream: %v", err)
	}
	stream.SetGlyphLayoutProcessor(glyphlayout.NewProcessor())
	if err := stream.BeginText(); err != nil {
		t.Fatal(err)
	}
	if err := stream.SetFont(embedded, 1); err != nil {
		t.Fatal(err)
	}
	if err := stream.NewLineAtOffset(0, 0); err != nil {
		t.Fatal(err)
	}

	err = stream.ShowText(emoji)
	if err == nil {
		t.Fatal("a Bengali font drew an emoji, so the test proves nothing")
	}
	if err.Error() != want {
		t.Errorf("the message is\n  %q\nwant\n  %q", err.Error(), want)
	}
	if strings.ContainsRune(err.Error(), '\uFFFD') {
		t.Error("the message carries a replacement character, which is the " +
			"unpaired surrogate written out")
	}
}
