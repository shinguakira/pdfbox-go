package processor_test

// PDFBOX-5660, Apache f7654f001, in the sync of 2026-09-07.
//
// ensureFontResources opened with
//
//	String daString = field.getDefaultAppearance();
//	if (daString.startsWith("/") && daString.length() > 1)
//
// and getDefaultAppearance answers null for anything that is not a COSString:
//
//	COSBase base = getInheritableAttribute(COSName.DA);
//	if (!(base instanceof COSString)) return null;
//
// So a widget carrying /DA as a name, which is the kind of thing this
// processor is here to survive, threw NullPointerException out of the fixup
// that was repairing it. Apache added an early return. The AcroForm's own /DA
// cannot be the one that does it: the defaults processor writes a COSString
// there before this one runs.
//
// The Go needs no change and never did. PDVariableText.DefaultAppearance
// answers the empty string where Java answers null, and HasPrefix("", "/") is
// false, so the port already returned. What is below pins it, because "the
// empty string stands in for null" is a decision a later reader could undo
// without noticing that this is what depends on it.

import (
	"testing"

	"github.com/shinguakira/pdfbox-go/go/pdfbox/cos"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/common"
	_ "github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/fixup"
	"github.com/shinguakira/pdfbox-go/go/pdfbox/pdmodel/interactive/form"
)

// TestOrphanWidgetWithADefaultAppearanceThatIsNotAString is the same document
// as the test beside it with the widget's /DA written as a name.
func TestOrphanWidgetWithADefaultAppearanceThatIsNotAString(t *testing.T) {
	document := pdmodel.NewPDDocument()
	defer document.Close()

	page := pdmodel.NewPDPageOfSize(common.A4)
	document.AddPage(page)

	widget := cos.NewDictionary()
	widget.SetItem(cos.Type, cos.Annot)
	widget.SetItem(cos.Subtype, cos.Widget)
	widget.SetItem(cos.FT, cos.Tx)
	widget.SetString(cos.T, "orphan")
	// Not a COSString, which is what makes getDefaultAppearance answer null.
	widget.SetItem(cos.DA, cos.GetPDFName("Helv"))
	widget.SetItem(cos.Rect, common.NewPDRectangleOf(50, 750, 150, 20).COSArray())
	page.Dictionary().SetItem(cos.Annots, cos.NewArrayOf([]cos.Base{widget}))

	acroFormDict := cos.NewDictionary()
	acroFormDict.SetItem(cos.Fields, cos.NewArray())
	acroFormDict.SetBoolean(cos.NeedAppearances, true)
	document.DocumentCatalog().Dictionary().SetItem(cos.AcroForm, acroFormDict)

	// The fixup runs here, and this is where the Java threw.
	acroForm := form.AcroFormOfCatalog(document.DocumentCatalog())
	if acroForm == nil {
		t.Fatal("the document has no AcroForm")
	}
	if got := len(acroForm.Fields()); got != 1 {
		t.Fatalf("the fixup rebuilt %d fields from the widget, want 1", got)
	}

	field, isVariableText := acroForm.Fields()[0].(*form.PDTextField)
	if !isVariableText {
		t.Fatalf("the rebuilt field is %T, want a text field", acroForm.Fields()[0])
	}
	if got := field.DefaultAppearance(); got != "" {
		t.Errorf("DefaultAppearance() = %q, want the empty string that stands in "+
			"for the null Java answers; the guard in ensureFontResources reads it",
			got)
	}
}
