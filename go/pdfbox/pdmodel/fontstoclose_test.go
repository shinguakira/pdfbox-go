package pdmodel

// registerTrueTypeFontForClosing, whose Java field is a Set.
//
// A subsetting embedder cannot close the font program when load returns,
// because the subset is not built until the document is saved, so the document
// owns it. Java holds those in a Set<TrueTypeFont>, so the same program
// registered twice is closed once; this checks the port does the same.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/fontbox/ttf"
	"github.com/shinguakira/pdfbox-go/go/pdfio"
)

// TestRegisterTrueTypeFontForClosingIsASet registers one font twice and a
// second font once.
func TestRegisterTrueTypeFontForClosingIsASet(t *testing.T) {
	const liberationSans = "../../../pdfbox/src/main/resources/org/apache/pdfbox/resources/" +
		"ttf/LiberationSans-Regular.ttf"

	parse := func() *ttf.TrueTypeFont {
		t.Helper()
		source, err := pdfio.OpenBufferedFile(liberationSans)
		if err != nil {
			t.Fatalf("opening %s: %v", liberationSans, err)
		}
		font, err := ttf.NewParser().Parse(source)
		if err != nil {
			t.Fatalf("Parse: %v", err)
		}
		return font
	}

	document := NewPDDocument()
	first := parse()
	second := parse()

	document.RegisterTrueTypeFontForClosing(first)
	document.RegisterTrueTypeFontForClosing(first)
	document.RegisterTrueTypeFontForClosing(second)

	if got := len(document.fontsToClose); got != 2 {
		t.Errorf("the document holds %d fonts to close, want 2; Java's field is a Set "+
			"and the same program was registered twice", got)
	}

	if err := document.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	if document.fontsToClose != nil {
		t.Error("Close left the set behind")
	}
}
