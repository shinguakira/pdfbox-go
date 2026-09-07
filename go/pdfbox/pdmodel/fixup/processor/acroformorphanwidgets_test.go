package processor_test

// The Java class that covers AcroFormOrphanWidgetsProcessor,
// PDAcroFormFromAnnotsTest, reads PDFs the Maven build downloads into
// target/pdfs, and is not ported -- migration/STATUS.md records it. The case
// below builds the same shape by hand instead: a document whose form has no
// /Fields but whose page carries a widget, and whose /DA names a font the
// default resources do not hold.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	_ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fixup"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// TestEnsureFontResourcesEmbedsTheReplacement is ensureFontResources: the
// widget's /DA names a font that is not in the form's /DR, so the processor
// looks one up through the font mapper and embeds it.
//
// Java calls PDType0Font.load(document, fontMapping.getFont(), false), which
// is what track/font-embedding brought. Before it the port did the lookup, logged
// that it could not embed, and left /DR without the font.
func TestEnsureFontResourcesEmbedsTheReplacement(t *testing.T) {
	// A name no system font can answer, so the mapper falls back and the case
	// does not depend on what is installed.
	const missing = "NoSuchFontFamilyXYZ"

	document := pdmodel.NewPDDocument()
	defer document.Close()

	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	widget := cos.NewDictionary()
	widget.SetItem(cos.Type, cos.Annot)
	widget.SetItem(cos.Subtype, cos.Widget)
	widget.SetItem(cos.FT, cos.Tx)
	widget.SetString(cos.T, "orphan")
	widget.SetString(cos.DA, "/"+missing+" 10 Tf 0 g")
	widget.SetItem(cos.Rect, common.NewPDRectangleOf(50, 750, 150, 20).COSArray())
	page.Dictionary().SetItem(cos.Annots, cos.NewArrayOf([]cos.Base{widget}))

	acroFormDict := cos.NewDictionary()
	acroFormDict.SetItem(cos.Fields, cos.NewArray())
	acroFormDict.SetBoolean(cos.NeedAppearances, true)
	document.DocumentCatalog().Dictionary().SetItem(cos.AcroForm, acroFormDict)

	// getAcroForm() with no argument, which applies the default fixup this
	// processor is part of.
	acroForm := form.AcroFormOfCatalog(document.DocumentCatalog())
	if acroForm == nil {
		t.Fatal("the document has no AcroForm")
	}
	if got := len(acroForm.Fields()); got != 1 {
		t.Fatalf("the fixup rebuilt %d fields from the widget, want 1", got)
	}

	name := cos.GetPDFName(missing)
	replacement, err := acroForm.DefaultResources().GetFont(name)
	if err != nil {
		t.Fatalf("GetFont(%s): %v", missing, err)
	}
	if replacement == nil {
		t.Fatalf("/DR has no font for %s; the replacement was looked up but not embedded", missing)
	}
	if !replacement.IsEmbedded() {
		t.Errorf("the replacement for %s is not embedded", missing)
	}
}
